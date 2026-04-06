package runner

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"kira/internal/kirarun/session"
)

func TestDeriveRunID(t *testing.T) {
	ts := time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)
	id, err := DeriveRunID("hello.go", ts)
	require.NoError(t, err)
	require.Equal(t, "hello-20060102150405", id)
}

func TestExecuteSuccess(t *testing.T) {
	root := t.TempDir()
	wf := filepath.Join("testdata", "ok.go")
	absWF, err := filepath.Abs(wf)
	require.NoError(t, err)

	var buf bytes.Buffer
	cfg := &Config{
		ProjectRoot:  root,
		WorkflowPath: absWF,
		KiraVersion:  "test",
		Stdout:       &buf,
	}
	_, err = cfg.Execute(context.Background())
	require.NoError(t, err)
	require.Contains(t, buf.String(), "run id:")

	sessionsDir, err := session.SessionsDir(root)
	require.NoError(t, err)
	entries, err := os.ReadDir(sessionsDir)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestExecuteFailNoRetry(t *testing.T) {
	root := t.TempDir()
	wf := filepath.Join("testdata", "fail.go")
	absWF, err := filepath.Abs(wf)
	require.NoError(t, err)

	cfg := &Config{
		ProjectRoot:  root,
		WorkflowPath: absWF,
		KiraVersion:  "test",
		Stdout:       io.Discard,
	}
	_, err = cfg.Execute(context.Background())
	require.Error(t, err)

	sessDir, err := session.SessionsDir(root)
	require.NoError(t, err)
	entries, err := os.ReadDir(sessDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
}
