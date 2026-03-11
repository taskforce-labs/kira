package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"kira/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sliceTestWorkItemPath = ".work/2_doing/001-test.prd.md"

func TestLoadSlicesFromFile(t *testing.T) {
	t.Run("returns content and slices when Slices section exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		content := `---
id: 001
title: Test
status: doing
kind: prd
---

# Test

## Requirements

## Slices

### S1
- [ ] T001: Task one
`
		require.NoError(t, os.WriteFile(sliceTestWorkItemPath, []byte(content), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		gotContent, slices, err := loadSlicesFromFile(sliceTestWorkItemPath, cfg)
		require.NoError(t, err)
		assert.NotNil(t, gotContent)
		require.Len(t, slices, 1)
		assert.Equal(t, "S1", slices[0].Name)
		require.Len(t, slices[0].Tasks, 1)
		assert.Equal(t, "T001", slices[0].Tasks[0].ID)
		assert.Equal(t, "Task one", slices[0].Tasks[0].Description)
		assert.False(t, slices[0].Tasks[0].Done)
	})

	t.Run("returns empty slices when no Slices section", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		content := `---
id: 001
status: doing
---
# Test
## Requirements
`
		require.NoError(t, os.WriteFile(sliceTestWorkItemPath, []byte(content), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		_, slices, err := loadSlicesFromFile(sliceTestWorkItemPath, cfg)
		require.NoError(t, err)
		require.NotNil(t, slices)
		assert.Len(t, slices, 0)
	})
}

func TestWriteSlicesToFile(t *testing.T) {
	t.Run("writes and preserves content outside Slices", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		content := `---
id: 001
status: doing
---
# Test
## Requirements
Req here
## Slices
### Old
- [ ] T001: Old
## Release
`
		require.NoError(t, os.WriteFile(sliceTestWorkItemPath, []byte(content), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		slices := []Slice{
			{Name: "New", Tasks: []Task{{ID: "T001", Description: "New task", Done: false}}},
		}
		err = writeSlicesToFile(sliceTestWorkItemPath, []byte(content), slices, cfg)
		require.NoError(t, err)

		got, err := os.ReadFile(sliceTestWorkItemPath)
		require.NoError(t, err)
		assert.Contains(t, string(got), "## Requirements")
		assert.Contains(t, string(got), "Req here")
		assert.Contains(t, string(got), "### 1. New")
		assert.Contains(t, string(got), "T001: New task")
		assert.Contains(t, string(got), "## Release")
		assert.NotContains(t, string(got), "### Old")
	})
}

func TestFindTaskByID(t *testing.T) {
	slices := []Slice{
		{Name: "A", Tasks: []Task{{ID: "T001", Description: "X"}, {ID: "T002", Description: "Y"}}},
		{Name: "B", Tasks: []Task{{ID: "T003", Description: "Z"}}},
	}
	si, ti := findTaskByID(slices, "T002")
	assert.Equal(t, 0, si)
	assert.Equal(t, 1, ti)
	si, ti = findTaskByID(slices, "T003")
	assert.Equal(t, 1, si)
	assert.Equal(t, 0, ti)
	si, ti = findTaskByID(slices, "T999")
	assert.Equal(t, -1, si)
	assert.Equal(t, -1, ti)
}

func TestFindSliceByName(t *testing.T) {
	slices := []Slice{
		{Name: "Auth", Tasks: nil},
		{Name: "API", Tasks: nil},
	}
	s := findSliceByName(slices, "API")
	require.NotNil(t, s)
	assert.Equal(t, "API", s.Name)
	s = findSliceByName(slices, "api")
	require.NotNil(t, s)
	assert.Equal(t, "API", s.Name)
	s = findSliceByName(slices, "None")
	assert.Nil(t, s)
}

func TestResolveSliceSelector(t *testing.T) {
	slices := []Slice{
		{Name: "First", Tasks: []Task{{ID: "T001", Done: true}}},
		{Name: "Second", Tasks: []Task{{ID: "T002", Done: false}}},
		{Name: "Third", Tasks: []Task{{ID: "T003", Done: false}}},
	}

	t.Run("by index 1 returns first slice", func(t *testing.T) {
		s, idx, err := resolveSliceSelector(slices, "1")
		require.NoError(t, err)
		require.NotNil(t, s)
		assert.Equal(t, "First", s.Name)
		assert.Equal(t, 1, idx)
	})

	t.Run("by index 2 returns second slice", func(t *testing.T) {
		s, idx, err := resolveSliceSelector(slices, "2")
		require.NoError(t, err)
		require.NotNil(t, s)
		assert.Equal(t, "Second", s.Name)
		assert.Equal(t, 2, idx)
	})

	t.Run("by name returns slice and index", func(t *testing.T) {
		s, idx, err := resolveSliceSelector(slices, "Third")
		require.NoError(t, err)
		require.NotNil(t, s)
		assert.Equal(t, "Third", s.Name)
		assert.Equal(t, 3, idx)
	})

	t.Run("current returns first slice with open tasks", func(t *testing.T) {
		s, idx, err := resolveSliceSelector(slices, "current")
		require.NoError(t, err)
		require.NotNil(t, s)
		assert.Equal(t, "Second", s.Name)
		assert.Equal(t, 2, idx)
	})

	t.Run("previous returns slice before current", func(t *testing.T) {
		s, idx, err := resolveSliceSelector(slices, "previous")
		require.NoError(t, err)
		require.NotNil(t, s)
		assert.Equal(t, "First", s.Name)
		assert.Equal(t, 1, idx)
	})

	t.Run("out of range index returns error", func(t *testing.T) {
		_, _, err := resolveSliceSelector(slices, "10")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "out of range")
	})

	t.Run("zero index returns error", func(t *testing.T) {
		_, _, err := resolveSliceSelector(slices, "0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("unknown name returns error", func(t *testing.T) {
		_, _, err := resolveSliceSelector(slices, "Nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestSliceShowWithNumericSelector(t *testing.T) {
	t.Run("slice show with slice number 1 succeeds and shows first slice", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		content := `---
id: 001
title: Num
status: doing
kind: prd
---
# Num
## Requirements
## Slices

### Alpha
- [ ] T001: First task

### Beta
- [ ] T002: Second task
`
		require.NoError(t, os.WriteFile(sliceTestWorkItemPath, []byte(content), 0o600))

		rootCmd.SetArgs([]string{"slice", "show", "001", "1", "--hide-summary"})
		err := rootCmd.Execute()
		require.NoError(t, err)
		// Command uses fmt.Print* so output goes to process stdout; we only verify no error.
		// resolveSliceSelector(..., "1") is unit-tested in TestResolveSliceSelector.
	})
}

func TestFirstSliceWithOpenTasks(t *testing.T) {
	slices := []Slice{
		{Name: "A", Tasks: []Task{{ID: "T001", Done: true}}},
		{Name: "B", Tasks: []Task{{ID: "T002", Done: false}}},
		{Name: "C", Tasks: []Task{{ID: "T003", Done: false}}},
	}
	s := firstSliceWithOpenTasks(slices)
	require.NotNil(t, s)
	assert.Equal(t, "B", s.Name)
	slices[0].Tasks[0].Done = false
	s = firstSliceWithOpenTasks(slices)
	require.NotNil(t, s)
	assert.Equal(t, "A", s.Name)
	allDone := []Slice{{Name: "X", Tasks: []Task{{ID: "T001", Done: true}}}}
	assert.Nil(t, firstSliceWithOpenTasks(allDone))
}

func TestFormatSliceSummary(t *testing.T) {
	t.Run("empty slices returns empty string", func(t *testing.T) {
		assert.Equal(t, "", formatSliceSummary(nil, ""))
		assert.Equal(t, "", formatSliceSummary([]Slice{}, ""))
	})
	t.Run("one slice all done", func(t *testing.T) {
		slices := []Slice{
			{Name: "S1", Tasks: []Task{{ID: "T001", Done: true}, {ID: "T002", Done: true}}},
		}
		got := formatSliceSummary(slices, "S1")
		assert.Contains(t, got, "1/1 slices")
		assert.Contains(t, got, "2/2 tasks")
		assert.Contains(t, got, "2/2 in current slice")
	})
	t.Run("two slices, one complete, current has open", func(t *testing.T) {
		slices := []Slice{
			{Name: "A", Tasks: []Task{{ID: "T001", Done: true}, {ID: "T002", Done: true}}},
			{Name: "B", Tasks: []Task{{ID: "T003", Done: true}, {ID: "T004", Done: false}}},
		}
		got := formatSliceSummary(slices, "B")
		assert.Contains(t, got, "1/2 slices")
		assert.Contains(t, got, "3/4 tasks")
		assert.Contains(t, got, "1/2 in current slice")
	})
	t.Run("current slice name empty uses first with open", func(t *testing.T) {
		slices := []Slice{
			{Name: "A", Tasks: []Task{{ID: "T001", Done: true}}},
			{Name: "B", Tasks: []Task{{ID: "T002", Done: false}}},
		}
		got := formatSliceSummary(slices, "")
		assert.Contains(t, got, "1/2 slices")
		assert.Contains(t, got, "1/2 tasks")
		// Current slice is B (first with open task), 0 done of 1 total
		assert.Contains(t, got, "0/1 in current slice")
	})
}

func TestDetectTaskChanges(t *testing.T) {
	current := []Slice{{Name: "S", Tasks: []Task{
		{ID: "T001", Description: "A", Done: true},
		{ID: "T002", Description: "B", Done: false},
		{ID: "T003", Description: "C", Done: false},
	}}}
	previous := []Slice{{Name: "S", Tasks: []Task{
		{ID: "T001", Description: "A", Done: false},
		{ID: "T002", Description: "B", Done: true},
	}}}
	completed, reopened, added := detectTaskChanges(previous, current)
	assert.Len(t, completed, 1)
	assert.Equal(t, "T001", completed[0].ID)
	assert.Len(t, reopened, 1)
	assert.Equal(t, "T002", reopened[0].ID)
	assert.Len(t, added, 1)
	assert.Equal(t, "T003", added[0].ID)
}

func TestFormatSliceCommitParts(t *testing.T) {
	completed := []Task{{ID: "T001", Description: "Done task"}}
	reopened := []Task{{ID: "T002", Description: "Reopened"}}
	added := []Task{{ID: "T003", Description: "New"}}
	msg := formatSliceCommitParts(completed, reopened, added)
	assert.Contains(t, msg, "Complete T001")
	assert.Contains(t, msg, "Reopen T002")
	assert.Contains(t, msg, "Add tasks T003")
	assert.Empty(t, formatSliceCommitParts(nil, nil, nil))
}

func TestLintSlicesSection(t *testing.T) {
	t.Run("reports missing Slices section", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		content := `---
id: 001
status: doing
---
# Test
## Requirements
`
		require.NoError(t, os.WriteFile(sliceTestWorkItemPath, []byte(content), 0o600))
		cfg, _ := config.LoadConfig()
		errs := lintSlicesSection(sliceTestWorkItemPath, cfg)
		require.Len(t, errs, 1)
		assert.Equal(t, "missing-section", errs[0].Rule)
		assert.Contains(t, errs[0].Message, "Slices section missing")
	})
	t.Run("reports duplicate task ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		content := `---
id: 001
status: doing
---
# Test
## Slices
### S1
- [ ] T001: A
### S2
- [ ] T001: B
`
		require.NoError(t, os.WriteFile(sliceTestWorkItemPath, []byte(content), 0o600))
		cfg, _ := config.LoadConfig()
		errs := lintSlicesSection(sliceTestWorkItemPath, cfg)
		require.Len(t, errs, 1)
		assert.Equal(t, "duplicate-task-id", errs[0].Rule)
		assert.Contains(t, errs[0].Message, "T001")
	})
	t.Run("reports duplicate slice name", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		content := `---
id: 001
status: doing
---
# Test
## Slices
### X
- [ ] T001: A
### X
- [ ] T002: B
`
		require.NoError(t, os.WriteFile(sliceTestWorkItemPath, []byte(content), 0o600))
		cfg, _ := config.LoadConfig()
		errs := lintSlicesSection(sliceTestWorkItemPath, cfg)
		require.Len(t, errs, 1)
		assert.Equal(t, "duplicate-slice-name", errs[0].Rule)
		assert.Contains(t, errs[0].Message, "Duplicate slice name")
	})
	t.Run("valid section returns no errors", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		content := `---
id: 001
status: doing
---
# Test
## Slices
### S1
- [ ] T001: A
- [x] T002: B
`
		require.NoError(t, os.WriteFile(sliceTestWorkItemPath, []byte(content), 0o600))
		cfg, _ := config.LoadConfig()
		errs := lintSlicesSection(sliceTestWorkItemPath, cfg)
		assert.Empty(t, errs)
	})
}

func TestPrintSliceSummaryIfPresent(t *testing.T) {
	t.Run("does not print when no Slices section", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		// Minimal work item without ## Slices (avoids goconst duplicate string)
		noSlicesContent := "---\nid: 099\nstatus: doing\n---\n# Test\n## Requirements\n"
		require.NoError(t, os.WriteFile(sliceTestWorkItemPath, []byte(noSlicesContent), 0o600))
		cfg, _ := config.LoadConfig()
		PrintSliceSummaryIfPresent(sliceTestWorkItemPath, cfg)
	})
	t.Run("prints when Slices section present", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		withSlices := "---\nid: 099\nstatus: doing\n---\n# Test\n## Slices\n### S1\n- [ ] T001: A\n- [x] T002: B\n"
		require.NoError(t, os.WriteFile(sliceTestWorkItemPath, []byte(withSlices), 0o600))
		cfg, _ := config.LoadConfig()
		PrintSliceSummaryIfPresent(sliceTestWorkItemPath, cfg)
	})
}

func TestSliceCommitRequiresSubcommand(t *testing.T) {
	t.Run("slice commit with no subcommand prints usage and exits non-zero", func(t *testing.T) {
		rootCmd.SetArgs([]string{"slice", "commit"})
		err := rootCmd.Execute()
		require.Error(t, err)
	})
	t.Run("slice commit with unknown subcommand prints usage and exits non-zero", func(t *testing.T) {
		rootCmd.SetArgs([]string{"slice", "commit", "unknownsub"})
		err := rootCmd.Execute()
		require.Error(t, err)
	})
}

// TestSliceCommandsWrongArgs ensures slice subcommands with required args return an error
// (and thus non-zero exit) when given too few args, and that the error message mentions arg count.
func TestSliceCommandsWrongArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string // substring of expected error (e.g. "accepts 2 arg(s)" or "requires at least")
	}{
		{"slice add with 0 args", []string{"slice", "add"}, "accepts 2 arg(s)"},
		{"slice add with 1 arg", []string{"slice", "add", "current"}, "accepts 2 arg(s)"},
		{"slice remove with 0 args", []string{"slice", "remove"}, "accepts 2 arg(s)"},
		{"slice remove with 1 arg", []string{"slice", "remove", "current"}, "accepts 2 arg(s)"},
		{"slice task add with 0 args", []string{"slice", "task", "add"}, "arg(s)"},
		{"slice task add with 1 arg", []string{"slice", "task", "add", "current"}, "arg(s)"},
		{"slice task add with 2 args", []string{"slice", "task", "add", "current", "MySlice"}, "arg(s)"},
		{"slice task remove with 0 args", []string{"slice", "task", "remove"}, "accepts 2 arg(s)"},
		{"slice task remove with 1 arg", []string{"slice", "task", "remove", "current"}, "accepts 2 arg(s)"},
		{"slice task edit with 0 args", []string{"slice", "task", "edit"}, "arg(s)"},
		{"slice task edit with 1 arg", []string{"slice", "task", "edit", "T001"}, "arg(s)"},
		{"slice task edit with 2 args", []string{"slice", "task", "edit", "current", "T001"}, "arg(s)"},
		{"slice task note with 0 args", []string{"slice", "task", "note"}, "arg(s)"},
		{"slice task note with 1 arg", []string{"slice", "task", "note", "current"}, "arg(s)"},
		{"slice task note with 2 args", []string{"slice", "task", "note", "current", "T001"}, "arg(s)"},
		{"slice show with 0 args", []string{"slice", "show"}, "arg(s)"},
		{"slice progress with 0 args", []string{"slice", "progress"}, "accepts 1 arg(s)"},
		{"slice commit add with 0 args", []string{"slice", "commit", "add"}, "arg(s)"},
		{"slice commit add with 1 arg", []string{"slice", "commit", "add", "MySlice"}, "arg(s)"},
		{"slice commit remove with 0 args", []string{"slice", "commit", "remove"}, "arg(s)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetHelpFlag(rootCmd)
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()
			require.Error(t, err, "Execute() should fail when args are missing")
			assert.Contains(t, err.Error(), tt.wantErr, "error should mention argument count")
		})
	}
}

// TestSliceCommandsWrongArgsPrintsUsage ensures that when wrong args are given,
// usage is printed (so SilenceUsage is not accidentally set on slice commands).
func TestSliceCommandsWrongArgsPrintsUsage(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetErr(&buf)
	rootCmd.SetOut(&buf)
	defer func() {
		rootCmd.SetErr(nil)
		rootCmd.SetOut(nil)
	}()

	resetHelpFlag(rootCmd)
	rootCmd.SetArgs([]string{"slice", "add"})
	_ = rootCmd.Execute()
	out := buf.String()
	assert.Contains(t, out, "Usage:", "output should contain usage when args are wrong")
	assert.Contains(t, out, "slice add", "output should contain the command use line")
}

func TestSliceCommitAdd(t *testing.T) {
	workItemWithSlices := `---
id: 041
title: slice commit
status: doing
kind: prd
---
# slice commit
## Slices
### MySlice
- [ ] T001: First task
`
	t.Run("add with explicit work-item-id adds task to slice", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(".work/2_doing/041-slice-commit.prd.md", []byte(workItemWithSlices), 0o600))

		rootCmd.SetArgs([]string{"slice", "commit", "add", "--no-commit", "041", "MySlice", "New task"})
		err := rootCmd.Execute()
		require.NoError(t, err)

		cfg, _ := config.LoadConfig()
		_, slices, err := loadSlicesFromFile(".work/2_doing/041-slice-commit.prd.md", cfg)
		require.NoError(t, err)
		require.Len(t, slices, 1)
		require.Len(t, slices[0].Tasks, 2)
		assert.Equal(t, "T002", slices[0].Tasks[1].ID)
		assert.Equal(t, "New task", slices[0].Tasks[1].Description)
	})
	t.Run("add with 2 args uses doing folder when one work item", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(".work/2_doing/041-slice-commit.prd.md", []byte(workItemWithSlices), 0o600))

		rootCmd.SetArgs([]string{"slice", "commit", "add", "--no-commit", "MySlice", "Another task"})
		err := rootCmd.Execute()
		require.NoError(t, err)

		cfg, _ := config.LoadConfig()
		_, slices, err := loadSlicesFromFile(".work/2_doing/041-slice-commit.prd.md", cfg)
		require.NoError(t, err)
		require.Len(t, slices[0].Tasks, 2)
		assert.Equal(t, "Another task", slices[0].Tasks[1].Description)
	})
	t.Run("add with no work-item-id and zero work items in doing returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		rootCmd.SetArgs([]string{"slice", "commit", "add", "SomeSlice", "desc"})
		err := rootCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no work item in doing folder")
	})
	t.Run("add with no work-item-id and multiple work items in doing returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(".work/2_doing/041-a.prd.md", []byte(workItemWithSlices), 0o600))
		other := `---
id: 042
title: other
status: doing
kind: prd
---
# other
## Slices
### S1
- [ ] T001: x
`
		require.NoError(t, os.WriteFile(".work/2_doing/042-b.prd.md", []byte(other), 0o600))

		rootCmd.SetArgs([]string{"slice", "commit", "add", "S1", "desc"})
		err := rootCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "multiple work items in doing folder")
	})
}

func TestSliceCommitRemove(t *testing.T) {
	workItemTwoSlices := `---
id: 041
title: slice commit
status: doing
kind: prd
---
# slice commit
## Slices
### Keep
- [ ] T001: Keep task
### RemoveMe
- [ ] T002: Remove task
`
	t.Run("remove with explicit work-item-id and slice name removes slice", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(".work/2_doing/041-slice-commit.prd.md", []byte(workItemTwoSlices), 0o600))

		rootCmd.SetArgs([]string{"slice", "commit", "remove", "--yes", "--no-commit", "041", "RemoveMe"})
		err := rootCmd.Execute()
		require.NoError(t, err)

		cfg, _ := config.LoadConfig()
		_, slices, err := loadSlicesFromFile(".work/2_doing/041-slice-commit.prd.md", cfg)
		require.NoError(t, err)
		require.Len(t, slices, 1)
		assert.Equal(t, "Keep", slices[0].Name)
	})
	t.Run("remove with 1 arg uses doing folder when one work item", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(".work/2_doing/041-slice-commit.prd.md", []byte(workItemTwoSlices), 0o600))

		rootCmd.SetArgs([]string{"slice", "commit", "remove", "--yes", "--no-commit", "RemoveMe"})
		err := rootCmd.Execute()
		require.NoError(t, err)

		cfg, _ := config.LoadConfig()
		_, slices, err := loadSlicesFromFile(".work/2_doing/041-slice-commit.prd.md", cfg)
		require.NoError(t, err)
		require.Len(t, slices, 1)
		assert.Equal(t, "Keep", slices[0].Name)
	})
	t.Run("remove with no work-item-id and zero work items in doing returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		rootCmd.SetArgs([]string{"slice", "commit", "remove", "SomeSlice"})
		err := rootCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no work item in doing folder")
	})
}

func TestSliceCommitGenerate(t *testing.T) {
	workItemGenerate := `---
id: 001
title: Implement OIDC login flow
status: doing
kind: prd
---
# Auth
## Slices
### Auth Token Validation
- [ ] T001: Implement OIDC login flow
- [ ] T002: Add JWT token validation
### Other Slice
- [ ] T003: Other task
`
	t.Run("generate output format: line1 id+message, line2 slug, line3 slice name, then task lines", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(".work/2_doing/001-auth.prd.md", []byte(workItemGenerate), 0o600))

		cfg, _ := config.LoadConfig()
		out, err := formatGeneratedCommitMessage(".work/2_doing/001-auth.prd.md", cfg, "001", "current")
		require.NoError(t, err)
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		require.GreaterOrEqual(t, len(lines), 6)
		assert.True(t, strings.HasPrefix(lines[0], "001 "))
		assert.Equal(t, "", lines[1])
		assert.Equal(t, "001-implement-oidc-login-flow", lines[2])
		assert.Equal(t, "", lines[3])
		assert.Equal(t, "Auth Token Validation:", lines[4])
		assert.Equal(t, "- T001 Implement OIDC login flow", lines[5])
		assert.Equal(t, "- T002 Add JWT token validation", lines[6])
	})
	t.Run("generate with named slice selector", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(".work/2_doing/001-auth.prd.md", []byte(workItemGenerate), 0o600))

		cfg, _ := config.LoadConfig()
		out, err := formatGeneratedCommitMessage(".work/2_doing/001-auth.prd.md", cfg, "001", "Other Slice")
		require.NoError(t, err)
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		require.GreaterOrEqual(t, len(lines), 5)
		assert.Equal(t, "Other Slice:", lines[4])
		assert.Equal(t, "- T003 Other task", lines[5])
	})
	t.Run("generate previous slice selector", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		// First slice all done so "current" is second slice; "previous" is first
		wi := `---
id: 001
title: Test
status: doing
kind: prd
---
# Test
## Slices
### First
- [x] T001: Done
### Second
- [ ] T002: Open
`
		require.NoError(t, os.WriteFile(".work/2_doing/001-test.prd.md", []byte(wi), 0o600))

		cfg, _ := config.LoadConfig()
		out, err := formatGeneratedCommitMessage(".work/2_doing/001-test.prd.md", cfg, "001", "previous")
		require.NoError(t, err)
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		assert.Equal(t, "First:", lines[4])
		assert.Equal(t, "- T001 Done", lines[5])
	})
	t.Run("generate with no work-item-id and zero work items in doing returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		rootCmd.SetArgs([]string{"slice", "commit", "generate"})
		err := rootCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no work item in doing folder")
	})
}

func TestSliceCommitCurrent(t *testing.T) {
	// Two slices: first all done, second has open. "previous" = first slice.
	wiPreviousComplete := `---
id: 047
title: slice commit current
status: doing
kind: prd
---
# slice commit current
## Slices
### First
- [x] T001: Done one
- [x] T002: Done two
### Second
- [ ] T003: Open
`
	// One slice with open tasks: no "previous" slice.
	wiNoPrevious := `---
id: 048
title: no previous
status: doing
kind: prd
---
# no previous
## Slices
### Only
- [ ] T001: Open
`
	t.Run("success when previous slice complete", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		path := ".work/2_doing/047-slice-commit-current.prd.md"
		require.NoError(t, os.WriteFile(path, []byte(wiPreviousComplete), 0o600))

		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test").Run())
		require.NoError(t, exec.Command("git", "add", path).Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial").Run())
		// Modify work item so we have something to commit
		require.NoError(t, os.WriteFile(path, []byte(strings.Replace(wiPreviousComplete, "Open", "Open task", 1)), 0o600))

		rootCmd.SetArgs([]string{"slice", "commit", "current", "047"})
		err := rootCmd.Execute()
		require.NoError(t, err)

		out, err := exec.Command("git", "log", "--oneline", "-1").Output()
		require.NoError(t, err)
		assert.Contains(t, string(out), "047")
		// Full message should reference the committed slice (First)
		fullMsg, _ := exec.Command("git", "log", "-1", "--format=%B").Output()
		assert.Contains(t, string(fullMsg), "First:")
	})
	t.Run("failure when no previous slice", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		path := ".work/2_doing/048-no-previous.prd.md"
		require.NoError(t, os.WriteFile(path, []byte(wiNoPrevious), 0o600))

		rootCmd.SetArgs([]string{"slice", "commit", "current", "048"})
		err := rootCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no previous slice")
	})
	t.Run("work-item resolution from doing folder when one work item", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		path := ".work/2_doing/047-slice-commit-current.prd.md"
		require.NoError(t, os.WriteFile(path, []byte(wiPreviousComplete), 0o600))

		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test").Run())
		require.NoError(t, exec.Command("git", "add", path).Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial").Run())
		require.NoError(t, os.WriteFile(path, []byte(strings.Replace(wiPreviousComplete, "Open", "Open task", 1)), 0o600))

		rootCmd.SetArgs([]string{"slice", "commit", "current"})
		err := rootCmd.Execute()
		require.NoError(t, err)
	})
	t.Run("failure when zero work items in doing folder", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		rootCmd.SetArgs([]string{"slice", "commit", "current"})
		err := rootCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no work item in doing folder")
	})
}

func TestSliceTaskDoneCurrent(t *testing.T) {
	workItemTwoTasks := `---
id: 001
title: Done current test
status: doing
kind: prd
---
# Done current test
## Slices
### S1
- [ ] T001: First task
- [ ] T002: Second task
`
	workItemAllDone := `---
id: 002
title: All done
status: doing
kind: prd
---
# All done
## Slices
### S1
- [x] T001: Done one
`
	t.Run("no open tasks returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(".work/2_doing/002-all-done.prd.md", []byte(workItemAllDone), 0o600))

		rootCmd.SetArgs([]string{"slice", "task", "done", "current", "002", "--hide-summary"})
		err := rootCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no open tasks")
	})
	t.Run("marks current task done and prints Completed line", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		path := ".work/2_doing/001-done-current.prd.md"
		require.NoError(t, os.WriteFile(path, []byte(workItemTwoTasks), 0o600))

		rootCmd.SetArgs([]string{"slice", "task", "done", "current", "001", "--hide-summary"})
		err := rootCmd.Execute()
		require.NoError(t, err)

		cfg, _ := config.LoadConfig()
		_, slices, err := loadSlicesFromFile(path, cfg)
		require.NoError(t, err)
		require.True(t, slices[0].Tasks[0].Done)
		assert.False(t, slices[0].Tasks[1].Done)
	})
	t.Run("done current --next shows next task and summary", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		path := ".work/2_doing/001-done-next.prd.md"
		require.NoError(t, os.WriteFile(path, []byte(workItemTwoTasks), 0o600))

		rootCmd.SetArgs([]string{"slice", "task", "done", "current", "001", "--next"})
		err := rootCmd.Execute()
		require.NoError(t, err)

		cfg, _ := config.LoadConfig()
		_, slices, err := loadSlicesFromFile(path, cfg)
		require.NoError(t, err)
		require.True(t, slices[0].Tasks[0].Done)
		// Output should contain Completed, Next (same slice), and summary line
		// We can't easily capture stdout in Execute; just verify state
		assert.False(t, slices[0].Tasks[1].Done)
	})
	t.Run("done current --next --hide-summary still shows next task", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		path := ".work/2_doing/001-done-hide.prd.md"
		require.NoError(t, os.WriteFile(path, []byte(workItemTwoTasks), 0o600))

		rootCmd.SetArgs([]string{"slice", "task", "done", "current", "001", "--next", "--hide-summary"})
		err := rootCmd.Execute()
		require.NoError(t, err)
		cfg, _ := config.LoadConfig()
		_, slices, err := loadSlicesFromFile(path, cfg)
		require.NoError(t, err)
		require.True(t, slices[0].Tasks[0].Done)
	})
}

func TestSliceAddAndShow(t *testing.T) {
	t.Run("slice add then show", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		content := `---
id: 026
title: Slices and tasks
status: doing
kind: prd
---
# Slices
## Requirements
## Acceptance Criteria
`
		path := filepath.Join(".work", "2_doing", "026-slices.prd.md")
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		_, slices, err := loadSlicesFromFile(path, cfg)
		require.NoError(t, err)
		slices = append(slices, Slice{Name: "NewSlice", Tasks: []Task{}})
		err = writeSlicesToFile(path, []byte(content), slices, cfg)
		require.NoError(t, err)

		_, slices, err = loadSlicesFromFile(path, cfg)
		require.NoError(t, err)
		require.Len(t, slices, 1)
		assert.Equal(t, "NewSlice", slices[0].Name)
		assert.Len(t, slices[0].Tasks, 0)
	})
}
