// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"kira/internal/config"
	"kira/internal/git"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Get work item information from current branch",
	Long: `Derives the work item from the current branch name and outputs PR title, body, slug, or PR list.
Used by CI workflows to update PR details from work items.

Examples:
  kira current --title          # Output PR title (e.g. "034: ci update pr details")
  kira current --body           # Output entire work item file content
  kira current --slug           # Output slug (e.g. "034-ci-update-pr-details") - matches worktree and branch name
  kira current prs              # Output JSON array of related PRs (main repo + project repos in polyrepo)`,
	RunE:         runCurrent,
	SilenceUsage: true,
}

var currentPRsCmd = &cobra.Command{
	Use:          "prs",
	Short:        "List related PRs",
	Long:         `Outputs a JSON array of related PRs across all repos in the workspace. Includes main repo (where kira.yml is located) and all project repos in polyrepo workspaces. Output format: [{"owner": "org", "repo": "repo-name", "pr_number": 123, "branch": "034-ci-update-pr-details"}, ...]`,
	RunE:         runCurrentPRs,
	SilenceUsage: true,
}

func init() {
	currentCmd.Flags().Bool("title", false, "Output PR title")
	currentCmd.Flags().Bool("body", false, "Output work item file content")
	currentCmd.Flags().Bool("slug", false, "Output slug (full branch name matching worktree and branch, e.g. '034-ci-update-pr-details')")
	currentCmd.AddCommand(currentPRsCmd)
}

func runCurrent(cmd *cobra.Command, _ []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}

	titleFlag, _ := cmd.Flags().GetBool("title")
	bodyFlag, _ := cmd.Flags().GetBool("body")
	slugFlag, _ := cmd.Flags().GetBool("slug")

	flagCount := 0
	if titleFlag {
		flagCount++
	}
	if bodyFlag {
		flagCount++
	}
	if slugFlag {
		flagCount++
	}

	if flagCount > 1 {
		return fmt.Errorf("cannot use multiple flags together (--title, --body, --slug)")
	}

	if titleFlag {
		return runCurrentTitle(cfg)
	}
	if bodyFlag {
		return runCurrentBody(cfg)
	}
	if slugFlag {
		return runCurrentSlug(cfg)
	}

	return cmd.Help()
}

func runCurrentTitle(cfg *config.Config) error {
	repoRoot, err := getRepoRoot()
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	currentBranch, err := getCurrentBranch(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to determine current branch: %w", err)
	}

	workItemID, err := parseWorkItemIDFromBranch(currentBranch, cfg)
	if err != nil {
		// Invalid branch name - exit without output, non-zero exit code
		os.Exit(1)
	}

	workItemPath, err := findWorkItemFileInAllStatusFolders(workItemID, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Work item %s not found\n", workItemID)
		os.Exit(1)
	}

	_, id, title, _, _, err := extractWorkItemMetadata(workItemPath, cfg)
	if err != nil {
		return fmt.Errorf("failed to read work item: %w", err)
	}

	// Output PR title format: "{id}: {title}" (same as kira start)
	prTitle := fmt.Sprintf("%s: %s", id, title)
	fmt.Println(prTitle)
	return nil
}

func runCurrentBody(cfg *config.Config) error {
	repoRoot, err := getRepoRoot()
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	currentBranch, err := getCurrentBranch(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to determine current branch: %w", err)
	}

	workItemID, err := parseWorkItemIDFromBranch(currentBranch, cfg)
	if err != nil {
		// Invalid branch name - exit without output, non-zero exit code
		os.Exit(1)
	}

	workItemPath, err := findWorkItemFileInAllStatusFolders(workItemID, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Work item %s not found\n", workItemID)
		os.Exit(1)
	}

	// Output entire work item file content (front matter + body)
	content, err := safeReadFile(workItemPath, cfg)
	if err != nil {
		return fmt.Errorf("failed to read work item file: %w", err)
	}

	fmt.Print(string(content))
	// Note: Don't add newline here - work item content should be output as-is
	return nil
}

func runCurrentSlug(cfg *config.Config) error {
	repoRoot, err := getRepoRoot()
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	currentBranch, err := getCurrentBranch(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to determine current branch: %w", err)
	}

	// Validate branch name format (must be {id}-{slug})
	_, err = parseWorkItemIDFromBranch(currentBranch, cfg)
	if err != nil {
		// Invalid branch name - exit without output, non-zero exit code
		os.Exit(1)
	}

	// Output the full branch name (which matches worktree and branch name)
	// Branch format is "{id}-{slug}" (e.g., "034-ci-update-pr-details")
	fmt.Println(currentBranch)
	return nil
}

// extractSlugFromBranch extracts the slug (kebab-case title) from a branch name.
// Branch format is "{id}-{slug}" (e.g., "034-ci-update-pr-details" -> "ci-update-pr-details").
// This is kept for backward compatibility but runCurrentSlug now outputs the full branch name.
func extractSlugFromBranch(branchName, workItemID string) string {
	// Expected format: "{id}-{slug}"
	prefix := workItemID + "-"
	if !strings.HasPrefix(branchName, prefix) {
		return ""
	}
	slug := strings.TrimPrefix(branchName, prefix)
	if slug == "" {
		return ""
	}
	return slug
}

// findWorkItemFileInAllStatusFolders searches for a work item file by ID across all status folders.
// This is similar to findWorkItemFile but explicitly searches all configured status folders.
func findWorkItemFileInAllStatusFolders(workItemID string, cfg *config.Config) (string, error) {
	workFolder := config.GetWorkFolderPath(cfg)

	// Get all status folder names from config
	statusFolders := getStatusFolders(cfg)

	// Search each status folder
	for _, statusFolder := range statusFolders {
		statusPath := filepath.Join(workFolder, statusFolder)

		// Check if folder exists
		if _, err := os.Stat(statusPath); os.IsNotExist(err) {
			continue
		}

		// Search for work item in this status folder
		foundPath, err := searchWorkItemInFolder(statusPath, workItemID, cfg)
		if err != nil {
			return "", fmt.Errorf("failed to search for work item: %w", err)
		}
		if foundPath != "" {
			return foundPath, nil
		}
	}

	return "", fmt.Errorf("work item with ID %s not found", workItemID)
}

// getStatusFolders returns the list of status folders to search.
func getStatusFolders(cfg *config.Config) []string {
	var statusFolders []string
	for _, folder := range cfg.StatusFolders {
		if folder != "" {
			statusFolders = append(statusFolders, folder)
		}
	}

	// If no status folders configured, use default folders
	if len(statusFolders) == 0 {
		return []string{"0_backlog", "1_todo", "2_doing", "3_review", "4_done"}
	}

	return statusFolders
}

// searchWorkItemInFolder searches for a work item file by ID in a specific folder.
func searchWorkItemInFolder(folderPath, workItemID string, cfg *config.Config) (string, error) {
	var foundPath string
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if this is a work item file with the matching ID
		if !isWorkItemFile(path) {
			return nil
		}

		// Read the file to check the ID
		content, err := safeReadFile(path, cfg)
		if err != nil {
			return err
		}

		// Simple check for ID in front matter (unquoted, double-quoted, or single-quoted)
		if hasWorkItemID(content, workItemID) {
			foundPath = path
			return filepath.SkipDir
		}

		return nil
	})

	return foundPath, err
}

// isWorkItemFile checks if a path is a work item file (not template or IDEAS.md).
func isWorkItemFile(path string) bool {
	return strings.HasSuffix(path, ".md") &&
		!strings.Contains(path, "template") &&
		!strings.HasSuffix(path, "IDEAS.md")
}

// hasWorkItemID checks if content contains the work item ID in front matter.
func hasWorkItemID(content []byte, workItemID string) bool {
	s := string(content)
	return strings.Contains(s, fmt.Sprintf("id: %s", workItemID)) ||
		strings.Contains(s, fmt.Sprintf("id: %q", workItemID)) ||
		strings.Contains(s, fmt.Sprintf("id: '%s'", workItemID))
}

// PRInfo represents information about a pull request
type PRInfo struct {
	Owner    string `json:"owner"`
	Repo     string `json:"repo"`
	PRNumber int    `json:"pr_number"`
	Branch   string `json:"branch"`
}

func runCurrentPRs(_ *cobra.Command, _ []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}

	repoRoot, currentBranch, _, err := getCurrentBranchAndWorkItemID(cfg)
	if err != nil {
		// Invalid branch or work item - output empty array and exit 0
		fmt.Println("[]")
		return nil
	}

	token := getGitHubToken()
	if token == "" {
		// No token available - output empty array and exit 0
		fmt.Println("[]")
		return nil
	}

	baseURL := ""
	if cfg.Workspace != nil {
		baseURL = cfg.Workspace.GitBaseURL
	}

	// Get trunk branch for filtering PRs
	trunkBranch := ""
	if cfg.Git != nil {
		trunkBranch = cfg.Git.TrunkBranch
	}

	prs := discoverPRs(cfg, repoRoot, currentBranch, token, baseURL, trunkBranch)

	// Output JSON array
	jsonOutput, err := json.Marshal(prs)
	if err != nil {
		return fmt.Errorf("failed to marshal PR list: %w", err)
	}
	fmt.Println(string(jsonOutput))
	return nil
}

// getCurrentBranchAndWorkItemID gets the current branch and work item ID, or returns error.
func getCurrentBranchAndWorkItemID(cfg *config.Config) (repoRoot, currentBranch, workItemID string, err error) {
	repoRoot, err = getRepoRoot()
	if err != nil {
		return "", "", "", err
	}

	currentBranch, err = getCurrentBranch(repoRoot)
	if err != nil {
		return "", "", "", err
	}

	workItemID, err = parseWorkItemIDFromBranch(currentBranch, cfg)
	if err != nil {
		return "", "", "", err
	}

	// Check if work item exists (but don't fail if not found - output empty array)
	_, err = findWorkItemFileInAllStatusFolders(workItemID, cfg)
	if err != nil {
		return "", "", "", err
	}

	return repoRoot, currentBranch, workItemID, nil
}

// getGitHubToken returns GITHUB_TOKEN or KIRA_GITHUB_TOKEN, or empty string if neither is set.
func getGitHubToken() string {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("KIRA_GITHUB_TOKEN")
	}
	return token
}

// discoverPRs discovers all related PRs (main repo + project repos in polyrepo).
func discoverPRs(cfg *config.Config, repoRoot, currentBranch, token, baseURL, trunkBranch string) []PRInfo {
	var prs []PRInfo

	// Always include main repo
	mainPR := discoverMainRepoPR(cfg, repoRoot, currentBranch, token, baseURL, trunkBranch)
	if mainPR != nil {
		prs = append(prs, *mainPR)
	}

	// Check if polyrepo workspace
	projects, err := resolvePolyrepoProjects(cfg, repoRoot)
	if err == nil && len(projects) > 0 {
		// Polyrepo workspace - include project repos
		projectPRs := discoverProjectRepoPRs(projects, currentBranch, token, baseURL, trunkBranch)
		prs = append(prs, projectPRs...)
	}

	return prs
}

// discoverMainRepoPR discovers PR in the main repo.
func discoverMainRepoPR(cfg *config.Config, repoRoot, currentBranch, token, baseURL, trunkBranch string) *PRInfo {
	mainRemoteName := resolveRemoteName(cfg, nil)
	mainRemoteURL, err := getRemoteURL(mainRemoteName, repoRoot)
	if err != nil || !isGitHubRemote(mainRemoteURL, baseURL) {
		return nil
	}

	owner, repo, err := git.ParseGitHubOwnerRepo(mainRemoteURL)
	if err != nil {
		return nil
	}

	return findPRForRepo(owner, repo, currentBranch, token, baseURL, trunkBranch)
}

// discoverProjectRepoPRs discovers PRs in all project repos.
func discoverProjectRepoPRs(projects []PolyrepoProject, currentBranch, token, baseURL, globalTrunkBranch string) []PRInfo {
	var prs []PRInfo

	for _, project := range projects {
		if project.Path == "" {
			continue
		}

		projectRemoteURL, err := getRemoteURL(project.Remote, project.Path)
		if err != nil || !isGitHubRemote(projectRemoteURL, baseURL) {
			continue
		}

		owner, repo, err := git.ParseGitHubOwnerRepo(projectRemoteURL)
		if err != nil {
			continue
		}

		// Use project-specific trunk branch if set, otherwise use global
		trunkBranch := project.TrunkBranch
		if trunkBranch == "" {
			trunkBranch = globalTrunkBranch
		}

		pr := findPRForRepo(owner, repo, currentBranch, token, baseURL, trunkBranch)
		if pr != nil {
			prs = append(prs, *pr)
		}
	}

	return prs
}

// findPRForRepo finds a PR for a specific repo and branch, filtering by trunk branch if provided.
func findPRForRepo(owner, repo, branch, token, baseURL, trunkBranch string) *PRInfo {
	prCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	client, err := git.NewClient(prCtx, token, baseURL)
	cancel()
	if err != nil {
		return nil
	}

	prCtx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	githubPRs, err := git.ListPullRequestsByHead(prCtx, client, owner, repo, branch)
	cancel()
	if err != nil || len(githubPRs) == 0 {
		return nil
	}

	// Filter PRs by trunk branch if specified
	var matchingPR *PRInfo
	for _, pr := range githubPRs {
		if pr.Number == nil {
			continue
		}

		// If trunk branch is specified, filter by base branch
		if trunkBranch != "" && pr.Base != nil && pr.Base.Ref != nil {
			if *pr.Base.Ref != trunkBranch {
				continue
			}
		}

		// Found a matching PR
		matchingPR = &PRInfo{
			Owner:    owner,
			Repo:     repo,
			PRNumber: *pr.Number,
			Branch:   branch,
		}
		break
	}

	return matchingPR
}
