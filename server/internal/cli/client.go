package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yannick2009/yily/internal/httputil"
)

// Client is a minimal HTTP client for the Yily API.
type Client struct {
	httpClient *http.Client
	BaseURL    string
	token      string
}

// NewClient creates a Client that authenticates every request with token.
func NewClient(baseURL, token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		BaseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		token:      strings.TrimSpace(token),
	}
}

// APIError describes a non-2xx response returned by the Yily API.
type APIError struct {
	StatusCode int    // StatusCode is the HTTP status code of the response.
	Code       string // Code is the stable machine-readable error code, when provided.
	Message    string // Message is the human-readable error message.
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s (HTTP %d, code %s)", e.Message, e.StatusCode, e.Code)
	}
	return fmt.Sprintf("%s (HTTP %d)", e.Message, e.StatusCode)
}

// Get sends a GET request to path and decodes the standard Yily response envelope.
// T is the type of the envelope's data field.
func (c *Client) Get[T any](ctx context.Context, path string) (*httputil.HTTPResponse[T], error) {
	return c.do[T](ctx, http.MethodGet, path, nil)
}

// Post sends a POST request with a JSON body to path.
func (c *Client) Post[T any](ctx context.Context, path string, body any) (*httputil.HTTPResponse[T], error) {
	return c.do[T](ctx, http.MethodPost, path, body)
}

// Patch sends a PATCH request with a JSON body to path.
func (c *Client) Patch[T any](ctx context.Context, path string, body any) (*httputil.HTTPResponse[T], error) {
	return c.do[T](ctx, http.MethodPatch, path, body)
}

// Put sends a PUT request with a JSON body to path.
func (c *Client) Put[T any](ctx context.Context, path string, body any) (*httputil.HTTPResponse[T], error) {
	return c.do[T](ctx, http.MethodPut, path, body)
}

// Delete sends a DELETE request to path.
func (c *Client) Delete[T any](ctx context.Context, path string) (*httputil.HTTPResponse[T], error) {
	return c.do[T](ctx, http.MethodDelete, path, nil)
}

// do encodes the optional body, sends the request, and decodes the response.
func (c *Client) do[T any](ctx context.Context, method, path string, body any) (*httputil.HTTPResponse[T], error) {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
	}

	contentType := ""
	if payload != nil {
		contentType = "application/json"
	}

	resp, err := c.request(ctx, method, c.endpoint(path), payload, contentType)
	if err != nil {
		return nil, err
	}

	return decodeResponse[T](resp)
}

// decodeResponse turns an HTTP response into the standard Yily envelope.
func decodeResponse[T any](resp *http.Response) (*httputil.HTTPResponse[T], error) {
	if resp == nil {
		return nil, errors.New("empty HTTP response")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// A non-2xx response is an API error even when the JSON envelope is valid.
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, newAPIError(resp.StatusCode, body)
	}

	result := new(httputil.HTTPResponse[T])
	if len(body) > 0 {
		if err := json.Unmarshal(body, result); err != nil {
			return nil, fmt.Errorf("decode response body: %w", err)
		}
	}

	return result, nil
}

// endpoint joins the configured base URL with an API path.
func (c *Client) endpoint(path string) string {
	return c.BaseURL + "/" + strings.TrimLeft(path, "/")
}

// request builds and sends an authenticated HTTP request.
func (c *Client) request(ctx context.Context, method, url string, body []byte, contentType string) (*http.Response, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("empty HTTP response")
	}

	return resp, nil
}

// newAPIError builds an APIError from a non-2xx response body. When the body is
// not a standard error envelope, it falls back to the HTTP status text.
func newAPIError(status int, body []byte) error {
	var envelope httputil.HTTPResponse[any]

	message := ""
	if err := json.Unmarshal(body, &envelope); err == nil {
		message = envelope.Message
	}
	if message == "" {
		message = http.StatusText(status)
	}
	if message == "" {
		message = "unexpected HTTP status"
	}

	return &APIError{StatusCode: status, Code: envelope.Code, Message: message}
}
