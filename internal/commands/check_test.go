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

		output, err := captureStdout(func() error { return runCheckRun(cfg) })
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
		err := runCheckRun(cfg)
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
		err := runCheckRun(cfg)
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
		err := runCheckRun(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "first")
		// Second check should not have run (we can't easily verify without side effects)
	})
}

func TestRunCheckList(t *testing.T) {
	t.Run("empty checks prints no checks message", func(t *testing.T) {
		cfg := &config.Config{Checks: []config.CheckEntry{}}

		output, err := captureStdout(func() error { return runCheckList(cfg) })
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

		output, err := captureStdout(func() error { return runCheckList(cfg) })
		assert.NoError(t, err)
		assert.Contains(t, output, "lint")
		assert.Contains(t, output, "Run linter")
		assert.Contains(t, output, "test")
		// Empty description prints as "-"
		assert.Contains(t, output, "-")
	})
}
