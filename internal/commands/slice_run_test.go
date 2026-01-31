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
