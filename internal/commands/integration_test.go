package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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
		assert.Contains(t, string(output), "No duplicate IDs found")
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
