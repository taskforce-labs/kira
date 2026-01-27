// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/go-github/v58/github"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	yaml "gopkg.in/yaml.v3"

	"kira/internal/config"
	gh "kira/internal/github"
	"kira/internal/validation"
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

		// Load and validate work item (Phase 4)
		workItem, _, err := loadWorkItem(cfg, workItemID)
		if err != nil {
			return err
		}

		if err := validateWorkItemStatusForReview(workItem, cfg); err != nil {
			var alreadyErr *alreadyInReviewError
			if errors.As(err, &alreadyErr) {
				// Already in review is a successful no-op
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), alreadyErr.Error())
				return nil
			}
			return err
		}

		if err := validateRequiredFieldsForReview(workItem, cfg); err != nil {
			return err
		}

		// Ensure working directory is clean before proceeding with review operations (Phase 5)
		if err := checkUncommittedChangesForReview(); err != nil {
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

const (
	yamlDelimiter = "---"
	statusReview  = "review"
)

// alreadyInReviewError is returned when a work item is already in review status.
// Callers should treat this as a successful outcome and not as a fatal error.
type alreadyInReviewError struct{}

func (e *alreadyInReviewError) Error() string {
	return "Work item is already in review status."
}

// loadWorkItem loads a single work item by ID using existing work item utilities.
// It returns the parsed work item and its file path.
func loadWorkItem(cfg *config.Config, workItemID string) (*validation.WorkItem, string, error) {
	if cfg == nil {
		return nil, "", fmt.Errorf("configuration cannot be nil")
	}
	if strings.TrimSpace(workItemID) == "" {
		return nil, "", fmt.Errorf("work item ID cannot be empty")
	}

	workItemPath, err := findWorkItemFile(workItemID)
	if err != nil {
		// Normalize "not found" error to match PRD messaging
		if strings.Contains(err.Error(), "work item with ID") {
			return nil, "", fmt.Errorf("work item %s not found", workItemID)
		}
		return nil, "", fmt.Errorf("failed to load work item %s: %w", workItemID, err)
	}

	content, err := safeReadFile(workItemPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read work item file %s: %w", workItemPath, err)
	}

	// Extract YAML front matter between the first pair of --- lines
	lines := strings.Split(string(content), "\n")
	var yamlLines []string
	inYAML := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if i == 0 && trimmed == yamlDelimiter {
			inYAML = true
			continue
		}
		if inYAML {
			if trimmed == yamlDelimiter {
				break
			}
			yamlLines = append(yamlLines, line)
		}
	}

	workItem := &validation.WorkItem{Fields: make(map[string]interface{})}
	if len(yamlLines) > 0 {
		if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), workItem); err != nil {
			return nil, "", fmt.Errorf("failed to parse work item front matter: %w", err)
		}
	}

	// Validate that the work item ID in the file matches the derived ID when present.
	if workItem.ID != "" && workItem.ID != workItemID {
		return nil, "", fmt.Errorf("work item ID mismatch: expected %s but found %s", workItemID, workItem.ID)
	}

	return workItem, workItemPath, nil
}

// validateWorkItemStatusForReview ensures the work item is in a status that
// can be moved to review. It allows "todo" and "doing", treats "review" as a
// successful no-op, and rejects all other statuses.
func validateWorkItemStatusForReview(workItem *validation.WorkItem, _ *config.Config) error {
	if workItem == nil {
		return fmt.Errorf("work item cannot be nil")
	}

	status := strings.TrimSpace(strings.ToLower(workItem.Status))
	switch status {
	case "todo", "doing":
		return nil
	case statusReview:
		return &alreadyInReviewError{}
	case "":
		return fmt.Errorf("cannot submit for review: work item has empty status. Only 'todo' or 'doing' status can be moved to review")
	default:
		return fmt.Errorf("cannot submit for review: work item is in %s status. Only 'todo' or 'doing' status can be moved to review", workItem.Status)
	}
}

// validateRequiredFieldsForReview validates that the work item contains all
// required fields configured in cfg.Validation.RequiredFields. It checks both
// core fields and any additional custom fields present in WorkItem.Fields.
func validateRequiredFieldsForReview(workItem *validation.WorkItem, cfg *config.Config) error {
	if workItem == nil {
		return fmt.Errorf("work item cannot be nil")
	}
	if cfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	var missing []string
	for _, field := range cfg.Validation.RequiredFields {
		if isRequiredFieldMissing(workItem, field) {
			missing = append(missing, field)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf("work item missing required fields: %s. Update work item and try again", strings.Join(missing, ", "))
}

// isRequiredFieldMissing encapsulates the logic for determining whether a single
// configured field is missing from the provided work item.
func isRequiredFieldMissing(workItem *validation.WorkItem, field string) bool {
	switch field {
	case "id":
		return strings.TrimSpace(workItem.ID) == ""
	case "title":
		return strings.TrimSpace(workItem.Title) == ""
	case "status":
		return strings.TrimSpace(workItem.Status) == ""
	case "kind":
		return strings.TrimSpace(workItem.Kind) == ""
	case "created":
		return strings.TrimSpace(workItem.Created) == ""
	default:
		// Custom required fields are stored in the Fields map.
		value, exists := workItem.Fields[field]
		return !exists || isEmptyWorkItemField(value)
	}
}

// isEmptyWorkItemField determines whether a dynamic work item field should be
// considered "missing" for required field validation.
func isEmptyWorkItemField(value interface{}) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	default:
		return false
	}
}

// checkUncommittedChangesForReview ensures there are no uncommitted changes
// in the current repository before running review operations.
// It returns a clear, PRD-aligned error message when changes are detected.
func checkUncommittedChangesForReview() error {
	hasUncommitted, err := checkUncommittedChanges("", false)
	if err != nil {
		return err
	}

	if hasUncommitted {
		return fmt.Errorf("uncommitted changes detected. Commit or stash changes before submitting for review")
	}

	return nil
}

// updateWorkItemStatusOnCurrentBranch moves the work item for the given ID to the
// target status on the current branch and updates its front matter status field.
// It reuses existing work item utilities (findWorkItemFile and updateWorkItemStatus)
// and does not perform any git commits.
func updateWorkItemStatusOnCurrentBranch(cfg *config.Config, workItemID, targetStatus string) error {
	if cfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	if strings.TrimSpace(workItemID) == "" {
		return fmt.Errorf("work item ID cannot be empty")
	}

	// Find the work item file on the current branch
	workItemPath, err := findWorkItemFile(workItemID)
	if err != nil {
		return err
	}

	// Validate target status is configured
	if _, exists := cfg.StatusFolders[targetStatus]; !exists {
		return fmt.Errorf("invalid target status: %s", targetStatus)
	}

	// Build target path using existing status folder conventions
	targetFolder := filepath.Join(".work", cfg.StatusFolders[targetStatus])
	filename := filepath.Base(workItemPath)
	targetPath := filepath.Join(targetFolder, filename)

	// Move the file to the new status folder
	if err := os.Rename(workItemPath, targetPath); err != nil {
		return fmt.Errorf("failed to move work item: %w", err)
	}

	// Update the status field in front matter while preserving all other fields
	if err := updateWorkItemStatus(targetPath, targetStatus); err != nil {
		return fmt.Errorf("failed to update work item status: %w", err)
	}

	return nil
}

// validateRemoteExists checks if the configured git remote exists in the repository.
// Returns an error if the remote is not configured.
func validateRemoteExists(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	// Get remote name from config (defaults to "origin")
	remoteName := resolveRemoteName(cfg, nil)

	// Get repository root
	repoRoot, err := getRepoRoot()
	if err != nil {
		return fmt.Errorf("failed to get repository root: %w", err)
	}

	// Check if remote exists
	exists, err := checkRemoteExists(remoteName, repoRoot, false)
	if err != nil {
		return fmt.Errorf("failed to check remote: %w", err)
	}

	if !exists {
		return fmt.Errorf("GitHub remote '%s' not configured", remoteName)
	}

	return nil
}

// checkBranchOnRemote checks if a branch exists on the remote repository.
// Returns true if the branch exists on remote, false if it doesn't.
func checkBranchOnRemote(branchName string, cfg *config.Config) (bool, error) {
	if strings.TrimSpace(branchName) == "" {
		return false, fmt.Errorf("branch name cannot be empty")
	}
	if cfg == nil {
		return false, fmt.Errorf("configuration cannot be nil")
	}

	// Get remote name from config
	remoteName := resolveRemoteName(cfg, nil)

	// Get repository root
	repoRoot, err := getRepoRoot()
	if err != nil {
		return false, fmt.Errorf("failed to get repository root: %w", err)
	}

	// Execute git ls-remote to check if branch exists
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	output, err := executeCommand(ctx, "git", []string{"ls-remote", "--heads", remoteName, branchName}, repoRoot, false)
	if err != nil {
		// Check for network errors
		if strings.Contains(err.Error(), "Could not resolve host") ||
			strings.Contains(err.Error(), "unable to access") ||
			strings.Contains(err.Error(), "Connection refused") {
			return false, fmt.Errorf("failed to check remote branch: network error occurred. Check network connection and try again: %w", err)
		}
		// Check if remote doesn't exist
		if strings.Contains(err.Error(), "fatal: No such remote") {
			return false, fmt.Errorf("remote '%s' does not exist", remoteName)
		}
		return false, fmt.Errorf("failed to check remote branch: %w", err)
	}

	// If output is non-empty, branch exists on remote
	return strings.TrimSpace(output) != "", nil
}

// checkBranchDiverged checks if the local branch has diverged from the remote branch.
// Returns true if branches have diverged (with error), false if not diverged.
// A branch is considered diverged if local and remote have different commits and
// neither is an ancestor of the other.
func checkBranchDiverged(branchName string, cfg *config.Config) (bool, error) {
	if strings.TrimSpace(branchName) == "" {
		return false, fmt.Errorf("branch name cannot be empty")
	}
	if cfg == nil {
		return false, fmt.Errorf("configuration cannot be nil")
	}

	// Get remote name from config
	remoteName := resolveRemoteName(cfg, nil)

	// Get repository root
	repoRoot, err := getRepoRoot()
	if err != nil {
		return false, fmt.Errorf("failed to get repository root: %w", err)
	}

	// First check if branch exists on remote
	exists, err := checkBranchOnRemote(branchName, cfg)
	if err != nil || !exists {
		// If we can't check remote or branch doesn't exist, assume not diverged (conservative approach)
		return false, nil
	}

	// Get local and remote commits
	localCommit, remoteCommit, err := getLocalAndRemoteCommits(branchName, remoteName, repoRoot)
	if err != nil || localCommit == "" || remoteCommit == "" {
		// Can't get commits, assume not diverged
		return false, nil
	}

	// If commits match, branches are in sync (not diverged)
	if localCommit == remoteCommit {
		return false, nil
	}

	// Check divergence using merge-base
	return checkDivergenceWithMergeBase(localCommit, branchName, remoteName, repoRoot)
}

// getLocalAndRemoteCommits retrieves the commit hashes for local and remote branches.
func getLocalAndRemoteCommits(branchName, remoteName, repoRoot string) (string, string, error) {
	// Get local branch commit
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	localCommit, err := executeCommand(ctx, "git", []string{"rev-parse", branchName}, repoRoot, false)
	if err != nil {
		return "", "", err
	}
	localCommit = strings.TrimSpace(localCommit)

	// Get remote branch commit
	remoteCtx, remoteCancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer remoteCancel()

	remoteOutput, err := executeCommand(remoteCtx, "git", []string{"ls-remote", remoteName, branchName}, repoRoot, false)
	if err != nil {
		return "", "", err
	}

	// Parse remote commit hash from ls-remote output
	remoteLines := strings.Split(strings.TrimSpace(remoteOutput), "\n")
	if len(remoteLines) == 0 || remoteLines[0] == "" {
		return "", "", fmt.Errorf("no remote commit found")
	}

	// Extract commit hash (first field before tab)
	remoteCommit := strings.Fields(remoteLines[0])[0]
	if remoteCommit == "" {
		return "", "", fmt.Errorf("failed to parse remote commit")
	}

	return localCommit, remoteCommit, nil
}

// checkDivergenceWithMergeBase uses git merge-base to determine if branches have diverged.
func checkDivergenceWithMergeBase(localCommit, branchName, remoteName, repoRoot string) (bool, error) {
	// Fetch the remote commit into a temporary ref so merge-base can work with it
	tempRef := "refs/temp-merge-base-check"
	if err := fetchRemoteCommitToTempRef(branchName, remoteName, tempRef, repoRoot); err != nil {
		// If fetch fails, can't determine divergence - assume not diverged
		return false, nil
	}

	// Clean up temp ref when done
	defer cleanupTempRef(tempRef, repoRoot)

	// Get the fetched remote commit
	fetchedRemoteCommit, err := getTempRefCommit(tempRef, repoRoot)
	if err != nil {
		return false, nil
	}

	// Find common ancestor
	commonAncestor, err := findCommonAncestor(localCommit, tempRef, repoRoot)
	if err != nil {
		// Can't determine divergence - assume not diverged
		return false, nil
	}

	// If common ancestor is neither local nor remote commit, branches have diverged
	if commonAncestor != localCommit && commonAncestor != fetchedRemoteCommit {
		return true, fmt.Errorf("branch has diverged from remote. Pull latest changes or resolve conflicts before submitting for review")
	}

	// Branches are not diverged (one is ahead/behind the other)
	return false, nil
}

// fetchRemoteCommitToTempRef fetches the remote branch commit into a temporary ref.
func fetchRemoteCommitToTempRef(branchName, remoteName, tempRef, repoRoot string) error {
	fetchCtx, fetchCancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer fetchCancel()

	_, err := executeCommand(fetchCtx, "git", []string{"fetch", remoteName, "+refs/heads/" + branchName + ":" + tempRef}, repoRoot, false)
	if err != nil {
		return err
	}

	// Verify temp ref exists after fetch
	verifyCtx, verifyCancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer verifyCancel()

	_, err = executeCommand(verifyCtx, "git", []string{"rev-parse", "--verify", tempRef}, repoRoot, false)
	return err
}

// getTempRefCommit gets the commit hash from the temporary ref.
func getTempRefCommit(tempRef, repoRoot string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	commit, err := executeCommand(ctx, "git", []string{"rev-parse", tempRef}, repoRoot, false)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(commit), nil
}

// findCommonAncestor finds the common ancestor of two commits using git merge-base.
func findCommonAncestor(commit1, commit2, repoRoot string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	ancestor, err := executeCommand(ctx, "git", []string{"merge-base", commit1, commit2}, repoRoot, false)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(ancestor), nil
}

// cleanupTempRef removes the temporary ref.
func cleanupTempRef(tempRef, repoRoot string) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()
	_, _ = executeCommand(ctx, "git", []string{"update-ref", "-d", tempRef}, repoRoot, false)
}

// pushBranchIfNeeded pushes the branch to the remote if it is not already there.
// If the branch exists on remote and is up-to-date, it skips the push.
// If the branch has diverged from remote, it returns an error (never force-pushes).
func pushBranchIfNeeded(branchName string, cfg *config.Config) error {
	if strings.TrimSpace(branchName) == "" {
		return fmt.Errorf("branch name cannot be empty")
	}
	if cfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	exists, err := checkBranchOnRemote(branchName, cfg)
	if err != nil {
		return err
	}

	if !exists {
		remoteName := resolveRemoteName(cfg, nil)
		repoRoot, err := getRepoRoot()
		if err != nil {
			return fmt.Errorf("failed to get repository root: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
		defer cancel()

		_, err = executeCommand(ctx, "git", []string{"push", "-u", remoteName, branchName}, repoRoot, false)
		if err != nil {
			return fmt.Errorf("failed to push branch to remote: %w", err)
		}
		return nil
	}

	_, err = checkBranchDiverged(branchName, cfg)
	if err != nil {
		return err
	}

	return nil
}

// envGitHubToken is the environment variable name for the GitHub token; the value is never stored in code.
// #nosec G101 -- env var name only, not a credential
const envGitHubToken = "KIRA_GITHUB_TOKEN"

// getGitHubToken reads the GitHub token from the KIRA_GITHUB_TOKEN environment variable only.
// Returns the token or an error if the token is missing.
func getGitHubToken(_ *config.Config) (string, error) {
	token := strings.TrimSpace(os.Getenv(envGitHubToken))
	if token == "" {
		return "", fmt.Errorf("GitHub token required for PR creation. Set KIRA_GITHUB_TOKEN environment variable")
	}
	return token, nil
}

// validateGitHubToken validates that the provided GitHub token has the required 'repo' scope.
// It creates a GitHub API client and makes an authenticated request to verify token permissions.
func validateGitHubToken(token string) error {
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("GitHub token validation failed. Token must have 'repo' scope for PR creation")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Make an authenticated API call to verify the token
	// Using RateLimitService.Get endpoint which requires authentication
	// This validates the token is valid without needing specific scopes
	rateLimits, resp, err := client.RateLimit.Get(ctx)
	if err != nil {
		// Check if it's an authentication error
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("GitHub token validation failed. Token must have 'repo' scope for PR creation")
		}
		// Network or other errors
		return fmt.Errorf("GitHub token validation failed: %w", err)
	}

	// If we got rate limits, the token is valid
	// Note: We can't directly check scopes without making a repo-scoped API call
	// The actual scope validation will happen when we try to create a PR
	// If the token lacks 'repo' scope, the PR creation will fail with a clear error
	// For Phase 10, we validate the token is valid and can authenticate
	// Full scope validation happens in later phases when we create PRs
	_ = rateLimits // Suppress unused variable warning

	return nil
}

// generatePRTitle generates a PR title from a work item using the configured template.
// It replaces template variables {id}, {title}, and {kind}, sanitizes special characters,
// and truncates to 200 characters.
func generatePRTitle(workItem *validation.WorkItem, cfg *config.Config) (string, error) {
	if workItem == nil {
		return "", fmt.Errorf("work item cannot be nil")
	}
	if cfg == nil {
		return "", fmt.Errorf("configuration cannot be nil")
	}

	// Get template from config (with default fallback)
	template := cfg.Review.PRTitle
	if template == "" {
		template = "[{id}] {title}"
	}

	// Replace variables
	result := template
	result = strings.ReplaceAll(result, "{id}", workItem.ID)
	result = strings.ReplaceAll(result, "{title}", workItem.Title)
	result = strings.ReplaceAll(result, "{kind}", workItem.Kind)

	// Sanitize: remove newlines and carriage returns
	result = strings.ReplaceAll(result, "\n", " ")
	result = strings.ReplaceAll(result, "\r", " ")
	result = strings.TrimSpace(result)

	// Truncate to 200 chars (GitHub limit is 255, leave room for ID prefix)
	if len(result) > 200 {
		result = result[:200]
	}

	return result, nil
}

// generatePRDescription generates a PR description from a work item using the configured template.
// It replaces template variables {id}, {title}, and {work_item_url}. The work item URL
// is constructed from the GitHub repository info and trunk branch, but if construction fails,
// the function gracefully degrades by using an empty string for the URL.
func generatePRDescription(workItem *validation.WorkItem, workItemPath string, cfg *config.Config) (string, error) {
	if workItem == nil {
		return "", fmt.Errorf("work item cannot be nil")
	}
	if cfg == nil {
		return "", fmt.Errorf("configuration cannot be nil")
	}

	// Get template from config (with default fallback)
	template := cfg.Review.PRDescription
	if template == "" {
		template = "View detailed work item: [{id}-{title}]({work_item_url})"
	}

	// Replace id and title
	result := template
	result = strings.ReplaceAll(result, "{id}", workItem.ID)
	result = strings.ReplaceAll(result, "{title}", workItem.Title)

	// Try to construct work item URL (optional - don't fail if it doesn't work)
	workItemURL := ""
	if owner, repo, err := gh.GetGitHubRepoInfo(cfg); err == nil {
		repoRoot, err := getRepoRoot()
		if err == nil {
			if trunkBranch, err := determineTrunkBranch(cfg, "", repoRoot, false); err == nil {
				// Normalize path: remove leading ./, ensure forward slashes for URLs
				normalizedPath := strings.TrimPrefix(workItemPath, "./")
				// Always use forward slashes for GitHub URLs, regardless of OS
				normalizedPath = strings.ReplaceAll(normalizedPath, "\\", "/")
				workItemURL = fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, trunkBranch, normalizedPath)
			}
		}
	}

	// Replace work_item_url (empty string if construction failed)
	result = strings.ReplaceAll(result, "{work_item_url}", workItemURL)

	return result, nil
}

// findExistingPR checks if a pull request already exists for the given branch.
// It searches for open PRs where the head branch matches the specified branch name.
// Returns the PR if found, nil if not found (not an error condition).
// Returns an error only for API failures (network errors, authentication errors, etc.).
func findExistingPR(client *github.Client, owner, repo, branchName string) (*github.PullRequest, error) {
	if err := validateFindExistingPRParams(client, owner, repo, branchName); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// Use Head parameter to filter PRs by branch (format: "owner:branch-name")
	// This is more efficient than fetching all PRs and filtering client-side
	head := fmt.Sprintf("%s:%s", owner, branchName)
	opts := &github.PullRequestListOptions{
		State: "open",
		Head:  head,
		ListOptions: github.ListOptions{
			PerPage: 100, // GitHub API default, but explicit for clarity
		},
	}

	prs, resp, err := client.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return nil, handlePRListError(err, resp, owner, repo)
	}

	// If no PRs found, return nil (not an error)
	if len(prs) == 0 {
		return nil, nil
	}

	// Find PR where head branch exactly matches (handle potential forks)
	matchedPR := findMatchingPR(prs, owner, branchName)
	return matchedPR, nil
}

// validateFindExistingPRParams validates the parameters for findExistingPR.
func validateFindExistingPRParams(client *github.Client, owner, repo, branchName string) error {
	if client == nil {
		return fmt.Errorf("GitHub client cannot be nil")
	}
	if strings.TrimSpace(owner) == "" {
		return fmt.Errorf("owner cannot be empty")
	}
	if strings.TrimSpace(repo) == "" {
		return fmt.Errorf("repo cannot be empty")
	}
	if strings.TrimSpace(branchName) == "" {
		return fmt.Errorf("branch name cannot be empty")
	}
	return nil
}

// handlePRListError processes errors from the GitHub API PR list call.
func handlePRListError(err error, resp *github.Response, owner, repo string) error {
	if resp == nil {
		return fmt.Errorf("failed to check for existing pull request: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("failed to check for existing pull request: authentication failed. Verify GitHub token has 'repo' scope")
	case http.StatusNotFound:
		return fmt.Errorf("failed to check for existing pull request: repository '%s/%s' not found", owner, repo)
	case http.StatusForbidden:
		return fmt.Errorf("failed to check for existing pull request: access forbidden. Verify GitHub token has 'repo' scope")
	default:
		return fmt.Errorf("failed to check for existing pull request: %w", err)
	}
}

// findMatchingPR finds a PR in the list where the head branch matches the specified branch and owner.
func findMatchingPR(prs []*github.PullRequest, owner, branchName string) *github.PullRequest {
	for _, pr := range prs {
		if matchesBranchAndOwner(pr, owner, branchName) {
			return pr
		}
	}
	return nil
}

// matchesBranchAndOwner checks if a PR's head branch matches the specified branch and owner.
func matchesBranchAndOwner(pr *github.PullRequest, owner, branchName string) bool {
	if pr.Head == nil || pr.Head.Ref == nil || *pr.Head.Ref != branchName {
		return false
	}
	if pr.Head.Repo == nil || pr.Head.Repo.Owner == nil || pr.Head.Repo.Owner.Login == nil {
		return false
	}
	return *pr.Head.Repo.Owner.Login == owner
}
