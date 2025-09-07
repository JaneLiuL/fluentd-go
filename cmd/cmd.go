package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/JaneLiuL/fluentd-go/pkg/plugin"
	"github.com/spf13/cobra"
)

var (
	inputFile string
	netAdress string
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "fluentd-go",
		Short: "fluentd written in go",
		Long:  `fluentd written in go, support input and output and filter`,
		Run: func(cmd *cobra.Command, args []string) {
			if inputFile == "" {
				fmt.Println("wrong, please use --input or -i")
			}
			if netAdress == "" {
				fmt.Println("please use --address or -a")
			}
			run(inputFile, netAdress)
		},
	}
	rootCmd.Flags().StringVarP(&inputFile, "input", "i", "/var/log/app.log", "intput file name")
	rootCmd.Flags().StringVarP(&netAdress, "address", "a", "0.0.0.0:24224", "network address")
	return rootCmd
}

func run(inputFile, netAdress string) {
	// 创建Fluentd实例
	fluent := plugin.NewFluentd()

	// 设置队列连接各组件
	inputQueue := plugin.NewQueue(1000)
	filterQueue := plugin.NewQueue(1000)
	outputQueue := plugin.NewQueue(1000)

	// 添加输入插件
	fileInput := plugin.NewTailInput("app.log", inputQueue, inputFile, "app_log.pos")
	tcpInput := plugin.NewTcpInput("network.log", inputQueue, netAdress)
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
