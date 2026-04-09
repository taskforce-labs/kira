package commands

import (
	"os"
	"path/filepath"
	"testing"

	"kira/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandCommaSeparated(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, expandCommaSeparated([]string{"a,b"}))
	assert.Equal(t, []string{"doing", "backlog"}, expandCommaSeparated([]string{"doing", "backlog"}))
}

func TestDeriveDocType(t *testing.T) {
	typed, typ := deriveDocType("039-kira-questions.prd.md")
	assert.True(t, typed)
	assert.Equal(t, "prd", typ)

	typed, _ = deriveDocType("README.md")
	assert.False(t, typed)

	typed, typ = deriveDocType("x.adr.md")
	assert.True(t, typed)
	assert.Equal(t, "adr", typ)

	typed, _ = deriveDocType("report.qmd")
	assert.False(t, typed)
}

func TestIncludeByDocType(t *testing.T) {
	assert.True(t, includeByDocType("a.prd.md", nil, false, false))

	assert.True(t, includeByDocType("a.prd.md", []string{"prd"}, false, true))
	assert.False(t, includeByDocType("README.md", []string{"prd"}, false, true))

	assert.True(t, includeByDocType("README.md", []string{}, true, true))

	assert.True(t, includeByDocType("a.adr.md", []string{"adr"}, true, true))
	assert.True(t, includeByDocType("README.md", []string{"adr"}, true, true))
	assert.False(t, includeByDocType("a.prd.md", []string{"adr"}, true, true))
}

func TestValidateStatusValues(t *testing.T) {
	cfg := &config.DefaultConfig
	require.NoError(t, validateStatusValues(cfg, []string{"doing", "backlog"}))
	assert.Error(t, validateStatusValues(cfg, []string{"nope"}))
}

func TestCollectQuestionFiles(t *testing.T) {
	tmp := t.TempDir()
	work := filepath.Join(tmp, ".work")
	docs := filepath.Join(tmp, ".docs")
	require.NoError(t, os.MkdirAll(filepath.Join(work, "2_doing"), 0o750))
	require.NoError(t, os.MkdirAll(docs, 0o750))

	require.NoError(t, os.WriteFile(filepath.Join(work, "2_doing", "a.prd.md"), []byte("# x\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(docs, "README.md"), []byte("# x\n"), 0o600))

	cfg := &config.Config{
		ConfigDir:     tmp,
		StatusFolders: config.DefaultConfig.StatusFolders,
		Validation:    config.DefaultConfig.Validation,
	}

	paths, err := collectQuestionFiles(cfg, work, docs, questionsRunOpts{
		searchWork: true, searchDocs: true, docTypeFilterOn: false,
	})
	require.NoError(t, err)
	require.Len(t, paths, 2)

	paths, err = collectQuestionFiles(cfg, work, docs, questionsRunOpts{
		searchWork: true, searchDocs: false, statuses: []string{"doing"}, docTypeFilterOn: false,
	})
	require.NoError(t, err)
	require.Len(t, paths, 1)
	assert.Contains(t, paths[0], "a.prd.md")

	paths, err = collectQuestionFiles(cfg, work, docs, questionsRunOpts{
		searchWork: true, searchDocs: true, docTypes: []string{"prd"}, docTypeFilterOn: true,
	})
	require.NoError(t, err)
	require.Len(t, paths, 1)

	paths, err = collectQuestionFiles(cfg, work, docs, questionsRunOpts{
		searchWork: true, searchDocs: true, noDocType: true, docTypeFilterOn: true,
	})
	require.NoError(t, err)
	require.Len(t, paths, 1)
	assert.Contains(t, paths[0], "README.md")
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
