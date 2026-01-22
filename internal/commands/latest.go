// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

The command can be called repeatedly to work through conflicts progressively.`,
	Args: cobra.NoArgs,
	RunE: runLatest,
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

func runLatest(_ *cobra.Command, _ []string) error {
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
