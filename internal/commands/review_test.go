package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kira/internal/config"
)

func TestParseWorkItemIDFromBranch(t *testing.T) {
	t.Run("valid kira branch returns ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		id, err := parseWorkItemIDFromBranch("012-submit-for-review", cfg)
		require.NoError(t, err)
		assert.Equal(t, "012", id)
	})

	t.Run("branch without hyphen returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		_, err = parseWorkItemIDFromBranch("main", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match kira branch format")
	})

	t.Run("branch with invalid ID format returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		cfg, err := config.LoadConfig()
		require.NoError(t, err)
		// Default IDFormat is ^\d{3}$ so "abc" is invalid
		_, err = parseWorkItemIDFromBranch("abc-foo-bar", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match kira branch format")
	})
}

func TestStatusFromWorkItemPath(t *testing.T) {
	t.Run("path in doing folder returns doing", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		path := filepath.Join(".work", "2_doing", "012-foo.prd.md")
		require.NoError(t, os.WriteFile(path, []byte("---\nid: 012\n---\n"), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		status, err := statusFromWorkItemPath(path, cfg)
		require.NoError(t, err)
		assert.Equal(t, "doing", status)
	})

	t.Run("path in review folder returns review", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/3_review", 0o700))
		path := filepath.Join(".work", "3_review", "012-foo.prd.md")
		require.NoError(t, os.WriteFile(path, []byte("---\nid: 012\n---\n"), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		status, err := statusFromWorkItemPath(path, cfg)
		require.NoError(t, err)
		assert.Equal(t, "review", status)
	})

	t.Run("path outside work dir returns empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		status, err := statusFromWorkItemPath("/tmp/other/file.md", cfg)
		require.NoError(t, err)
		assert.Equal(t, "", status)
	})
}

const reviewTestWorkItemPath = ".work/2_doing/012-foo.prd.md"

func TestCheckSliceReadiness(t *testing.T) {
	t.Run("no slices is ready", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		path := reviewTestWorkItemPath
		content := `---
id: 012
title: Foo
status: doing
kind: prd
---

# Foo
`
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		err = checkSliceReadiness(path, cfg)
		require.NoError(t, err)
	})

	t.Run("open tasks returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		path := reviewTestWorkItemPath
		content := `---
id: 012
title: Foo
status: doing
kind: prd
---

# Foo

## Slices

### SliceA
- [ ] T001: task one
`
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		err = checkSliceReadiness(path, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "open task(s)")
	})

	t.Run("all tasks done is ready", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		path := reviewTestWorkItemPath
		content := `---
id: 012
title: Foo
status: doing
kind: prd
---

# Foo

## Slices

### SliceA
- [x] T001: task one
`
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		err = checkSliceReadiness(path, cfg)
		require.NoError(t, err)
	})
}

func TestRunReviewDryRun(t *testing.T) {
	t.Run("dry-run runs validation and prints planned steps", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Git repo with main and branch 012-foo
		// #nosec G204 - tmpDir is from t.TempDir(), safe for test use
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, os.WriteFile("f", []byte("x"), 0o600))
		// #nosec G204 - args are fixed test strings
		require.NoError(t, exec.Command("git", "add", "f").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "init").Run())
		// #nosec G204 - branch name is fixed test string
		require.NoError(t, exec.Command("git", "checkout", "-b", "012-foo").Run())

		// Work item 012 in doing
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		wiContent := `---
id: 012
title: Foo
status: doing
kind: prd
---

# Foo
`
		require.NoError(t, os.WriteFile(".work/2_doing/012-foo.prd.md", []byte(wiContent), 0o600))

		rootCmd.SetArgs([]string{"review", "--dry-run"})
		err := rootCmd.Execute()
		require.NoError(t, err)
	})
}
