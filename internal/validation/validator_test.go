package validation

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kira/internal/config"
)

const testWorkItemPath = ".work/1_todo/001-test-feature.prd.md"

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
