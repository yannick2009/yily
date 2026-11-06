package cli

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aquasecurity/table"
	"github.com/spf13/cobra"

	"github.com/yannick2009/yily/internal/api"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/domain/types"
)

var (
	environmentDescription  string
	environmentCopyTags     bool
	environmentDeleteYes    bool
	environmentExportFormat string
	environmentExportOutput string
)

// EnvironmentsCmd groups all environment-related commands.
var EnvironmentsCmd = &cobra.Command{
	Use:     "environments",
	Aliases: []string{"environment", "env"},
	Short:   "Manage Yily environments",
}

// environmentsListCmd lists the environments of a project.
var environmentsListCmd = &cobra.Command{
	Use:     "list <project-id>",
	Aliases: []string{"ls"},
	Short:   "List a project's environments",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := parseID(args[0], "project")
		if err != nil {
			return err
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Get[[]api.EnvironmentSummaryResponse](cmd.Context(), "/api/v1/projects/"+projectID+"/environments")
		if err != nil {
			return err
		}
		if len(resp.Data) == 0 {
			return PrintInfo(cmd.OutOrStdout(), "No environments found.")
		}

		printEnvironmentsTable(cmd.OutOrStdout(), resp.Data)
		return nil
	},
}

// environmentsGetCmd shows a single environment's details.
var environmentsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show an environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0], "environment")
		if err != nil {
			return err
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Get[model.Environment](cmd.Context(), "/api/v1/environments/"+id)
		if err != nil {
			return err
		}

		printEnvironmentTable(cmd.OutOrStdout(), resp.Data)
		return nil
	},
}

// environmentsCreateCmd creates an environment in a project.
var environmentsCreateCmd = &cobra.Command{
	Use:   "create <project-id> <name>",
	Short: "Create an environment",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := parseID(args[0], "project")
		if err != nil {
			return err
		}
		name := strings.TrimSpace(args[1])
		if name == "" {
			return &UserError{Message: "environment name cannot be empty"}
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Post[model.Environment](cmd.Context(), "/api/v1/projects/"+projectID+"/environments", api.CreateEnvironmentRequest{
			Name:        name,
			Description: environmentDescription,
		})
		if err != nil {
			return err
		}

		return PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Environment %q created (id: %s).", resp.Data.Name, resp.Data.ID))
	},
}

// environmentsCloneCmd clones an environment into the same project.
var environmentsCloneCmd = &cobra.Command{
	Use:   "clone <id> <new-name>",
	Short: "Clone an environment",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0], "environment")
		if err != nil {
			return err
		}
		name := strings.TrimSpace(args[1])
		if name == "" {
			return &UserError{Message: "environment name cannot be empty"}
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		if _, err := client.Post[any](cmd.Context(), "/api/v1/environments/"+id+"/clone", api.CloneEnvironmentRequest{
			Name:     name,
			CopyTags: environmentCopyTags,
		}); err != nil {
			return err
		}

		return PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Environment cloned as %q.", name))
	},
}

// environmentsSetNameCmd sets an environment's name.
var environmentsSetNameCmd = &cobra.Command{
	Use:     "set-name <id> <name>",
	Aliases: []string{"rename", "update"},
	Short:   "Set an environment's name",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0], "environment")
		if err != nil {
			return err
		}
		name := strings.TrimSpace(args[1])
		if name == "" {
			return &UserError{Message: "environment name cannot be empty"}
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		if _, err := client.Patch[any](cmd.Context(), "/api/v1/environments/"+id+"/name", api.UpdateEnvironmentNameRequest{Name: name}); err != nil {
			return err
		}

		return PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Environment name set to %q.", name))
	},
}

// environmentsSetDescriptionCmd sets or clears an environment's description.
var environmentsSetDescriptionCmd = &cobra.Command{
	Use:     "set-description <id> <description>",
	Aliases: []string{"describe"},
	Short:   "Set an environment's description",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0], "environment")
		if err != nil {
			return err
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		if _, err := client.Patch[any](cmd.Context(), "/api/v1/environments/"+id+"/description", api.UpdateEnvironmentDescriptionRequest{
			Description: args[1],
		}); err != nil {
			return err
		}

		if args[1] == "" {
			return PrintSuccess(cmd.OutOrStdout(), "Environment description cleared.")
		}
		return PrintSuccess(cmd.OutOrStdout(), "Environment description updated.")
	},
}

// environmentsDeleteCmd deletes an environment from the Yily instance.
var environmentsDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"rm"},
	Short:   "Delete an environment",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0], "environment")
		if err != nil {
			return err
		}

		ok, err := confirm(cmd, fmt.Sprintf("Delete environment %s? This cannot be undone", id), environmentDeleteYes)
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

		if _, err := client.Delete[any](cmd.Context(), "/api/v1/environments/"+id); err != nil {
			return err
		}

		return PrintSuccess(cmd.OutOrStdout(), "Environment deleted.")
	},
}

// environmentsExportCmd downloads an environment's variables to a file.
var environmentsExportCmd = &cobra.Command{
	Use:   "export <id>",
	Short: "Export an environment's variables to a file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0], "environment")
		if err != nil {
			return err
		}

		formatValue := strings.ToLower(strings.TrimSpace(environmentExportFormat))
		// If an output path is given but no explicit format, infer it from the extension.
		if !cmd.Flags().Changed("format") && environmentExportOutput != "" {
			if inferred, err := types.FromFilename(environmentExportOutput); err == nil {
				formatValue = string(inferred)
			}
		}
		fileType, err := types.FromString(formatValue)
		if err != nil {
			return &UserError{Message: fmt.Sprintf("unsupported format %q (use json, yaml or env)", formatValue)}
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		content, serverFilename, err := client.GetFile(cmd.Context(), "/api/v1/environments/"+id+"/export?format="+url.QueryEscape(string(fileType)))
		if err != nil {
			return err
		}

		filename := serverFilename
		if filename == "" {
			filename = "export_" + id + fileType.Extension()
		}

		output := environmentExportOutput
		if output == "" {
			output = filename
		} else if info, err := os.Stat(output); err == nil && info.IsDir() {
			output = filepath.Join(output, filename)
		}

		if err := writePrivateFile(output, content); err != nil {
			return err
		}

		return PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Exported to %s.", filepath.Clean(output)))
	},
}

// environmentsImportCmd uploads a variables file into an environment.
var environmentsImportCmd = &cobra.Command{
	Use:   "import <id> <file>",
	Short: "Import variables into an environment from a file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0], "environment")
		if err != nil {
			return err
		}

		filePath := args[1]
		if _, err := types.FromFilename(filePath); err != nil {
			return &UserError{Message: fmt.Sprintf("unsupported file %q (use .json, .yaml, .yml or .env)", filePath)}
		}
		info, err := os.Stat(filePath)
		if err != nil {
			return &UserError{Message: fmt.Sprintf("cannot read %q", filePath)}
		}
		if info.IsDir() {
			return &UserError{Message: fmt.Sprintf("%q is a directory, not a file", filePath)}
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		if _, err := client.PostFile[any](cmd.Context(), "/api/v1/environments/"+id+"/imports", filePath); err != nil {
			return err
		}

		return PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Variables imported from %s.", filepath.Clean(filePath)))
	},
}

// init registers the environment command flags and subcommands.
func init() {
	environmentsCreateCmd.Flags().StringVarP(&environmentDescription, "description", "d", "", "Environment description")
	environmentsCloneCmd.Flags().BoolVar(&environmentCopyTags, "copy-tags", false, "Copy the source environment's tags")
	environmentsDeleteCmd.Flags().BoolVarP(&environmentDeleteYes, "yes", "y", false, "Skip the confirmation prompt")
	environmentsExportCmd.Flags().StringVarP(&environmentExportFormat, "format", "f", "env", "Export format: json, yaml or env")
	environmentsExportCmd.Flags().StringVarP(&environmentExportOutput, "output", "o", "", "Output file or directory (default: current directory)")

	EnvironmentsCmd.AddCommand(
		environmentsListCmd,
		environmentsGetCmd,
		environmentsCreateCmd,
		environmentsCloneCmd,
		environmentsSetNameCmd,
		environmentsSetDescriptionCmd,
		environmentsDeleteCmd,
		environmentsExportCmd,
		environmentsImportCmd,
	)
}

// printEnvironmentsTable renders the environment list as a terminal table.
func printEnvironmentsTable(w io.Writer, environments []api.EnvironmentSummaryResponse) {
	t := newTable(w)
	t.SetHeaders("ID", "NAME", "DESCRIPTION", "VARIABLES", "CREATED")

	alignments := []table.Alignment{
		table.AlignLeft,
		table.AlignLeft,
		table.AlignLeft,
		table.AlignRight,
		table.AlignLeft,
	}
	t.SetAlignment(alignments...)
	t.SetHeaderAlignment(alignments...)

	for _, environment := range environments {
		t.AddRow(
			environment.ID.String(),
			environment.Name,
			orDash(environment.Description),
			strconv.Itoa(environment.VariableCount),
			formatTime(environment.CreatedAt),
		)
	}

	t.Render()
}

// printEnvironmentTable renders a single environment as a field/value table.
func printEnvironmentTable(w io.Writer, environment model.Environment) {
	t := newTable(w)
	t.SetHeaders("FIELD", "VALUE")

	t.AddRow("ID", environment.ID.String())
	t.AddRow("Name", environment.Name)
	t.AddRow("Description", orDash(environment.Description))
	t.AddRow("Project", environment.ProjectID.String())
	t.AddRow("Created", formatTime(environment.CreatedAt))
	t.AddRow("Updated", formatTime(environment.UpdatedAt))

	t.Render()
}

// orDash returns the value, or an em dash when it is empty.
func orDash(value string) string {
	if value == "" {
		return "—"
	}
	return value
}
