package commands

import (
	"os"
	"path/filepath"
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
		assert.Contains(t, string(got), "### New")
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
