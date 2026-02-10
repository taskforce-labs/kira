package commands

import (
	"context"
	"os"
	"os/exec"
	"strings"
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

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		workItemType, id, title, currentStatus, repos, err := extractWorkItemMetadata(testFilePath, cfg)
		require.NoError(t, err)
		assert.Equal(t, "prd", workItemType)
		assert.Equal(t, "001", id)
		assert.Equal(t, "Test Feature", title)
		assert.Equal(t, "todo", currentStatus)
		assert.Nil(t, repos)
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

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		workItemType, id, title, currentStatus, repos, err := extractWorkItemMetadata(testFilePath, cfg)
		require.NoError(t, err)
		assert.Equal(t, "unknown", workItemType)
		assert.Equal(t, "001", id)
		assert.Equal(t, "unknown", title)
		assert.Equal(t, "unknown", currentStatus)
		assert.Nil(t, repos)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		_, _, _, _, _, err = extractWorkItemMetadata(".work/nonexistent.md", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read work item file")
	})

	t.Run("extracts repos from front matter when present", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 010
title: Draft PR feature
status: doing
kind: prd
repos:
  - frontend
  - backend
---
# Draft PR
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(workItemContent), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		workItemType, id, title, currentStatus, repos, err := extractWorkItemMetadata(testFilePath, cfg)
		require.NoError(t, err)
		assert.Equal(t, "prd", workItemType)
		assert.Equal(t, "010", id)
		assert.Equal(t, "Draft PR feature", title)
		assert.Equal(t, "doing", currentStatus)
		require.Len(t, repos, 2)
		assert.Equal(t, []string{"frontend", "backend"}, repos)
	})

	t.Run("repos nil when absent from front matter", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		_, _, _, _, repos, err := extractWorkItemMetadata(testFilePath, cfg)
		require.NoError(t, err)
		assert.Nil(t, repos)
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

// verifyStagedMove verifies that both deletion and addition are staged (Git may show as D/A or R for rename)
func verifyStagedMove(t *testing.T, oldPath, newPath string) {
	cmd := exec.Command("git", "diff", "--cached", "--name-status")
	output, err := cmd.Output()
	require.NoError(t, err)
	outputStr := string(output)
	// Git may show as deletion+addition (D/A) or rename (R)
	assert.True(t, strings.Contains(outputStr, "D\t"+oldPath) || (strings.Contains(outputStr, "R") && strings.Contains(outputStr, oldPath)),
		"Deletion should be staged. Output: %s", outputStr)
	assert.True(t, strings.Contains(outputStr, "A\t"+newPath) || (strings.Contains(outputStr, "R") && strings.Contains(outputStr, newPath)),
		"Addition should be staged. Output: %s", outputStr)
}

func TestStageFileChanges(t *testing.T) {
	t.Run("stages deletion and addition when git rm --cached succeeds", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create and commit a file
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))
		require.NoError(t, exec.Command("git", "add", testFilePath).Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Move file on disk (simulating moveWorkItem)
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.Rename(testFilePath, testTargetPath))

		// Stage the changes
		ctx := context.Background()
		err := stageFileChanges(ctx, testFilePath, testTargetPath, false)
		require.NoError(t, err)

		// Verify deletion is staged (Git may show as D or R for rename)
		cmd := exec.Command("git", "diff", "--cached", "--name-status")
		output, err := cmd.Output()
		require.NoError(t, err)
		outputStr := string(output)
		// Git may show as deletion+addition (D/A) or rename (R)
		assert.True(t, strings.Contains(outputStr, "D\t"+testFilePath) || (strings.Contains(outputStr, "R") && strings.Contains(outputStr, testFilePath)),
			"Deletion should be staged. Output: %s", outputStr)
		assert.True(t, strings.Contains(outputStr, "A\t"+testTargetPath) || (strings.Contains(outputStr, "R") && strings.Contains(outputStr, testTargetPath)),
			"Addition should be staged. Output: %s", outputStr)
	})

	t.Run("stages deletion using git add -u when git rm --cached fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create and commit a file
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))
		require.NoError(t, exec.Command("git", "add", testFilePath).Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Move file on disk
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.Rename(testFilePath, testTargetPath))

		// Manually remove from git index to simulate git rm --cached failure scenario
		// This simulates a case where the file was already removed from index
		require.NoError(t, exec.Command("git", "rm", "--cached", testFilePath).Run())
		require.NoError(t, exec.Command("git", "reset", "HEAD").Run()) // Unstage the deletion

		// Now stageFileChanges should use git add -u fallback
		ctx := context.Background()
		err := stageFileChanges(ctx, testFilePath, testTargetPath, false)
		require.NoError(t, err)

		// Verify deletion is staged (Git may show as D or R for rename)
		cmd := exec.Command("git", "diff", "--cached", "--name-status")
		output, err := cmd.Output()
		require.NoError(t, err)
		outputStr := string(output)
		// Git may show as deletion+addition (D/A) or rename (R)
		assert.True(t, strings.Contains(outputStr, "D\t"+testFilePath) || (strings.Contains(outputStr, "R") && strings.Contains(outputStr, testFilePath)),
			"Deletion should be staged. Output: %s", outputStr)
		assert.True(t, strings.Contains(outputStr, "A\t"+testTargetPath) || (strings.Contains(outputStr, "R") && strings.Contains(outputStr, testTargetPath)),
			"Addition should be staged. Output: %s", outputStr)
	})

	t.Run("returns error when both git rm --cached and git add -u fail", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create file but don't commit it (file doesn't exist in git)
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))

		// Move file on disk
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.Rename(testFilePath, testTargetPath))

		// Try to stage changes - should fail because file was never tracked
		ctx := context.Background()
		err := stageFileChanges(ctx, testFilePath, testTargetPath, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to stage deletion")
	})

	t.Run("verifies deletion is staged before proceeding", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create and commit a file
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))
		require.NoError(t, exec.Command("git", "add", testFilePath).Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Move file on disk
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.Rename(testFilePath, testTargetPath))

		// Stage the changes
		ctx := context.Background()
		err := stageFileChanges(ctx, testFilePath, testTargetPath, false)
		require.NoError(t, err)

		// Verify that both deletion and addition are staged
		verifyStagedMove(t, testFilePath, testTargetPath)
	})

	t.Run("dry run does not stage changes", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repo
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create and commit a file
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))
		require.NoError(t, exec.Command("git", "add", testFilePath).Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Move file on disk
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.Rename(testFilePath, testTargetPath))

		// Stage with dry run
		ctx := context.Background()
		err := stageFileChanges(ctx, testFilePath, testTargetPath, true)
		require.NoError(t, err)

		// Verify nothing is staged
		cmd := exec.Command("git", "diff", "--cached", "--name-only")
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Empty(t, strings.TrimSpace(string(output)))
	})
}

func TestCommitMove(t *testing.T) {
	t.Run("creates single commit with both deletion and addition", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

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

		// Create and commit work item
		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))
		require.NoError(t, exec.Command("git", "add", testFilePath).Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Add work item").Run())

		// Move file on disk (simulating moveWorkItemWithoutCommit)
		require.NoError(t, os.Rename(testFilePath, testTargetPath))

		// Update status in file
		content, err := os.ReadFile(testTargetPath)
		require.NoError(t, err)
		contentStr := strings.ReplaceAll(string(content), "status: todo", "status: doing")
		require.NoError(t, os.WriteFile(testTargetPath, []byte(contentStr), 0o600))

		// Commit the move
		err = commitMove(testFilePath, testTargetPath, "Move prd 001 to doing", "Test Feature (todo -> doing)", false)
		require.NoError(t, err)

		// Verify commit was created
		cmd := exec.Command("git", "log", "--oneline", "-1")
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Contains(t, string(output), "Move prd 001 to doing")

		// Verify commit contains both deletion and addition (Git may show as D/A or R for rename)
		cmd = exec.Command("git", "show", "--name-status", "--pretty=format:", "HEAD")
		output, err = cmd.Output()
		require.NoError(t, err)
		outputStr := string(output)
		lines := strings.Split(outputStr, "\n")
		foundDeletion := false
		foundAddition := false
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if (strings.HasPrefix(line, "D\t") && strings.Contains(line, testFilePath)) ||
				(strings.HasPrefix(line, "R") && strings.Contains(line, testFilePath)) {
				foundDeletion = true
			}
			if (strings.HasPrefix(line, "A\t") && strings.Contains(line, testTargetPath)) ||
				(strings.HasPrefix(line, "R") && strings.Contains(line, testTargetPath)) {
				foundAddition = true
			}
		}
		assert.True(t, foundDeletion, "Commit should contain deletion of %s. Output: %s", testFilePath, outputStr)
		assert.True(t, foundAddition, "Commit should contain addition of %s. Output: %s", testTargetPath, outputStr)
	})
}
