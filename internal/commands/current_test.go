package commands

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"kira/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindWorkItemFileInAllStatusFolders(t *testing.T) {
	t.Run("finds work item across all status folders", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory structure with multiple status folders
		require.NoError(t, os.MkdirAll(".work/0_backlog", 0o700))
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.MkdirAll(".work/4_done", 0o700))

		// Create work item in doing folder
		require.NoError(t, os.WriteFile(testTargetPath, []byte(testWorkItemContent), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// Find the work item
		foundPath, err := findWorkItemFileInAllStatusFolders("001", cfg)
		require.NoError(t, err)
		assert.Equal(t, testTargetPath, foundPath)
	})

	t.Run("finds work item in backlog folder", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/0_backlog", 0o700))
		filePath := ".work/0_backlog/002-backlog-item.prd.md"
		content := strings.Replace(testWorkItemContent, "id: 001", "id: 002", 1)
		require.NoError(t, os.WriteFile(filePath, []byte(content), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		foundPath, err := findWorkItemFileInAllStatusFolders("002", cfg)
		require.NoError(t, err)
		assert.Equal(t, filePath, foundPath)
	})

	t.Run("returns error when work item not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		_, err = findWorkItemFileInAllStatusFolders("999", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "work item with ID 999 not found")
	})
}

func TestRunCurrentTitle(t *testing.T) {
	t.Run("outputs PR title for valid branch", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo with initial commit
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "commit", "--allow-empty", "-m", "initial").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "001-test-feature").Run())

		// Create work item
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(testTargetPath, []byte(testWorkItemContent), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// Capture stdout
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w

		err = runCurrentTitle(cfg)

		// Restore stdout and read captured output
		require.NoError(t, w.Close())
		os.Stdout = oldStdout
		var buf strings.Builder
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		require.NoError(t, err)
		assert.Equal(t, "001: Test Feature", strings.TrimSpace(output))
	})

	t.Run("exits with non-zero for invalid branch name", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo with invalid branch name (main doesn't match kira format)
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "commit", "--allow-empty", "-m", "initial").Run())
		// Stay on default branch (main/master) which doesn't match kira format

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// This should exit with non-zero, but we can't easily test os.Exit in unit tests
		// So we'll test the underlying function instead
		repoRoot, err := getRepoRoot()
		require.NoError(t, err)
		currentBranch, err := getCurrentBranch(repoRoot)
		require.NoError(t, err)
		// Branch name "main" doesn't match kira format {id}-{kebab-title}
		_, err = parseWorkItemIDFromBranch(currentBranch, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match kira branch format")
	})

	t.Run("exits with non-zero when work item not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "commit", "--allow-empty", "-m", "initial").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "001-test-feature").Run())

		// Create .work directory but don't create work item file
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// This should exit with non-zero, but we can't easily test os.Exit in unit tests
		// So we'll test the underlying function instead
		repoRoot, err := getRepoRoot()
		require.NoError(t, err)
		currentBranch, err := getCurrentBranch(repoRoot)
		require.NoError(t, err)
		workItemID, err := parseWorkItemIDFromBranch(currentBranch, cfg)
		require.NoError(t, err)
		_, err = findWorkItemFileInAllStatusFolders(workItemID, cfg)
		require.Error(t, err)
	})
}

func TestRunCurrentBody(t *testing.T) {
	t.Run("outputs work item body without frontmatter", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "commit", "--allow-empty", "-m", "initial").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "001-test-feature").Run())

		// Create work item
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		workItemContent := `---
id: 001
title: Test Feature
status: doing
kind: prd
---

# Test Feature

This is the body content.
`
		require.NoError(t, os.WriteFile(testTargetPath, []byte(workItemContent), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// Capture stdout
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w

		err = runCurrentBody(cfg)

		// Restore stdout and read captured output
		require.NoError(t, w.Close())
		os.Stdout = oldStdout
		var buf strings.Builder
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		require.NoError(t, err)
		// Body should NOT contain YAML frontmatter
		assert.NotContains(t, output, "---")
		assert.NotContains(t, output, "id: 001")
		assert.NotContains(t, output, "title: Test Feature")
		assert.NotContains(t, output, "status: doing")
		// Body should contain only the markdown content
		assert.Contains(t, output, "# Test Feature")
		assert.Contains(t, output, "This is the body content.")
	})
}

func TestRunCurrentSlug(t *testing.T) {
	t.Run("outputs slug for valid branch", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "commit", "--allow-empty", "-m", "initial").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "001-test-feature").Run())

		// Create work item
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(testTargetPath, []byte(testWorkItemContent), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// Capture stdout
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w

		err = runCurrentSlug(cfg)

		// Restore stdout and read captured output
		require.NoError(t, w.Close())
		os.Stdout = oldStdout
		var buf strings.Builder
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		require.NoError(t, err)
		assert.Equal(t, "001-test-feature", strings.TrimSpace(output))
	})

	t.Run("exits with non-zero for invalid branch name", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo with invalid branch name
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "commit", "--allow-empty", "-m", "initial").Run())
		// Stay on default branch (main/master) which doesn't match kira format

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// This should exit with non-zero, but we can't easily test os.Exit in unit tests
		// So we'll test the underlying function instead
		repoRoot, err := getRepoRoot()
		require.NoError(t, err)
		currentBranch, err := getCurrentBranch(repoRoot)
		require.NoError(t, err)
		_, err = parseWorkItemIDFromBranch(currentBranch, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match kira branch format")
	})

	t.Run("exits with non-zero when branch has no slug", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo with branch that has ID but no slug
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "commit", "--allow-empty", "-m", "initial").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "001").Run())

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// This should exit with non-zero because parseWorkItemIDFromBranch validates
		// that branch name has format {id}-{kebab-title}, so "001" (without slug) fails
		repoRoot, err := getRepoRoot()
		require.NoError(t, err)
		currentBranch, err := getCurrentBranch(repoRoot)
		require.NoError(t, err)
		_, err = parseWorkItemIDFromBranch(currentBranch, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match kira branch format")
	})
}

func TestRunCurrentPRs(t *testing.T) {
	t.Run("outputs empty array when not a git repo", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory (required by checkWorkDir)
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		// Don't initialize git repo

		// Capture stdout
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w

		err = runCurrentPRs(nil, nil)

		// Restore stdout and read captured output
		require.NoError(t, w.Close())
		os.Stdout = oldStdout
		var buf strings.Builder
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		require.NoError(t, err)
		assert.Equal(t, "[]\n", output)
	})

	t.Run("outputs empty array for invalid branch name", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo with invalid branch name (main doesn't match kira format)
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "commit", "--allow-empty", "-m", "initial").Run())
		// Stay on default branch (main/master) which doesn't match kira format

		// Create .work directory
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		// Capture stdout
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w

		err = runCurrentPRs(nil, nil)

		// Restore stdout and read captured output
		require.NoError(t, w.Close())
		os.Stdout = oldStdout
		var buf strings.Builder
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		require.NoError(t, err)
		assert.Equal(t, "[]\n", output)
	})

	t.Run("outputs empty array when work item not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "commit", "--allow-empty", "-m", "initial").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "001-test-feature").Run())

		// Create .work directory but don't create work item file
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		// Capture stdout
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w

		err = runCurrentPRs(nil, nil)

		// Restore stdout and read captured output
		require.NoError(t, w.Close())
		os.Stdout = oldStdout
		var buf strings.Builder
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		require.NoError(t, err)
		assert.Equal(t, "[]\n", output)
	})
}
