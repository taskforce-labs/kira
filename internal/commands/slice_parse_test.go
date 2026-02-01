package commands

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSlicesSection(t *testing.T) {
	t.Run("returns nil when no Slices section", func(t *testing.T) {
		content := []byte("# Doc\n\n## Requirements\n\nSome text")
		slices, err := ParseSlicesSection(content)
		require.NoError(t, err)
		assert.Nil(t, slices)
	})

	t.Run("parses single slice with tasks", func(t *testing.T) {
		content := []byte(`## Slices

### Foundation
- [ ] T001: First task
- [x] T002: Second task
`)
		slices, err := ParseSlicesSection(content)
		require.NoError(t, err)
		require.Len(t, slices, 1)
		assert.Equal(t, "Foundation", slices[0].Name)
		require.Len(t, slices[0].Tasks, 2)
		assert.Equal(t, "T001", slices[0].Tasks[0].ID)
		assert.Equal(t, "First task", slices[0].Tasks[0].Description)
		assert.False(t, slices[0].Tasks[0].Done)
		assert.Equal(t, "T002", slices[0].Tasks[1].ID)
		assert.True(t, slices[0].Tasks[1].Done)
	})

	t.Run("parses Commit line under slice heading", func(t *testing.T) {
		content := []byte(`## Slices

### Foundation
Commit: Complete Foundation
- [ ] T001: Task one
`)
		slices, err := ParseSlicesSection(content)
		require.NoError(t, err)
		require.Len(t, slices, 1)
		assert.Equal(t, "Complete Foundation", slices[0].CommitSummary)
	})

	t.Run("parses multiple slices", func(t *testing.T) {
		content := []byte(`## Slices

### One
- [ ] T001: A

### Two
- [x] T002: B
`)
		slices, err := ParseSlicesSection(content)
		require.NoError(t, err)
		require.Len(t, slices, 2)
		assert.Equal(t, "One", slices[0].Name)
		require.Len(t, slices[0].Tasks, 1)
		assert.Equal(t, "T001", slices[0].Tasks[0].ID)
		assert.Equal(t, "Two", slices[1].Name)
		require.Len(t, slices[1].Tasks, 1)
		assert.Equal(t, "T002", slices[1].Tasks[0].ID)
		assert.True(t, slices[1].Tasks[0].Done)
	})

	t.Run("parses [open]/[done] format", func(t *testing.T) {
		content := []byte(`## Slices

### Test
- [open] T001: Open task
- [done] T002: Done task
`)
		slices, err := ParseSlicesSection(content)
		require.NoError(t, err)
		require.Len(t, slices, 1)
		require.Len(t, slices[0].Tasks, 2)
		assert.False(t, slices[0].Tasks[0].Done)
		assert.True(t, slices[0].Tasks[1].Done)
	})

	t.Run("parses notes line after task", func(t *testing.T) {
		content := []byte(`## Slices

### Test
- [ ] T001: Task one
  - Notes: Some note here
`)
		slices, err := ParseSlicesSection(content)
		require.NoError(t, err)
		require.Len(t, slices, 1)
		require.Len(t, slices[0].Tasks, 1)
		assert.Equal(t, "Some note here", slices[0].Tasks[0].Notes)
	})

	t.Run("finds real Slices section when ## Slices appears in prose and code block first", func(t *testing.T) {
		// Backtick in prose and ## Slices inside a fenced block must be skipped; only the real ## Slices is used.
		content := []byte("# PRD\n\n- Edit the work item directly (e.g. add a full \x60## Slices\x60 section).\n- Example in code block:\n```markdown\n## Slices\n\n### Fake\n- [ ] T001: Not real\n```\n\n## Requirements\nSome text.\n\n## Slices\n\n### Foundation\n- [ ] T001: Real task one\n- [x] T002: Real task two\n")
		slices, err := ParseSlicesSection(content)
		require.NoError(t, err)
		require.Len(t, slices, 1)
		assert.Equal(t, "Foundation", slices[0].Name)
		require.Len(t, slices[0].Tasks, 2)
		assert.Equal(t, "T001", slices[0].Tasks[0].ID)
		assert.Equal(t, "Real task one", slices[0].Tasks[0].Description)
		assert.Equal(t, "T002", slices[0].Tasks[1].ID)
		assert.True(t, slices[0].Tasks[1].Done)
	})
}

func TestGenerateSlicesSection(t *testing.T) {
	t.Run("formats slices and tasks with checkboxes", func(t *testing.T) {
		slices := []Slice{
			{
				Name: "Auth",
				Tasks: []Task{
					{ID: "T001", Description: "Login", Done: false},
					{ID: "T002", Description: "Logout", Done: true},
				},
			},
		}
		out := GenerateSlicesSection(slices, "T%03d")
		assert.Contains(t, string(out), "## Slices")
		assert.Contains(t, string(out), "### Auth")
		assert.Contains(t, string(out), "- [ ] T001: Login")
		assert.Contains(t, string(out), "- [x] T002: Logout")
	})

	t.Run("includes Commit line when set", func(t *testing.T) {
		slices := []Slice{
			{Name: "S1", CommitSummary: "Complete S1", Tasks: []Task{{ID: "T001", Description: "X", Done: false}}},
		}
		out := GenerateSlicesSection(slices, "T%03d")
		assert.Contains(t, string(out), "Commit: Complete S1")
	})

	t.Run("includes notes when set", func(t *testing.T) {
		slices := []Slice{
			{
				Name:  "S1",
				Tasks: []Task{{ID: "T001", Description: "X", Done: false, Notes: "A note"}},
			},
		}
		out := GenerateSlicesSection(slices, "T%03d")
		assert.Contains(t, string(out), "  - Notes: A note")
	})
}

func TestNextTaskID(t *testing.T) {
	t.Run("returns T001 when no tasks", func(t *testing.T) {
		id, err := NextTaskID(nil, "T%03d")
		require.NoError(t, err)
		assert.Equal(t, "T001", id)
	})

	t.Run("returns next sequential ID", func(t *testing.T) {
		slices := []Slice{
			{Tasks: []Task{{ID: "T001"}, {ID: "T002"}, {ID: "T003"}}},
		}
		id, err := NextTaskID(slices, "T%03d")
		require.NoError(t, err)
		assert.Equal(t, "T004", id)
	})

	t.Run("never reuses ID after task removed", func(t *testing.T) {
		slices := []Slice{
			{Tasks: []Task{{ID: "T001"}, {ID: "T003"}}}, // T002 missing
		}
		id, err := NextTaskID(slices, "T%03d")
		require.NoError(t, err)
		assert.Equal(t, "T004", id)
	})
}

func TestReplaceSlicesSection(t *testing.T) {
	t.Run("replaces existing Slices section", func(t *testing.T) {
		content := []byte(`# Doc

## Requirements
Req text

## Slices

### Old
- [ ] T001: Old task

## Release Notes
`)
		newSection := []byte(`## Slices

### New
- [ ] T001: New task
`)
		out, err := ReplaceSlicesSection(content, newSection)
		require.NoError(t, err)
		assert.Contains(t, string(out), "### New")
		assert.Contains(t, string(out), "## Release Notes")
		assert.NotContains(t, string(out), "### Old")
	})

	t.Run("inserts after Requirements when no Slices section", func(t *testing.T) {
		content := []byte(`# Doc

## Requirements
Req text

## Release Notes
`)
		newSection := []byte(`## Slices

### S1
- [ ] T001: Task
`)
		out, err := ReplaceSlicesSection(content, newSection)
		require.NoError(t, err)
		assert.Contains(t, string(out), "## Requirements")
		assert.Contains(t, string(out), "## Slices")
		assert.Contains(t, string(out), "## Release Notes")
		// Slices should appear after Requirements and before Release Notes
		reqPos := findSubstring(string(out), "## Requirements")
		slicesPos := findSubstring(string(out), "## Slices")
		releasePos := findSubstring(string(out), "## Release Notes")
		assert.True(t, reqPos < slicesPos && slicesPos < releasePos)
	})

	t.Run("preserves content before and after", func(t *testing.T) {
		content := []byte("Preamble\n\n## Slices\n\n### X\n- [ ] T001: Y\n\n## Trailer")
		newSection := []byte("## Slices\n\n### X\n- [x] T001: Y")
		out, err := ReplaceSlicesSection(content, newSection)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(string(out), "Preamble"))
		assert.Contains(t, string(out), "## Trailer")
	})
}

func findSubstring(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
