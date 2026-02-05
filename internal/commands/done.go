// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/go-github/v61/github"
	"github.com/spf13/cobra"

	"kira/internal/config"
	"kira/internal/git"
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

// doneContext holds resolved work item and optional PR for the done command.
type doneContext struct {
	Cfg          *config.Config
	WorkItemID   string
	WorkItemPath string
	Status       string
	RepoRoot     string
	RemoteURL    string
	PR           *github.PullRequest
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
	workItemID := args[0]
	ctx, err := resolveDoneWorkItemAndPR(cfg, workItemID)
	if err != nil {
		return err
	}
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		return printDoneDryRun(cmd, ctx)
	}
	_ = ctx // used in later slices for merge/checks
	return nil
}

func resolveDoneWorkItemAndPR(cfg *config.Config, workItemID string) (*doneContext, error) {
	if err := validateWorkItemID(workItemID, cfg); err != nil {
		return nil, err
	}
	workItemPath, err := findWorkItemFile(workItemID, cfg)
	if err != nil {
		return nil, fmt.Errorf("work item %s not found", workItemID)
	}
	status, err := statusFromWorkItemPath(workItemPath, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to derive work item status: %w", err)
	}
	repoRoot, err := getRepoRoot()
	if err != nil {
		return nil, err
	}
	remoteURL, baseURL, err := resolveDoneRemote(cfg, repoRoot)
	if err != nil {
		return nil, err
	}
	ctx := &doneContext{Cfg: cfg, WorkItemID: workItemID, WorkItemPath: workItemPath, Status: status, RepoRoot: repoRoot, RemoteURL: remoteURL}
	pr, err := resolveDonePR(remoteURL, baseURL, workItemID, status)
	if err != nil {
		return nil, err
	}
	ctx.PR = pr
	return ctx, nil
}

func resolveDoneRemote(cfg *config.Config, repoRoot string) (remoteURL, baseURL string, err error) {
	remoteName := defaultRemoteName
	if cfg.Git != nil && cfg.Git.Remote != "" {
		remoteName = cfg.Git.Remote
	}
	remoteURL, err = getRemoteURL(remoteName, repoRoot)
	if err != nil {
		return "", "", fmt.Errorf("GitHub remote %q not configured", remoteName)
	}
	if cfg.Workspace != nil {
		baseURL = cfg.Workspace.GitBaseURL
	}
	return remoteURL, baseURL, nil
}

func resolveDonePR(remoteURL, baseURL, workItemID, status string) (*github.PullRequest, error) {
	if !isGitHubRemote(remoteURL, baseURL) {
		return nil, nil
	}
	token := os.Getenv("KIRA_GITHUB_TOKEN")
	if token == "" && status != "done" {
		return nil, fmt.Errorf("GitHub token required for PR merge. Set KIRA_GITHUB_TOKEN or run with --dry-run")
	}
	if token == "" {
		return nil, nil
	}
	apiCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := git.NewClient(apiCtx, token, baseURL)
	if err != nil {
		return nil, err
	}
	owner, repoName, err := git.ParseGitHubOwnerRepo(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("invalid remote URL: %w", err)
	}
	pr, err := git.FindPullRequestByWorkItemID(apiCtx, client, owner, repoName, workItemID)
	if err != nil {
		return nil, fmt.Errorf("failed to find pull request: %w", err)
	}
	if pr == nil && status != "done" {
		return nil, fmt.Errorf("no pull request found for work item %s. Ensure the branch exists and a PR is open, or that the PR was already merged", workItemID)
	}
	return pr, nil
}

func printDoneDryRun(cmd *cobra.Command, ctx *doneContext) error {
	baseURL := ""
	if ctx.Cfg.Workspace != nil {
		baseURL = ctx.Cfg.Workspace.GitBaseURL
	}
	if !isGitHubRemote(ctx.RemoteURL, baseURL) {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "[DRY RUN] kira done: trunk and work item validated; remote is not GitHub, PR steps skipped.")
		return nil
	}
	msg := "[DRY RUN] kira done: trunk and work item validated; PR resolved."
	if ctx.PR != nil && git.IsPRClosedOrMerged(ctx.PR) {
		msg += " PR already closed/merged (idempotent path)."
	} else if ctx.PR != nil {
		msg += " Open PR found; merge and remaining steps would run."
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), msg)
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
