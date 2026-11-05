package cli

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"syscall"
)

// ErrNotAuthenticated is returned when a command needs a saved token but none exists.
var ErrNotAuthenticated = errors.New("not authenticated")

const (
	ansiReset = "\x1b[0m"
	ansiBold  = "\x1b[1m"
	ansiDim   = "\x1b[2m"
	ansiRed   = "\x1b[31m"
	ansiGreen = "\x1b[32m"
)

// PrintSuccess writes a successfully completed action to w.
func PrintSuccess(w io.Writer, message string) error {
	_, err := fmt.Fprintf(w, "%s %s\n", paint(w, ansiGreen, "✔"), message)
	return err
}

// PrintError writes a short, user-friendly error message to w.
func PrintError(w io.Writer, err error) {
	var apiErr *APIError
	var urlErr *url.Error

	switch {
	case errors.Is(err, ErrNotAuthenticated):
		printBlock(w, "Not logged in", "Run `yily login --url <server> --token <token>` first.")
	case errors.As(err, &apiErr):
		detail := apiErr.Message
		if apiErr.Code != "" {
			detail += fmt.Sprintf(" [%s]", apiErr.Code)
		}
		printBlock(w, fmt.Sprintf("Request failed (HTTP %d)", apiErr.StatusCode), detail)
	case errors.As(err, &urlErr) && errors.Is(urlErr.Err, syscall.ECONNREFUSED):
		printBlock(w, "Cannot reach the Yily server", urlErr.URL+"\n  Is it running?")
	case errors.As(err, &urlErr) && urlErr.Timeout():
		printBlock(w, "The Yily server did not respond in time", urlErr.URL)
	case errors.As(err, &urlErr):
		printBlock(w, "Cannot reach the Yily server", urlErr.Err.Error())
	default:
		printBlock(w, "Something went wrong", err.Error())
	}
}

// printBlock writes a red error title followed by an indented detail line.
func printBlock(w io.Writer, title, detail string) {
	_, _ = fmt.Fprintf(w, "%s %s\n", paint(w, ansiRed, "✖"), paint(w, ansiBold, title))
	if detail != "" {
		_, _ = fmt.Fprintf(w, "  %s\n", paint(w, ansiDim, detail))
	}
}

// paint wraps text in an ANSI color when w is a terminal that supports it.
func paint(w io.Writer, code, text string) string {
	if !colorEnabled(w) {
		return text
	}
	return code + text + ansiReset
}

// colorEnabled reports whether ANSI colors should be written to w.
func colorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil || info == nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
