// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"kira/internal/config"
)

var doneCmd = &cobra.Command{
	Use:   "done <work-item-id>",
	Short: "Complete work item by merging PR, pulling trunk, and updating status to done",
	Long: `Completes the work item workflow: merge the associated pull request (if open),
pull latest trunk, update the work item to "done" on trunk, and optionally delete the feature branch.
Must be run on the trunk branch. Work item ID is required (e.g. kira done 014).`,
	Args: cobra.ExactArgs(1),
	RunE: runDone,
}

func init() {
	doneCmd.SilenceUsage = true
	doneCmd.Flags().String("merge-strategy", "", "Merge strategy: merge, squash, or rebase (default from config)")
	doneCmd.Flags().Bool("no-cleanup", false, "Do not delete the feature branch after merge")
	doneCmd.Flags().Bool("force", false, "Force merge even if PR has failing checks or unresolved comments")
	doneCmd.Flags().Bool("dry-run", false, "Preview what would be done without executing")
}

func runDone(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	if err := validateTrunkBranch(cfg); err != nil {
		return err
	}
	_ = args[0] // work-item-id, used in later slices
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "[DRY RUN] kira done: trunk validated; remaining steps not yet implemented.")
		return nil
	}
	// Stub: validation passed; full flow in later slices
	return nil
}

// validateTrunkBranch ensures the current branch is the configured trunk branch.
// Used by kira done so that the feature branch can be removed after merge.
func validateTrunkBranch(cfg *config.Config) error {
	repoRoot, err := getRepoRoot()
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}
	currentBranch, err := getCurrentBranch(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to determine current branch: %w", err)
	}
	trunkBranch, err := resolveTrunkBranchForLatest(cfg, nil, repoRoot)
	if err != nil {
		return fmt.Errorf("failed to resolve trunk branch: %w", err)
	}
	if currentBranch != trunkBranch {
		return fmt.Errorf("cannot run 'kira done' on a feature branch. Check out the trunk branch (%s) first so the feature branch can be removed after merge", trunkBranch)
	}
	return nil
}
