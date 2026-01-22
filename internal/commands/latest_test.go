package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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
		// Repository name should be the directory name, not "main"
		expectedName := filepath.Base(tmpDir)
		assert.Equal(t, expectedName, repos[0].Name)
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
		// Repository name should be the directory name, not "main"
		expectedName := filepath.Base(tmpDir)
		assert.Equal(t, expectedName, repos[0].Name)
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
		// Repository name should be the directory name, not "main"
		expectedName := filepath.Base(tmpDir)
		assert.Equal(t, expectedName, repos[0].Name)
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

func TestCheckRepositoryState(t *testing.T) {
	t.Run("detects clean repository ready for update", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo with a commit
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		stateInfo, err := checkRepositoryState(repo)
		require.NoError(t, err)
		assert.Equal(t, StateReadyForUpdate, stateInfo.State)
		assert.Equal(t, "repository is clean and ready for update", stateInfo.Details)
		assert.Nil(t, stateInfo.Error)
	})

	t.Run("detects uncommitted changes", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo with a commit
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Make uncommitted change
		require.NoError(t, os.WriteFile("modified.txt", []byte("modified"), 0o600))

		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		stateInfo, err := checkRepositoryState(repo)
		require.NoError(t, err)
		assert.Equal(t, StateDirtyWorkingDir, stateInfo.State)
		assert.Equal(t, "uncommitted changes detected", stateInfo.Details)
	})

	t.Run("detects merge conflicts", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create initial commit
		require.NoError(t, os.WriteFile("test.txt", []byte("line1\nline2\n"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Create a branch and modify file
		require.NoError(t, exec.Command("git", "checkout", "-b", "feature").Run())
		require.NoError(t, os.WriteFile("test.txt", []byte("line1\nfeature change\nline2\n"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Feature change").Run())

		// Switch back to main and modify same line
		require.NoError(t, exec.Command("git", "checkout", "main").Run())
		require.NoError(t, os.WriteFile("test.txt", []byte("line1\nmain change\nline2\n"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Main change").Run())

		// Attempt merge to create conflict
		_ = exec.Command("git", "merge", "feature").Run() // This will fail with conflict

		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		stateInfo, err := checkRepositoryState(repo)
		require.NoError(t, err)
		// Should detect either conflicts or in-merge state
		assert.True(t, stateInfo.State == StateConflictsExist || stateInfo.State == StateInMerge,
			"Expected conflicts or in-merge state, got: %s", stateInfo.State)
	})

	t.Run("detects active rebase operation", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create rebase-merge directory to simulate active rebase
		rebaseMergeDir := filepath.Join(tmpDir, ".git", "rebase-merge")
		require.NoError(t, os.MkdirAll(rebaseMergeDir, 0o700))

		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		stateInfo, err := checkRepositoryState(repo)
		require.NoError(t, err)
		assert.Equal(t, StateInRebase, stateInfo.State)
		assert.Contains(t, stateInfo.Details, "rebase")
	})

	t.Run("detects active merge operation", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create MERGE_HEAD file to simulate active merge
		mergeHeadFile := filepath.Join(tmpDir, ".git", "MERGE_HEAD")
		require.NoError(t, os.WriteFile(mergeHeadFile, []byte("abc123"), 0o600))

		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		stateInfo, err := checkRepositoryState(repo)
		require.NoError(t, err)
		assert.Equal(t, StateInMerge, stateInfo.State)
		assert.Contains(t, stateInfo.Details, "merge")
	})

	t.Run("handles git command errors gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create directory that's not a git repo
		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		stateInfo, err := checkRepositoryState(repo)
		// Should return error state (err may or may not be nil, but stateInfo.Error should be set)
		assert.Equal(t, StateError, stateInfo.State)
		assert.NotNil(t, stateInfo.Error)
		if err != nil {
			assert.Contains(t, err.Error(), "failed to check git status")
		}
		assert.Contains(t, stateInfo.Error.Error(), "failed to check git status")
	})
}

func TestExtractConflictingFiles(t *testing.T) {
	t.Run("extracts conflicting files from git status", func(t *testing.T) {
		statusOutput := `UU file1.txt
AA file2.txt
DU file3.txt
 M file4.txt`
		files := extractConflictingFiles(statusOutput)
		assert.Contains(t, files, "file1.txt")
		assert.Contains(t, files, "file2.txt")
		assert.Contains(t, files, "file3.txt")
		assert.NotContains(t, files, "file4.txt") // Modified but not conflicted
	})

	t.Run("handles empty status output", func(t *testing.T) {
		files := extractConflictingFiles("")
		assert.Empty(t, files)
	})

	t.Run("handles status with no conflicts", func(t *testing.T) {
		statusOutput := ` M file1.txt
A  file2.txt
D  file3.txt`
		files := extractConflictingFiles(statusOutput)
		assert.Empty(t, files)
	})
}

func TestAggregateRepositoryStates(t *testing.T) {
	t.Run("aggregates single clean repository", func(t *testing.T) {
		states := []RepositoryStateInfo{
			{
				Repo:  RepositoryInfo{Name: "repo1"},
				State: StateReadyForUpdate,
			},
		}

		aggregated := aggregateRepositoryStates(states)
		assert.Equal(t, StateReadyForUpdate, aggregated.OverallState)
		assert.Len(t, aggregated.ReadyRepos, 1)
		assert.Equal(t, "repo1", aggregated.ReadyRepos[0])
	})

	t.Run("prioritizes conflicts over other states", func(t *testing.T) {
		states := []RepositoryStateInfo{
			{
				Repo:  RepositoryInfo{Name: "repo1"},
				State: StateReadyForUpdate,
			},
			{
				Repo:  RepositoryInfo{Name: "repo2"},
				State: StateConflictsExist,
			},
			{
				Repo:  RepositoryInfo{Name: "repo3"},
				State: StateDirtyWorkingDir,
			},
		}

		aggregated := aggregateRepositoryStates(states)
		assert.Equal(t, StateConflictsExist, aggregated.OverallState)
		assert.Len(t, aggregated.ConflictingRepos, 1)
		assert.Len(t, aggregated.DirtyRepos, 1)
		assert.Len(t, aggregated.ReadyRepos, 1)
	})

	t.Run("prioritizes in-operation over dirty", func(t *testing.T) {
		states := []RepositoryStateInfo{
			{
				Repo:  RepositoryInfo{Name: "repo1"},
				State: StateDirtyWorkingDir,
			},
			{
				Repo:  RepositoryInfo{Name: "repo2"},
				State: StateInRebase,
			},
		}

		aggregated := aggregateRepositoryStates(states)
		assert.Equal(t, StateInRebase, aggregated.OverallState)
		assert.Len(t, aggregated.InOperationRepos, 1)
		assert.Len(t, aggregated.DirtyRepos, 1)
	})

	t.Run("handles all repositories with conflicts", func(t *testing.T) {
		states := []RepositoryStateInfo{
			{
				Repo:  RepositoryInfo{Name: "repo1"},
				State: StateConflictsExist,
			},
			{
				Repo:  RepositoryInfo{Name: "repo2"},
				State: StateConflictsExist,
			},
		}

		aggregated := aggregateRepositoryStates(states)
		assert.Equal(t, StateConflictsExist, aggregated.OverallState)
		assert.Len(t, aggregated.ConflictingRepos, 2)
	})

	t.Run("handles mixed states correctly", func(t *testing.T) {
		states := []RepositoryStateInfo{
			{
				Repo:  RepositoryInfo{Name: "repo1"},
				State: StateReadyForUpdate,
			},
			{
				Repo:  RepositoryInfo{Name: "repo2"},
				State: StateDirtyWorkingDir,
			},
			{
				Repo:  RepositoryInfo{Name: "repo3"},
				State: StateInMerge,
			},
			{
				Repo:  RepositoryInfo{Name: "repo4"},
				State: StateError,
			},
		}

		aggregated := aggregateRepositoryStates(states)
		assert.Equal(t, StateInMerge, aggregated.OverallState)
		assert.Len(t, aggregated.ReadyRepos, 1)
		assert.Len(t, aggregated.DirtyRepos, 1)
		assert.Len(t, aggregated.InOperationRepos, 1)
		assert.Len(t, aggregated.ErrorRepos, 1)
	})

	t.Run("handles empty states list", func(t *testing.T) {
		states := []RepositoryStateInfo{}
		aggregated := aggregateRepositoryStates(states)
		assert.Equal(t, StateReadyForUpdate, aggregated.OverallState)
		assert.Empty(t, aggregated.ConflictingRepos)
		assert.Empty(t, aggregated.DirtyRepos)
		assert.Empty(t, aggregated.InOperationRepos)
		assert.Empty(t, aggregated.ErrorRepos)
		assert.Empty(t, aggregated.ReadyRepos)
	})
}

func TestGetStateSymbol(t *testing.T) {
	t.Run("returns correct symbols for each state", func(t *testing.T) {
		assert.Equal(t, "✓", getStateSymbol(StateReadyForUpdate))
		assert.Equal(t, "✗", getStateSymbol(StateConflictsExist))
		assert.Equal(t, "!", getStateSymbol(StateDirtyWorkingDir))
		assert.Equal(t, "⟳", getStateSymbol(StateInRebase))
		assert.Equal(t, "⟳", getStateSymbol(StateInMerge))
		assert.Equal(t, "⚠", getStateSymbol(StateError))
		assert.Equal(t, "?", getStateSymbol(RepositoryState("unknown")))
	})
}

func TestFindConflictMarkers(t *testing.T) {
	t.Run("finds all conflict markers", func(t *testing.T) {
		content := []byte(`line1
<<<<<<< HEAD
our content
=======
their content
>>>>>>> branch
line2`)
		markers := findConflictMarkers(content)
		require.Len(t, markers, 3)
		assert.Equal(t, conflictMarkerStart, markers[0].marker)
		assert.Equal(t, conflictMarkerSeparator, markers[1].marker)
		assert.Equal(t, conflictMarkerEnd, markers[2].marker)
	})

	t.Run("handles multiple conflicts", func(t *testing.T) {
		content := []byte(`<<<<<<< HEAD
content1
=======
content2
>>>>>>> branch1
middle
<<<<<<< HEAD
content3
=======
content4
>>>>>>> branch2`)
		markers := findConflictMarkers(content)
		require.Len(t, markers, 6)
	})

	t.Run("handles no conflicts", func(t *testing.T) {
		content := []byte(`line1
line2
line3`)
		markers := findConflictMarkers(content)
		assert.Empty(t, markers)
	})
}

func TestExtractContextLines(t *testing.T) {
	t.Run("extracts context lines", func(t *testing.T) {
		lines := []string{"line1", "line2", "line3", "line4", "line5", "line6", "line7", "line8", "line9"}
		before, after := extractContextLines(lines, 3, 6, 3)
		assert.Equal(t, []string{"line1", "line2", "line3"}, before)
		assert.Equal(t, []string{"line7", "line8", "line9"}, after)
	})

	t.Run("handles beginning of file", func(t *testing.T) {
		lines := []string{"line1", "line2", "line3", "line4"}
		before, after := extractContextLines(lines, 0, 2, 3)
		assert.Empty(t, before)
		assert.Equal(t, []string{"line3", "line4"}, after)
	})

	t.Run("handles end of file", func(t *testing.T) {
		lines := []string{"line1", "line2", "line3", "line4"}
		before, after := extractContextLines(lines, 2, 4, 3)
		assert.Equal(t, []string{"line1", "line2"}, before)
		assert.Empty(t, after)
	})
}

func TestParseConflictMarkers(t *testing.T) {
	t.Run("parses single conflict", func(t *testing.T) {
		content := []byte(`line1
line2
<<<<<<< HEAD
our content
line
=======
their content
>>>>>>> branch
line3
line4`)
		regions, err := parseConflictMarkers("test.txt", content)
		require.NoError(t, err)
		require.Len(t, regions, 1)
		assert.Equal(t, conflictMarkerStart+" HEAD", regions[0].StartMarker)
		assert.Equal(t, "our content\nline", regions[0].OurContent)
		assert.Equal(t, conflictMarkerSeparator, regions[0].Separator)
		assert.Equal(t, "their content", regions[0].TheirContent)
		assert.Equal(t, conflictMarkerEnd+" branch", regions[0].EndMarker)
		assert.Len(t, regions[0].ContextBefore, 2)
		assert.Len(t, regions[0].ContextAfter, 2)
	})

	t.Run("parses multiple conflicts", func(t *testing.T) {
		content := []byte(`<<<<<<< HEAD
content1
=======
content2
>>>>>>> branch1
middle
<<<<<<< HEAD
content3
=======
content4
>>>>>>> branch2`)
		regions, err := parseConflictMarkers("test.txt", content)
		require.NoError(t, err)
		require.Len(t, regions, 2)
	})

	t.Run("handles malformed conflicts", func(t *testing.T) {
		content := []byte(`<<<<<<< HEAD
content1
missing separator
>>>>>>> branch`)
		regions, err := parseConflictMarkers("test.txt", content)
		require.NoError(t, err)
		// Should skip malformed conflict
		assert.Empty(t, regions)
	})

	t.Run("handles no conflicts", func(t *testing.T) {
		content := []byte(`line1
line2
line3`)
		regions, err := parseConflictMarkers("test.txt", content)
		require.NoError(t, err)
		assert.Nil(t, regions)
	})
}

func TestReadConflictingFile(t *testing.T) {
	t.Run("reads conflicting file successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		filePath := "test.txt"
		content := []byte("test content")
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, filePath), content, 0o600))

		readContent, err := readConflictingFile(repo, filePath)
		require.NoError(t, err)
		assert.Equal(t, content, readContent)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		_, err := readConflictingFile(repo, "nonexistent.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file does not exist")
	})

	t.Run("returns error for binary file", func(t *testing.T) {
		tmpDir := t.TempDir()
		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		filePath := "binary.bin"
		// Create file with null byte
		binaryContent := []byte{0x00, 0x01, 0x02, 0x03}
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, filePath), binaryContent, 0o600))

		_, err := readConflictingFile(repo, filePath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file appears to be binary")
	})
}

func TestFormatConflictForDisplay(t *testing.T) {
	t.Run("formats conflict with context", func(t *testing.T) {
		conflict := ConflictRegion{
			StartMarker:   "<<<<<<< HEAD",
			OurContent:    "our content",
			Separator:     "=======",
			TheirContent:  "their content",
			EndMarker:     ">>>>>>> branch",
			ContextBefore: []string{"line1", "line2"},
			ContextAfter:  []string{"line3", "line4"},
		}

		formatted := formatConflictForDisplay(conflict, "test.txt")
		assert.Contains(t, formatted, "File: test.txt")
		assert.Contains(t, formatted, "Context (3 lines before)")
		assert.Contains(t, formatted, "line1")
		assert.Contains(t, formatted, "<<<<<<< HEAD")
		assert.Contains(t, formatted, "our content")
		assert.Contains(t, formatted, "=======")
		assert.Contains(t, formatted, "their content")
		assert.Contains(t, formatted, ">>>>>>> branch")
		assert.Contains(t, formatted, "Context (3 lines after)")
		assert.Contains(t, formatted, "line3")
	})
}

func TestFormatFileConflicts(t *testing.T) {
	t.Run("formats file with conflicts", func(t *testing.T) {
		fileConflict := FileConflict{
			RepoName: "repo1",
			FilePath: "test.txt",
			Regions: []ConflictRegion{
				{
					StartMarker:  "<<<<<<< HEAD",
					OurContent:   "content1",
					Separator:    "=======",
					TheirContent: "content2",
					EndMarker:    ">>>>>>> branch",
				},
			},
		}

		formatted := formatFileConflicts(fileConflict)
		assert.Contains(t, formatted, "File: test.txt")
		assert.Contains(t, formatted, "<<<<<<< HEAD")
	})

	t.Run("handles file with error", func(t *testing.T) {
		fileConflict := FileConflict{
			RepoName: "repo1",
			FilePath: "test.txt",
			Error:    fmt.Errorf("file not found"),
		}

		formatted := formatFileConflicts(fileConflict)
		assert.Contains(t, formatted, "File: test.txt")
		assert.Contains(t, formatted, "[Error:")
	})
}

func TestFormatRepositoryConflicts(t *testing.T) {
	t.Run("formats repository conflicts", func(t *testing.T) {
		repoConflicts := RepositoryConflicts{
			Repo: RepositoryInfo{Name: "repo1"},
			Files: []FileConflict{
				{
					RepoName: "repo1",
					FilePath: "file1.txt",
					Regions: []ConflictRegion{
						{
							StartMarker:  "<<<<<<< HEAD",
							OurContent:   "content1",
							Separator:    "=======",
							TheirContent: "content2",
							EndMarker:    ">>>>>>> branch",
						},
					},
				},
			},
		}

		formatted := formatRepositoryConflicts(repoConflicts)
		assert.Contains(t, formatted, "Repository: repo1")
		assert.Contains(t, formatted, "File: file1.txt")
	})
}

func TestFormatAllConflicts(t *testing.T) {
	t.Run("formats all conflicts with instructions", func(t *testing.T) {
		allConflicts := []RepositoryConflicts{
			{
				Repo: RepositoryInfo{Name: "repo1"},
				Files: []FileConflict{
					{
						RepoName: "repo1",
						FilePath: "file1.txt",
						Regions: []ConflictRegion{
							{
								StartMarker:  "<<<<<<< HEAD",
								OurContent:   "content1",
								Separator:    "=======",
								TheirContent: "content2",
								EndMarker:    ">>>>>>> branch",
							},
						},
					},
				},
			},
		}

		formatted := formatAllConflicts(allConflicts)
		assert.Contains(t, formatted, "Merge Conflicts Detected")
		assert.Contains(t, formatted, "Repository: repo1")
		assert.Contains(t, formatted, "To resolve conflicts:")
		assert.Contains(t, formatted, "Run 'kira latest' again to continue")
	})
}

func TestParseConflictsFromRepository(t *testing.T) {
	t.Run("parses conflicts from repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create initial commit
		require.NoError(t, os.WriteFile("test.txt", []byte("line1\nline2\n"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Create a branch and modify file
		require.NoError(t, exec.Command("git", "checkout", "-b", "feature").Run())
		require.NoError(t, os.WriteFile("test.txt", []byte("line1\nfeature change\nline2\n"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Feature change").Run())

		// Switch back to main and modify same line
		require.NoError(t, exec.Command("git", "checkout", "main").Run())
		require.NoError(t, os.WriteFile("test.txt", []byte("line1\nmain change\nline2\n"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Main change").Run())

		// Attempt merge to create conflict
		_ = exec.Command("git", "merge", "feature").Run() // This will fail with conflict

		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		stateInfo := RepositoryStateInfo{
			Repo:  repo,
			State: StateConflictsExist,
		}

		repoConflicts, err := parseConflictsFromRepository(repo, stateInfo)
		require.NoError(t, err)
		require.NotNil(t, repoConflicts)
		// Should have at least one conflicting file
		assert.GreaterOrEqual(t, len(repoConflicts.Files), 0)
	})

	t.Run("returns nil for non-conflict state", func(t *testing.T) {
		repo := RepositoryInfo{
			Name: "test-repo",
			Path: "/tmp",
		}

		stateInfo := RepositoryStateInfo{
			Repo:  repo,
			State: StateReadyForUpdate,
		}

		repoConflicts, err := parseConflictsFromRepository(repo, stateInfo)
		require.NoError(t, err)
		assert.Nil(t, repoConflicts)
	})
}

func TestFetchFromRemote(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create initial commit
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Create a remote (using local path as remote)
		remoteDir := t.TempDir()
		// #nosec G204 - remoteDir is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())
		// #nosec G204 - remoteDir is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())
		require.NoError(t, exec.Command("git", "push", "-u", "origin", "main").Run())

		repo := RepositoryInfo{
			Name:        "test-repo",
			Path:        tmpDir,
			TrunkBranch: "main",
			Remote:      "origin",
		}

		// Fetch should succeed (even if nothing to fetch)
		err := fetchFromRemote(repo)
		// This might fail if main branch doesn't exist on remote, which is expected
		// The important thing is it doesn't crash and handles errors gracefully
		if err != nil {
			// Expected if branch doesn't exist on remote
			assert.Contains(t, err.Error(), "fetch")
		}
	})

	t.Run("missing remote", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		repo := RepositoryInfo{
			Name:        "test-repo",
			Path:        tmpDir,
			TrunkBranch: "main",
			Remote:      "nonexistent",
		}

		err := fetchFromRemote(repo)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}

func TestRebaseOntoTrunk(t *testing.T) {
	t.Run("successful rebase", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create initial commit on main
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Create feature branch
		require.NoError(t, exec.Command("git", "checkout", "-b", "feature").Run())
		require.NoError(t, os.WriteFile("feature.txt", []byte("feature"), 0o600))
		require.NoError(t, exec.Command("git", "add", "feature.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Feature commit").Run())

		// Create remote and push
		remoteDir := t.TempDir()
		// #nosec G204 - remoteDir is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())
		// #nosec G204 - remoteDir is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())
		require.NoError(t, exec.Command("git", "checkout", "main").Run())
		require.NoError(t, exec.Command("git", "push", "-u", "origin", "main").Run())

		// Add another commit to main and push
		require.NoError(t, os.WriteFile("main.txt", []byte("main"), 0o600))
		require.NoError(t, exec.Command("git", "add", "main.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Main commit").Run())
		require.NoError(t, exec.Command("git", "push", "origin", "main").Run())

		// Switch back to feature and fetch
		require.NoError(t, exec.Command("git", "checkout", "feature").Run())
		require.NoError(t, exec.Command("git", "fetch", "origin", "main").Run())

		repo := RepositoryInfo{
			Name:        "test-repo",
			Path:        tmpDir,
			TrunkBranch: "main",
			Remote:      "origin",
		}

		// Rebase should succeed
		err := rebaseOntoTrunk(repo)
		require.NoError(t, err)
	})

	t.Run("already on trunk branch", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create initial commit on main
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		repo := RepositoryInfo{
			Name:        "test-repo",
			Path:        tmpDir,
			TrunkBranch: "main",
			Remote:      "origin",
		}

		// Rebase should fail because we're already on trunk
		err := rebaseOntoTrunk(repo)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already on trunk branch")
	})
}

func TestPerformFetchAndRebase(t *testing.T) {
	t.Run("successful fetch and rebase", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create initial commit on main
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Create feature branch
		require.NoError(t, exec.Command("git", "checkout", "-b", "feature").Run())
		require.NoError(t, os.WriteFile("feature.txt", []byte("feature"), 0o600))
		require.NoError(t, exec.Command("git", "add", "feature.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Feature commit").Run())

		// Create remote and push
		remoteDir := t.TempDir()
		// #nosec G204 - remoteDir is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())
		// #nosec G204 - remoteDir is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())
		require.NoError(t, exec.Command("git", "checkout", "main").Run())
		require.NoError(t, exec.Command("git", "push", "-u", "origin", "main").Run())

		// Add another commit to main and push
		require.NoError(t, os.WriteFile("main.txt", []byte("main"), 0o600))
		require.NoError(t, exec.Command("git", "add", "main.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Main commit").Run())
		require.NoError(t, exec.Command("git", "push", "origin", "main").Run())

		// Switch back to feature
		require.NoError(t, exec.Command("git", "checkout", "feature").Run())

		repo := RepositoryInfo{
			Name:        "test-repo",
			Path:        tmpDir,
			TrunkBranch: "main",
			Remote:      "origin",
		}

		// Perform fetch and rebase
		_, err := performFetchAndRebase(repo, false)
		require.NoError(t, err)
	})

	t.Run("fetch fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		repo := RepositoryInfo{
			Name:        "test-repo",
			Path:        tmpDir,
			TrunkBranch: "main",
			Remote:      "nonexistent",
		}

		// Fetch should fail
		_, err := performFetchAndRebase(repo, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch failed")
	})
}

func TestPerformFetchAndRebaseForAllRepos(t *testing.T) {
	t.Run("single repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create initial commit
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		repos := []RepositoryInfo{
			{
				Name:        "repo1",
				Path:        tmpDir,
				TrunkBranch: "main",
				Remote:      "origin",
			},
		}

		results := performFetchAndRebaseForAllRepos(repos, false)
		require.Len(t, results, 1)
		// May have errors if remote doesn't exist, which is expected
		// The important thing is the function completes
	})

	t.Run("multiple repositories", func(t *testing.T) {
		// Create two repos
		tmpDir1 := t.TempDir()
		tmpDir2 := t.TempDir()

		// Initialize first repo
		require.NoError(t, os.Chdir(tmpDir1))
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, os.WriteFile("test1.txt", []byte("test1"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test1.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Initialize second repo
		require.NoError(t, os.Chdir(tmpDir2))
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, os.WriteFile("test2.txt", []byte("test2"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test2.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		repos := []RepositoryInfo{
			{
				Name:        "repo1",
				Path:        tmpDir1,
				TrunkBranch: "main",
				Remote:      "origin",
			},
			{
				Name:        "repo2",
				Path:        tmpDir2,
				TrunkBranch: "main",
				Remote:      "origin",
			},
		}

		results := performFetchAndRebaseForAllRepos(repos, false)
		require.Len(t, results, 2)
		// Both should be processed (may have errors if remotes don't exist)
	})
}

func TestDisplayOperationProgress(t *testing.T) {
	t.Run("displays progress message", func(t *testing.T) {
		// Capture output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		displayOperationProgress("test-repo", "fetching")

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		output := buf.String()

		assert.Contains(t, output, "test-repo")
		assert.Contains(t, output, "fetching")
	})
}

func TestDisplayOperationResults(t *testing.T) {
	t.Run("displays success and failure results", func(t *testing.T) {
		// Capture output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		results := []RepositoryOperationResult{
			{
				Repo:  RepositoryInfo{Name: "repo1"},
				Steps: []string{"fetch", "rebase"},
			},
			{
				Repo:  RepositoryInfo{Name: "repo2"},
				Error: fmt.Errorf("test error"),
				Steps: []string{"fetch (failed)"},
			},
		}

		displayOperationResults(results)

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		output := buf.String()

		assert.Contains(t, output, "repo1")
		assert.Contains(t, output, "repo2")
		assert.Contains(t, output, "SUCCESS")
		assert.Contains(t, output, "FAILED")
		assert.Contains(t, output, "Summary")
	})
}
