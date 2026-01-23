package validation

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

		err := ApplyFieldDefaults(workItem, cfg)
		require.NoError(t, err)

		assert.Equal(t, "medium", workItem.Fields["priority"])
		assert.Equal(t, 5, workItem.Fields["estimate"])
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

		err := ApplyFieldDefaults(workItem, cfg)
		require.NoError(t, err)

		assert.Equal(t, "high", workItem.Fields["priority"]) // Should not be overridden
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

		err := ApplyFieldDefaults(workItem, cfg)
		require.NoError(t, err)

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

		err := ApplyFieldDefaults(workItem, cfg)
		require.NoError(t, err)

		// ID should not be changed (it's a hardcoded field)
		assert.Equal(t, "001", workItem.ID)
		assert.NotEqual(t, "999", workItem.ID)
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

		cfg := &config.Config{
			Fields: map[string]config.FieldConfig{
				"priority": {
					Type:          "enum",
					AllowedValues: []string{"low", "medium", "high"},
					CaseSensitive: false, // Case-insensitive, so MEDIUM should be fixed to medium
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
