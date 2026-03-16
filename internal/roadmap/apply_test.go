package roadmap

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyPromotions_DryRun(t *testing.T) {
	f, _ := Parse([]byte(`
roadmap:
  - id: A
  - title: Ad-hoc one
    meta:
      period: Q1
  - group: G
    meta:
      workstream: auth
    items:
      - title: Nested ad-hoc
`))
	flt := &Filter{Workstream: "auth"}
	promoteCount := 0
	promote := func(_ map[string]interface{}, _ string) (string, error) {
		promoteCount++
		return "001", nil
	}
	errs := ApplyPromotions(f, flt, promote)
	assert.Empty(t, errs)
	// Only "Nested ad-hoc" is under workstream:auth
	assert.Equal(t, 1, promoteCount)
	// Nested ad-hoc should now have id 001
	var found bool
	for i := range f.Roadmap {
		if f.Roadmap[i].Group == "G" && len(f.Roadmap[i].Items) > 0 {
			assert.Equal(t, "001", f.Roadmap[i].Items[0].ID)
			assert.Empty(t, f.Roadmap[i].Items[0].Title)
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestApplyPromotions_BestEffort(t *testing.T) {
	f, _ := Parse([]byte(`
roadmap:
  - title: First
  - title: Second
`))
	flt := &Filter{}
	failSecond := 0
	promote := func(_ map[string]interface{}, _ string) (string, error) {
		failSecond++
		if failSecond == 2 {
			return "", errors.New("fail second")
		}
		return fmt.Sprintf("%03d", failSecond), nil
	}
	errs := ApplyPromotions(f, flt, promote)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "fail second")
	// First should be promoted
	assert.Equal(t, "001", f.Roadmap[0].ID)
	assert.Empty(t, f.Roadmap[0].Title)
	// Second should remain ad-hoc (promote failed)
	assert.Empty(t, f.Roadmap[1].ID)
	assert.Equal(t, "Second", f.Roadmap[1].Title)
}
