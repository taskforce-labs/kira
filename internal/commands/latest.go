// Package commands implements the CLI commands for the kira tool.
package commands

import (
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
	Name        string // Project name or "main" for standalone
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

	// For Phase 2, just display discovered repositories
	if len(repos) == 0 {
		return fmt.Errorf("no repositories found for current work item")
	}

	fmt.Printf("Discovered %d repository(ies) for current work item:\n", len(repos))
	for _, repo := range repos {
		fmt.Printf("  - %s: %s (trunk: %s, remote: %s)\n", repo.Name, repo.Path, repo.TrunkBranch, repo.Remote)
	}

	return nil
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

		return []RepositoryInfo{
			{
				Name:        "main",
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
