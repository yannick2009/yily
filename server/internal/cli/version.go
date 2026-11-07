package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yannick2009/yily/internal/version"
)

// VersionCmd prints detailed build information.
var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		info := version.Get()
		if _, err := fmt.Fprintf(
			cmd.OutOrStdout(),
			"%s %s\ncommit: %s\nbuilt:  %s\ngo:     %s %s\n",
			cmd.Root().Name(),
			info.Number,
			info.Commit,
			info.Date,
			info.Go,
			info.Platform,
		); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		return nil
	},
}
