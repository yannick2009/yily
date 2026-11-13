package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aquasecurity/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/yannick2009/yily/internal/api"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/id"
)

// resourcePrefixes maps the resource names used in CLI messages to the type
// trigram their identifiers must carry.
var resourcePrefixes = map[string]string{
	"project":     id.PrefixProject,
	"environment": id.PrefixEnvironment,
	"variable":    id.PrefixVariable,
	"version":     id.PrefixVersion,
	"tag":         id.PrefixTag,
}

// parseID validates an identifier command-line argument against the expected
// resource type and returns it as a string.
func parseID(raw, resource string) (string, error) {
	prefix, ok := resourcePrefixes[resource]
	if !ok {
		return "", &UserError{Message: fmt.Sprintf("unknown resource %q", resource)}
	}
	value, err := id.Parse(prefix, strings.TrimSpace(raw))
	if err != nil {
		return "", &UserError{Message: fmt.Sprintf("invalid %s ID %q", resource, raw)}
	}
	return value.String(), nil
}

// confirm asks a yes/no question unless yes is already true.
func confirm(cmd *cobra.Command, question string, yes bool) (bool, error) {
	if yes {
		return true, nil
	}

	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s [y/N]: ", question); err != nil {
		return false, fmt.Errorf("write prompt: %w", err)
	}

	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

// printProjectsTable renders the project list as a terminal table.
func printProjectsTable(w io.Writer, projects []api.ProjectSummaryResponse) {
	t := newTable(w)
	t.SetHeaders("ID", "NAME", "ENVIRONMENTS", "TAGS", "CREATED")

	alignments := []table.Alignment{
		table.AlignLeft,
		table.AlignLeft,
		table.AlignRight,
		table.AlignRight,
		table.AlignLeft,
	}
	t.SetAlignment(alignments...)
	t.SetHeaderAlignment(alignments...)

	for _, project := range projects {
		t.AddRow(
			project.ID.String(),
			project.Name,
			strconv.Itoa(project.EnvironmentCount),
			strconv.Itoa(project.TagCount),
			formatTime(project.CreatedAt),
		)
	}

	t.Render()
}

// printProjectTable renders a single project as a field/value table.
func printProjectTable(w io.Writer, project model.Project) {
	t := newTable(w)
	t.SetHeaders("FIELD", "VALUE")

	t.AddRow("ID", project.ID.String())
	t.AddRow("Name", project.Name)
	t.AddRow("Created", formatTime(project.CreatedAt))
	t.AddRow("Updated", formatTime(project.UpdatedAt))

	t.Render()
}

// newTable creates a table with the styling shared by all CLI tables.
func newTable(w io.Writer) *table.Table {
	t := table.New(w)
	t.SetRowLines(true)
	t.SetBorders(true)
	t.SetDividers(table.UnicodeRoundedDividers)
	if colorEnabled(w) {
		t.SetHeaderStyle(table.StyleBold)
	}
	return t
}

// formatTime renders API timestamps in the local timezone.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Local().Format("2006-01-02 15:04")
}

// readSecret reads a secret without echo when stdin is a terminal, or from a
// pipe otherwise.
func readSecret(cmd *cobra.Command, label string) (string, error) {
	in := cmd.InOrStdin()
	if file, ok := in.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "%s: ", label); err != nil {
			return "", fmt.Errorf("write prompt: %w", err)
		}
		data, err := term.ReadPassword(int(file.Fd()))
		_, _ = fmt.Fprintln(cmd.ErrOrStderr())
		if err != nil {
			return "", fmt.Errorf("read secret: %w", err)
		}
		return string(data), nil
	}

	data, err := io.ReadAll(in)
	if err != nil {
		return "", fmt.Errorf("read secret: %w", err)
	}
	return strings.TrimRight(string(data), "\r\n"), nil
}
