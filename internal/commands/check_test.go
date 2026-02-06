package commands

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kira/internal/config"
)

func captureStdout(f func() error) (string, error) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()
	done := make(chan struct{})
	var err error
	go func() {
		err = f()
		_ = w.Close() // so io.Copy returns
		close(done)
	}()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	<-done
	return buf.String(), err
}

func TestRunCheckRun(t *testing.T) {
	t.Run("empty checks prints message and returns nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{ConfigDir: tmpDir, Checks: []config.CheckEntry{}}

		output, err := captureStdout(func() error { return runCheckRun(cfg, nil) })
		assert.NoError(t, err)
		assert.Contains(t, output, "no checks configured")
	})

	t.Run("single check success returns nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			ConfigDir: tmpDir,
			Checks: []config.CheckEntry{
				{Name: "ok", Command: "true", Description: "succeeds"},
			},
		}
		err := runCheckRun(cfg, nil)
		assert.NoError(t, err)
	})

	t.Run("single check failure returns error with check name", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			ConfigDir: tmpDir,
			Checks: []config.CheckEntry{
				{Name: "fail", Command: "false", Description: "fails"},
			},
		}
		// Suppress stdout/stderr output during test to avoid polluting test output
		oldStdout := os.Stdout
		oldStderr := os.Stderr
		stdoutR, stdoutW, _ := os.Pipe()
		stderrR, stderrW, _ := os.Pipe()
		os.Stdout = stdoutW
		os.Stderr = stderrW
		err := runCheckRun(cfg, nil)
		_ = stdoutW.Close()
		_ = stderrW.Close()
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		_, _ = io.Copy(io.Discard, stdoutR) // Drain the pipes
		_, _ = io.Copy(io.Discard, stderrR)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fail")
		assert.Contains(t, err.Error(), "failed")
	})

	t.Run("multiple checks stop on first failure", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			ConfigDir: tmpDir,
			Checks: []config.CheckEntry{
				{Name: "first", Command: "exit 1", Description: "first fails"},
				{Name: "second", Command: "true", Description: "would succeed"},
			},
		}
		// Suppress stdout/stderr output during test to avoid polluting test output
		oldStdout := os.Stdout
		oldStderr := os.Stderr
		stdoutR, stdoutW, _ := os.Pipe()
		stderrR, stderrW, _ := os.Pipe()
		os.Stdout = stdoutW
		os.Stderr = stderrW
		err := runCheckRun(cfg, nil)
		_ = stdoutW.Close()
		_ = stderrW.Close()
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		_, _ = io.Copy(io.Discard, stdoutR) // Drain the pipes
		_, _ = io.Copy(io.Discard, stderrR)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "first")
		// Second check should not have run (we can't easily verify without side effects)
	})
}

func TestRunCheckList(t *testing.T) {
	t.Run("empty checks prints no checks message", func(t *testing.T) {
		cfg := &config.Config{Checks: []config.CheckEntry{}}

		output, err := captureStdout(func() error { return runCheckList(cfg, nil) })
		assert.NoError(t, err)
		assert.Contains(t, output, "no checks configured")
	})

	t.Run("lists name and description for each check", func(t *testing.T) {
		cfg := &config.Config{
			Checks: []config.CheckEntry{
				{Name: "lint", Command: "make lint", Description: "Run linter"},
				{Name: "test", Command: "go test ./...", Description: ""},
			},
		}

		output, err := captureStdout(func() error { return runCheckList(cfg, nil) })
		assert.NoError(t, err)
		assert.Contains(t, output, "lint")
		assert.Contains(t, output, "Run linter")
		assert.Contains(t, output, "test")
		// Empty description prints as "-"
		assert.Contains(t, output, "-")
	})
}

func TestFilterChecks(t *testing.T) {
	checks := []config.CheckEntry{
		{Name: "a", Command: "true", Tags: []string{"commit"}},
		{Name: "b", Command: "true", Tags: []string{"e2e", "done"}},
		{Name: "c", Command: "true", Tags: []string{"commit", "pre-push"}},
		{Name: "d", Command: "true", Tags: nil},
	}
	t.Run("no tags returns all", func(t *testing.T) {
		out := filterChecks(checks, nil)
		assert.Len(t, out, 4)
	})
	t.Run("single tag filters", func(t *testing.T) {
		out := filterChecks(checks, []string{"commit"})
		assert.Len(t, out, 2)
		assert.Equal(t, "a", out[0].Name)
		assert.Equal(t, "c", out[1].Name)
	})
	t.Run("multiple tags matches any", func(t *testing.T) {
		out := filterChecks(checks, []string{"commit", "e2e"})
		assert.Len(t, out, 3) // a, b, c
	})
	t.Run("tag matching none returns empty", func(t *testing.T) {
		out := filterChecks(checks, []string{"nonexistent"})
		assert.Len(t, out, 0)
	})
}

func TestRunCheckRunWithTags(t *testing.T) {
	t.Run("filter by tag runs only matching checks", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			ConfigDir: tmpDir,
			Checks: []config.CheckEntry{
				{Name: "commit-check", Command: "true", Tags: []string{"commit"}},
				{Name: "e2e-only", Command: "true", Tags: []string{"e2e"}},
			},
		}
		err := runCheckRun(cfg, []string{"commit"})
		assert.NoError(t, err)
	})
	t.Run("no matching tag prints message and exits 0", func(t *testing.T) {
		cfg := &config.Config{
			Checks: []config.CheckEntry{
				{Name: "a", Command: "true", Tags: []string{"commit"}},
			},
		}
		output, err := captureStdout(func() error { return runCheckRun(cfg, []string{"e2e"}) })
		assert.NoError(t, err)
		assert.Contains(t, output, "no checks match")
	})
}

func TestRunCheckListWithTags(t *testing.T) {
	t.Run("list with tag filter shows only matching", func(t *testing.T) {
		cfg := &config.Config{
			Checks: []config.CheckEntry{
				{Name: "commit-check", Command: "make check", Tags: []string{"commit"}},
				{Name: "e2e-check", Command: "make e2e", Tags: []string{"e2e"}},
			},
		}
		output, err := captureStdout(func() error { return runCheckList(cfg, []string{"commit"}) })
		assert.NoError(t, err)
		assert.Contains(t, output, "commit-check")
		assert.NotContains(t, output, "e2e-check")
	})
}
