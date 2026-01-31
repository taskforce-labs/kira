package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kira/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindWorkItemFile(t *testing.T) {
	t.Run("finds work item by ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		// Create a work item file (using testWorkItemContent constant from move_test.go)
		filePath := ".work/1_todo/001-test-feature.prd.md"
		require.NoError(t, os.WriteFile(filePath, []byte(testWorkItemContent), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// Find the work item
		foundPath, err := findWorkItemFile("001", cfg)
		require.NoError(t, err)
		assert.Equal(t, filePath, foundPath)
	})

	t.Run("returns error when work item not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// Try to find non-existent work item
		_, err = findWorkItemFile("999", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "work item with ID 999 not found")
	})
}

func TestResolveSliceWorkItem(t *testing.T) {
	workItemContentDoing := `---
id: 001
title: Test Feature
status: doing
kind: prd
---
# Test
`
	t.Run("resolves by work item ID when provided", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.WriteFile(".work/1_todo/001-test.prd.md", []byte(testWorkItemContent), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		path, err := resolveSliceWorkItem("001", cfg)
		require.NoError(t, err)
		assert.Equal(t, ".work/1_todo/001-test.prd.md", path)
	})

	t.Run("returns error when no work item in doing folder", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		_, err = resolveSliceWorkItem("", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no work item in doing folder")
	})

	t.Run("returns path when exactly one work item in doing folder", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(".work/2_doing/001-test.prd.md", []byte(workItemContentDoing), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		path, err := resolveSliceWorkItem("", cfg)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(".work", "2_doing", "001-test.prd.md"), path)
	})

	t.Run("returns error when multiple work items in doing folder", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(".work/2_doing/001-a.prd.md", []byte(workItemContentDoing), 0o600))
		content2 := strings.Replace(workItemContentDoing, "id: 001", "id: 002", 1)
		require.NoError(t, os.WriteFile(".work/2_doing/002-b.prd.md", []byte(content2), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		_, err = resolveSliceWorkItem("", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "multiple work items in doing folder")
	})
}

func TestUpdateWorkItemStatus(t *testing.T) {
	t.Run("updates status in work item", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create a work item file
		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
---

# Test Feature
`
		// Create .work directory and file
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		filePath := ".work/1_todo/test-work-item.md"
		require.NoError(t, os.WriteFile(filePath, []byte(workItemContent), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// Update status
		err = updateWorkItemStatus(filePath, "doing", cfg)
		require.NoError(t, err)

		// Check that status was updated
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)

		assert.Contains(t, string(content), "status: doing")
		assert.NotContains(t, string(content), "status: todo")
	})
}

func TestGetWorkItemFiles(t *testing.T) {
	t.Run("finds all work item files in directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create directory structure
		require.NoError(t, os.MkdirAll("test-dir", 0o700))

		// Create work item files
		workItem1 := `---
id: 001
title: Test Feature 1
---
# Test Feature 1
`
		workItem2 := `---
id: 002
title: Test Feature 2
---
# Test Feature 2
`
		require.NoError(t, os.WriteFile("test-dir/001-feature1.md", []byte(workItem1), 0o600))
		require.NoError(t, os.WriteFile("test-dir/002-feature2.md", []byte(workItem2), 0o600))
		require.NoError(t, os.WriteFile("test-dir/not-a-work-item.txt", []byte("not a work item"), 0o600))

		// Get work item files
		files, err := getWorkItemFiles("test-dir")
		require.NoError(t, err)

		assert.Len(t, files, 2)
		assert.Contains(t, files, "test-dir/001-feature1.md")
		assert.Contains(t, files, "test-dir/002-feature2.md")
		assert.NotContains(t, files, "test-dir/not-a-work-item.txt")
	})
}

func TestArchiveWorkItems(t *testing.T) {
	t.Run("archives work items to archive directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work", 0o700))

		// Create work item files
		workItem1 := `---
id: 001
title: Test Feature 1
---
# Test Feature 1
`
		workItem2 := `---
id: 002
title: Test Feature 2
---
# Test Feature 2
`
		require.NoError(t, os.WriteFile(".work/work-item1.md", []byte(workItem1), 0o600))
		require.NoError(t, os.WriteFile(".work/work-item2.md", []byte(workItem2), 0o600))

		workItems := []string{".work/work-item1.md", ".work/work-item2.md"}

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// Archive work items
		archivePath, err := archiveWorkItems(workItems, "source-dir", cfg)
		require.NoError(t, err)

		// Check that archive directory was created
		assert.DirExists(t, archivePath)

		// Check that work items were copied to archive
		archivedFile1 := filepath.Join(archivePath, "work-item1.md")
		archivedFile2 := filepath.Join(archivePath, "work-item2.md")

		assert.FileExists(t, archivedFile1)
		assert.FileExists(t, archivedFile2)

		// Check that content was preserved
		content1, err := safeReadFile(archivedFile1, cfg)
		require.NoError(t, err)
		assert.Contains(t, string(content1), "Test Feature 1")

		content2, err := safeReadFile(archivedFile2, cfg)
		require.NoError(t, err)
		assert.Contains(t, string(content2), "Test Feature 2")
	})
}

func TestFormatCommandPreview(t *testing.T) {
	t.Run("formats command with no args", func(t *testing.T) {
		result := formatCommandPreview("git", []string{})
		assert.Equal(t, "[DRY RUN] git", result)
	})

	t.Run("formats command with single arg", func(t *testing.T) {
		result := formatCommandPreview("git", []string{"status"})
		assert.Equal(t, "[DRY RUN] git status", result)
	})

	t.Run("formats command with multiple args", func(t *testing.T) {
		result := formatCommandPreview("git", []string{"commit", "-m", "test message"})
		assert.Equal(t, "[DRY RUN] git commit -m test message", result)
	})
}

func TestExecuteCommand(t *testing.T) {
	t.Run("dry run returns empty string and prints preview", func(t *testing.T) {
		ctx := context.Background()

		// Capture stdout
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w

		output, err := executeCommand(ctx, "echo", []string{"hello"}, "", true)

		// Restore stdout and read captured output
		require.NoError(t, w.Close())
		os.Stdout = oldStdout
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		captured := buf.String()

		require.NoError(t, err)
		assert.Empty(t, output)
		assert.Contains(t, captured, "[DRY RUN] echo hello")
	})

	t.Run("dry run with directory shows directory in preview", func(t *testing.T) {
		ctx := context.Background()
		tmpDir := t.TempDir()

		// Capture stdout
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w

		output, err := executeCommand(ctx, "ls", []string{"-la"}, tmpDir, true)

		// Restore stdout and read captured output
		require.NoError(t, w.Close())
		os.Stdout = oldStdout
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		captured := buf.String()

		require.NoError(t, err)
		assert.Empty(t, output)
		assert.Contains(t, captured, "[DRY RUN] ls -la")
		assert.Contains(t, captured, tmpDir)
	})

	t.Run("successful command execution returns stdout", func(t *testing.T) {
		ctx := context.Background()

		output, err := executeCommand(ctx, "echo", []string{"hello world"}, "", false)
		require.NoError(t, err)
		assert.Equal(t, "hello world\n", output)
	})

	t.Run("command execution respects working directory", func(t *testing.T) {
		ctx := context.Background()
		tmpDir := t.TempDir()

		// Create a test file in the temp directory
		testFile := filepath.Join(tmpDir, "testfile.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0o600))

		// Run ls in the temp directory
		output, err := executeCommand(ctx, "ls", []string{}, tmpDir, false)
		require.NoError(t, err)
		assert.Contains(t, output, "testfile.txt")
	})

	t.Run("failed command returns error with stderr", func(t *testing.T) {
		ctx := context.Background()

		// Run a command that will fail and produce stderr
		_, err := executeCommand(ctx, "ls", []string{"nonexistent-directory-12345"}, "", false)
		require.Error(t, err)
		// Error message should contain info about the failure
		assert.Contains(t, err.Error(), "exit status")
	})

	t.Run("context cancellation stops command", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		// Run a command that takes longer than the timeout
		_, err := executeCommand(ctx, "sleep", []string{"10"}, "", false)
		require.Error(t, err)
	})
}

func TestExecuteCommandCombinedOutput(t *testing.T) {
	t.Run("dry run returns empty string and prints preview", func(t *testing.T) {
		ctx := context.Background()

		// Capture stdout
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w

		output, err := executeCommandCombinedOutput(ctx, "echo", []string{"hello"}, "", true)

		// Restore stdout and read captured output
		require.NoError(t, w.Close())
		os.Stdout = oldStdout
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		captured := buf.String()

		require.NoError(t, err)
		assert.Empty(t, output)
		assert.Contains(t, captured, "[DRY RUN] echo hello")
	})

	t.Run("successful command execution returns combined output", func(t *testing.T) {
		ctx := context.Background()

		output, err := executeCommandCombinedOutput(ctx, "echo", []string{"hello world"}, "", false)
		require.NoError(t, err)
		assert.Equal(t, "hello world\n", output)
	})

	t.Run("failed command includes output in error", func(t *testing.T) {
		ctx := context.Background()

		// Run a command that will fail
		_, err := executeCommandCombinedOutput(ctx, "ls", []string{"nonexistent-directory-12345"}, "", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exit status")
	})
}
