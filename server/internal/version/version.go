// Package version exposes the build-time version metadata for the server and CLI.
package version

import (
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

const (
	// Dev is used when no version was injected at build time.
	Dev = "0.1.3-dev"
	// Unknown is used when a build attribute cannot be determined.
	Unknown = "unknown"
)

var (
	// Number, Commit and Date are injected with -ldflags at build time.
	Number = Dev
	Commit = ""
	Date   = ""
)

// Info is the resolved build information.
type Info struct {
	Number   string
	Commit   string
	Date     string
	Go       string
	Platform string
}

// Get resolves the build information, falling back to Go's embedded build
// metadata when fields were not injected with -ldflags.
func Get() Info {
	info := Info{
		Number:   Number,
		Commit:   Commit,
		Date:     Date,
		Go:       runtime.Version(),
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
	}

	commitInjected := info.Commit != ""
	dirty := false
	if build, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range build.Settings {
			switch setting.Key {
			case "vcs.revision":
				if info.Commit == "" {
					info.Commit = setting.Value
				}
			case "vcs.time":
				if info.Date == "" {
					info.Date = setting.Value
				}
			case "vcs.modified":
				dirty = setting.Value == "true"
			}
		}
	}

	if info.Number == "" {
		info.Number = Dev
	}
	if info.Commit == "" {
		info.Commit = Unknown
	} else {
		if len(info.Commit) > 7 {
			info.Commit = info.Commit[:7]
		}
		if dirty && !commitInjected && !strings.HasSuffix(info.Commit, "-dirty") {
			info.Commit += "-dirty"
		}
	}
	info.Date = formatDate(info.Date)

	return info
}

// formatDate accepts the Nix YYYYMMDDHHMMSS format and normalizes it to RFC3339.
func formatDate(value string) string {
	if value == "" {
		return Unknown
	}
	if len(value) == len("20060102150405") {
		if parsed, err := time.Parse("20060102150405", value); err == nil {
			return parsed.UTC().Format(time.RFC3339)
		}
	}
	return value
}
