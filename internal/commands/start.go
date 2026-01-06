// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"kira/internal/config"
)

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
	IDECommand      string
	TrunkBranch     string
	StatusAction    string
}

// StartContext holds all validated inputs for the start command
type StartContext struct {
	WorkItemID     string
	WorkItemPath   string
	Metadata       workItemMetadata
	SanitizedTitle string
	BranchName     string
	WorktreeRoot   string
	WorktreePaths  []string // For polyrepo
	Behavior       WorkspaceBehavior
	Config         *config.Config
	Flags          StartFlags
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
	startCmd.Flags().String("ide", "", "Override IDE command (e.g., --ide cursor)")
	startCmd.Flags().String("trunk-branch", "", "Override trunk branch (e.g., --trunk-branch develop)")
	startCmd.Flags().String("status-action", "", "Override status action (none|commit_only|commit_and_push|commit_only_branch)")
}

func runStart(cmd *cobra.Command, args []string) error {
	if err := checkWorkDir(); err != nil {
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	workItemID := args[0]

	// Parse flags
	flags := StartFlags{}
	flags.DryRun, _ = cmd.Flags().GetBool("dry-run")
	flags.Override, _ = cmd.Flags().GetBool("override")
	flags.SkipStatusCheck, _ = cmd.Flags().GetBool("skip-status-check")
	flags.ReuseBranch, _ = cmd.Flags().GetBool("reuse-branch")
	flags.NoIDE, _ = cmd.Flags().GetBool("no-ide")
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

	// In Phase 1, we only do validation - no git operations yet
	if flags.DryRun {
		return printDryRunPreview(ctx)
	}

	// Phase 1: Only validation implemented
	// Git operations will be added in Phase 2
	fmt.Printf("Work item %s validated successfully.\n", workItemID)
	fmt.Printf("  Title: %s\n", ctx.Metadata.title)
	fmt.Printf("  Branch: %s\n", ctx.BranchName)
	fmt.Printf("  Worktree root: %s\n", ctx.WorktreeRoot)
	fmt.Printf("  Workspace behavior: %s\n", ctx.Behavior)
	fmt.Println("\nNote: Git operations will be implemented in Phase 2")

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
	workItemPath, err := findWorkItemFile(workItemID)
	if err != nil {
		return nil, fmt.Errorf("work item '%s' not found: no work item file exists with that ID", workItemID)
	}
	ctx.WorkItemPath = workItemPath

	// Step 3: Extract work item metadata
	workItemType, id, title, currentStatus, err := extractWorkItemMetadata(workItemPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read work item '%s': could not extract metadata: %w", workItemID, err)
	}
	ctx.Metadata = workItemMetadata{
		workItemType:  workItemType,
		id:            id,
		title:         title,
		currentStatus: currentStatus,
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

	// Step 8: Check work item status (if status_action is not "none")
	statusAction := cfg.Start.StatusAction
	if flags.StatusAction != "" {
		statusAction = flags.StatusAction
	}

	if statusAction != "none" && !flags.SkipStatusCheck {
		targetStatus := cfg.Start.MoveTo
		if currentStatus == targetStatus {
			return nil, fmt.Errorf("work item %s is already in '%s' status: use --skip-status-check to restart work or review elsewhere", workItemID, targetStatus)
		}
	}

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

	fmt.Printf("Work Item:\n")
	fmt.Printf("  ID: %s\n", ctx.WorkItemID)
	fmt.Printf("  Title: %s\n", ctx.Metadata.title)
	fmt.Printf("  Current Status: %s\n", ctx.Metadata.currentStatus)
	fmt.Println()

	fmt.Printf("Workspace:\n")
	fmt.Printf("  Behavior: %s\n", ctx.Behavior)
	fmt.Printf("  Worktree Root: %s\n", ctx.WorktreeRoot)
	fmt.Println()

	fmt.Printf("Git Operations:\n")
	fmt.Printf("  Branch Name: %s\n", ctx.BranchName)

	// Determine worktree path
	worktreePath := filepath.Join(ctx.WorktreeRoot, ctx.BranchName)
	fmt.Printf("  Worktree Path: %s\n", worktreePath)
	fmt.Println()

	// Status change info
	statusAction := ctx.Config.Start.StatusAction
	if ctx.Flags.StatusAction != "" {
		statusAction = ctx.Flags.StatusAction
	}

	fmt.Printf("Status Management:\n")
	if statusAction == "none" || ctx.Flags.SkipStatusCheck {
		fmt.Println("  Status Change: No change")
	} else {
		fmt.Printf("  Status Change: %s -> %s\n", ctx.Metadata.currentStatus, ctx.Config.Start.MoveTo)
		fmt.Printf("  Status Action: %s\n", statusAction)
	}
	fmt.Println()

	// IDE info
	fmt.Printf("IDE:\n")
	if ctx.Flags.NoIDE {
		fmt.Println("  Action: Skip (--no-ide flag)")
	} else if ctx.Flags.IDECommand != "" {
		fmt.Printf("  Command: %s\n", ctx.Flags.IDECommand)
	} else if ctx.Config.IDE != nil && ctx.Config.IDE.Command != "" {
		fmt.Printf("  Command: %s\n", ctx.Config.IDE.Command)
		if len(ctx.Config.IDE.Args) > 0 {
			fmt.Printf("  Args: %s\n", strings.Join(ctx.Config.IDE.Args, " "))
		}
	} else {
		fmt.Println("  Action: Skip (no IDE configured)")
	}
	fmt.Println()

	// Setup commands
	fmt.Printf("Setup:\n")
	if ctx.Config.Workspace != nil && ctx.Config.Workspace.Setup != "" {
		fmt.Printf("  Main Project: %s\n", ctx.Config.Workspace.Setup)
	} else {
		fmt.Println("  Main Project: None configured")
	}

	return nil
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
