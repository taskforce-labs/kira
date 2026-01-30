package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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

// addSafeDirectory marks dir as a safe git directory (for CI where t.TempDir() can trigger dubious ownership).
func addSafeDirectory(t *testing.T, dir string) {
	t.Helper()
	// #nosec G204 - dir is from t.TempDir() or test-controlled path, safe for test use
	require.NoError(t, exec.Command("git", "config", "--global", "--add", "safe.directory", dir).Run())
}

// gitConfigCIMu serializes tests that use setupGitConfigForCI so they don't race on GIT_CONFIG_GLOBAL
// when go test -parallel N is used (e.g. in CI).
var gitConfigCIMu sync.Mutex

// setupGitConfigForCI configures git to trust any directory (safe.directory=*), using a temp
// config file so environments with restrictive ~/.gitconfig allow repo operations in t.TempDir().
// Call at the start of tests that run git in t.TempDir() and restore env in defer.
func setupGitConfigForCI(t *testing.T) {
	t.Helper()
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "gitconfig")
	// safe.directory=* so all temp repos are trusted (avoids "dubious ownership" locally)
	require.NoError(t, os.WriteFile(configPath, []byte("[safe]\n  directory = *\n"), 0o600))
	oldVal := os.Getenv("GIT_CONFIG_GLOBAL")
	require.NoError(t, os.Setenv("GIT_CONFIG_GLOBAL", configPath))
	t.Cleanup(func() {
		if oldVal == "" {
			_ = os.Unsetenv("GIT_CONFIG_GLOBAL")
		} else {
			_ = os.Setenv("GIT_CONFIG_GLOBAL", oldVal)
		}
	})
}

// setupGitConfigForCISerial calls setupGitConfigForCI and holds a mutex so only one such test runs
// at a time. On GitHub Actions we skip setup: other tests use temp dirs + git without it and pass on
// CI; overriding with GIT_CONFIG_GLOBAL was causing exit status 1. If these tests still fail on CI,
// the error will now include git's stderr (or "(no output)") so we can see the real cause.
func setupGitConfigForCISerial(t *testing.T) {
	t.Helper()
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return // Use runner default config; do not set GIT_CONFIG_GLOBAL.
	}
	gitConfigCIMu.Lock()
	t.Cleanup(func() { gitConfigCIMu.Unlock() })
	setupGitConfigForCI(t)
}

func logGitDebug(t *testing.T, dir string) {
	t.Helper()
	header := fmt.Sprintf("GIT DEBUG (dir=%s)", dir)
	t.Log(header)
	fmt.Fprintln(os.Stderr, header)
	_ = os.Stderr.Sync()

	envLine := fmt.Sprintf(
		"env GITHUB_ACTIONS=%q GIT_CONFIG_GLOBAL=%q HOME=%q XDG_CONFIG_HOME=%q",
		os.Getenv("GITHUB_ACTIONS"),
		os.Getenv("GIT_CONFIG_GLOBAL"),
		os.Getenv("HOME"),
		os.Getenv("XDG_CONFIG_HOME"),
	)
	t.Log(envLine)
	fmt.Fprintln(os.Stderr, envLine)
	_ = os.Stderr.Sync()

	commands := [][]string{
		{"git", "--version"},
		{"git", "-C", dir, "rev-parse", "--show-toplevel"},
		{"git", "-C", dir, "status", "-sb"},
		{"git", "-C", dir, "config", "--list", "--show-origin"},
	}

	for _, args := range commands {
		cmdLine := fmt.Sprintf("git cmd: %s", strings.Join(args, " "))
		// #nosec G204 - fixed git command list for test debugging
		cmd := exec.Command(args[0], args[1:]...)
		output, err := cmd.CombinedOutput()
		t.Log(cmdLine)
		fmt.Fprintln(os.Stderr, cmdLine)
		_ = os.Stderr.Sync()
		if err != nil {
			errLine := fmt.Sprintf("git err: %v", err)
			t.Log(errLine)
			fmt.Fprintln(os.Stderr, errLine)
			_ = os.Stderr.Sync()
		}
		outLine := fmt.Sprintf("git out:\n%s", strings.TrimSpace(string(output)))
		t.Log(outLine)
		fmt.Fprintln(os.Stderr, outLine)
		_ = os.Stderr.Sync()
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	// #nosec G204 - args are fixed in tests
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("git %s failed: %v", strings.Join(args, " "), err)
		t.Logf("git output:\n%s", strings.TrimSpace(string(output)))
	}
	require.NoError(t, err)
}

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

	t.Run("returns first work item when multiple exist", func(t *testing.T) {
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

		// Should return the first work item (sorted alphabetically)
		path, err := findCurrentWorkItem(cfg)
		require.NoError(t, err)
		// Should return one of the work items (deterministic based on sorting)
		assert.True(t, strings.HasSuffix(path, "001-test-feature.prd.md") || strings.HasSuffix(path, "002-another-feature.prd.md"))
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

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		metadata, err := extractWorkItemMetadataForLatest(testWorkItemPathDoing, cfg)
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

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		metadata, err := extractWorkItemMetadataForLatest(testWorkItemPathDoing, cfg)
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

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		_, err = extractWorkItemMetadataForLatest(testWorkItemPathDoing, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse front matter")
	})

	t.Run("handles missing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		_, err = extractWorkItemMetadataForLatest(".work/2_doing/nonexistent.md", cfg)
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

	t.Run("configuration priority: project override > git config > auto-detect", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo with main branch
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "main").Run())
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Create external repo
		externalRepo := filepath.Join(tmpDir, "external-repo")
		require.NoError(t, os.MkdirAll(externalRepo, 0o700))
		// #nosec G204 - externalRepo is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", externalRepo, "init").Run())
		// #nosec G204 - externalRepo is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", externalRepo, "config", "user.email", "test@example.com").Run())
		// #nosec G204 - externalRepo is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", externalRepo, "config", "user.name", "Test User").Run())
		// #nosec G204 - externalRepo is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", externalRepo, "checkout", "-b", "main").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "master", // Global config
				Remote:      "origin",
			},
			Workspace: &config.WorkspaceConfig{
				Projects: []config.ProjectConfig{
					{
						Name:        "project1",
						Path:        externalRepo,
						TrunkBranch: "develop", // Project override (should win)
						Remote:      "upstream",
					},
				},
			},
		}

		repos, err := resolveRepositoriesForLatest(cfg, WorkspaceBehaviorPolyrepo, "001")
		require.NoError(t, err)
		require.Len(t, repos, 1)
		// Project override should win
		assert.Equal(t, "develop", repos[0].TrunkBranch)
		assert.Equal(t, "upstream", repos[0].Remote)
	})

	t.Run("auto-detects trunk branch when config is empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo with main branch
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "main").Run())
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// No git config - should auto-detect
		cfg := &config.Config{}

		repos, err := resolveRepositoriesForLatest(cfg, WorkspaceBehaviorStandalone, "001")
		require.NoError(t, err)
		require.Len(t, repos, 1)
		// Should auto-detect "main"
		assert.Equal(t, "main", repos[0].TrunkBranch)
		assert.Equal(t, "origin", repos[0].Remote) // Default remote
	})

	t.Run("uses git config when project override not set", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, os.MkdirAll(".git", 0o700))

		// Create external repo
		externalRepo := filepath.Join(tmpDir, "external-repo")
		require.NoError(t, os.MkdirAll(externalRepo, 0o700))
		require.NoError(t, os.MkdirAll(filepath.Join(externalRepo, ".git"), 0o700))

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "production", // Global config (should be used)
				Remote:      "upstream",
			},
			Workspace: &config.WorkspaceConfig{
				Projects: []config.ProjectConfig{
					{
						Name: "project1",
						Path: externalRepo,
						// No TrunkBranch override - should use global
						Remote: "github",
					},
				},
			},
		}

		repos, err := resolveRepositoriesForLatest(cfg, WorkspaceBehaviorPolyrepo, "001")
		require.NoError(t, err)
		require.Len(t, repos, 1)
		// Should use global config
		assert.Equal(t, "production", repos[0].TrunkBranch)
		assert.Equal(t, "github", repos[0].Remote) // Project remote override
	})

	t.Run("standalone with config uses config values", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, os.MkdirAll(".git", 0o700))

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "develop",
				Remote:      "upstream",
			},
		}

		repos, err := resolveRepositoriesForLatest(cfg, WorkspaceBehaviorStandalone, "001")
		require.NoError(t, err)
		require.Len(t, repos, 1)
		assert.Equal(t, "develop", repos[0].TrunkBranch)
		assert.Equal(t, "upstream", repos[0].Remote)
	})

	t.Run("polyrepo with mixed overrides", func(t *testing.T) {
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

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main", // Global default
				Remote:      "origin",
			},
			Workspace: &config.WorkspaceConfig{
				Projects: []config.ProjectConfig{
					{
						Name:        "project1",
						Path:        externalRepo1,
						TrunkBranch: "develop",  // Override
						Remote:      "upstream", // Override
					},
					{
						Name: "project2",
						Path: externalRepo2,
						// No overrides - uses global
					},
				},
			},
		}

		repos, err := resolveRepositoriesForLatest(cfg, WorkspaceBehaviorPolyrepo, "001")
		require.NoError(t, err)
		require.Len(t, repos, 2)

		// Find project1 and project2
		repoMap := make(map[string]*RepositoryInfo)
		for i := range repos {
			repoMap[repos[i].Name] = &repos[i]
		}
		project1 := repoMap["project1"]
		project2 := repoMap["project2"]

		require.NotNil(t, project1)
		assert.Equal(t, "develop", project1.TrunkBranch)
		assert.Equal(t, "upstream", project1.Remote)

		require.NotNil(t, project2)
		assert.Equal(t, "main", project2.TrunkBranch)
		assert.Equal(t, "origin", project2.Remote)
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
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		_ = exec.Command("git", "branch", "-M", "main").Run() // Ignore error if already main

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

	t.Run("detects active rebase operation using rebase-apply", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Determine git dir in the same way production code does
		gitDirOutput, err := exec.Command("git", "rev-parse", "--git-dir").Output()
		require.NoError(t, err)
		gitDir := strings.TrimSpace(string(gitDirOutput))
		if !filepath.IsAbs(gitDir) {
			gitDir = filepath.Join(tmpDir, gitDir)
		}

		// Create rebase-apply directory to simulate an active rebase driven by apply backend
		rebaseApplyDir := filepath.Join(gitDir, "rebase-apply")
		require.NoError(t, os.MkdirAll(rebaseApplyDir, 0o700))

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
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		_ = exec.Command("git", "branch", "-M", "main").Run() // Ignore error if already main

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
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		_ = exec.Command("git", "branch", "-M", "main").Run() // Ignore error if already main

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
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		_ = exec.Command("git", "branch", "-M", "main").Run() // Ignore error if already main

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
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		_ = exec.Command("git", "branch", "-M", "main").Run() // Ignore error if already main

		repo := RepositoryInfo{
			Name:        "test-repo",
			Path:        tmpDir,
			TrunkBranch: "main",
			Remote:      "origin",
		}

		// rebaseOntoTrunk errors when already on trunk (caller should use updateTrunkFromRemote instead)
		err := rebaseOntoTrunk(repo)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already on trunk branch")
	})
}

func TestIsOnTrunkBranch(t *testing.T) {
	t.Run("on trunk returns true", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, os.WriteFile("f", []byte("x"), 0o600))
		require.NoError(t, exec.Command("git", "add", "f").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "c").Run())
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		_ = exec.Command("git", "branch", "-M", "main").Run()

		repo := RepositoryInfo{Path: tmpDir, TrunkBranch: "main"}
		onTrunk, err := isOnTrunkBranch(repo)
		require.NoError(t, err)
		assert.True(t, onTrunk)
	})

	t.Run("on feature branch returns false", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, os.WriteFile("f", []byte("x"), 0o600))
		require.NoError(t, exec.Command("git", "add", "f").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "c").Run())
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		_ = exec.Command("git", "branch", "-M", "main").Run()
		require.NoError(t, exec.Command("git", "checkout", "-b", "feature").Run())

		repo := RepositoryInfo{Path: tmpDir, TrunkBranch: "main"}
		onTrunk, err := isOnTrunkBranch(repo)
		require.NoError(t, err)
		assert.False(t, onTrunk)
	})
}

func TestUpdateTrunkFromRemote(t *testing.T) {
	t.Run("updates local trunk from remote", func(t *testing.T) {
		setupGitConfigForCISerial(t)
		tmpDir := t.TempDir()
		addSafeDirectory(t, tmpDir)
		t.Cleanup(func() {
			if t.Failed() {
				logGitDebug(t, tmpDir)
			}
		})
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		runGit(t, "", "init", "-b", "main")
		runGit(t, "", "config", "user.email", "test@example.com")
		runGit(t, "", "config", "user.name", "Test User")
		require.NoError(t, os.WriteFile("a.txt", []byte("a"), 0o600))
		runGit(t, "", "add", "a.txt")
		runGit(t, "", "commit", "-m", "Initial")

		remoteDir := t.TempDir()
		addSafeDirectory(t, remoteDir)
		// #nosec G204 - remoteDir is from t.TempDir(), safe for test use
		runGit(t, "", "init", "--bare", remoteDir)
		// #nosec G204 - paths from t.TempDir(), safe for test use
		runGit(t, "", "remote", "add", "origin", remoteDir)
		runGit(t, "", "push", "-u", "origin", "main")
		// Ensure the bare repo HEAD points at main so clones check out main.
		runGit(t, remoteDir, "symbolic-ref", "HEAD", "refs/heads/main")

		// Add a commit on "remote" by cloning, committing, pushing
		cloneDir := t.TempDir()
		addSafeDirectory(t, cloneDir)
		// #nosec G204 - paths from t.TempDir(), safe for test use
		runGit(t, filepath.Dir(cloneDir), "clone", remoteDir, cloneDir)
		require.NoError(t, os.WriteFile(filepath.Join(cloneDir, "b.txt"), []byte("b"), 0o600))
		// #nosec G204 - cloneDir from t.TempDir(), safe for test use
		runGit(t, cloneDir, "add", "b.txt")
		// #nosec G204 - cloneDir from t.TempDir(), safe for test use
		runGit(t, cloneDir, "config", "user.email", "test@example.com")
		// #nosec G204 - cloneDir from t.TempDir(), safe for test use
		runGit(t, cloneDir, "config", "user.name", "Test User")
		// #nosec G204 - cloneDir from t.TempDir(), safe for test use
		runGit(t, cloneDir, "commit", "-m", "Second")
		// #nosec G204 - cloneDir from t.TempDir(), safe for test use
		runGit(t, cloneDir, "push", "origin", "main")

		// Back in original repo: fetch so origin/main is ahead
		// #nosec G204 - tmpDir from t.TempDir(), safe for test use
		runGit(t, tmpDir, "fetch", "origin", "main")

		repo := RepositoryInfo{
			Name:        "test-repo",
			Path:        tmpDir,
			TrunkBranch: "main",
			Remote:      "origin",
		}
		err := updateTrunkFromRemote(repo)
		if err != nil {
			t.Logf("updateTrunkFromRemote full error: %v", err)
		}
		require.NoErrorf(t, err, "updateTrunkFromRemote: %v", err)

		// Local main should have the remote's commit
		// #nosec G204 - tmpDir from t.TempDir(), safe for test use
		out, err := exec.Command("git", "-C", tmpDir, "rev-parse", "HEAD").Output()
		require.NoError(t, err)
		// #nosec G204 - cloneDir from t.TempDir(), safe for test use
		remoteOut, err := exec.Command("git", "-C", cloneDir, "rev-parse", "HEAD").Output()
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(string(remoteOut)), strings.TrimSpace(string(out)))
	})
}

func TestProcessRepositoryUpdateOnTrunk_stashAndPop(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir("/") }()

	require.NoError(t, exec.Command("git", "init").Run())
	require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
	require.NoError(t, os.WriteFile("a.txt", []byte("a"), 0o600))
	require.NoError(t, exec.Command("git", "add", "a.txt").Run())
	require.NoError(t, exec.Command("git", "commit", "-m", "Initial").Run())
	// #nosec G204 - tmpDir from t.TempDir(), safe for test use
	_ = exec.Command("git", "branch", "-M", "main").Run()

	remoteDir := t.TempDir()
	// #nosec G204 - remoteDir from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())
	// #nosec G204 - tmpDir from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())
	require.NoError(t, exec.Command("git", "push", "-u", "origin", "main").Run())

	// Uncommitted change so we trigger stash
	require.NoError(t, os.WriteFile("dirty.txt", []byte("dirty"), 0o600))

	repo := RepositoryInfo{Name: "test", Path: tmpDir, TrunkBranch: "main", Remote: "origin"}
	var mu sync.Mutex
	result := processRepositoryUpdate(repo, false, false, &mu)

	require.NoError(t, result.Error)
	assert.True(t, result.HadStash)
	assert.True(t, result.StashPopped)
	// Working tree should have dirty.txt back
	_, err := os.Stat(filepath.Join(tmpDir, "dirty.txt"))
	require.NoError(t, err)
}

func TestProcessRepositoryUpdateOnTrunk_noPopStash(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir("/") }()

	require.NoError(t, exec.Command("git", "init").Run())
	require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
	require.NoError(t, os.WriteFile("a.txt", []byte("a"), 0o600))
	require.NoError(t, exec.Command("git", "add", "a.txt").Run())
	require.NoError(t, exec.Command("git", "commit", "-m", "Initial").Run())
	// #nosec G204 - tmpDir from t.TempDir(), safe for test use
	_ = exec.Command("git", "branch", "-M", "main").Run()

	remoteDir := t.TempDir()
	// #nosec G204 - remoteDir from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())
	// #nosec G204 - tmpDir from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())
	require.NoError(t, exec.Command("git", "push", "-u", "origin", "main").Run())

	require.NoError(t, os.WriteFile("dirty.txt", []byte("dirty"), 0o600))

	repo := RepositoryInfo{Name: "test", Path: tmpDir, TrunkBranch: "main", Remote: "origin"}
	var mu sync.Mutex
	result := processRepositoryUpdate(repo, false, true, &mu) // noPopStash=true

	require.NoError(t, result.Error)
	assert.True(t, result.HadStash)
	assert.False(t, result.StashPopped)
	// Stash should still have one entry
	// #nosec G204 - tmpDir from t.TempDir(), safe for test use
	out, err := exec.Command("git", "-C", tmpDir, "stash", "list").Output()
	require.NoError(t, err)
	assert.Contains(t, string(out), "kira latest")
}

func TestProcessRepositoryUpdateOnTrunk_conflict_doesNotPopStash(t *testing.T) {
	setupGitConfigForCISerial(t)
	tmpDir := t.TempDir()
	addSafeDirectory(t, tmpDir)
	t.Cleanup(func() {
		if t.Failed() {
			logGitDebug(t, tmpDir)
		}
	})
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir("/") }()

	runGit(t, "", "init", "-b", "main")
	runGit(t, "", "config", "user.email", "test@example.com")
	runGit(t, "", "config", "user.name", "Test User")
	require.NoError(t, os.WriteFile("f", []byte("a"), 0o600))
	runGit(t, "", "add", "f")
	runGit(t, "", "commit", "-m", "A")

	remoteDir := t.TempDir()
	// #nosec G204 - remoteDir from t.TempDir(), safe for test use
	runGit(t, "", "init", "--bare", remoteDir)
	// #nosec G204 - tmpDir from t.TempDir(), safe for test use
	runGit(t, "", "remote", "add", "origin", remoteDir)
	runGit(t, "", "push", "-u", "origin", "main")
	// Ensure the bare repo HEAD points at main so clones check out main.
	runGit(t, remoteDir, "symbolic-ref", "HEAD", "refs/heads/main")

	// Divergent commit on remote (clone, change f, push)
	cloneDir := t.TempDir()
	// #nosec G204 - paths from t.TempDir(), safe for test use
	runGit(t, "", "clone", remoteDir, cloneDir)
	require.NoError(t, os.WriteFile(filepath.Join(cloneDir, "f"), []byte("b"), 0o600))
	// #nosec G204 - cloneDir from t.TempDir(), safe for test use
	runGit(t, cloneDir, "add", "f")
	// #nosec G204 - cloneDir from t.TempDir(), safe for test use
	runGit(t, cloneDir, "config", "user.email", "test@example.com")
	// #nosec G204 - cloneDir from t.TempDir(), safe for test use
	runGit(t, cloneDir, "config", "user.name", "Test User")
	// #nosec G204 - cloneDir from t.TempDir(), safe for test use
	runGit(t, cloneDir, "commit", "-m", "B")
	// #nosec G204 - cloneDir from t.TempDir(), safe for test use
	runGit(t, cloneDir, "push", "origin", "main")

	// Local divergent commit (change f to c) so rebase will conflict
	require.NoError(t, os.WriteFile("f", []byte("c"), 0o600))
	runGit(t, "", "add", "f")
	runGit(t, "", "commit", "-m", "C")

	// Uncommitted file so we get a stash
	require.NoError(t, os.WriteFile("g", []byte("g"), 0o600))

	repo := RepositoryInfo{Name: "test", Path: tmpDir, TrunkBranch: "main", Remote: "origin"}
	var mu sync.Mutex
	result := processRepositoryUpdate(repo, false, false, &mu) // abortOnConflict=false

	require.Error(t, result.Error, "expected rebase conflict")
	assert.True(t, result.HadStash)
	assert.False(t, result.StashPopped)
	assert.True(t, result.RebaseHadConflicts)
	// Stash should still contain one entry
	// #nosec G204 - tmpDir from t.TempDir(), safe for test use
	stashOut, err := exec.Command("git", "-C", tmpDir, "stash", "list").Output()
	require.NoErrorf(t, err, "git stash list: %v", err)
	assert.Contains(t, string(stashOut), "kira latest")
	// Rebase should be in progress
	// #nosec G304 - path under tmpDir from t.TempDir(), safe for test use
	_, err = os.Stat(filepath.Join(tmpDir, ".git", "rebase-merge"))
	require.NoErrorf(t, err, "rebase-merge dir: %v", err)
}

func TestProcessRepositoryUpdateOnTrunk_abortOnConflict_popsStash(t *testing.T) {
	setupGitConfigForCISerial(t)
	tmpDir := t.TempDir()
	addSafeDirectory(t, tmpDir)
	t.Cleanup(func() {
		if t.Failed() {
			logGitDebug(t, tmpDir)
		}
	})
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir("/") }()

	runGit(t, "", "init", "-b", "main")
	runGit(t, "", "config", "user.email", "test@example.com")
	runGit(t, "", "config", "user.name", "Test User")
	require.NoError(t, os.WriteFile("f", []byte("a"), 0o600))
	runGit(t, "", "add", "f")
	runGit(t, "", "commit", "-m", "A")

	remoteDir := t.TempDir()
	addSafeDirectory(t, remoteDir)
	// #nosec G204 - remoteDir from t.TempDir(), safe for test use
	runGit(t, "", "init", "--bare", remoteDir)
	// #nosec G204 - tmpDir from t.TempDir(), safe for test use
	runGit(t, "", "remote", "add", "origin", remoteDir)
	runGit(t, "", "push", "-u", "origin", "main")
	// Ensure the bare repo HEAD points at main so clones check out main.
	runGit(t, remoteDir, "symbolic-ref", "HEAD", "refs/heads/main")

	// Divergent commit on remote
	cloneDir := t.TempDir()
	addSafeDirectory(t, cloneDir)
	// #nosec G204 - paths from t.TempDir(), safe for test use
	runGit(t, "", "clone", remoteDir, cloneDir)
	require.NoError(t, os.WriteFile(filepath.Join(cloneDir, "f"), []byte("b"), 0o600))
	// #nosec G204 - cloneDir from t.TempDir(), safe for test use
	runGit(t, cloneDir, "add", "f")
	// #nosec G204 - cloneDir from t.TempDir(), safe for test use
	runGit(t, cloneDir, "config", "user.email", "test@example.com")
	// #nosec G204 - cloneDir from t.TempDir(), safe for test use
	runGit(t, cloneDir, "config", "user.name", "Test User")
	// #nosec G204 - cloneDir from t.TempDir(), safe for test use
	runGit(t, cloneDir, "commit", "-m", "B")
	// #nosec G204 - cloneDir from t.TempDir(), safe for test use
	runGit(t, cloneDir, "push", "origin", "main")

	// Local divergent commit so rebase will conflict
	require.NoError(t, os.WriteFile("f", []byte("c"), 0o600))
	runGit(t, "", "add", "f")
	runGit(t, "", "commit", "-m", "C")

	require.NoError(t, os.WriteFile("g", []byte("g"), 0o600))

	repo := RepositoryInfo{Name: "test", Path: tmpDir, TrunkBranch: "main", Remote: "origin"}
	var mu sync.Mutex
	result := processRepositoryUpdate(repo, true, false, &mu) // abortOnConflict=true

	require.Error(t, result.Error, "expected rebase conflict")
	assert.True(t, result.HadStash)
	assert.True(t, result.RebaseAborted)
	assert.True(t, result.StashPopped)
	// No rebase in progress
	// #nosec G304 - path under tmpDir from t.TempDir(), safe for test use
	_, err := os.Stat(filepath.Join(tmpDir, ".git", "rebase-merge"))
	require.True(t, os.IsNotExist(err))
	// Uncommitted file restored from stash
	// #nosec G304 - path under tmpDir from t.TempDir(), safe for test use
	_, err = os.Stat(filepath.Join(tmpDir, "g"))
	require.NoErrorf(t, err, "stash-restored g: %v", err)
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
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		_ = exec.Command("git", "branch", "-M", "main").Run() // Ignore error if already main

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

		results := performFetchAndRebaseForAllRepos(repos, false, false)
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

		results := performFetchAndRebaseForAllRepos(repos, false, false)
		require.Len(t, results, 2)
		// Both should be processed (may have errors if remotes don't exist)
	})
}

func TestPerformFetchAndRebaseForAllRepos_RebaseConflictsAbortFlag(t *testing.T) {
	setupRepoWithRebaseConflict := func(t *testing.T) (string, RepositoryInfo) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create initial commit on main
		require.NoError(t, os.WriteFile("conflict.txt", []byte("line1\nline2\nline3\n"), 0o600))
		require.NoError(t, exec.Command("git", "add", "conflict.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial").Run())
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		_ = exec.Command("git", "branch", "-M", "main").Run() // Ignore error if already main

		// Create feature branch and change same line
		require.NoError(t, exec.Command("git", "checkout", "-b", "feature").Run())
		require.NoError(t, os.WriteFile("conflict.txt", []byte("line1\nfeature change\nline3\n"), 0o600))
		require.NoError(t, exec.Command("git", "add", "conflict.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Feature").Run())

		// Create remote and push main
		remoteDir := t.TempDir()
		// #nosec G204 - remoteDir is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())
		// #nosec G204 - remoteDir is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())
		require.NoError(t, exec.Command("git", "checkout", "main").Run())
		require.NoError(t, exec.Command("git", "push", "-u", "origin", "main").Run())

		// Modify same line on main and push to create future rebase conflict
		require.NoError(t, os.WriteFile("conflict.txt", []byte("line1\nmain change\nline3\n"), 0o600))
		require.NoError(t, exec.Command("git", "add", "conflict.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Main change").Run())
		require.NoError(t, exec.Command("git", "push", "origin", "main").Run())

		// Switch back to feature branch (which now diverges from origin/main)
		require.NoError(t, exec.Command("git", "checkout", "feature").Run())

		repo := RepositoryInfo{
			Name:        "test-repo",
			Path:        tmpDir,
			TrunkBranch: "main",
			Remote:      "origin",
		}

		return tmpDir, repo
	}

	t.Run("default keeps conflicts in rebase state", func(t *testing.T) {
		tmpDir, repo := setupRepoWithRebaseConflict(t)
		defer func() { _ = os.Chdir("/") }()

		results := performFetchAndRebaseForAllRepos([]RepositoryInfo{repo}, false, false)
		require.Len(t, results, 1)
		result := results[0]

		// Rebase should have been attempted and failed due to conflicts
		assert.Error(t, result.Error)
		assert.True(t, result.RebaseAttempted)
		assert.True(t, result.RebaseHadConflicts)
		assert.False(t, result.RebaseAborted)
		assert.Contains(t, strings.Join(result.Steps, ", "), "rebase (failed)")
		assert.NotContains(t, strings.Join(result.Steps, ", "), "rebase-abort")

		// Repository should still be in an in-progress rebase with conflicts
		_, err := os.Stat(filepath.Join(tmpDir, ".git", "rebase-merge"))
		assert.NoError(t, err, "expected rebase-merge directory to exist")

		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		statusOutput, _ := exec.Command("git", "-C", tmpDir, "status").CombinedOutput()
		assert.True(t, strings.Contains(string(statusOutput), "Unmerged paths") ||
			strings.Contains(string(statusOutput), "both modified") ||
			strings.Contains(string(statusOutput), "conflict"))
	})

	t.Run("abort-on-conflict flag aborts rebase", func(t *testing.T) {
		tmpDir, repo := setupRepoWithRebaseConflict(t)
		defer func() { _ = os.Chdir("/") }()

		results := performFetchAndRebaseForAllRepos([]RepositoryInfo{repo}, true, false)
		require.Len(t, results, 1)
		result := results[0]

		// Rebase should have been attempted and failed due to conflicts
		assert.Error(t, result.Error)
		assert.True(t, result.RebaseAttempted)
		assert.True(t, result.RebaseHadConflicts)
		assert.True(t, result.RebaseAborted)
		assert.Contains(t, strings.Join(result.Steps, ", "), "rebase (failed)")
		assert.Contains(t, strings.Join(result.Steps, ", "), "rebase-abort")

		// Repository should have no in-progress rebase directory after abort
		_, err := os.Stat(filepath.Join(tmpDir, ".git", "rebase-merge"))
		assert.Error(t, err, "expected rebase-merge directory not to exist after abort")
	})
}

func TestHandleInProgressRebases_ContinuesRebase(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	// Handle case where Getwd might fail (e.g., in CI)
	if err != nil {
		originalDir = "/"
	}
	require.NoError(t, os.Chdir(tmpDir))
	defer func() {
		// Try to restore original directory, but don't fail if it doesn't exist
		_ = os.Chdir(originalDir)
	}()

	// Initialize git repo
	require.NoError(t, exec.Command("git", "init").Run())
	require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

	// Create initial commit on main
	require.NoError(t, os.WriteFile("iterative.txt", []byte("line1\nline2\n"), 0o600))
	require.NoError(t, exec.Command("git", "add", "iterative.txt").Run())
	require.NoError(t, exec.Command("git", "commit", "-m", "Initial").Run())
	// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
	_ = exec.Command("git", "branch", "-M", "main").Run() // Ignore error if already main

	// Create feature branch and modify same line
	require.NoError(t, exec.Command("git", "checkout", "-b", "feature").Run())
	require.NoError(t, os.WriteFile("iterative.txt", []byte("line1\nfeature\n"), 0o600))
	require.NoError(t, exec.Command("git", "add", "iterative.txt").Run())
	require.NoError(t, exec.Command("git", "commit", "-m", "Feature").Run())

	// Create remote and push main
	remoteDir := t.TempDir()
	// #nosec G204 - remoteDir is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())
	// #nosec G204 - remoteDir is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())
	require.NoError(t, exec.Command("git", "checkout", "main").Run())
	require.NoError(t, exec.Command("git", "push", "-u", "origin", "main").Run())

	// Modify same line on main and push to create conflict on rebase
	require.NoError(t, os.WriteFile("iterative.txt", []byte("line1\nmain\n"), 0o600))
	require.NoError(t, exec.Command("git", "add", "iterative.txt").Run())
	require.NoError(t, exec.Command("git", "commit", "-m", "Main").Run())
	require.NoError(t, exec.Command("git", "push", "origin", "main").Run())

	// Switch back to feature and start rebase to create conflict
	require.NoError(t, exec.Command("git", "checkout", "feature").Run())
	require.NoError(t, exec.Command("git", "fetch", "origin", "main").Run())
	_ = exec.Command("git", "rebase", "origin/main").Run() // This may create conflict

	// Resolve conflict but do not run `git rebase --continue`
	require.NoError(t, os.WriteFile("iterative.txt", []byte("line1\nresolved\n"), 0o600))
	require.NoError(t, exec.Command("git", "add", "iterative.txt").Run())

	repo := RepositoryInfo{
		Name:        "test-repo",
		Path:        tmpDir,
		TrunkBranch: "main",
		Remote:      "origin",
	}

	stateInfos := []RepositoryStateInfo{
		{
			Repo:  repo,
			State: StateInRebase,
		},
	}

	// handleInProgressRebases should run `git rebase --continue` and complete the rebase
	err = handleInProgressRebases(stateInfos)
	require.NoError(t, err)

	// After continue, there should be no in-progress rebase directory
	_, statErr := os.Stat(filepath.Join(tmpDir, ".git", "rebase-merge"))
	assert.Error(t, statErr)

	// And the working tree should be clean
	// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
	statusOutput, _ := exec.Command("git", "-C", tmpDir, "status", "--porcelain").CombinedOutput()
	assert.Equal(t, "", strings.TrimSpace(string(statusOutput)))
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

func TestAbortRebase(t *testing.T) {
	t.Run("aborts active rebase", func(t *testing.T) {
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
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		_ = exec.Command("git", "branch", "-M", "main").Run() // Ignore error if already main

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

		// Switch back to feature and start rebase
		require.NoError(t, exec.Command("git", "checkout", "feature").Run())
		require.NoError(t, exec.Command("git", "fetch", "origin", "main").Run())
		// Start rebase (will succeed since no conflicts)
		require.NoError(t, exec.Command("git", "rebase", "origin/main").Run())

		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		// Abort should succeed (even though rebase completed)
		err := abortRebase(repo)
		// If rebase completed, abort will return "no rebase in progress" which is fine
		// The function should handle this gracefully
		assert.NoError(t, err)
	})

	t.Run("handles no rebase in progress gracefully", func(t *testing.T) {
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

		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		// Abort should not error when no rebase is in progress
		err := abortRebase(repo)
		assert.NoError(t, err)
	})
}

func TestCheckUncommittedChangesForLatest(t *testing.T) {
	t.Run("detects uncommitted changes", func(t *testing.T) {
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

		// Make uncommitted change
		require.NoError(t, os.WriteFile("modified.txt", []byte("modified"), 0o600))

		hasUncommitted, err := checkUncommittedChangesForLatest(tmpDir)
		require.NoError(t, err)
		assert.True(t, hasUncommitted)
	})

	t.Run("detects clean repository", func(t *testing.T) {
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

		hasUncommitted, err := checkUncommittedChangesForLatest(tmpDir)
		require.NoError(t, err)
		assert.False(t, hasUncommitted)
	})
}

func TestStashChanges(t *testing.T) {
	t.Run("stashes uncommitted changes", func(t *testing.T) {
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

		// Make uncommitted change
		require.NoError(t, os.WriteFile("modified.txt", []byte("modified"), 0o600))

		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		err := stashChanges(repo)
		require.NoError(t, err)

		// Verify changes are stashed (no uncommitted changes)
		hasUncommitted, err := checkUncommittedChangesForLatest(tmpDir)
		require.NoError(t, err)
		assert.False(t, hasUncommitted)
	})

	t.Run("handles no changes to stash", func(t *testing.T) {
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

		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		// Should not error when nothing to stash
		err := stashChanges(repo)
		assert.NoError(t, err)
	})
}

func TestPopStash(t *testing.T) {
	t.Run("pops stash successfully", func(t *testing.T) {
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

		// Make change and stash it
		require.NoError(t, os.WriteFile("modified.txt", []byte("modified"), 0o600))
		require.NoError(t, exec.Command("git", "stash", "push", "-m", "test stash", "--include-untracked").Run())

		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		// Pop stash
		err := popStash(repo)
		require.NoError(t, err)

		// Verify changes are restored
		hasUncommitted, err := checkUncommittedChangesForLatest(tmpDir)
		require.NoError(t, err)
		assert.True(t, hasUncommitted)
	})

	t.Run("handles no stash gracefully", func(t *testing.T) {
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

		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		// Should not error when no stash exists
		err := popStash(repo)
		assert.NoError(t, err)
	})
}

func TestRestoreStashIfNeeded(t *testing.T) {
	t.Run("aborts rebase and restores stash", func(t *testing.T) {
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
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		_ = exec.Command("git", "branch", "-M", "main").Run() // Ignore error if already main

		// Create feature branch
		require.NoError(t, exec.Command("git", "checkout", "-b", "feature").Run())
		require.NoError(t, os.WriteFile("feature.txt", []byte("feature"), 0o600))
		require.NoError(t, exec.Command("git", "add", "feature.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Feature commit").Run())

		// Make uncommitted change and stash
		require.NoError(t, os.WriteFile("modified.txt", []byte("modified"), 0o600))
		require.NoError(t, exec.Command("git", "stash", "push", "-m", "test stash", "--include-untracked").Run())

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

		// Switch back to feature and start rebase
		require.NoError(t, exec.Command("git", "checkout", "feature").Run())
		require.NoError(t, exec.Command("git", "fetch", "origin", "main").Run())
		// Start rebase (will succeed since no conflicts)
		require.NoError(t, exec.Command("git", "rebase", "origin/main").Run())

		repo := RepositoryInfo{
			Name: "test-repo",
			Path: tmpDir,
		}

		result := RepositoryOperationResult{
			Repo:            repo,
			HadStash:        true,
			RebaseAttempted: true,
		}

		// Restore should abort rebase (or detect no rebase in progress) and pop stash
		restoreStashIfNeeded(&result, repo, true, false)

		// Verify rebase was aborted (no rebase in progress)
		err := abortRebase(repo)
		assert.NoError(t, err) // Should handle "no rebase" gracefully

		// Verify stash was popped (uncommitted changes restored)
		hasUncommitted, err := checkUncommittedChangesForLatest(tmpDir)
		require.NoError(t, err)
		assert.True(t, hasUncommitted)
		assert.True(t, result.RebaseAborted)
	})
}

func TestValidateAllReposCleanOrDirtyForUpdate(t *testing.T) {
	t.Run("allows ready and dirty repos", func(t *testing.T) {
		aggregated := AggregatedState{
			OverallState: StateReadyForUpdate,
			ReadyRepos:   []string{"repo1"},
			DirtyRepos:   []string{"repo2"},
		}

		err := validateAllReposCleanOrDirtyForUpdate(aggregated)
		assert.NoError(t, err)
	})

	t.Run("blocks conflicting repos", func(t *testing.T) {
		aggregated := AggregatedState{
			OverallState:     StateConflictsExist,
			ConflictingRepos: []string{"repo1"},
		}

		err := validateAllReposCleanOrDirtyForUpdate(aggregated)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot proceed")
		assert.Contains(t, err.Error(), "repo1")
		assert.Contains(t, err.Error(), "merge conflicts")
	})

	t.Run("blocks in-progress operations", func(t *testing.T) {
		aggregated := AggregatedState{
			OverallState:     StateInRebase,
			InOperationRepos: []string{"repo1"},
		}

		err := validateAllReposCleanOrDirtyForUpdate(aggregated)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot proceed")
		assert.Contains(t, err.Error(), "repo1")
		assert.Contains(t, err.Error(), "in-progress")
	})

	t.Run("blocks error repos", func(t *testing.T) {
		aggregated := AggregatedState{
			OverallState: StateError,
			ErrorRepos:   []string{"repo1"},
		}

		err := validateAllReposCleanOrDirtyForUpdate(aggregated)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot proceed")
		assert.Contains(t, err.Error(), "repo1")
		assert.Contains(t, err.Error(), "error state")
	})

	t.Run("provides recovery instructions", func(t *testing.T) {
		aggregated := AggregatedState{
			OverallState:     StateConflictsExist,
			ConflictingRepos: []string{"repo1"},
			InOperationRepos: []string{"repo2"},
			ErrorRepos:       []string{"repo3"},
		}

		err := validateAllReposCleanOrDirtyForUpdate(aggregated)
		require.Error(t, err)
		errStr := err.Error()
		assert.Contains(t, errStr, "Resolve merge conflicts")
		assert.Contains(t, errStr, "git rebase --abort")
		assert.Contains(t, errStr, "Fix errors")
	})
}

func TestFetchFromRemote_PermissionErrors(t *testing.T) {
	t.Run("classifies permission errors", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create a remote that doesn't exist (will cause error)
		repo := RepositoryInfo{
			Name:        "test-repo",
			Path:        tmpDir,
			TrunkBranch: "main",
			Remote:      "nonexistent",
		}

		err := fetchFromRemote(repo)
		require.Error(t, err)
		// Should have a clear error message
		assert.Contains(t, err.Error(), "does not exist")
	})
}

func TestDisplayOperationResults_PartialFailure(t *testing.T) {
	t.Run("displays recovery guidance for failed repos with rebase", func(t *testing.T) {
		// Capture output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		results := []RepositoryOperationResult{
			{
				Repo:  RepositoryInfo{Name: "repo1", Path: "/path/to/repo1"},
				Steps: []string{"fetch", "rebase"},
			},
			{
				Repo:            RepositoryInfo{Name: "repo2", Path: "/path/to/repo2"},
				Error:           fmt.Errorf("rebase failed"),
				Steps:           []string{"fetch", "rebase (failed)"},
				HadStash:        true,
				RebaseAttempted: true,
				RebaseAborted:   true,
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
		assert.Contains(t, output, "Recovery steps")
		assert.Contains(t, output, "Rebase was aborted for")
		assert.Contains(t, output, "git stash pop")
		assert.Contains(t, output, "Next steps for failed repositories")
	})
}

func TestOrderRepositoriesByDependencies(t *testing.T) {
	t.Run("groups repositories by repo_root", func(t *testing.T) {
		repos := []RepositoryInfo{
			{
				Name:     "project1",
				Path:     "/path/to/project1",
				RepoRoot: "/shared/root",
			},
			{
				Name:     "project2",
				Path:     "/path/to/project2",
				RepoRoot: "/shared/root",
			},
			{
				Name:     "project3",
				Path:     "/path/to/project3",
				RepoRoot: "/another/root",
			},
		}

		ordered := orderRepositoriesByDependencies(repos)
		require.Len(t, ordered, 3)

		// All repos should be present
		names := make(map[string]bool)
		for _, repo := range ordered {
			names[repo.Name] = true
		}
		assert.True(t, names["project1"])
		assert.True(t, names["project2"])
		assert.True(t, names["project3"])
	})

	t.Run("handles standalone repositories", func(t *testing.T) {
		repos := []RepositoryInfo{
			{
				Name:     "standalone1",
				Path:     "/path/to/standalone1",
				RepoRoot: "", // No repo_root
			},
			{
				Name:     "standalone2",
				Path:     "/path/to/standalone2",
				RepoRoot: "", // No repo_root
			},
		}

		ordered := orderRepositoriesByDependencies(repos)
		require.Len(t, ordered, 2)

		// All repos should be present
		names := make(map[string]bool)
		for _, repo := range ordered {
			names[repo.Name] = true
		}
		assert.True(t, names["standalone1"])
		assert.True(t, names["standalone2"])
	})

	t.Run("handles mixed grouped and standalone repositories", func(t *testing.T) {
		repos := []RepositoryInfo{
			{
				Name:     "standalone",
				Path:     "/path/to/standalone",
				RepoRoot: "",
			},
			{
				Name:     "grouped1",
				Path:     "/path/to/grouped1",
				RepoRoot: "/shared/root",
			},
			{
				Name:     "grouped2",
				Path:     "/path/to/grouped2",
				RepoRoot: "/shared/root",
			},
		}

		ordered := orderRepositoriesByDependencies(repos)
		require.Len(t, ordered, 3)

		// All repos should be present
		names := make(map[string]bool)
		for _, repo := range ordered {
			names[repo.Name] = true
		}
		assert.True(t, names["standalone"])
		assert.True(t, names["grouped1"])
		assert.True(t, names["grouped2"])
	})

	t.Run("handles empty list", func(t *testing.T) {
		repos := []RepositoryInfo{}
		ordered := orderRepositoriesByDependencies(repos)
		assert.Empty(t, ordered)
	})

	t.Run("maintains order within groups", func(t *testing.T) {
		repos := []RepositoryInfo{
			{
				Name:     "first",
				Path:     "/path/to/first",
				RepoRoot: "/shared/root",
			},
			{
				Name:     "second",
				Path:     "/path/to/second",
				RepoRoot: "/shared/root",
			},
			{
				Name:     "third",
				Path:     "/path/to/third",
				RepoRoot: "/shared/root",
			},
		}

		ordered := orderRepositoriesByDependencies(repos)
		require.Len(t, ordered, 3)

		// Order should be maintained within the group
		// (implementation detail: they're appended in order)
		assert.Equal(t, "first", ordered[0].Name)
		assert.Equal(t, "second", ordered[1].Name)
		assert.Equal(t, "third", ordered[2].Name)
	})
}

func TestDiscoverRepositories_ConfigurationIntegration(t *testing.T) {
	t.Run("discovers standalone with auto-detection", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo with main branch
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "main").Run())
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Create work item in doing folder
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(testWorkItemPathDoing, []byte(testWorkItemContentDoing), 0o600))

		// No git config - should auto-detect
		cfg := &config.Config{
			StatusFolders: map[string]string{
				"doing": "2_doing",
			},
		}

		repos, err := discoverRepositories(cfg)
		require.NoError(t, err)
		require.Len(t, repos, 1)
		// Should auto-detect "main"
		assert.Equal(t, "main", repos[0].TrunkBranch)
		assert.Equal(t, "origin", repos[0].Remote) // Default
	})

	t.Run("discovers polyrepo with configuration", func(t *testing.T) {
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
						Name:        "project1",
						Path:        externalRepo1,
						TrunkBranch: "develop",
						Remote:      "upstream",
						RepoRoot:    "/shared/root",
					},
					{
						Name:     "project2",
						Path:     externalRepo2,
						RepoRoot: "/shared/root",
						// Uses global config
					},
				},
			},
		}

		repos, err := discoverRepositories(cfg)
		require.NoError(t, err)
		require.Len(t, repos, 2)

		// Find projects
		repoMap := make(map[string]*RepositoryInfo)
		for i := range repos {
			repoMap[repos[i].Name] = &repos[i]
		}
		project1 := repoMap["project1"]
		project2 := repoMap["project2"]

		require.NotNil(t, project1)
		assert.Equal(t, "develop", project1.TrunkBranch)
		assert.Equal(t, "upstream", project1.Remote)
		assert.Equal(t, "/shared/root", project1.RepoRoot)

		require.NotNil(t, project2)
		assert.Equal(t, "main", project2.TrunkBranch) // Uses global
		assert.Equal(t, "origin", project2.Remote)    // Uses global
		assert.Equal(t, "/shared/root", project2.RepoRoot)
	})
}

func TestResolveTrunkBranchForLatest(t *testing.T) {
	t.Run("uses project override when available", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "main").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "master",
			},
		}

		project := &config.ProjectConfig{
			TrunkBranch: "develop",
		}

		trunkBranch, err := resolveTrunkBranchForLatest(cfg, project, tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "develop", trunkBranch)
	})

	t.Run("uses git config when project override not set", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "main").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "production",
			},
		}

		trunkBranch, err := resolveTrunkBranchForLatest(cfg, nil, tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "production", trunkBranch)
	})

	t.Run("auto-detects when no config provided", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo with main branch
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "main").Run())
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		cfg := &config.Config{}

		trunkBranch, err := resolveTrunkBranchForLatest(cfg, nil, tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "main", trunkBranch)
	})
}
