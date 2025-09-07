package plugin

import (
	"compress/gzip"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type OutputPlugin interface {
	Start()
	Stop()
}

type BaseOutput struct {
	inputQueue    *Queue
	matchTags     string
	bufferSize    int
	flushInterval time.Duration
	buffer        []*Event
	running       bool
	mu            sync.Mutex
	wg            sync.WaitGroup
	lastFlush     time.Time
}

func NewBaseOutput(inputQueue *Queue, matchTags string, bufferSize int, flushInterval time.Duration) *BaseOutput {
	return &BaseOutput{
		inputQueue:    inputQueue,
		matchTags:     matchTags,
		bufferSize:    bufferSize,
		flushInterval: flushInterval,
		buffer:        make([]*Event, 0, bufferSize),
		running:       false,
		lastFlush:     time.Now(),
	}
}

func (o *BaseOutput) IsRunning() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.running
}

func (o *BaseOutput) SetRunning(running bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.running = running
}

func (o *BaseOutput) Matches(tag string) bool {
	// for _, pattern := range o.matchTags {
	// 	// 简单的通配符匹配，*匹配任意字符
	// 	if pattern == "*" {
	// 		return true
	// 	}
	// 	if pattern[len(pattern)-1] == '*' && len(tag) >= len(pattern)-1 &&
	// 		tag[:len(pattern)-1] == pattern[:len(pattern)-1] {
	// 		return true
	// 		return true
	// 	}
	// 	if tag == pattern {
	// 		return true
	// 	}
	// }
	// return false
	return true
}

// AddToBuffer 将事件添加到缓冲区
func (o *BaseOutput) AddToBuffer(event *Event) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.buffer = append(o.buffer, event)
}

// GetBuffer 获取并清空缓冲区
func (o *BaseOutput) GetBuffer() []*Event {
	o.mu.Lock()
	defer o.mu.Unlock()

	buffer := o.buffer
	o.buffer = make([]*Event, 0, o.bufferSize)
	o.lastFlush = time.Now()
	return buffer
}

// ShouldFlush 检查是否需要刷新缓冲区
func (o *BaseOutput) ShouldFlush() bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	return len(o.buffer) >= o.bufferSize || time.Since(o.lastFlush) >= o.flushInterval
}

// Flush 刷新缓冲区，子类需要实现具体的输出逻辑
func (o *BaseOutput) Flush(events []*Event) error {
	return nil
}

// StdoutOutput 输出到标准输出的插件
type StdoutOutput struct {
	*BaseOutput
}

// NewStdoutOutput 创建一个新的标准输出插件
func NewStdoutOutput(inputQueue *Queue, matchTags string, bufferSize int, flushInterval int) *StdoutOutput {
	return &StdoutOutput{
		BaseOutput: NewBaseOutput(inputQueue, matchTags, bufferSize, time.Duration(flushInterval)*time.Second),
	}
}

// Flush 刷新缓冲区，输出到标准输出
func (s *StdoutOutput) Flush(events []*Event) error {
	for _, event := range events {
		log.Printf("[%s] %s: %v", event.Timestamp.Format(time.RFC3339), event.Tag, event.Record)
	}
	return nil
}

func (s *StdoutOutput) Start() {
	if s.IsRunning() {
		return
	}

	s.SetRunning(true)
	s.BaseOutput.wg.Add(1)

	go func() {
		defer s.BaseOutput.wg.Done()
		log.Println("Starting StdoutOutput")

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for s.IsRunning() {
			select {
			case <-ticker.C:
				// 检查是否需要刷新
				if s.ShouldFlush() {
					buffer := s.GetBuffer()
					if len(buffer) > 0 {
						s.Flush(buffer)
					}
				}
			default:
				// 尝试获取事件
				event, ok := s.inputQueue.Get()
				if !ok {
					// 队列已关闭或无数据，短暂休眠
					time.Sleep(100 * time.Millisecond)
					continue
				}

				if s.Matches(event.Tag) {
					s.AddToBuffer(event)

					// 检查是否需要刷新
					if s.ShouldFlush() {
						buffer := s.GetBuffer()
						if len(buffer) > 0 {
							s.Flush(buffer)
						}
					}
				}
			}
		}

		// 停止前最后一次刷新
		buffer := s.GetBuffer()
		if len(buffer) > 0 {
			s.Flush(buffer)
		}
	}()
}

func (s *StdoutOutput) Stop() {
	if !s.IsRunning() {
		return
	}

	s.SetRunning(false)
	s.BaseOutput.wg.Wait()
	log.Println("Stopped StdoutOutput")
}

type FileOutput struct {
	*BaseOutput
	path        string
	compression bool
}

func NewFileOutput(inputQueue *Queue, matchTags string, path string, bufferSize int, flushInterval int, compression bool) *FileOutput {

	if compression && filepath.Ext(path) != ".gz" {
		path += ".gz"
	}

	// TODO
	// _, err := os.Stat(path)
	// if os.IsNotExist(err) {
	// 	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	// 	if err != nil {

	// 		log.Fatal("cannot create file: %v", err)
	// 	}
	// 	defer file.Close()
	// 	log.Println("file '%s' already create\n", path)
	// 	return nil
	// } else if err != nil {
	// 	// 其他错误
	// 	log.Fatal("check file error: %v", err)
	// }

	fo := &FileOutput{
		BaseOutput:  NewBaseOutput(inputQueue, matchTags, bufferSize, time.Duration(flushInterval)*time.Second),
		path:        path,
		compression: compression,
	}
	return fo
}

// Flush 刷新缓冲区，输出到文件
func (f *FileOutput) Flush(events []*Event) error {
	// 创建目录（如果需要）
	if err := os.MkdirAll(filepath.Dir(f.path), 0755); err != nil {
		return err
	}

	// 打开文件，追加模式
	file, err := os.OpenFile(f.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	var writer *gzip.Writer
	var closeFunc func() error
	if f.compression {
		writer = gzip.NewWriter(file)
		closeFunc = func() error {
			if err := writer.Close(); err != nil {
				return err
			}
			return file.Close()
		}
	} else {
		closeFunc = file.Close
	}

	// 写入事件
	for _, event := range events {
		data, err := json.Marshal(map[string]interface{}{
			"tag":       event.Tag,
			"timestamp": event.Timestamp.UnixNano() / 1e6, // 毫秒时间戳
			"record":    event.Record,
		})
		if err != nil {
			log.Printf("Error marshaling event: %v", err)
			continue
		}

		data = append(data, '\n') // 添加换行符

		var writeErr error
		if f.compression && writer != nil {
			_, writeErr = writer.Write(data)
		} else {
			_, writeErr = file.Write(data)
		}

		if writeErr != nil {
			log.Printf("Error writing to file: %v", writeErr)
		}
	}

	// 关闭 writer
	return closeFunc()
}

func (f *FileOutput) Start() {
	if f.IsRunning() {
		return
	}

	f.SetRunning(true)
	f.BaseOutput.wg.Add(1)

	go func() {
		defer f.BaseOutput.wg.Done()
		log.Printf("Starting FileOutput to %s", f.path)

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for f.IsRunning() {
			select {
			case <-ticker.C:
				// 检查是否需要刷新
				if f.ShouldFlush() {
					buffer := f.GetBuffer()
					if len(buffer) > 0 {
						f.Flush(buffer)
					}
				}
			default:
				// 尝试获取事件
				event, ok := f.inputQueue.Get()
				if !ok {
					// 队列已关闭或无数据，短暂休眠
					time.Sleep(100 * time.Millisecond)
					continue
				}

				if f.Matches(event.Tag) {
					f.AddToBuffer(event)

					// 检查是否需要刷新
					if f.ShouldFlush() {
						buffer := f.GetBuffer()
						if len(buffer) > 0 {
							f.Flush(buffer)
						}
					}
				}
			}
		}

		// 停止前最后一次刷新
		buffer := f.GetBuffer()
		if len(buffer) > 0 {
			f.Flush(buffer)
		}
	}()
}

func (f *FileOutput) Stop() {
	if !f.IsRunning() {
		return
	}

	f.SetRunning(false)
	f.BaseOutput.wg.Wait()
	log.Printf("Stopped FileOutput to %s", f.path)
}
