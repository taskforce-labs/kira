// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	yaml "gopkg.in/yaml.v3"

	"kira/internal/config"

	"github.com/spf13/cobra"
)

var latestCmd = &cobra.Command{
	Use:   "latest",
	Short: "Update feature branches with latest trunk changes through iterative conflict resolution",
	Long: `A command that keeps feature branches updated with trunk changes through iterative conflict resolution.

The command first checks for existing merge conflicts, displays them for external LLM resolution,
and only performs fetch/rebase when conflicts are resolved. Since kira supports polyrepo workflows
(managing work across multiple repositories), the command handles rebasing across multiple repos
simultaneously, ensuring consistency and coordination between related repositories.

The command can be called repeatedly to work through conflicts progressively.

If uncommitted changes are detected, they will be automatically stashed before rebase
and popped after successful rebase (unless --no-pop-stash is specified).`,
	Args: cobra.NoArgs,
	RunE: runLatest,
}

func init() {
	latestCmd.Flags().Bool("no-pop-stash", false, "Stash uncommitted changes before rebase but do not automatically pop them after")
}

// RepositoryInfo contains information about a repository that needs to be updated
type RepositoryInfo struct {
	Name        string // Project name or directory name for standalone/monorepo
	Path        string // Absolute path to repository
	TrunkBranch string // Resolved trunk branch (project override > git.trunk_branch > auto-detect)
	Remote      string // Resolved remote name (project override > git.remote > "origin")
	RepoRoot    string // For polyrepo: repo_root value if present
}

// WorkItemMetadata contains extracted metadata from a work item file
type WorkItemMetadata struct {
	ID       string
	Title    string
	Status   string
	Kind     string
	Filepath string
}

// RepositoryState represents the current state of a repository
type RepositoryState string

// Repository state constants
const (
	// StateReadyForUpdate indicates the repository is clean and ready for update operations
	StateReadyForUpdate RepositoryState = "ready_for_update"
	// StateConflictsExist indicates merge conflicts are present in the repository
	StateConflictsExist RepositoryState = "conflicts_exist"
	// StateDirtyWorkingDir indicates uncommitted changes exist in the working directory
	StateDirtyWorkingDir RepositoryState = "dirty_working_directory"
	// StateInRebase indicates the repository is in the middle of a rebase operation
	StateInRebase RepositoryState = "in_rebase"
	// StateInMerge indicates the repository is in the middle of a merge operation
	StateInMerge RepositoryState = "in_merge"
	// StateError indicates an error occurred while checking repository state
	StateError RepositoryState = "error"
)

// RepositoryStateInfo contains the detected state of a repository
type RepositoryStateInfo struct {
	Repo    RepositoryInfo
	State   RepositoryState
	Error   error
	Details string // Additional context (e.g., which files have conflicts)
}

// AggregatedState represents the overall state across all repositories
type AggregatedState struct {
	OverallState     RepositoryState
	StateInfos       []RepositoryStateInfo
	ConflictingRepos []string
	DirtyRepos       []string
	InOperationRepos []string
	ErrorRepos       []string
	ReadyRepos       []string
}

// ConflictRegion represents a single conflict region with markers and content
type ConflictRegion struct {
	StartMarker   string   // <<<<<<< HEAD or <<<<<<< branch-name
	OurContent    string   // Content between <<<<<<< and =======
	Separator     string   // =======
	TheirContent  string   // Content between ======= and >>>>>>>
	EndMarker     string   // >>>>>>> branch-name
	ContextBefore []string // 3 lines before conflict
	ContextAfter  []string // 3 lines after conflict
}

// FileConflict represents all conflicts in a single file with path and conflict regions
type FileConflict struct {
	RepoName string
	FilePath string
	Regions  []ConflictRegion
	Error    error // Error if file couldn't be read or parsed
}

// RepositoryConflicts represents all conflicts in a repository grouped by file
type RepositoryConflicts struct {
	Repo  RepositoryInfo
	Files []FileConflict
}

func runLatest(cmd *cobra.Command, _ []string) error {
	if err := checkWorkDir(); err != nil {
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	repos, err := discoverRepositories(cfg)
	if err != nil {
		return err
	}

	if len(repos) == 0 {
		return fmt.Errorf("no repositories found for current work item")
	}

	displayDiscoveredRepositories(repos)

	// Phase 3: Check state for each repository
	stateInfos := checkAllRepositoryStates(repos)
	aggregated := aggregateRepositoryStates(stateInfos)

	displayStateSummary(stateInfos, aggregated)

	// Phase 4: Display conflicts if any exist
	if aggregated.OverallState == StateConflictsExist {
		displayAllConflicts(stateInfos)
		return nil
	}

	// Get flag value for stash handling
	noPopStash, _ := cmd.Flags().GetBool("no-pop-stash")

	// Phase 5: Perform fetch and rebase if repositories are ready
	// Also handle repositories with uncommitted changes (stash them)
	if aggregated.OverallState == StateReadyForUpdate || len(aggregated.DirtyRepos) > 0 {
		// Phase 6: Pre-flight validation - ensure no blocking states
		if err := validateAllReposCleanOrDirtyForUpdate(aggregated); err != nil {
			return err
		}

		displayUpdateMessage(aggregated.DirtyRepos, noPopStash)

		reposToProcess := getReposToProcess(stateInfos)
		if len(reposToProcess) == 0 {
			return fmt.Errorf("no repositories ready for update")
		}

		results := performFetchAndRebaseForAllRepos(reposToProcess, noPopStash)
		return handleUpdateResults(results)
	}

	// For other states (dirty, in_rebase, in_merge, error), just return
	// The state summary already displayed the issue
	return nil
}

// displayUpdateMessage displays the appropriate message before starting updates
func displayUpdateMessage(dirtyRepos []string, noPopStash bool) {
	if len(dirtyRepos) > 0 {
		fmt.Println("\nSome repositories have uncommitted changes. They will be stashed before rebase.")
		if !noPopStash {
			fmt.Println("Changes will be automatically popped after successful rebase.")
		} else {
			fmt.Println("Changes will remain stashed (--no-pop-stash was specified).")
		}
		fmt.Println()
	} else {
		fmt.Println("\nAll repositories are ready for update. Proceeding with fetch and rebase...")
		fmt.Println()
	}
}

// getReposToProcess collects repositories that are ready for update or have uncommitted changes
func getReposToProcess(stateInfos []RepositoryStateInfo) []RepositoryInfo {
	var reposToProcess []RepositoryInfo
	for _, stateInfo := range stateInfos {
		if stateInfo.State == StateReadyForUpdate || stateInfo.State == StateDirtyWorkingDir {
			reposToProcess = append(reposToProcess, stateInfo.Repo)
		}
	}
	return reposToProcess
}

// handleUpdateResults processes the results and returns appropriate error
func handleUpdateResults(results []RepositoryOperationResult) error {
	displayOperationResults(results)

	// Check if any operations failed
	for _, result := range results {
		if result.Error != nil {
			return fmt.Errorf("some repositories failed to update")
		}
	}

	fmt.Println("\n✓ All repositories updated successfully!")
	return nil
}

// displayDiscoveredRepositories displays the list of discovered repositories
func displayDiscoveredRepositories(repos []RepositoryInfo) {
	fmt.Printf("Discovered %d repository(ies) for current work item:\n", len(repos))
	for _, repo := range repos {
		fmt.Printf("  - %s: %s (trunk: %s, remote: %s)\n", repo.Name, repo.Path, repo.TrunkBranch, repo.Remote)
	}
}

// checkAllRepositoryStates checks the state of all repositories
func checkAllRepositoryStates(repos []RepositoryInfo) []RepositoryStateInfo {
	fmt.Println("\nChecking repository state...")
	var stateInfos []RepositoryStateInfo
	for _, repo := range repos {
		stateInfo, err := checkRepositoryState(repo)
		if err != nil {
			// Continue checking other repos even if one fails
			stateInfo.State = StateError
			stateInfo.Error = err
			stateInfo.Details = fmt.Sprintf("error checking state: %v", err)
		}
		stateInfos = append(stateInfos, stateInfo)
	}
	return stateInfos
}

// displayStateSummary displays the state summary for all repositories
func displayStateSummary(stateInfos []RepositoryStateInfo, aggregated AggregatedState) {
	fmt.Println("\nRepository State Summary:")
	for _, stateInfo := range stateInfos {
		displayRepositoryState(stateInfo)
	}

	fmt.Printf("\nOverall State: %s\n", aggregated.OverallState)
	displayAggregatedStateDetails(aggregated)
}

// displayRepositoryState displays the state of a single repository
func displayRepositoryState(stateInfo RepositoryStateInfo) {
	stateSymbol := getStateSymbol(stateInfo.State)
	fmt.Printf("  %s %s: %s", stateSymbol, stateInfo.Repo.Name, stateInfo.State)
	if stateInfo.Details != "" {
		fmt.Printf(" (%s)", stateInfo.Details)
	}
	if stateInfo.Error != nil {
		fmt.Printf(" - Error: %v", stateInfo.Error)
	}
	fmt.Println()
}

// displayAggregatedStateDetails displays details about the aggregated state
func displayAggregatedStateDetails(aggregated AggregatedState) {
	if len(aggregated.ConflictingRepos) > 0 {
		fmt.Printf("  Repositories with conflicts: %s\n", strings.Join(aggregated.ConflictingRepos, ", "))
	}
	if len(aggregated.InOperationRepos) > 0 {
		fmt.Printf("  Repositories in operation: %s\n", strings.Join(aggregated.InOperationRepos, ", "))
	}
	if len(aggregated.DirtyRepos) > 0 {
		fmt.Printf("  Repositories with uncommitted changes: %s\n", strings.Join(aggregated.DirtyRepos, ", "))
	}
	if len(aggregated.ErrorRepos) > 0 {
		fmt.Printf("  Repositories with errors: %s\n", strings.Join(aggregated.ErrorRepos, ", "))
	}
	if len(aggregated.ReadyRepos) > 0 {
		fmt.Printf("  Repositories ready for update: %s\n", strings.Join(aggregated.ReadyRepos, ", "))
	}
}

// getStateSymbol returns a symbol for displaying repository state
func getStateSymbol(state RepositoryState) string {
	switch state {
	case StateReadyForUpdate:
		return "✓"
	case StateConflictsExist:
		return "✗"
	case StateDirtyWorkingDir:
		return "!"
	case StateInRebase, StateInMerge:
		return "⟳"
	case StateError:
		return "⚠"
	default:
		return "?"
	}
}

// discoverRepositories is the main entry point for repository discovery
func discoverRepositories(cfg *config.Config) ([]RepositoryInfo, error) {
	// Step 1: Find current work item in doing folder
	workItemPath, err := findCurrentWorkItem(cfg)
	if err != nil {
		return nil, err
	}

	// Step 2: Extract work item metadata
	metadata, err := extractWorkItemMetadataForLatest(workItemPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract work item metadata: %w", err)
	}

	// Step 3: Detect workspace behavior
	behavior := detectWorkspaceBehavior(cfg)

	// Step 4: Resolve repositories based on behavior
	repos, err := resolveRepositoriesForLatest(cfg, behavior, metadata.ID)
	if err != nil {
		return nil, err
	}

	// Step 5: Validate repositories
	if err := validateRepositories(repos); err != nil {
		return nil, err
	}

	return repos, nil
}

// findCurrentWorkItem locates the work item file in the doing folder
func findCurrentWorkItem(cfg *config.Config) (string, error) {
	doingFolder := cfg.StatusFolders["doing"]
	if doingFolder == "" {
		doingFolder = "2_doing" // default
	}

	doingPath := filepath.Join(".work", doingFolder)
	entries, err := os.ReadDir(doingPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("doing folder not found at %s: no work item in progress", doingPath)
		}
		return "", fmt.Errorf("failed to read doing folder: %w", err)
	}

	var workItemFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			workItemFiles = append(workItemFiles, entry.Name())
		}
	}

	if len(workItemFiles) == 0 {
		return "", fmt.Errorf("no work item found in doing folder (%s): start a work item first", doingPath)
	}

	if len(workItemFiles) > 1 {
		return "", fmt.Errorf("multiple work items found in doing folder (%s): %v. Only one work item allowed at a time", doingPath, workItemFiles)
	}

	return filepath.Join(doingPath, workItemFiles[0]), nil
}

const yamlFrontMatterDelimiter = "---"

// extractWorkItemMetadataForLatest parses YAML front matter from work item file
func extractWorkItemMetadataForLatest(filePath string) (*WorkItemMetadata, error) {
	content, err := safeReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read work item file: %w", err)
	}

	// Extract YAML front matter between the first pair of --- lines
	lines := strings.Split(string(content), "\n")
	var yamlLines []string
	inYAML := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if i == 0 && trimmed == yamlFrontMatterDelimiter {
			inYAML = true
			continue
		}
		if inYAML {
			if trimmed == yamlFrontMatterDelimiter {
				break
			}
			yamlLines = append(yamlLines, line)
		}
	}

	// Parse YAML to extract fields
	type workItemFields struct {
		ID     string `yaml:"id"`
		Title  string `yaml:"title"`
		Status string `yaml:"status"`
		Kind   string `yaml:"kind"`
	}

	fields := &workItemFields{}
	if len(yamlLines) > 0 {
		if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), fields); err != nil {
			return nil, fmt.Errorf("failed to parse front matter: %w", err)
		}
	}

	return &WorkItemMetadata{
		ID:       fields.ID,
		Title:    fields.Title,
		Status:   fields.Status,
		Kind:     fields.Kind,
		Filepath: filePath,
	}, nil
}

// detectWorkspaceBehavior determines the workspace type from configuration
func detectWorkspaceBehavior(cfg *config.Config) WorkspaceBehavior {
	// Reuse the existing function from start.go
	return inferWorkspaceBehavior(cfg)
}

// resolveRepositoriesForLatest discovers repositories based on workspace behavior
func resolveRepositoriesForLatest(cfg *config.Config, behavior WorkspaceBehavior, _ string) ([]RepositoryInfo, error) {
	switch behavior {
	case WorkspaceBehaviorStandalone, WorkspaceBehaviorMonorepo:
		// For standalone/monorepo, return single repository (current directory)
		repoRoot, err := getRepoRoot()
		if err != nil {
			return nil, fmt.Errorf("failed to get repository root: %w", err)
		}

		trunkBranch := ""
		if cfg.Git != nil && cfg.Git.TrunkBranch != "" {
			trunkBranch = cfg.Git.TrunkBranch
		}

		remote := "origin"
		if cfg.Git != nil && cfg.Git.Remote != "" {
			remote = cfg.Git.Remote
		}

		// Use directory name as repository identifier to avoid confusion with branch names
		repoName := filepath.Base(repoRoot)

		return []RepositoryInfo{
			{
				Name:        repoName,
				Path:        repoRoot,
				TrunkBranch: trunkBranch,
				Remote:      remote,
			},
		}, nil

	case WorkspaceBehaviorPolyrepo:
		// For polyrepo, resolve all projects
		repoRoot, err := getRepoRoot()
		if err != nil {
			return nil, fmt.Errorf("failed to get repository root: %w", err)
		}

		projects, err := resolvePolyrepoProjects(cfg, repoRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve polyrepo projects: %w", err)
		}

		var repos []RepositoryInfo
		for _, project := range projects {
			if project.Path == "" {
				continue // Skip projects without paths
			}

			repos = append(repos, RepositoryInfo{
				Name:        project.Name,
				Path:        project.Path,
				TrunkBranch: project.TrunkBranch,
				Remote:      project.Remote,
				RepoRoot:    project.RepoRoot,
			})
		}

		return repos, nil

	default:
		return nil, fmt.Errorf("unknown workspace behavior: %v", behavior)
	}
}

// validateRepositories checks that all repositories exist and are valid git repositories
func validateRepositories(repos []RepositoryInfo) error {
	var errors []string

	for _, repo := range repos {
		// Check if path exists
		if _, err := os.Stat(repo.Path); os.IsNotExist(err) {
			errors = append(errors, fmt.Sprintf("repository path does not exist: %s (for %s)", repo.Path, repo.Name))
			continue
		}

		// Check if path is a git repository
		if !isExternalGitRepo(repo.Path) {
			errors = append(errors, fmt.Sprintf("path is not a git repository: %s (for %s)", repo.Path, repo.Name))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("repository validation failed:\n  %s", strings.Join(errors, "\n  "))
	}

	return nil
}

// checkRepositoryState detects the current state of a repository
func checkRepositoryState(repo RepositoryInfo) (RepositoryStateInfo, error) {
	stateInfo := RepositoryStateInfo{
		Repo: repo,
	}

	// Check for active operations first (rebase/merge)
	if state := checkActiveOperations(repo); state != nil {
		return *state, nil
	}

	// Check git status for uncommitted changes and conflicts
	return checkGitStatus(repo, stateInfo)
}

// checkActiveOperations checks if repository is in the middle of a rebase or merge
func checkActiveOperations(repo RepositoryInfo) *RepositoryStateInfo {
	// Check for active rebase operation
	rebaseMergePath := filepath.Join(repo.Path, ".git", "rebase-merge")
	if _, err := os.Stat(rebaseMergePath); err == nil {
		return &RepositoryStateInfo{
			Repo:    repo,
			State:   StateInRebase,
			Details: "repository is in the middle of a rebase operation",
		}
	}

	// Check for active merge operation
	mergeHeadPath := filepath.Join(repo.Path, ".git", "MERGE_HEAD")
	if _, err := os.Stat(mergeHeadPath); err == nil {
		return &RepositoryStateInfo{
			Repo:    repo,
			State:   StateInMerge,
			Details: "repository is in the middle of a merge operation",
		}
	}

	return nil
}

// checkGitStatus checks git status for conflicts and uncommitted changes
func checkGitStatus(repo RepositoryInfo, stateInfo RepositoryStateInfo) (RepositoryStateInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	statusOutput, err := executeCommand(ctx, "git", []string{"status", "--porcelain"}, repo.Path, false)
	if err != nil {
		stateInfo.State = StateError
		stateInfo.Error = fmt.Errorf("failed to check git status: %w", err)
		return stateInfo, stateInfo.Error
	}

	hasUncommitted := strings.TrimSpace(statusOutput) != ""
	hasConflicts := checkForConflicts(ctx, repo)

	// Determine state based on checks
	if hasConflicts {
		stateInfo.State = StateConflictsExist
		conflictingFiles := extractConflictingFiles(statusOutput)
		if len(conflictingFiles) > 0 {
			stateInfo.Details = fmt.Sprintf("conflicts in: %s", strings.Join(conflictingFiles, ", "))
		} else {
			stateInfo.Details = "merge conflicts detected"
		}
		return stateInfo, nil
	}

	if hasUncommitted {
		stateInfo.State = StateDirtyWorkingDir
		stateInfo.Details = "uncommitted changes detected"
		return stateInfo, nil
	}

	stateInfo.State = StateReadyForUpdate
	stateInfo.Details = "repository is clean and ready for update"
	return stateInfo, nil
}

// checkForConflicts checks for conflict markers in the repository
func checkForConflicts(ctx context.Context, repo RepositoryInfo) bool {
	// Check for conflict markers in tracked files
	diffCheckOutput, err := executeCommand(ctx, "git", []string{"diff", "--check"}, repo.Path, false)
	if err != nil {
		// git diff --check returns non-zero exit code if conflicts found
		// This is expected behavior, so we check the output
		diffCheckOutput, _ = executeCommandCombinedOutput(ctx, "git", []string{"diff", "--check"}, repo.Path, false)
	}

	hasConflicts := strings.Contains(diffCheckOutput, "<<<<<<<") ||
		strings.Contains(diffCheckOutput, "conflict") ||
		strings.Contains(diffCheckOutput, "CONFLICT")

	// Also check for conflict markers in working directory files
	if !hasConflicts {
		hasConflicts = checkStatusForConflicts(ctx, repo)
	}

	return hasConflicts
}

// checkStatusForConflicts checks git status output for conflict indicators
func checkStatusForConflicts(ctx context.Context, repo RepositoryInfo) bool {
	statusFullOutput, err := executeCommand(ctx, "git", []string{"status"}, repo.Path, false)
	if err != nil {
		return false
	}

	return strings.Contains(statusFullOutput, "Unmerged paths") ||
		strings.Contains(statusFullOutput, "both modified") ||
		strings.Contains(statusFullOutput, "both added") ||
		strings.Contains(statusFullOutput, "deleted by them") ||
		strings.Contains(statusFullOutput, "deleted by us")
}

// extractConflictingFiles extracts file paths from git status output that have conflicts
func extractConflictingFiles(statusOutput string) []string {
	var conflictingFiles []string
	lines := strings.Split(statusOutput, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Git status porcelain format: XY filename
		// For conflicts, X or Y can be U (unmerged), AA (both added), DD (both deleted), etc.
		if len(line) >= 2 {
			status := line[:2]
			if strings.Contains(status, "U") || status == "AA" || status == "DD" || status == "AU" || status == "UA" || status == "DU" || status == "UD" {
				// Extract filename (skip status and whitespace)
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					conflictingFiles = append(conflictingFiles, parts[1])
				}
			}
		}
	}
	return conflictingFiles
}

// validateAllReposCleanOrDirtyForUpdate validates that all repositories are in a state
// that allows update operations (ready or dirty). Blocks if any repo has conflicts,
// in-progress operations, or errors.
func validateAllReposCleanOrDirtyForUpdate(aggregated AggregatedState) error {
	var blockingRepos []string
	var blockingReasons []string

	// Check for blocking states
	if len(aggregated.ConflictingRepos) > 0 {
		blockingRepos = append(blockingRepos, aggregated.ConflictingRepos...)
		blockingReasons = append(blockingReasons, "merge conflicts detected")
	}
	if len(aggregated.InOperationRepos) > 0 {
		blockingRepos = append(blockingRepos, aggregated.InOperationRepos...)
		blockingReasons = append(blockingReasons, "in-progress rebase or merge operation")
	}
	if len(aggregated.ErrorRepos) > 0 {
		blockingRepos = append(blockingRepos, aggregated.ErrorRepos...)
		blockingReasons = append(blockingReasons, "error state detected")
	}

	if len(blockingRepos) > 0 {
		// Build detailed error message
		var msg strings.Builder
		msg.WriteString("cannot proceed with update: repositories have blocking states:\n")
		for i, repo := range blockingRepos {
			if i < len(blockingReasons) {
				msg.WriteString(fmt.Sprintf("  - %s: %s\n", repo, blockingReasons[i%len(blockingReasons)]))
			}
		}
		msg.WriteString("\nTo resolve:\n")
		if len(aggregated.ConflictingRepos) > 0 {
			msg.WriteString("  - Resolve merge conflicts in affected repositories\n")
		}
		if len(aggregated.InOperationRepos) > 0 {
			msg.WriteString("  - Complete or abort in-progress rebase/merge operations:\n")
			msg.WriteString("    Run 'git rebase --abort' or 'git merge --abort' in affected repositories\n")
		}
		if len(aggregated.ErrorRepos) > 0 {
			msg.WriteString("  - Fix errors in affected repositories\n")
		}
		return fmt.Errorf("%s", msg.String())
	}

	return nil
}

// aggregateRepositoryStates combines states across multiple repositories
// Priority: conflicts > in_rebase/in_merge > dirty > ready
func aggregateRepositoryStates(states []RepositoryStateInfo) AggregatedState {
	aggregated := AggregatedState{
		StateInfos:       states,
		ConflictingRepos: []string{},
		DirtyRepos:       []string{},
		InOperationRepos: []string{},
		ErrorRepos:       []string{},
		ReadyRepos:       []string{},
	}

	// Categorize repositories by state
	for _, stateInfo := range states {
		switch stateInfo.State {
		case StateConflictsExist:
			aggregated.ConflictingRepos = append(aggregated.ConflictingRepos, stateInfo.Repo.Name)
		case StateDirtyWorkingDir:
			aggregated.DirtyRepos = append(aggregated.DirtyRepos, stateInfo.Repo.Name)
		case StateInRebase, StateInMerge:
			aggregated.InOperationRepos = append(aggregated.InOperationRepos, stateInfo.Repo.Name)
		case StateError:
			aggregated.ErrorRepos = append(aggregated.ErrorRepos, stateInfo.Repo.Name)
		case StateReadyForUpdate:
			aggregated.ReadyRepos = append(aggregated.ReadyRepos, stateInfo.Repo.Name)
		}
	}

	// Determine overall state based on priority
	if len(aggregated.ConflictingRepos) > 0 {
		aggregated.OverallState = StateConflictsExist
	} else if len(aggregated.InOperationRepos) > 0 {
		// If any repo is in rebase, use that state; otherwise use merge
		hasRebase := false
		for _, stateInfo := range states {
			if stateInfo.State == StateInRebase {
				hasRebase = true
				break
			}
		}
		if hasRebase {
			aggregated.OverallState = StateInRebase
		} else {
			aggregated.OverallState = StateInMerge
		}
	} else if len(aggregated.DirtyRepos) > 0 {
		aggregated.OverallState = StateDirtyWorkingDir
	} else if len(aggregated.ErrorRepos) > 0 {
		aggregated.OverallState = StateError
	} else {
		aggregated.OverallState = StateReadyForUpdate
	}

	return aggregated
}

// readConflictingFile safely reads a conflicting file from repository path
func readConflictingFile(repo RepositoryInfo, filePath string) ([]byte, error) {
	// Resolve absolute path
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(repo.Path, filePath)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", absPath)
	}

	// Read file content
	// #nosec G304 - file path is from git status output, validated by git
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", absPath, err)
	}

	// Check if file appears to be binary (simple heuristic: contains null bytes)
	if bytes.Contains(content, []byte{0}) {
		return nil, fmt.Errorf("file appears to be binary: %s", absPath)
	}

	// Check for very large files (warn if > 1MB)
	const maxFileSize = 1024 * 1024 // 1MB
	if len(content) > maxFileSize {
		return nil, fmt.Errorf("file is too large (%d bytes, max %d): %s", len(content), maxFileSize, absPath)
	}

	return content, nil
}

// extractContextLines extracts N lines before and after conflict region
func extractContextLines(lines []string, conflictStart, conflictEnd, contextSize int) (before, after []string) {
	// Extract context before
	beforeStart := conflictStart - contextSize
	if beforeStart < 0 {
		beforeStart = 0
	}
	if beforeStart < conflictStart {
		before = lines[beforeStart:conflictStart]
	}

	// Extract context after
	afterEnd := conflictEnd + contextSize
	if afterEnd > len(lines) {
		afterEnd = len(lines)
	}
	if afterEnd > conflictEnd {
		after = lines[conflictEnd:afterEnd]
	}

	return before, after
}

// Conflict marker constants
const (
	conflictMarkerStart     = "<<<<<<<"
	conflictMarkerSeparator = "======="
	conflictMarkerEnd       = ">>>>>>>"
)

// conflictMarkerPosition represents the position of a conflict marker in a file
type conflictMarkerPosition struct {
	lineIndex int
	marker    string // "<<<<<<<", "=======", ">>>>>>>"
	content   string // The full line including any branch name
}

// findConflictMarkers finds all conflict marker positions in file content
func findConflictMarkers(content []byte) []conflictMarkerPosition {
	var markers []conflictMarkerPosition
	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineIndex := 0

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, conflictMarkerStart) {
			markers = append(markers, conflictMarkerPosition{
				lineIndex: lineIndex,
				marker:    conflictMarkerStart,
				content:   line,
			})
		} else if strings.HasPrefix(trimmed, conflictMarkerSeparator) && len(trimmed) == 7 {
			// Only match exact "=======" separator, not lines that contain it
			markers = append(markers, conflictMarkerPosition{
				lineIndex: lineIndex,
				marker:    conflictMarkerSeparator,
				content:   line,
			})
		} else if strings.HasPrefix(trimmed, conflictMarkerEnd) {
			markers = append(markers, conflictMarkerPosition{
				lineIndex: lineIndex,
				marker:    conflictMarkerEnd,
				content:   line,
			})
		}

		lineIndex++
	}

	return markers
}

// parseConflictMarkers parses conflict markers from file content and extracts conflict regions
func parseConflictMarkers(_ string, content []byte) ([]ConflictRegion, error) {
	lines := strings.Split(string(content), "\n")
	markers := findConflictMarkers(content)

	if len(markers) == 0 {
		return nil, nil
	}

	var regions []ConflictRegion
	const contextSize = 3

	// Group markers into conflict regions (<<<<<<< ... ======= ... >>>>>>>)
	i := 0
	for i < len(markers) {
		// Find start marker (<<<<<<<)
		if markers[i].marker != conflictMarkerStart {
			i++
			continue
		}

		startIdx := markers[i].lineIndex
		startMarker := markers[i].content
		i++

		// Find separator (=======)
		if i >= len(markers) || markers[i].marker != conflictMarkerSeparator {
			// Malformed: missing separator, skip this conflict
			// Try to find next start marker
			for i < len(markers) && markers[i].marker != conflictMarkerStart {
				i++
			}
			continue
		}
		separatorIdx := markers[i].lineIndex
		separator := markers[i].content
		i++

		// Find end marker (>>>>>>>)
		if i >= len(markers) || markers[i].marker != conflictMarkerEnd {
			// Malformed: missing end marker, skip this conflict
			// Try to find next start marker
			for i < len(markers) && markers[i].marker != conflictMarkerStart {
				i++
			}
			continue
		}
		endIdx := markers[i].lineIndex
		endMarker := markers[i].content
		i++

		// Extract content sections
		// Our content: between start marker (inclusive) and separator (exclusive)
		ourLines := lines[startIdx+1 : separatorIdx]
		ourContent := strings.Join(ourLines, "\n")

		// Their content: between separator (exclusive) and end marker (exclusive)
		theirLines := lines[separatorIdx+1 : endIdx]
		theirContent := strings.Join(theirLines, "\n")

		// Extract context
		contextBefore, contextAfter := extractContextLines(lines, startIdx, endIdx+1, contextSize)

		regions = append(regions, ConflictRegion{
			StartMarker:   startMarker,
			OurContent:    ourContent,
			Separator:     separator,
			TheirContent:  theirContent,
			EndMarker:     endMarker,
			ContextBefore: contextBefore,
			ContextAfter:  contextAfter,
		})
	}

	return regions, nil
}

// parseConflictsFromRepository parses all conflicts from a repository
func parseConflictsFromRepository(repo RepositoryInfo, stateInfo RepositoryStateInfo) (*RepositoryConflicts, error) {
	if stateInfo.State != StateConflictsExist {
		return nil, nil
	}

	// Get list of conflicting files
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	statusOutput, err := executeCommand(ctx, "git", []string{"status", "--porcelain"}, repo.Path, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get git status: %w", err)
	}

	conflictingFiles := extractConflictingFiles(statusOutput)
	if len(conflictingFiles) == 0 {
		// No conflicting files found, return empty structure
		return &RepositoryConflicts{
			Repo:  repo,
			Files: []FileConflict{},
		}, nil
	}

	var fileConflicts []FileConflict

	// Parse conflicts from each file
	for _, filePath := range conflictingFiles {
		content, err := readConflictingFile(repo, filePath)
		if err != nil {
			// Add a conflict entry with error to indicate the file couldn't be read
			fileConflicts = append(fileConflicts, FileConflict{
				RepoName: repo.Name,
				FilePath: filePath,
				Regions:  []ConflictRegion{},
				Error:    err,
			})
			continue
		}

		regions, err := parseConflictMarkers(filePath, content)
		if err != nil {
			// Add conflict entry with parsing error
			fileConflicts = append(fileConflicts, FileConflict{
				RepoName: repo.Name,
				FilePath: filePath,
				Regions:  []ConflictRegion{},
				Error:    fmt.Errorf("failed to parse conflict markers: %w", err),
			})
			continue
		}

		// Add file conflict even if no regions found (might have been resolved)
		fileConflicts = append(fileConflicts, FileConflict{
			RepoName: repo.Name,
			FilePath: filePath,
			Regions:  regions,
		})
	}

	return &RepositoryConflicts{
		Repo:  repo,
		Files: fileConflicts,
	}, nil
}

// formatConflictForDisplay formats a single conflict region for terminal display
func formatConflictForDisplay(conflict ConflictRegion, filePath string) string {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("File: %s\n\n", filePath))

	// Context before
	if len(conflict.ContextBefore) > 0 {
		buf.WriteString("Context (3 lines before):\n")
		for _, line := range conflict.ContextBefore {
			buf.WriteString(fmt.Sprintf("  %s\n", line))
		}
		buf.WriteString("\n")
	}

	// Conflict markers and content (display as-is for easy copy-paste)
	buf.WriteString(conflict.StartMarker)
	buf.WriteString("\n")
	if conflict.OurContent != "" {
		// Display our content as-is (no indentation for copy-paste)
		buf.WriteString(conflict.OurContent)
		if !strings.HasSuffix(conflict.OurContent, "\n") {
			buf.WriteString("\n")
		}
	}
	buf.WriteString(conflict.Separator)
	buf.WriteString("\n")
	if conflict.TheirContent != "" {
		// Display their content as-is (no indentation for copy-paste)
		buf.WriteString(conflict.TheirContent)
		if !strings.HasSuffix(conflict.TheirContent, "\n") {
			buf.WriteString("\n")
		}
	}
	buf.WriteString(conflict.EndMarker)
	buf.WriteString("\n")

	// Context after
	if len(conflict.ContextAfter) > 0 {
		buf.WriteString("\nContext (3 lines after):\n")
		for _, line := range conflict.ContextAfter {
			buf.WriteString(fmt.Sprintf("  %s\n", line))
		}
	}

	return buf.String()
}

// formatFileConflicts formats all conflicts in a file
func formatFileConflicts(fileConflict FileConflict) string {
	// Handle error cases
	if fileConflict.Error != nil {
		return fmt.Sprintf("File: %s\n  [Error: %v]\n\n", fileConflict.FilePath, fileConflict.Error)
	}

	if len(fileConflict.Regions) == 0 {
		// File has no conflict regions (might have been resolved or is empty)
		return fmt.Sprintf("File: %s\n  [No conflict regions found - file may have been resolved]\n\n", fileConflict.FilePath)
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("File: %s\n\n", fileConflict.FilePath))

	for i, region := range fileConflict.Regions {
		if i > 0 {
			buf.WriteString("\n───────────────────────────────────────────────────────────────\n\n")
		}
		buf.WriteString(formatConflictForDisplay(region, fileConflict.FilePath))
	}

	return buf.String()
}

// formatRepositoryConflicts formats all conflicts for a repository
func formatRepositoryConflicts(repoConflicts RepositoryConflicts) string {
	if len(repoConflicts.Files) == 0 {
		return ""
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("Repository: %s\n", repoConflicts.Repo.Name))
	buf.WriteString("───────────────────────────────────────────────────────────────\n\n")

	for i, fileConflict := range repoConflicts.Files {
		if i > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(formatFileConflicts(fileConflict))
		if i < len(repoConflicts.Files)-1 {
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

// formatAllConflicts formats conflicts across all repositories
func formatAllConflicts(allConflicts []RepositoryConflicts) string {
	if len(allConflicts) == 0 {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("═══════════════════════════════════════════════════════════════\n")
	buf.WriteString("Merge Conflicts Detected\n")
	buf.WriteString("═══════════════════════════════════════════════════════════════\n\n")

	for i, repoConflicts := range allConflicts {
		if i > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(formatRepositoryConflicts(repoConflicts))
	}

	buf.WriteString("\n\n")
	buf.WriteString("───────────────────────────────────────────────────────────────\n")
	buf.WriteString("[Instructions: Copy the conflict sections above and paste into Cursor or ChatGPT for resolution assistance]\n\n")
	buf.WriteString("To resolve conflicts:\n")
	buf.WriteString("1. Copy the conflict sections above\n")
	buf.WriteString("2. Paste into your LLM tool (Cursor, ChatGPT, etc.)\n")
	buf.WriteString("3. Ask for help resolving the conflicts\n")
	buf.WriteString("4. Apply the resolved code\n")
	buf.WriteString("5. Run 'kira latest' again to continue\n\n")
	buf.WriteString("To abort an in-progress rebase in a repository, run 'git rebase --abort' in that repository.\n")

	return buf.String()
}

// displayAllConflicts parses and displays all conflicts from repositories with conflicts
func displayAllConflicts(stateInfos []RepositoryStateInfo) {
	var allConflicts []RepositoryConflicts

	// Parse conflicts from all repositories that have conflicts
	for _, stateInfo := range stateInfos {
		if stateInfo.State == StateConflictsExist {
			repoConflicts, err := parseConflictsFromRepository(stateInfo.Repo, stateInfo)
			if err != nil {
				// Log error but continue
				fmt.Printf("Warning: Failed to parse conflicts from repository %s: %v\n", stateInfo.Repo.Name, err)
				continue
			}
			if repoConflicts != nil && len(repoConflicts.Files) > 0 {
				allConflicts = append(allConflicts, *repoConflicts)
			}
		}
	}

	// Display formatted conflicts
	if len(allConflicts) > 0 {
		fmt.Println()
		fmt.Print(formatAllConflicts(allConflicts))
	}
}

// RepositoryOperationResult contains the result of a fetch/rebase operation for a repository
type RepositoryOperationResult struct {
	Repo            RepositoryInfo
	Error           error
	Steps           []string // e.g., ["fetch", "rebase"] for progress tracking
	HadStash        bool     // Whether changes were stashed before rebase
	StashPopped     bool     // Whether stash was successfully popped after rebase
	RebaseAttempted bool     // Whether rebase operation was attempted (for rollback purposes)
	RebaseAborted   bool     // Whether rebase was aborted during rollback
}

// isNetworkError checks if an error string indicates a network error
func isNetworkError(errStr string) bool {
	networkPatterns := []string{
		"could not resolve host",
		"unable to access",
		"connection refused",
		"network",
		"timeout",
		"connection timed out",
	}
	for _, pattern := range networkPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// isPermissionError checks if an error string indicates a permission/authentication error
func isPermissionError(errStr string) bool {
	permissionPatterns := []string{
		"permission denied",
		"authentication failed",
		"403",
		"401",
		"could not read from remote",
		"access denied",
		"auth failed",
		"unauthorized",
		"forbidden",
	}
	for _, pattern := range permissionPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// isBranchError checks if an error string indicates a branch/ref error
func isBranchError(errStr string) bool {
	return strings.Contains(errStr, "fatal:") && strings.Contains(errStr, "doesn't exist")
}

// classifyFetchError classifies fetch errors into user-friendly categories
func classifyFetchError(err error, repo RepositoryInfo) error {
	if err == nil {
		return nil
	}

	errStr := strings.ToLower(err.Error())

	// Network errors
	if isNetworkError(errStr) {
		return fmt.Errorf("failed to fetch from %s/%s: network error occurred. Check network connection and try again: %w", repo.Remote, repo.TrunkBranch, err)
	}

	// Permission/authentication errors
	if isPermissionError(errStr) {
		return fmt.Errorf("failed to fetch from %s/%s: permission or authentication error. Check remote access and credentials: %w", repo.Remote, repo.TrunkBranch, err)
	}

	// Branch/ref errors
	if isBranchError(errStr) {
		return fmt.Errorf("failed to fetch from %s/%s: branch '%s' does not exist on remote '%s'", repo.Remote, repo.TrunkBranch, repo.TrunkBranch, repo.Remote)
	}

	// Generic fetch error
	return fmt.Errorf("failed to fetch from %s/%s: %w", repo.Remote, repo.TrunkBranch, err)
}

// fetchFromRemote fetches latest changes from the remote trunk branch
func fetchFromRemote(repo RepositoryInfo) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// Check if remote exists first
	remoteExists, err := checkRemoteExistsForLatest(repo.Remote, repo.Path)
	if err != nil {
		return fmt.Errorf("failed to check remote: %w", err)
	}
	if !remoteExists {
		return fmt.Errorf("remote '%s' does not exist for repository %s", repo.Remote, repo.Name)
	}

	// Fetch from remote
	_, err = executeCommand(ctx, "git", []string{"fetch", repo.Remote, repo.TrunkBranch}, repo.Path, false)
	if err != nil {
		return classifyFetchError(err, repo)
	}

	return nil
}

// checkRemoteExistsForLatest checks if a remote exists in the repository
func checkRemoteExistsForLatest(remoteName, dir string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	_, err := executeCommand(ctx, "git", []string{"remote", "get-url", remoteName}, dir, false)
	if err != nil {
		// Remote doesn't exist
		return false, nil
	}

	return true, nil
}

// checkUncommittedChangesForLatest checks if there are uncommitted changes in the repository
func checkUncommittedChangesForLatest(dir string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	output, err := executeCommand(ctx, "git", []string{"status", "--porcelain"}, dir, false)
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	return strings.TrimSpace(output) != "", nil
}

// stashChanges stashes uncommitted changes in the repository
func stashChanges(repo RepositoryInfo) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// Use git stash push to stash all changes (including untracked files if needed)
	// We use --include-untracked to catch all changes
	_, err := executeCommand(ctx, "git", []string{"stash", "push", "-m", fmt.Sprintf("kira latest: auto-stash before rebase on %s", repo.Name), "--include-untracked"}, repo.Path, false)
	if err != nil {
		errStr := err.Error()
		// If there's nothing to stash, git stash returns an error but that's okay
		if strings.Contains(errStr, "No local changes to save") {
			return nil // Nothing to stash, which is fine
		}
		return fmt.Errorf("failed to stash changes: %w", err)
	}

	return nil
}

// popStash pops the most recent stash in the repository
func popStash(repo RepositoryInfo) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	_, err := executeCommand(ctx, "git", []string{"stash", "pop"}, repo.Path, false)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "No stash entries found") {
			return nil // Nothing to pop, which is fine
		}
		// If pop fails due to conflicts, that's a real error
		if strings.Contains(errStr, "CONFLICT") || strings.Contains(errStr, "conflict") {
			return fmt.Errorf("stash pop failed due to conflicts. Resolve conflicts manually: %w", err)
		}
		return fmt.Errorf("failed to pop stash: %w", err)
	}

	return nil
}

// abortRebase aborts an in-progress rebase operation in the repository
// Returns nil if no rebase is in progress (not an error condition)
func abortRebase(repo RepositoryInfo) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	_, err := executeCommand(ctx, "git", []string{"rebase", "--abort"}, repo.Path, false)
	if err != nil {
		errStr := err.Error()
		// If no rebase is in progress, git rebase --abort returns an error, but that's okay
		if strings.Contains(errStr, "No rebase in progress") ||
			strings.Contains(errStr, "no rebase") ||
			strings.Contains(errStr, "fatal:") && strings.Contains(errStr, "rebase") {
			return nil // No rebase to abort, which is fine
		}
		return fmt.Errorf("failed to abort rebase: %w", err)
	}

	return nil
}

// rebaseOntoTrunk rebases the current branch onto the remote trunk branch
func rebaseOntoTrunk(repo RepositoryInfo) error {
	// Get current branch name
	currentBranch, err := getCurrentBranch(repo.Path)
	if err != nil {
		return fmt.Errorf("failed to determine current branch: %w", err)
	}

	// Don't rebase if already on trunk branch
	if currentBranch == repo.TrunkBranch {
		return fmt.Errorf("already on trunk branch '%s', cannot rebase onto itself", repo.TrunkBranch)
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// Rebase onto remote/trunkBranch
	remoteRef := fmt.Sprintf("%s/%s", repo.Remote, repo.TrunkBranch)
	_, err = executeCommandCombinedOutput(ctx, "git", []string{"rebase", remoteRef}, repo.Path, false)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "CONFLICT") || strings.Contains(errStr, "conflict") {
			return fmt.Errorf("rebase failed due to conflicts. Resolve conflicts and run 'kira latest' again: %w", err)
		}
		if strings.Contains(errStr, "fatal:") && strings.Contains(errStr, "doesn't exist") {
			return fmt.Errorf("rebase failed: remote reference '%s' does not exist. Ensure fetch completed successfully", remoteRef)
		}
		return fmt.Errorf("rebase failed: %w", err)
	}

	return nil
}

// performFetchAndRebase performs both fetch and rebase operations for a repository
// It handles stashing uncommitted changes if present
func performFetchAndRebase(repo RepositoryInfo, noPopStash bool) (bool, error) {
	// Check for uncommitted changes and stash if needed
	hasUncommitted, err := checkUncommittedChangesForLatest(repo.Path)
	if err != nil {
		return false, fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}

	hadStash := false
	if hasUncommitted {
		// Stash changes before proceeding
		if err := stashChanges(repo); err != nil {
			return false, fmt.Errorf("failed to stash changes: %w", err)
		}
		hadStash = true
	}

	// Fetch first
	if err := fetchFromRemote(repo); err != nil {
		// If fetch fails and we stashed, try to restore stash
		if hadStash && !noPopStash {
			_ = popStash(repo) // Best effort to restore
		}
		return hadStash, fmt.Errorf("fetch failed: %w", err)
	}

	// Then rebase
	if err := rebaseOntoTrunk(repo); err != nil {
		// If rebase fails, abort rebase first, then restore stash
		_ = abortRebase(repo) // Best effort to abort rebase
		if hadStash && !noPopStash {
			_ = popStash(repo) // Best effort to restore stash
		}
		return hadStash, fmt.Errorf("rebase failed: %w", err)
	}

	// If we stashed and rebase succeeded, pop the stash (unless flag is set)
	if hadStash && !noPopStash {
		if err := popStash(repo); err != nil {
			// Stash pop failed - this is a problem but rebase succeeded
			return hadStash, fmt.Errorf("rebase succeeded but failed to pop stash: %w. Use 'git stash pop' to restore your changes", err)
		}
	}

	return hadStash, nil
}

// performFetchAndRebaseForAllRepos performs fetch and rebase operations for all repositories in parallel
func performFetchAndRebaseForAllRepos(repos []RepositoryInfo, noPopStash bool) []RepositoryOperationResult {
	var wg sync.WaitGroup
	results := make([]RepositoryOperationResult, len(repos))
	var mu sync.Mutex

	for i, repo := range repos {
		wg.Add(1)
		go func(index int, repository RepositoryInfo) {
			defer wg.Done()
			result := processRepositoryUpdate(repository, noPopStash, &mu)
			mu.Lock()
			results[index] = result
			mu.Unlock()
		}(i, repo)
	}

	wg.Wait()
	return results
}

// processRepositoryUpdate handles the update process for a single repository
func processRepositoryUpdate(repo RepositoryInfo, noPopStash bool, mu *sync.Mutex) RepositoryOperationResult {
	result := RepositoryOperationResult{
		Repo:  repo,
		Steps: []string{},
	}

	// Handle stashing if needed
	if hadStash := handleStashing(&result, repo, mu); !hadStash && result.Error != nil {
		return result
	}

	// Perform fetch
	if err := performFetchStep(&result, repo, mu); err != nil {
		restoreStashIfNeeded(&result, repo, noPopStash)
		return result
	}

	// Perform rebase
	if err := performRebaseStep(&result, repo, mu); err != nil {
		restoreStashIfNeeded(&result, repo, noPopStash)
		return result
	}

	// Pop stash if needed
	handleStashPop(&result, repo, noPopStash, mu)

	mu.Lock()
	displayOperationProgress(repo.Name, "complete")
	mu.Unlock()

	return result
}

// handleStashing checks for uncommitted changes and stashes them if needed
func handleStashing(result *RepositoryOperationResult, repo RepositoryInfo, mu *sync.Mutex) bool {
	hasUncommitted, err := checkUncommittedChangesForLatest(repo.Path)
	if err != nil || !hasUncommitted {
		return false
	}

	mu.Lock()
	displayOperationProgress(repo.Name, "stashing changes")
	mu.Unlock()

	if err := stashChanges(repo); err != nil {
		result.Error = fmt.Errorf("failed to stash changes: %w", err)
		result.Steps = append(result.Steps, "stash (failed)")
		return false
	}

	result.HadStash = true
	result.Steps = append(result.Steps, "stash")
	return true
}

// performFetchStep performs the fetch operation
func performFetchStep(result *RepositoryOperationResult, repo RepositoryInfo, mu *sync.Mutex) error {
	mu.Lock()
	displayOperationProgress(repo.Name, "fetching")
	mu.Unlock()

	if err := fetchFromRemote(repo); err != nil {
		result.Error = fmt.Errorf("fetch failed: %w", err)
		result.Steps = append(result.Steps, "fetch (failed)")
		return err
	}

	result.Steps = append(result.Steps, "fetch")
	return nil
}

// performRebaseStep performs the rebase operation
func performRebaseStep(result *RepositoryOperationResult, repo RepositoryInfo, mu *sync.Mutex) error {
	mu.Lock()
	displayOperationProgress(repo.Name, "rebasing")
	mu.Unlock()

	// Mark that we're attempting rebase (for rollback purposes)
	result.RebaseAttempted = true

	if err := rebaseOntoTrunk(repo); err != nil {
		result.Error = fmt.Errorf("rebase failed: %w", err)
		result.Steps = append(result.Steps, "rebase (failed)")
		return err
	}

	result.Steps = append(result.Steps, "rebase")
	return nil
}

// restoreStashIfNeeded attempts to restore repository state if operation failed
// It aborts rebase first (if rebase was attempted), then restores stash
func restoreStashIfNeeded(result *RepositoryOperationResult, repo RepositoryInfo, noPopStash bool) {
	// If rebase was attempted and failed, abort it first to restore pre-rebase state
	if result.RebaseAttempted {
		if err := abortRebase(repo); err == nil {
			result.RebaseAborted = true
			result.Steps = append(result.Steps, "rebase-abort")
		} else {
			// Log but don't fail - best effort
			result.Steps = append(result.Steps, "rebase-abort (failed)")
		}
	}

	// Then restore stash if we had one
	if result.HadStash && !noPopStash {
		_ = popStash(repo) // Best effort to restore
	}
}

// handleStashPop handles popping the stash after successful rebase
func handleStashPop(result *RepositoryOperationResult, repo RepositoryInfo, noPopStash bool, mu *sync.Mutex) {
	if !result.HadStash {
		return
	}

	if !noPopStash {
		mu.Lock()
		displayOperationProgress(repo.Name, "popping stash")
		mu.Unlock()

		if err := popStash(repo); err != nil {
			result.Error = fmt.Errorf("rebase succeeded but failed to pop stash: %w. Use 'git stash pop' to restore your changes", err)
			result.Steps = append(result.Steps, "stash-pop (failed)")
			return
		}

		result.StashPopped = true
		result.Steps = append(result.Steps, "stash-pop")
	} else {
		result.Steps = append(result.Steps, "stash (kept)")
	}
}

// displayOperationProgress displays progress for a repository operation
func displayOperationProgress(repoName, operation string) {
	fmt.Printf("  Updating %s: %s...\n", repoName, operation)
}

// getRecoverySteps generates recovery steps for a failed repository operation
func getRecoverySteps(result RepositoryOperationResult) []string {
	var recoverySteps []string
	if result.RebaseAttempted && !result.RebaseAborted {
		recoverySteps = append(recoverySteps, fmt.Sprintf("Run 'git rebase --abort' in %s to return to pre-rebase state", result.Repo.Path))
	}
	if result.HadStash && !result.StashPopped {
		if result.RebaseAborted {
			recoverySteps = append(recoverySteps, fmt.Sprintf("Run 'git stash pop' in %s to restore stashed changes", result.Repo.Path))
		} else {
			recoverySteps = append(recoverySteps, fmt.Sprintf("Changes were stashed and not restored. Run 'git stash pop' in %s to restore", result.Repo.Path))
		}
	}
	return recoverySteps
}

// displayFailedResult displays information about a failed repository operation
func displayFailedResult(result RepositoryOperationResult) {
	fmt.Printf("  ✗ %s: FAILED\n", result.Repo.Name)
	fmt.Printf("    Error: %v\n", result.Error)
	if len(result.Steps) > 0 {
		fmt.Printf("    Completed steps: %s\n", strings.Join(result.Steps, ", "))
	}

	recoverySteps := getRecoverySteps(result)
	if len(recoverySteps) > 0 {
		fmt.Printf("    Recovery steps:\n")
		for _, step := range recoverySteps {
			fmt.Printf("      - %s\n", step)
		}
	}
}

// displaySuccessfulResult displays information about a successful repository operation
func displaySuccessfulResult(result RepositoryOperationResult) {
	fmt.Printf("  ✓ %s: SUCCESS\n", result.Repo.Name)
	if len(result.Steps) > 0 {
		fmt.Printf("    Completed: %s\n", strings.Join(result.Steps, ", "))
	}
	if result.HadStash && !result.StashPopped {
		fmt.Printf("    Note: Changes were stashed and remain in stash (use 'git stash pop' to restore)\n")
	}
}

// displayFailedReposGuidance displays overall guidance for failed repositories
func displayFailedReposGuidance(failedRepos []RepositoryOperationResult) {
	if len(failedRepos) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("Next steps for failed repositories:")
	for _, result := range failedRepos {
		fmt.Printf("  %s:\n", result.Repo.Name)
		if result.RebaseAttempted {
			fmt.Printf("    1. If rebase is still in progress, run 'git rebase --abort' in %s\n", result.Repo.Path)
		}
		if result.HadStash && !result.StashPopped {
			fmt.Printf("    2. Restore stashed changes: 'git stash pop' in %s\n", result.Repo.Path)
		}
		fmt.Printf("    3. Fix the issue and run 'kira latest' again\n")
	}
}

// displayOperationResults displays the results of all repository operations
func displayOperationResults(results []RepositoryOperationResult) {
	fmt.Println("\nOperation Results:")
	fmt.Println("───────────────────────────────────────────────────────────────")

	successCount := 0
	failureCount := 0
	var failedRepos []RepositoryOperationResult

	for _, result := range results {
		if result.Error != nil {
			failureCount++
			failedRepos = append(failedRepos, result)
			displayFailedResult(result)
		} else {
			successCount++
			displaySuccessfulResult(result)
		}
	}

	fmt.Println("───────────────────────────────────────────────────────────────")
	fmt.Printf("Summary: %d succeeded, %d failed\n", successCount, failureCount)

	displayFailedReposGuidance(failedRepos)
}
