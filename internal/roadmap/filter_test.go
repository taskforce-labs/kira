package roadmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePathFilter_Simple(t *testing.T) {
	segs, err := ParsePathFilter("workstream:auth/epic:oauth")
	require.NoError(t, err)
	require.Len(t, segs, 2)
	assert.Equal(t, "workstream", segs[0].Key)
	assert.Equal(t, "auth", segs[0].Value)
	assert.Equal(t, "epic", segs[1].Key)
	assert.Equal(t, "oauth", segs[1].Value)
}

func TestParsePathFilter_Quoted(t *testing.T) {
	segs, err := ParsePathFilter(`workstream:"auth/sso"`)
	require.NoError(t, err)
	require.Len(t, segs, 1)
	assert.Equal(t, "auth/sso", segs[0].Value)
}

func TestParsePathFilter_Empty(t *testing.T) {
	segs, err := ParsePathFilter("")
	require.NoError(t, err)
	assert.Nil(t, segs)
}

func TestMatchMeta(t *testing.T) {
	f := &Filter{Period: "Q1-26", Workstream: "auth"}
	assert.True(t, f.MatchMeta(map[string]interface{}{"period": "Q1-26", "workstream": "auth"}))
	assert.False(t, f.MatchMeta(map[string]interface{}{"period": "Q2-26", "workstream": "auth"}))
	assert.False(t, f.MatchMeta(map[string]interface{}{"period": "Q1-26", "workstream": "platform"}))
	f2 := &Filter{}
	assert.True(t, f2.MatchMeta(nil))
}

func TestBuildPath_PrefixMatch(t *testing.T) {
	ancestors := []map[string]interface{}{
		{"workstream": "auth"},
		{"epic": "oauth"},
	}
	entryMeta := map[string]interface{}{"owner": "wayne"}
	filterPath := []PathSegment{
		{Key: "workstream", Value: "auth"},
		{Key: "epic", Value: "oauth"},
	}
	path := BuildPath(ancestors, entryMeta, filterPath)
	require.Len(t, path, 2)
	assert.Equal(t, "auth", path[0].Value)
	assert.Equal(t, "oauth", path[1].Value)
	assert.True(t, PathMatches(path, filterPath))
}

func TestPathMatches_Prefix(t *testing.T) {
	entryPath := []PathSegment{
		{Key: "workstream", Value: "auth"},
		{Key: "epic", Value: "oauth"},
		{Key: "owner", Value: "wayne"},
	}
	filterPath := []PathSegment{
		{Key: "workstream", Value: "auth"},
		{Key: "epic", Value: "oauth"},
	}
	assert.True(t, PathMatches(entryPath, filterPath))
}

func TestPathMatches_NoMatch(t *testing.T) {
	entryPath := []PathSegment{{Key: "workstream", Value: "platform"}}
	filterPath := []PathSegment{{Key: "workstream", Value: "auth"}}
	assert.False(t, PathMatches(entryPath, filterPath))
}

func TestSelectEntries_NoFilter(t *testing.T) {
	f, _ := Parse([]byte("roadmap:\n  - id: \"001\"\n  - title: X\n"))
	out := SelectEntries(f, nil)
	assert.Len(t, out, 2)
}

func TestSelectEntries_MetaFilter(t *testing.T) {
	f, _ := Parse([]byte(`
roadmap:
  - id: A
    meta:
      period: Q1
      workstream: auth
  - title: B
    meta:
      period: Q2
  - group: G
    meta:
      workstream: auth
    items:
      - id: C
        meta:
          period: Q1
`))
	flt := &Filter{Period: "Q1", Workstream: "auth"}
	out := SelectEntries(f, flt)
	// A matches (Q1, auth); B no (Q2); C matches (under G with auth, self Q1)
	assert.GreaterOrEqual(t, len(out), 1)
	var ids []string
	for _, e := range out {
		if e.ID != "" {
			ids = append(ids, e.ID)
		}
	}
	assert.Contains(t, ids, "A")
	assert.Contains(t, ids, "C")
}

func TestSelectEntries_PathFilter(t *testing.T) {
	f, _ := Parse([]byte(`
roadmap:
  - group: Auth
    meta:
      workstream: auth
    items:
      - group: OAuth
        meta:
          epic: oauth
        items:
          - id: "001"
          - title: spike
`))
	pathSegs, _ := ParsePathFilter("workstream:auth/epic:oauth")
	flt := &Filter{Path: pathSegs}
	out := SelectEntries(f, flt)
	// 001 and spike are under workstream:auth and epic:oauth
	assert.GreaterOrEqual(t, len(out), 1)
}
