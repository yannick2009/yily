package cli

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"syscall"
)

// ErrNotAuthenticated is returned when a command needs a saved token but none exists.
var ErrNotAuthenticated = errors.New("not authenticated")

// UserError is an error caused by invalid user input rather than a failed request.
type UserError struct {
	Title   string // Title overrides the default "Invalid input" heading.
	Message string
}

// Error implements the error interface.
func (e *UserError) Error() string {
	return e.Message
}

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

// PrintInfo writes an informational message to w.
func PrintInfo(w io.Writer, message string) error {
	_, err := fmt.Fprintf(w, "%s %s\n", paint(w, ansiDim, "•"), message)
	return err
}

// PrintError writes a short, user-friendly error message to w.
func PrintError(w io.Writer, err error) {
	var apiErr *APIError
	var urlErr *url.Error
	var userErr *UserError

	switch {
	case errors.Is(err, ErrNotAuthenticated):
		printBlock(w, "Not logged in", "Run `yily-cli login --url <server> --token <token>` first.")
	case errors.As(err, &userErr):
		title := userErr.Title
		if title == "" {
			title = "Invalid input"
		}
		printBlock(w, title, userErr.Message)
	case isUsageError(err):
		printBlock(w, "Invalid input", err.Error())
	case strings.Contains(err.Error(), "required flag"):
		printBlock(w, "Missing required flag", err.Error()+"\n  Run `yily-cli <command> --help` for usage.")
	case errors.As(err, &apiErr):
		detail := apiErr.Message
		if apiErr.Code != "" {
			detail += fmt.Sprintf(" [%s]", apiErr.Code)
		}
		detail += apiErrorHint(apiErr)
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

// isUsageError reports whether err is a Cobra syntax error rather than a
// runtime failure. In that case Cobra has already printed the usage.
func isUsageError(err error) bool {
	message := err.Error()
	return strings.Contains(message, "arg(s)") ||
		strings.Contains(message, "unknown flag") ||
		strings.Contains(message, "unknown shorthand flag") ||
		strings.Contains(message, "unknown command")
}

// apiErrorHint returns an actionable hint for the most common API errors.
func apiErrorHint(err *APIError) string {
	switch {
	case err.StatusCode == 401:
		return "\n  Check your token or run `yily-cli login`."
	case err.StatusCode == 403 && err.Code == "TOKEN_WRITE_FORBIDDEN":
		return "\n  This command needs a write-scoped token."
	case err.StatusCode == 403:
		return "\n  This token does not have permission for this command."
	case err.StatusCode == 404:
		return "\n  Check the ID, or list the resource first."
	default:
		return ""
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
