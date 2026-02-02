// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v61/github"
	"github.com/spf13/cobra"

	"kira/internal/config"
	"kira/internal/git"
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
	reviewCmd.Flags().Bool("draft", false, "Create or leave PR as draft (default: create/update as ready for review)")
	reviewCmd.Flags().Bool("no-draft", false, "Create or update PR as ready for review (default; overrides --draft)")
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
	return runReviewExecute(ctx, cfg)
}

func runReviewExecute(ctx *reviewContext, cfg *config.Config) error {
	effectiveNoTrunkUpdate := ctx.NoTrunkUpdate || reviewConfigTrunkUpdateDisabled(cfg)
	effectiveNoRebase := ctx.NoRebase || reviewConfigRebaseDisabled(cfg)
	if err := runReviewTrunkUpdateAndRebase(cfg, ctx.WorkItemPath, effectiveNoTrunkUpdate, effectiveNoRebase); err != nil {
		return err
	}
	repos, err := discoverRepositoriesFromPath(cfg, ctx.WorkItemPath)
	if err != nil {
		return fmt.Errorf("failed to discover repositories for push: %w", err)
	}
	if runReviewWouldCreateOrUpdatePR(cfg, repos) && os.Getenv("KIRA_GITHUB_TOKEN") == "" {
		return fmt.Errorf("KIRA_GITHUB_TOKEN is not set. Set it to create or update PRs, or use --dry-run to skip")
	}
	if err := runReviewMoveToReview(ctx, cfg); err != nil {
		return err
	}
	if err := runReviewPush(ctx, repos); err != nil {
		return err
	}
	fmt.Println("Moved to review and pushed.")
	if err := runReviewCreateOrUpdatePR(ctx, cfg, repos); err != nil {
		log.Printf("Warning: PR create/update failed: %v", err)
	}
	return nil
}

func reviewCommitMove(cfg *config.Config) bool {
	return cfg.Review != nil && cfg.Review.CommitMove != nil && *cfg.Review.CommitMove
}

func runReviewMoveToReview(ctx *reviewContext, cfg *config.Config) error {
	var moveMetadata workItemMetadata
	if reviewCommitMove(cfg) {
		var err error
		moveMetadata.workItemType, moveMetadata.id, moveMetadata.title, moveMetadata.currentStatus, moveMetadata.repos, err = extractWorkItemMetadata(ctx.WorkItemPath, cfg)
		if err != nil {
			return fmt.Errorf("failed to extract work item metadata for commit: %w", err)
		}
	}
	fmt.Println("Moving work item to review...")
	if err := moveWorkItemWithoutCommit(cfg, ctx.WorkItemID, "review"); err != nil {
		return fmt.Errorf("failed to move work item to review: %w", err)
	}
	reviewFolder := cfg.StatusFolders["review"]
	if reviewFolder == "" {
		reviewFolder = "3_review"
	}
	newPath := filepath.Join(config.GetWorkFolderPath(cfg), reviewFolder, filepath.Base(ctx.WorkItemPath))
	if reviewCommitMove(cfg) {
		subject, body, err := buildCommitMessage(cfg, moveMetadata.workItemType, moveMetadata.id, moveMetadata.title, moveMetadata.currentStatus, "review")
		if err != nil {
			return fmt.Errorf("failed to build commit message: %w", err)
		}
		if err := commitMove(ctx.WorkItemPath, newPath, subject, body, false); err != nil {
			return fmt.Errorf("failed to commit move: %w", err)
		}
		fmt.Println("Moved work item to review and committed.")
	} else {
		fmt.Println("Moved work item to review.")
	}
	return nil
}

func runReviewPush(ctx *reviewContext, repos []RepositoryInfo) error {
	fmt.Println("Pushing branch...")
	for _, repo := range repos {
		if err := pushBranchForceWithLease(repo.Remote, ctx.CurrentBranch, repo.Path); err != nil {
			return fmt.Errorf("push failed for %s: %w (work item was moved to review; fix push then re-run if needed)", repo.Name, err)
		}
		fmt.Printf("Pushed %s to %s\n", ctx.CurrentBranch, repo.Remote)
	}
	return nil
}

func runReviewWouldCreateOrUpdatePR(cfg *config.Config, repos []RepositoryInfo) bool {
	baseURL := ""
	if cfg.Workspace != nil {
		baseURL = cfg.Workspace.GitBaseURL
	}
	for _, repo := range repos {
		remoteURL, err := getRemoteURL(repo.Remote, repo.Path)
		if err != nil {
			continue
		}
		if isGitHubRemote(remoteURL, baseURL) {
			return true
		}
	}
	return false
}

func runReviewCreateOrUpdatePR(ctx *reviewContext, cfg *config.Config, repos []RepositoryInfo) error {
	token := os.Getenv("KIRA_GITHUB_TOKEN")
	if token == "" {
		return nil
	}
	baseURL := ""
	if cfg.Workspace != nil {
		baseURL = cfg.Workspace.GitBaseURL
	}
	workItemPath, err := findWorkItemFile(ctx.WorkItemID, cfg)
	if err != nil {
		return err
	}
	_, id, title, _, _, err := extractWorkItemMetadata(workItemPath, cfg)
	if err != nil {
		return err
	}
	prTitle := id + ": " + title
	prBody, _ := extractWorkItemBody(workItemPath, cfg)
	prCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := git.NewClient(prCtx, token, baseURL)
	if err != nil {
		return err
	}
	for _, repo := range repos {
		runReviewCreateOrUpdatePRForRepo(prCtx, ctx, repo, prTitle, prBody, client, baseURL)
	}
	return nil
}

func runReviewCreateOrUpdatePRForRepo(prCtx context.Context, ctx *reviewContext, repo RepositoryInfo, prTitle, prBody string, client *github.Client, baseURL string) {
	remoteURL, err := getRemoteURL(repo.Remote, repo.Path)
	if err != nil || !isGitHubRemote(remoteURL, baseURL) {
		return
	}
	owner, repoName, err := git.ParseGitHubOwnerRepo(remoteURL)
	if err != nil {
		log.Printf("Warning: could not parse GitHub remote %s: %v", remoteURL, err)
		return
	}
	prs, err := git.ListPullRequestsByHead(prCtx, client, owner, repoName, ctx.CurrentBranch)
	if err != nil {
		log.Printf("Warning: failed to list PRs for %s: %v", repo.Name, err)
		return
	}
	if len(prs) > 0 {
		runReviewUpdateExistingPR(prCtx, client, owner, repoName, ctx, prs[0])
		return
	}
	prURL, err := git.CreatePR(prCtx, client, owner, repoName, ctx.TrunkBranch, ctx.CurrentBranch, prTitle, prBody, ctx.Draft, ctx.Reviewers)
	if err != nil {
		log.Printf("Warning: failed to create PR for %s: %v", repo.Name, err)
		return
	}
	fmt.Printf("PR: %s\n", prURL)
}

func runReviewUpdateExistingPR(prCtx context.Context, client *github.Client, owner, repoName string, ctx *reviewContext, pr *github.PullRequest) {
	if pr.HTMLURL != nil && (pr.Draft == nil || !*pr.Draft) {
		fmt.Printf("PR: %s\n", *pr.HTMLURL)
		return
	}
	if pr.Number == nil {
		return
	}
	if pr.Draft == nil || !*pr.Draft || ctx.Draft {
		if pr.HTMLURL != nil {
			fmt.Printf("PR: %s\n", *pr.HTMLURL)
		}
		return
	}
	if err := git.UpdateDraftToReady(prCtx, client, owner, repoName, *pr.Number); err != nil {
		log.Printf("Warning: failed to update draft to ready: %v", err)
		return
	}
	if len(ctx.Reviewers) > 0 {
		_ = git.SetReviewers(prCtx, client, owner, repoName, *pr.Number, ctx.Reviewers)
	}
	if pr.HTMLURL != nil {
		fmt.Printf("PR ready for review: %s\n", *pr.HTMLURL)
	}
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
