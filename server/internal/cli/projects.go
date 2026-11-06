package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// ProjectSummary mirrors the list endpoint's project DTO, including its counts.
type ProjectSummary struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	EnvironmentCount int       `json:"environment_count"`
	TagCount         int       `json:"tag_count"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Project is the subset of a project returned by the detail endpoint.
type Project struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// projectRequest is the JSON body accepted by the create and rename endpoints.
type projectRequest struct {
	Name string `json:"name"`
}

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
		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Get[[]ProjectSummary](cmd.Context(), "/api/v1/projects")
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
		id, err := parseID(args[0], "project")
		if err != nil {
			return err
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Get[Project](cmd.Context(), "/api/v1/projects/"+id)
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
		name := strings.TrimSpace(args[0])
		if name == "" {
			return &UserError{Message: "project name cannot be empty"}
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Post[Project](cmd.Context(), "/api/v1/projects", projectRequest{Name: name})
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

		if _, err := client.Patch[any](cmd.Context(), "/api/v1/projects/"+id, projectRequest{Name: name}); err != nil {
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
