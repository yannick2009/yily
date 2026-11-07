package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/yannick2009/yily/internal/httputil"
)

// GetFile downloads a file from path and returns its content and server-provided filename.
func (c *Client) GetFile(ctx context.Context, path string) ([]byte, string, error) {
	resp, err := c.request(ctx, http.MethodGet, c.endpoint(path), nil, "")
	if err != nil {
		return nil, "", err
	}
	if resp == nil {
		return nil, "", errors.New("empty HTTP response")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read response body: %w", err)
	}

	// A non-2xx response is an API error even when the body is valid JSON.
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, "", newAPIError(resp.StatusCode, body)
	}

	// Prefer the filename announced by the server, stripped of any path component.
	filename := ""
	if disposition := resp.Header.Get("Content-Disposition"); disposition != "" {
		if _, params, err := mime.ParseMediaType(disposition); err == nil {
			if name := params["filename"]; name != "" {
				filename = filepath.Base(name)
			}
		}
	}

	return body, filename, nil
}

// PostFile uploads filePath as multipart form data and decodes the response envelope.
func (c *Client) PostFile[T any](ctx context.Context, path, filePath string) (*httputil.HTTPResponse[T], error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("build multipart form: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("build multipart form: %w", err)
	}

	resp, err := c.request(ctx, http.MethodPost, c.endpoint(path), payload.Bytes(), writer.FormDataContentType())
	if err != nil {
		return nil, err
	}

	return decodeResponse[T](resp)
}

// writePrivateFile writes data to path with owner-only permissions.
func writePrivateFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	// WriteFile only applies perm on creation, so enforce it for existing files too.
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("secure file: %w", err)
	}
	return nil
}
