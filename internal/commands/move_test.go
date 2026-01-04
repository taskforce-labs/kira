package commands

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kira/internal/config"
)

const (
	testWorkItemContent = `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
---

# Test Feature
`
	testFilePath   = ".work/1_todo/001-test-feature.prd.md"
	testTargetPath = ".work/2_doing/001-test-feature.prd.md"
)

func TestExtractWorkItemMetadata(t *testing.T) {
	t.Run("extracts all metadata fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))

		workItemType, id, title, currentStatus, err := extractWorkItemMetadata(testFilePath)
		require.NoError(t, err)
		assert.Equal(t, "prd", workItemType)
		assert.Equal(t, "001", id)
		assert.Equal(t, "Test Feature", title)
		assert.Equal(t, "todo", currentStatus)
	})

	t.Run("handles missing fields gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
---

# Test Feature
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(workItemContent), 0o600))

		workItemType, id, title, currentStatus, err := extractWorkItemMetadata(testFilePath)
		require.NoError(t, err)
		assert.Equal(t, "unknown", workItemType)
		assert.Equal(t, "001", id)
		assert.Equal(t, "unknown", title)
		assert.Equal(t, "unknown", currentStatus)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		_, _, _, _, err := extractWorkItemMetadata(".work/nonexistent.md")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read work item file")
	})
}

func TestBuildCommitMessage(t *testing.T) {
	t.Run("builds message with default templates", func(t *testing.T) {
		cfg := &config.Config{
			Commit: config.CommitConfig{
				MoveSubjectTemplate: "",
				MoveBodyTemplate:    "",
			},
		}

		subject, body, err := buildCommitMessage(cfg, "prd", "001", "Test Feature", "todo", "doing")
		require.NoError(t, err)
		assert.Equal(t, "Move prd 001 to doing", subject)
		assert.Equal(t, "Test Feature (todo -> doing)", body)
	})

	t.Run("builds message with custom templates", func(t *testing.T) {
		cfg := &config.Config{
			Commit: config.CommitConfig{
				MoveSubjectTemplate: "Custom: {type} {id} -> {target_status}",
				MoveBodyTemplate:    "{title} moved from {current_status}",
			},
		}

		subject, body, err := buildCommitMessage(cfg, "issue", "002", "Fix Bug", "doing", "review")
		require.NoError(t, err)
		assert.Equal(t, "Custom: issue 002 -> review", subject)
		assert.Equal(t, "Fix Bug moved from doing", body)
	})

	t.Run("replaces all template variables", func(t *testing.T) {
		cfg := &config.Config{
			Commit: config.CommitConfig{
				MoveSubjectTemplate: "{type} {id} {title} {current_status} {target_status}",
				MoveBodyTemplate:    "{type} {id} {title} {current_status} {target_status}",
			},
		}

		subject, body, err := buildCommitMessage(cfg, "task", "003", "Do Something", "backlog", "todo")
		require.NoError(t, err)
		assert.Equal(t, "task 003 Do Something backlog todo", subject)
		assert.Equal(t, "task 003 Do Something backlog todo", body)
	})
}

func TestCheckStagedChanges(t *testing.T) {
	t.Run("returns false when no staged changes", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		hasStaged, err := checkStagedChanges([]string{})
		require.NoError(t, err)
		assert.False(t, hasStaged)
	})

	t.Run("excludes specified paths", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create and stage a file
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())

		// Check excluding a different file
		hasStaged, err := checkStagedChanges([]string{"other.txt"})
		require.NoError(t, err)
		assert.True(t, hasStaged)

		// Check excluding the staged file
		hasStaged, err = checkStagedChanges([]string{"test.txt"})
		require.NoError(t, err)
		assert.False(t, hasStaged)
	})

	t.Run("returns error when git not available", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Don't initialize git repo

		_, err := checkStagedChanges([]string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git is not available")
	})
}

func TestMoveWorkItem(t *testing.T) {
	t.Run("moves work item without commit flag", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.DefaultConfig

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))

		err := moveWorkItem(cfg, "001", "doing", false, false)
		require.NoError(t, err)

		// Check file was moved
		_, err = os.Stat(testTargetPath)
		require.NoError(t, err)

		// Check old file doesn't exist
		_, err = os.Stat(testFilePath)
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("moves work item with commit flag", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.DefaultConfig

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create initial commit with a placeholder file
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(".work/.gitkeep", []byte(""), 0o600))
		require.NoError(t, exec.Command("git", "add", ".work/").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))
		require.NoError(t, exec.Command("git", "add", testFilePath).Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Add work item").Run())

		err := moveWorkItem(cfg, "001", "doing", true, false)
		require.NoError(t, err)

		// Check file was moved
		targetPath := ".work/2_doing/001-test-feature.prd.md"
		_, err = os.Stat(targetPath)
		require.NoError(t, err)

		// Check git commit was created
		cmd := exec.Command("git", "log", "--oneline", "-1")
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Contains(t, string(output), "Move prd 001 to doing")
	})

	t.Run("move succeeds even if commit fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.DefaultConfig

		// Create .work directory structure (no git repo)
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))

		err := moveWorkItem(cfg, "001", "doing", true, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit")

		// Check file was still moved
		targetPath := ".work/2_doing/001-test-feature.prd.md"
		_, err = os.Stat(targetPath)
		require.NoError(t, err)
	})

	t.Run("dry run does not move file", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.DefaultConfig

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))

		// Run with dry-run flag
		err := moveWorkItem(cfg, "001", "doing", false, true)
		require.NoError(t, err)

		// Check file was NOT moved - should still be at original location
		_, err = os.Stat(testFilePath)
		require.NoError(t, err)

		// Check file was NOT moved to target
		_, err = os.Stat(testTargetPath)
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("dry run with commit flag shows git commands", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.DefaultConfig

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))

		// Run with both commit and dry-run flags
		err := moveWorkItem(cfg, "001", "doing", true, true)
		require.NoError(t, err)

		// Check file was NOT moved
		_, err = os.Stat(testFilePath)
		require.NoError(t, err)
	})

	t.Run("dry run requires target status", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.DefaultConfig

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))

		// Run with dry-run but no target status
		err := moveWorkItem(cfg, "001", "", false, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target status must be provided when using --dry-run")
	})
}
