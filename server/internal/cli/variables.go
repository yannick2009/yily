package cli

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"uuid"

	"github.com/aquasecurity/table"
	"github.com/spf13/cobra"

	"github.com/yannick2009/yily/internal/api"
	"github.com/yannick2009/yily/internal/domain/model"
)

var (
	variableSearch       string
	variableTagFilter    string
	variableSort         string
	variableOrder        string
	variableCreateValue  string
	variableVersionValue string
	variableDeleteYes    bool
	variableClearYes     bool
)

// VariablesCmd groups all variable-related commands.
var VariablesCmd = &cobra.Command{
	Use:     "variables",
	Aliases: []string{"variable", "var"},
	Short:   "Manage Yily variables and their versions",
}

// variablesListCmd lists the variables of an environment.
var variablesListCmd = &cobra.Command{
	Use:     "list <environment-id>",
	Aliases: []string{"ls"},
	Short:   "List an environment's variables",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		environmentID, err := parseID(args[0], "environment")
		if err != nil {
			return err
		}

		sortValue := strings.ToLower(strings.TrimSpace(variableSort))
		switch sortValue {
		case "name", "created_at":
		default:
			return &UserError{Message: fmt.Sprintf("unsupported sort %q (use name or created_at)", variableSort)}
		}
		orderValue := strings.ToLower(strings.TrimSpace(variableOrder))
		switch orderValue {
		case "asc", "desc":
		default:
			return &UserError{Message: fmt.Sprintf("unsupported order %q (use asc or desc)", variableOrder)}
		}

		params := url.Values{}
		params.Set("sort", sortValue)
		params.Set("order", orderValue)
		if search := strings.TrimSpace(variableSearch); search != "" {
			params.Set("search", search)
		}
		if raw := strings.TrimSpace(variableTagFilter); raw != "" {
			tagIDs, err := parseTagIDs(strings.Split(raw, ","))
			if err != nil {
				return err
			}
			parts := make([]string, 0, len(tagIDs))
			for _, id := range tagIDs {
				parts = append(parts, id.String())
			}
			params.Set("tag_ids", strings.Join(parts, ","))
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Get[[]model.Variable](cmd.Context(), "/api/v1/environments/"+environmentID+"/variables?"+params.Encode())
		if err != nil {
			return err
		}
		if len(resp.Data) == 0 {
			return PrintInfo(cmd.OutOrStdout(), "No variables found.")
		}

		printVariablesTable(cmd.OutOrStdout(), resp.Data)
		return nil
	},
}

// variablesGetCmd shows a single variable's details.
var variablesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show a variable",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0], "variable")
		if err != nil {
			return err
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Get[model.Variable](cmd.Context(), "/api/v1/variables/"+id)
		if err != nil {
			return err
		}

		printVariableTable(cmd.OutOrStdout(), resp.Data)
		return nil
	},
}

// variablesValueCmd prints a variable's current decrypted value.
var variablesValueCmd = &cobra.Command{
	Use:     "value <id>",
	Aliases: []string{"reveal"},
	Short:   "Print a variable's current value",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0], "variable")
		if err != nil {
			return err
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Get[api.VariableValueResponse](cmd.Context(), "/api/v1/variables/"+id+"/value")
		if err != nil {
			return err
		}

		// Print the raw value so it can be piped into other commands.
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), resp.Data.Value); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		return nil
	},
}

// variablesCreateCmd creates a variable in an environment.
var variablesCreateCmd = &cobra.Command{
	Use:   "create <environment-id> <name>",
	Short: "Create a variable",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		environmentID, err := parseID(args[0], "environment")
		if err != nil {
			return err
		}
		name := strings.TrimSpace(args[1])
		if name == "" {
			return &UserError{Message: "variable name cannot be empty"}
		}

		value, err := resolveVariableValue(cmd, variableCreateValue)
		if err != nil {
			return err
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Post[model.Variable](cmd.Context(), "/api/v1/environments/"+environmentID+"/variables", api.CreateVariableRequest{
			Name:  name,
			Value: value,
		})
		if err != nil {
			return err
		}

		return PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Variable %q created (id: %s).", resp.Data.Name, resp.Data.ID))
	},
}

// variablesSetNameCmd sets a variable's name.
var variablesSetNameCmd = &cobra.Command{
	Use:     "set-name <id> <name>",
	Aliases: []string{"rename", "update"},
	Short:   "Set a variable's name",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0], "variable")
		if err != nil {
			return err
		}
		name := strings.TrimSpace(args[1])
		if name == "" {
			return &UserError{Message: "variable name cannot be empty"}
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		if _, err := client.Patch[any](cmd.Context(), "/api/v1/variables/"+id, api.UpdateVariableRequest{Name: name}); err != nil {
			return err
		}

		return PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Variable name set to %q.", name))
	},
}

// variablesSetTagsCmd replaces a variable's tags.
var variablesSetTagsCmd = &cobra.Command{
	Use:   "set-tags <id> [tag-id...]",
	Short: "Replace a variable's tags",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0], "variable")
		if err != nil {
			return err
		}

		tagIDs, err := parseTagIDs(args[1:])
		if err != nil {
			return err
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		if _, err := client.Put[any](cmd.Context(), "/api/v1/variables/"+id+"/tags", api.SetVariableTagsRequest{TagIDs: tagIDs}); err != nil {
			return err
		}

		if len(tagIDs) == 0 {
			return PrintSuccess(cmd.OutOrStdout(), "Variable tags cleared.")
		}
		return PrintSuccess(cmd.OutOrStdout(), "Variable tags updated.")
	},
}

// variablesClearValueCmd clears a variable's current value.
var variablesClearValueCmd = &cobra.Command{
	Use:     "clear-value <id>",
	Aliases: []string{"unset"},
	Short:   "Clear a variable's current value",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0], "variable")
		if err != nil {
			return err
		}

		ok, err := confirm(cmd, fmt.Sprintf("Clear the current value of variable %s?", id), variableClearYes)
		if err != nil {
			return err
		}
		if !ok {
			return PrintInfo(cmd.OutOrStdout(), "Cancelled.")
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		if _, err := client.Delete[any](cmd.Context(), "/api/v1/variables/"+id+"/value"); err != nil {
			return err
		}

		return PrintSuccess(cmd.OutOrStdout(), "Variable value cleared.")
	},
}

// variablesDeleteCmd deletes a variable and all its versions.
var variablesDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"rm"},
	Short:   "Delete a variable",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0], "variable")
		if err != nil {
			return err
		}

		ok, err := confirm(cmd, fmt.Sprintf("Delete variable %s? This cannot be undone", id), variableDeleteYes)
		if err != nil {
			return err
		}
		if !ok {
			return PrintInfo(cmd.OutOrStdout(), "Cancelled.")
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		if _, err := client.Delete[any](cmd.Context(), "/api/v1/variables/"+id); err != nil {
			return err
		}

		return PrintSuccess(cmd.OutOrStdout(), "Variable deleted.")
	},
}

// variableVersionsCmd groups the version commands of a variable.
var variableVersionsCmd = &cobra.Command{
	Use:     "versions",
	Aliases: []string{"version"},
	Short:   "Manage a variable's versions",
}

// variableVersionsListCmd lists all versions of a variable, newest first.
var variableVersionsListCmd = &cobra.Command{
	Use:     "list <variable-id>",
	Aliases: []string{"ls"},
	Short:   "List a variable's versions",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		variableID, err := parseID(args[0], "variable")
		if err != nil {
			return err
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Get[[]model.VariableVersion](cmd.Context(), "/api/v1/variables/"+variableID+"/versions")
		if err != nil {
			return err
		}
		if len(resp.Data) == 0 {
			return PrintInfo(cmd.OutOrStdout(), "No versions found.")
		}

		printVariableVersionsTable(cmd.OutOrStdout(), resp.Data)
		return nil
	},
}

// variableVersionsCurrentCmd shows a variable's current active version.
var variableVersionsCurrentCmd = &cobra.Command{
	Use:   "current <variable-id>",
	Short: "Show the current active version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		variableID, err := parseID(args[0], "variable")
		if err != nil {
			return err
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Get[model.VariableVersion](cmd.Context(), "/api/v1/variables/"+variableID+"/versions/current")
		if err != nil {
			return err
		}

		printVariableVersionTable(cmd.OutOrStdout(), resp.Data)
		return nil
	},
}

// variableVersionsAddCmd adds a new active version to a variable.
var variableVersionsAddCmd = &cobra.Command{
	Use:   "add <variable-id>",
	Short: "Add a new active version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		variableID, err := parseID(args[0], "variable")
		if err != nil {
			return err
		}

		rawValue := variableVersionValue
		if rawValue == "" {
			rawValue = "-" // prompt without echo when no value was provided
		}
		value, err := resolveVariableValue(cmd, rawValue)
		if err != nil {
			return err
		}
		if value == "" {
			return &UserError{Message: "version value cannot be empty"}
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Post[model.VariableVersion](cmd.Context(), "/api/v1/variables/"+variableID+"/versions", api.AddVariableVersionRequest{Value: value})
		if err != nil {
			return err
		}

		return PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Version %d created (id: %s).", resp.Data.Version, resp.Data.ID))
	},
}

// variableVersionsActivateCmd activates a version of a variable.
var variableVersionsActivateCmd = &cobra.Command{
	Use:   "activate <variable-id> <version-id>",
	Short: "Activate a version",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setVariableVersionActive(cmd, args, true)
	},
}

// variableVersionsDeactivateCmd deactivates a version of a variable.
var variableVersionsDeactivateCmd = &cobra.Command{
	Use:   "deactivate <variable-id> <version-id>",
	Short: "Deactivate a version",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setVariableVersionActive(cmd, args, false)
	},
}

// init registers the variable command flags and subcommands.
func init() {
	variablesListCmd.Flags().StringVarP(&variableSearch, "search", "s", "", "Filter variables by name")
	variablesListCmd.Flags().StringVar(&variableTagFilter, "tags", "", "Filter by comma-separated tag IDs")
	variablesListCmd.Flags().StringVar(&variableSort, "sort", "name", "Sort by name or created_at")
	variablesListCmd.Flags().StringVar(&variableOrder, "order", "asc", "Sort order: asc or desc")

	variablesCreateCmd.Flags().StringVarP(&variableCreateValue, "value", "v", "", "Variable value (omit for none; use '-' to enter it without echo or pipe it)")
	variablesClearValueCmd.Flags().BoolVarP(&variableClearYes, "yes", "y", false, "Skip the confirmation prompt")
	variablesDeleteCmd.Flags().BoolVarP(&variableDeleteYes, "yes", "y", false, "Skip the confirmation prompt")
	variableVersionsAddCmd.Flags().StringVarP(&variableVersionValue, "value", "v", "", "Version value (prompted without echo when omitted; use '-' to pipe it)")

	variableVersionsCmd.AddCommand(
		variableVersionsListCmd,
		variableVersionsCurrentCmd,
		variableVersionsAddCmd,
		variableVersionsActivateCmd,
		variableVersionsDeactivateCmd,
	)

	VariablesCmd.AddCommand(
		variablesListCmd,
		variablesGetCmd,
		variablesValueCmd,
		variablesCreateCmd,
		variablesSetNameCmd,
		variablesSetTagsCmd,
		variablesClearValueCmd,
		variablesDeleteCmd,
		variableVersionsCmd,
	)
}

// resolveVariableValue returns value, or reads it without echo when value is "-".
func resolveVariableValue(cmd *cobra.Command, value string) (string, error) {
	if value != "-" {
		return value, nil
	}
	return readSecret(cmd, "Value")
}

// setVariableVersionActive activates or deactivates a variable version.
func setVariableVersionActive(cmd *cobra.Command, args []string, active bool) error {
	variableID, err := parseID(args[0], "variable")
	if err != nil {
		return err
	}
	versionID, err := parseID(args[1], "version")
	if err != nil {
		return err
	}

	client, err := NewClientFromConfig()
	if err != nil {
		return err
	}

	if _, err := client.Patch[any](cmd.Context(), "/api/v1/variables/"+variableID+"/versions/"+versionID, api.SetVariableVersionActiveRequest{Active: active}); err != nil {
		return err
	}

	if active {
		return PrintSuccess(cmd.OutOrStdout(), "Version activated.")
	}
	return PrintSuccess(cmd.OutOrStdout(), "Version deactivated.")
}

// parseTagIDs parses zero or more tag UUIDs.
func parseTagIDs(rawIDs []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(rawIDs))
	for _, raw := range rawIDs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, &UserError{Message: fmt.Sprintf("invalid tag ID %q", raw)}
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// printVariablesTable renders the variable list as a terminal table.
func printVariablesTable(w io.Writer, variables []model.Variable) {
	t := newTable(w)
	t.SetHeaders("NAME", "ACTIVE", "TAGS", "CREATED", "ID")

	alignments := []table.Alignment{
		table.AlignLeft,
		table.AlignLeft,
		table.AlignLeft,
		table.AlignLeft,
		table.AlignLeft,
	}
	t.SetAlignment(alignments...)
	t.SetHeaderAlignment(alignments...)

	for _, variable := range variables {
		t.AddRow(
			variable.Name,
			yesNo(variable.Active),
			tagNames(variable.Tags),
			formatTime(variable.CreatedAt),
			variable.ID.String(),
		)
	}

	t.Render()
}

// printVariableTable renders a single variable as a field/value table.
func printVariableTable(w io.Writer, variable model.Variable) {
	t := newTable(w)
	t.SetHeaders("FIELD", "VALUE")

	t.AddRow("ID", variable.ID.String())
	t.AddRow("Name", variable.Name)
	t.AddRow("Environment", variable.EnvironmentID.String())
	t.AddRow("Tags", tagNames(variable.Tags))
	t.AddRow("Created", formatTime(variable.CreatedAt))
	t.AddRow("Updated", formatTime(variable.UpdatedAt))

	t.Render()
}

// printVariableVersionsTable renders the version list as a terminal table.
func printVariableVersionsTable(w io.Writer, versions []model.VariableVersion) {
	t := newTable(w)
	t.SetHeaders("VERSION", "ACTIVE", "CREATED", "UPDATED", "ID")

	alignments := []table.Alignment{
		table.AlignRight,
		table.AlignLeft,
		table.AlignLeft,
		table.AlignLeft,
		table.AlignLeft,
	}
	t.SetAlignment(alignments...)
	t.SetHeaderAlignment(alignments...)

	for _, version := range versions {
		t.AddRow(
			strconv.Itoa(version.Version),
			yesNo(version.Active),
			formatTime(version.CreatedAt),
			formatTime(version.UpdatedAt),
			version.ID.String(),
		)
	}

	t.Render()
}

// printVariableVersionTable renders a single version as a field/value table.
func printVariableVersionTable(w io.Writer, version model.VariableVersion) {
	t := newTable(w)
	t.SetHeaders("FIELD", "VALUE")

	t.AddRow("ID", version.ID.String())
	t.AddRow("Variable", version.VariableID.String())
	t.AddRow("Version", strconv.Itoa(version.Version))
	t.AddRow("Active", yesNo(version.Active))
	t.AddRow("Created", formatTime(version.CreatedAt))
	t.AddRow("Updated", formatTime(version.UpdatedAt))

	t.Render()
}

// tagNames joins tag names for display, or returns a dash when there are none.
func tagNames(tags []model.Tag) string {
	if len(tags) == 0 {
		return "—"
	}
	names := make([]string, 0, len(tags))
	for _, tag := range tags {
		names = append(names, tag.Name)
	}
	return strings.Join(names, ", ")
}

// yesNo renders a boolean for table output.
func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
