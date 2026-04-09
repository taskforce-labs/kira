package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverMarkdownFiles(t *testing.T) {
	tmp := t.TempDir()
	work := filepath.Join(tmp, ".work")
	docs := filepath.Join(tmp, ".docs")
	require.NoError(t, os.MkdirAll(filepath.Join(work, "2_doing"), 0o750))
	require.NoError(t, os.MkdirAll(docs, 0o750))

	require.NoError(t, os.WriteFile(filepath.Join(work, "2_doing", "a.prd.md"), []byte("# x\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(docs, "README.md"), []byte("# x\n"), 0o600))

	paths, err := discoverMarkdownFiles(work, docs)
	require.NoError(t, err)
	require.Len(t, paths, 2)
}

func TestValidatePathUnderRoot(t *testing.T) {
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "nest")
	require.NoError(t, os.MkdirAll(sub, 0o750))
	good := filepath.Join(sub, "f.md")
	require.NoError(t, os.WriteFile(good, []byte("x"), 0o600))
	require.NoError(t, validatePathUnderRoot(tmp, good))
	assert.Error(t, validatePathUnderRoot(tmp, filepath.Join(tmp, "..", "outside.md")))
}

func TestIsMarkdownFile(t *testing.T) {
	assert.True(t, isMarkdownFile("x.md"))
	assert.True(t, isMarkdownFile("x.qmd"))
	assert.False(t, isMarkdownFile("x.txt"))
}
