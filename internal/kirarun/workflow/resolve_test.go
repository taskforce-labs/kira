package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"kira/internal/config"
)

func TestResolveNamed(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".workflows"), 0o700))
	wf := filepath.Join(root, ".workflows", "hello.go")
	require.NoError(t, os.WriteFile(wf, []byte("package main\n"), 0o600))

	cfg := &config.Config{
		ConfigDir: root,
		Workflows: &config.WorkflowsConfig{
			Root: ".workflows",
			Scripts: map[string]string{
				"hello": "hello.go",
			},
		},
	}
	abs, name, err := Resolve(cfg, "hello")
	require.NoError(t, err)
	require.Equal(t, wf, abs)
	require.Equal(t, "hello", name)
}

func TestResolveBareName(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".workflows"), 0o700))
	wf := filepath.Join(root, ".workflows", "foo.go")
	require.NoError(t, os.WriteFile(wf, []byte("package main\n"), 0o600))

	cfg := &config.Config{ConfigDir: root}
	abs, name, err := Resolve(cfg, "foo")
	require.NoError(t, err)
	require.Equal(t, wf, abs)
	require.Equal(t, "foo", name)
}
