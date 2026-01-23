package validation

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"

	"kira/internal/config"
)

const testWorkItemPath = ".work/1_todo/001-test-feature.prd.md"

const minimalWorkItemContent = `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
---
# Test Feature
`

func TestValidateWorkItems(t *testing.T) {
	t.Run("validates work items successfully", func(t *testing.T) {
		// Create a temporary workspace
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		// Create a valid work item
		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
---

# Test Feature

## Context
This is a test feature.
`

		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(workItemContent), 0o600))

		cfg := &config.DefaultConfig
		result, err := ValidateWorkItems(cfg)
		require.NoError(t, err)

		assert.False(t, result.HasErrors())
	})

	t.Run("detects missing required fields", func(t *testing.T) {
		// Create a temporary workspace
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		// Create an invalid work item (missing title in front matter)
		workItemContent := `---
id: 001
status: todo
kind: prd
created: 2024-01-01
---

# Test Feature
`

		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(workItemContent), 0o600))

		cfg := &config.DefaultConfig
		result, err := ValidateWorkItems(cfg)
		require.NoError(t, err)

		// Now that YAML front matter is parsed, missing title should be detected
		assert.True(t, result.HasErrors())
	})
}

func TestGetNextID(t *testing.T) {
	t.Run("generates first ID when no work items exist", func(t *testing.T) {
		// Create a temporary workspace
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))

		id, err := GetNextID()
		require.NoError(t, err)
		assert.Equal(t, "001", id)
	})

	t.Run("generates next sequential ID", func(t *testing.T) {
		// Create a temporary workspace
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		// Create a work item with ID 001
		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
---

# Test Feature
`

		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(workItemContent), 0o600))

		id, err := GetNextID()
		require.NoError(t, err)
		assert.Equal(t, "002", id)
	})
}

func TestFixDuplicateIDs(t *testing.T) {
	t.Run("fixes duplicate IDs", func(t *testing.T) {
		// Create a temporary workspace
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		// Create two work items with the same ID
		workItemContent1 := `---
id: 001
title: First Feature
status: todo
kind: prd
created: 2024-01-01
---

# First Feature
`

		workItemContent2 := `---
id: 001
title: Second Feature
status: todo
kind: prd
created: 2024-01-02
---

# Second Feature
`

		require.NoError(t, os.WriteFile(".work/1_todo/001-first-feature.prd.md", []byte(workItemContent1), 0o600))
		require.NoError(t, os.WriteFile(".work/1_todo/001-second-feature.prd.md", []byte(workItemContent2), 0o600))

		result, err := FixDuplicateIDs()
		require.NoError(t, err)

		// Should not have errors (duplicates should be fixed)
		assert.False(t, result.HasErrors())
	})
}

func TestFieldValidation(t *testing.T) {
	t.Run("validates string field with format", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
epic: EPIC-001
---

# Test Feature
`

		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(workItemContent), 0o600))

		cfg := &config.Config{
			Version:       "1.0",
			Templates:     config.DefaultConfig.Templates,
			StatusFolders: config.DefaultConfig.StatusFolders,
			Validation:    config.DefaultConfig.Validation,
			Fields: map[string]config.FieldConfig{
				"epic": {
					Type:   "string",
					Format: "^[A-Z]+-\\d+$",
				},
			},
		}

		result, err := ValidateWorkItems(cfg)
		require.NoError(t, err)
		assert.False(t, result.HasErrors())
	})

	t.Run("rejects string field with invalid format", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
epic: invalid-epic
---

# Test Feature
`

		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(workItemContent), 0o600))

		cfg := &config.Config{
			Version:       "1.0",
			Templates:     config.DefaultConfig.Templates,
			StatusFolders: config.DefaultConfig.StatusFolders,
			Validation:    config.DefaultConfig.Validation,
			Fields: map[string]config.FieldConfig{
				"epic": {
					Type:   "string",
					Format: "^[A-Z]+-\\d+$",
				},
			},
		}

		result, err := ValidateWorkItems(cfg)
		require.NoError(t, err)
		assert.True(t, result.HasErrors())
		assert.Contains(t, result.Error(), "epic")
		assert.Contains(t, result.Error(), "does not match format pattern")
	})

	t.Run("validates email field", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
assigned: user@example.com
---

# Test Feature
`

		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(workItemContent), 0o600))

		cfg := &config.Config{
			Version:       "1.0",
			Templates:     config.DefaultConfig.Templates,
			StatusFolders: config.DefaultConfig.StatusFolders,
			Validation:    config.DefaultConfig.Validation,
			Fields: map[string]config.FieldConfig{
				"assigned": {
					Type: "email",
				},
			},
		}

		result, err := ValidateWorkItems(cfg)
		require.NoError(t, err)
		assert.False(t, result.HasErrors())
	})

	t.Run("rejects invalid email field", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
assigned: not-an-email
---

# Test Feature
`

		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(workItemContent), 0o600))

		cfg := &config.Config{
			Version:       "1.0",
			Templates:     config.DefaultConfig.Templates,
			StatusFolders: config.DefaultConfig.StatusFolders,
			Validation:    config.DefaultConfig.Validation,
			Fields: map[string]config.FieldConfig{
				"assigned": {
					Type: "email",
				},
			},
		}

		result, err := ValidateWorkItems(cfg)
		require.NoError(t, err)
		assert.True(t, result.HasErrors())
		assert.Contains(t, result.Error(), "assigned")
		assert.Contains(t, result.Error(), "invalid email format")
	})

	t.Run("validates enum field", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
priority: high
---

# Test Feature
`

		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(workItemContent), 0o600))

		cfg := &config.Config{
			Version:       "1.0",
			Templates:     config.DefaultConfig.Templates,
			StatusFolders: config.DefaultConfig.StatusFolders,
			Validation:    config.DefaultConfig.Validation,
			Fields: map[string]config.FieldConfig{
				"priority": {
					Type:          "enum",
					AllowedValues: []string{"low", "medium", "high", "critical"},
				},
			},
		}

		result, err := ValidateWorkItems(cfg)
		require.NoError(t, err)
		assert.False(t, result.HasErrors())
	})

	t.Run("rejects invalid enum value", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
priority: invalid
---

# Test Feature
`

		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(workItemContent), 0o600))

		cfg := &config.Config{
			Version:       "1.0",
			Templates:     config.DefaultConfig.Templates,
			StatusFolders: config.DefaultConfig.StatusFolders,
			Validation:    config.DefaultConfig.Validation,
			Fields: map[string]config.FieldConfig{
				"priority": {
					Type:          "enum",
					AllowedValues: []string{"low", "medium", "high", "critical"},
				},
			},
		}

		result, err := ValidateWorkItems(cfg)
		require.NoError(t, err)
		assert.True(t, result.HasErrors())
		assert.Contains(t, result.Error(), "priority")
		assert.Contains(t, result.Error(), "not in allowed values")
	})

	t.Run("validates required fields from config", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
---

# Test Feature
`

		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(workItemContent), 0o600))

		cfg := &config.Config{
			Version:       "1.0",
			Templates:     config.DefaultConfig.Templates,
			StatusFolders: config.DefaultConfig.StatusFolders,
			Validation:    config.DefaultConfig.Validation,
			Fields: map[string]config.FieldConfig{
				"assigned": {
					Type:     "email",
					Required: true,
				},
			},
		}

		result, err := ValidateWorkItems(cfg)
		require.NoError(t, err)
		assert.True(t, result.HasErrors())
		assert.Contains(t, result.Error(), "missing required field: assigned")
	})

	t.Run("validates date field with field config", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
due: 2024-12-31
---

# Test Feature
`

		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(workItemContent), 0o600))

		cfg := &config.Config{
			Version:       "1.0",
			Templates:     config.DefaultConfig.Templates,
			StatusFolders: config.DefaultConfig.StatusFolders,
			Validation:    config.DefaultConfig.Validation,
			Fields: map[string]config.FieldConfig{
				"due": {
					Type:   "date",
					Format: "2006-01-02",
				},
			},
		}

		result, err := ValidateWorkItems(cfg)
		require.NoError(t, err)
		assert.False(t, result.HasErrors())
	})

	t.Run("validates number field with min/max", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
estimate: 5
---

# Test Feature
`

		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(workItemContent), 0o600))

		minVal := 0.0
		maxVal := 100.0
		cfg := &config.Config{
			Version:       "1.0",
			Templates:     config.DefaultConfig.Templates,
			StatusFolders: config.DefaultConfig.StatusFolders,
			Validation:    config.DefaultConfig.Validation,
			Fields: map[string]config.FieldConfig{
				"estimate": {
					Type:     "number",
					MinValue: &minVal,
					MaxValue: &maxVal,
				},
			},
		}

		result, err := ValidateWorkItems(cfg)
		require.NoError(t, err)
		assert.False(t, result.HasErrors())
	})

	t.Run("rejects number field outside range", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
estimate: 150
---

# Test Feature
`

		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(workItemContent), 0o600))

		minVal := 0.0
		maxVal := 100.0
		cfg := &config.Config{
			Version:       "1.0",
			Templates:     config.DefaultConfig.Templates,
			StatusFolders: config.DefaultConfig.StatusFolders,
			Validation:    config.DefaultConfig.Validation,
			Fields: map[string]config.FieldConfig{
				"estimate": {
					Type:     "number",
					MinValue: &minVal,
					MaxValue: &maxVal,
				},
			},
		}

		result, err := ValidateWorkItems(cfg)
		require.NoError(t, err)
		assert.True(t, result.HasErrors())
		assert.Contains(t, result.Error(), "estimate")
		assert.Contains(t, result.Error(), "greater than max")
	})

	t.Run("validates array field", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
tags: [implementation, testing]
---

# Test Feature
`

		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(workItemContent), 0o600))

		cfg := &config.Config{
			Version:       "1.0",
			Templates:     config.DefaultConfig.Templates,
			StatusFolders: config.DefaultConfig.StatusFolders,
			Validation:    config.DefaultConfig.Validation,
			Fields: map[string]config.FieldConfig{
				"tags": {
					Type:     "array",
					ItemType: "string",
				},
			},
		}

		result, err := ValidateWorkItems(cfg)
		require.NoError(t, err)
		assert.False(t, result.HasErrors())
	})

	t.Run("maintains backward compatibility without field config", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
due: 2024-12-31
---

# Test Feature
`

		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(workItemContent), 0o600))

		cfg := &config.DefaultConfig
		result, err := ValidateWorkItems(cfg)
		require.NoError(t, err)
		assert.False(t, result.HasErrors())
	})
}

func TestApplyFieldDefaults(t *testing.T) {
	t.Run("applies default values to missing fields", func(t *testing.T) {
		workItem := &WorkItem{
			ID:      "001",
			Title:   "Test",
			Status:  "todo",
			Kind:    "prd",
			Created: "2024-01-01",
			Fields:  make(map[string]interface{}),
		}

		cfg := &config.Config{
			Fields: map[string]config.FieldConfig{
				"priority": {
					Type:          "enum",
					Default:       "medium",
					AllowedValues: []string{"low", "medium", "high"},
				},
				"estimate": {
					Type:    "number",
					Default: 5,
				},
			},
		}

		added, err := ApplyFieldDefaults(workItem, cfg)
		require.NoError(t, err)

		assert.Equal(t, "medium", workItem.Fields["priority"])
		assert.Equal(t, 5, workItem.Fields["estimate"])
		assert.Len(t, added, 2)
		assert.Contains(t, added, "priority")
		assert.Contains(t, added, "estimate")
	})

	t.Run("does not override existing field values", func(t *testing.T) {
		workItem := &WorkItem{
			ID:      "001",
			Title:   "Test",
			Status:  "todo",
			Kind:    "prd",
			Created: "2024-01-01",
			Fields: map[string]interface{}{
				"priority": "high",
			},
		}

		cfg := &config.Config{
			Fields: map[string]config.FieldConfig{
				"priority": {
					Type:          "enum",
					Default:       "medium",
					AllowedValues: []string{"low", "medium", "high"},
				},
			},
		}

		added, err := ApplyFieldDefaults(workItem, cfg)
		require.NoError(t, err)

		assert.Equal(t, "high", workItem.Fields["priority"]) // Should not be overridden
		assert.Empty(t, added)
	})

	t.Run("applies 'today' default for date fields", func(t *testing.T) {
		workItem := &WorkItem{
			ID:      "001",
			Title:   "Test",
			Status:  "todo",
			Kind:    "prd",
			Created: "2024-01-01",
			Fields:  make(map[string]interface{}),
		}

		cfg := &config.Config{
			Fields: map[string]config.FieldConfig{
				"due": {
					Type:    "date",
					Default: "today",
				},
			},
		}

		added, err := ApplyFieldDefaults(workItem, cfg)
		require.NoError(t, err)

		assert.Len(t, added, 1)
		assert.Contains(t, added, "due")
		dueValue, exists := workItem.Fields["due"]
		require.True(t, exists)
		dueStr, ok := dueValue.(string)
		require.True(t, ok)
		// Should be today's date in YYYY-MM-DD format
		expectedDate := time.Now().Format("2006-01-02")
		assert.Equal(t, expectedDate, dueStr)
	})

	t.Run("skips hardcoded fields", func(t *testing.T) {
		workItem := &WorkItem{
			ID:      "001",
			Title:   "Test",
			Status:  "todo",
			Kind:    "prd",
			Created: "2024-01-01",
			Fields:  make(map[string]interface{}),
		}

		cfg := &config.Config{
			Fields: map[string]config.FieldConfig{
				"id": {
					Type:    "string",
					Default: "999",
				},
			},
		}

		added, err := ApplyFieldDefaults(workItem, cfg)
		require.NoError(t, err)

		// ID should not be changed (it's a hardcoded field)
		assert.Equal(t, "001", workItem.ID)
		assert.NotEqual(t, "999", workItem.ID)
		assert.Empty(t, added)
	})
}

func TestFixFieldIssues(t *testing.T) {
	t.Run("marks file as modified when required field with default is applied", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		filePath := testWorkItemPath
		require.NoError(t, os.WriteFile(filePath, []byte(minimalWorkItemContent), 0o600))

		// Get original file modification time
		originalInfo, err := os.Stat(filePath)
		require.NoError(t, err)
		originalModTime := originalInfo.ModTime()

		cfg := &config.Config{
			Fields: map[string]config.FieldConfig{
				"priority": {
					Type:          "enum",
					Required:      true,
					Default:       "medium",
					AllowedValues: []string{"low", "medium", "high"},
				},
			},
		}

		// Wait a bit to ensure mod time would change if file is written
		time.Sleep(10 * time.Millisecond)

		result, err := FixFieldIssues(cfg)
		require.NoError(t, err)

		// File should have been modified (default was applied)
		newInfo, err := os.Stat(filePath)
		require.NoError(t, err)
		assert.True(t, newInfo.ModTime().After(originalModTime), "File should have been modified when default was applied")

		// Verify the default was applied
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "priority: medium")

		// Verify the fix is reported to the user
		require.True(t, result.HasErrors(), "Result should contain error messages about fixes")
		found := false
		for _, validationErr := range result.Errors {
			if validationErr.File == filePath && strings.Contains(validationErr.Message, "fixed field 'priority': applied default value") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should report that default value was applied for priority field")
	})

	t.Run("does not mark file as modified when required field is missing but has no default", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		filePath := testWorkItemPath
		require.NoError(t, os.WriteFile(filePath, []byte(minimalWorkItemContent), 0o600))

		// Get original file modification time
		originalInfo, err := os.Stat(filePath)
		require.NoError(t, err)
		originalModTime := originalInfo.ModTime()

		cfg := &config.Config{
			Fields: map[string]config.FieldConfig{
				"assigned": {
					Type:     "email",
					Required: true,
					// No default configured
				},
			},
		}

		// Wait a bit to ensure mod time would change if file is written
		time.Sleep(10 * time.Millisecond)

		_, err = FixFieldIssues(cfg)
		require.NoError(t, err)

		// File should NOT have been modified (no default was applied)
		newInfo, err := os.Stat(filePath)
		require.NoError(t, err)
		assert.Equal(t, originalModTime, newInfo.ModTime(), "File should NOT have been modified when required field has no default")

		// Verify the field was not added
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.NotContains(t, string(content), "assigned:")
	})

	t.Run("marks file as modified when required field with empty string default is applied", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		filePath := testWorkItemPath
		require.NoError(t, os.WriteFile(filePath, []byte(minimalWorkItemContent), 0o600))

		// Get original file modification time
		originalInfo, err := os.Stat(filePath)
		require.NoError(t, err)
		originalModTime := originalInfo.ModTime()

		cfg := &config.Config{
			Fields: map[string]config.FieldConfig{
				"assigned": {
					Type:     "email",
					Required: true,
					Default:  "", // Empty string default
				},
			},
		}

		// Wait a bit to ensure mod time would change if file is written
		time.Sleep(10 * time.Millisecond)

		_, err = FixFieldIssues(cfg)
		require.NoError(t, err)

		// File should have been modified (empty string default was applied)
		newInfo, err := os.Stat(filePath)
		require.NoError(t, err)
		assert.True(t, newInfo.ModTime().After(originalModTime), "File should have been modified when empty string default was applied")

		// Verify the empty string default was applied
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "assigned:")
	})

	t.Run("marks file as modified when non-required field with default is applied", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		filePath := testWorkItemPath
		require.NoError(t, os.WriteFile(filePath, []byte(minimalWorkItemContent), 0o600))

		originalInfo, err := os.Stat(filePath)
		require.NoError(t, err)
		originalModTime := originalInfo.ModTime()

		cfg := &config.Config{
			Fields: map[string]config.FieldConfig{
				"priority": {
					Type:          "enum",
					Required:      false, // non-required
					Default:       "medium",
					AllowedValues: []string{"low", "medium", "high"},
				},
			},
		}

		time.Sleep(10 * time.Millisecond)

		result, err := FixFieldIssues(cfg)
		require.NoError(t, err)

		newInfo, err := os.Stat(filePath)
		require.NoError(t, err)
		assert.True(t, newInfo.ModTime().After(originalModTime), "File should have been modified when default was applied for non-required field")

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "priority: medium")

		require.True(t, result.HasErrors())
		found := false
		for _, validationErr := range result.Errors {
			if validationErr.File == filePath && strings.Contains(validationErr.Message, "fixed field 'priority': applied default value") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should report that default value was applied for priority field")
	})

	t.Run("marks file as modified when field value is fixed", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
priority: MEDIUM
---
# Test Feature
`

		filePath := testWorkItemPath
		require.NoError(t, os.WriteFile(filePath, []byte(workItemContent), 0o600))

		// Get original file modification time
		originalInfo, err := os.Stat(filePath)
		require.NoError(t, err)
		originalModTime := originalInfo.ModTime()

		caseSensitive := false
		cfg := &config.Config{
			Fields: map[string]config.FieldConfig{
				"priority": {
					Type:          "enum",
					AllowedValues: []string{"low", "medium", "high"},
					CaseSensitive: &caseSensitive, // Case-insensitive, so MEDIUM should be fixed to medium
				},
			},
		}

		// Wait a bit to ensure mod time would change if file is written
		time.Sleep(10 * time.Millisecond)

		_, err = FixFieldIssues(cfg)
		require.NoError(t, err)

		// File should have been modified (value was fixed)
		newInfo, err := os.Stat(filePath)
		require.NoError(t, err)
		assert.True(t, newInfo.ModTime().After(originalModTime), "File should have been modified when field value was fixed")

		// Verify the value was fixed
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "priority: medium")
		assert.NotContains(t, string(content), "priority: MEDIUM")
	})
}

func TestFixHardcodedDateFormats(t *testing.T) {
	t.Run("fixes ISO 8601 timestamp format", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-15T10:30:00Z
---
# Test Feature

## Context
This is a test feature with body content.
`

		filePath := testWorkItemPath
		require.NoError(t, os.WriteFile(filePath, []byte(workItemContent), 0o600))

		result, err := FixHardcodedDateFormats()
		require.NoError(t, err)
		assert.True(t, result.HasErrors()) // Should have a fix message

		// Verify the file was updated
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "created: 2024-01-15")
		assert.NotContains(t, string(content), "created: 2024-01-15T10:30:00Z")

		// Verify body content is preserved
		assert.Contains(t, string(content), "# Test Feature")
		assert.Contains(t, string(content), "This is a test feature with body content.")
	})

	t.Run("fixes ISO 8601 with timezone format", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-15T10:30:00-05:00
---
# Test Feature
`

		filePath := testWorkItemPath
		require.NoError(t, os.WriteFile(filePath, []byte(workItemContent), 0o600))

		result, err := FixHardcodedDateFormats()
		require.NoError(t, err)
		assert.True(t, result.HasErrors()) // Should have a fix message

		// Verify the file was updated
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "created: 2024-01-15")
	})

	t.Run("fixes slash date format", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024/01/15
---
# Test Feature
`

		filePath := testWorkItemPath
		require.NoError(t, os.WriteFile(filePath, []byte(workItemContent), 0o600))

		result, err := FixHardcodedDateFormats()
		require.NoError(t, err)
		assert.True(t, result.HasErrors()) // Should have a fix message

		// Verify the file was updated
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "created: 2024-01-15")
	})

	t.Run("does not modify already correct date format", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-15
---
# Test Feature
`

		filePath := testWorkItemPath
		require.NoError(t, os.WriteFile(filePath, []byte(workItemContent), 0o600))

		result, err := FixHardcodedDateFormats()
		require.NoError(t, err)
		assert.False(t, result.HasErrors()) // Should not have any fixes

		// Verify the file was not modified
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "created: 2024-01-15")
	})

	t.Run("preserves body content when fixing dates", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-15T10:30:00Z
---
# Test Feature

## Context
This is a test feature.

## Requirements
- Requirement 1
- Requirement 2

## Notes
Some important notes here.
`

		filePath := testWorkItemPath
		require.NoError(t, os.WriteFile(filePath, []byte(workItemContent), 0o600))

		result, err := FixHardcodedDateFormats()
		require.NoError(t, err)
		assert.True(t, result.HasErrors()) // Should have a fix message

		// Verify body content is fully preserved
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		contentStr := string(content)
		assert.Contains(t, contentStr, "# Test Feature")
		assert.Contains(t, contentStr, "## Context")
		assert.Contains(t, contentStr, "This is a test feature.")
		assert.Contains(t, contentStr, "## Requirements")
		assert.Contains(t, contentStr, "- Requirement 1")
		assert.Contains(t, contentStr, "- Requirement 2")
		assert.Contains(t, contentStr, "## Notes")
		assert.Contains(t, contentStr, "Some important notes here.")
	})
}

func TestIsNumeric(t *testing.T) {
	t.Run("returns true for all integer types", func(t *testing.T) {
		assert.True(t, IsNumeric(int(42)))
		assert.True(t, IsNumeric(int8(42)))
		assert.True(t, IsNumeric(int16(42)))
		assert.True(t, IsNumeric(int32(42)))
		assert.True(t, IsNumeric(int64(42)))
	})

	t.Run("returns true for all unsigned integer types", func(t *testing.T) {
		assert.True(t, IsNumeric(uint(42)))
		assert.True(t, IsNumeric(uint8(42)))
		assert.True(t, IsNumeric(uint16(42)))
		assert.True(t, IsNumeric(uint32(42)))
		assert.True(t, IsNumeric(uint64(42)))
	})

	t.Run("returns true for floating point types", func(t *testing.T) {
		assert.True(t, IsNumeric(float32(3.14)))
		assert.True(t, IsNumeric(float64(3.14)))
	})

	t.Run("returns false for non-numeric types", func(t *testing.T) {
		assert.False(t, IsNumeric("42"))
		assert.False(t, IsNumeric(true))
		assert.False(t, IsNumeric(false))
		assert.False(t, IsNumeric(nil))
		assert.False(t, IsNumeric([]int{1, 2, 3}))
		assert.False(t, IsNumeric(map[string]int{"a": 1}))
		assert.False(t, IsNumeric(struct{}{}))
	})
}

func TestYAMLQuoting(t *testing.T) {
	t.Run("quotes strings with colons", func(t *testing.T) {
		var sb strings.Builder
		err := writeYAMLField(&sb, "priority", "high:urgent")
		require.NoError(t, err)
		output := sb.String()
		assert.Contains(t, output, `priority: "high:urgent"`)
	})

	t.Run("quotes strings with hash symbols", func(t *testing.T) {
		var sb strings.Builder
		err := writeYAMLField(&sb, "note", "# comment")
		require.NoError(t, err)
		output := sb.String()
		assert.Contains(t, output, `note: "# comment"`)
	})

	t.Run("quotes strings with brackets", func(t *testing.T) {
		var sb strings.Builder
		err := writeYAMLField(&sb, "tags", "[urgent, critical]")
		require.NoError(t, err)
		output := sb.String()
		assert.Contains(t, output, `tags: "[urgent, critical]"`)
	})

	t.Run("quotes strings with curly braces", func(t *testing.T) {
		var sb strings.Builder
		err := writeYAMLField(&sb, "config", "{key: value}")
		require.NoError(t, err)
		output := sb.String()
		assert.Contains(t, output, `config: "{key: value}"`)
	})

	t.Run("quotes strings with quotes", func(t *testing.T) {
		var sb strings.Builder
		err := writeYAMLField(&sb, "message", `say "hello"`)
		require.NoError(t, err)
		output := sb.String()
		assert.Contains(t, output, `message: "say \"hello\""`)
	})

	t.Run("quotes strings with backslashes", func(t *testing.T) {
		var sb strings.Builder
		err := writeYAMLField(&sb, "path", `C:\Users\name`)
		require.NoError(t, err)
		output := sb.String()
		assert.Contains(t, output, `path: "C:\\Users\\name"`)
	})

	t.Run("quotes strings with newlines", func(t *testing.T) {
		var sb strings.Builder
		err := writeYAMLField(&sb, "description", "line1\nline2")
		require.NoError(t, err)
		output := sb.String()
		assert.Contains(t, output, `description: "line1\nline2"`)
	})

	t.Run("quotes strings with leading/trailing spaces", func(t *testing.T) {
		var sb strings.Builder
		err := writeYAMLField(&sb, "value", "  spaced  ")
		require.NoError(t, err)
		output := sb.String()
		assert.Contains(t, output, `value: "  spaced  "`)
	})

	t.Run("quotes empty strings", func(t *testing.T) {
		var sb strings.Builder
		err := writeYAMLField(&sb, "empty", "")
		require.NoError(t, err)
		output := sb.String()
		assert.Contains(t, output, `empty: ""`)
	})

	t.Run("does not quote simple strings", func(t *testing.T) {
		var sb strings.Builder
		err := writeYAMLField(&sb, "status", "todo")
		require.NoError(t, err)
		output := sb.String()
		assert.Contains(t, output, "status: todo")
		assert.NotContains(t, output, `status: "todo"`)
	})

	t.Run("quotes array items with special characters", func(t *testing.T) {
		var sb strings.Builder
		err := writeYAMLField(&sb, "tags", []interface{}{"urgent", "high:priority", "normal"})
		require.NoError(t, err)
		output := sb.String()
		// Simple strings don't need quotes, only ones with special characters
		assert.Contains(t, output, "urgent")
		assert.Contains(t, output, `"high:priority"`)
		assert.Contains(t, output, "normal")
		assert.NotContains(t, output, `"urgent"`)
		assert.NotContains(t, output, `"normal"`)
	})

	t.Run("round-trip preserves special characters", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		// Create a work item with special characters
		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-15
priority: high:urgent
note: # important comment
tags: [urgent, critical]
---
# Test Feature
`

		filePath := testWorkItemPath
		require.NoError(t, os.WriteFile(filePath, []byte(workItemContent), 0o600))

		// Parse the work item
		workItem, err := parseWorkItemFile(filePath)
		require.NoError(t, err)

		// Modify fields to include special characters
		workItem.Fields["priority"] = "high:urgent"
		workItem.Fields["note"] = "# important comment"
		workItem.Fields["tags"] = []interface{}{"urgent", "critical", "high:priority"}

		// Write it back
		err = writeWorkItemFile(filePath, workItem)
		require.NoError(t, err)

		// Read and parse again to verify it's valid YAML
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)

		// Verify the YAML is valid by parsing it
		var parsed map[string]interface{}
		lines := strings.Split(string(content), "\n")
		var yamlLines []string
		inYAML := false
		for _, line := range lines {
			if strings.TrimSpace(line) == "---" {
				if !inYAML {
					inYAML = true
					continue
				}
				break
			}
			if inYAML {
				yamlLines = append(yamlLines, line)
			}
		}
		yamlContent := strings.Join(yamlLines, "\n")

		err = yaml.Unmarshal([]byte(yamlContent), &parsed)
		require.NoError(t, err, "Generated YAML should be valid and parseable")

		// Verify values are preserved
		assert.Equal(t, "high:urgent", parsed["priority"])
		assert.Equal(t, "# important comment", parsed["note"])
		assert.Contains(t, string(content), `priority: "high:urgent"`)
		assert.Contains(t, string(content), `note: "# important comment"`)
	})

	t.Run("round-trip with complex special characters", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		workItemContent := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-15
---
# Test Feature
`

		filePath := testWorkItemPath
		require.NoError(t, os.WriteFile(filePath, []byte(workItemContent), 0o600))

		// Parse the work item
		workItem, err := parseWorkItemFile(filePath)
		require.NoError(t, err)

		// Add fields with various special characters
		workItem.Fields["path"] = `C:\Users\name\file.txt`
		workItem.Fields["message"] = `say "hello" and 'goodbye'`
		workItem.Fields["config"] = "{key: value, nested: {a: b}}"
		workItem.Fields["multiline"] = "line1\nline2\nline3"

		// Write it back
		err = writeWorkItemFile(filePath, workItem)
		require.NoError(t, err)

		// Read and verify it parses correctly
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)

		// Parse the YAML front matter
		lines := strings.Split(string(content), "\n")
		var yamlLines []string
		inYAML := false
		for _, line := range lines {
			if strings.TrimSpace(line) == "---" {
				if !inYAML {
					inYAML = true
					continue
				}
				break
			}
			if inYAML {
				yamlLines = append(yamlLines, line)
			}
		}
		yamlContent := strings.Join(yamlLines, "\n")

		var parsed map[string]interface{}
		err = yaml.Unmarshal([]byte(yamlContent), &parsed)
		require.NoError(t, err, "Generated YAML should be valid and parseable")

		// Verify all values are preserved correctly
		assert.Equal(t, `C:\Users\name\file.txt`, parsed["path"])
		assert.Equal(t, `say "hello" and 'goodbye'`, parsed["message"])
		assert.Equal(t, "{key: value, nested: {a: b}}", parsed["config"])
		assert.Equal(t, "line1\nline2\nline3", parsed["multiline"])
	})
}

func TestWriteYAMLFieldErrorPropagation(t *testing.T) {
	t.Run("writeYAMLField returns error for unmarshalable types", func(t *testing.T) {
		var sb strings.Builder
		// Use a channel type which cannot be marshaled to YAML
		ch := make(chan int)
		err := writeYAMLField(&sb, "test_field", ch)
		require.Error(t, err, "writeYAMLField should return error for unmarshalable types")
		assert.Contains(t, err.Error(), "yaml:", "error should mention YAML marshaling")
	})

	t.Run("writeYAMLFrontMatter propagates error from writeYAMLField", func(t *testing.T) {
		var sb strings.Builder
		workItem := &WorkItem{
			ID:      "001",
			Title:   "Test",
			Status:  "todo",
			Kind:    "prd",
			Created: "2024-01-01",
			Fields: map[string]interface{}{
				"unmarshalable": make(chan int), // Channel cannot be marshaled
			},
		}

		err := writeYAMLFrontMatter(&sb, workItem)
		require.Error(t, err, "writeYAMLFrontMatter should return error when writeYAMLField fails")
		assert.Contains(t, err.Error(), "failed to write field 'unmarshalable'", "error should include field name")
		assert.Contains(t, err.Error(), "yaml:", "error should wrap the original YAML error")
	})

	t.Run("writeWorkItemFile propagates error from writeYAMLFrontMatter", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		// Create a work item file
		filePath := testWorkItemPath
		require.NoError(t, os.WriteFile(filePath, []byte(minimalWorkItemContent), 0o600))

		// Parse the work item
		workItem, err := parseWorkItemFile(filePath)
		require.NoError(t, err)

		// Add an unmarshalable field
		workItem.Fields["unmarshalable"] = make(chan int)

		// Attempt to write it back - should fail
		err = writeWorkItemFile(filePath, workItem)
		require.Error(t, err, "writeWorkItemFile should return error when writeYAMLFrontMatter fails")
		assert.Contains(t, err.Error(), "failed to write YAML front matter", "error should mention YAML front matter")
		assert.Contains(t, err.Error(), "failed to write field 'unmarshalable'", "error should include field name in chain")
	})

	t.Run("writeYAMLField handles function types", func(t *testing.T) {
		var sb strings.Builder
		// Use a function type which cannot be marshaled to YAML
		testFunc := func() {}
		err := writeYAMLField(&sb, "test_field", testFunc)
		require.Error(t, err, "writeYAMLField should return error for function types")
		assert.Contains(t, err.Error(), "yaml:", "error should mention YAML marshaling")
	})
}
