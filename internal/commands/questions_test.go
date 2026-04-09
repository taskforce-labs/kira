package commands

import (
	"os"
	"path/filepath"
	"testing"

	"kira/internal/config"

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

func TestBuildQuestionRecords(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "wi.md")
	content := `## Questions

### 1. Open Q

#### Options
- [ ] A
`
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	cfg := &config.Config{ConfigDir: tmp}
	recs, err := buildQuestionRecords(cfg, []string{p})
	require.NoError(t, err)
	require.Len(t, recs, 1)
	assert.Equal(t, "wi.md", recs[0].File)
	assert.Contains(t, recs[0].Question, "Open Q")
}
