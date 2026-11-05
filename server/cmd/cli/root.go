package main

import (
	_ "embed"

	"github.com/spf13/cobra"
	"github.com/yannick2009/yily/internal/cli"
)

//go:embed ascii-art.txt
var asciiArt string

var rootCmd = &cobra.Command{
	Use:           "yily",
	Short:         "Yily is a CLI tool for interacting with the Yily server.",
	Long:          asciiArt + "\nYily is a command-line interface (CLI) tool that allows users to interact with the Yily server for various operations.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(cli.LoginCmd)
	rootCmd.AddCommand(cli.LogoutCmd)
	rootCmd.AddCommand(cli.WhoamiCmd)
}
