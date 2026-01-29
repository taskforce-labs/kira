// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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

		noTrunkUpdate, _ := cmd.Flags().GetBool("no-trunk-update")
		noRebase, _ := cmd.Flags().GetBool("no-rebase")
		trunkUpdateEnabled := !noTrunkUpdate && (cfg.Review == nil || cfg.Review.UpdateTrunkStatus == nil || *cfg.Review.UpdateTrunkStatus)
		rebaseEnabled := trunkUpdateEnabled && !noRebase && (cfg.Review == nil || cfg.Review.RebaseAfterTrunkUpdate == nil || *cfg.Review.RebaseAfterTrunkUpdate)

		// Step 5: Update work item status on current branch (move to review)
		if err := updateWorkItemStatusOnCurrentBranch(cfg, workItemID, statusReview); err != nil {
			return fmt.Errorf("failed to move work item to review: %w", err)
		}

		// Steps 6-7: Validate remote and push branch if needed (before trunk update)
		if err := validateRemoteExists(cfg); err != nil {
			return err
		}
		if err := pushBranchIfNeeded(currentBranch, cfg); err != nil {
			return err
		}

		// Step 17: Update trunk status (if enabled)
		if trunkUpdateEnabled {
			if err := updateTrunkStatus(workItemID, cfg); err != nil {
				return fmt.Errorf("failed to update trunk status: %w", err)
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Work item moved to review on current branch. Trunk branch status updated.")
			if rebaseEnabled {
				if err := performRebase(cfg); err != nil {
					return fmt.Errorf("failed to rebase onto trunk: %w", err)
				}
			}
		} else {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Work item moved to review on current branch.")
		}

		return nil
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

// createGitHubPR creates a new pull request on GitHub using the GitHub API.
// It sets the head branch, base branch, title, description, and draft status.
// Returns the created PR with URL or an error if creation fails.
func createGitHubPR(client *github.Client, owner, repo, branchName, baseBranch, title, description string, isDraft bool) (*github.PullRequest, error) {
	if err := validateCreatePRParams(client, owner, repo, branchName, baseBranch, title); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// Create the new pull request struct
	newPR := &github.NewPullRequest{
		Title: &title,
		Head:  &branchName,
		Base:  &baseBranch,
		Draft: &isDraft,
	}

	// Set body/description if provided (GitHub allows empty descriptions)
	if description != "" {
		newPR.Body = &description
	}

	// Create the PR via GitHub API
	pr, resp, err := client.PullRequests.Create(ctx, owner, repo, newPR)
	if err != nil {
		return nil, handlePRCreateError(err, resp, owner, repo, branchName, baseBranch)
	}

	return pr, nil
}

// validateCreatePRParams validates all input parameters for createGitHubPR.
func validateCreatePRParams(client *github.Client, owner, repo, branchName, baseBranch, title string) error {
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
	if strings.TrimSpace(baseBranch) == "" {
		return fmt.Errorf("base branch cannot be empty")
	}
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("PR title cannot be empty")
	}
	return nil
}

// handlePRCreateError processes errors from the GitHub API PR creation call.
// It provides user-friendly error messages based on HTTP status codes.
func handlePRCreateError(err error, resp *github.Response, owner, repo, branchName, baseBranch string) error {
	if resp == nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("failed to create pull request: authentication failed. Verify GitHub token has 'repo' scope")
	case http.StatusForbidden:
		return fmt.Errorf("failed to create pull request: access forbidden. Verify GitHub token has 'repo' scope")
	case http.StatusNotFound:
		return fmt.Errorf("failed to create pull request: repository '%s/%s' not found", owner, repo)
	case http.StatusUnprocessableEntity:
		// 422 can mean various validation errors
		// Common cases: branch doesn't exist, PR already exists, invalid branch names
		return fmt.Errorf("failed to create pull request: validation error. Check that branch '%s' exists and base branch '%s' is valid", branchName, baseBranch)
	default:
		return fmt.Errorf("failed to create pull request: %w", err)
	}
}

// extractTagsFromWorkItem extracts tags from a work item's Fields map.
// It handles different tag formats (array of strings, array of interface{}, etc.)
// and returns an empty slice if tags field doesn't exist or is invalid.
func extractTagsFromWorkItem(workItem *validation.WorkItem) []string {
	if workItem == nil || workItem.Fields == nil {
		return []string{}
	}

	tagsValue, exists := workItem.Fields["tags"]
	if !exists {
		return []string{}
	}

	var tags []string

	switch v := tagsValue.(type) {
	case []string:
		// Already a slice of strings
		tags = v
	case []interface{}:
		// Convert []interface{} to []string (handles both []interface{} and []any)
		for _, item := range v {
			if str, ok := item.(string); ok {
				tags = append(tags, str)
			}
		}
	default:
		// Unknown type, return empty slice
		return []string{}
	}

	// Filter out empty strings and trim whitespace
	var result []string
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// addPRLabels adds labels to a GitHub pull request based on work item tags.
// It uses a 1:1 mapping (tag → label). If a label doesn't exist on GitHub,
// it logs a warning and skips it (doesn't fail the operation).
func addPRLabels(client *github.Client, owner, repo string, prNumber int, tags []string) error {
	if err := validateAddPRLabelsParams(client, owner, repo, prNumber, tags); err != nil {
		return err
	}

	if len(tags) == 0 {
		// No tags to add, nothing to do
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// Add labels one by one to handle individual failures gracefully
	for _, tag := range tags {
		// Use the tag as the label name (1:1 mapping)
		label := tag
		labels := []string{label}

		_, resp, err := client.Issues.AddLabelsToIssue(ctx, owner, repo, prNumber, labels)
		if err != nil {
			// Handle 404 errors (label doesn't exist) by logging warning and continuing
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				// Label doesn't exist on GitHub - log warning and skip
				_, _ = fmt.Fprintf(os.Stderr, "Warning: label '%s' does not exist on GitHub repository '%s/%s'. Skipping label.\n", label, owner, repo)
				continue
			}

			// For other errors, log but don't fail the entire operation
			// This allows other labels to be added even if one fails
			_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to add label '%s' to PR #%d: %v. Continuing with other labels.\n", label, prNumber, err)
			continue
		}
	}

	return nil
}

// validateAddPRLabelsParams validates all input parameters for addPRLabels.
func validateAddPRLabelsParams(client *github.Client, owner, repo string, prNumber int, tags []string) error {
	if client == nil {
		return fmt.Errorf("GitHub client cannot be nil")
	}
	if strings.TrimSpace(owner) == "" {
		return fmt.Errorf("owner cannot be empty")
	}
	if strings.TrimSpace(repo) == "" {
		return fmt.Errorf("repo cannot be empty")
	}
	if prNumber <= 0 {
		return fmt.Errorf("PR number must be greater than 0")
	}
	if tags == nil {
		return fmt.Errorf("tags cannot be nil")
	}
	return nil
}

// getNumberedUsers collects and sorts users using the same logic as the `kira users` command.
// It returns a slice of UserInfo with assigned numbers (1-based indexing).
// This ensures consistent numbering between `kira users` and reviewer resolution.
func getNumberedUsers(cfg *config.Config) ([]UserInfo, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	useGitHistory := getUseGitHistorySetting(cfg)
	commitLimit := getCommitLimit(0, false, cfg) // Use default limit (0 = no limit)

	userMap, err := collectUsers(useGitHistory, commitLimit, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to collect users: %w", err)
	}

	users := processAndSortUsers(userMap, useGitHistory)
	return users, nil
}

// resolveUserByNumber resolves a user number (from `kira users` command) to an email address.
// User numbers are 1-based (first user is 1, not 0).
// Returns the user's email address or an error if the user number is invalid or not found.
func resolveUserByNumber(userNumber string, cfg *config.Config) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("configuration cannot be nil")
	}

	// Parse user number string to integer
	num, err := strconv.Atoi(userNumber)
	if err != nil {
		return "", fmt.Errorf("invalid user number '%s': must be a positive integer", userNumber)
	}

	// Validate number is positive
	if num <= 0 {
		return "", fmt.Errorf("invalid user number '%s': must be a positive integer", userNumber)
	}

	// Get numbered users using same logic as `kira users` command
	users, err := getNumberedUsers(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to get users: %w", err)
	}

	// Check if any users exist
	if len(users) == 0 {
		return "", fmt.Errorf("no users available. Run 'kira users' to see available users")
	}

	// Look up user by number (1-based indexing)
	if num > len(users) {
		return "", fmt.Errorf("user number '%d' not found. Run 'kira users' to see available users (1-%d)", num, len(users))
	}

	// User numbers are 1-based, so subtract 1 for array index
	user := users[num-1]
	return user.Email, nil
}

// resolveReviewers resolves reviewer specifications to GitHub usernames or email addresses.
// Reviewers can be specified as:
//   - User numbers (digits only): Resolved via `kira users` command (e.g., "1", "2")
//   - Email addresses (contains "@"): Used as-is (e.g., "user@example.com")
//   - GitHub usernames (otherwise): Used as-is (e.g., "octocat")
//
// Returns a slice of resolved reviewer identifiers (emails or usernames).
// Returns an error if any user number cannot be resolved.
func resolveReviewers(reviewerSpecs []string, cfg *config.Config) ([]string, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	// Handle empty slice
	if len(reviewerSpecs) == 0 {
		return []string{}, nil
	}

	var reviewers []string
	userNumberRegex := regexp.MustCompile(`^\d+$`)

	for _, spec := range reviewerSpecs {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue // Skip empty specs
		}

		// Check if spec is a user number (digits only)
		if userNumberRegex.MatchString(spec) {
			// Resolve user number to email
			email, err := resolveUserByNumber(spec, cfg)
			if err != nil {
				return nil, err
			}
			reviewers = append(reviewers, email)
		} else if strings.Contains(spec, "@") {
			// Email address - use as-is
			reviewers = append(reviewers, spec)
		} else {
			// Assume GitHub username - use as-is
			reviewers = append(reviewers, spec)
		}
	}

	return reviewers, nil
}

// requestPRReviews requests reviews from specified reviewers on a GitHub pull request.
// It only requests reviews if cfg.Review.AutoRequestReviews is true (default).
// Returns an error if the API call fails or if validation fails.
func requestPRReviews(client *github.Client, owner, repo string, prNumber int, reviewers []string, cfg *config.Config) error {
	if err := validateRequestPRReviewsParams(client, owner, repo, prNumber, reviewers, cfg); err != nil {
		return err
	}

	// Check if auto-request reviews is enabled
	// Default to true if config.Review is nil or AutoRequestReviews is nil
	autoRequestReviews := true
	if cfg.Review != nil && cfg.Review.AutoRequestReviews != nil {
		autoRequestReviews = *cfg.Review.AutoRequestReviews
	}

	// If auto-request reviews is disabled, skip requesting reviews
	if !autoRequestReviews {
		return nil
	}

	// If no reviewers specified, nothing to do
	if len(reviewers) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// Request reviews using GitHub API
	reviewersRequest := github.ReviewersRequest{
		Reviewers: reviewers,
	}

	_, resp, err := client.PullRequests.RequestReviewers(ctx, owner, repo, prNumber, reviewersRequest)
	if err != nil {
		return handleRequestReviewersError(err, resp, owner, repo, prNumber)
	}

	return nil
}

// validateRequestPRReviewsParams validates all input parameters for requestPRReviews.
func validateRequestPRReviewsParams(client *github.Client, owner, repo string, prNumber int, reviewers []string, cfg *config.Config) error {
	if client == nil {
		return fmt.Errorf("GitHub client cannot be nil")
	}
	if strings.TrimSpace(owner) == "" {
		return fmt.Errorf("owner cannot be empty")
	}
	if strings.TrimSpace(repo) == "" {
		return fmt.Errorf("repo cannot be empty")
	}
	if prNumber <= 0 {
		return fmt.Errorf("PR number must be greater than 0")
	}
	if reviewers == nil {
		return fmt.Errorf("reviewers cannot be nil")
	}
	if cfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	return nil
}

// handleRequestReviewersError processes errors from the GitHub API review request call.
// It provides user-friendly error messages based on HTTP status codes.
func handleRequestReviewersError(err error, resp *github.Response, owner, repo string, prNumber int) error {
	if resp == nil {
		return fmt.Errorf("failed to request reviews: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("failed to request reviews: authentication failed. Verify GitHub token has 'repo' scope")
	case http.StatusForbidden:
		return fmt.Errorf("failed to request reviews: access forbidden. Verify GitHub token has 'repo' scope")
	case http.StatusNotFound:
		return fmt.Errorf("failed to request reviews: pull request #%d not found in repository '%s/%s'", prNumber, owner, repo)
	case http.StatusUnprocessableEntity:
		// 422 can mean various validation errors (invalid reviewers, etc.)
		return fmt.Errorf("failed to request reviews: validation error. Check that reviewers are valid GitHub usernames")
	default:
		return fmt.Errorf("failed to request reviews: %w", err)
	}
}

// updateTrunkStatus updates the work item status on the trunk branch.
// It pulls latest trunk, switches to trunk branch, copies work item if needed,
// updates status to review, commits, and switches back to original branch.
func updateTrunkStatus(workItemID string, cfg *config.Config) error {
	// Validate inputs
	if err := validateUpdateTrunkStatusInputs(workItemID, cfg); err != nil {
		return err
	}

	// Prepare trunk update context
	ctx, err := prepareTrunkUpdateContext(cfg)
	if err != nil {
		return err
	}

	// Pull latest trunk before updating (similar to kira start pattern)
	// This will skip if remote doesn't exist (not an error for local repos)
	if err := pullLatestTrunk(ctx.trunkBranch, ctx.remoteName, ctx.repoRoot, cfg); err != nil {
		return fmt.Errorf("failed to pull latest trunk: %w", err)
	}

	// Stash any uncommitted changes (handle failures gracefully)
	hasStash, err := stashUncommittedChanges(ctx.repoRoot)
	if err != nil {
		return err
	}

	// Switch to trunk branch
	if err := checkoutBranch(ctx.trunkBranch, ctx.repoRoot); err != nil {
		return fmt.Errorf("failed to checkout trunk branch '%s': %w", ctx.trunkBranch, err)
	}

	// Always switch back to original branch, even on error
	defer restoreBranchAndStash(ctx.currentBranch, ctx.repoRoot, hasStash)

	// Perform trunk status update operations
	return performTrunkStatusUpdate(workItemID, ctx, cfg)
}

// trunkUpdateContext holds context for trunk status update operations.
type trunkUpdateContext struct {
	repoRoot      string
	currentBranch string
	trunkBranch   string
	remoteName    string
}

// validateUpdateTrunkStatusInputs validates inputs for updateTrunkStatus.
func validateUpdateTrunkStatusInputs(workItemID string, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	if strings.TrimSpace(workItemID) == "" {
		return fmt.Errorf("work item ID cannot be empty")
	}
	return nil
}

// prepareTrunkUpdateContext prepares the context for trunk status update.
func prepareTrunkUpdateContext(cfg *config.Config) (*trunkUpdateContext, error) {
	repoRoot, err := getRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository root: %w", err)
	}

	currentBranch, err := getCurrentBranch("")
	if err != nil {
		return nil, fmt.Errorf("failed to determine current branch: %w", err)
	}

	trunkBranch, err := determineTrunkBranch(cfg, "", repoRoot, false)
	if err != nil {
		return nil, fmt.Errorf("failed to determine trunk branch: %w", err)
	}

	remoteName := resolveRemoteName(cfg, nil)

	return &trunkUpdateContext{
		repoRoot:      repoRoot,
		currentBranch: currentBranch,
		trunkBranch:   trunkBranch,
		remoteName:    remoteName,
	}, nil
}

// performTrunkStatusUpdate performs the actual trunk status update operations.
func performTrunkStatusUpdate(workItemID string, ctx *trunkUpdateContext, cfg *config.Config) error {
	// Ensure work item exists on trunk (copy if needed)
	if err := ensureWorkItemOnTrunk(workItemID, ctx.currentBranch, ctx.repoRoot); err != nil {
		return err
	}

	// Update work item status to review
	if err := updateWorkItemStatusOnTrunk(workItemID, cfg); err != nil {
		return fmt.Errorf("failed to update trunk status: %w", err)
	}

	// Commit the status update
	if err := commitTrunkStatusUpdate(workItemID, ctx.trunkBranch, ctx.repoRoot, cfg); err != nil {
		return fmt.Errorf("failed to commit trunk status update: %w", err)
	}

	// Push to remote if configured (optional - don't fail if push is disabled or fails)
	// Note: Per PRD, push is optional and should not fail the operation
	_ = pushTrunkStatusUpdate(ctx.trunkBranch, ctx.remoteName, ctx.repoRoot, cfg)

	return nil
}

// pullLatestTrunk pulls the latest changes from the remote trunk branch.
func pullLatestTrunk(trunkBranch, remoteName, repoRoot string, _ *config.Config) error {
	// Check if remote exists
	remoteExists, err := checkRemoteExists(remoteName, repoRoot, false)
	if err != nil {
		// If we can't check, assume remote doesn't exist and skip pull (not an error)
		return nil
	}

	if !remoteExists {
		// No remote configured - skip pull (not an error for local-only repos)
		return nil
	}

	// Use the same pattern as pullLatestChanges from start.go
	// Wrap error to provide context
	if err := pullLatestChanges(remoteName, trunkBranch, repoRoot, false); err != nil {
		// Check if error is due to remote not existing (might have been created between check and fetch)
		if strings.Contains(err.Error(), "No such remote") || strings.Contains(err.Error(), "does not appear to be a git repository") {
			// Remote doesn't exist - skip pull (not an error)
			return nil
		}
		return err
	}

	return nil
}

// workItemExistsOnTrunk checks if a work item exists on the current branch (trunk).
func workItemExistsOnTrunk(workItemID string) (bool, error) {
	_, err := findWorkItemFile(workItemID)
	if err != nil {
		// If work item not found, it doesn't exist
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		// Other errors (permissions, etc.) should be returned
		return false, err
	}
	return true, nil
}

// copyWorkItemFromFeatureBranch copies a work item from the feature branch to the trunk branch.
func copyWorkItemFromFeatureBranch(workItemID, featureBranch, repoRoot string) error {
	// Find work item path on feature branch
	workItemPath, err := findWorkItemPathOnBranch(workItemID, featureBranch, repoRoot)
	if err != nil {
		return err
	}

	// Get file content and write it
	return copyWorkItemFile(workItemID, featureBranch, workItemPath, repoRoot)
}

// findWorkItemPathOnBranch finds the path of a work item on a specific branch.
func findWorkItemPathOnBranch(workItemID, branchName, repoRoot string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// Try to find the work item file path by searching on the feature branch
	lsTreeOutput, err := executeCommand(ctx, "git", []string{"ls-tree", "-r", "--name-only", branchName, ".work"}, repoRoot, false)
	if err != nil {
		// If ls-tree fails, try searching all files
		return findWorkItemPathInAllFiles(workItemID, branchName, repoRoot)
	}

	// Search for the work item file in .work directory
	lines := strings.Split(strings.TrimSpace(lsTreeOutput), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Check if this file contains the work item ID
		if strings.Contains(line, workItemID) && strings.HasSuffix(line, ".md") {
			return line, nil
		}
	}

	return "", fmt.Errorf("work item %s not found on feature branch '%s'", workItemID, branchName)
}

// findWorkItemPathInAllFiles searches all files on a branch for the work item.
func findWorkItemPathInAllFiles(workItemID, branchName, repoRoot string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	allFilesOutput, err := executeCommand(ctx, "git", []string{"ls-tree", "-r", "--name-only", branchName}, repoRoot, false)
	if err != nil {
		return "", fmt.Errorf("failed to list files on feature branch '%s': %w", branchName, err)
	}

	// Search in all files
	lines := strings.Split(strings.TrimSpace(allFilesOutput), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Check if this is a work item file with the matching ID
		if strings.HasPrefix(line, ".work/") && strings.Contains(line, workItemID) && strings.HasSuffix(line, ".md") {
			return line, nil
		}
	}

	return "", fmt.Errorf("work item %s not found on feature branch '%s'", workItemID, branchName)
}

// copyWorkItemFile copies a work item file from a branch to the current working directory.
func copyWorkItemFile(_, branchName, workItemPath, repoRoot string) error { //nolint:revive // workItemID unused but kept for API consistency
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// Get the file content from the feature branch
	showOutput, err := executeCommand(ctx, "git", []string{"show", branchName + ":" + workItemPath}, repoRoot, false)
	if err != nil {
		return fmt.Errorf("failed to get work item content from feature branch: %w", err)
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(workItemPath)
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Validate the path is within .work directory
	if err := validateWorkPath(workItemPath); err != nil {
		return fmt.Errorf("invalid work item path: %w", err)
	}

	// Write the file content
	if err := os.WriteFile(workItemPath, []byte(showOutput), 0o600); err != nil {
		return fmt.Errorf("failed to write work item file: %w", err)
	}

	return nil
}

// updateWorkItemStatusOnTrunk updates the work item status to review on the trunk branch.
func updateWorkItemStatusOnTrunk(workItemID string, cfg *config.Config) error {
	// Use moveWorkItemWithoutCommit to update status
	return moveWorkItemWithoutCommit(cfg, workItemID, statusReview)
}

// commitTrunkStatusUpdate commits the status update on the trunk branch.
func commitTrunkStatusUpdate(workItemID, _, repoRoot string, cfg *config.Config) error {
	// Find the work item file to get metadata
	workItemPath, err := findWorkItemFile(workItemID)
	if err != nil {
		return fmt.Errorf("failed to find work item file: %w", err)
	}

	// Extract metadata for commit message
	workItemType, id, title, currentStatus, err := extractWorkItemMetadata(workItemPath)
	if err != nil {
		return fmt.Errorf("failed to extract work item metadata: %w", err)
	}

	// Build commit message
	subject, body, err := buildCommitMessage(cfg, workItemType, id, title, currentStatus, statusReview)
	if err != nil {
		return fmt.Errorf("failed to build commit message: %w", err)
	}

	// Get old and new paths for commit
	oldPath := workItemPath
	// The file should already be in the review folder after moveWorkItemWithoutCommit
	// But let's verify by finding it again
	newPath, err := findWorkItemFile(workItemID)
	if err != nil {
		return fmt.Errorf("failed to find work item after move: %w", err)
	}

	// Stage the changes
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// Stage both old and new paths (git will handle the move)
	if oldPath != newPath {
		// File was moved - stage both deletion and addition
		if _, err := executeCommand(ctx, "git", []string{"add", oldPath, newPath}, repoRoot, false); err != nil {
			return fmt.Errorf("failed to stage work item changes: %w", err)
		}
	} else {
		// File was updated in place - just stage it
		if _, err := executeCommand(ctx, "git", []string{"add", newPath}, repoRoot, false); err != nil {
			return fmt.Errorf("failed to stage work item changes: %w", err)
		}
	}

	// Commit the changes
	commitCtx, commitCancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer commitCancel()

	commitMsg := subject
	if body != "" {
		commitMsg = subject + "\n\n" + body
	}

	if _, err := executeCommand(commitCtx, "git", []string{"commit", "-m", commitMsg}, repoRoot, false); err != nil {
		return fmt.Errorf("failed to commit status update: %w", err)
	}

	return nil
}

// pushTrunkStatusUpdate pushes the trunk branch changes to remote (optional).
func pushTrunkStatusUpdate(trunkBranch, remoteName, repoRoot string, _ *config.Config) error {
	// Check if remote exists
	remoteExists, err := checkRemoteExists(remoteName, repoRoot, false)
	if err != nil {
		return fmt.Errorf("failed to check remote: %w", err)
	}

	if !remoteExists {
		// No remote configured - skip push (not an error)
		return nil
	}

	// Push to remote
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	if _, err := executeCommand(ctx, "git", []string{"push", remoteName, trunkBranch}, repoRoot, false); err != nil {
		// Push failure is non-fatal per PRD - just log warning
		_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to push trunk status update to %s/%s: %v\n", remoteName, trunkBranch, err)
		return nil // Don't return error - push is optional
	}

	return nil
}

// checkoutBranch switches to the specified branch.
func checkoutBranch(branchName, repoRoot string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	_, err := executeCommand(ctx, "git", []string{"checkout", branchName}, repoRoot, false)
	if err != nil {
		return fmt.Errorf("failed to checkout branch '%s': %w", branchName, err)
	}

	return nil
}

// stashUncommittedChanges stashes uncommitted changes and returns whether a stash was created.
func stashUncommittedChanges(repoRoot string) (bool, error) {
	stashOutput, err := executeCommand(context.Background(), "git", []string{"stash", "push", "-m", "kira-review-stash"}, repoRoot, false)
	if err != nil {
		// Check if stash failed because there were no changes to stash
		// This is not an error - we can continue
		if strings.Contains(err.Error(), "No local changes") || strings.Contains(err.Error(), "nothing to stash") {
			return false, nil
		}
		// Check for uncommitted changes to provide better error message
		hasUncommitted, checkErr := checkUncommittedChanges(repoRoot, false)
		if checkErr == nil && hasUncommitted {
			return false, fmt.Errorf("failed to stash uncommitted changes. Commit or stash changes manually before submitting for review: %w", err)
		}
		// Other stash errors - assume no stash needed
		return false, nil
	}
	// Stash succeeded - we have something to restore later
	return strings.TrimSpace(stashOutput) != "", nil
}

// restoreBranchAndStash restores the original branch and stashed changes.
func restoreBranchAndStash(currentBranch, repoRoot string, hasStash bool) {
	if restoreErr := checkoutBranch(currentBranch, repoRoot); restoreErr != nil {
		// Log error but don't override original error
		_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to switch back to branch '%s': %v\n", currentBranch, restoreErr)
	}

	// Restore stashed changes if any
	if hasStash {
		_, _ = executeCommand(context.Background(), "git", []string{"stash", "pop"}, repoRoot, false)
	}
}

// ensureWorkItemOnTrunk ensures the work item exists on trunk, copying from feature branch if needed.
func ensureWorkItemOnTrunk(workItemID, featureBranch, repoRoot string) error {
	// Check if work item exists on trunk
	exists, err := workItemExistsOnTrunk(workItemID)
	if err != nil {
		return fmt.Errorf("failed to check if work item exists on trunk: %w", err)
	}

	// If work item doesn't exist on trunk, copy from feature branch
	if !exists {
		if err := copyWorkItemFromFeatureBranch(workItemID, featureBranch, repoRoot); err != nil {
			return fmt.Errorf("failed to copy work item to trunk: %w", err)
		}
	}

	return nil
}

// performRebase rebases the current branch onto the trunk branch.
// It is called after trunk status update when rebase is enabled.
// Returns a clear error with resolution instructions on conflict (exit code 1).
func performRebase(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	repoRoot, err := getRepoRoot()
	if err != nil {
		return fmt.Errorf("failed to get repository root: %w", err)
	}

	trunkBranch, err := determineTrunkBranch(cfg, "", repoRoot, false)
	if err != nil {
		return fmt.Errorf("failed to determine trunk branch: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	_, err = executeCommand(ctx, "git", []string{"rebase", trunkBranch}, repoRoot, false)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return fmt.Errorf("rebase conflicts detected. Resolve conflicts manually, then run 'git rebase --continue' and re-run 'kira review'")
		}
		return fmt.Errorf("rebase failed: %w", err)
	}

	return nil
}
