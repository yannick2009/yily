package main

import "os"

// entry point for the CLI application.
func main() {
	if err := rootCmd.Execute(); err != nil {
		// Cobra already printed the error to stderr.
		os.Exit(1)
	}
}
