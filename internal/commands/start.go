// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"kira/internal/config"
)

// BranchStatus represents the status of a branch
type BranchStatus int

const (
	// BranchNotExists indicates the branch does not exist
	BranchNotExists BranchStatus = iota
	// BranchPointsToTrunk indicates the branch exists but points to trunk (no commits)
	BranchPointsToTrunk
	// BranchHasCommits indicates the branch exists and has commits beyond trunk
	BranchHasCommits
)

// defaultTrunkBranch is the default trunk branch name used in dry-run mode
const defaultTrunkBranch = "main"

// defaultMasterBranch is the fallback trunk branch name
const defaultMasterBranch = "master"

// statusActionNone is the status_action value that skips all status operations
const statusActionNone = "none"

// statusActionCommitAndPush commits and pushes status change on trunk
const statusActionCommitAndPush = "commit_and_push"

// statusActionCommitOnlyBranch commits status change on the new branch
const statusActionCommitOnlyBranch = "commit_only_branch"

// WorkspaceBehavior represents the inferred workspace type
type WorkspaceBehavior int

const (
	// WorkspaceBehaviorStandalone indicates a single repository workspace
	WorkspaceBehaviorStandalone WorkspaceBehavior = iota
	// WorkspaceBehaviorMonorepo indicates a monorepo (projects list for LLM context only)
	WorkspaceBehaviorMonorepo
	// WorkspaceBehaviorPolyrepo indicates multiple separate repositories
	WorkspaceBehaviorPolyrepo
)

// String returns the string representation of WorkspaceBehavior
func (w WorkspaceBehavior) String() string {
	switch w {
	case WorkspaceBehaviorStandalone:
		return "standalone"
	case WorkspaceBehaviorMonorepo:
		return "monorepo"
	case WorkspaceBehaviorPolyrepo:
		return "polyrepo"
	default:
		return "unknown"
	}
}

// WorktreeStatus represents the status of a worktree path
type WorktreeStatus int

const (
	// WorktreeNotExists indicates the path does not exist
	WorktreeNotExists WorktreeStatus = iota
	// WorktreeValidSameItem indicates a valid worktree for the same work item
	WorktreeValidSameItem
	// WorktreeValidDifferentItem indicates a valid worktree for a different work item
	WorktreeValidDifferentItem
	// WorktreeInvalidPath indicates the path exists but is not a valid git worktree
	WorktreeInvalidPath
)

// StartFlags holds all flags for the start command
type StartFlags struct {
	DryRun          bool
	Override        bool
	SkipStatusCheck bool
	ReuseBranch     bool
	NoIDE           bool
	NoDraftPR       bool
	IDECommand      string
	TrunkBranch     string
	StatusAction    string
}

// StartContext holds all validated inputs for the start command
type StartContext struct {
	WorkItemID       string
	WorkItemPath     string
	Metadata         workItemMetadata
	SanitizedTitle   string
	BranchName       string
	WorktreeRoot     string
	WorktreePaths    []string // For polyrepo
	Behavior         WorkspaceBehavior
	Config           *config.Config
	Flags            StartFlags
	SkipStatusUpdate bool // Set when --skip-status-check is used and status matches target
}

// Maximum length for sanitized title before truncation
const maxTitleLength = 100

var startCmd = &cobra.Command{
	Use:   "start <work-item-id>",
	Short: "Create a git worktree for parallel development work",
	Long: `Creates a git worktree from the trunk branch for the specified work item,
enabling isolated development work. This is especially useful for agentic workflows
where multiple tasks are worked on simultaneously.

The command will:
1. Validate the work item exists
2. Pull latest changes from origin on trunk branch
3. Optionally move the work item to "doing" status
4. Create a git worktree and branch
5. Open your IDE in the worktree (if configured)
6. Run setup commands (if configured)`,
	Args: cobra.ExactArgs(1),
	RunE: runStart,
}

func init() {
	startCmd.Flags().Bool("dry-run", false, "Preview what would be done without executing")
	startCmd.Flags().Bool("override", false, "Remove existing worktree if it exists")
	startCmd.Flags().Bool("skip-status-check", false, "Skip status validation (allow starting work item already in target status)")
	startCmd.Flags().Bool("reuse-branch", false, "Checkout existing branch in new worktree if branch exists")
	startCmd.Flags().Bool("no-ide", false, "Skip IDE opening (useful for agents)")
	startCmd.Flags().Bool("no-draft-pr", false, "Skip pushing branch and creating draft PR")
	startCmd.Flags().String("ide", "", "Override IDE command (e.g., --ide cursor)")
	startCmd.Flags().String("trunk-branch", "", "Override trunk branch (e.g., --trunk-branch develop)")
	startCmd.Flags().String("status-action", "", "Override status action (none|commit_only|commit_and_push|commit_only_branch)")
}

func runStart(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}

	workItemID := args[0]

	// Parse flags
	flags := StartFlags{}
	flags.DryRun, _ = cmd.Flags().GetBool("dry-run")
	flags.Override, _ = cmd.Flags().GetBool("override")
	flags.SkipStatusCheck, _ = cmd.Flags().GetBool("skip-status-check")
	flags.ReuseBranch, _ = cmd.Flags().GetBool("reuse-branch")
	flags.NoIDE, _ = cmd.Flags().GetBool("no-ide")
	flags.NoDraftPR, _ = cmd.Flags().GetBool("no-draft-pr")
	flags.IDECommand, _ = cmd.Flags().GetString("ide")
	flags.TrunkBranch, _ = cmd.Flags().GetString("trunk-branch")
	flags.StatusAction, _ = cmd.Flags().GetString("status-action")

	// Validate status-action flag if provided
	if flags.StatusAction != "" {
		valid := false
		for _, action := range config.ValidStatusActions {
			if flags.StatusAction == action {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid status_action value '%s': use one of: %s",
				flags.StatusAction, strings.Join(config.ValidStatusActions, ", "))
		}
	}

	// Build and validate start context
	ctx, err := buildStartContext(cfg, workItemID, flags)
	if err != nil {
		return err
	}

	// If dry-run, show preview and exit
	if flags.DryRun {
		return printDryRunPreview(ctx)
	}

	// Execute git operations
	return executeGitOperations(ctx)
}

// executeGitOperations performs all git operations for the start command
func executeGitOperations(ctx *StartContext) error {
	repoRoot, err := getRepoRoot()
	if err != nil {
		return fmt.Errorf("not a git repository: current directory is not a git repository. Run this command from within a git repository")
	}

	// Step 1: Determine trunk branch
	trunkBranch, err := determineTrunkBranch(ctx.Config, ctx.Flags.TrunkBranch, repoRoot, ctx.Flags.DryRun)
	if err != nil {
		return err
	}
	fmt.Printf("Using trunk branch: %s\n", trunkBranch)

	// Step 2: Validate we're on trunk branch
	if err := validateOnTrunkBranch(trunkBranch, repoRoot, ctx.Flags.DryRun); err != nil {
		return err
	}

	// Step 3: Resolve remote name
	remoteName := resolveRemoteName(ctx.Config, nil)

	// Step 4: Check for uncommitted changes and pull latest
	if err := validateAndPullLatest(ctx, repoRoot, trunkBranch, remoteName); err != nil {
		return err
	}

	// Step 5: Check work item status (after pull to ensure up-to-date status)
	if err := performStatusCheck(ctx); err != nil {
		return err
	}

	// Step 6: Status update for commit_only/commit_and_push (before worktree creation)
	if err := performStatusUpdate(ctx, repoRoot, trunkBranch, remoteName); err != nil {
		return err
	}

	// Step 7: Create worktrees and handle post-worktree status update
	return createWorktreesAndFinalize(ctx, trunkBranch)
}

// validateAndPullLatest checks for uncommitted changes and pulls latest from remote
func validateAndPullLatest(ctx *StartContext, repoRoot, trunkBranch, remoteName string) error {
	hasUncommitted, err := checkUncommittedChanges(repoRoot, ctx.Flags.DryRun)
	if err != nil {
		return err
	}
	if hasUncommitted {
		return fmt.Errorf("trunk branch has uncommitted changes: cannot proceed with pull operation. Commit or stash changes before starting work")
	}

	if ctx.Behavior == WorkspaceBehaviorPolyrepo {
		return pullAllProjects(ctx, trunkBranch, remoteName)
	}

	return pullStandaloneOrMonorepo(ctx, repoRoot, trunkBranch, remoteName)
}

// pullStandaloneOrMonorepo pulls latest changes for standalone/monorepo workspaces
func pullStandaloneOrMonorepo(ctx *StartContext, repoRoot, trunkBranch, remoteName string) error {
	remoteExists, err := checkRemoteExists(remoteName, repoRoot, ctx.Flags.DryRun)
	if err != nil {
		return err
	}

	if !remoteExists {
		fmt.Printf("Warning: No remote '%s' configured. Skipping pull step. Worktree will be created from local trunk branch.\n", remoteName)
		return nil
	}

	fmt.Printf("Pulling latest changes from %s/%s\n", remoteName, trunkBranch)
	return pullLatestChanges(remoteName, trunkBranch, repoRoot, ctx.Flags.DryRun)
}

// createWorktreesAndFinalize creates worktrees and handles status update on branch
func createWorktreesAndFinalize(ctx *StartContext, trunkBranch string) error {
	if err := os.MkdirAll(ctx.WorktreeRoot, 0o700); err != nil {
		return fmt.Errorf("failed to create worktree root directory: %w", err)
	}

	worktreePath := filepath.Join(ctx.WorktreeRoot, ctx.BranchName)
	if ctx.Behavior == WorkspaceBehaviorPolyrepo {
		if err := executePolyrepoStart(ctx, trunkBranch); err != nil {
			return err
		}
		worktreePath = filepath.Join(worktreePath, "main")
	} else {
		if err := executeStandaloneStart(ctx, trunkBranch); err != nil {
			return err
		}
	}

	// Status update for commit_only_branch (after worktree creation)
	if err := performStatusUpdateOnBranch(ctx, worktreePath); err != nil {
		return err
	}

	displayPath := filepath.Join(ctx.WorktreeRoot, ctx.BranchName)
	fmt.Printf("\nSuccessfully started work on %s\n", ctx.WorkItemID)
	fmt.Printf("  Worktree: %s\n", displayPath)
	fmt.Printf("  Branch: %s\n", ctx.BranchName)

	// Step 9: Launch IDE (before setup commands)
	// IDE opens first so user can start working while setup runs
	launchIDE(ctx, displayPath)

	// Step 10: Run setup commands (after IDE opening)
	if err := executeSetupCommands(ctx, displayPath); err != nil {
		return err
	}

	return nil
}

// buildStartContext validates all inputs and builds a StartContext
func buildStartContext(cfg *config.Config, workItemID string, flags StartFlags) (*StartContext, error) {
	ctx := &StartContext{
		WorkItemID: workItemID,
		Config:     cfg,
		Flags:      flags,
	}

	// Step 1: Validate work item ID format
	if err := validateWorkItemID(workItemID, cfg); err != nil {
		return nil, err
	}

	// Step 2: Find and validate work item exists
	workItemPath, err := findWorkItemFile(workItemID, cfg)
	if err != nil {
		return nil, fmt.Errorf("work item '%s' not found: no work item file exists with that ID", workItemID)
	}
	ctx.WorkItemPath = workItemPath

	// Step 3: Extract work item metadata
	workItemType, id, title, currentStatus, repos, err := extractWorkItemMetadata(workItemPath, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to read work item '%s': could not extract metadata: %w", workItemID, err)
	}
	ctx.Metadata = workItemMetadata{
		workItemType:  workItemType,
		id:            id,
		title:         title,
		currentStatus: currentStatus,
		repos:         repos,
	}

	// Step 4: Sanitize title for branch/worktree name
	sanitizedTitle, err := sanitizeTitle(title, workItemID)
	if err != nil {
		return nil, err
	}
	ctx.SanitizedTitle = sanitizedTitle

	// Step 5: Build branch name
	ctx.BranchName = fmt.Sprintf("%s-%s", workItemID, sanitizedTitle)

	// Step 6: Infer workspace behavior
	ctx.Behavior = inferWorkspaceBehavior(cfg)

	// Step 7: Derive worktree root
	worktreeRoot, err := deriveWorktreeRoot(cfg, ctx.Behavior)
	if err != nil {
		return nil, err
	}
	ctx.WorktreeRoot = worktreeRoot

	// Note: Status check is performed in executeGitOperations after git pull (step 5)
	// to ensure we're checking against the most up-to-date status

	return ctx, nil
}

// validateWorkItemID validates the work item ID format and protects against path traversal
func validateWorkItemID(id string, cfg *config.Config) error {
	// Check for path traversal attempts
	if strings.Contains(id, "..") || strings.Contains(id, "/") || strings.Contains(id, "\\") {
		return fmt.Errorf("invalid work item ID '%s': ID format is invalid (expected format: %s)", id, cfg.Validation.IDFormat)
	}

	// Validate against configured ID format
	matched, err := regexp.MatchString(cfg.Validation.IDFormat, id)
	if err != nil {
		return fmt.Errorf("invalid ID format regex in configuration: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid work item ID '%s': ID format is invalid (expected format: %s)", id, cfg.Validation.IDFormat)
	}

	return nil
}

// sanitizeTitle sanitizes a work item title for use in branch/directory names
func sanitizeTitle(title, workItemID string) (string, error) {
	// Handle missing or empty title
	if title == "" || title == unknownValue {
		fmt.Printf("Warning: Work item %s has no title field. Using work item ID '%s' for worktree directory and branch name.\n", workItemID, workItemID)
		return "", nil // Empty string means use just the ID
	}

	// Apply kebab-case conversion (same algorithm as kira new)
	sanitized := kebabCase(title)

	// Check if sanitization resulted in empty string
	if sanitized == "" || sanitized == "-" {
		return "", fmt.Errorf("work item '%s' title sanitization resulted in empty string: please update the title to include valid characters", workItemID)
	}

	// Remove leading/trailing hyphens
	sanitized = strings.Trim(sanitized, "-")

	// If still empty after trimming
	if sanitized == "" {
		return "", fmt.Errorf("work item '%s' title sanitization resulted in empty string: please update the title to include valid characters", workItemID)
	}

	// Truncate if too long, add hash for uniqueness
	if len(sanitized) > maxTitleLength {
		// Calculate hash of full sanitized title for uniqueness
		hash := sha256.Sum256([]byte(sanitized))
		hashSuffix := fmt.Sprintf("-%x", hash[:3]) // 6 hex chars

		// Truncate and append hash
		sanitized = sanitized[:maxTitleLength-len(hashSuffix)] + hashSuffix
	}

	return sanitized, nil
}

// inferWorkspaceBehavior determines the workspace type from configuration
func inferWorkspaceBehavior(cfg *config.Config) WorkspaceBehavior {
	// No workspace config = standalone
	if cfg.Workspace == nil {
		return WorkspaceBehaviorStandalone
	}

	// No projects = standalone
	if len(cfg.Workspace.Projects) == 0 {
		return WorkspaceBehaviorStandalone
	}

	// Check for repo_root - immediate polyrepo indicator
	for _, project := range cfg.Workspace.Projects {
		if project.RepoRoot != "" {
			return WorkspaceBehaviorPolyrepo
		}
	}

	// Check if any project has a path field pointing to a separate git repository
	for _, project := range cfg.Workspace.Projects {
		if project.Path != "" {
			// If path is specified, check if it's a separate git repository
			if isExternalGitRepo(project.Path) {
				return WorkspaceBehaviorPolyrepo
			}
		}
	}

	// Projects exist but no external repos = monorepo (LLM context)
	return WorkspaceBehaviorMonorepo
}

// isExternalGitRepo checks if a path is a separate git repository
func isExternalGitRepo(path string) bool {
	// Clean and resolve the path
	cleanPath := filepath.Clean(path)

	// Check if the path contains a .git directory
	gitPath := filepath.Join(cleanPath, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}

	// .git can be a directory or a file (in case of worktrees)
	return info.IsDir() || info.Mode().IsRegular()
}

// deriveWorktreeRoot determines the worktree root path
func deriveWorktreeRoot(cfg *config.Config, behavior WorkspaceBehavior) (string, error) {
	// If explicitly configured, use that
	if cfg.Workspace != nil && cfg.Workspace.WorktreeRoot != "" {
		return validateAndCleanPath(cfg.Workspace.WorktreeRoot)
	}

	// Get current repo root
	repoRoot, err := getRepoRoot()
	if err != nil {
		// Fallback to current directory
		repoRoot, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to determine current directory: %w", err)
		}
	}

	switch behavior {
	case WorkspaceBehaviorStandalone, WorkspaceBehaviorMonorepo:
		// Default: ../{project_name}_worktrees
		projectName := filepath.Base(repoRoot)
		worktreeRoot := filepath.Join(filepath.Dir(repoRoot), projectName+"_worktrees")
		return worktreeRoot, nil

	case WorkspaceBehaviorPolyrepo:
		// Find common parent of all project paths
		if cfg.Workspace != nil && len(cfg.Workspace.Projects) > 0 {
			return derivePolyrepoWorktreeRoot(cfg, repoRoot)
		}
		// Fallback to standalone behavior
		projectName := filepath.Base(repoRoot)
		worktreeRoot := filepath.Join(filepath.Dir(repoRoot), projectName+"_worktrees")
		return worktreeRoot, nil

	default:
		projectName := filepath.Base(repoRoot)
		worktreeRoot := filepath.Join(filepath.Dir(repoRoot), projectName+"_worktrees")
		return worktreeRoot, nil
	}
}

// derivePolyrepoWorktreeRoot derives the worktree root for polyrepo workspaces
func derivePolyrepoWorktreeRoot(cfg *config.Config, repoRoot string) (string, error) {
	var absolutePaths []string

	for _, project := range cfg.Workspace.Projects {
		if project.Path == "" {
			continue
		}

		// Resolve path relative to repo root
		absPath := project.Path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(repoRoot, project.Path)
		}
		absPath = filepath.Clean(absPath)

		absolutePaths = append(absolutePaths, absPath)
	}

	if len(absolutePaths) == 0 {
		// No paths specified, use standalone default
		projectName := filepath.Base(repoRoot)
		return filepath.Join(filepath.Dir(repoRoot), projectName+"_worktrees"), nil
	}

	// Find common prefix
	commonPrefix := findCommonPathPrefix(absolutePaths)

	if commonPrefix != "" {
		parentDir := filepath.Base(commonPrefix)
		return filepath.Join(filepath.Dir(commonPrefix), parentDir+"_worktrees"), nil
	}

	// No common prefix, use first project's parent
	firstProjectPath := absolutePaths[0]
	parentDir := filepath.Base(filepath.Dir(firstProjectPath))
	return filepath.Join(filepath.Dir(filepath.Dir(firstProjectPath)), parentDir+"_worktrees"), nil
}

// findCommonPathPrefix finds the longest common path prefix among paths
func findCommonPathPrefix(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	if len(paths) == 1 {
		return filepath.Dir(paths[0])
	}

	// Start with the first path's directory
	commonPrefix := filepath.Dir(paths[0])

	for _, path := range paths[1:] {
		pathDir := filepath.Dir(path)
		commonPrefix = findCommonPrefix(commonPrefix, pathDir)
		if commonPrefix == "" {
			return ""
		}
	}

	return commonPrefix
}

// findCommonPrefix finds the common prefix between two paths
func findCommonPrefix(a, b string) string {
	// Normalize paths
	a = filepath.Clean(a)
	b = filepath.Clean(b)

	// Check if both paths are absolute
	aIsAbs := filepath.IsAbs(a)
	bIsAbs := filepath.IsAbs(b)

	// If one is absolute and the other is not, no common prefix
	if aIsAbs != bIsAbs {
		return ""
	}

	// Split into components
	aParts := strings.Split(a, string(filepath.Separator))
	bParts := strings.Split(b, string(filepath.Separator))

	var commonParts []string
	minLen := len(aParts)
	if len(bParts) < minLen {
		minLen = len(bParts)
	}

	for i := 0; i < minLen; i++ {
		if aParts[i] == bParts[i] {
			commonParts = append(commonParts, aParts[i])
		} else {
			break
		}
	}

	if len(commonParts) == 0 {
		return ""
	}

	result := filepath.Join(commonParts...)

	// Preserve absolute path prefix on Unix-like systems
	if aIsAbs && !filepath.IsAbs(result) {
		result = string(filepath.Separator) + result
	}

	return result
}

// validateAndCleanPath validates and cleans a path for safety
func validateAndCleanPath(path string) (string, error) {
	cleanPath := filepath.Clean(path)

	// Check for path traversal after cleaning
	// This is a basic check - more sophisticated validation may be needed
	if strings.Contains(cleanPath, "..") && !strings.HasPrefix(cleanPath, "..") {
		return "", fmt.Errorf("invalid path '%s': path contains invalid characters or path traversal attempts", path)
	}

	return cleanPath, nil
}

// getRepoRoot returns the git repository root directory
func getRepoRoot() (string, error) {
	// Try to find .git directory by walking up
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a git repository")
		}
		dir = parent
	}
}

// checkWorktreeExists checks if a worktree path already exists and its status
func checkWorktreeExists(path, workItemID string) (WorktreeStatus, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return WorktreeNotExists, nil
	}
	if err != nil {
		return WorktreeInvalidPath, fmt.Errorf("failed to check path: %w", err)
	}

	if !info.IsDir() {
		return WorktreeInvalidPath, nil
	}

	// Check if it's a valid git worktree
	gitPath := filepath.Join(path, ".git")
	gitInfo, err := os.Stat(gitPath)
	if err != nil {
		return WorktreeInvalidPath, nil
	}

	// In a worktree, .git is a file (not a directory) that points to the main repo
	if !gitInfo.Mode().IsRegular() {
		// .git is a directory - this is either a regular repo or invalid
		return WorktreeInvalidPath, nil
	}

	// Check if the work item ID is in the path (indicates same work item)
	if strings.Contains(filepath.Base(path), workItemID) {
		return WorktreeValidSameItem, nil
	}

	return WorktreeValidDifferentItem, nil
}

// checkWorkItemStatus checks if the work item status matches the target
func checkWorkItemStatus(currentStatus, targetStatus string, skipCheck bool) error {
	if skipCheck {
		return nil
	}

	if currentStatus == targetStatus {
		return fmt.Errorf("work item status already matches target status '%s'", targetStatus)
	}

	return nil
}

// printDryRunPreview prints a preview of what the start command would do
func printDryRunPreview(ctx *StartContext) error {
	fmt.Println("[DRY RUN] Would perform the following operations:")
	fmt.Println()

	printDryRunWorkItem(ctx)
	printDryRunWorkspace(ctx)

	trunkBranch := determineDryRunTrunkBranch(ctx)
	remoteName := resolveRemoteName(ctx.Config, nil)
	worktreePath := filepath.Join(ctx.WorktreeRoot, ctx.BranchName)

	printDryRunGitOps(ctx, trunkBranch, remoteName, worktreePath)
	printDryRunPolyrepo(ctx, worktreePath)
	printDryRunStatus(ctx)
	printDryRunIDE(ctx)
	printDryRunSetup(ctx)

	return nil
}

func printDryRunWorkItem(ctx *StartContext) {
	fmt.Printf("Work Item:\n")
	fmt.Printf("  ID: %s\n", ctx.WorkItemID)
	fmt.Printf("  Title: %s\n", ctx.Metadata.title)
	fmt.Printf("  Current Status: %s\n", ctx.Metadata.currentStatus)
	fmt.Println()
}

func printDryRunWorkspace(ctx *StartContext) {
	fmt.Printf("Workspace:\n")
	fmt.Printf("  Behavior: %s\n", ctx.Behavior)
	fmt.Printf("  Worktree Root: %s\n", ctx.WorktreeRoot)
	fmt.Println()
}

func determineDryRunTrunkBranch(ctx *StartContext) string {
	if ctx.Flags.TrunkBranch != "" {
		return ctx.Flags.TrunkBranch
	}
	if ctx.Config.Git != nil && ctx.Config.Git.TrunkBranch != "" {
		return ctx.Config.Git.TrunkBranch
	}
	return defaultTrunkBranch
}

func printDryRunGitOps(ctx *StartContext, trunkBranch, remoteName, worktreePath string) {
	fmt.Printf("Git Operations:\n")
	fmt.Printf("  Trunk Branch: %s\n", trunkBranch)
	fmt.Printf("  Remote: %s\n", remoteName)
	fmt.Printf("  Branch Name: %s\n", ctx.BranchName)
	fmt.Printf("  Worktree Path: %s\n", worktreePath)
	fmt.Println()

	fmt.Printf("Commands:\n")
	fmt.Printf("  [DRY RUN] git fetch %s %s\n", remoteName, trunkBranch)
	fmt.Printf("  [DRY RUN] git merge %s/%s\n", remoteName, trunkBranch)
	fmt.Printf("  [DRY RUN] git worktree add -b %s %s %s\n", ctx.BranchName, worktreePath, trunkBranch)
	fmt.Println()
}

func printDryRunPolyrepo(ctx *StartContext, worktreePath string) {
	if ctx.Behavior != WorkspaceBehaviorPolyrepo || ctx.Config.Workspace == nil {
		return
	}

	fmt.Printf("Polyrepo Projects:\n")
	fmt.Printf("  Main Project: %s/main/\n", worktreePath)
	for _, p := range ctx.Config.Workspace.Projects {
		if p.Path != "" {
			mount := p.Mount
			if mount == "" {
				mount = p.Name
			}
			fmt.Printf("  %s: %s/%s/\n", p.Name, worktreePath, mount)
		}
	}
	fmt.Println()
}

func printDryRunStatus(ctx *StartContext) {
	statusAction := ctx.Config.Start.StatusAction
	if ctx.Flags.StatusAction != "" {
		statusAction = ctx.Flags.StatusAction
	}

	fmt.Printf("Status Management:\n")
	if statusAction == statusActionNone || ctx.Flags.SkipStatusCheck {
		fmt.Println("  Status Change: No change")
	} else {
		fmt.Printf("  Status Change: %s -> %s\n", ctx.Metadata.currentStatus, ctx.Config.Start.MoveTo)
		fmt.Printf("  Status Action: %s\n", statusAction)
	}
	fmt.Println()
}

func printDryRunIDE(ctx *StartContext) {
	fmt.Printf("IDE:\n")
	switch {
	case ctx.Flags.NoIDE:
		fmt.Println("  Action: Skip (--no-ide flag)")
	case ctx.Flags.IDECommand != "":
		fmt.Printf("  Command: %s\n", ctx.Flags.IDECommand)
	case ctx.Config.IDE != nil && ctx.Config.IDE.Command != "":
		fmt.Printf("  Command: %s\n", ctx.Config.IDE.Command)
		if len(ctx.Config.IDE.Args) > 0 {
			fmt.Printf("  Args: %s\n", strings.Join(ctx.Config.IDE.Args, " "))
		}
	default:
		fmt.Println("  Action: Skip (no IDE configured)")
	}
	fmt.Println()
}

func printDryRunSetup(ctx *StartContext) {
	fmt.Printf("Setup:\n")
	if ctx.Config.Workspace != nil && ctx.Config.Workspace.Setup != "" {
		fmt.Printf("  Main Project: %s\n", ctx.Config.Workspace.Setup)
	} else {
		fmt.Println("  Main Project: None configured")
	}

	// Show project-specific setups for polyrepo
	if ctx.Behavior == WorkspaceBehaviorPolyrepo && ctx.Config.Workspace != nil {
		hasProjectSetup := false
		for _, p := range ctx.Config.Workspace.Projects {
			if p.Setup != "" {
				if !hasProjectSetup {
					fmt.Println("  Project Setups:")
					hasProjectSetup = true
				}
				fmt.Printf("    %s: %s\n", p.Name, p.Setup)
			}
		}
	}
}

// getValidStatuses returns a sorted list of valid status keys
func getValidStatuses(cfg *config.Config) []string {
	statuses := make([]string, 0, len(cfg.StatusFolders))
	for status := range cfg.StatusFolders {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)
	return statuses
}

// ============================================================================
// Phase 2: Git Operations
// ============================================================================

// determineTrunkBranch determines the trunk branch using priority order:
// 1. --trunk-branch flag
// 2. git.trunk_branch config
// 3. Auto-detect: check "main" first, then "master"
// Returns error if both main and master exist (ambiguous)
func determineTrunkBranch(cfg *config.Config, flagValue, dir string, dryRun bool) (string, error) {
	// Priority 1: Flag value
	if flagValue != "" {
		// Verify the branch exists
		exists, err := branchExists(flagValue, dir, dryRun)
		if err != nil {
			return "", err
		}
		if !exists && !dryRun {
			return "", fmt.Errorf("trunk branch '%s' not found: configured branch does not exist", flagValue)
		}
		return flagValue, nil
	}

	// Priority 2: Config value
	if cfg.Git != nil && cfg.Git.TrunkBranch != "" {
		exists, err := branchExists(cfg.Git.TrunkBranch, dir, dryRun)
		if err != nil {
			return "", err
		}
		if !exists && !dryRun {
			return "", fmt.Errorf("trunk branch '%s' not found: configured branch does not exist and auto-detection failed. Verify the branch name in `git.trunk_branch` configuration or ensure 'main' or 'master' branch exists", cfg.Git.TrunkBranch)
		}
		return cfg.Git.TrunkBranch, nil
	}

	// Priority 3: Auto-detect
	return autoDetectTrunkBranch(dir, dryRun)
}

// autoDetectTrunkBranch auto-detects trunk branch (main or master)
// Returns error if both exist or neither exists
func autoDetectTrunkBranch(dir string, dryRun bool) (string, error) {
	if dryRun {
		// In dry-run mode, assume "main" exists
		return defaultTrunkBranch, nil
	}

	mainExists, err := branchExists(defaultTrunkBranch, dir, false)
	if err != nil {
		return "", fmt.Errorf("failed to check for '%s' branch: %w", defaultTrunkBranch, err)
	}

	masterExists, err := branchExists(defaultMasterBranch, dir, false)
	if err != nil {
		return "", fmt.Errorf("failed to check for '%s' branch: %w", defaultMasterBranch, err)
	}

	if mainExists && masterExists {
		return "", fmt.Errorf("both '%s' and '%s' branches exist: cannot auto-detect trunk branch. Configure `git.trunk_branch` explicitly in kira.yml to specify which branch to use", defaultTrunkBranch, defaultMasterBranch)
	}

	if mainExists {
		return defaultTrunkBranch, nil
	}

	if masterExists {
		return defaultMasterBranch, nil
	}

	return "", fmt.Errorf("trunk branch not found: neither '%s' nor '%s' branch exists. Create a trunk branch or configure `git.trunk_branch` in kira.yml", defaultTrunkBranch, defaultMasterBranch)
}

// branchExists checks if a branch exists in the repository
func branchExists(branchName, dir string, dryRun bool) (bool, error) {
	if dryRun {
		return true, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// Use git show-ref to check if branch exists
	_, err := executeCommand(ctx, "git", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}, dir, false)
	if err != nil {
		// Exit code 1 means branch doesn't exist, which is not an error
		if strings.Contains(err.Error(), "exit status 1") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// validateOnTrunkBranch validates that the current branch is the trunk branch
func validateOnTrunkBranch(trunkBranch, dir string, dryRun bool) error {
	if dryRun {
		return nil
	}

	currentBranch, err := getCurrentBranch(dir)
	if err != nil {
		return fmt.Errorf("failed to determine current branch: %w", err)
	}

	if currentBranch != trunkBranch {
		return fmt.Errorf("start command can only be run from trunk branch '%s', currently on '%s': checkout the trunk branch and try again", trunkBranch, currentBranch)
	}

	return nil
}

// resolveRemoteName determines the remote name using priority order:
// For main repo: git.remote or "origin"
// For polyrepo project: project.remote > git.remote > "origin"
func resolveRemoteName(cfg *config.Config, project *config.ProjectConfig) string {
	// For polyrepo project
	if project != nil {
		if project.Remote != "" {
			return project.Remote
		}
	}

	// Workspace/main repo default
	if cfg.Git != nil && cfg.Git.Remote != "" {
		return cfg.Git.Remote
	}

	return "origin"
}

// checkRemoteExists checks if a remote exists in the repository
func checkRemoteExists(remoteName, dir string, dryRun bool) (bool, error) {
	if dryRun {
		return true, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	_, err := executeCommand(ctx, "git", []string{"remote", "get-url", remoteName}, dir, false)
	if err != nil {
		// Remote doesn't exist
		return false, nil
	}

	return true, nil
}

// getRemoteURL returns the URL of the given remote in the repository at dir.
func getRemoteURL(remoteName, dir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	output, err := executeCommand(ctx, "git", []string{"remote", "get-url", remoteName}, dir, false)
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// isGitHubRemote returns true if remoteURL is a GitHub or GitHub Enterprise URL.
// baseURL is the optional workspace git_base_url (e.g. https://github.example.com); empty means github.com.
func isGitHubRemote(remoteURL, baseURL string) bool {
	if remoteURL == "" {
		return false
	}

	// Handle git@host:path format
	if strings.HasPrefix(remoteURL, "git@") {
		rest := strings.TrimPrefix(remoteURL, "git@")
		idx := strings.Index(rest, ":")
		if idx == -1 {
			return false
		}
		host := rest[:idx]
		if host == "github.com" {
			return true
		}
		if baseURL != "" {
			u, err := url.Parse(baseURL)
			if err != nil {
				return false
			}
			return u.Host == host
		}
		return false
	}

	// Handle https:// or similar
	u, err := url.Parse(remoteURL)
	if err != nil {
		return false
	}
	host := strings.TrimSuffix(u.Host, ":443")
	if host == "github.com" {
		return true
	}
	if baseURL != "" {
		base, err := url.Parse(baseURL)
		if err != nil {
			return false
		}
		baseHost := strings.TrimSuffix(base.Host, ":443")
		return baseHost == host
	}
	return false
}

// checkUncommittedChanges checks if there are uncommitted changes in the repository
func checkUncommittedChanges(dir string, dryRun bool) (bool, error) {
	if dryRun {
		return false, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	output, err := executeCommand(ctx, "git", []string{"status", "--porcelain"}, dir, false)
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	return strings.TrimSpace(output) != "", nil
}

// pullLatestChanges pulls latest changes from remote using fetch + merge
func pullLatestChanges(remoteName, trunkBranch, dir string, dryRun bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// Fetch from remote
	_, err := executeCommand(ctx, "git", []string{"fetch", remoteName, trunkBranch}, dir, dryRun)
	if err != nil {
		if strings.Contains(err.Error(), "Could not resolve host") ||
			strings.Contains(err.Error(), "unable to access") ||
			strings.Contains(err.Error(), "Connection refused") {
			return fmt.Errorf("failed to fetch changes from %s: network error occurred. Check network connection and try again: %w", remoteName, err)
		}
		return fmt.Errorf("failed to fetch changes from %s/%s: %w", remoteName, trunkBranch, err)
	}

	if dryRun {
		return nil
	}

	// Merge from remote
	mergeCtx, mergeCancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer mergeCancel()

	_, err = executeCommand(mergeCtx, "git", []string{"merge", remoteName + "/" + trunkBranch}, dir, false)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "CONFLICT") || strings.Contains(errStr, "Automatic merge failed") {
			return fmt.Errorf("failed to merge latest changes from %s/%s: merge conflicts detected. Resolve conflicts manually and try again", remoteName, trunkBranch)
		}
		if strings.Contains(errStr, "diverged") || strings.Contains(errStr, "have diverged") {
			return fmt.Errorf("trunk branch has diverged from %s/%s: local and remote branches have different commits. Rebase or merge manually before starting work", remoteName, trunkBranch)
		}
		return fmt.Errorf("failed to merge changes from %s/%s: %w", remoteName, trunkBranch, err)
	}

	return nil
}

// checkBranchStatus checks the status of a branch relative to trunk
func checkBranchStatus(branchName, trunkBranch, dir string, dryRun bool) (BranchStatus, error) {
	if dryRun {
		return BranchNotExists, nil
	}

	// Check if branch exists
	exists, err := branchExists(branchName, dir, false)
	if err != nil {
		return BranchNotExists, err
	}

	if !exists {
		return BranchNotExists, nil
	}

	// Get commit hashes for comparison
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	branchCommit, err := executeCommand(ctx, "git", []string{"rev-parse", branchName}, dir, false)
	if err != nil {
		return BranchNotExists, fmt.Errorf("failed to get branch commit: %w", err)
	}

	trunkCtx, trunkCancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer trunkCancel()

	trunkCommit, err := executeCommand(trunkCtx, "git", []string{"rev-parse", trunkBranch}, dir, false)
	if err != nil {
		return BranchNotExists, fmt.Errorf("failed to get trunk commit: %w", err)
	}

	if strings.TrimSpace(branchCommit) == strings.TrimSpace(trunkCommit) {
		return BranchPointsToTrunk, nil
	}

	return BranchHasCommits, nil
}

// createWorktree creates a git worktree at the specified path
func createWorktree(worktreePath, trunkBranch string, dryRun bool) error {
	// Get repo root to run git command from
	repoRoot, err := getRepoRoot()
	if err != nil {
		return fmt.Errorf("failed to get repository root: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// Create worktree without branch (we'll create branch separately)
	_, err = executeCommand(ctx, "git", []string{"worktree", "add", "--detach", worktreePath, trunkBranch}, repoRoot, dryRun)
	if err != nil {
		return fmt.Errorf("failed to create worktree at %s: %w", worktreePath, err)
	}

	return nil
}

// createWorktreeWithBranch creates a git worktree with a new branch
func createWorktreeWithBranch(worktreePath, branchName, trunkBranch string, dryRun bool) error {
	// Get repo root to run git command from
	repoRoot, err := getRepoRoot()
	if err != nil {
		return fmt.Errorf("failed to get repository root: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// Create worktree with new branch
	_, err = executeCommand(ctx, "git", []string{"worktree", "add", "-b", branchName, worktreePath, trunkBranch}, repoRoot, dryRun)
	if err != nil {
		return fmt.Errorf("failed to create worktree at %s with branch %s: %w", worktreePath, branchName, err)
	}

	return nil
}

// createBranchInWorktree creates or checks out a branch in an existing worktree
func createBranchInWorktree(branchName, worktreePath string, reuseBranch, dryRun bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	var args []string
	if reuseBranch {
		// Checkout existing branch
		args = []string{"checkout", branchName}
	} else {
		// Create and checkout new branch
		args = []string{"checkout", "-b", branchName}
	}

	_, err := executeCommand(ctx, "git", args, worktreePath, dryRun)
	if err != nil {
		if reuseBranch {
			return fmt.Errorf("failed to checkout branch '%s' in worktree: %w", branchName, err)
		}
		return fmt.Errorf("failed to create branch '%s' in worktree: %w", branchName, err)
	}

	return nil
}

// removeWorktree removes a git worktree
func removeWorktree(worktreePath string, force, dryRun bool) error {
	repoRoot, err := getRepoRoot()
	if err != nil {
		return fmt.Errorf("failed to get repository root: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, worktreePath)

	_, err = executeCommand(ctx, "git", args, repoRoot, dryRun)
	if err != nil {
		return fmt.Errorf("failed to remove worktree at %s: %w", worktreePath, err)
	}

	return nil
}

// rollbackWorktrees removes worktrees in reverse order (for polyrepo failure handling)
func rollbackWorktrees(worktrees []string, dryRun bool) error {
	// Remove in reverse order
	for i := len(worktrees) - 1; i >= 0; i-- {
		worktreePath := worktrees[i]
		fmt.Printf("Rolling back worktree: %s\n", worktreePath)

		err := removeWorktree(worktreePath, true, dryRun)
		if err != nil {
			return fmt.Errorf("failed to remove worktree at %s during rollback: rollback aborted: %w", worktreePath, err)
		}
	}

	return nil
}

// handleExistingWorktree handles an existing worktree path based on --override flag
func handleExistingWorktree(worktreePath, workItemID string, override, dryRun bool) error {
	status, err := checkWorktreeExists(worktreePath, workItemID)
	if err != nil {
		return err
	}

	switch status {
	case WorktreeNotExists:
		return nil // Nothing to do

	case WorktreeValidSameItem:
		if !override {
			return fmt.Errorf("worktree already exists at %s for work item %s: use `--override` to remove existing worktree and create a new one, or use the existing worktree", worktreePath, workItemID)
		}
		// Remove existing worktree
		fmt.Printf("Removing existing worktree at %s (--override)\n", worktreePath)
		return removeWorktree(worktreePath, true, dryRun)

	case WorktreeValidDifferentItem:
		if !override {
			return fmt.Errorf("worktree path %s already exists for a different work item: use `--override` to remove existing worktree, or choose a different work item", worktreePath)
		}
		// Remove existing worktree
		fmt.Printf("Removing existing worktree at %s (--override)\n", worktreePath)
		return removeWorktree(worktreePath, true, dryRun)

	case WorktreeInvalidPath:
		if !override {
			return fmt.Errorf("path %s already exists but is not a valid git worktree: remove it manually and try again, or use `--override` to remove it automatically", worktreePath)
		}
		// Remove the directory
		fmt.Printf("Removing existing path at %s (--override)\n", worktreePath)
		if !dryRun {
			if err := os.RemoveAll(worktreePath); err != nil {
				return fmt.Errorf("failed to remove existing path at %s: %w", worktreePath, err)
			}
		}
		return nil
	}

	return nil
}

// executeStandaloneStart executes the start command for standalone/monorepo workspaces
func executeStandaloneStart(ctx *StartContext, trunkBranch string) error {
	worktreePath := filepath.Join(ctx.WorktreeRoot, ctx.BranchName)

	// Handle existing worktree
	if err := handleExistingWorktree(worktreePath, ctx.WorkItemID, ctx.Flags.Override, ctx.Flags.DryRun); err != nil {
		return err
	}

	// Check branch status and create worktree accordingly
	if err := createWorktreeForBranch(ctx, worktreePath, trunkBranch); err != nil {
		return err
	}

	fmt.Printf("Created worktree at %s with branch %s\n", worktreePath, ctx.BranchName)
	return nil
}

// createWorktreeForBranch creates worktree based on branch status
func createWorktreeForBranch(ctx *StartContext, worktreePath, trunkBranch string) error {
	branchStatus, err := checkBranchStatus(ctx.BranchName, trunkBranch, "", ctx.Flags.DryRun)
	if err != nil {
		return fmt.Errorf("failed to check branch status: %w", err)
	}

	switch branchStatus {
	case BranchHasCommits:
		return fmt.Errorf("branch %s already exists and has commits: delete the branch first if you want to start fresh: `git branch -D %s`, or use a different work item", ctx.BranchName, ctx.BranchName)

	case BranchPointsToTrunk:
		return handleExistingBranchWorktree(ctx, worktreePath, trunkBranch)

	case BranchNotExists:
		return createWorktreeWithBranch(worktreePath, ctx.BranchName, trunkBranch, ctx.Flags.DryRun)

	default:
		return createWorktreeWithBranch(worktreePath, ctx.BranchName, trunkBranch, ctx.Flags.DryRun)
	}
}

// handleExistingBranchWorktree handles worktree creation when branch exists and points to trunk
func handleExistingBranchWorktree(ctx *StartContext, worktreePath, trunkBranch string) error {
	if !ctx.Flags.ReuseBranch {
		return fmt.Errorf("branch %s already exists and points to trunk: use `--reuse-branch` to checkout existing branch in new worktree, or delete the branch first: `git branch -d %s`", ctx.BranchName, ctx.BranchName)
	}

	// Create worktree without branch, then checkout existing branch
	if err := createWorktree(worktreePath, trunkBranch, ctx.Flags.DryRun); err != nil {
		return err
	}

	if err := createBranchInWorktree(ctx.BranchName, worktreePath, true, ctx.Flags.DryRun); err != nil {
		// Rollback worktree creation
		_ = removeWorktree(worktreePath, true, ctx.Flags.DryRun)
		return err
	}

	return nil
}

// PolyrepoProject represents a project in a polyrepo workspace with resolved paths
type PolyrepoProject struct {
	Name        string
	Path        string // Absolute path to the project repository
	Mount       string // Folder name in worktree
	RepoRoot    string // Shared root (if any)
	TrunkBranch string // Project-specific trunk branch
	Remote      string // Project-specific remote
}

// resolvePolyrepoProjects resolves all projects in a polyrepo workspace
func resolvePolyrepoProjects(cfg *config.Config, repoRoot string) ([]PolyrepoProject, error) {
	if cfg.Workspace == nil || len(cfg.Workspace.Projects) == 0 {
		return nil, nil
	}

	var projects []PolyrepoProject
	for _, p := range cfg.Workspace.Projects {
		project := PolyrepoProject{
			Name:     p.Name,
			Mount:    p.Mount,
			RepoRoot: p.RepoRoot,
			Remote:   resolveRemoteName(cfg, &p),
		}

		// Resolve path
		if p.Path != "" {
			if filepath.IsAbs(p.Path) {
				project.Path = p.Path
			} else {
				project.Path = filepath.Join(repoRoot, p.Path)
			}
			project.Path = filepath.Clean(project.Path)
		}

		// Resolve trunk branch with priority: project.trunk_branch > git.trunk_branch > auto-detect
		if p.TrunkBranch != "" {
			project.TrunkBranch = p.TrunkBranch
		} else if cfg.Git != nil && cfg.Git.TrunkBranch != "" {
			project.TrunkBranch = cfg.Git.TrunkBranch
		}
		// If still empty, will be auto-detected per project

		projects = append(projects, project)
	}

	return projects, nil
}

// groupProjectsByRepoRoot groups projects by their repo_root value
func groupProjectsByRepoRoot(projects []PolyrepoProject) map[string][]PolyrepoProject {
	groups := make(map[string][]PolyrepoProject)

	for _, p := range projects {
		key := p.RepoRoot
		if key == "" {
			// Standalone projects use their own path as key
			key = p.Path
		}
		groups[key] = append(groups[key], p)
	}

	return groups
}

// validatePolyrepoProjects validates all project repositories exist
func validatePolyrepoProjects(projects []PolyrepoProject, dryRun bool) error {
	if dryRun {
		return nil
	}

	for _, p := range projects {
		if p.Path == "" {
			continue
		}

		// Check if path exists and is a git repository
		if !isExternalGitRepo(p.Path) {
			return fmt.Errorf("project repository not found at %s: path does not exist or is not a git repository. Verify path exists and is a git repository, or update project configuration in kira.yml", p.Path)
		}
	}

	return nil
}

// preValidatePolyrepoWorktrees checks all worktree paths before creating any
func preValidatePolyrepoWorktrees(ctx *StartContext, worktreePaths map[string]string) error {
	if ctx.Flags.DryRun {
		return nil
	}

	var conflictingPaths []string

	for _, worktreePath := range worktreePaths {
		status, err := checkWorktreeExists(worktreePath, ctx.WorkItemID)
		if err != nil {
			return err
		}

		if status != WorktreeNotExists {
			conflictingPaths = append(conflictingPaths, worktreePath)
		}
	}

	if len(conflictingPaths) > 0 {
		if !ctx.Flags.Override {
			return fmt.Errorf("worktree path(s) already exist: %s. Use `--override` to remove existing worktrees and create new ones", strings.Join(conflictingPaths, ", "))
		}

		// Remove all conflicting paths
		for _, path := range conflictingPaths {
			if err := handleExistingWorktree(path, ctx.WorkItemID, true, ctx.Flags.DryRun); err != nil {
				return fmt.Errorf("failed to remove existing worktree at %s: cannot proceed with --override: %w", path, err)
			}
		}
	}

	return nil
}

// preValidatePolyrepoBranches checks branch existence in all repositories
func preValidatePolyrepoBranches(ctx *StartContext, projects []PolyrepoProject, mainRepoTrunkBranch string) error {
	if ctx.Flags.DryRun {
		return nil
	}

	var branchesWithCommits []string
	var branchesPointingToTrunk []string

	// Check main repo
	repoRoot, err := getRepoRoot()
	if err != nil {
		return err
	}

	withCommits, pointsToTrunk := checkMainRepoBranch(ctx, mainRepoTrunkBranch, repoRoot)
	branchesWithCommits = append(branchesWithCommits, withCommits...)
	branchesPointingToTrunk = append(branchesPointingToTrunk, pointsToTrunk...)

	// Check each project
	projectWithCommits, projectPointsToTrunk, err := checkProjectBranches(ctx, projects)
	if err != nil {
		return err
	}
	branchesWithCommits = append(branchesWithCommits, projectWithCommits...)
	branchesPointingToTrunk = append(branchesPointingToTrunk, projectPointsToTrunk...)

	return validateBranchResults(ctx, branchesWithCommits, branchesPointingToTrunk)
}

func checkMainRepoBranch(ctx *StartContext, mainRepoTrunkBranch, repoRoot string) (withCommits, pointsToTrunk []string) {
	status, err := checkBranchStatus(ctx.BranchName, mainRepoTrunkBranch, repoRoot, false)
	if err != nil {
		return nil, nil
	}

	switch status {
	case BranchHasCommits:
		return []string{"main project"}, nil
	case BranchPointsToTrunk:
		return nil, []string{"main project"}
	default:
		return nil, nil
	}
}

func checkProjectBranches(ctx *StartContext, projects []PolyrepoProject) (withCommits, pointsToTrunk []string, err error) {
	for _, p := range projects {
		if p.Path == "" {
			continue
		}

		trunkBranch := p.TrunkBranch
		if trunkBranch == "" {
			trunkBranch, err = autoDetectTrunkBranch(p.Path, false)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to detect trunk branch for project %s: %w", p.Name, err)
			}
		}

		status, err := checkBranchStatus(ctx.BranchName, trunkBranch, p.Path, false)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to check branch status in project %s: %w", p.Name, err)
		}

		switch status {
		case BranchHasCommits:
			withCommits = append(withCommits, p.Name)
		case BranchPointsToTrunk:
			pointsToTrunk = append(pointsToTrunk, p.Name)
		}
	}

	return withCommits, pointsToTrunk, nil
}

func validateBranchResults(ctx *StartContext, branchesWithCommits, branchesPointingToTrunk []string) error {
	// If any branch has commits, abort
	if len(branchesWithCommits) > 0 {
		return fmt.Errorf("branch %s already exists and has commits in: %s. Delete the branches first if you want to start fresh, or use a different work item",
			ctx.BranchName, strings.Join(branchesWithCommits, ", "))
	}

	// If any branch points to trunk and --reuse-branch is not provided, abort
	if len(branchesPointingToTrunk) > 0 && !ctx.Flags.ReuseBranch {
		return fmt.Errorf("branch %s already exists and points to trunk in: %s. Use `--reuse-branch` to checkout existing branches in new worktrees, or delete the branches first",
			ctx.BranchName, strings.Join(branchesPointingToTrunk, ", "))
	}

	return nil
}

// executePolyrepoStart executes the start command for polyrepo workspaces
func executePolyrepoStart(ctx *StartContext, trunkBranch string) error {
	repoRoot, err := getRepoRoot()
	if err != nil {
		return err
	}

	// Resolve and validate projects
	projects, err := resolvePolyrepoProjects(ctx.Config, repoRoot)
	if err != nil {
		return err
	}

	if err := validatePolyrepoProjects(projects, ctx.Flags.DryRun); err != nil {
		return err
	}

	// Build worktree paths
	baseWorktreePath := filepath.Join(ctx.WorktreeRoot, ctx.BranchName)
	mainWorktreePath := filepath.Join(baseWorktreePath, "main")
	worktreePaths := buildPolyrepoWorktreePaths(projects, baseWorktreePath, mainWorktreePath)

	// Pre-validate
	if err := preValidatePolyrepoWorktrees(ctx, worktreePaths); err != nil {
		return err
	}
	if err := preValidatePolyrepoBranches(ctx, projects, trunkBranch); err != nil {
		return err
	}

	// Phase 1: Create worktrees
	createdWorktrees, err := createPolyrepoWorktrees(ctx, projects, repoRoot, trunkBranch, mainWorktreePath, baseWorktreePath)
	if err != nil {
		return err
	}

	// Phase 2: Create branches
	if err := createPolyrepoBranches(ctx, projects, createdWorktrees, mainWorktreePath, baseWorktreePath); err != nil {
		return err
	}

	fmt.Printf("Created polyrepo worktrees at %s with branch %s\n", baseWorktreePath, ctx.BranchName)
	return nil
}

// buildPolyrepoWorktreePaths builds a map of project names to worktree paths
func buildPolyrepoWorktreePaths(projects []PolyrepoProject, baseWorktreePath, mainWorktreePath string) map[string]string {
	worktreePaths := make(map[string]string)
	worktreePaths["main"] = mainWorktreePath

	processedRoots := make(map[string]bool)
	for _, p := range projects {
		if p.Path == "" {
			continue
		}

		worktreePath := getProjectWorktreePath(p, baseWorktreePath, processedRoots)
		if worktreePath != "" {
			worktreePaths[p.Name] = worktreePath
		}
	}

	return worktreePaths
}

// getProjectWorktreePath returns the worktree path for a project, updating processedRoots
func getProjectWorktreePath(p PolyrepoProject, baseWorktreePath string, processedRoots map[string]bool) string {
	if p.RepoRoot != "" {
		if processedRoots[p.RepoRoot] {
			return "" // Already processed
		}
		processedRoots[p.RepoRoot] = true
		rootName := kebabCase(filepath.Base(filepath.Clean(p.RepoRoot)))
		return filepath.Join(baseWorktreePath, rootName)
	}
	return filepath.Join(baseWorktreePath, p.Mount)
}

// createPolyrepoWorktrees creates all worktrees for polyrepo projects (Phase 1)
func createPolyrepoWorktrees(ctx *StartContext, projects []PolyrepoProject, repoRoot, trunkBranch, mainWorktreePath, baseWorktreePath string) ([]string, error) {
	var createdWorktrees []string

	// Create main project worktree
	if err := createMainWorktree(mainWorktreePath, trunkBranch, ctx.Flags.DryRun); err != nil {
		return nil, err
	}
	createdWorktrees = append(createdWorktrees, mainWorktreePath)

	// Create project worktrees
	processedRoots := make(map[string]bool)
	for _, p := range projects {
		if p.Path == "" {
			continue
		}

		worktreePath, repoPath := resolveProjectPaths(p, baseWorktreePath, repoRoot, processedRoots)
		if worktreePath == "" {
			continue // Already processed
		}

		if err := createProjectWorktree(ctx, p, worktreePath, repoPath, createdWorktrees); err != nil {
			return nil, err
		}
		createdWorktrees = append(createdWorktrees, worktreePath)
	}

	return createdWorktrees, nil
}

// createMainWorktree creates the main project worktree
func createMainWorktree(mainWorktreePath, trunkBranch string, dryRun bool) error {
	fmt.Printf("Creating main project worktree at %s\n", mainWorktreePath)
	if err := os.MkdirAll(filepath.Dir(mainWorktreePath), 0o750); err != nil && !dryRun {
		return fmt.Errorf("failed to create worktree parent directory: %w", err)
	}
	return createWorktree(mainWorktreePath, trunkBranch, dryRun)
}

// resolveProjectPaths resolves worktree and repo paths for a project
func resolveProjectPaths(p PolyrepoProject, baseWorktreePath, repoRoot string, processedRoots map[string]bool) (worktreePath, repoPath string) {
	if p.RepoRoot != "" {
		if processedRoots[p.RepoRoot] {
			return "", "" // Already processed
		}
		processedRoots[p.RepoRoot] = true
		rootName := kebabCase(filepath.Base(filepath.Clean(p.RepoRoot)))
		worktreePath = filepath.Join(baseWorktreePath, rootName)

		if filepath.IsAbs(p.RepoRoot) {
			repoPath = p.RepoRoot
		} else {
			repoPath = filepath.Join(repoRoot, p.RepoRoot)
		}
	} else {
		worktreePath = filepath.Join(baseWorktreePath, p.Mount)
		repoPath = p.Path
	}
	return worktreePath, repoPath
}

// createProjectWorktree creates a worktree for a single project
func createProjectWorktree(ctx *StartContext, p PolyrepoProject, worktreePath, repoPath string, createdWorktrees []string) error {
	projectTrunk, err := resolveProjectTrunkBranch(p, repoPath, ctx.Flags.DryRun)
	if err != nil {
		_ = rollbackWorktrees(createdWorktrees, ctx.Flags.DryRun)
		return fmt.Errorf("failed to detect trunk branch for project %s: %w", p.Name, err)
	}

	fmt.Printf("Creating worktree for %s at %s\n", p.Name, worktreePath)

	createCtx, createCancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	_, createErr := executeCommand(createCtx, "git", []string{"worktree", "add", "--detach", worktreePath, projectTrunk}, repoPath, ctx.Flags.DryRun)
	createCancel()

	if createErr != nil {
		_ = rollbackWorktrees(createdWorktrees, ctx.Flags.DryRun)
		return fmt.Errorf("failed to create worktree for project %s: %w", p.Name, createErr)
	}

	return nil
}

// resolveProjectTrunkBranch determines the trunk branch for a project
func resolveProjectTrunkBranch(p PolyrepoProject, repoPath string, dryRun bool) (string, error) {
	if p.TrunkBranch != "" {
		return p.TrunkBranch, nil
	}
	if dryRun {
		return defaultTrunkBranch, nil
	}
	return autoDetectTrunkBranch(repoPath, false)
}

// createPolyrepoBranches creates branches in all worktrees (Phase 2)
func createPolyrepoBranches(ctx *StartContext, projects []PolyrepoProject, createdWorktrees []string, mainWorktreePath, baseWorktreePath string) error {
	// Main project branch
	if err := createBranchInWorktree(ctx.BranchName, mainWorktreePath, ctx.Flags.ReuseBranch, ctx.Flags.DryRun); err != nil {
		_ = rollbackWorktrees(createdWorktrees, ctx.Flags.DryRun)
		return fmt.Errorf("failed to create branch in main project worktree: %w", err)
	}

	// Project branches
	processedRoots := make(map[string]bool)
	for _, p := range projects {
		if p.Path == "" {
			continue
		}

		worktreePath := getProjectWorktreePath(p, baseWorktreePath, processedRoots)
		if worktreePath == "" {
			continue // Already processed
		}

		if err := createBranchInWorktree(ctx.BranchName, worktreePath, ctx.Flags.ReuseBranch, ctx.Flags.DryRun); err != nil {
			_ = rollbackWorktrees(createdWorktrees, ctx.Flags.DryRun)
			return fmt.Errorf("failed to create branch in project %s worktree: %w", p.Name, err)
		}
	}

	return nil
}

// pullAllProjects pulls latest changes for all projects in a polyrepo
func pullAllProjects(ctx *StartContext, mainTrunkBranch, mainRemoteName string) error {
	repoRoot, err := getRepoRoot()
	if err != nil {
		return err
	}

	// Pull main repo first
	remoteExists, err := checkRemoteExists(mainRemoteName, repoRoot, ctx.Flags.DryRun)
	if err != nil {
		return err
	}

	if !remoteExists {
		fmt.Printf("Warning: No remote '%s' configured for main project. Skipping pull step.\n", mainRemoteName)
	} else {
		fmt.Printf("Pulling latest changes for main project from %s/%s\n", mainRemoteName, mainTrunkBranch)
		if err := pullLatestChanges(mainRemoteName, mainTrunkBranch, repoRoot, ctx.Flags.DryRun); err != nil {
			return err
		}
	}

	// Pull for each project
	projects, err := resolvePolyrepoProjects(ctx.Config, repoRoot)
	if err != nil {
		return err
	}

	for _, p := range projects {
		if p.Path == "" {
			continue
		}

		// Check remote exists
		remoteExists, err := checkRemoteExists(p.Remote, p.Path, ctx.Flags.DryRun)
		if err != nil {
			return err
		}

		if !remoteExists {
			fmt.Printf("Warning: No remote '%s' configured for project '%s'. Skipping pull step.\n", p.Remote, p.Name)
			continue
		}

		// Determine trunk branch
		projectTrunk := p.TrunkBranch
		if projectTrunk == "" && !ctx.Flags.DryRun {
			projectTrunk, err = autoDetectTrunkBranch(p.Path, false)
			if err != nil {
				return fmt.Errorf("failed to detect trunk branch for project %s: %w", p.Name, err)
			}
		}
		if projectTrunk == "" {
			projectTrunk = defaultTrunkBranch
		}

		fmt.Printf("Pulling latest changes for %s from %s/%s\n", p.Name, p.Remote, projectTrunk)
		if err := pullLatestChanges(p.Remote, projectTrunk, p.Path, ctx.Flags.DryRun); err != nil {
			return fmt.Errorf("failed to pull changes for project %s: %w", p.Name, err)
		}
	}

	return nil
}

// ============================================================================
// Phase 3: Status Management
// ============================================================================

// shouldSkipDraftPR returns true when --no-draft-pr is set (skip push and draft PR for all repos).
func shouldSkipDraftPR(flags StartFlags) bool {
	return flags.NoDraftPR
}

// shouldCreateDraftPR returns true if draft PR should be created for the given project.
// Priority: 1) --no-draft-pr flag 2) work item repos list 3) project draft_pr 4) workspace draft_pr 5) default true.
// For standalone/monorepo pass projectName "" and project nil.
func shouldCreateDraftPR(ctx *StartContext, projectName string, project *config.ProjectConfig) bool {
	if ctx.Flags.NoDraftPR {
		return false
	}
	// Work item repos: when set, only create for projects in the list
	if len(ctx.Metadata.repos) > 0 {
		for _, r := range ctx.Metadata.repos {
			if r == projectName {
				return true
			}
		}
		return false
	}
	// Project-level draft_pr override
	if project != nil && project.DraftPR != nil && !*project.DraftPR {
		return false
	}
	// Workspace-level draft_pr
	if ctx.Config.Workspace != nil && ctx.Config.Workspace.DraftPR != nil && !*ctx.Config.Workspace.DraftPR {
		return false
	}
	return true
}

// getEffectiveStatusAction returns the status action to use (flag overrides config)
func getEffectiveStatusAction(ctx *StartContext) string {
	if ctx.Flags.StatusAction != "" {
		return ctx.Flags.StatusAction
	}
	return ctx.Config.Start.StatusAction
}

// performStatusCheck checks if the work item status matches the target status.
// Returns error if already in target status (unless --skip-status-check).
// Sets ctx.SkipStatusUpdate if --skip-status-check is used with matching status.
func performStatusCheck(ctx *StartContext) error {
	statusAction := getEffectiveStatusAction(ctx)

	// Skip check entirely if status_action is "none"
	if statusAction == statusActionNone {
		ctx.SkipStatusUpdate = true
		return nil
	}

	targetStatus := ctx.Config.Start.MoveTo
	currentStatus := ctx.Metadata.currentStatus

	// If status matches target
	if currentStatus == targetStatus {
		if ctx.Flags.SkipStatusCheck {
			// Allow proceeding but skip the status update
			ctx.SkipStatusUpdate = true
			fmt.Printf("Skipping status update (--skip-status-check): work item already in '%s' status\n", targetStatus)
			return nil
		}
		return fmt.Errorf("work item %s is already in '%s' status: use --skip-status-check to restart work or review elsewhere", ctx.WorkItemID, targetStatus)
	}

	return nil
}

// performStatusUpdate performs status update for commit_only and commit_and_push actions.
// This runs on the trunk branch BEFORE worktree creation.
func performStatusUpdate(ctx *StartContext, repoRoot, trunkBranch, remoteName string) error {
	statusAction := getEffectiveStatusAction(ctx)

	// Skip if status_action is "none" or "commit_only_branch" or if skip flag is set
	if statusAction == statusActionNone || statusAction == statusActionCommitOnlyBranch || ctx.SkipStatusUpdate {
		return nil
	}

	targetStatus := ctx.Config.Start.MoveTo

	fmt.Printf("Moving work item %s to '%s' status\n", ctx.WorkItemID, targetStatus)

	// Get the old path before moving
	oldPath := ctx.WorkItemPath

	// Move the work item file and update status field
	// Use moveWorkItem with commitFlag=false since we handle commit separately
	if err := moveWorkItemWithoutCommit(ctx.Config, ctx.WorkItemID, targetStatus); err != nil {
		return fmt.Errorf("failed to move work item to '%s' status: %w", targetStatus, err)
	}

	// Get the new path after moving
	newPath := filepath.Join(config.GetWorkFolderPath(ctx.Config), ctx.Config.StatusFolders[targetStatus], filepath.Base(oldPath))

	// Update ctx.WorkItemPath to the new location
	ctx.WorkItemPath = newPath

	// Build commit message
	commitMsg, err := buildStatusCommitMessage(ctx, targetStatus)
	if err != nil {
		return fmt.Errorf("failed to build commit message: %w", err)
	}

	// Commit the status change
	if err := commitStatusChange(repoRoot, oldPath, newPath, commitMsg); err != nil {
		return fmt.Errorf("failed to commit status change: %w", err)
	}

	fmt.Printf("Committed status change: %s\n", commitMsg)

	// Push if status_action is commit_and_push
	if statusAction == statusActionCommitAndPush {
		// Check if remote exists before attempting to push
		remoteExists, err := checkRemoteExists(remoteName, repoRoot, false)
		if err != nil {
			return fmt.Errorf("failed to check remote existence: %w", err)
		}

		if !remoteExists {
			fmt.Printf("Warning: No remote '%s' configured. Skipping push step. Status change was committed locally.\n", remoteName)
		} else {
			if err := pushStatusChange(repoRoot, remoteName, trunkBranch); err != nil {
				return fmt.Errorf("failed to push status change to %s/%s: %w", remoteName, trunkBranch, err)
			}
			fmt.Printf("Pushed status change to %s/%s\n", remoteName, trunkBranch)
		}
	}

	return nil
}

// performStatusUpdateOnBranch performs status update for commit_only_branch action.
// This runs on the new branch AFTER worktree creation.
func performStatusUpdateOnBranch(ctx *StartContext, worktreePath string) error {
	statusAction := getEffectiveStatusAction(ctx)

	// Skip if status_action is not "commit_only_branch" or if skip flag is set
	if statusAction != statusActionCommitOnlyBranch || ctx.SkipStatusUpdate {
		return nil
	}

	targetStatus := ctx.Config.Start.MoveTo

	fmt.Printf("Moving work item %s to '%s' status (on branch)\n", ctx.WorkItemID, targetStatus)

	// Get the old path before moving
	oldPath := ctx.WorkItemPath

	// Move the work item file and update status field
	if err := moveWorkItemWithoutCommit(ctx.Config, ctx.WorkItemID, targetStatus); err != nil {
		return fmt.Errorf("failed to move work item to '%s' status: %w", targetStatus, err)
	}

	// Get the new path after moving
	newPath := filepath.Join(config.GetWorkFolderPath(ctx.Config), ctx.Config.StatusFolders[targetStatus], filepath.Base(oldPath))

	// Update ctx.WorkItemPath to the new location
	ctx.WorkItemPath = newPath

	// Build commit message
	commitMsg, err := buildStatusCommitMessage(ctx, targetStatus)
	if err != nil {
		return fmt.Errorf("failed to build commit message: %w", err)
	}

	// Commit the status change in the worktree (on the new branch)
	// Note: The worktree sees the same work directory as the main repo
	if err := commitStatusChange(worktreePath, oldPath, newPath, commitMsg); err != nil {
		return fmt.Errorf("failed to commit status change on branch: %w", err)
	}

	fmt.Printf("Committed status change on branch: %s\n", commitMsg)

	return nil
}

// moveWorkItemWithoutCommit moves a work item to target status without committing.
// This mirrors the logic in moveWorkItem but without the commit step.
func moveWorkItemWithoutCommit(cfg *config.Config, workItemID, targetStatus string) error {
	// Find the work item file
	workItemPath, err := findWorkItemFile(workItemID, cfg)
	if err != nil {
		return err
	}

	// Validate target status
	if _, exists := cfg.StatusFolders[targetStatus]; !exists {
		return fmt.Errorf("invalid target status: %s", targetStatus)
	}

	// Get target folder path
	targetFolder := filepath.Join(config.GetWorkFolderPath(cfg), cfg.StatusFolders[targetStatus])
	filename := filepath.Base(workItemPath)
	targetPath := filepath.Join(targetFolder, filename)

	// Move the file
	if err := os.Rename(workItemPath, targetPath); err != nil {
		return fmt.Errorf("failed to move work item: %w", err)
	}

	// Update the status in the file
	if err := updateWorkItemStatus(targetPath, targetStatus, cfg); err != nil {
		return fmt.Errorf("failed to update work item status: %w", err)
	}

	return nil
}

// buildStatusCommitMessage builds a commit message from the template.
// Template variables: {type}, {id}, {title}, {move_to}
func buildStatusCommitMessage(ctx *StartContext, targetStatus string) (string, error) {
	template := ctx.Config.Start.StatusCommitMessage
	if template == "" {
		template = "Move {type} {id} to {move_to}"
	}

	msg := template
	msg = strings.ReplaceAll(msg, "{type}", ctx.Metadata.workItemType)
	msg = strings.ReplaceAll(msg, "{id}", ctx.WorkItemID)
	msg = strings.ReplaceAll(msg, "{title}", ctx.Metadata.title)
	msg = strings.ReplaceAll(msg, "{move_to}", targetStatus)

	// Sanitize the message
	sanitized, err := sanitizeCommitMessage(msg)
	if err != nil {
		return "", fmt.Errorf("invalid commit message: %w", err)
	}

	return sanitized, nil
}

// commitStatusChange stages the moved files and commits with the given message.
func commitStatusChange(dir, oldPath, newPath, message string) error {
	cmdCtx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// Stage the old file deletion using git rm --cached
	_, err := executeCommand(cmdCtx, "git", []string{"rm", "--cached", oldPath}, dir, false)
	if err != nil {
		// If git rm fails (file wasn't tracked), try git add -u to stage deletions
		oldDir := filepath.Dir(oldPath)
		addCtx, addCancel := context.WithTimeout(context.Background(), gitCommandTimeout)
		_, _ = executeCommand(addCtx, "git", []string{"add", "-u", oldDir}, dir, false)
		addCancel()
	}

	// Stage the new file addition
	addNewCtx, addNewCancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer addNewCancel()

	_, err = executeCommand(addNewCtx, "git", []string{"add", newPath}, dir, false)
	if err != nil {
		return fmt.Errorf("failed to stage new file: %w", err)
	}

	// Commit
	commitCtx, commitCancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer commitCancel()

	_, err = executeCommandCombinedOutput(commitCtx, "git", []string{"commit", "-m", message}, dir, false)
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// pushStatusChange pushes the status change to the remote.
func pushStatusChange(dir, remoteName, branchName string) error {
	pushCtx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	_, err := executeCommand(pushCtx, "git", []string{"push", remoteName, branchName}, dir, false)
	if err != nil {
		return err
	}

	return nil
}

// ============================================================================
// Phase 4: IDE & Setup Integration
// ============================================================================

// launchIDE opens the IDE for the worktree.
// Priority order: --no-ide flag (skip) > --ide flag > ide.command config > no IDE
// IDE launch failures are logged as warnings; worktree creation still succeeds.
func launchIDE(ctx *StartContext, worktreePath string) {
	// Priority 1: --no-ide flag (highest priority - silent skip)
	if ctx.Flags.NoIDE {
		return
	}

	// Priority 2: --ide flag override
	if ctx.Flags.IDECommand != "" {
		launchIDECommand(ctx.Flags.IDECommand, nil, worktreePath, ctx.Flags.DryRun)
		return
	}

	// Priority 3: ide.command from config
	if ctx.Config.IDE != nil && ctx.Config.IDE.Command != "" {
		launchIDECommand(ctx.Config.IDE.Command, ctx.Config.IDE.Args, worktreePath, ctx.Flags.DryRun)
		return
	}

	// No IDE configured
	fmt.Printf("Info: No IDE configured. Worktree created at %s. Configure `ide.command` in kira.yml or use `--ide <command>` flag to automatically open IDE.\n", worktreePath)
}

// launchIDECommand executes the IDE command with the worktree path.
// The command is run in the background so we don't wait for the IDE to close.
func launchIDECommand(command string, args []string, worktreePath string, dryRun bool) {
	// Build full command arguments: args + worktreePath
	fullArgs := make([]string, 0, len(args)+1)
	fullArgs = append(fullArgs, args...)
	fullArgs = append(fullArgs, worktreePath)

	if dryRun {
		preview := formatCommandPreview(command, fullArgs)
		fmt.Println(preview)
		return
	}

	fmt.Printf("Opening IDE: %s %s\n", command, worktreePath)

	// Create command - use Start() instead of Run() for background execution
	// #nosec G204 - IDE command is intentionally user-configured via kira.yml or --ide flag
	cmd := exec.Command(command, fullArgs...)

	// Detach from parent process so IDE keeps running after kira exits
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	err := cmd.Start()
	if err != nil {
		// Check if command not found
		if isCommandNotFound(err) {
			fmt.Printf("Warning: IDE command '%s' not found. Verify `--ide` flag value or `ide.command` in kira.yml. You can manually open the IDE at %s.\n", command, worktreePath)
		} else {
			fmt.Printf("Warning: Failed to launch IDE. IDE command execution failed. Worktree created successfully. You can manually open the IDE at %s.\n", worktreePath)
		}
		return
	}

	// Don't wait for the process - let it run in background
	// Release the process so it doesn't become a zombie
	go func() {
		_ = cmd.Wait()
	}()
}

// isCommandNotFound checks if an error indicates the command was not found.
func isCommandNotFound(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "executable file not found") ||
		strings.Contains(errStr, "no such file or directory") ||
		strings.Contains(errStr, "not found")
}

// executeSetupCommands runs workspace.setup and project.setup commands.
// For standalone/monorepo: runs workspace.setup in worktreePath
// For polyrepo: runs workspace.setup in {worktreePath}/main/, then project.setup for each project
func executeSetupCommands(ctx *StartContext, worktreePath string) error {
	if ctx.Config.Workspace == nil {
		return nil // No workspace config, nothing to do
	}

	// Determine main project worktree path
	mainWorktreePath := worktreePath
	if ctx.Behavior == WorkspaceBehaviorPolyrepo {
		mainWorktreePath = filepath.Join(worktreePath, "main")
	}

	// Run workspace.setup (main project setup)
	if ctx.Config.Workspace.Setup != "" {
		fmt.Printf("Running setup for main project: %s\n", ctx.Config.Workspace.Setup)
		if err := executeSetup(ctx.Config.Workspace.Setup, mainWorktreePath, ctx.Flags.DryRun); err != nil {
			return fmt.Errorf("setup command failed: %w", err)
		}
	}

	// For polyrepo, run project-specific setups
	if ctx.Behavior == WorkspaceBehaviorPolyrepo {
		if err := executeProjectSetups(ctx, worktreePath); err != nil {
			return err
		}
	}

	return nil
}

// executeProjectSetups runs setup commands for each polyrepo project.
func executeProjectSetups(ctx *StartContext, baseWorktreePath string) error {
	if ctx.Config.Workspace == nil {
		return nil
	}

	processedRoots := make(map[string]bool)

	for _, p := range ctx.Config.Workspace.Projects {
		if p.Setup == "" {
			continue // No setup configured for this project
		}

		// Determine worktree path for this project
		projectWorktreePath := getProjectSetupPath(p, baseWorktreePath, processedRoots)
		if projectWorktreePath == "" {
			continue // Already processed this repo_root group
		}

		fmt.Printf("Running setup for %s: %s\n", p.Name, p.Setup)
		if err := executeSetup(p.Setup, projectWorktreePath, ctx.Flags.DryRun); err != nil {
			return fmt.Errorf("setup command failed for project '%s': %w", p.Name, err)
		}
	}

	return nil
}

// getProjectSetupPath returns the worktree path for a project's setup command.
func getProjectSetupPath(p config.ProjectConfig, baseWorktreePath string, processedRoots map[string]bool) string {
	if p.Path == "" {
		return "" // No path, no worktree
	}

	if p.RepoRoot != "" {
		if processedRoots[p.RepoRoot] {
			return "" // Already processed this repo_root group
		}
		processedRoots[p.RepoRoot] = true
		rootName := kebabCase(filepath.Base(filepath.Clean(p.RepoRoot)))
		return filepath.Join(baseWorktreePath, rootName)
	}

	mount := p.Mount
	if mount == "" {
		mount = p.Name
	}
	return filepath.Join(baseWorktreePath, mount)
}

// executeSetup runs a single setup command or script.
// If the setup string looks like a script path (contains / or starts with ./),
// it's executed via shell. Otherwise, it's executed directly.
func executeSetup(setupCmd, workDir string, dryRun bool) error {
	if dryRun {
		fmt.Printf("[DRY RUN] Would execute setup: %s (in %s)\n", setupCmd, workDir)
		return nil
	}

	// Check if workDir exists
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		return fmt.Errorf("setup directory does not exist: %s", workDir)
	}

	// Determine if this is a script path (starts with ./ or is just a relative path to a script file)
	// Note: We only check for script existence if it starts with "./" as that's the common pattern
	// Commands like "echo test > /tmp/file" should not be treated as scripts
	isScriptPath := strings.HasPrefix(setupCmd, "./") || strings.HasPrefix(setupCmd, "/")

	// Only validate script existence if it looks like a direct script invocation (not a shell command)
	// A shell command typically has spaces (arguments) or special characters
	isSimpleScriptPath := isScriptPath && !strings.ContainsAny(setupCmd, " \t|&;<>")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // 5 minute timeout for setup
	defer cancel()

	if isSimpleScriptPath {
		// Check if script exists
		scriptPath := setupCmd
		if !filepath.IsAbs(scriptPath) {
			scriptPath = filepath.Join(workDir, setupCmd)
		}

		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			return fmt.Errorf("setup script not found: %s. Script file does not exist. Verify `workspace.setup` or `project.setup` configuration in kira.yml", setupCmd)
		}
	}

	// Execute via shell (handles pipes, redirects, etc.)
	cmd := exec.CommandContext(ctx, "sh", "-c", setupCmd)

	cmd.Dir = workDir

	// Capture output for error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("setup command timed out. Command did not complete within timeout period. Verify setup command completes successfully or increase timeout if needed")
		}
		if outputStr != "" {
			return fmt.Errorf("setup command exited with error: %s. Check command output for details: %s", err, outputStr)
		}
		return fmt.Errorf("setup command exited with error: %w", err)
	}

	// Print output if any
	if len(output) > 0 {
		fmt.Printf("%s", string(output))
	}

	return nil
}
