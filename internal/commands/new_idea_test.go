package commands

import (
	"os"
	"path/filepath"
	"testing"

	"kira/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWorkItemArgsWithIdea(t *testing.T) {
	cfg := &config.Config{
		StatusFolders: map[string]string{
			"todo":  "1_todo",
			"doing": "2_doing",
		},
	}

	t.Run("parses idea keyword with template and status", func(t *testing.T) {
		args := []string{"prd", "todo", "idea", "1"}
		result, err := parseWorkItemArgs(cfg, args)
		require.NoError(t, err)

		assert.Equal(t, "prd", result.template)
		assert.Equal(t, "todo", result.status)
		assert.Equal(t, 1, result.ideaNumber)
	})

	t.Run("parses idea keyword with just template", func(t *testing.T) {
		args := []string{"prd", "idea", "2"}
		result, err := parseWorkItemArgs(cfg, args)
		require.NoError(t, err)

		assert.Equal(t, "prd", result.template)
		assert.Equal(t, "", result.status)
		assert.Equal(t, 2, result.ideaNumber)
	})

	t.Run("parses idea keyword as first arg", func(t *testing.T) {
		args := []string{"idea", "3"}
		result, err := parseWorkItemArgs(cfg, args)
		require.NoError(t, err)

		assert.Equal(t, "", result.template)
		assert.Equal(t, "", result.status)
		assert.Equal(t, 3, result.ideaNumber)
	})

	t.Run("returns error for invalid idea number", func(t *testing.T) {
		args := []string{"prd", "todo", "idea", "abc"}
		_, err := parseWorkItemArgs(cfg, args)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid idea number")
	})

	t.Run("returns error for zero idea number", func(t *testing.T) {
		args := []string{"prd", "todo", "idea", "0"}
		_, err := parseWorkItemArgs(cfg, args)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid idea number")
	})

	t.Run("returns error for negative idea number", func(t *testing.T) {
		args := []string{"prd", "todo", "idea", "-1"}
		_, err := parseWorkItemArgs(cfg, args)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid idea number")
	})

	t.Run("returns error if idea keyword without number", func(t *testing.T) {
		args := []string{"prd", "todo", "idea"}
		_, err := parseWorkItemArgs(cfg, args)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "idea number required")
	})
}

func TestConvertIdeaToWorkItem(t *testing.T) {
	t.Run("converts idea to work item and removes idea", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Setup workspace
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.MkdirAll(".work/templates", 0o700))

		// Create IDEAS.md with an idea
		ideasContent := `# Ideas

## List

1. [2025-01-01] dark mode: allow the user to toggle between light and dark mode
`
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasContent), 0o600))

		// Create a simple template using the correct input placeholder format
		templateContent := `---
id: <!--input-number:id:"Work item ID"-->
title: <!--input-string:title:"Feature title"-->
status: <!--input-string:status:"Current status"-->
kind: prd
created: <!--input-datetime:created:"Creation date"-->
---
# <!--input-string:title:"Feature title"-->

<!--input-string:description:"Description"-->
`
		require.NoError(t, os.WriteFile(".work/templates/template.prd.md", []byte(templateContent), 0o600))

		// Create config
		cfg := &config.Config{
			Templates: map[string]string{
				"prd": "templates/template.prd.md",
			},
			StatusFolders: map[string]string{
				"todo": "1_todo",
			},
			DefaultStatus: "todo",
			Validation: config.ValidationConfig{
				RequiredFields: []string{"id", "title", "status", "kind", "created"},
				IDFormat:       "^\\d{3}$",
				StatusValues:   []string{"todo"},
			},
		}

		// Convert idea to work item
		err := convertIdeaToWorkItem(cfg, 1, "prd", "todo", false, nil, false)
		require.NoError(t, err)

		// Verify work item was created
		files, err := filepath.Glob(".work/1_todo/*.md")
		require.NoError(t, err)
		require.Len(t, files, 1)

		// Verify idea was removed - check IDEAS.md directly
		ideasContentAfter, err := os.ReadFile(".work/IDEAS.md")
		require.NoError(t, err)
		assert.NotContains(t, string(ideasContentAfter), "dark mode")
	})

	t.Run("parses title and description from idea with colon", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Setup workspace
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.MkdirAll(".work/templates", 0o700))

		// Create IDEAS.md with an idea
		ideasContent := `# Ideas

## List

1. [2025-01-01] dark mode: allow the user to toggle between light and dark mode
`
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasContent), 0o600))

		// Create a simple template using the correct input placeholder format
		templateContent := `---
id: <!--input-number:id:"Work item ID"-->
title: <!--input-string:title:"Feature title"-->
status: <!--input-string:status:"Current status"-->
kind: prd
created: <!--input-datetime:created:"Creation date"-->
---
# <!--input-string:title:"Feature title"-->

<!--input-string:description:"Description"-->
`
		require.NoError(t, os.WriteFile(".work/templates/template.prd.md", []byte(templateContent), 0o600))

		// Create config
		cfg := &config.Config{
			Templates: map[string]string{
				"prd": "templates/template.prd.md",
			},
			StatusFolders: map[string]string{
				"todo": "1_todo",
			},
			DefaultStatus: "todo",
			Validation: config.ValidationConfig{
				RequiredFields: []string{"id", "title", "status", "kind", "created"},
				IDFormat:       "^\\d{3}$",
				StatusValues:   []string{"todo"},
			},
		}

		// Convert idea to work item
		err := convertIdeaToWorkItem(cfg, 1, "prd", "todo", false, nil, false)
		require.NoError(t, err)

		// Read the created work item - work items have pattern like 001-dark-mode.prd.md
		files, err := filepath.Glob(".work/1_todo/001-*.md")
		require.NoError(t, err)
		require.Len(t, files, 1, "Expected exactly one work item file matching pattern 001-*.md")
		// Verify we're reading the work item file, not the template
		assert.Contains(t, files[0], "1_todo")
		assert.NotContains(t, files[0], "templates")

		content, err := os.ReadFile(files[0])
		require.NoError(t, err)

		contentStr := string(content)
		// Verify template was processed (no template syntax)
		assert.NotContains(t, contentStr, "{{.title}}")
		assert.NotContains(t, contentStr, "{{.id}}")
		// Verify content
		assert.Contains(t, contentStr, "title: dark mode")
		assert.Contains(t, contentStr, "allow the user to toggle between light and dark mode")
	})

	t.Run("returns error if idea not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Setup workspace
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.MkdirAll(".work/templates", 0o700))

		// Create IDEAS.md without the idea
		ideasContent := `# Ideas

## List

1. [2025-01-01] First idea
`
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasContent), 0o600))

		// Create config
		cfg := &config.Config{
			Templates: map[string]string{
				"prd": "templates/template.prd.md",
			},
			StatusFolders: map[string]string{
				"todo": "1_todo",
			},
			DefaultStatus: "todo",
			Validation: config.ValidationConfig{
				RequiredFields: []string{"id", "title", "status", "kind", "created"},
				IDFormat:       "^\\d{3}$",
				StatusValues:   []string{"todo"},
			},
		}

		// Try to convert non-existent idea
		err := convertIdeaToWorkItem(cfg, 100, "prd", "todo", false, nil, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Idea 100 not found")
	})
}
