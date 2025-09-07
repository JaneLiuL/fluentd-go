package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/JaneLiuL/fluentd-go/pkg/config"
	"github.com/JaneLiuL/fluentd-go/pkg/plugin"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	positionFile string
	configFile   string
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "fluentd-go",
		Short: "fluentd written in go",
		Long:  `fluentd written in go, support input and output and filter`,
		Run: func(cmd *cobra.Command, args []string) {
			if configFile == "" {
				fmt.Println("wrong, please use configFile or -c")
			}
			if positionFile == "" {
				fmt.Println("please use --positionFile or -p")
			}
			// if outputFile == "" {
			// 	fmt.Println("please use --outputFile or -o")
			// }
			// if filterKeyWord == "" {
			// 	fmt.Println("please use --filterKeyWord or -f")
			// }

			run(positionFile, configFile)
		},
	}
	rootCmd.Flags().StringVarP(&positionFile, "positionFile", "i", "/tmp/app_log.pos", "position file name")
	// rootCmd.Flags().StringVarP(&netAdress, "address", "a", "0.0.0.0:24224", "input network address reading from connection")
	// rootCmd.Flags().StringVarP(&outputFile, "output", "o", "/tmp/filtered_errors.log", "output file name")
	// rootCmd.Flags().StringVarP(&filterKeyWord, "filterKeyWord", "f", "Sender", "filter key word")
	rootCmd.Flags().StringVarP(&configFile, "configFile", "c", "./pkg/config/config.yaml", "config file")

	return rootCmd
}

func run(positionFile, path string) {
	fluent := plugin.NewFluentd()

	inputQueue := plugin.NewQueue(1000)
	// filterQueue := plugin.NewQueue(1000)
	outputQueue := plugin.NewQueue(1000)

	configFile, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("load config fail: %v", err)
	}

	for _, input := range configFile.Input {
		switch input.Type {
		case "file":
			fileInput := plugin.NewTailInput(input.Tag, inputQueue, input.Path, positionFile)
			fluent.AddInput(fileInput)
		case "tcp":
			tcpInput := plugin.NewTcpInput(input.Tag, inputQueue, input.Address)
			fluent.AddInput(tcpInput)
		default:
			log.Printf("not support type: %s", input.Type)
		}
	}

	for _, filter := range configFile.Filters {
		switch filter.Type {
		case "match":
			grepFilter := plugin.NewGrepFilter(inputQueue, outputQueue, filter.Tag, "message", filter.Pattern, false)
			fluent.AddFilter(grepFilter)
		case "exclude":
			grepFilter := plugin.NewGrepFilter(inputQueue, outputQueue, filter.Tag, "message", filter.Pattern, true)
			fluent.AddFilter(grepFilter)
		}
	}

	// transformFilter := plugin.NewRecordTransformerFilter(filterQueue, outputQueue, []string{"app.log", "network.log"},
	// 	map[string]interface{}{"environment": "production", "source": "fluentd-go"},
	// 	[]string{},
	// )

	// fluent.AddFilter(transformFilter)

	for _, outout := range configFile.Output {
		switch outout.Type {
		case "stdout":
			stdoutOutput := plugin.NewStdoutOutput(outputQueue, outout.Tag, 10, 5)
			fluent.AddOutput(stdoutOutput)
		case "file":
			fileOutput := plugin.NewFileOutput(outputQueue, outout.Tag, outout.Path, 10, 5, outout.Compression)
			fluent.AddOutput(fileOutput)
		case "elasticsearch":
			// TODO
		}
	}
	fluent.Start()
	log.Println("Fluentd clone is running. Press Ctrl+C to stop.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fluent.Stop()
	log.Println("Fluentd clone stopped.")
}

func loadConfig(path string) (*config.Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config config.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
