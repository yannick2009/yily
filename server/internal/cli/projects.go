package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yannick2009/yily/internal/api"
	"github.com/yannick2009/yily/internal/domain/model"
)

var (
	deleteProjectYes bool
	clearProjectsYes bool
)

// ProjectsCmd groups all project-related commands.
var ProjectsCmd = &cobra.Command{
	Use:     "projects",
	Aliases: []string{"project"},
	Short:   "Manage Yily projects",
}

// projectsListCmd lists all projects in the Yily instance.
var projectsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all projects",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Get[[]api.ProjectSummaryResponse](cmd.Context(), "/api/v1/projects")
		if err != nil {
			return err
		}
		if len(resp.Data) == 0 {
			return PrintInfo(cmd.OutOrStdout(), "No projects found.")
		}

		printProjectsTable(cmd.OutOrStdout(), resp.Data)
		return nil
	},
}

// projectsGetCmd shows a single project's details.
var projectsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		id, err := parseID(args[0], "project")
		if err != nil {
			return err
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Get[model.Project](cmd.Context(), "/api/v1/projects/"+id)
		if err != nil {
			return err
		}

		printProjectTable(cmd.OutOrStdout(), resp.Data)
		return nil
	},
}

// projectsCreateCmd creates a new project in the Yily instance.
var projectsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		name := strings.TrimSpace(args[0])
		if name == "" {
			return &UserError{Message: "project name cannot be empty"}
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Post[model.Project](cmd.Context(), "/api/v1/projects", api.CreateProjectRequest{Name: name})
		if err != nil {
			return err
		}

		return PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Project %q created (id: %s).", resp.Data.Name, resp.Data.ID))
	},
}

// projectsSetNameCmd renames an existing project in the Yily instance.
var projectsSetNameCmd = &cobra.Command{
	Use:     "set-name <id> <name>",
	Aliases: []string{"rename", "update"},
	Short:   "Set a project's name",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		id, err := parseID(args[0], "project")
		if err != nil {
			return err
		}
		name := strings.TrimSpace(args[1])
		if name == "" {
			return &UserError{Message: "project name cannot be empty"}
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		if _, err := client.Patch[any](cmd.Context(), "/api/v1/projects/"+id, api.UpdateProjectRequest{Name: name}); err != nil {
			return err
		}

		return PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Project name set to %q.", name))
	},
}

// projectsDeleteCmd deletes a project from the Yily instance.
var projectsDeleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"rm"},
	Short:   "Delete a project",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		id, err := parseID(args[0], "project")
		if err != nil {
			return err
		}

		ok, err := confirm(cmd, fmt.Sprintf("Delete project %s? This cannot be undone", id), deleteProjectYes)
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

		if _, err := client.Delete[any](cmd.Context(), "/api/v1/projects/"+id); err != nil {
			return err
		}

		return PrintSuccess(cmd.OutOrStdout(), "Project deleted.")
	},
}

// projectsClearCmd deletes all projects from the Yily instance.
var projectsClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete all projects",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		ok, err := confirm(cmd, "Delete ALL projects? This cannot be undone", clearProjectsYes)
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

		if _, err := client.Delete[any](cmd.Context(), "/api/v1/projects"); err != nil {
			return err
		}

		return PrintSuccess(cmd.OutOrStdout(), "All projects deleted.")
	},
}

// init registers the project command flags and subcommands.
func init() {
	projectsDeleteCmd.Flags().BoolVarP(&deleteProjectYes, "yes", "y", false, "Skip the confirmation prompt")
	projectsClearCmd.Flags().BoolVarP(&clearProjectsYes, "yes", "y", false, "Skip the confirmation prompt")

	ProjectsCmd.AddCommand(
		projectsListCmd,
		projectsGetCmd,
		projectsCreateCmd,
		projectsSetNameCmd,
		projectsDeleteCmd,
		projectsClearCmd,
	)
}
