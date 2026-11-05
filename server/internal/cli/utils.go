package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/aquasecurity/table"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// parseID validates a UUID command-line argument and returns its canonical form.
func parseID(raw, resource string) (string, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", &UserError{Message: fmt.Sprintf("invalid %s ID %q", resource, raw)}
	}
	return id.String(), nil
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
func printProjectsTable(w io.Writer, projects []ProjectSummary) {
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
func printProjectTable(w io.Writer, project Project) {
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
