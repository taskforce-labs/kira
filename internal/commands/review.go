// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"kira/internal/config"
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Submit work item for review",
	Long: `Submit the current work item for review: optionally update trunk and rebase,
move the work item to review status, push the branch, and create or update a GitHub PR.

Run from a kira feature branch (e.g. 012-submit-for-review). Derives work item ID from
the branch name. Use --no-trunk-update or --no-rebase to skip update/rebase, and
--dry-run to preview steps without making changes.`,
	Args: cobra.NoArgs,
	RunE: runReview,
}

func init() {
	reviewCmd.SilenceUsage = true
	reviewCmd.Flags().StringSlice("reviewer", nil, "Reviewer (user number from kira user or email); can be repeated")
	reviewCmd.Flags().Bool("draft", true, "Create or leave PR as draft (default true)")
	reviewCmd.Flags().Bool("no-draft", false, "Create or update PR as ready for review (overrides --draft)")
	reviewCmd.Flags().Bool("no-trunk-update", false, "Skip updating trunk branch from remote before rebase")
	reviewCmd.Flags().Bool("no-rebase", false, "Skip rebasing current branch onto trunk")
	reviewCmd.Flags().Bool("dry-run", false, "Show what would be done without executing")
}

// reviewContext holds validated inputs for the review command.
type reviewContext struct {
	Cfg           *config.Config
	WorkItemID    string
	WorkItemPath  string
	CurrentBranch string
	TrunkBranch   string
	DryRun        bool
	NoTrunkUpdate bool
	NoRebase      bool
	Draft         bool
	Reviewers     []string
}

func runReview(cmd *cobra.Command, _ []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	noTrunkUpdate, _ := cmd.Flags().GetBool("no-trunk-update")
	noRebase, _ := cmd.Flags().GetBool("no-rebase")
	draft, _ := cmd.Flags().GetBool("draft")
	noDraft, _ := cmd.Flags().GetBool("no-draft")
	if noDraft {
		draft = false
	}
	reviewers, _ := cmd.Flags().GetStringSlice("reviewer")

	ctx := &reviewContext{
		Cfg:           cfg,
		DryRun:        dryRun,
		NoTrunkUpdate: noTrunkUpdate,
		NoRebase:      noRebase,
		Draft:         draft,
		Reviewers:     reviewers,
	}

	// Branch derivation and validation
	if err := validateBranchAndWorkItem(ctx); err != nil {
		return err
	}

	// Slice/task readiness (same behavior as original kira review)
	if err := checkSliceReadiness(ctx.WorkItemPath, cfg); err != nil {
		return err
	}

	if ctx.DryRun {
		return printReviewDryRun(ctx)
	}

	// Trunk update and rebase (flags override config)
	effectiveNoTrunkUpdate := ctx.NoTrunkUpdate || reviewConfigTrunkUpdateDisabled(cfg)
	effectiveNoRebase := ctx.NoRebase || reviewConfigRebaseDisabled(cfg)
	if err := runReviewTrunkUpdateAndRebase(cfg, ctx.WorkItemPath, effectiveNoTrunkUpdate, effectiveNoRebase); err != nil {
		return err
	}

	// Move, push, and PR are implemented in slices 3–4.
	return nil
}

func reviewConfigTrunkUpdateDisabled(cfg *config.Config) bool {
	return cfg.Review != nil && cfg.Review.TrunkUpdate != nil && !*cfg.Review.TrunkUpdate
}

func reviewConfigRebaseDisabled(cfg *config.Config) bool {
	return cfg.Review != nil && cfg.Review.Rebase != nil && !*cfg.Review.Rebase
}

// validateBranchAndWorkItem sets ctx.CurrentBranch, TrunkBranch, WorkItemID, WorkItemPath and validates.
func validateBranchAndWorkItem(ctx *reviewContext) error {
	repoRoot, err := getRepoRoot()
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	currentBranch, err := getCurrentBranch(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to determine current branch: %w", err)
	}
	ctx.CurrentBranch = currentBranch

	trunkBranch, err := resolveTrunkBranchForLatest(ctx.Cfg, nil, repoRoot)
	if err != nil {
		return fmt.Errorf("failed to resolve trunk branch: %w", err)
	}
	ctx.TrunkBranch = trunkBranch

	if currentBranch == trunkBranch {
		return fmt.Errorf("cannot run review on trunk branch %s; checkout a feature branch first", trunkBranch)
	}

	workItemID, err := parseWorkItemIDFromBranch(currentBranch, ctx.Cfg)
	if err != nil {
		return err
	}
	ctx.WorkItemID = workItemID

	workItemPath, err := findWorkItemFile(workItemID, ctx.Cfg)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("work item %s not found", workItemID)
		}
		return err
	}
	ctx.WorkItemPath = workItemPath

	status, err := statusFromWorkItemPath(workItemPath, ctx.Cfg)
	if err != nil {
		return fmt.Errorf("failed to derive work item status: %w", err)
	}
	const statusDoing = "doing"
	if status != statusDoing {
		return fmt.Errorf("work item %s is not in %s status (current: %s); only work items in %s can be submitted for review", workItemID, statusDoing, status, statusDoing)
	}

	return nil
}

// parseWorkItemIDFromBranch extracts work item ID from branch name (e.g. 012-submit-for-review -> 012) and validates format.
func parseWorkItemIDFromBranch(branchName string, cfg *config.Config) (string, error) {
	idx := strings.Index(branchName, "-")
	if idx <= 0 {
		return "", fmt.Errorf("branch %q does not match kira branch format (expected {id}-{kebab-title}); checkout a kira feature branch or use kira move for status changes", branchName)
	}
	id := branchName[:idx]
	if err := validateWorkItemID(id, cfg); err != nil {
		return "", fmt.Errorf("branch %q does not match kira branch format (expected %s): %w", branchName, cfg.Validation.IDFormat, err)
	}
	return id, nil
}

// checkSliceReadiness runs the same slice/task check as the original kira review: warn and error if tasks are open.
func checkSliceReadiness(path string, cfg *config.Config) error {
	_, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	if len(slices) == 0 {
		return nil
	}
	var total, open int
	for _, s := range slices {
		for _, t := range s.Tasks {
			total++
			if !t.Done {
				open++
			}
		}
	}
	if open > 0 {
		fmt.Printf("Warning: %d of %d tasks are still open. Run 'kira slice show' or 'kira slice progress' to see details.\n", open, total)
		return fmt.Errorf("work item has %d open task(s); complete them before review", open)
	}
	return nil
}

// printReviewDryRun prints planned steps and returns nil (no side effects).
func printReviewDryRun(ctx *reviewContext) error {
	fmt.Println("[DRY RUN] Planned steps:")
	fmt.Println("  1. Trunk update and rebase (unless --no-trunk-update / --no-rebase)")
	fmt.Println("  2. Move work item to review folder and update status")
	fmt.Println("  3. Push current branch to remote with --force-with-lease")
	fmt.Println("  4. Create or update GitHub PR (draft or ready per --draft)")
	fmt.Printf("[DRY RUN] Work item: %s at %s\n", ctx.WorkItemID, ctx.WorkItemPath)
	fmt.Printf("[DRY RUN] Branch: %s (trunk: %s)\n", ctx.CurrentBranch, ctx.TrunkBranch)
	return nil
}
