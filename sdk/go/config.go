package yily

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout   = 5 * time.Second // defaultTimeout is the default duration for HTTP requests if not specified.
	defaultAPIPrefix = "/api/v1"       // defaultAPIPrefix is the base path for API version 1.
)

// Config holds options for the Yily client.
type Config struct {
	BaseURL       string        // BaseURL is the root URL of the Yily instance.
	EnvironmentID string        // EnvironmentID identifies the environment to read variables from.
	Token         string        // Token is the read-scoped PAT prefixed with "yily_".
	HTTPClient    *http.Client  // HTTPClient optionally configures proxies, transports, and other HTTP behavior.
	Timeout       time.Duration // Timeout limits each request; it defaults to 5 seconds.
}

// validate checks that all required configuration fields are set.
func (c *Config) validate() error {
	baseURL := strings.TrimSpace(c.BaseURL)
	if baseURL == "" {
		return ErrMissingBaseURL
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return ErrInvalidBaseURL
	}
	if !parsedURL.IsAbs() || parsedURL.Host == "" || parsedURL.User != nil ||
		(!strings.EqualFold(parsedURL.Scheme, "http") && !strings.EqualFold(parsedURL.Scheme, "https")) ||
		parsedURL.RawQuery != "" || parsedURL.ForceQuery || parsedURL.Fragment != "" {
		return ErrInvalidBaseURL
	}

	token := strings.TrimSpace(c.Token)
	if token == "" {
		return ErrMissingToken
	}
	if !strings.HasPrefix(token, "yily_") || len(token) == len("yily_") {
		return ErrInvalidToken
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

// normalizeVariableName matches the server's variable naming convention.
func normalizeVariableName(name string) string {
	return strings.Join(strings.Fields(strings.ToUpper(name)), "_")
}
