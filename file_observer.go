package main

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileEventType string

const (
	FileEventCreate FileEventType = "create"
	FileEventModify FileEventType = "modify"
	FileEventDelete FileEventType = "delete"
)

type FileEvent struct {
	Path string
	Type FileEventType
}

// FileObserver 用于监控文件变化
type FileObserver struct {
	path     string
	callback func(FileEvent)
	running  bool
	mu       sync.Mutex
	wg       sync.WaitGroup
	lastMod  map[string]time.Time
}

func NewFileObserver(path string, callback func(FileEvent)) *FileObserver {
	return &FileObserver{
		path:     path,
		callback: callback,
		lastMod:  make(map[string]time.Time),
	}
}

func (f *FileObserver) getFileModTime(path string) (time.Time, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return fileInfo.ModTime(), nil
}

func (f *FileObserver) scan() {
	files, err := filepath.Glob(filepath.Join(f.path, "*"))
	if err != nil {
		log.Printf("Error scanning directory: %v", err)
		return
	}

	currentFiles := make(map[string]bool)

	// 检查现有文件
	for _, file := range files {
		currentFiles[file] = true

		modTime, err := f.getFileModTime(file)
		if err != nil {
			continue
		}

		lastMod, exists := f.lastMod[file]
		if !exists {
			// 新文件
			f.lastMod[file] = modTime
			f.callback(FileEvent{Path: file, Type: FileEventCreate})
		} else if modTime.After(lastMod) {
			// 文件已修改
			f.lastMod[file] = modTime
			f.callback(FileEvent{Path: file, Type: FileEventModify})
		}
	}

	// 检查已删除的文件
	for file := range f.lastMod {
		if !currentFiles[file] {
			delete(f.lastMod, file)
			f.callback(FileEvent{Path: file, Type: FileEventDelete})
		}
	}
}

// Start 启动文件观察器
func (f *FileObserver) Start() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.running {
		return
	}

	f.running = true
	f.wg.Add(1)

	// 初始化最后修改时间
	f.scan()

	go func() {
		defer f.wg.Done()

		// 定期扫描目录
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for f.running {
			select {
			case <-ticker.C:
				f.scan()
			}
		}
	}()
}

// Stop 停止文件观察器
func (f *FileObserver) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.running {
		return
	}

	f.running = false
	f.wg.Wait()
}
