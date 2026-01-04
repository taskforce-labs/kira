package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kira/internal/config"
)

func TestSanitizeCommitMessage(t *testing.T) {
	t.Run("sanitizes valid message", func(t *testing.T) {
		msg, err := sanitizeCommitMessage("Update work items")
		require.NoError(t, err)
		assert.Equal(t, "Update work items", msg)
	})

	t.Run("removes newlines", func(t *testing.T) {
		msg, err := sanitizeCommitMessage("Line 1\nLine 2")
		require.NoError(t, err)
		assert.Equal(t, "Line 1 Line 2", msg)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		msg, err := sanitizeCommitMessage("  message  ")
		require.NoError(t, err)
		assert.Equal(t, "message", msg)
	})

	t.Run("rejects empty message", func(t *testing.T) {
		_, err := sanitizeCommitMessage("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("rejects message too long", func(t *testing.T) {
		longMsg := strings.Repeat("a", 1001)
		_, err := sanitizeCommitMessage(longMsg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too long")
	})

	t.Run("rejects dangerous characters", func(t *testing.T) {
		dangerous := []string{"`", "$", "{", "}", "[", "]", "|", "&", ";"}
		for _, char := range dangerous {
			_, err := sanitizeCommitMessage("test" + char + "message")
			require.Error(t, err, "should reject character: %s", char)
			assert.Contains(t, err.Error(), "invalid character")
		}
	})
}

func TestCheckExternalChanges(t *testing.T) {
	t.Run("returns false when no external changes", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create .work directory with changes
		require.NoError(t, os.MkdirAll(".work", 0o700))
		require.NoError(t, os.WriteFile(".work/test.md", []byte("test"), 0o600))

		hasExternal, err := checkExternalChanges()
		require.NoError(t, err)
		assert.False(t, hasExternal)
	})

	t.Run("returns false when git not available", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Don't initialize git repo
		hasExternal, err := checkExternalChanges()
		require.NoError(t, err)
		assert.False(t, hasExternal)
	})
}

func TestStageWorkChanges(t *testing.T) {
	t.Run("stages work directory changes", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create .work directory with a file
		require.NoError(t, os.MkdirAll(".work", 0o700))
		require.NoError(t, os.WriteFile(".work/test.md", []byte("test"), 0o600))

		// Stage changes
		err := stageWorkChanges(false)
		require.NoError(t, err)

		// Verify file is staged
		cmd := exec.Command("git", "diff", "--cached", "--name-only")
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Contains(t, string(output), ".work/test.md")
	})

	t.Run("dry run does not stage changes", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create .work directory with a file
		require.NoError(t, os.MkdirAll(".work", 0o700))
		require.NoError(t, os.WriteFile(".work/test.md", []byte("test"), 0o600))

		// Stage with dry run
		err := stageWorkChanges(true)
		require.NoError(t, err)

		// Verify file is NOT staged
		cmd := exec.Command("git", "diff", "--cached", "--name-only")
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.NotContains(t, string(output), ".work/test.md")
	})
}

func TestCommitChanges(t *testing.T) {
	t.Run("commits with valid message", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create and stage a file
		require.NoError(t, os.MkdirAll(".work", 0o700))
		require.NoError(t, os.WriteFile(".work/test.md", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", ".work/").Run())

		// Commit
		err := commitChanges("Test commit message", false)
		require.NoError(t, err)

		// Verify commit was created
		cmd := exec.Command("git", "log", "--oneline", "-1")
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Contains(t, string(output), "Test commit message")
	})

	t.Run("dry run does not create commit", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create and stage a file
		require.NoError(t, os.MkdirAll(".work", 0o700))
		require.NoError(t, os.WriteFile(".work/test.md", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", ".work/").Run())

		// Commit with dry run
		err := commitChanges("Test commit message", true)
		require.NoError(t, err)

		// Verify no commit was created (git log should fail since there are no commits)
		cmd := exec.Command("git", "log", "--oneline", "-1")
		err = cmd.Run()
		require.Error(t, err) // Should error because no commits exist
	})
}

func TestSaveWorkItems(t *testing.T) {
	t.Run("saves work items successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.DefaultConfig

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create .work directory structure with a valid work item
		// Using testWorkItemContent constant from move_test.go
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.WriteFile(".work/1_todo/001-test.prd.md", []byte(testWorkItemContent), 0o600))

		// Save work items
		err := saveWorkItems(cfg, "Save work items", false)
		require.NoError(t, err)

		// Verify commit was created
		cmd := exec.Command("git", "log", "--oneline", "-1")
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Contains(t, string(output), "Save work items")
	})

	t.Run("dry run does not modify files or commit", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.DefaultConfig

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create .work directory structure with a valid work item
		// Using testWorkItemContent constant from move_test.go
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		workItemPath := ".work/1_todo/001-test.prd.md"
		require.NoError(t, os.WriteFile(workItemPath, []byte(testWorkItemContent), 0o600))

		// Get original content
		// #nosec G304 - workItemPath is a hardcoded test path
		originalContent, err := os.ReadFile(workItemPath)
		require.NoError(t, err)

		// Save work items with dry run
		err = saveWorkItems(cfg, "Save work items", true)
		require.NoError(t, err)

		// Verify file was NOT modified (no updated timestamp added)
		// #nosec G304 - workItemPath is a hardcoded test path
		currentContent, err := os.ReadFile(workItemPath)
		require.NoError(t, err)
		assert.Equal(t, string(originalContent), string(currentContent))

		// Verify no commit was created
		cmd := exec.Command("git", "log", "--oneline", "-1")
		err = cmd.Run()
		require.Error(t, err) // Should error because no commits exist
	})
}

func TestUpdateFileTimestamp(t *testing.T) {
	t.Run("updates existing timestamp", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory and file
		require.NoError(t, os.MkdirAll(".work", 0o700))
		content := `---
id: 001
title: Test
created: 2024-01-01
updated: 2024-01-01T00:00:00Z
---

# Test
`
		filePath := filepath.Join(".work", "test.md")
		require.NoError(t, os.WriteFile(filePath, []byte(content), 0o600))

		// Update timestamp
		err := updateFileTimestamp(filePath, "2024-12-25T12:00:00Z")
		require.NoError(t, err)

		// Verify timestamp was updated
		// #nosec G304 - filePath is constructed from hardcoded components in test
		updatedContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Contains(t, string(updatedContent), "updated: 2024-12-25T12:00:00Z")
		assert.NotContains(t, string(updatedContent), "updated: 2024-01-01T00:00:00Z")
	})

	t.Run("adds timestamp after created field if missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory and file without updated field
		require.NoError(t, os.MkdirAll(".work", 0o700))
		content := `---
id: 001
title: Test
created: 2024-01-01
---

# Test
`
		filePath := filepath.Join(".work", "test.md")
		require.NoError(t, os.WriteFile(filePath, []byte(content), 0o600))

		// Update timestamp
		err := updateFileTimestamp(filePath, "2024-12-25T12:00:00Z")
		require.NoError(t, err)

		// Verify timestamp was added
		// #nosec G304 - filePath is constructed from hardcoded components in test
		updatedContent, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Contains(t, string(updatedContent), "updated: 2024-12-25T12:00:00Z")
	})
}
