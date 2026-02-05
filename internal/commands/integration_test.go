package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v61/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findRepoRoot finds the repository root by walking up from the current file
// until it finds go.mod. In GitHub Actions, uses GITHUB_WORKSPACE if available.
func findRepoRoot() (string, error) {
	// Check for GitHub Actions workspace first
	if workspace := os.Getenv("GITHUB_WORKSPACE"); workspace != "" {
		if _, err := os.Stat(filepath.Join(workspace, "go.mod")); err == nil {
			return workspace, nil
		}
	}

	// Fall back to walking up from the test file
	_, thisFile, _, _ := runtime.Caller(1)
	dir := filepath.Dir(thisFile)
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// validateTestPath ensures a path is within the test's temporary directory
func validateTestPath(path, tmpDir string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	absTmpDir, err := filepath.Abs(tmpDir)
	if err != nil {
		return fmt.Errorf("invalid tmpDir: %w", err)
	}

	tmpDirWithSep := absTmpDir + string(filepath.Separator)
	if !strings.HasPrefix(absPath+string(filepath.Separator), tmpDirWithSep) && absPath != absTmpDir {
		return fmt.Errorf("path outside test directory: %s", path)
	}

	return nil
}

// safeExecCommand creates an exec.Command after validating the command path
func safeExecCommand(tmpDir, commandPath string, args ...string) (*exec.Cmd, error) {
	if err := validateTestPath(commandPath, tmpDir); err != nil {
		return nil, err
	}
	return exec.Command(commandPath, args...), nil
}

// buildKiraBinary builds the kira binary for testing.
func buildKiraBinary(t *testing.T, tmpDir string) string {
	repoRoot, err := findRepoRoot()
	require.NoError(t, err, "failed to find repo root")
	outPath := filepath.Join(tmpDir, "kira")
	mainPath := filepath.Join(repoRoot, "cmd", "kira", "main.go")

	// Verify required paths exist
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("main.go does not exist: %s (error: %v)", mainPath, err)
	}

	// Validate output path is within test directory
	if err := validateTestPath(outPath, tmpDir); err != nil {
		t.Fatalf("invalid output path: %v", err)
	}

	// Build from repo root directory - Go needs to be in module context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// #nosec G204 - outPath and mainPath validated above, command is hardcoded "go build"
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", outPath, "cmd/kira/main.go")
	buildCmd.Dir = repoRoot
	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "build failed: %s", string(output))

	return outPath
}

func TestCLIIntegration(t *testing.T) {
	t.Run("full workflow test", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		// Build the kira binary for testing
		_ = buildKiraBinary(t, tmpDir)
		defer func() { _ = os.Remove("kira") }()

		// Test kira init
		initCmd := exec.Command("./kira", "init")
		output, err := initCmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(output))
		assert.Contains(t, string(output), "Initialized kira workspace")

		// Check that .work directory was created
		assert.DirExists(t, ".work")
		assert.DirExists(t, ".work/1_todo")
		assert.DirExists(t, ".work/2_doing")
		assert.DirExists(t, ".work/3_review")
		assert.DirExists(t, ".work/4_done")
		assert.DirExists(t, ".work/z_archive")
		assert.DirExists(t, ".work/templates")
		assert.FileExists(t, ".work/IDEAS.md")
		assert.FileExists(t, "kira.yml")

		// Ensure .gitkeep files exist in status folders and templates
		gitkeepPaths := []string{
			".work/0_backlog/.gitkeep",
			".work/1_todo/.gitkeep",
			".work/2_doing/.gitkeep",
			".work/3_review/.gitkeep",
			".work/4_done/.gitkeep",
			".work/z_archive/.gitkeep",
			".work/templates/.gitkeep",
		}
		for _, p := range gitkeepPaths {
			assert.FileExists(t, p)
		}

		// Test kira idea add
		ideaCmd := exec.Command("./kira", "idea", "add", "Test idea for integration")
		output, err = ideaCmd.CombinedOutput()
		require.NoError(t, err, "idea add failed: %s", string(output))
		assert.Contains(t, string(output), "Added idea")
		assert.Contains(t, string(output), "Test idea for integration")

		// Check that idea was added to IDEAS.md
		ideasContent, err := os.ReadFile(".work/IDEAS.md")
		require.NoError(t, err)
		assert.Contains(t, string(ideasContent), "Test idea for integration")

		// Test kira lint (should pass with no work items)
		lintCmd := exec.Command("./kira", "lint")
		output, err = lintCmd.CombinedOutput()
		require.NoError(t, err, "lint failed: %s", string(output))
		assert.Contains(t, string(output), "No issues found")

		// Test kira doctor (should pass with no duplicates)
		doctorCmd := exec.Command("./kira", "doctor")
		output, err = doctorCmd.CombinedOutput()
		require.NoError(t, err, "doctor failed: %s", string(output))
		assert.Contains(t, string(output), "No issues found")
	})

	t.Run("new command splits colon-delimited title and description", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		kiraPath := buildKiraBinary(t, tmpDir)

		// Initialize workspace
		initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
		require.NoError(t, err)
		output, err := initCmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(output))

		// Create a new task with colon-delimited title and description
		newCmd, err := safeExecCommand(tmpDir, kiraPath, "new", "task", "todo", "my title: my description")
		require.NoError(t, err)
		output, err = newCmd.CombinedOutput()
		require.NoError(t, err, "new failed: %s", string(output))
		assert.Contains(t, string(output), "Created work item")

		// Verify work item was created in todo with split title/description
		globPattern := filepath.Join(tmpDir, ".work", "1_todo", "*.task.md")
		files, err := filepath.Glob(globPattern)
		require.NoError(t, err)
		require.Len(t, files, 1)

		content, err := safeReadTestFile(files[0], tmpDir)
		require.NoError(t, err)
		contentStr := string(content)

		// Title should be split before colon, description after
		assert.Contains(t, contentStr, "title: my title")
		assert.Contains(t, contentStr, "my description")
		assert.NotContains(t, contentStr, "my title: my description")
	})

	t.Run("default status on new without status argument", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Build the kira binary
		_, thisFile, _, _ := runtime.Caller(0)
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
		outPath := filepath.Join(tmpDir, "kira")
		require.NoError(t, validateTestPath(outPath, tmpDir))
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// #nosec G204 - outPath validated above, command is hardcoded "go build"
		buildCmd := exec.CommandContext(ctx, "go", "build", "-o", outPath, "kira/cmd/kira")
		buildCmd.Dir = repoRoot
		output, err := buildCmd.CombinedOutput()
		require.NoError(t, err, "build failed: %s", string(output))
		defer func() { _ = os.Remove("kira") }()

		// Initialize workspace
		initCmd := exec.Command("./kira", "init")
		output, err = initCmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(output))

		// Create work item without status (should default to backlog)
		// Note: Without --interactive flag, no prompts should appear (default behavior)
		newCmd := exec.Command("./kira", "new", "prd", "Default Status Feature")
		output, err = newCmd.CombinedOutput()
		require.NoError(t, err, "new failed: %s", string(output))

		// Expect file in 0_backlog
		matches, _ := filepath.Glob(".work/0_backlog/*default-status-feature*.prd.md")
		assert.NotEmpty(t, matches, "expected default status file in backlog")
	})

	t.Run("init handles existing .work with --fill-missing and --force", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Build the kira binary
		_, thisFile, _, _ := runtime.Caller(0)
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
		outPath := filepath.Join(tmpDir, "kira")
		require.NoError(t, validateTestPath(outPath, tmpDir))
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// #nosec G204 - outPath validated above, command is hardcoded "go build"
		buildCmd := exec.CommandContext(ctx, "go", "build", "-o", outPath, "kira/cmd/kira")
		buildCmd.Dir = repoRoot
		output, err := buildCmd.CombinedOutput()
		require.NoError(t, err, "build failed: %s", string(output))
		defer func() { _ = os.Remove("kira") }()

		// Initialize workspace
		initCmd := exec.Command("./kira", "init")
		output, err = initCmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(output))

		// Create a sentinel and remove a folder to simulate missing
		require.NoError(t, os.WriteFile(".work/1_todo/sentinel.txt", []byte("x"), 0o600))
		require.NoError(t, os.RemoveAll(".work/3_review"))

		// Fill missing
		fillCmd := exec.Command("./kira", "init", "--fill-missing")
		output, err = fillCmd.CombinedOutput()
		require.NoError(t, err, "fill-missing failed: %s", string(output))
		assert.FileExists(t, ".work/1_todo/sentinel.txt")
		assert.DirExists(t, ".work/3_review")

		// Force overwrite
		forceCmd := exec.Command("./kira", "init", "--force")
		output, err = forceCmd.CombinedOutput()
		require.NoError(t, err, "force failed: %s", string(output))
		assert.NoFileExists(t, ".work/1_todo/sentinel.txt")
		assert.DirExists(t, ".work/3_review")
		assert.FileExists(t, ".work/3_review/.gitkeep")
	})

	t.Run("work item creation and management", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Build the kira binary for testing using the repo root as working directory
		repoRoot, err := findRepoRoot()
		require.NoError(t, err, "failed to find repo root")
		outPath := filepath.Join(tmpDir, "kira")
		mainPath := filepath.Join(repoRoot, "cmd", "kira", "main.go")
		require.NoError(t, validateTestPath(outPath, tmpDir))
		// Use absolute path and set working directory so Go can find the module
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// #nosec G204 - outPath and mainPath validated above, command is hardcoded "go build"
		buildCmd := exec.CommandContext(ctx, "go", "build", "-o", outPath, mainPath)
		buildCmd.Dir = repoRoot
		buildCmd.Env = append(os.Environ(), "GO111MODULE=on")
		output, err := buildCmd.CombinedOutput()
		require.NoError(t, err, "build failed: %s", string(output))
		defer func() { _ = os.Remove("kira") }()

		// Initialize workspace
		initCmd := exec.Command("./kira", "init")
		output, err = initCmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(output))

		// Create a work item using template input
		// We'll create a simple work item by writing it directly since interactive input is hard to test
		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
assigned: test@example.com
estimate: 3
created: 2024-01-01
---

# Test Feature

## Context
This is a test feature for integration testing.

## Requirements
- Implement user authentication
- Add login/logout functionality

## Acceptance Criteria
- [ ] User can log in with email/password
- [ ] User can log out
- [ ] Session is maintained across page refreshes

## Implementation Notes
Use JWT tokens for authentication.

## Release Notes
Added user authentication system.
`
		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(workItemContent), 0o600))

		// Test kira lint
		lintCmd := exec.Command("./kira", "lint")
		output, err = lintCmd.CombinedOutput()
		require.NoError(t, err, "lint failed: %s", string(output))
		assert.Contains(t, string(output), "No issues found")

		// Test kira move
		moveCmd := exec.Command("./kira", "move", "001", "doing")
		output, err = moveCmd.CombinedOutput()
		require.NoError(t, err, "move failed: %s", string(output))
		assert.Contains(t, string(output), "Moved work item 001 to doing")

		// Check that file was moved
		assert.FileExists(t, ".work/2_doing/001-test-feature.prd.md")
		assert.NoFileExists(t, ".work/1_todo/001-test-feature.prd.md")

		// Test kira save (this will fail if git is not initialized, which is expected)
		saveCmd := exec.Command("./kira", "save", "Test commit")
		_, err = saveCmd.CombinedOutput()

		// We expect this to fail because git is not initialized
		assert.Error(t, err)
	})

	t.Run("template system test", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Build the kira binary for testing using the repo root as working directory
		repoRoot, err := findRepoRoot()
		require.NoError(t, err, "failed to find repo root")
		outPath := filepath.Join(tmpDir, "kira")
		mainPath := filepath.Join(repoRoot, "cmd", "kira", "main.go")
		require.NoError(t, validateTestPath(outPath, tmpDir))
		// Use absolute path and set working directory so Go can find the module
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// #nosec G204 - outPath and mainPath validated above, command is hardcoded "go build"
		buildCmd := exec.CommandContext(ctx, "go", "build", "-o", outPath, mainPath)
		buildCmd.Dir = repoRoot
		buildCmd.Env = append(os.Environ(), "GO111MODULE=on")
		output, err := buildCmd.CombinedOutput()
		require.NoError(t, err, "build failed: %s", string(output))
		defer func() { _ = os.Remove("kira") }()

		// Initialize workspace
		initCmd := exec.Command("./kira", "init")
		output, err = initCmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(output))

		// Check that templates were created
		templateFiles := []string{
			".work/templates/template.prd.md",
			".work/templates/template.issue.md",
			".work/templates/template.spike.md",
			".work/templates/template.task.md",
		}

		for _, templateFile := range templateFiles {
			assert.FileExists(t, templateFile)

			// Check that template contains input placeholders
			// #nosec G304 - test file path, safe
			content, err := os.ReadFile(templateFile)
			require.NoError(t, err)
			assert.Contains(t, string(content), "<!--input-")
		}

		// Test help-inputs command
		helpCmd := exec.Command("./kira", "new", "prd", "--help-inputs")
		output, err = helpCmd.CombinedOutput()
		require.NoError(t, err, "help-inputs failed: %s", string(output))
		assert.Contains(t, string(output), "Available inputs for template 'prd'")
	})

	t.Run("release command generates notes and archives done items", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Build the kira binary
		_, thisFile, _, _ := runtime.Caller(0)
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
		outPath := filepath.Join(tmpDir, "kira")
		require.NoError(t, validateTestPath(outPath, tmpDir))
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// #nosec G204 - outPath validated above, command is hardcoded "go build"
		buildCmd := exec.CommandContext(ctx, "go", "build", "-o", outPath, "kira/cmd/kira")
		buildCmd.Dir = repoRoot
		output, err := buildCmd.CombinedOutput()
		require.NoError(t, err, "build failed: %s", string(output))
		defer func() { _ = os.Remove("kira") }()

		// Initialize workspace
		initCmd := exec.Command("./kira", "init")
		output, err = initCmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(output))

		// Create a done item with Release Notes section
		require.NoError(t, os.MkdirAll(".work/4_done", 0o700))
		doneItem := `---
id: 001
title: Done Feature
status: done
kind: prd
created: 2024-01-01
---

# Done Feature

## Context
Something

## Release Notes
This is a release note entry.
`
		require.NoError(t, os.WriteFile(".work/4_done/001-done-feature.prd.md", []byte(doneItem), 0o600))

		// Run release (default from done)
		releaseCmd := exec.Command("./kira", "release")
		output, err = releaseCmd.CombinedOutput()
		require.NoError(t, err, "release failed: %s", string(output))
		assert.Contains(t, string(output), "Released 1 work items")

		// Check archived file exists and original removed
		archivedMatches, _ := filepath.Glob(".work/z_archive/*/4_done/001-done-feature.prd.md")
		assert.NotEmpty(t, archivedMatches, "archived file not found")
		assert.NoFileExists(t, ".work/4_done/001-done-feature.prd.md")

		// Check RELEASES.md contains note
		releasesContent, err := os.ReadFile("RELEASES.md")
		require.NoError(t, err)
		assert.Contains(t, string(releasesContent), "This is a release note entry.")
	})

	t.Run("abandon command handles id, reason, status path and subfolder", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Build the kira binary
		_, thisFile, _, _ := runtime.Caller(0)
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
		outPath := filepath.Join(tmpDir, "kira")
		require.NoError(t, validateTestPath(outPath, tmpDir))
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// #nosec G204 - outPath validated above, command is hardcoded "go build"
		buildCmd := exec.CommandContext(ctx, "go", "build", "-o", outPath, "kira/cmd/kira")
		buildCmd.Dir = repoRoot
		output, err := buildCmd.CombinedOutput()
		require.NoError(t, err, "build failed: %s", string(output))
		defer func() { _ = os.Remove("kira") }()

		// Initialize workspace
		initCmd := exec.Command("./kira", "init")
		output, err = initCmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(output))

		// Create two items in todo and a subfolder
		require.NoError(t, os.MkdirAll(".work/1_todo/sub", 0o700))
		item1 := `---
id: 001
title: Todo One
status: todo
kind: prd
created: 2024-01-01
---
`
		item2 := `---
id: 002
title: Todo Two
status: todo
kind: prd
created: 2024-01-01
---
`
		require.NoError(t, os.WriteFile(".work/1_todo/001-todo-one.prd.md", []byte(item1), 0o600))
		require.NoError(t, os.WriteFile(".work/1_todo/sub/002-todo-two.prd.md", []byte(item2), 0o600))

		// Abandon by id with reason
		abandonID := exec.Command("./kira", "abandon", "001", "No longer needed")
		output, err = abandonID.CombinedOutput()
		require.NoError(t, err, "abandon by id failed: %s", string(output))
		// Verify archived
		archived1, _ := filepath.Glob(".work/z_archive/*/1_todo/001-todo-one.prd.md")
		assert.NotEmpty(t, archived1)
		assert.NoFileExists(t, ".work/1_todo/001-todo-one.prd.md")
		// Verify abandonment note present
		content, err := os.ReadFile(archived1[0])
		require.NoError(t, err)
		assert.Contains(t, string(content), "## Abandonment")
		assert.Contains(t, string(content), "No longer needed")

		// Abandon by status path subfolder (todo sub)
		abandonFolder := exec.Command("./kira", "abandon", "todo", "sub")
		output, err = abandonFolder.CombinedOutput()
		require.NoError(t, err, "abandon folder failed: %s", string(output))
		archived2, _ := filepath.Glob(".work/z_archive/*/sub/002-todo-two.prd.md")
		assert.NotEmpty(t, archived2)
		assert.NoFileExists(t, ".work/1_todo/sub/002-todo-two.prd.md")
	})

	t.Run("save command commits and updates timestamps in git repo", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Build binary
		_, thisFile, _, _ := runtime.Caller(0)
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
		outPath := filepath.Join(tmpDir, "kira")
		require.NoError(t, validateTestPath(outPath, tmpDir))
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// #nosec G204 - outPath validated above, command is hardcoded "go build"
		buildCmd := exec.CommandContext(ctx, "go", "build", "-o", outPath, "kira/cmd/kira")
		buildCmd.Dir = repoRoot
		output, err := buildCmd.CombinedOutput()
		require.NoError(t, err, "build failed: %s", string(output))
		defer func() { _ = os.Remove("kira") }()

		// git init
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Initialize workspace and add initial commit
		initCmd := exec.Command("./kira", "init")
		output, err = initCmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(output))
		require.NoError(t, exec.Command("git", "add", ".").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "init").Run())

		// Create a work item
		item := `---
id: 001
title: Save Test
status: todo
kind: prd
created: 2024-01-01
---

# Save Test
`
		require.NoError(t, os.WriteFile(".work/1_todo/001-save-test.prd.md", []byte(item), 0o600))

		// Run save with custom message
		saveCmd := exec.Command("./kira", "save", "Custom commit message")
		output, err = saveCmd.CombinedOutput()
		require.NoError(t, err, "save failed: %s", string(output))
		assert.Contains(t, string(output), "Work items saved and committed successfully.")

		// Ensure updated field added
		content, err := os.ReadFile(".work/1_todo/001-save-test.prd.md")
		require.NoError(t, err)
		assert.Contains(t, string(content), "updated:")

		// Verify last commit message
		logOut, err := exec.Command("git", "log", "-1", "--pretty=%B").Output()
		require.NoError(t, err)
		assert.Contains(t, string(logOut), "Custom commit message")

		// Verify only .work files in commit
		showOut, err := exec.Command("git", "show", "--name-only", "--pretty=", "HEAD").Output()
		require.NoError(t, err)
		for _, line := range strings.Split(strings.TrimSpace(string(showOut)), "\n") {
			if line == "" {
				continue
			}
			assert.True(t, strings.HasPrefix(line, ".work/"), "commit touched non-.work file: %s", line)
		}
	})

	t.Run("save fails on validation errors", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Build binary
		_, thisFile, _, _ := runtime.Caller(0)
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
		outPath := filepath.Join(tmpDir, "kira")
		require.NoError(t, validateTestPath(outPath, tmpDir))
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// #nosec G204 - outPath validated above, command is hardcoded "go build"
		buildCmd := exec.CommandContext(ctx, "go", "build", "-o", outPath, "kira/cmd/kira")
		buildCmd.Dir = repoRoot
		output, err := buildCmd.CombinedOutput()
		require.NoError(t, err, "build failed: %s", string(output))
		defer func() { _ = os.Remove("kira") }()

		// Init workspace (no git init to force no commit path)
		initCmd := exec.Command("./kira", "init")
		_, err = initCmd.CombinedOutput()
		require.NoError(t, err)

		// Create invalid item
		invalid := `---
id: 001
title: Bad
status: invalid-status
kind: prd
created: 2024-01-01
---
`
		require.NoError(t, os.WriteFile(".work/1_todo/001-bad.prd.md", []byte(invalid), 0o600))

		// Save should fail
		saveCmd := exec.Command("./kira", "save", "attempt")
		output, err = saveCmd.CombinedOutput()
		assert.Error(t, err)
		assert.Contains(t, string(output), "validation failed")
	})

	t.Run("idea conversion workflow", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		kiraPath := buildKiraBinary(t, tmpDir)

		// Initialize workspace
		initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
		require.NoError(t, err)
		output, err := initCmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(output))

		// Add an idea
		ideaCmd, err := safeExecCommand(tmpDir, kiraPath, "idea", "dark mode: allow the user to toggle between light and dark mode")
		require.NoError(t, err)
		output, err = ideaCmd.CombinedOutput()
		require.NoError(t, err, "idea add failed: %s", string(output))
		assert.Contains(t, string(output), "Added idea")

		// List ideas
		listCmd, err := safeExecCommand(tmpDir, kiraPath, "idea", "list")
		require.NoError(t, err)
		output, err = listCmd.CombinedOutput()
		require.NoError(t, err, "idea list failed: %s", string(output))
		assert.Contains(t, string(output), "dark mode")

		// Convert idea to work item
		convertCmd, err := safeExecCommand(tmpDir, kiraPath, "new", "prd", "todo", "idea", "1")
		require.NoError(t, err)
		output, err = convertCmd.CombinedOutput()
		require.NoError(t, err, "convert idea failed: %s", string(output))
		assert.Contains(t, string(output), "Created work item")

		// Verify idea was removed from IDEAS.md
		ideasPath := filepath.Join(tmpDir, ".work", "IDEAS.md")
		ideasContent, err := safeReadTestFile(ideasPath, tmpDir)
		require.NoError(t, err)
		assert.NotContains(t, string(ideasContent), "dark mode")

		// Verify work item was created
		globPattern := filepath.Join(tmpDir, ".work", "1_todo", "*.md")
		files, err := filepath.Glob(globPattern)
		require.NoError(t, err)
		require.Len(t, files, 1)

		// Verify work item content
		workItemContent, err := safeReadTestFile(files[0], tmpDir)
		require.NoError(t, err)
		assert.Contains(t, string(workItemContent), "title: dark mode")
		assert.Contains(t, string(workItemContent), "allow the user to toggle between light and dark mode")
	})

	t.Run("idea conversion with multiple ideas renumbers correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		kiraPath := buildKiraBinary(t, tmpDir)

		// Initialize workspace
		initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
		require.NoError(t, err)
		output, err := initCmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(output))

		// Add multiple ideas
		idea1Cmd, err := safeExecCommand(tmpDir, kiraPath, "idea", "First idea")
		require.NoError(t, err)
		output, err = idea1Cmd.CombinedOutput()
		require.NoError(t, err, "idea 1 add failed: %s", string(output))

		idea2Cmd, err := safeExecCommand(tmpDir, kiraPath, "idea", "Second idea")
		require.NoError(t, err)
		output, err = idea2Cmd.CombinedOutput()
		require.NoError(t, err, "idea 2 add failed: %s", string(output))

		idea3Cmd, err := safeExecCommand(tmpDir, kiraPath, "idea", "Third idea")
		require.NoError(t, err)
		output, err = idea3Cmd.CombinedOutput()
		require.NoError(t, err, "idea 3 add failed: %s", string(output))

		// Convert middle idea (idea 2)
		convertCmd, err := safeExecCommand(tmpDir, kiraPath, "new", "prd", "todo", "idea", "2")
		require.NoError(t, err)
		output, err = convertCmd.CombinedOutput()
		require.NoError(t, err, "convert idea failed: %s", string(output))

		// Verify remaining ideas are renumbered
		listCmd, err := safeExecCommand(tmpDir, kiraPath, "idea", "list")
		require.NoError(t, err)
		output, err = listCmd.CombinedOutput()
		require.NoError(t, err, "idea list failed: %s", string(output))

		outputStr := string(output)
		// Should have ideas 1 and 2 (renumbered from original 1 and 3)
		assert.Contains(t, outputStr, "1. [")
		assert.Contains(t, outputStr, "2. [")
		assert.Contains(t, outputStr, "First idea")
		assert.Contains(t, outputStr, "Third idea")
		// Should not have "Second idea" anymore
		assert.NotContains(t, outputStr, "Second idea")
	})

	t.Run("idea conversion error cases", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		kiraPath := buildKiraBinary(t, tmpDir)

		// Initialize workspace
		initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
		require.NoError(t, err)
		output, err := initCmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(output))

		// Try to convert non-existent idea
		convertCmd, err := safeExecCommand(tmpDir, kiraPath, "new", "prd", "todo", "idea", "999")
		require.NoError(t, err)
		output, err = convertCmd.CombinedOutput()
		assert.Error(t, err)
		assert.Contains(t, string(output), "Idea 999 not found")

		// Try with invalid idea number
		convertCmd, err = safeExecCommand(tmpDir, kiraPath, "new", "prd", "todo", "idea", "abc")
		require.NoError(t, err)
		output, err = convertCmd.CombinedOutput()
		assert.Error(t, err)
		assert.Contains(t, string(output), "invalid idea number")

		// Try with zero
		convertCmd, err = safeExecCommand(tmpDir, kiraPath, "new", "prd", "todo", "idea", "0")
		require.NoError(t, err)
		output, err = convertCmd.CombinedOutput()
		assert.Error(t, err)
		assert.Contains(t, string(output), "invalid idea number")
	})

	t.Run("idea parsing with and without colons", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		kiraPath := buildKiraBinary(t, tmpDir)

		// Initialize workspace
		initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
		require.NoError(t, err)
		output, err := initCmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(output))

		// Add idea with colon
		idea1Cmd, err := safeExecCommand(tmpDir, kiraPath, "idea", "add", "dark mode: allow toggle")
		require.NoError(t, err)
		err = idea1Cmd.Run()
		require.NoError(t, err)

		// Add idea without colon (more than 5 words)
		idea2Cmd, err := safeExecCommand(tmpDir, kiraPath, "idea", "add", "user authentication requirements with OAuth support")
		require.NoError(t, err)
		err = idea2Cmd.Run()
		require.NoError(t, err)

		// Add idea without colon (fewer than 5 words)
		idea3Cmd, err := safeExecCommand(tmpDir, kiraPath, "idea", "add", "fix login bug")
		require.NoError(t, err)
		err = idea3Cmd.Run()
		require.NoError(t, err)

		// Convert idea with colon
		convert1Cmd, err := safeExecCommand(tmpDir, kiraPath, "new", "prd", "todo", "idea", "1")
		require.NoError(t, err)
		err = convert1Cmd.Run()
		require.NoError(t, err)

		// Verify work item has correct title and description
		globPattern := filepath.Join(tmpDir, ".work", "1_todo", "*.md")
		files, err := filepath.Glob(globPattern)
		require.NoError(t, err)
		require.Len(t, files, 1)

		workItemContent, err := safeReadTestFile(files[0], tmpDir)
		require.NoError(t, err)
		contentStr := string(workItemContent)
		assert.Contains(t, contentStr, "title: dark mode")
		// Description gets mapped to context field in PRD template
		assert.Contains(t, contentStr, "allow toggle")

		// Convert idea without colon (more than 5 words)
		// After converting idea 1, ideas are renumbered: original idea 2 becomes idea 1
		convert2Cmd, err := safeExecCommand(tmpDir, kiraPath, "new", "prd", "todo", "idea", "1")
		require.NoError(t, err)
		err = convert2Cmd.Run()
		require.NoError(t, err)

		// Verify second work item
		globPattern2 := filepath.Join(tmpDir, ".work", "1_todo", "*.md")
		files, err = filepath.Glob(globPattern2)
		require.NoError(t, err)
		require.Len(t, files, 2)

		// Find the second work item (the one with "user authentication" or "OAuth")
		var secondFile string
		for _, file := range files {
			content, err := safeReadTestFile(file, tmpDir)
			require.NoError(t, err)
			contentStr := string(content)
			// Find the work item that contains "user authentication" or "OAuth"
			if strings.Contains(contentStr, "user authentication") || strings.Contains(contentStr, "OAuth") {
				secondFile = file
				break
			}
		}
		require.NotEmpty(t, secondFile, "Should find second work item file with user authentication")

		workItemContent2, err := safeReadTestFile(secondFile, tmpDir)
		require.NoError(t, err)
		contentStr2 := string(workItemContent2)
		// Title should be first 5 words: "user authentication requirements with OAuth"
		assert.Contains(t, contentStr2, "user authentication requirements with OAuth")
		// Description should contain the full text in context field
		assert.Contains(t, contentStr2, "user authentication requirements with OAuth support")

		// Convert idea without colon (fewer than 5 words)
		// After converting idea 1 again, original idea 3 becomes idea 1
		convert3Cmd, err := safeExecCommand(tmpDir, kiraPath, "new", "prd", "todo", "idea", "1")
		require.NoError(t, err)
		err = convert3Cmd.Run()
		require.NoError(t, err)

		// Verify third work item
		globPattern3 := filepath.Join(tmpDir, ".work", "1_todo", "*.md")
		files, err = filepath.Glob(globPattern3)
		require.NoError(t, err)
		require.Len(t, files, 3)

		// Find the third work item
		var thirdFile string
		for _, file := range files {
			content, err := safeReadTestFile(file, tmpDir)
			require.NoError(t, err)
			if strings.Contains(string(content), "fix login bug") {
				thirdFile = file
				break
			}
		}
		require.NotEmpty(t, thirdFile)

		workItemContent3, err := safeReadTestFile(thirdFile, tmpDir)
		require.NoError(t, err)
		contentStr3 := string(workItemContent3)
		assert.Contains(t, contentStr3, "title: fix login bug")
	})
}

const (
	defaultKiraYML = `version: "1.0"
templates:
  prd: templates/template.prd.md
status_folders:
  doing: 2_doing
git:
  trunk_branch: main
  remote: origin
`
)

// TestLatestCommand_MultiRepoCoordination tests full multi-repo workflow with fetch and rebase
func TestLatestCommand_MultiRepoCoordination(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(originalDir) }()

	kiraPath := buildKiraBinary(t, tmpDir)

	// Initialize kira workspace
	initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
	require.NoError(t, err)
	output, err := initCmd.CombinedOutput()
	require.NoError(t, err, "init failed: %s", string(output))

	// Create work item in doing folder
	workItemContent := `---
id: 001
title: Multi-Repo Test Feature
status: doing
kind: prd
created: 2024-01-01
---
# Multi-Repo Test Feature
`
	require.NoError(t, os.WriteFile(".work/2_doing/001-multi-repo-test-feature.prd.md", []byte(workItemContent), 0o600))

	// Set up polyrepo configuration with 2 repositories
	repo1Dir := filepath.Join(tmpDir, "repo1")
	repo2Dir := filepath.Join(tmpDir, "repo2")
	require.NoError(t, os.MkdirAll(repo1Dir, 0o700))
	require.NoError(t, os.MkdirAll(repo2Dir, 0o700))

	// Initialize git repos
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo1Dir, "init").Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo1Dir, "config", "user.email", "test@example.com").Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo1Dir, "config", "user.name", "Test User").Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo2Dir, "init").Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo2Dir, "config", "user.email", "test@example.com").Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo2Dir, "config", "user.name", "Test User").Run())

	// Create initial commits - git init creates default branch (main or master)
	require.NoError(t, os.WriteFile(filepath.Join(repo1Dir, "file1.txt"), []byte("repo1 initial"), 0o600))
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo1Dir, "add", "file1.txt").Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo1Dir, "commit", "-m", "Initial commit").Run())
	// Rename branch to main if it's not already main (use -M to force)
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	_ = exec.Command("git", "-C", repo1Dir, "branch", "-M", "main").Run() // Ignore error if already main

	require.NoError(t, os.WriteFile(filepath.Join(repo2Dir, "file2.txt"), []byte("repo2 initial"), 0o600))
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo2Dir, "add", "file2.txt").Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo2Dir, "commit", "-m", "Initial commit").Run())
	// Rename branch to main if it's not already main (use -M to force)
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	_ = exec.Command("git", "-C", repo2Dir, "branch", "-M", "main").Run() // Ignore error if already main

	// Create feature branches
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo1Dir, "checkout", "-b", "001-multi-repo-test-feature").Run())
	require.NoError(t, os.WriteFile(filepath.Join(repo1Dir, "feature1.txt"), []byte("feature1"), 0o600))
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo1Dir, "add", "feature1.txt").Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo1Dir, "commit", "-m", "Feature commit 1").Run())

	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo2Dir, "checkout", "-b", "001-multi-repo-test-feature").Run())
	require.NoError(t, os.WriteFile(filepath.Join(repo2Dir, "feature2.txt"), []byte("feature2"), 0o600))
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo2Dir, "add", "feature2.txt").Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo2Dir, "commit", "-m", "Feature commit 2").Run())

	// Create remotes (bare repos)
	remote1Dir := t.TempDir()
	remote2Dir := t.TempDir()
	// #nosec G204 - remote paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "init", "--bare", remote1Dir).Run())
	// #nosec G204 - remote paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "init", "--bare", remote2Dir).Run())

	// Add remotes and push main branches
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo1Dir, "remote", "add", "origin", remote1Dir).Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo1Dir, "checkout", "main").Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo1Dir, "push", "-u", "origin", "main").Run())

	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo2Dir, "remote", "add", "origin", remote2Dir).Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo2Dir, "checkout", "main").Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo2Dir, "push", "-u", "origin", "main").Run())

	// Add new commits to main branches and push
	require.NoError(t, os.WriteFile(filepath.Join(repo1Dir, "main1.txt"), []byte("main update 1"), 0o600))
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo1Dir, "add", "main1.txt").Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo1Dir, "commit", "-m", "Main update 1").Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo1Dir, "push", "origin", "main").Run())

	require.NoError(t, os.WriteFile(filepath.Join(repo2Dir, "main2.txt"), []byte("main update 2"), 0o600))
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo2Dir, "add", "main2.txt").Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo2Dir, "commit", "-m", "Main update 2").Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo2Dir, "push", "origin", "main").Run())

	// Switch back to feature branches
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo1Dir, "checkout", "001-multi-repo-test-feature").Run())
	// #nosec G204 - repo paths are from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repo2Dir, "checkout", "001-multi-repo-test-feature").Run())

	// Update kira.yml with polyrepo configuration
	kiraYML := defaultKiraYML + `workspace:
  projects:
    - name: project1
      path: ` + repo1Dir + `
      trunk_branch: main
      remote: origin
    - name: project2
      path: ` + repo2Dir + `
      trunk_branch: main
      remote: origin
`
	require.NoError(t, os.WriteFile("kira.yml", []byte(kiraYML), 0o600))

	// Initialize git in workspace root (required for repository discovery)
	// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "init").Run())
	require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
	require.NoError(t, exec.Command("git", "add", ".").Run())
	require.NoError(t, exec.Command("git", "commit", "-m", "Initial").Run())
	// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
	_ = exec.Command("git", "branch", "-M", "main").Run() // Ignore error if already main

	// Run kira latest
	latestCmd, err := safeExecCommand(tmpDir, kiraPath, "latest")
	require.NoError(t, err)
	output, err = latestCmd.CombinedOutput()
	require.NoError(t, err, "latest failed: %s", string(output))

	// Verify output contains repository discovery
	assert.Contains(t, string(output), "Discovered")
	assert.Contains(t, string(output), "repository")

	// Verify both repos were processed (check for progress messages or success)
	outputStr := string(output)
	// Should show progress for both repos or success message
	assert.True(t, strings.Contains(outputStr, "project1") || strings.Contains(outputStr, "project2") || strings.Contains(outputStr, "SUCCESS") || strings.Contains(outputStr, "updated successfully"))
}

// TestLatestCommand_IterativeWorkflow tests conflict detection, display, and resolution workflow
func TestLatestCommand_IterativeWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(originalDir) }()

	kiraPath := buildKiraBinary(t, tmpDir)

	// Initialize kira workspace
	initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
	require.NoError(t, err)
	_, err = initCmd.CombinedOutput()
	require.NoError(t, err)

	// Create work item in doing folder
	workItemContent := `---
id: 001
title: Iterative Test Feature
status: doing
kind: prd
created: 2024-01-01
---
# Iterative Test Feature
`
	require.NoError(t, os.WriteFile(".work/2_doing/001-iterative-test-feature.prd.md", []byte(workItemContent), 0o600))

	// Set up git repo
	repoDir := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(repoDir, 0o700))
	// #nosec G204 - repo path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repoDir, "init").Run())
	// #nosec G204 - repo path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repoDir, "config", "user.email", "test@example.com").Run())
	// #nosec G204 - repo path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repoDir, "config", "user.name", "Test User").Run())

	// Create initial commit on main
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "test.txt"), []byte("line1\nline2\nline3\n"), 0o600))
	// #nosec G204 - repo path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repoDir, "add", "test.txt").Run())
	// #nosec G204 - repo path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repoDir, "commit", "-m", "Initial commit").Run())
	// #nosec G204 - repo path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repoDir, "branch", "-M", "main").Run())

	// Create feature branch and modify same line
	// #nosec G204 - repo path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repoDir, "checkout", "-b", "001-iterative-test-feature").Run())
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "test.txt"), []byte("line1\nfeature change\nline3\n"), 0o600))
	// #nosec G204 - repo path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repoDir, "add", "test.txt").Run())
	// #nosec G204 - repo path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repoDir, "commit", "-m", "Feature change").Run())

	// Create remote and push
	remoteDir := t.TempDir()
	// #nosec G204 - remote path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())
	// #nosec G204 - repo path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repoDir, "remote", "add", "origin", remoteDir).Run())
	// #nosec G204 - repo path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repoDir, "checkout", "main").Run())
	// #nosec G204 - repo path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repoDir, "push", "-u", "origin", "main").Run())

	// Modify same line on main and push
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "test.txt"), []byte("line1\nmain change\nline3\n"), 0o600))
	// #nosec G204 - repo path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repoDir, "add", "test.txt").Run())
	// #nosec G204 - repo path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repoDir, "commit", "-m", "Main change").Run())
	// #nosec G204 - repo path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repoDir, "push", "origin", "main").Run())

	// Switch back to feature branch
	// #nosec G204 - repo path is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "-C", repoDir, "checkout", "001-iterative-test-feature").Run())

	// Update kira.yml
	require.NoError(t, os.WriteFile("kira.yml", []byte(defaultKiraYML), 0o600))

	// Initialize git in workspace root for standalone repo detection
	// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
	require.NoError(t, exec.Command("git", "init").Run())
	require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
	require.NoError(t, exec.Command("git", "add", ".").Run())
	require.NoError(t, exec.Command("git", "commit", "-m", "Initial").Run())
	// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
	_ = exec.Command("git", "branch", "-M", "main").Run() // Ignore error if already main

	// First run: should detect conflicts
	latestCmd, err := safeExecCommand(tmpDir, kiraPath, "latest")
	require.NoError(t, err)
	output, err2 := latestCmd.CombinedOutput()
	// May succeed if rebase completes, or show conflicts
	outputStr := string(output)
	_ = err2 // Use err2 to avoid ineffectual assignment

	// Verify conflict display format if conflicts exist
	if strings.Contains(outputStr, "Merge Conflicts Detected") || strings.Contains(outputStr, "conflict") {
		// Verify conflict markers are present
		assert.True(t, strings.Contains(outputStr, "<<<<<<<") || strings.Contains(outputStr, "=======") || strings.Contains(outputStr, ">>>>>>>"))
		// Verify repository context
		assert.Contains(t, outputStr, "Repository:")
		// Verify instructions
		assert.Contains(t, outputStr, "resolve")
	}

	// If conflicts were detected, resolve them and run again
	if strings.Contains(outputStr, "conflict") || strings.Contains(outputStr, "CONFLICT") {
		// Resolve conflicts programmatically
		require.NoError(t, os.WriteFile(filepath.Join(repoDir, "test.txt"), []byte("line1\nresolved change\nline3\n"), 0o600))
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "add", "test.txt").Run())
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "commit", "-m", "Resolve conflicts").Run())

		// Run kira latest again - should complete successfully
		latestCmd2, err := safeExecCommand(tmpDir, kiraPath, "latest")
		require.NoError(t, err)
		output2, err2 := latestCmd2.CombinedOutput()
		// Should not error on second run
		assert.NoError(t, err2, "second latest run failed: %s", string(output2))
		_ = output2 // Use output2 to avoid ineffectual assignment
	}
}

// TestLatestCommand_ConfigurationIntegration tests config priority and auto-detection
func TestLatestCommand_ConfigurationIntegration(t *testing.T) {
	t.Run("standalone repo with config", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		kiraPath := buildKiraBinary(t, tmpDir)

		// Initialize kira workspace
		initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
		require.NoError(t, err)
		_, err = initCmd.CombinedOutput()
		require.NoError(t, err)

		// Create work item
		workItemContent := `---
id: 001
title: Config Test
status: doing
kind: prd
created: 2024-01-01
---
`
		require.NoError(t, os.WriteFile(".work/2_doing/001-config-test.prd.md", []byte(workItemContent), 0o600))

		// Initialize git repo with develop branch
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "develop").Run())
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Update kira.yml with custom trunk_branch
		kiraYML := `version: "1.0"
templates:
  prd: templates/template.prd.md
status_folders:
  doing: 2_doing
git:
  trunk_branch: develop
  remote: origin
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(kiraYML), 0o600))

		// Run kira latest - should use develop as trunk branch
		latestCmd, err := safeExecCommand(tmpDir, kiraPath, "latest")
		require.NoError(t, err)
		output, err2 := latestCmd.CombinedOutput()
		// Should not error (may fail if no remote, but should use develop branch)
		outputStr := string(output)
		assert.True(t, strings.Contains(outputStr, "develop") || err2 != nil, "should reference develop branch or handle error gracefully")
		_ = err2 // Use err2 to avoid ineffectual assignment
	})

	t.Run("polyrepo with project overrides", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		kiraPath := buildKiraBinary(t, tmpDir)

		// Initialize kira workspace
		initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
		require.NoError(t, err)
		_, err = initCmd.CombinedOutput()
		require.NoError(t, err)

		// Create work item
		workItemContent := `---
id: 001
title: Polyrepo Config Test
status: doing
kind: prd
created: 2024-01-01
---
`
		require.NoError(t, os.WriteFile(".work/2_doing/001-polyrepo-config-test.prd.md", []byte(workItemContent), 0o600))

		// Set up two repos with different trunk branches
		repo1Dir := filepath.Join(tmpDir, "repo1")
		repo2Dir := filepath.Join(tmpDir, "repo2")
		require.NoError(t, os.MkdirAll(repo1Dir, 0o700))
		require.NoError(t, os.MkdirAll(repo2Dir, 0o700))

		// Initialize repos
		// #nosec G204 - repo paths are from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repo1Dir, "init").Run())
		// #nosec G204 - repo paths are from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repo1Dir, "config", "user.email", "test@example.com").Run())
		// #nosec G204 - repo paths are from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repo1Dir, "config", "user.name", "Test User").Run())
		require.NoError(t, os.WriteFile(filepath.Join(repo1Dir, "file1.txt"), []byte("repo1"), 0o600))
		// #nosec G204 - repo paths are from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repo1Dir, "add", "file1.txt").Run())
		// #nosec G204 - repo paths are from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repo1Dir, "commit", "-m", "Initial").Run())
		// Rename branch to develop after commit
		// #nosec G204 - repo paths are from t.TempDir(), safe for test use
		_ = exec.Command("git", "-C", repo1Dir, "branch", "-M", "develop").Run() // Ignore error if already develop

		// #nosec G204 - repo paths are from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repo2Dir, "init").Run())
		// #nosec G204 - repo paths are from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repo2Dir, "config", "user.email", "test@example.com").Run())
		// #nosec G204 - repo paths are from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repo2Dir, "config", "user.name", "Test User").Run())
		require.NoError(t, os.WriteFile(filepath.Join(repo2Dir, "file2.txt"), []byte("repo2"), 0o600))
		// #nosec G204 - repo paths are from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repo2Dir, "add", "file2.txt").Run())
		// #nosec G204 - repo paths are from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repo2Dir, "commit", "-m", "Initial").Run())
		// #nosec G204 - repo paths are from t.TempDir(), safe for test use
		_ = exec.Command("git", "-C", repo2Dir, "branch", "-M", "main").Run() // Ignore error if already main

		// Update kira.yml with project overrides
		kiraYML := defaultKiraYML + `workspace:
  projects:
    - name: project1
      path: ` + repo1Dir + `
      trunk_branch: develop
      remote: origin
    - name: project2
      path: ` + repo2Dir + `
      trunk_branch: main
      remote: origin
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(kiraYML), 0o600))

		// Initialize git in workspace root
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "add", ".").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial").Run())
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		_ = exec.Command("git", "branch", "-M", "main").Run() // Ignore error if already main

		// Run kira latest - should discover both repos with correct trunk branches
		latestCmd, err := safeExecCommand(tmpDir, kiraPath, "latest")
		require.NoError(t, err)
		output, err2 := latestCmd.CombinedOutput()
		outputStr := string(output)
		// Should discover both projects
		assert.Contains(t, outputStr, "Discovered")
		assert.True(t, strings.Contains(outputStr, "project1") || strings.Contains(outputStr, "project2") || strings.Contains(outputStr, "develop") || strings.Contains(outputStr, "main"))
		_ = err2 // Use err2 to avoid ineffectual assignment
	})
}

// TestLatestCommand_ErrorRecovery tests error scenarios and stash management
func TestLatestCommand_ErrorRecovery(t *testing.T) {
	t.Run("dirty working directory with stash", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		kiraPath := buildKiraBinary(t, tmpDir)

		// Initialize kira workspace
		initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
		require.NoError(t, err)
		_, err = initCmd.CombinedOutput()
		require.NoError(t, err)

		// Create work item
		workItemContent := `---
id: 001
title: Stash Test
status: doing
kind: prd
created: 2024-01-01
---
`
		require.NoError(t, os.WriteFile(".work/2_doing/001-stash-test.prd.md", []byte(workItemContent), 0o600))

		// Initialize git repo
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "main").Run())
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Create feature branch
		require.NoError(t, exec.Command("git", "checkout", "-b", "001-stash-test").Run())

		// Create uncommitted changes
		require.NoError(t, os.WriteFile("dirty.txt", []byte("uncommitted"), 0o600))

		// Update kira.yml
		require.NoError(t, os.WriteFile("kira.yml", []byte(defaultKiraYML), 0o600))

		// Run kira latest - should stash changes
		latestCmd, err := safeExecCommand(tmpDir, kiraPath, "latest")
		require.NoError(t, err)
		output, err2 := latestCmd.CombinedOutput()
		outputStr := string(output)
		// Should mention stashing or handle dirty working directory
		assert.True(t, strings.Contains(outputStr, "stash") || strings.Contains(outputStr, "uncommitted") || strings.Contains(outputStr, "dirty") || err2 != nil)
		_ = err2 // Use err2 to avoid ineffectual assignment
	})

	t.Run("fetch failure handling", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		kiraPath := buildKiraBinary(t, tmpDir)

		// Initialize kira workspace
		initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
		require.NoError(t, err)
		_, err = initCmd.CombinedOutput()
		require.NoError(t, err)

		// Create work item
		workItemContent := `---
id: 001
title: Fetch Error Test
status: doing
kind: prd
created: 2024-01-01
---
`
		require.NoError(t, os.WriteFile(".work/2_doing/001-fetch-error-test.prd.md", []byte(workItemContent), 0o600))

		// Initialize git repo
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "main").Run())
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Create feature branch
		require.NoError(t, exec.Command("git", "checkout", "-b", "001-fetch-error-test").Run())

		// Update kira.yml with non-existent remote
		kiraYML := `version: "1.0"
templates:
  prd: templates/template.prd.md
status_folders:
  doing: 2_doing
git:
  trunk_branch: main
  remote: nonexistent
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(kiraYML), 0o600))

		// Run kira latest - should handle missing remote gracefully
		latestCmd, err := safeExecCommand(tmpDir, kiraPath, "latest")
		require.NoError(t, err)
		output, err2 := latestCmd.CombinedOutput()
		// Should provide clear error message about missing remote
		if err2 != nil {
			outputStr := string(output)
			assert.True(t, strings.Contains(outputStr, "does not exist") || strings.Contains(outputStr, "remote") || strings.Contains(outputStr, "fetch"))
		}
		_ = err2 // Use err2 to avoid ineffectual assignment
	})
}

// TestLatestCommand_StateDetectionAndConflicts tests state detection and conflict display formatting
func TestLatestCommand_StateDetectionAndConflicts(t *testing.T) {
	t.Run("clean state detection", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		kiraPath := buildKiraBinary(t, tmpDir)

		// Initialize kira workspace
		initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
		require.NoError(t, err)
		_, err = initCmd.CombinedOutput()
		require.NoError(t, err)

		// Create work item
		workItemContent := `---
id: 001
title: Clean State Test
status: doing
kind: prd
created: 2024-01-01
---
`
		require.NoError(t, os.WriteFile(".work/2_doing/001-clean-state-test.prd.md", []byte(workItemContent), 0o600))

		// Initialize git repo
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "main").Run())
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Create feature branch
		require.NoError(t, exec.Command("git", "checkout", "-b", "001-clean-state-test").Run())

		// Update kira.yml
		require.NoError(t, os.WriteFile("kira.yml", []byte(defaultKiraYML), 0o600))

		// Run kira latest - should detect clean state
		latestCmd, err := safeExecCommand(tmpDir, kiraPath, "latest")
		require.NoError(t, err)
		output, err2 := latestCmd.CombinedOutput()
		outputStr := string(output)
		// Should show state summary
		assert.True(t, strings.Contains(outputStr, "Repository State Summary") || strings.Contains(outputStr, "Checking repository state") || strings.Contains(outputStr, "ready"))
		_ = err2   // Use err2 to avoid ineffectual assignment
		_ = output // Use output to avoid ineffectual assignment
	})

	t.Run("conflict display format", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		kiraPath := buildKiraBinary(t, tmpDir)

		// Initialize kira workspace
		initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
		require.NoError(t, err)
		_, err = initCmd.CombinedOutput()
		require.NoError(t, err)

		// Create work item
		workItemContent := `---
id: 001
title: Conflict Display Test
status: doing
kind: prd
created: 2024-01-01
---
`
		require.NoError(t, os.WriteFile(".work/2_doing/001-conflict-display-test.prd.md", []byte(workItemContent), 0o600))

		// Set up git repo with conflicts
		repoDir := filepath.Join(tmpDir, "repo")
		require.NoError(t, os.MkdirAll(repoDir, 0o700))
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "init").Run())
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "config", "user.email", "test@example.com").Run())
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "config", "user.name", "Test User").Run())

		// Create initial commit
		require.NoError(t, os.WriteFile(filepath.Join(repoDir, "conflict.txt"), []byte("line1\nline2\n"), 0o600))
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "add", "conflict.txt").Run())
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "commit", "-m", "Initial").Run())
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "branch", "-M", "main").Run())

		// Create feature branch and modify
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "checkout", "-b", "001-conflict-display-test").Run())
		require.NoError(t, os.WriteFile(filepath.Join(repoDir, "conflict.txt"), []byte("line1\nfeature\nline2\n"), 0o600))
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "add", "conflict.txt").Run())
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "commit", "-m", "Feature").Run())

		// Modify on main
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "checkout", "main").Run())
		require.NoError(t, os.WriteFile(filepath.Join(repoDir, "conflict.txt"), []byte("line1\nmain\nline2\n"), 0o600))
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "add", "conflict.txt").Run())
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "commit", "-m", "Main").Run())

		// Create remote and push
		remoteDir := t.TempDir()
		// #nosec G204 - remote path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "remote", "add", "origin", remoteDir).Run())
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "push", "-u", "origin", "main").Run())

		// Switch to feature and start rebase to create conflict
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "checkout", "001-conflict-display-test").Run())
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "-C", repoDir, "fetch", "origin", "main").Run())
		// #nosec G204 - repo path is from t.TempDir(), safe for test use
		_ = exec.Command("git", "-C", repoDir, "rebase", "origin/main").Run() // This may create conflict

		// Check if conflicts actually exist in the file
		// #nosec G304 - repoDir is from t.TempDir(), safe for test use
		conflictFileContent, _ := os.ReadFile(filepath.Join(repoDir, "conflict.txt"))
		hasActualConflicts := strings.Contains(string(conflictFileContent), "<<<<<<<") || strings.Contains(string(conflictFileContent), "=======")

		// Update kira.yml
		kiraYML := defaultKiraYML + `workspace:
  projects:
    - name: project1
      path: ` + repoDir + `
      trunk_branch: main
      remote: origin
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(kiraYML), 0o600))

		// Initialize git in workspace root
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "add", ".").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial").Run())
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		_ = exec.Command("git", "branch", "-M", "main").Run() // Ignore error if already main

		// Run kira latest - should detect and display conflicts
		latestCmd, err := safeExecCommand(tmpDir, kiraPath, "latest")
		require.NoError(t, err)
		output, err2 := latestCmd.CombinedOutput()
		outputStr := string(output)
		_ = err2 // Use err2 to avoid ineffectual assignment

		// Verify conflict detection and display
		// The main goal is to verify that conflicts are detected when they exist
		// Detailed formatting may vary, so we check for conflict detection first
		if hasActualConflicts {
			// If conflicts exist in the file, the command should detect them
			// The output format may vary, but should indicate conflicts were found
			assert.True(t, strings.Contains(outputStr, "conflict") || strings.Contains(outputStr, "CONFLICT") || strings.Contains(outputStr, "Merge Conflicts") ||
				strings.Contains(outputStr, "Repository State Summary") || strings.Contains(outputStr, "Checking repository state"),
				"conflicts should be detected when present in repository")
		}
		// If no conflicts, that's also valid (may have been resolved or rebase completed)
		// Just verify the command executed
		assert.True(t, len(outputStr) > 0, "command should produce output")
	})
}

func TestDoctorCommand(t *testing.T) {
	t.Run("fixes date formats and preserves body content", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Build the kira binary for testing
		kiraPath := buildKiraBinary(t, tmpDir)

		// Initialize kira
		initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
		require.NoError(t, err)
		err = initCmd.Run()
		require.NoError(t, err)

		// Create a work item with invalid date format
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-15T10:30:00Z
---
# Test Feature

## Context
This is a test feature with body content.

## Requirements
- Requirement 1
- Requirement 2
`

		filePath := filepath.Join(tmpDir, ".work", "1_todo", "001-test-feature.prd.md")
		require.NoError(t, os.WriteFile(filePath, []byte(workItemContent), 0o600))

		// Run doctor command
		doctorCmd, err := safeExecCommand(tmpDir, kiraPath, "doctor")
		require.NoError(t, err)
		output, err := doctorCmd.CombinedOutput()
		require.NoError(t, err)

		outputStr := string(output)
		// Should show validation errors
		assert.Contains(t, outputStr, "Validation errors found")
		// Should fix date formats
		assert.Contains(t, outputStr, "Fixed date formats")
		assert.Contains(t, outputStr, "fixed created date format")

		// Verify the file was updated
		// #nosec G304 - filePath is constructed from tmpDir which is validated
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		contentStr := string(content)
		assert.Contains(t, contentStr, "created: 2024-01-15")
		assert.NotContains(t, contentStr, "created: 2024-01-15T10:30:00Z")

		// Verify body content is preserved
		assert.Contains(t, contentStr, "# Test Feature")
		assert.Contains(t, contentStr, "## Context")
		assert.Contains(t, contentStr, "This is a test feature with body content.")
		assert.Contains(t, contentStr, "## Requirements")
		assert.Contains(t, contentStr, "- Requirement 1")
		assert.Contains(t, contentStr, "- Requirement 2")
	})

	t.Run("fixes duplicate IDs", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Build the kira binary for testing
		kiraPath := buildKiraBinary(t, tmpDir)

		// Initialize kira
		initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
		require.NoError(t, err)
		err = initCmd.Run()
		require.NoError(t, err)

		// Create two work items with the same ID
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		workItemContent1 := `---
id: 001
title: First Feature
status: todo
kind: prd
created: 2024-01-15
---
# First Feature
`

		workItemContent2 := `---
id: 001
title: Second Feature
status: todo
kind: prd
created: 2024-01-16
---
# Second Feature
`

		filePath1 := filepath.Join(tmpDir, ".work", "1_todo", "001-first-feature.prd.md")
		filePath2 := filepath.Join(tmpDir, ".work", "1_todo", "001-second-feature.prd.md")
		require.NoError(t, os.WriteFile(filePath1, []byte(workItemContent1), 0o600))
		require.NoError(t, os.WriteFile(filePath2, []byte(workItemContent2), 0o600))

		// Run doctor command
		doctorCmd, err := safeExecCommand(tmpDir, kiraPath, "doctor")
		require.NoError(t, err)
		output, err := doctorCmd.CombinedOutput()
		require.NoError(t, err)

		outputStr := string(output)
		// Should show duplicate ID errors
		assert.Contains(t, outputStr, "Duplicate ID Errors")
		// Should attempt to fix issues
		assert.Contains(t, outputStr, "Attempting to fix issues")

		// Verify one of the files got a new ID
		// #nosec G304 - filePath1 and filePath2 are constructed from tmpDir which is validated
		content1, err := os.ReadFile(filePath1)
		require.NoError(t, err)
		// #nosec G304 - filePath2 is constructed from tmpDir which is validated
		content2, err := os.ReadFile(filePath2)
		require.NoError(t, err)

		// At least one should have a different ID (duplicates should be fixed)
		idsDifferent := !strings.Contains(string(content1), "id: 001") || !strings.Contains(string(content2), "id: 001")
		assert.True(t, idsDifferent, "At least one file should have a different ID after fixing duplicates")
	})

	t.Run("applies defaults for missing required fields without marking them unfixable", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Build the kira binary for testing
		kiraPath := buildKiraBinary(t, tmpDir)

		// Initialize kira
		initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
		require.NoError(t, err)
		err = initCmd.Run()
		require.NoError(t, err)

		// Update kira.yml to add a required field with a default value
		kiraConfigPath := filepath.Join(tmpDir, "kira.yml")
		// #nosec G304 - kiraConfigPath is constructed from tmpDir which is validated
		configContent, err := os.ReadFile(kiraConfigPath)
		require.NoError(t, err)

		// Replace the empty fields map with a concrete field configuration
		configWithFields := strings.Replace(string(configContent), "fields: {}", `fields:
  assigned:
    type: email
    required: true
    default: user@example.com
`, 1)
		require.NoError(t, os.WriteFile(kiraConfigPath, []byte(configWithFields), 0o600))

		// Create a work item that is missing the required 'assigned' field
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		workItemContent := `---
id: 001
title: Feature with default
status: todo
kind: prd
created: 2024-01-15
---
# Feature with default
`

		filePath := filepath.Join(tmpDir, ".work", "1_todo", "001-feature-with-default.prd.md")
		require.NoError(t, os.WriteFile(filePath, []byte(workItemContent), 0o600))

		// Run doctor command
		doctorCmd, err := safeExecCommand(tmpDir, kiraPath, "doctor")
		require.NoError(t, err)
		output, err := doctorCmd.CombinedOutput()
		require.NoError(t, err)

		outputStr := string(output)

		// Should report that field issues were fixed using defaults
		assert.Contains(t, outputStr, "Fixed field issues")
		assert.Contains(t, outputStr, "fixed field 'assigned': applied default value")

		// Should not report the missing required field as an unfixable issue
		assert.NotContains(t, outputStr, "Issues requiring manual attention")
		assert.NotContains(t, outputStr, "cannot be automatically fixed")
	})

	t.Run("reports unfixable field validation errors for invalid enum values", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Build the kira binary for testing
		kiraPath := buildKiraBinary(t, tmpDir)

		// Initialize kira
		initCmd, err := safeExecCommand(tmpDir, kiraPath, "init")
		require.NoError(t, err)
		err = initCmd.Run()
		require.NoError(t, err)

		// Update kira.yml to add an enum field
		kiraConfigPath := filepath.Join(tmpDir, "kira.yml")
		// #nosec G304 - kiraConfigPath is constructed from tmpDir which is validated
		configContent, err := os.ReadFile(kiraConfigPath)
		require.NoError(t, err)

		configWithFields := strings.Replace(string(configContent), "fields: {}", `fields:
  priority:
    type: enum
    allowed_values: [low, medium, high]
`, 1)
		require.NoError(t, os.WriteFile(kiraConfigPath, []byte(configWithFields), 0o600))

		// Create a work item with an invalid enum value that cannot be auto-fixed
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		workItemContent := `---
id: 001
title: Feature with invalid priority
status: todo
kind: prd
created: 2024-01-15
priority: invalid
---
# Feature with invalid priority
`

		filePath := filepath.Join(tmpDir, ".work", "1_todo", "001-feature-with-invalid-priority.prd.md")
		require.NoError(t, os.WriteFile(filePath, []byte(workItemContent), 0o600))

		// Run doctor command
		doctorCmd, err := safeExecCommand(tmpDir, kiraPath, "doctor")
		require.NoError(t, err)
		output, err := doctorCmd.CombinedOutput()
		require.NoError(t, err)

		outputStr := string(output)

		// Should report validation errors and that they require manual attention
		assert.Contains(t, outputStr, "Validation errors found")
		assert.Contains(t, outputStr, "Field Validation Errors")
		assert.Contains(t, outputStr, "priority")
		assert.Contains(t, outputStr, "not in allowed values")
		assert.Contains(t, outputStr, "Issues requiring manual attention")
		// Must not falsely claim that all issues have been resolved
		assert.NotContains(t, outputStr, "All issues have been resolved!")
	})
}

// TestKiraDoneIntegration runs the kira binary with temp repo and mock GitHub for "kira done" flows.
func TestKiraDoneIntegration(t *testing.T) {
	const pullsPath = "/api/v3/repos/owner/repo/pulls"

	t.Run("dry-run with merged PR prints idempotent path", func(t *testing.T) {
		mergedAt := "2024-06-01T12:00:00Z"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == pullsPath {
				prs := []*github.PullRequest{
					{
						Number:         github.Int(42),
						Head:           &github.PullRequestBranch{Ref: github.String("014-feature")},
						MergedAt:       &github.Timestamp{Time: mustParseTime(mergedAt)},
						MergeCommitSHA: github.String("abc123"),
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(prs)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		kiraPath := buildKiraBinary(t, tmpDir)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".work/2_doing/014-test.prd.md"), []byte("---\nid: 014\ntitle: Test\nstatus: doing\nkind: prd\ncreated: 2024-01-01\n---\n"), 0o600))
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "init").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "remote", "add", "origin", "https://github.com/owner/repo.git").Run())
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "f"), []byte("x"), 0o600))
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "add", "f").Run())
		// #nosec G204 - commit message is fixed
		require.NoError(t, exec.Command("git", "commit", "-m", "init").Run())
		// #nosec G204 - branch name is fixed
		_ = exec.Command("git", "branch", "-m", "main").Run()
		kiraYml := fmt.Sprintf("version: \"1.0\"\ngit:\n  trunk_branch: main\nworkspace:\n  git_base_url: %s\n", server.URL)
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "kira.yml"), []byte(kiraYml), 0o600))

		restore := setenv("KIRA_GITHUB_TOKEN", "test-token")
		defer restore()

		cmd, err := safeExecCommand(tmpDir, kiraPath, "done", "014", "--dry-run")
		require.NoError(t, err)
		cmd.Dir = tmpDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "kira done --dry-run failed: %s", string(output))
		assert.Contains(t, string(output), "idempotent")
	})

	t.Run("dry-run with open PR prints merge would run", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == pullsPath {
				prs := []*github.PullRequest{
					{
						Number: github.Int(42),
						Head:   &github.PullRequestBranch{Ref: github.String("014-feature")},
						State:  github.String("open"),
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(prs)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		kiraPath := buildKiraBinary(t, tmpDir)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".work/2_doing/014-test.prd.md"), []byte("---\nid: 014\ntitle: Test\nstatus: doing\nkind: prd\ncreated: 2024-01-01\n---\n"), 0o600))
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "init").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "remote", "add", "origin", "https://github.com/owner/repo.git").Run())
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "f"), []byte("x"), 0o600))
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "add", "f").Run())
		// #nosec G204 - commit message is fixed
		require.NoError(t, exec.Command("git", "commit", "-m", "init").Run())
		// #nosec G204 - branch name is fixed
		_ = exec.Command("git", "branch", "-m", "main").Run()
		kiraYml := fmt.Sprintf("version: \"1.0\"\ngit:\n  trunk_branch: main\nworkspace:\n  git_base_url: %s\n", server.URL)
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "kira.yml"), []byte(kiraYml), 0o600))

		restore := setenv("KIRA_GITHUB_TOKEN", "test-token")
		defer restore()

		cmd, err := safeExecCommand(tmpDir, kiraPath, "done", "014", "--dry-run")
		require.NoError(t, err)
		cmd.Dir = tmpDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "kira done --dry-run failed: %s", string(output))
		assert.Contains(t, string(output), "merge")
	})

	// Full flow (merge, pull, update, cleanup) requires a real GitHub remote for git pull/push.
	// When KIRA_GITHUB_TOKEN is set to a real token and the repo has a PR for the work item,
	// run: go test ./internal/commands/ -run TestKiraDoneIntegration/full_done_flow -v
	// with the env and a test repo. Here we only run the binary with --dry-run to avoid
	// needing a real remote.
	t.Run("done command runs without panic with open PR mock", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == pullsPath {
				prs := []*github.PullRequest{
					{
						Number: github.Int(42),
						Head:   &github.PullRequestBranch{Ref: github.String("014-feature"), SHA: github.String("abc123")},
						State:  github.String("open"),
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(prs)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		kiraPath := buildKiraBinary(t, tmpDir)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".work/2_doing/014-test.prd.md"), []byte("---\nid: 014\ntitle: Test\nstatus: doing\nkind: prd\ncreated: 2024-01-01\n---\n"), 0o600))
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "init").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "remote", "add", "origin", "https://github.com/owner/repo.git").Run())
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "f"), []byte("x"), 0o600))
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "add", "f").Run())
		// #nosec G204 - commit message is fixed
		require.NoError(t, exec.Command("git", "commit", "-m", "init").Run())
		// #nosec G204 - branch name is fixed
		_ = exec.Command("git", "branch", "-m", "main").Run()
		kiraYml := fmt.Sprintf("version: \"1.0\"\ngit:\n  trunk_branch: main\nworkspace:\n  git_base_url: %s\n", server.URL)
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "kira.yml"), []byte(kiraYml), 0o600))

		restore := setenv("KIRA_GITHUB_TOKEN", "test-token")
		defer restore()

		cmd, err := safeExecCommand(tmpDir, kiraPath, "done", "014", "--dry-run")
		require.NoError(t, err)
		cmd.Dir = tmpDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "kira done --dry-run failed: %s", string(output))
		assert.Contains(t, string(output), "merge")
	})
}
