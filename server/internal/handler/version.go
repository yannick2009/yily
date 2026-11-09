package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/yannick2009/yily/internal/api"
	"github.com/yannick2009/yily/internal/httputil"
	"github.com/yannick2009/yily/internal/version"
	"golang.org/x/mod/semver"
)

const (
	updateRepository = "yannick2009/yily"
	updateCacheTTL   = 30 * time.Minute
)

// VersionHandler defines the interface for serving the server version and updates.
type VersionHandler interface {
	Get(c *echo.Context) error
	GetUpdate(c *echo.Context) error
}

// versionHandler handles HTTP requests for the server version and updates.
type versionHandler struct {
	client *http.Client

	mu        sync.Mutex
	latest    string
	checkedAt time.Time
}

// NewVersionHandler creates a new VersionHandler.
func NewVersionHandler() VersionHandler {
	return &versionHandler{
		client: &http.Client{Timeout: 3 * time.Second},
	}
}

// Get handles GET /version.
func (h *versionHandler) Get(c *echo.Context) error {
	return httputil.NewHTTPResponse(c, http.StatusOK, "Version retrieved successfully", api.VersionResponse{
		Version: version.Get().Number,
	})
}

// GetUpdate handles GET /version/update.
func (h *versionHandler) GetUpdate(c *echo.Context) error {
	tag, err := h.latestTag(c.Request().Context())
	if err != nil {
		return httputil.NewHTTPResponse(c, http.StatusOK, "Update information unavailable", api.UpdateResponse{})
	}

	current := normalizedVersion(version.Get().Number)
	latest := normalizedVersion(tag)

	return httputil.NewHTTPResponse(c, http.StatusOK, "Update information retrieved successfully", api.UpdateResponse{
		LatestVersion:   strings.TrimPrefix(tag, "v"),
		ReleaseURL:      fmt.Sprintf("https://github.com/%s/tree/%s", updateRepository, url.PathEscape(tag)),
		UpdateAvailable: current != "" && semver.Compare(latest, current) > 0,
	})
}

// latestTag returns the latest tag from the GitHub API for the update repository, using a cached value if available and not expired.
func (h *versionHandler) latestTag(ctx context.Context) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.latest != "" && time.Since(h.checkedAt) < updateCacheTTL {
		return h.latest, nil
	}

	tag, err := h.fetchLatestTag(ctx)
	if err != nil {
		return "", err
	}

	h.latest = tag
	h.checkedAt = time.Now()

	return tag, nil
}

// fetchLatestTag fetches the latest tag from the GitHub API for the update repository.
func (h *versionHandler) fetchLatestTag(ctx context.Context) (string, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/tags?per_page=1", updateRepository)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "yily-update-check")

	resp, err := h.client.Do(req)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", errors.New("github tags: empty response")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github tags: unexpected status %d", resp.StatusCode)
	}

	var tags []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&tags); err != nil {
		return "", err
	}
	if len(tags) == 0 {
		return "", errors.New("github tags: no tags found")
	}

	tag := strings.TrimSpace(tags[0].Name)
	if normalizedVersion(tag) == "" {
		return "", errors.New("github tags: invalid version")
	}

	return tag, nil
}

// normalizedVersion normalizes a version string to a valid semantic version format.
func normalizedVersion(value string) string {
	value = strings.TrimSpace(value)
	// Dev builds do not have a meaningful release version to compare.
	if value == "" || value == version.Dev {
		return ""
	}
	if !strings.HasPrefix(value, "v") {
		value = "v" + value
	}
	if !semver.IsValid(value) {
		return ""
	}
	return value
}
