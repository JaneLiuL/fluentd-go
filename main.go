package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/JaneLiuL/fluentd-go/pkg/plugin"
)

func main() {
	// 创建Fluentd实例
	fluent := plugin.NewFluentd()

	// 设置队列连接各组件
	inputQueue := plugin.NewQueue(1000)
	filterQueue := plugin.NewQueue(1000)
	outputQueue := plugin.NewQueue(1000)

	// 添加输入插件
	fileInput := plugin.NewTailInput("app.log", inputQueue, "/var/log/app.log", "app_log.pos")
	tcpInput := plugin.NewTcpInput("network.log", inputQueue, "0.0.0.0:24224")
	fluent.AddInput(fileInput)
	fluent.AddInput(tcpInput)

	// 添加过滤插件
	grepFilter := plugin.NewGrepFilter(inputQueue, filterQueue, []string{"app.log", "network.log"}, "message", "error", false)
	transformFilter := plugin.NewRecordTransformerFilter(filterQueue, outputQueue, []string{"app.log", "network.log"},
		map[string]interface{}{"environment": "production", "source": "fluentd-go"},
		[]string{},
	)
	fluent.AddFilter(grepFilter)
	fluent.AddFilter(transformFilter)

	// 添加输出插件
	stdoutOutput := plugin.NewStdoutOutput(outputQueue, []string{"app.log", "network.log"}, 10, 5)
	fileOutput := plugin.NewFileOutput(outputQueue, []string{"app.log", "network.log"}, "/var/log/filtered_errors.log", 10, 5, true)
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
