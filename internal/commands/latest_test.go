package commands

import (
	"os"
	"path/filepath"
	"testing"

	"kira/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testWorkItemContentDoing = `---
id: 001
title: Test Feature
status: doing
kind: prd
---
# Test Feature
`
	testWorkItemPathDoing = ".work/2_doing/001-test-feature.prd.md"
)

func TestRunLatest(t *testing.T) {
	t.Run("validates workspace exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// No .work directory exists
		err := runLatest(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a kira workspace")
		assert.Contains(t, err.Error(), "Run 'kira init' first")
	})

	t.Run("returns error when no work item in doing folder", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory and doing folder
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		err := runLatest(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no work item found in doing folder")
	})
}

func TestFindCurrentWorkItem(t *testing.T) {
	t.Run("finds work item in doing folder", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.Config{
			StatusFolders: map[string]string{
				"doing": "2_doing",
			},
		}

		// Create doing folder and work item
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(testWorkItemPathDoing, []byte(testWorkItemContentDoing), 0o600))

		path, err := findCurrentWorkItem(cfg)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(".work", "2_doing", "001-test-feature.prd.md"), path)
	})

	t.Run("uses default doing folder when not configured", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.Config{
			StatusFolders: map[string]string{},
		}

		// Create default doing folder
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(testWorkItemPathDoing, []byte(testWorkItemContentDoing), 0o600))

		path, err := findCurrentWorkItem(cfg)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(".work", "2_doing", "001-test-feature.prd.md"), path)
	})

	t.Run("returns error when no work item exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.Config{
			StatusFolders: map[string]string{
				"doing": "2_doing",
			},
		}

		// Create empty doing folder
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		_, err := findCurrentWorkItem(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no work item found in doing folder")
	})

	t.Run("returns error when multiple work items exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.Config{
			StatusFolders: map[string]string{
				"doing": "2_doing",
			},
		}

		// Create doing folder with multiple work items
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(testWorkItemPathDoing, []byte(testWorkItemContentDoing), 0o600))
		require.NoError(t, os.WriteFile(".work/2_doing/002-another-feature.prd.md", []byte(testWorkItemContentDoing), 0o600))

		_, err := findCurrentWorkItem(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "multiple work items found in doing folder")
	})

	t.Run("returns error when doing folder does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.Config{
			StatusFolders: map[string]string{
				"doing": "2_doing",
			},
		}

		// Create .work but not doing folder
		require.NoError(t, os.MkdirAll(".work", 0o700))

		_, err := findCurrentWorkItem(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "doing folder not found")
	})
}

func TestExtractWorkItemMetadataForLatest(t *testing.T) {
	t.Run("parses valid work item metadata", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: doing
kind: prd
---
# Test Feature
Content here
`
		require.NoError(t, os.WriteFile(testWorkItemPathDoing, []byte(workItemContent), 0o600))

		metadata, err := extractWorkItemMetadataForLatest(testWorkItemPathDoing)
		require.NoError(t, err)
		assert.Equal(t, "001", metadata.ID)
		assert.Equal(t, "Test Feature", metadata.Title)
		assert.Equal(t, "doing", metadata.Status)
		assert.Equal(t, "prd", metadata.Kind)
		assert.Equal(t, testWorkItemPathDoing, metadata.Filepath)
	})

	t.Run("handles missing fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		workItemContent := `---
id: 001
---
# Test Feature
`
		require.NoError(t, os.WriteFile(testWorkItemPathDoing, []byte(workItemContent), 0o600))

		metadata, err := extractWorkItemMetadataForLatest(testWorkItemPathDoing)
		require.NoError(t, err)
		assert.Equal(t, "001", metadata.ID)
		assert.Empty(t, metadata.Title)
		assert.Empty(t, metadata.Status)
		assert.Empty(t, metadata.Kind)
	})

	t.Run("handles invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		workItemContent := `---
id: 001
title: [invalid
---
# Test Feature
`
		require.NoError(t, os.WriteFile(testWorkItemPathDoing, []byte(workItemContent), 0o600))

		_, err := extractWorkItemMetadataForLatest(testWorkItemPathDoing)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse front matter")
	})

	t.Run("handles missing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		_, err := extractWorkItemMetadataForLatest(".work/2_doing/nonexistent.md")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read work item file")
	})
}

func TestDetectWorkspaceBehavior(t *testing.T) {
	t.Run("detects standalone workspace", func(t *testing.T) {
		cfg := &config.Config{
			Workspace: nil,
		}

		behavior := detectWorkspaceBehavior(cfg)
		assert.Equal(t, WorkspaceBehaviorStandalone, behavior)
	})

	t.Run("detects monorepo workspace", func(t *testing.T) {
		cfg := &config.Config{
			Workspace: &config.WorkspaceConfig{
				Projects: []config.ProjectConfig{
					{
						Name: "project1",
						Path: "subdir/project1", // Relative path, not external repo
					},
				},
			},
		}

		behavior := detectWorkspaceBehavior(cfg)
		assert.Equal(t, WorkspaceBehaviorMonorepo, behavior)
	})

	t.Run("detects polyrepo workspace with repo_root", func(t *testing.T) {
		cfg := &config.Config{
			Workspace: &config.WorkspaceConfig{
				Projects: []config.ProjectConfig{
					{
						Name:     "project1",
						RepoRoot: "/some/root",
					},
				},
			},
		}

		behavior := detectWorkspaceBehavior(cfg)
		assert.Equal(t, WorkspaceBehaviorPolyrepo, behavior)
	})
}

func TestResolveRepositoriesForLatest(t *testing.T) {
	t.Run("resolves standalone repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, os.MkdirAll(".git", 0o700))

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
				Remote:      "origin",
			},
		}

		repos, err := resolveRepositoriesForLatest(cfg, WorkspaceBehaviorStandalone, "001")
		require.NoError(t, err)
		require.Len(t, repos, 1)
		assert.Equal(t, "main", repos[0].Name)
		// Use filepath.Clean to handle symlink differences on macOS
		expectedPath, _ := filepath.EvalSymlinks(tmpDir)
		actualPath, _ := filepath.EvalSymlinks(repos[0].Path)
		assert.Equal(t, expectedPath, actualPath)
		assert.Equal(t, "main", repos[0].TrunkBranch)
		assert.Equal(t, "origin", repos[0].Remote)
	})

	t.Run("resolves monorepo as single repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, os.MkdirAll(".git", 0o700))

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "master",
				Remote:      "upstream",
			},
		}

		repos, err := resolveRepositoriesForLatest(cfg, WorkspaceBehaviorMonorepo, "001")
		require.NoError(t, err)
		require.Len(t, repos, 1)
		assert.Equal(t, "main", repos[0].Name)
		assert.Equal(t, "master", repos[0].TrunkBranch)
		assert.Equal(t, "upstream", repos[0].Remote)
	})

	t.Run("resolves polyrepo repositories", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize main git repo
		require.NoError(t, os.MkdirAll(".git", 0o700))

		// Create external repo
		externalRepo := filepath.Join(tmpDir, "external-repo")
		require.NoError(t, os.MkdirAll(externalRepo, 0o700))
		require.NoError(t, os.MkdirAll(filepath.Join(externalRepo, ".git"), 0o700))

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
				Remote:      "origin",
			},
			Workspace: &config.WorkspaceConfig{
				Projects: []config.ProjectConfig{
					{
						Name:        "project1",
						Path:        externalRepo,
						TrunkBranch: "develop",
						Remote:      "upstream",
						RepoRoot:    "/some/root",
					},
				},
			},
		}

		repos, err := resolveRepositoriesForLatest(cfg, WorkspaceBehaviorPolyrepo, "001")
		require.NoError(t, err)
		require.Len(t, repos, 1)
		assert.Equal(t, "project1", repos[0].Name)
		assert.Equal(t, externalRepo, repos[0].Path)
		assert.Equal(t, "develop", repos[0].TrunkBranch)
		assert.Equal(t, "upstream", repos[0].Remote)
		assert.Equal(t, "/some/root", repos[0].RepoRoot)
	})

	t.Run("returns error when not a git repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// No .git directory
		cfg := &config.Config{}

		_, err := resolveRepositoriesForLatest(cfg, WorkspaceBehaviorStandalone, "001")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get repository root")
	})
}

func TestValidateRepositories(t *testing.T) {
	t.Run("validates valid repositories", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create valid git repo
		repoPath := filepath.Join(tmpDir, "repo1")
		require.NoError(t, os.MkdirAll(repoPath, 0o700))
		require.NoError(t, os.MkdirAll(filepath.Join(repoPath, ".git"), 0o700))

		repos := []RepositoryInfo{
			{
				Name: "repo1",
				Path: repoPath,
			},
		}

		err := validateRepositories(repos)
		require.NoError(t, err)
	})

	t.Run("returns error for non-existent path", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		repos := []RepositoryInfo{
			{
				Name: "repo1",
				Path: filepath.Join(tmpDir, "nonexistent"),
			},
		}

		err := validateRepositories(repos)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repository path does not exist")
	})

	t.Run("returns error for non-git directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create directory without .git
		repoPath := filepath.Join(tmpDir, "not-a-repo")
		require.NoError(t, os.MkdirAll(repoPath, 0o700))

		repos := []RepositoryInfo{
			{
				Name: "repo1",
				Path: repoPath,
			},
		}

		err := validateRepositories(repos)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not a git repository")
	})

	t.Run("aggregates multiple errors", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create one valid and two invalid repos
		validRepo := filepath.Join(tmpDir, "valid")
		require.NoError(t, os.MkdirAll(validRepo, 0o700))
		require.NoError(t, os.MkdirAll(filepath.Join(validRepo, ".git"), 0o700))

		invalidRepo := filepath.Join(tmpDir, "invalid")
		require.NoError(t, os.MkdirAll(invalidRepo, 0o700))

		repos := []RepositoryInfo{
			{
				Name: "valid",
				Path: validRepo,
			},
			{
				Name: "invalid",
				Path: invalidRepo,
			},
			{
				Name: "nonexistent",
				Path: filepath.Join(tmpDir, "nonexistent"),
			},
		}

		err := validateRepositories(repos)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repository validation failed")
		assert.Contains(t, err.Error(), "is not a git repository")
		assert.Contains(t, err.Error(), "repository path does not exist")
	})
}

func TestDiscoverRepositories(t *testing.T) {
	t.Run("discovers standalone repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, os.MkdirAll(".git", 0o700))

		// Create work item in doing folder
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(testWorkItemPathDoing, []byte(testWorkItemContentDoing), 0o600))

		cfg := &config.Config{
			StatusFolders: map[string]string{
				"doing": "2_doing",
			},
			Git: &config.GitConfig{
				TrunkBranch: "main",
				Remote:      "origin",
			},
		}

		repos, err := discoverRepositories(cfg)
		require.NoError(t, err)
		require.Len(t, repos, 1)
		assert.Equal(t, "main", repos[0].Name)
		// Use filepath.EvalSymlinks to handle symlink differences on macOS
		expectedPath, _ := filepath.EvalSymlinks(tmpDir)
		actualPath, _ := filepath.EvalSymlinks(repos[0].Path)
		assert.Equal(t, expectedPath, actualPath)
	})

	t.Run("returns error when no work item in doing folder", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, os.MkdirAll(".git", 0o700))

		// Create empty doing folder
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		cfg := &config.Config{
			StatusFolders: map[string]string{
				"doing": "2_doing",
			},
		}

		_, err := discoverRepositories(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no work item found in doing folder")
	})

	t.Run("discovers polyrepo repositories", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize main git repo
		require.NoError(t, os.MkdirAll(".git", 0o700))

		// Create external repos
		externalRepo1 := filepath.Join(tmpDir, "repo1")
		require.NoError(t, os.MkdirAll(externalRepo1, 0o700))
		require.NoError(t, os.MkdirAll(filepath.Join(externalRepo1, ".git"), 0o700))

		externalRepo2 := filepath.Join(tmpDir, "repo2")
		require.NoError(t, os.MkdirAll(externalRepo2, 0o700))
		require.NoError(t, os.MkdirAll(filepath.Join(externalRepo2, ".git"), 0o700))

		// Create work item in doing folder
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(testWorkItemPathDoing, []byte(testWorkItemContentDoing), 0o600))

		cfg := &config.Config{
			StatusFolders: map[string]string{
				"doing": "2_doing",
			},
			Git: &config.GitConfig{
				TrunkBranch: "main",
				Remote:      "origin",
			},
			Workspace: &config.WorkspaceConfig{
				Projects: []config.ProjectConfig{
					{
						Name: "project1",
						Path: externalRepo1,
					},
					{
						Name: "project2",
						Path: externalRepo2,
					},
				},
			},
		}

		repos, err := discoverRepositories(cfg)
		require.NoError(t, err)
		require.Len(t, repos, 2)
		assert.Equal(t, "project1", repos[0].Name)
		assert.Equal(t, externalRepo1, repos[0].Path)
		assert.Equal(t, "project2", repos[1].Name)
		assert.Equal(t, externalRepo2, repos[1].Path)
	})
}
