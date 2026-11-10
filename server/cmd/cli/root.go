package main

import (
	_ "embed"

	"github.com/spf13/cobra"
	"github.com/yannick2009/yily/internal/cli"
	"github.com/yannick2009/yily/internal/version"
)

//go:embed ascii-art.txt
var asciiArt string

var rootCmd = &cobra.Command{
	Use:           "yily-cli",
	Short:         "Yily-cli is a CLI tool for interacting with the Yily server.",
	Long:          asciiArt + "\nYily-cli is a command-line interface (CLI) tool that allows users to interact with the Yily server for various operations.",
	Version:       version.Get().Number,
	SilenceUsage:  false,
	SilenceErrors: true,
}

func init() {
	rootCmd.SetVersionTemplate("{{.Name}} {{.Version}}\n")
	rootCmd.AddCommand(cli.VersionCmd)
	rootCmd.AddCommand(cli.UpgradeCmd)
	rootCmd.AddCommand(cli.UninstallCmd)
	rootCmd.AddCommand(cli.LoginCmd)
	rootCmd.AddCommand(cli.LogoutCmd)
	rootCmd.AddCommand(cli.WhoamiCmd)
	rootCmd.AddCommand(cli.ProjectsCmd)
	rootCmd.AddCommand(cli.EnvironmentsCmd)
	rootCmd.AddCommand(cli.VariablesCmd)
	rootCmd.AddCommand(cli.TagsCmd)
	rootCmd.AddCommand(cli.RecapCmd)
}
