package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/aquasecurity/table"
	"github.com/spf13/cobra"

	"github.com/yannick2009/yily/internal/api"
	"github.com/yannick2009/yily/internal/domain/model"
)

var tagDeleteYes bool

// TagsCmd groups all tag-related commands.
var TagsCmd = &cobra.Command{
	Use:     "tags",
	Aliases: []string{"tag"},
	Short:   "Manage Yily tags",
}

// tagsListCmd lists the tags of a project.
var tagsListCmd = &cobra.Command{
	Use:     "list <project-id>",
	Aliases: []string{"ls"},
	Short:   "List a project's tags",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		projectID, err := parseID(args[0], "project")
		if err != nil {
			return err
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Get[[]model.Tag](cmd.Context(), "/api/v1/projects/"+projectID+"/tags")
		if err != nil {
			return err
		}
		if len(resp.Data) == 0 {
			return PrintInfo(cmd.OutOrStdout(), "No tags found.")
		}

		printTagsTable(cmd.OutOrStdout(), resp.Data)
		return nil
	},
}

// tagsCreateCmd creates one or more tags in a project.
var tagsCreateCmd = &cobra.Command{
	Use:   "create <project-id> <name> [name...]",
	Short: "Create one or more tags",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		projectID, err := parseID(args[0], "project")
		if err != nil {
			return err
		}

		names := make([]string, 0, len(args)-1)
		for _, raw := range args[1:] {
			if name := strings.TrimSpace(raw); name != "" {
				names = append(names, name)
			}
		}
		if len(names) == 0 {
			return &UserError{Message: "at least one tag name is required"}
		}

		client, err := NewClientFromConfig()
		if err != nil {
			return err
		}

		resp, err := client.Post[api.CreateTagResponse](cmd.Context(), "/api/v1/projects/"+projectID+"/tags", api.CreateTagRequest{Names: names})
		if err != nil {
			return err
		}

		if len(resp.Data.Created) > 0 {
			created := make([]string, 0, len(resp.Data.Created))
			for _, tag := range resp.Data.Created {
				created = append(created, tag.Name)
			}
			if err := PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Created %d tag(s): %s.", len(created), strings.Join(created, ", "))); err != nil {
				return err
			}
		}
		if len(resp.Data.Skipped) > 0 {
			if err := PrintInfo(cmd.OutOrStdout(), fmt.Sprintf("Already existed: %s.", strings.Join(resp.Data.Skipped, ", "))); err != nil {
				return err
			}
		}
		return nil
	},
}

// tagsDeleteCmd deletes a tag from a project.
var tagsDeleteCmd = &cobra.Command{
	Use:     "delete <project-id> <tag-id>",
	Aliases: []string{"rm"},
	Short:   "Delete a tag",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		projectID, err := parseID(args[0], "project")
		if err != nil {
			return err
		}
		tagID, err := parseID(args[1], "tag")
		if err != nil {
			return err
		}

		ok, err := confirm(cmd, fmt.Sprintf("Delete tag %s from project %s? This cannot be undone", tagID, projectID), tagDeleteYes)
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

		if _, err := client.Delete[any](cmd.Context(), "/api/v1/projects/"+projectID+"/tags/"+tagID); err != nil {
			return err
		}

		return PrintSuccess(cmd.OutOrStdout(), "Tag deleted.")
	},
}

// init registers the tag command flags and subcommands.
func init() {
	tagsDeleteCmd.Flags().BoolVarP(&tagDeleteYes, "yes", "y", false, "Skip the confirmation prompt")

	TagsCmd.AddCommand(
		tagsListCmd,
		tagsCreateCmd,
		tagsDeleteCmd,
	)
}

// printTagsTable renders the tag list as a terminal table.
func printTagsTable(w io.Writer, tags []model.Tag) {
	t := newTable(w)
	t.SetHeaders("ID", "NAME", "CREATED", "UPDATED")

	alignments := []table.Alignment{
		table.AlignLeft,
		table.AlignLeft,
		table.AlignLeft,
		table.AlignLeft,
	}
	t.SetAlignment(alignments...)
	t.SetHeaderAlignment(alignments...)

	for _, tag := range tags {
		t.AddRow(
			tag.ID.String(),
			tag.Name,
			formatTime(tag.CreatedAt),
			formatTime(tag.UpdatedAt),
		)
	}

	t.Render()
}
