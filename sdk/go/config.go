package yily

import (
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout   = 5 * time.Second // defaultTimeout is the default duration for HTTP requests if not specified.
	defaultAPIPrefix = "/api/v1"       // defaultAPIPrefix is the base path for API version 1.
)

// Config holds the config options for the Yily client
type Config struct {
	BaseURL       string        // BaseURL is the root URL of Yily instance.
	EnvironmentID string        // EnvironmentID is the identifier of the environment to read varibles form.
	Token         string        // Token is the PAT prefixed with "yily_".
	HTTPClient    *http.Client  // HTTPClient is an optional custom HTTP client. If nil, a default client is used.
	Timeout       time.Duration // Timeout is the default timeout for HTTP requests. Defaults to 5 seconds if not set.
}

// validate checks that all required configuration fields are set.
func (c *Config) validate() error {
	if strings.TrimSpace(c.BaseURL) == "" {
		return ErrMissingBaseURL
	}

	if strings.TrimSpace(c.Token) == "" {
		return ErrMissingToken
	}

	if strings.TrimSpace(c.EnvironmentID) == "" {
		return ErrMissingEnvironmentID
	}

	return nil
}

// normalizeBaseURL ensures the base URL is trimmed and properly formatted with the /api/v1 prefix.
func normalizeBaseURL(raw string) string {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	raw = strings.TrimSuffix(raw, defaultAPIPrefix)
	return raw + defaultAPIPrefix
}
