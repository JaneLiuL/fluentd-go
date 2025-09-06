package plugin

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// InputPlugin 输入插件接口
type InputPlugin interface {
	Start()
	Stop()
}

// BaseInput 输入插件基类
type BaseInput struct {
	tag         string
	outputQueue *Queue
	running     bool
	mu          sync.Mutex
	wg          sync.WaitGroup
}

// NewBaseInput 创建一个新的基础输入插件
func NewBaseInput(tag string, outputQueue *Queue) *BaseInput {
	return &BaseInput{
		tag:         tag,
		outputQueue: outputQueue,
		running:     false,
	}
}

// IsRunning 检查插件是否在运行
func (i *BaseInput) IsRunning() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.running
}

// SetRunning 设置插件运行状态
func (i *BaseInput) SetRunning(running bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.running = running
}

// TailInput 文件尾输入插件，监控文件变化并读取新内容
type TailInput struct {
	*BaseInput
	path      string
	posFile   string
	positions map[string]int64
	observer  *FileObserver
}

// NewTailInput 创建一个新的文件尾输入插件
func NewTailInput(tag string, outputQueue *Queue, path, posFile string) *TailInput {
	input := &TailInput{
		BaseInput: NewBaseInput(tag, outputQueue),
		path:      path,
		posFile:   posFile,
		positions: make(map[string]int64),
	}

	// 加载文件位置记录
	input.loadPositions()

	// 创建文件观察器
	input.observer = NewFileObserver(filepath.Dir(path), func(event FileEvent) {
		if event.Path == path && event.Type == FileEventModify {
			input.readNewContent()
		}
	})

	return input
}

// 加载文件读取位置
func (t *TailInput) loadPositions() {
	if _, err := os.Stat(t.posFile); err == nil {
		data, err := os.ReadFile(t.posFile)
		if err == nil {
			json.Unmarshal(data, &t.positions)
		}
	}

	// 确保当前文件有位置记录
	if _, exists := t.positions[t.path]; !exists {
		t.positions[t.path] = 0
	}
}

// 保存文件读取位置
func (t *TailInput) savePositions() {
	data, err := json.Marshal(t.positions)
	if err != nil {
		log.Printf("Error saving positions: %v", err)
		return
	}

	// 创建目录（如果需要）
	if err := os.MkdirAll(filepath.Dir(t.posFile), 0755); err != nil {
		log.Printf("Error creating pos file directory: %v", err)
		return
	}

	if err := os.WriteFile(t.posFile, data, 0644); err != nil {
		log.Printf("Error writing pos file: %v", err)
	}
}

// 读取文件新内容
func (t *TailInput) readNewContent() {
	file, err := os.Open(t.path)
	if err != nil {
		log.Printf("Error opening file %s: %v", t.path, err)
		return
	}
	defer file.Close()

	// 移动到上次读取的位置
	pos := t.positions[t.path]
	if _, err := file.Seek(pos, 0); err != nil {
		log.Printf("Error seeking file %s: %v", t.path, err)
		return
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			event := NewEvent(t.tag, map[string]interface{}{
				"message": line,
			})
			t.outputQueue.Put(event)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading file %s: %v", t.path, err)
		return
	}

	// 获取当前文件位置并保存
	newPos, err := file.Seek(0, 1)
	if err != nil {
		log.Printf("Error getting file position: %v", err)
		return
	}

	if newPos != pos {
		t.positions[t.path] = newPos
		t.savePositions()
	}
}

// Start 启动输入插件
func (t *TailInput) Start() {
	if t.IsRunning() {
		return
	}

	t.SetRunning(true)
	t.BaseInput.wg.Add(1)

	go func() {
		defer t.BaseInput.wg.Done()
		log.Printf("Starting TailInput for %s with tag %s", t.path, t.tag)

		// 初始读取
		t.readNewContent()

		// 启动文件观察器
		t.observer.Start()

		// 保持运行
		for t.IsRunning() {
			time.Sleep(1 * time.Second)
		}

		// 停止观察器
		t.observer.Stop()
	}()
}

// Stop 停止输入插件
func (t *TailInput) Stop() {
	if !t.IsRunning() {
		return
	}

	t.SetRunning(false)
	t.BaseInput.wg.Wait()
	log.Printf("Stopped TailInput for %s", t.path)
}

// TcpInput TCP输入插件，接收网络日志
type TcpInput struct {
	*BaseInput
	address  string
	listener net.Listener
}

// NewTcpInput 创建一个新的TCP输入插件
func NewTcpInput(tag string, outputQueue *Queue, address string) *TcpInput {
	return &TcpInput{
		BaseInput: NewBaseInput(tag, outputQueue),
		address:   address,
	}
}

// 处理客户端连接
func (t *TcpInput) handleClient(conn net.Conn) {
	defer conn.Close()
	log.Printf("Accepted connection from %s", conn.RemoteAddr())

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() && t.IsRunning() {
		line := scanner.Text()
		if line != "" {
			event := NewEvent(t.tag, map[string]interface{}{
				"message": line,
			})
			t.outputQueue.Put(event)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from connection: %v", err)
	}

	log.Printf("Connection from %s closed", conn.RemoteAddr())
}

// Start 启动输入插件
func (t *TcpInput) Start() {
	if t.IsRunning() {
		return
	}

	var err error
	t.listener, err = net.Listen("tcp", t.address)
	if err != nil {
		log.Printf("Error starting TCP listener: %v", err)
		return
	}

	t.SetRunning(true)
	t.BaseInput.wg.Add(1)

	go func() {
		defer t.BaseInput.wg.Done()
		log.Printf("Starting TcpInput on %s with tag %s", t.address, t.tag)

		for t.IsRunning() {
			conn, err := t.listener.Accept()
			if err != nil {
				// 如果是正常关闭，不打印错误
				if !t.IsRunning() {
					break
				}
				log.Printf("Error accepting connection: %v", err)
				continue
			}

			// 启动新的goroutine处理客户端
			go t.handleClient(conn)
		}
	}()
}

// Stop 停止输入插件
func (t *TcpInput) Stop() {
	if !t.IsRunning() {
		return
	}

	t.SetRunning(false)
	if t.listener != nil {
		t.listener.Close()
	}
	t.BaseInput.wg.Wait()
	log.Printf("Stopped TcpInput on %s", t.address)
}
