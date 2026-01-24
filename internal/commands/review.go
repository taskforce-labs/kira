// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"kira/internal/config"
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
		// Load configuration
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Validate branch context (Phase 2)
		if err := validateBranchContext(cfg); err != nil {
			return err
		}

		// Get current branch and derive work item ID (Phase 2)
		currentBranch, err := getCurrentBranch("")
		if err != nil {
			return fmt.Errorf("failed to determine current branch: %w", err)
		}

		workItemID, err := deriveWorkItemFromBranch(currentBranch)
		if err != nil {
			return err
		}

		// Extract flags to validate parsing (actual logic will be implemented in later phases)
		_, _ = cmd.Flags().GetStringArray("reviewer")
		_, _ = cmd.Flags().GetBool("draft")
		_, _ = cmd.Flags().GetBool("no-trunk-update")
		_, _ = cmd.Flags().GetBool("no-rebase")
		_, _ = cmd.Flags().GetString("title")
		_, _ = cmd.Flags().GetString("description")

		// Placeholder: return help for now (will be replaced in later phases)
		_ = workItemID // Suppress unused variable warning
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

// deriveWorkItemFromBranch extracts the work item ID from a branch name.
// Branch format: {id}-{title} (e.g., "012-submit-for-review")
// Returns the 3-digit work item ID or an error if the format is invalid.
func deriveWorkItemFromBranch(branchName string) (string, error) {
	if branchName == "" {
		return "", fmt.Errorf("branch name cannot be empty")
	}

	// Find first dash in branch name
	dashIndex := strings.Index(branchName, "-")
	if dashIndex == -1 {
		return "", fmt.Errorf("branch name '%s' does not follow kira naming convention: expected format '{id}-{title}'", branchName)
	}

	// Extract ID from beginning of branch name
	workItemID := branchName[:dashIndex]
	if workItemID == "" {
		return "", fmt.Errorf("branch name '%s' does not follow kira naming convention: work item ID is missing", branchName)
	}

	// Validate ID format (must be exactly 3 digits)
	idRegex := regexp.MustCompile(`^\d{3}$`)
	if !idRegex.MatchString(workItemID) {
		return "", fmt.Errorf("invalid work item ID '%s' in branch name '%s': ID must be exactly 3 digits", workItemID, branchName)
	}

	return workItemID, nil
}

// validateBranchContext ensures the command is not run on the trunk branch.
// Returns an error if the current branch is the trunk branch, nil otherwise.
func validateBranchContext(cfg *config.Config) error {
	// Get current branch
	currentBranch, err := getCurrentBranch("")
	if err != nil {
		return fmt.Errorf("failed to determine current branch: %w", err)
	}

	// Determine trunk branch (using same logic as start command)
	trunkBranch, err := determineTrunkBranch(cfg, "", "", false)
	if err != nil {
		return fmt.Errorf("failed to determine trunk branch: %w", err)
	}

	// Check if current branch is trunk branch
	if currentBranch == trunkBranch {
		return fmt.Errorf("cannot run 'kira review' on trunk branch '%s'", trunkBranch)
	}

	return nil
}
