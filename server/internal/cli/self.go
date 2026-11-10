package cli

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"

	"github.com/yannick2009/yily/internal/version"
)

var (
	upgradeVersion    string
	upgradePreRelease bool
	uninstallYes      bool
	uninstallPurge    bool
)

// UpgradeCmd replaces the running binary with a release downloaded from GitHub.
var UpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade yily-cli to the latest release",
	Long: `Upgrade the yily-cli binary in place from releases.

By default the latest stable release is installed. Use --pre-release to
include prereleases, or --version to install a specific version.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true

		exePath, err := executablePath()
		if err != nil {
			return err
		}

		rel, err := resolveRelease(cmd.Context(), upgradeVersion, upgradePreRelease)
		if err != nil {
			return &UserError{Title: "Upgrade failed", Message: err.Error()}
		}

		current := version.Get().Number
		latest := rel.Version()

		if upgradeVersion == "" {
			cmp := compareVersions(current, latest)
			if cmp == 0 {
				return PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("yily-cli is already up to date (%s).", current))
			}
			if cmp > 0 {
				message := fmt.Sprintf("Installed version (%s) is newer than the latest release (%s).", current, latest)
				if !upgradePreRelease {
					message += " Use --pre-release to include prereleases."
				}
				return PrintInfo(cmd.OutOrStdout(), message)
			}
		}

		if err := PrintInfo(cmd.OutOrStdout(), fmt.Sprintf("Downloading yily-cli %s for %s/%s...", latest, runtime.GOOS, runtime.GOARCH)); err != nil {
			return fmt.Errorf("write output: %w", err)
		}

		if err := installRelease(cmd.Context(), exePath, rel); err != nil {
			return &UserError{Title: "Upgrade failed", Message: err.Error()}
		}

		if current == latest {
			return PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Reinstalled yily-cli %s.", latest))
		}
		return PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Upgraded yily-cli %s → %s.", current, latest))
	},
}

// UninstallCmd removes the running binary from the system.
var UninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the yily-cli binary",
	Long: `Remove the yily-cli binary from this system.

The saved login configuration is kept unless --purge is used. The PATH entry
added by the install script, if any, is left untouched.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true

		exePath, err := executablePath()
		if err != nil {
			return err
		}

		ok, err := confirm(cmd, fmt.Sprintf("Remove %s?", exePath), uninstallYes)
		if err != nil {
			return err
		}
		if !ok {
			return PrintInfo(cmd.OutOrStdout(), "Uninstall cancelled.")
		}

		if err := removeExecutable(exePath); err != nil {
			return &UserError{Title: "Uninstall failed", Message: err.Error()}
		}
		if err := PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Removed %s.", exePath)); err != nil {
			return fmt.Errorf("write output: %w", err)
		}

		if uninstallPurge {
			if err := CleanupConfigFile(); err != nil {
				return fmt.Errorf("remove configuration: %w", err)
			}
			return PrintSuccess(cmd.OutOrStdout(), "Removed the saved configuration.")
		}

		if path, err := configPath(); err == nil {
			return PrintInfo(cmd.OutOrStdout(), fmt.Sprintf("Saved configuration kept in %s (use --purge to remove it).", path))
		}
		return nil
	},
}

// compareVersions returns -1, 0 or 1 like semver.Compare. Invalid versions
// (such as development builds) are treated as older to allow the upgrade.
func compareVersions(current, latest string) int {
	if !strings.HasPrefix(current, "v") {
		current = "v" + current
	}
	if !strings.HasPrefix(latest, "v") {
		latest = "v" + latest
	}
	if !semver.IsValid(current) || !semver.IsValid(latest) {
		return -1
	}
	return semver.Compare(current, latest)
}

func init() {
	UpgradeCmd.Flags().StringVar(&upgradeVersion, "version", "", "version to install (default: latest stable)")
	UpgradeCmd.Flags().BoolVar(&upgradePreRelease, "pre-release", false, "include prereleases when resolving the latest version")

	UninstallCmd.Flags().BoolVarP(&uninstallYes, "yes", "y", false, "do not ask for confirmation")
	UninstallCmd.Flags().BoolVar(&uninstallPurge, "purge", false, "also remove the saved login configuration")
}
