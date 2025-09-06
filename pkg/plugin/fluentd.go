package plugin

import (
	"sync"
)

// Fluentd 是日志处理系统的主结构
type Fluentd struct {
	inputs  []InputPlugin
	filters []FilterPlugin
	outputs []OutputPlugin
	queues  []*Queue
	wg      sync.WaitGroup
	running bool
	mu      sync.Mutex
}

// NewFluentd 创建一个新的Fluentd实例
func NewFluentd() *Fluentd {
	return &Fluentd{
		inputs:  []InputPlugin{},
		filters: []FilterPlugin{},
		outputs: []OutputPlugin{},
		queues:  []*Queue{},
		running: false,
	}
}

// AddInput 添加输入插件
func (f *Fluentd) AddInput(input InputPlugin) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inputs = append(f.inputs, input)
}

// AddFilter 添加过滤插件
func (f *Fluentd) AddFilter(filter FilterPlugin) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.filters = append(f.filters, filter)
}

// AddOutput 添加输出插件
func (f *Fluentd) AddOutput(output OutputPlugin) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.outputs = append(f.outputs, output)
}

// Start 启动所有组件
func (f *Fluentd) Start() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.running {
		return
	}

	f.running = true

	// 启动输出插件
	for _, output := range f.outputs {
		output.Start()
	}

	// 启动过滤插件
	for _, filter := range f.filters {
		filter.Start()
	}

	// 启动输入插件
	for _, input := range f.inputs {
		input.Start()
	}
}

// Stop 停止所有组件
func (f *Fluentd) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.running {
		return
	}

	// 先停止输入，防止新事件进入
	for _, input := range f.inputs {
		input.Stop()
	}

	// 再停止过滤
	for _, filter := range f.filters {
		filter.Stop()
	}

	// 最后停止输出，确保所有事件都被处理
	for _, output := range f.outputs {
		output.Stop()
	}

	// 关闭所有队列
	for _, queue := range f.queues {
		queue.Close()
	}

	f.running = false
}
