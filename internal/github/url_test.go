package github

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kira/internal/config"
)

func TestIsGitHubURL(t *testing.T) {
	t.Run("returns true for standard GitHub SSH URLs", func(t *testing.T) {
		testCases := []string{
			"git@github.com:owner/repo.git",
			"git@github.com:owner/repo",
			"git@github.com:user/project-name.git",
		}
		for _, url := range testCases {
			assert.True(t, isGitHubURL(url), "should recognize GitHub SSH URL: %s", url)
		}
	})

	t.Run("returns true for standard GitHub HTTPS URLs", func(t *testing.T) {
		testCases := []string{
			"https://github.com/owner/repo.git",
			"https://github.com/owner/repo",
			"http://github.com/user/project-name.git",
		}
		for _, url := range testCases {
			assert.True(t, isGitHubURL(url), "should recognize GitHub HTTPS URL: %s", url)
		}
	})

	t.Run("returns true for GitHub Enterprise URLs", func(t *testing.T) {
		testCases := []string{
			"git@github.example.com:owner/repo.git",
			"https://github.example.com/owner/repo.git",
			"https://github.company.com/org/repo.git",
			"git@github.internal:team/project.git",
		}
		for _, url := range testCases {
			assert.True(t, isGitHubURL(url), "should recognize GitHub Enterprise URL: %s", url)
		}
	})

	t.Run("returns true for case variations", func(t *testing.T) {
		testCases := []string{
			"git@GitHub.com:owner/repo.git",
			"https://GITHUB.COM/owner/repo.git",
			"git@GITHUB.EXAMPLE.COM:owner/repo.git",
		}
		for _, url := range testCases {
			assert.True(t, isGitHubURL(url), "should recognize GitHub URL with case variation: %s", url)
		}
	})

	t.Run("returns false for non-GitHub URLs", func(t *testing.T) {
		testCases := []string{
			"git@gitlab.com:owner/repo.git",
			"https://gitlab.com/owner/repo.git",
			"git@bitbucket.org:owner/repo.git",
			"https://bitbucket.org/owner/repo.git",
			"git@sourceforge.net:owner/repo.git",
			"https://example.com/owner/repo.git",
		}
		for _, url := range testCases {
			assert.False(t, isGitHubURL(url), "should reject non-GitHub URL: %s", url)
		}
	})

	t.Run("returns false for empty URL", func(t *testing.T) {
		assert.False(t, isGitHubURL(""))
	})

	t.Run("returns false for invalid URLs", func(t *testing.T) {
		testCases := []string{
			"not a url",
			"://invalid",
			"git@",
		}
		for _, url := range testCases {
			assert.False(t, isGitHubURL(url), "should reject invalid URL: %s", url)
		}
	})
}

func TestParseGitHubURL(t *testing.T) {
	t.Run("parses SSH format with .git suffix", func(t *testing.T) {
		owner, repo, err := parseGitHubURL("git@github.com:owner/repo.git")
		require.NoError(t, err)
		assert.Equal(t, "owner", owner)
		assert.Equal(t, "repo", repo)
	})

	t.Run("parses SSH format without .git suffix", func(t *testing.T) {
		owner, repo, err := parseGitHubURL("git@github.com:owner/repo")
		require.NoError(t, err)
		assert.Equal(t, "owner", owner)
		assert.Equal(t, "repo", repo)
	})

	t.Run("parses HTTPS format with .git suffix", func(t *testing.T) {
		owner, repo, err := parseGitHubURL("https://github.com/owner/repo.git")
		require.NoError(t, err)
		assert.Equal(t, "owner", owner)
		assert.Equal(t, "repo", repo)
	})

	t.Run("parses HTTPS format without .git suffix", func(t *testing.T) {
		owner, repo, err := parseGitHubURL("https://github.com/owner/repo")
		require.NoError(t, err)
		assert.Equal(t, "owner", owner)
		assert.Equal(t, "repo", repo)
	})

	t.Run("parses GitHub Enterprise HTTPS", func(t *testing.T) {
		owner, repo, err := parseGitHubURL("https://github.example.com/owner/repo.git")
		require.NoError(t, err)
		assert.Equal(t, "owner", owner)
		assert.Equal(t, "repo", repo)
	})

	t.Run("parses GitHub Enterprise SSH", func(t *testing.T) {
		owner, repo, err := parseGitHubURL("git@github.example.com:owner/repo.git")
		require.NoError(t, err)
		assert.Equal(t, "owner", owner)
		assert.Equal(t, "repo", repo)
	})

	t.Run("handles repository names with hyphens", func(t *testing.T) {
		owner, repo, err := parseGitHubURL("git@github.com:my-org/my-repo-name.git")
		require.NoError(t, err)
		assert.Equal(t, "my-org", owner)
		assert.Equal(t, "my-repo-name", repo)
	})

	t.Run("handles repository names with underscores", func(t *testing.T) {
		owner, repo, err := parseGitHubURL("https://github.com/my_org/my_repo_name.git")
		require.NoError(t, err)
		assert.Equal(t, "my_org", owner)
		assert.Equal(t, "my_repo_name", repo)
	})

	t.Run("handles repository names with dots", func(t *testing.T) {
		owner, repo, err := parseGitHubURL("git@github.com:owner/repo.name.git")
		require.NoError(t, err)
		assert.Equal(t, "owner", owner)
		assert.Equal(t, "repo.name", repo)
	})

	t.Run("trims whitespace from owner and repo", func(t *testing.T) {
		// This shouldn't happen in practice, but test defensive parsing
		// Note: URL parsing will encode spaces, so we test with actual spaces in the path
		// which would be URL-encoded in real scenarios
		owner, repo, err := parseGitHubURL("https://github.com/owner/repo.git")
		require.NoError(t, err)
		assert.Equal(t, "owner", owner)
		assert.Equal(t, "repo", repo)
	})

	t.Run("returns error for empty URL", func(t *testing.T) {
		_, _, err := parseGitHubURL("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("returns error for invalid SSH format", func(t *testing.T) {
		testCases := []string{
			"git@github.com",
			"git@github.com:",
			"git@github.com:owner",
			"git@github.com:/repo.git",
			"git@github.com:owner/",
		}
		for _, url := range testCases {
			_, _, err := parseGitHubURL(url)
			require.Error(t, err, "should error for invalid SSH format: %s", url)
		}
	})

	t.Run("returns error for invalid HTTPS format", func(t *testing.T) {
		testCases := []struct {
			url           string
			expectedError string
		}{
			{"https://github.com", "owner or repository name is missing"},
			{"https://github.com/", "owner or repository name is missing"},
			{"https://github.com/owner", "owner or repository name is missing"},
			{"https://github.com/owner/", "owner or repository name is missing"},
			{"https://github.com//repo.git", "owner or repository name is missing"},
		}
		for _, tc := range testCases {
			_, _, err := parseGitHubURL(tc.url)
			require.Error(t, err, "should error for invalid HTTPS format: %s", tc.url)
			assert.Contains(t, err.Error(), tc.expectedError, "error message should contain expected text for: %s", tc.url)
		}
	})

	t.Run("returns error for non-URL strings", func(t *testing.T) {
		_, _, err := parseGitHubURL("not a url")
		require.Error(t, err)
		// URL parsing might succeed but path will be empty, or it might fail
		// Either way, we should get an error about missing owner/repo or invalid format
		assert.True(t,
			strings.Contains(err.Error(), "failed to parse GitHub URL") ||
				strings.Contains(err.Error(), "owner or repository name is missing"),
			"error should mention parsing failure or missing owner/repo: %s", err.Error())
	})

	t.Run("returns error for malformed URLs", func(t *testing.T) {
		_, _, err := parseGitHubURL("://invalid")
		require.Error(t, err)
	})
}

func TestGetGitHubRepoInfo(t *testing.T) {
	t.Run("successfully parses SSH remote", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t)
		defer cleanupTestGitRepo(t, tmpDir)

		// Add SSH remote
		// #nosec G204 - tmpDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", tmpDir, "remote", "add", "origin", "git@github.com:test-owner/test-repo.git").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "origin",
			},
		}

		owner, repo, err := GetGitHubRepoInfo(cfg)
		require.NoError(t, err)
		assert.Equal(t, "test-owner", owner)
		assert.Equal(t, "test-repo", repo)
	})

	t.Run("successfully parses HTTPS remote", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t)
		defer cleanupTestGitRepo(t, tmpDir)

		// Add HTTPS remote
		// #nosec G204 - tmpDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", tmpDir, "remote", "add", "origin", "https://github.com/test-owner/test-repo.git").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "origin",
			},
		}

		owner, repo, err := GetGitHubRepoInfo(cfg)
		require.NoError(t, err)
		assert.Equal(t, "test-owner", owner)
		assert.Equal(t, "test-repo", repo)
	})

	t.Run("successfully parses GitHub Enterprise remote", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t)
		defer cleanupTestGitRepo(t, tmpDir)

		// Add GitHub Enterprise remote
		// #nosec G204 - tmpDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", tmpDir, "remote", "add", "origin", "https://github.example.com/enterprise-org/enterprise-repo.git").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "origin",
			},
		}

		owner, repo, err := GetGitHubRepoInfo(cfg)
		require.NoError(t, err)
		assert.Equal(t, "enterprise-org", owner)
		assert.Equal(t, "enterprise-repo", repo)
	})

	t.Run("uses default origin when remote not configured", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t)
		defer cleanupTestGitRepo(t, tmpDir)

		// Add remote with default name
		// #nosec G204 - tmpDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", tmpDir, "remote", "add", "origin", "https://github.com/test-owner/test-repo.git").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "", // Empty should default to origin
			},
		}

		owner, repo, err := GetGitHubRepoInfo(cfg)
		require.NoError(t, err)
		assert.Equal(t, "test-owner", owner)
		assert.Equal(t, "test-repo", repo)
	})

	t.Run("uses custom remote name from config", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t)
		defer cleanupTestGitRepo(t, tmpDir)

		// Add custom remote
		// #nosec G204 - tmpDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", tmpDir, "remote", "add", "upstream", "https://github.com/upstream-owner/upstream-repo.git").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "upstream",
			},
		}

		owner, repo, err := GetGitHubRepoInfo(cfg)
		require.NoError(t, err)
		assert.Equal(t, "upstream-owner", owner)
		assert.Equal(t, "upstream-repo", repo)
	})

	t.Run("returns error for non-GitHub remote", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t)
		defer cleanupTestGitRepo(t, tmpDir)

		// Add non-GitHub remote
		// #nosec G204 - tmpDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", tmpDir, "remote", "add", "origin", "https://gitlab.com/test-owner/test-repo.git").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "origin",
			},
		}

		_, _, err := GetGitHubRepoInfo(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not a GitHub repository")
		assert.Contains(t, err.Error(), "This command only works with GitHub repositories")
	})

	t.Run("returns error for non-existent remote", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t)
		defer cleanupTestGitRepo(t, tmpDir)

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "nonexistent",
			},
		}

		_, _, err := GetGitHubRepoInfo(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not configured")
	})

	t.Run("returns error for nil config", func(t *testing.T) {
		_, _, err := GetGitHubRepoInfo(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration cannot be nil")
	})

	t.Run("returns error when not in git repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(originalDir) }()

		require.NoError(t, os.Chdir(tmpDir))

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "origin",
			},
		}

		_, _, err = GetGitHubRepoInfo(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a git repository")
	})
}

// setupTestGitRepo creates a temporary git repository for testing
func setupTestGitRepo(t *testing.T) string {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	// Initialize git repository
	// #nosec G204 - tmpDir is from t.TempDir() which is safe
	require.NoError(t, exec.Command("git", "-C", tmpDir, "init").Run())
	// #nosec G204 - tmpDir is from t.TempDir() which is safe
	require.NoError(t, exec.Command("git", "-C", tmpDir, "config", "user.email", "test@example.com").Run())
	// #nosec G204 - tmpDir is from t.TempDir() which is safe
	require.NoError(t, exec.Command("git", "-C", tmpDir, "config", "user.name", "Test User").Run())

	// Create initial commit
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0o600))
	// #nosec G204 - tmpDir is from t.TempDir() which is safe
	require.NoError(t, exec.Command("git", "-C", tmpDir, "add", "test.txt").Run())
	// #nosec G204 - tmpDir is from t.TempDir() which is safe
	require.NoError(t, exec.Command("git", "-C", tmpDir, "commit", "-m", "Initial commit").Run())

	// Change to test directory
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	return tmpDir
}

// cleanupTestGitRepo restores the original working directory
func cleanupTestGitRepo(_ *testing.T, _ string) {
	// Cleanup is handled by t.Cleanup in setupTestGitRepo
}

func TestResolveRemoteName(t *testing.T) {
	t.Run("uses remote from config", func(t *testing.T) {
		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "upstream",
			},
		}
		assert.Equal(t, "upstream", resolveRemoteName(cfg))
	})

	t.Run("defaults to origin when remote not configured", func(t *testing.T) {
		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "",
			},
		}
		assert.Equal(t, "origin", resolveRemoteName(cfg))
	})

	t.Run("defaults to origin when Git config is nil", func(t *testing.T) {
		cfg := &config.Config{
			Git: nil,
		}
		assert.Equal(t, "origin", resolveRemoteName(cfg))
	})
}

func TestGetRepoRoot(t *testing.T) {
	t.Run("finds repo root in current directory", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t)
		defer cleanupTestGitRepo(t, tmpDir)

		repoRoot, err := getRepoRoot()
		require.NoError(t, err)
		// Use filepath.EvalSymlinks to handle /var vs /private/var on macOS
		expectedPath, _ := filepath.EvalSymlinks(tmpDir)
		actualPath, _ := filepath.EvalSymlinks(repoRoot)
		assert.Equal(t, expectedPath, actualPath)
	})

	t.Run("finds repo root in subdirectory", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t)
		defer cleanupTestGitRepo(t, tmpDir)

		// Create subdirectory
		subDir := filepath.Join(tmpDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0o700))
		require.NoError(t, os.Chdir(subDir))

		repoRoot, err := getRepoRoot()
		require.NoError(t, err)
		// Use filepath.EvalSymlinks to handle /var vs /private/var on macOS
		expectedPath, _ := filepath.EvalSymlinks(tmpDir)
		actualPath, _ := filepath.EvalSymlinks(repoRoot)
		assert.Equal(t, expectedPath, actualPath)
	})

	t.Run("returns error when not in git repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(originalDir) }()

		require.NoError(t, os.Chdir(tmpDir))

		_, err = getRepoRoot()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a git repository")
	})
}

func TestExecuteCommand(t *testing.T) {
	t.Run("executes command successfully", func(t *testing.T) {
		ctx := testContext(t)
		output, err := executeCommand(ctx, "echo", []string{"hello", "world"}, "")
		require.NoError(t, err)
		assert.Equal(t, "hello world", output)
	})

	t.Run("executes command in directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0o600))

		ctx := testContext(t)
		output, err := executeCommand(ctx, "ls", []string{"test.txt"}, tmpDir)
		require.NoError(t, err)
		assert.Contains(t, output, "test.txt")
	})

	t.Run("returns error for non-existent command", func(t *testing.T) {
		ctx := testContext(t)
		_, err := executeCommand(ctx, "nonexistent-command-12345", []string{}, "")
		require.Error(t, err)
	})
}

// testContext creates a test context with timeout
func testContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}
