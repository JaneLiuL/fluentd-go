package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 创建Fluentd实例
	fluent := NewFluentd()

	// 设置队列连接各组件
	inputQueue := NewQueue(1000)
	filterQueue := NewQueue(1000)
	outputQueue := NewQueue(1000)

	// 添加输入插件
	fileInput := NewTailInput("app.log", inputQueue, "/var/log/app.log", "app_log.pos")
	tcpInput := NewTcpInput("network.log", inputQueue, "0.0.0.0:24224")
	fluent.AddInput(fileInput)
	fluent.AddInput(tcpInput)

	// 添加过滤插件
	grepFilter := NewGrepFilter(inputQueue, filterQueue, []string{"app.log", "network.log"}, "message", "error", false)
	transformFilter := NewRecordTransformerFilter(filterQueue, outputQueue, []string{"app.log", "network.log"},
		map[string]interface{}{"environment": "production", "source": "fluentd-go"},
		[]string{},
	)
	fluent.AddFilter(grepFilter)
	fluent.AddFilter(transformFilter)

	// 添加输出插件
	stdoutOutput := NewStdoutOutput(outputQueue, []string{"app.log", "network.log"}, 10, 5)
	fileOutput := NewFileOutput(outputQueue, []string{"app.log", "network.log"}, "/var/log/filtered_errors.log", 10, 5)
	fluent.AddOutput(stdoutOutput)
	fluent.AddOutput(fileOutput)

	// 启动服务
	fluent.Start()
	log.Println("Fluentd clone is running. Press Ctrl+C to stop.")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 停止服务
	fluent.Stop()
	log.Println("Fluentd clone stopped.")
}
