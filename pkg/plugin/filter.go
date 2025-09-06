package plugin

import (
	"log"
	"regexp"
	"sync"
	"time"
)

// FilterPlugin 过滤插件接口
type FilterPlugin interface {
	Start()
	Stop()
}

// BaseFilter 过滤插件基类
type BaseFilter struct {
	inputQueue  *Queue
	outputQueue *Queue
	matchTags   []string
	running     bool
	mu          sync.Mutex
	wg          sync.WaitGroup
}

// NewBaseFilter 创建一个新的基础过滤插件
func NewBaseFilter(inputQueue, outputQueue *Queue, matchTags []string) *BaseFilter {
	return &BaseFilter{
		inputQueue:  inputQueue,
		outputQueue: outputQueue,
		matchTags:   matchTags,
		running:     false,
	}
}

// IsRunning 检查插件是否在运行
func (f *BaseFilter) IsRunning() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.running
}

// SetRunning 设置插件运行状态
func (f *BaseFilter) SetRunning(running bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.running = running
}

// Matches 检查事件标签是否匹配
func (f *BaseFilter) Matches(tag string) bool {
	for _, pattern := range f.matchTags {
		// 简单的通配符匹配，*匹配任意字符
		if pattern == "*" {
			return true
		}
		if pattern[len(pattern)-1] == '*' && len(tag) >= len(pattern)-1 &&
			tag[:len(pattern)-1] == pattern[:len(pattern)-1] {
			return true
		}
		if tag == pattern {
			return true
		}
	}
	return false
}

// GrepFilter 基于正则表达式过滤事件
type GrepFilter struct {
	*BaseFilter
	key     string
	pattern *regexp.Regexp
	exclude bool
}

// NewGrepFilter 创建一个新的Grep过滤插件
func NewGrepFilter(inputQueue, outputQueue *Queue, matchTags []string, key, pattern string, exclude bool) *GrepFilter {
	return &GrepFilter{
		BaseFilter: NewBaseFilter(inputQueue, outputQueue, matchTags),
		key:        key,
		pattern:    regexp.MustCompile(pattern),
		exclude:    exclude,
	}
}

// Filter 执行过滤操作
func (g *GrepFilter) Filter(event *Event) *Event {
	value, ok := event.Record[g.key].(string)
	if !ok {
		// 如果字段不存在或不是字符串，根据exclude决定是否保留
		if g.exclude {
			return event
		}
		return nil
	}

	match := g.pattern.MatchString(value)

	// 根据exclude决定是否保留匹配的事件
	if (match && !g.exclude) || (!match && g.exclude) {
		return event
	}

	return nil
}

// Start 启动过滤插件
func (g *GrepFilter) Start() {
	if g.IsRunning() {
		return
	}

	g.SetRunning(true)
	g.BaseFilter.wg.Add(1)

	go func() {
		defer g.BaseFilter.wg.Done()
		log.Println("Starting GrepFilter")

		for g.IsRunning() {
			event, ok := g.inputQueue.Get()
			if !ok {
				// 队列已关闭或无数据，短暂休眠
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if g.Matches(event.Tag) {
				filteredEvent := g.Filter(event)
				if filteredEvent != nil {
					g.outputQueue.Put(filteredEvent)
				}
			} else {
				// 不匹配的事件直接传递
				g.outputQueue.Put(event)
			}
		}
	}()
}

// Stop 停止过滤插件
func (g *GrepFilter) Stop() {
	if !g.IsRunning() {
		return
	}

	g.SetRunning(false)
	g.BaseFilter.wg.Wait()
	log.Println("Stopped GrepFilter")
}

// RecordTransformerFilter 用于修改事件记录的过滤插件
type RecordTransformerFilter struct {
	*BaseFilter
	addFields    map[string]interface{}
	removeFields []string
}

// NewRecordTransformerFilter 创建一个新的记录转换过滤插件
func NewRecordTransformerFilter(inputQueue, outputQueue *Queue, matchTags []string, addFields map[string]interface{}, removeFields []string) *RecordTransformerFilter {
	return &RecordTransformerFilter{
		BaseFilter:   NewBaseFilter(inputQueue, outputQueue, matchTags),
		addFields:    addFields,
		removeFields: removeFields,
	}
}

// Filter 执行记录转换操作
func (r *RecordTransformerFilter) Filter(event *Event) *Event {
	// 添加字段
	for key, value := range r.addFields {
		event.Record[key] = value
	}

	// 移除字段
	for _, key := range r.removeFields {
		delete(event.Record, key)
	}

	return event
}

// Start 启动过滤插件
func (r *RecordTransformerFilter) Start() {
	if r.IsRunning() {
		return
	}

	r.SetRunning(true)
	r.BaseFilter.wg.Add(1)

	go func() {
		defer r.BaseFilter.wg.Done()
		log.Println("Starting RecordTransformerFilter")

		for r.IsRunning() {
			event, ok := r.inputQueue.Get()
			if !ok {
				// 队列已关闭或无数据，短暂休眠
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if r.Matches(event.Tag) {
				transformedEvent := r.Filter(event)
				if transformedEvent != nil {
					r.outputQueue.Put(transformedEvent)
				}
			} else {
				// 不匹配的事件直接传递
				r.outputQueue.Put(event)
			}
		}
	}()
}

// Stop 停止过滤插件
func (r *RecordTransformerFilter) Stop() {
	if !r.IsRunning() {
		return
	}

	r.SetRunning(false)
	r.BaseFilter.wg.Wait()
	log.Println("Stopped RecordTransformerFilter")
}
