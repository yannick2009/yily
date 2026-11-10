package cli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	repository   = "yannick2009/yily"       // Official GitHub repository.
	binaryName   = "yily-cli"               // Name of the binary.
	apiBase      = "https://api.github.com" // GitHub API base URL.
	downloadBase = "https://github.com"     // GitHub download base URL.
)

var httpClient = &http.Client{
	Timeout: 5 * time.Minute,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return errors.New("refusing non-HTTPS redirect")
		}
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		return nil
	},
}

// release is a resolved GitHub release.
type release struct {
	Tag string // Tag is the release tag, for example "v1.0.0".
}

// Version returns the tag without its leading "v".
func (r release) Version() string {
	return strings.TrimPrefix(strings.ToLower(r.Tag), "v")
}

// resolveRelease returns the requested release. When version is empty, it
// fetches the latest release from GitHub, optionally including prereleases.
func resolveRelease(ctx context.Context, version string, preRelease bool) (release, error) {
	if version != "" {
		version = strings.TrimPrefix(strings.TrimSpace(version), "v")
		if err := validateVersion(version); err != nil {
			return release{}, err
		}
		return release{Tag: "v" + version}, nil
	}

	url := apiBase + "/repos/" + repository + "/releases/latest"
	if preRelease {
		url = apiBase + "/repos/" + repository + "/releases?per_page=1"
	}

	body, err := get(ctx, url)
	if err != nil {
		if status, ok := errors.AsType[*statusError](err); ok {
			switch status.Code {
			case http.StatusNotFound:
				if preRelease {
					return release{}, errors.New("no release found")
				}
				return release{}, errors.New("no stable release found (use --pre-release)")
			case http.StatusForbidden, http.StatusTooManyRequests:
				return release{}, errors.New("GitHub API rate limit reached; retry later or use --version")
			}
		}
		return release{}, fmt.Errorf("resolve latest release: %w", err)
	}

	var tag string
	if preRelease {
		var releases []struct {
			TagName string `json:"tag_name"`
		}
		if err := json.Unmarshal(body, &releases); err != nil {
			return release{}, fmt.Errorf("decode releases: %w", err)
		}
		if len(releases) == 0 {
			return release{}, errors.New("no release found")
		}
		tag = releases[0].TagName
	} else {
		var item struct {
			TagName string `json:"tag_name"`
		}
		if err := json.Unmarshal(body, &item); err != nil {
			return release{}, fmt.Errorf("decode release: %w", err)
		}
		tag = item.TagName
	}

	version = strings.TrimPrefix(strings.TrimSpace(tag), "v")
	if err := validateVersion(version); err != nil {
		return release{}, fmt.Errorf("invalid release tag %q", tag)
	}
	return release{Tag: "v" + version}, nil
}

// installRelease downloads rel for the current platform, verifies its
// checksum, and atomically replaces the binary at exePath.
func installRelease(ctx context.Context, exePath string, rel release) error {
	label, err := assetLabel(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	version := rel.Version()
	asset := fmt.Sprintf("%s_%s_%s.tar.gz", binaryName, version, label)
	base := downloadBase + "/" + repository + "/releases/download/" + rel.Tag

	tmpDir, err := os.MkdirTemp("", "yily-upgrade-")
	if err != nil {
		return fmt.Errorf("create temporary directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	archivePath := filepath.Join(tmpDir, asset)
	if err := download(ctx, base+"/"+asset, archivePath); err != nil {
		return fmt.Errorf("download %s: %w", asset, err)
	}

	checksumsPath := filepath.Join(tmpDir, "checksums.txt")
	if err := download(ctx, base+"/checksums.txt", checksumsPath); err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}

	if err := verifyChecksum(archivePath, checksumsPath, asset); err != nil {
		return err
	}

	binary, err := extractBinary(archivePath, filepath.Dir(exePath))
	if err != nil {
		return err
	}

	return replaceExecutable(binary, exePath)
}

// assetLabel maps a Go platform to the release archive label.
func assetLabel(goos, goarch string) (string, error) {
	switch goos {
	case "linux":
		switch goarch {
		case "amd64", "arm64":
			return "linux_" + goarch, nil
		case "arm":
			return "linux_armv7", nil
		}
	case "darwin":
		if goarch == "amd64" || goarch == "arm64" {
			return "darwin_" + goarch, nil
		}
	}
	return "", fmt.Errorf("unsupported platform %s/%s; download the release manually", goos, goarch)
}

// validateVersion accepts the version characters used by release tags.
func validateVersion(version string) error {
	if version == "" {
		return errors.New("version cannot be empty")
	}
	if version[0] < '0' || version[0] > '9' {
		return fmt.Errorf("invalid version %q", version)
	}
	for _, r := range version {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '.' || r == '-' {
			continue
		}
		return fmt.Errorf("invalid version %q", version)
	}
	return nil
}

// statusError is an unexpected HTTP status returned by the GitHub API.
type statusError struct {
	Code int
	URL  string
}

// Error implements the error interface.
func (e *statusError) Error() string {
	return fmt.Sprintf("GET %s: HTTP %d", e.URL, e.Code)
}

// get sends a JSON request to the GitHub API.
func get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "yily-cli")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("empty response")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, &statusError{Code: resp.StatusCode, URL: url}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return body, nil
}

// download streams url into dest.
func download(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "yily-cli")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp == nil {
		return errors.New("empty response")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	file, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}
	closed = true
	return nil
}

// verifyChecksum compares the archive hash with the entry published in checksums.txt.
func verifyChecksum(archivePath, checksumsPath, asset string) error {
	data, err := os.ReadFile(checksumsPath)
	if err != nil {
		return fmt.Errorf("read checksums: %w", err)
	}

	expected, err := findChecksum(data, asset)
	if err != nil {
		return err
	}

	actual, err := fileSHA256(archivePath)
	if err != nil {
		return err
	}

	if !strings.EqualFold(expected, actual) {
		return fmt.Errorf("checksum mismatch for %s", asset)
	}
	return nil
}

// findChecksum returns the hash listed for asset in a sha256sum-style file.
func findChecksum(data []byte, asset string) (string, error) {
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "./")
		name = strings.TrimPrefix(name, "*")
		if name == asset {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksum for %s not found", asset)
}

// fileSHA256 returns the lowercase hex SHA-256 of path.
func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// extractBinary writes the yily-cli entry from a tar.gz archive into destDir.
func extractBinary(archivePath, destDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("open archive: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("read archive: %w", err)
	}
	defer func() {
		_ = gz.Close()
	}()

	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read archive: %w", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(header.Name) != binaryName {
			continue
		}
		return writeExecutable(reader, destDir)
	}

	return "", fmt.Errorf("%s not found in archive", binaryName)
}

// writeExecutable writes source to a temporary executable file in dir.
func writeExecutable(source io.Reader, dir string) (string, error) {
	file, err := os.CreateTemp(dir, ".yily-cli-upgrade-")
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return "", fmt.Errorf("cannot write to %s (permission denied); reinstall with the install script or your package manager", dir)
		}
		return "", fmt.Errorf("create temporary binary: %w", err)
	}

	path := file.Name()
	remove := true
	defer func() {
		if remove {
			_ = os.Remove(path)
		}
	}()

	if _, err := io.Copy(file, source); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("write binary: %w", err)
	}
	if err := file.Chmod(0o755); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("make binary executable: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("flush binary: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close binary: %w", err)
	}

	remove = false
	return path, nil
}
