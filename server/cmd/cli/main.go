package main

import (
	"os"

	"github.com/yannick2009/yily/internal/cli"
)

// entry point for the CLI application.
func main() {
	if err := rootCmd.Execute(); err != nil {
		cli.PrintError(os.Stderr, err)
		os.Exit(1)
	}
}
