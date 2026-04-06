package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSessionsDir(t *testing.T) {
	root := t.TempDir()
	dir, err := SessionsDir(root)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, ".workflows", "sessions"), dir)
}

func TestFilePath(t *testing.T) {
	root := t.TempDir()
	p, err := FilePath(root, "hello-20060102-150405")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, ".workflows", "sessions", "hello-20060102-150405.yml"), p)
}

func TestLoadHappy(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "run-abc.yml")
	script := filepath.Join(tmp, "w.go")
	content := `path: ` + script + `
kira-version: 1.0.0
run-id: run-abc
attempt: 1
attempts: []
steps: []
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	s, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, script, s.Path)
	require.Equal(t, "run-abc", s.RunID)
	require.Equal(t, 1, s.Attempt)
}

func TestLoadInvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "x.yml")
	require.NoError(t, os.WriteFile(path, []byte(":\n  bad"), 0o600))
	_, err := Load(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), path)
	require.Contains(t, err.Error(), "invalid YAML")
}

func TestLoadWrongSchema(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "run-abc.yml")
	// run-id does not match stem
	content := `path: /abs/script.go
kira-version: 1.0.0
run-id: other-id
attempt: 1
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	_, err := Load(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid session")
	require.Contains(t, err.Error(), "filename stem")
}

func TestLoadRelativePathField(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "run-abc.yml")
	content := `path: relative.go
kira-version: 1.0.0
run-id: run-abc
attempt: 1
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	_, err := Load(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "path must be absolute")
}

func TestSaveAndRemove(t *testing.T) {
	root := t.TempDir()
	s := &Session{
		Path:        filepath.Join(root, "wf.go"),
		KiraVersion: "0.1.0",
		RunID:       "myrun-20060102-150405",
		Attempt:     1,
	}
	require.NoError(t, Save(root, s))

	sessionPath, err := FilePath(root, s.RunID)
	require.NoError(t, err)
	_, err = Load(sessionPath)
	require.NoError(t, err)

	require.NoError(t, Remove(sessionPath))
	_, err = os.Stat(sessionPath)
	require.True(t, os.IsNotExist(err))
}

func TestTryLockContention(t *testing.T) {
	tmp := t.TempDir()
	lockPath := filepath.Join(tmp, "run.lock")

	l1, err := TryLock(lockPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l1.Unlock() })

	_, err = TryLock(lockPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "another process")
}

func TestAttemptRecordTimestamps(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "run-abc.yml")
	ts := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	content := `path: ` + filepath.Join(tmp, "s.go") + `
kira-version: 1.0.0
run-id: run-abc
attempt: 1
attempts:
  - attempt: 1
    name: run
    started_at: "` + ts + `"
    failed_at: "` + ts + `"
    error:
      message: boom
steps: []
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	s, err := Load(path)
	require.NoError(t, err)
	require.Len(t, s.Attempts, 1)
	require.Equal(t, "boom", s.Attempts[0].Error["message"])
}
