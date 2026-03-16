package roadmap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_FlatStructure(t *testing.T) {
	yaml := `
roadmap:
  - id: AUTH-001
  - title: OAuth provider spike
    meta:
      period: Q1-26
      workstream: auth
      owner: wayne
      outcome: decision
`
	f, err := Parse([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, f.Roadmap, 2)
	assert.True(t, f.Roadmap[0].IsWorkItemRef())
	assert.Equal(t, "AUTH-001", f.Roadmap[0].ID)
	assert.True(t, f.Roadmap[1].IsAdHoc())
	assert.Equal(t, "OAuth provider spike", f.Roadmap[1].Title)
	assert.Equal(t, "Q1-26", metaStr(f.Roadmap[1].Meta, "period"))
	assert.Equal(t, "auth", metaStr(f.Roadmap[1].Meta, "workstream"))
}

func TestParse_HierarchicalStructure(t *testing.T) {
	yaml := `
roadmap:
  - group: Authentication Workstream
    meta:
      period: Q1-26
      workstream: auth
      owner: wayne
    items:
      - group: OAuth Integration Epic
        meta:
          epic: oauth
        items:
          - id: AUTH-001
          - title: OAuth provider spike
            meta:
              owner: wayne
              outcome: decision
          - title: OAuth token refresh
            meta:
              tags: [security, oauth]
      - group: SSO Features
        meta:
          epic: sso
        items:
          - id: AUTH-002
          - title: SAML integration
            meta:
              depends_on: [AUTH-001]
  - group: Platform Migration
    meta:
      period: Q2-26
      workstream: platform
    items:
      - id: PLAT-002
        meta:
          tags: [migration, infrastructure]
`
	f, err := Parse([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, f.Roadmap, 2)
	assert.True(t, f.Roadmap[0].IsGroup())
	assert.Equal(t, "Authentication Workstream", f.Roadmap[0].Group)
	require.Len(t, f.Roadmap[0].Items, 2)
	assert.Equal(t, "OAuth Integration Epic", f.Roadmap[0].Items[0].Group)
	require.Len(t, f.Roadmap[0].Items[0].Items, 3)
	assert.Equal(t, "AUTH-001", f.Roadmap[0].Items[0].Items[0].ID)
	assert.Equal(t, "OAuth provider spike", f.Roadmap[0].Items[0].Items[1].Title)
	assert.Equal(t, "AUTH-002", f.Roadmap[0].Items[1].Items[0].ID)
	assert.Equal(t, "Platform Migration", f.Roadmap[1].Group)
	assert.Equal(t, "PLAT-002", f.Roadmap[1].Items[0].ID)
}

func TestParse_NestedWorkItems(t *testing.T) {
	yaml := `
roadmap:
  - id: EPIC-001
    items:
      - id: FEAT-001
        items:
          - id: TASK-001
          - title: Ad-hoc task
      - id: FEAT-002
  - id: EPIC-002
    items:
      - title: New feature (ad-hoc, will be promoted)
`
	f, err := Parse([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, f.Roadmap, 2)
	assert.Equal(t, "EPIC-001", f.Roadmap[0].ID)
	require.Len(t, f.Roadmap[0].Items, 2)
	assert.Equal(t, "FEAT-001", f.Roadmap[0].Items[0].ID)
	require.Len(t, f.Roadmap[0].Items[0].Items, 2)
	assert.Equal(t, "TASK-001", f.Roadmap[0].Items[0].Items[0].ID)
	assert.Equal(t, "Ad-hoc task", f.Roadmap[0].Items[0].Items[1].Title)
	assert.Equal(t, "FEAT-002", f.Roadmap[0].Items[1].ID)
	assert.Equal(t, "EPIC-002", f.Roadmap[1].ID)
	assert.Equal(t, "New feature (ad-hoc, will be promoted)", f.Roadmap[1].Items[0].Title)
}

func TestRoundTrip_Flat(t *testing.T) {
	yaml := `
roadmap:
  - id: AUTH-001
  - title: OAuth provider spike
    meta:
      period: Q1-26
      workstream: auth
`
	f, err := Parse([]byte(yaml))
	require.NoError(t, err)
	data, err := Serialize(f)
	require.NoError(t, err)
	f2, err := Parse(data)
	require.NoError(t, err)
	assert.Equal(t, f.Roadmap[0].ID, f2.Roadmap[0].ID)
	assert.Equal(t, f.Roadmap[1].Title, f2.Roadmap[1].Title)
	assert.Equal(t, metaStr(f.Roadmap[1].Meta, "period"), metaStr(f2.Roadmap[1].Meta, "period"))
}

func TestRoundTrip_Hierarchical(t *testing.T) {
	yaml := `
roadmap:
  - group: Auth
    meta:
      workstream: auth
    items:
      - id: AUTH-001
      - title: Spike
        meta:
          owner: wayne
`
	f, err := Parse([]byte(yaml))
	require.NoError(t, err)
	data, err := Serialize(f)
	require.NoError(t, err)
	f2, err := Parse(data)
	require.NoError(t, err)
	require.Len(t, f2.Roadmap, 1)
	assert.Equal(t, "Auth", f2.Roadmap[0].Group)
	require.Len(t, f2.Roadmap[0].Items, 2)
	assert.Equal(t, "AUTH-001", f2.Roadmap[0].Items[0].ID)
	assert.Equal(t, "Spike", f2.Roadmap[0].Items[1].Title)
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ROADMAP.yml")
	err := os.WriteFile(path, []byte("roadmap:\n  - id: 001\n"), 0o600)
	require.NoError(t, err)
	f, err := LoadFile(dir, path)
	require.NoError(t, err)
	require.Len(t, f.Roadmap, 1)
	assert.Equal(t, "001", f.Roadmap[0].ID)
}

// TestLoadFile_OutsideDir ensures LoadFile rejects paths outside baseDir.
func TestLoadFile_OutsideDir(t *testing.T) {
	baseDir := t.TempDir()
	// Path under a different tree (e.g. /etc on Unix)
	absPath := filepath.Join(string(filepath.Separator), "etc", "passwd")
	if os.PathSeparator != '/' {
		absPath = filepath.Join(string(filepath.Separator), "nonexistent_roadmap_xxx")
	}
	_, err := LoadFile(baseDir, absPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside")
}

func TestSaveFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ROADMAP.yml")
	f := &File{
		Roadmap: []Entry{{ID: "001"}},
	}
	err := SaveFile(dir, path, f)
	require.NoError(t, err)
	// #nosec G304 - path is under dir (t.TempDir()) and validated by SaveFile
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	f2, err := Parse(data)
	require.NoError(t, err)
	require.Len(t, f2.Roadmap, 1)
	assert.Equal(t, "001", f2.Roadmap[0].ID)
}

func metaStr(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}
