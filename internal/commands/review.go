// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Submit work item for review and create GitHub pull request",
	Long: `Automatically derives work item ID from current branch name and creates a draft PR.

The command will:
1. Derive work item ID from current branch name
2. Move work item to review status
3. Create or update GitHub pull request
4. Optionally update trunk branch status and rebase`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Extract flags to validate parsing (actual logic will be implemented in later phases)
		_, _ = cmd.Flags().GetStringArray("reviewer")
		_, _ = cmd.Flags().GetBool("draft")
		_, _ = cmd.Flags().GetBool("no-trunk-update")
		_, _ = cmd.Flags().GetBool("no-rebase")
		_, _ = cmd.Flags().GetString("title")
		_, _ = cmd.Flags().GetString("description")

		// Placeholder: return help for now
		return cmd.Help()
	},
}

func init() {
	reviewCmd.Flags().StringArray("reviewer", []string{}, "Specify reviewer (can be used multiple times). Can be user number from 'kira user' command or email address")
	reviewCmd.Flags().Bool("draft", true, "Create as draft PR (default: true)")
	reviewCmd.Flags().Bool("no-trunk-update", false, "Skip updating trunk branch status (overrides config)")
	reviewCmd.Flags().Bool("no-rebase", false, "Skip rebasing current branch after trunk update (overrides config)")
	reviewCmd.Flags().String("title", "", "Custom PR title (derived from work item if not provided)")
	reviewCmd.Flags().String("description", "", "Custom PR description (uses work item content if not provided)")
}
