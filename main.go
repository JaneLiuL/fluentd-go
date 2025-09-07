package main

import (
	"os"

	"github.com/JaneLiuL/fluentd-go/cmd"
)

func main() {
	rootCmd := cmd.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}

}
