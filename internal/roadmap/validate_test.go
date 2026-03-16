package roadmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_ValidFlat(t *testing.T) {
	f := &File{
		Roadmap: []Entry{
			{ID: "AUTH-001"},
			{Title: "OAuth spike", Meta: map[string]interface{}{"period": "Q1-26"}},
		},
	}
	errs := Validate(f)
	assert.False(t, HasErrors(errs), "expected no errors: %v", errs)
}

func TestValidate_ValidHierarchical(t *testing.T) {
	f := &File{
		Roadmap: []Entry{
			{
				Group: "Auth",
				Meta:  map[string]interface{}{"workstream": "auth"},
				Items: []Entry{
					{ID: "001"},
					{Title: "Spike"},
				},
			},
		},
	}
	errs := Validate(f)
	assert.False(t, HasErrors(errs), "expected no errors: %v", errs)
}

func TestValidate_EmptyEntry(t *testing.T) {
	f := &File{
		Roadmap: []Entry{
			{}, // empty
		},
	}
	errs := Validate(f)
	require.True(t, HasErrors(errs))
	assert.GreaterOrEqual(t, len(errs), 1)
	assert.Contains(t, errs[0].Message, "empty")
}

func TestValidate_GroupWithoutItems(t *testing.T) {
	f := &File{
		Roadmap: []Entry{
			{Group: "X", Items: nil},
		},
	}
	errs := Validate(f)
	require.True(t, HasErrors(errs))
	var found bool
	for _, e := range errs {
		if e.Message == "group must have items" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected group must have items: %v", errs)
}

func TestValidate_ItemsWithoutParent(t *testing.T) {
	f := &File{
		Roadmap: []Entry{
			{Items: []Entry{{ID: "001"}}}, // items but no id/title/group
		},
	}
	errs := Validate(f)
	require.True(t, HasErrors(errs))
	var found bool
	for _, e := range errs {
		if e.Message == "entry with items must have group, id, or title" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected entry with items must have group/id/title: %v", errs)
}

func TestValidate_BothIDAndTitle(t *testing.T) {
	f := &File{
		Roadmap: []Entry{
			{ID: "001", Title: "X"},
		},
	}
	errs := Validate(f)
	require.True(t, HasErrors(errs))
	var found bool
	for _, e := range errs {
		if e.Message == "entry cannot have both id and title" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestValidate_BothIDAndGroup(t *testing.T) {
	f := &File{
		Roadmap: []Entry{
			{ID: "001", Group: "X", Items: []Entry{{ID: "002"}}},
		},
	}
	errs := Validate(f)
	require.True(t, HasErrors(errs))
	var found bool
	for _, e := range errs {
		if e.Message == "entry cannot have both id and group" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestValidate_NilFile(t *testing.T) {
	errs := Validate(nil)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "nil")
}
