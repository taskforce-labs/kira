package roadmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectWorkItemIDs(t *testing.T) {
	f, err := Parse([]byte(`
roadmap:
  - id: AUTH-001
  - group: X
    items:
      - id: "002"
      - title: ad-hoc
`))
	require.NoError(t, err)
	ids := CollectWorkItemIDs(f)
	assert.ElementsMatch(t, []string{"AUTH-001", "002"}, ids)
}

func TestCollectAdHoc(t *testing.T) {
	f, err := Parse([]byte(`
roadmap:
  - id: "001"
  - title: First ad-hoc
    meta:
      period: Q1
  - group: G
    items:
      - title: Nested ad-hoc
`))
	require.NoError(t, err)
	refs := CollectAdHoc(f)
	require.Len(t, refs, 2)
	assert.Equal(t, "First ad-hoc", refs[0].Title)
	assert.Equal(t, "Nested ad-hoc", refs[1].Title)
	assert.Contains(t, refs[0].Path, "roadmap")
	assert.Contains(t, refs[1].Path, "items")
}

func TestCollectDependsOn(t *testing.T) {
	f, err := Parse([]byte(`
roadmap:
  - id: A
    meta:
      depends_on: [B, C]
  - id: B
    meta:
      depends_on: [C]
  - title: no-id
    meta:
      depends_on: [A]
`))
	require.NoError(t, err)
	refs := CollectDependsOn(f)
	require.Len(t, refs, 2) // A and B; title-only has no id so not included
	assert.Equal(t, "A", refs[0].ID)
	assert.Equal(t, []string{"B", "C"}, refs[0].DependsOn)
	assert.Equal(t, "B", refs[1].ID)
	assert.Equal(t, []string{"C"}, refs[1].DependsOn)
}

func TestDependencyCycle_NoCycle(t *testing.T) {
	f, err := Parse([]byte(`
roadmap:
  - id: A
    meta:
      depends_on: [B]
  - id: B
    meta:
      depends_on: [C]
  - id: C
`))
	require.NoError(t, err)
	known := map[string]bool{"A": true, "B": true, "C": true}
	cycle := DependencyCycle(f, known)
	assert.Nil(t, cycle)
}

func TestDependencyCycle_HasCycle(t *testing.T) {
	f, err := Parse([]byte(`
roadmap:
  - id: A
    meta:
      depends_on: [B]
  - id: B
    meta:
      depends_on: [C]
  - id: C
    meta:
      depends_on: [A]
`))
	require.NoError(t, err)
	known := map[string]bool{"A": true, "B": true, "C": true}
	cycle := DependencyCycle(f, known)
	require.NotNil(t, cycle)
	assert.Len(t, cycle, 4) // e.g. A -> B -> C -> A
	assert.Contains(t, cycle, "A")
	assert.Contains(t, cycle, "B")
	assert.Contains(t, cycle, "C")
}
