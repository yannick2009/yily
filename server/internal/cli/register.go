package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	baseURL string
	token   string
)

// tokenInfo is the subset of the token introspection response used by login.
type tokenInfo struct {
	Name      string     `json:"name"`
	Scope     string     `json:"scope"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// LoginCmd is a Cobra command that authenticates with a Yily server using a personal access token.
var LoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with a Yily server",
	Long: `Authenticate with a Yily server using a personal access token.

The token is validated by the server and, if accepted, saved to the
configuration file for future commands. Subsequent CLI commands then run
with the permissions granted to that token.`,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		rawToken := strings.TrimSpace(token)
		client := NewClient(baseURL, rawToken)

		resp, err := client.Get[tokenInfo](cmd.Context(), "/api/v1/tokens/me")
		if err != nil {
			return err
		}

		cfg := &Config{
			BaseURL:  client.BaseURL,
			Token:    rawToken,
			Scope:    resp.Data.Scope,
			ExpireAt: resp.Data.ExpiresAt,
		}
		if err := WriteConfigFile(cfg); err != nil {
			return fmt.Errorf("save configuration: %w", err)
		}

		if err := PrintSuccess(cmd.OutOrStdout(), "Authenticated successfully."); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		return nil
	},
}

// LogoutCmd is a Cobra command that removes the saved authentication token from the configuration file.
var LogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove the saved authentication token",
	Long:  `Remove the saved authentication token from the configuration file.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := CleanupConfigFile(); err != nil {
			return fmt.Errorf("remove configuration: %w", err)
		}
		if err := PrintSuccess(cmd.OutOrStdout(), "Logged out successfully."); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		return nil
	},
}

// WhoamiCmd is a Cobra command that displays the current authenticated user based on the saved authentication token.
var WhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display the current authenticated user",
	Long:  `Display the current authenticated user based on the saved authentication token.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}
		resp, err := client.Get[tokenInfo](cmd.Context(), "/api/v1/tokens/me")
		if err != nil {
			return err
		}

		expires := "never"
		if resp.Data.ExpiresAt != nil {
			expires = resp.Data.ExpiresAt.Local().Format("2006-01-02 15:04:05")
		}

		if err := PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Authenticated as %q (scope: %s)", resp.Data.Name, resp.Data.Scope)); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  Expires: %s\n", expires); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		return nil
	},
}

// init initializes the command-line flags for the login command.
func init() {
	LoginCmd.Flags().StringVarP(&baseURL, "url", "u", "", "Base URL of the Yily server (e.g., https://yily.example.com)")
	LoginCmd.Flags().StringVarP(&token, "token", "t", "", "Personal access token to authenticate with the Yily server")
	_ = LoginCmd.MarkFlagRequired("url")
	_ = LoginCmd.MarkFlagRequired("token")
}
