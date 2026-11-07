package cli

import (
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/yannick2009/yily/internal/api"
)

// RecapCmd shows a numeric recap of the Yily instance.
var RecapCmd = &cobra.Command{
	Use:     "recap",
	Aliases: []string{"dashboard", "stats"},
	Short:   "Show a numeric recap of the Yily instance",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Get[api.DashboardRecapResponse](cmd.Context(), "/api/v1/dashboard")
		if err != nil {
			return err
		}

		return printRecap(cmd.OutOrStdout(), resp.Data.Stats)
	},
}

// printRecap renders the aggregate counts as an aligned, borderless block.
func printRecap(w io.Writer, stats api.DashboardStats) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	rows := [][2]string{
		{"Projects", strconv.Itoa(stats.Projects)},
		{"Environments", strconv.Itoa(stats.Environments)},
		{"Variables", strconv.Itoa(stats.Variables)},
		{"Tokens", strconv.Itoa(stats.Tokens)},
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(tw, "%s\t%s\n", row[0], row[1]); err != nil {
			return err
		}
	}

	return tw.Flush()
}
