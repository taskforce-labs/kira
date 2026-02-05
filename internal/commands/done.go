// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"

	"kira/internal/config"
)

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
