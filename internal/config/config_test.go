package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("loads default config when no file exists", func(t *testing.T) {
		// Remove any existing config files
		_ = os.Remove("kira.yml")
		_ = os.Remove(".work/kira.yml")

		config, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, "1.0", config.Version)
		assert.NotEmpty(t, config.Templates)
		assert.NotEmpty(t, config.StatusFolders)
	})

	t.Run("loads config from file when exists", func(t *testing.T) {
		// Create a test config file at root-level
		testConfig := `version: "2.0"
templates:
  prd: "custom/prd.md"
status_folders:
  todo: "custom_todo"
`

		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, "2.0", config.Version)
		assert.Equal(t, "custom/prd.md", config.Templates["prd"])
		assert.Equal(t, "custom_todo", config.StatusFolders["todo"])
	})
}

func TestSaveConfig(t *testing.T) {
	t.Run("saves config to file", func(t *testing.T) {
		defer func() { _ = os.Remove("kira.yml") }()

		config := &Config{
			Version: "1.0",
			Templates: map[string]string{
				"prd": "test/prd.md",
			},
		}

		err := SaveConfig(config)
		require.NoError(t, err)

		// Verify file was created at root
		_, err = os.Stat("kira.yml")
		assert.NoError(t, err)
	})
}

func TestGitConfigDefaults(t *testing.T) {
	t.Run("applies default git config when not specified", func(t *testing.T) {
		_ = os.Remove("kira.yml")
		_ = os.Remove(".work/kira.yml")

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Git)
		assert.Equal(t, "origin", config.Git.Remote)
		assert.Equal(t, "", config.Git.TrunkBranch) // Empty means auto-detect
	})

	t.Run("preserves custom git config", func(t *testing.T) {
		testConfig := `version: "1.0"
git:
  trunk_branch: develop
  remote: upstream
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Git)
		assert.Equal(t, "develop", config.Git.TrunkBranch)
		assert.Equal(t, "upstream", config.Git.Remote)
	})
}

func TestStartConfigDefaults(t *testing.T) {
	t.Run("applies default start config when not specified", func(t *testing.T) {
		_ = os.Remove("kira.yml")
		_ = os.Remove(".work/kira.yml")

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Start)
		assert.Equal(t, "doing", config.Start.MoveTo)
		assert.Equal(t, "commit_and_push", config.Start.StatusAction)
		assert.Equal(t, "", config.Start.StatusCommitMessage)
	})

	t.Run("preserves custom start config", func(t *testing.T) {
		testConfig := `version: "1.0"
start:
  move_to: review
  status_action: commit_only
  status_commit_message: "Start {type} {id}"
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Start)
		assert.Equal(t, "review", config.Start.MoveTo)
		assert.Equal(t, "commit_only", config.Start.StatusAction)
		assert.Equal(t, "Start {type} {id}", config.Start.StatusCommitMessage)
	})
}

func TestStartConfigValidation(t *testing.T) {
	t.Run("rejects invalid move_to status", func(t *testing.T) {
		testConfig := `version: "1.0"
start:
  move_to: invalid_status
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status 'invalid_status'")
		assert.Contains(t, err.Error(), "status must be one of")
	})

	t.Run("rejects invalid status_action", func(t *testing.T) {
		testConfig := `version: "1.0"
start:
  status_action: invalid_action
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status_action value 'invalid_action'")
	})

	t.Run("accepts all valid status_actions", func(t *testing.T) {
		for _, action := range ValidStatusActions {
			testConfig := `version: "1.0"
start:
  status_action: ` + action + `
`
			require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))

			config, err := LoadConfig()
			require.NoError(t, err, "status_action %s should be valid", action)
			assert.Equal(t, action, config.Start.StatusAction)

			_ = os.Remove("kira.yml")
		}
	})
}

func TestWorkspaceConfigDefaults(t *testing.T) {
	t.Run("no workspace config results in nil workspace", func(t *testing.T) {
		_ = os.Remove("kira.yml")
		_ = os.Remove(".work/kira.yml")

		config, err := LoadConfig()
		require.NoError(t, err)
		assert.Nil(t, config.Workspace)
	})

	t.Run("applies default workspace root when not specified", func(t *testing.T) {
		testConfig := `version: "1.0"
workspace:
  description: "Test workspace"
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Workspace)
		assert.Equal(t, "../", config.Workspace.Root)
	})

	t.Run("preserves custom workspace config", func(t *testing.T) {
		testConfig := `version: "1.0"
workspace:
  root: "../../repos"
  worktree_root: "../worktrees"
  description: "My workspace"
  draft_pr: true
  setup: "make setup"
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Workspace)
		assert.Equal(t, "../../repos", config.Workspace.Root)
		assert.Equal(t, "../worktrees", config.Workspace.WorktreeRoot)
		assert.Equal(t, "My workspace", config.Workspace.Description)
		assert.True(t, config.Workspace.DraftPR)
		assert.Equal(t, "make setup", config.Workspace.Setup)
	})
}

func TestProjectConfigDefaults(t *testing.T) {
	t.Run("project mount defaults to name", func(t *testing.T) {
		testConfig := `version: "1.0"
workspace:
  projects:
    - name: frontend
      path: ../frontend
    - name: backend
      path: ../backend
      mount: api
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Workspace)
		require.Len(t, config.Workspace.Projects, 2)

		// First project: mount should default to name
		assert.Equal(t, "frontend", config.Workspace.Projects[0].Name)
		assert.Equal(t, "frontend", config.Workspace.Projects[0].Mount)

		// Second project: mount explicitly set
		assert.Equal(t, "backend", config.Workspace.Projects[1].Name)
		assert.Equal(t, "api", config.Workspace.Projects[1].Mount)
	})

	t.Run("preserves full project config", func(t *testing.T) {
		draftPR := true
		testConfig := `version: "1.0"
workspace:
  projects:
    - name: frontend
      path: ../frontend
      mount: fe
      repo_root: ../monorepo
      kind: app
      description: "Frontend app"
      draft_pr: true
      remote: upstream
      trunk_branch: develop
      setup: "npm install"
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Workspace)
		require.Len(t, config.Workspace.Projects, 1)

		project := config.Workspace.Projects[0]
		assert.Equal(t, "frontend", project.Name)
		assert.Equal(t, "../frontend", project.Path)
		assert.Equal(t, "fe", project.Mount)
		assert.Equal(t, "../monorepo", project.RepoRoot)
		assert.Equal(t, "app", project.Kind)
		assert.Equal(t, "Frontend app", project.Description)
		assert.NotNil(t, project.DraftPR)
		assert.Equal(t, draftPR, *project.DraftPR)
		assert.Equal(t, "upstream", project.Remote)
		assert.Equal(t, "develop", project.TrunkBranch)
		assert.Equal(t, "npm install", project.Setup)
	})
}

func TestIDEConfig(t *testing.T) {
	t.Run("no IDE config results in nil", func(t *testing.T) {
		_ = os.Remove("kira.yml")
		_ = os.Remove(".work/kira.yml")

		config, err := LoadConfig()
		require.NoError(t, err)
		assert.Nil(t, config.IDE)
	})

	t.Run("loads IDE config", func(t *testing.T) {
		testConfig := `version: "1.0"
ide:
  command: cursor
  args:
    - "--new-window"
    - "--wait"
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.IDE)
		assert.Equal(t, "cursor", config.IDE.Command)
		assert.Equal(t, []string{"--new-window", "--wait"}, config.IDE.Args)
	})
}

func TestFieldConfig(t *testing.T) {
	t.Run("loads field configuration", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  assigned:
    type: email
    required: false
    description: "Assigned user email address"
  priority:
    type: enum
    required: false
    allowed_values:
      - low
      - medium
      - high
    default: medium
    description: "Priority level"
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Fields)
		assert.Len(t, config.Fields, 2)

		assignedField, exists := config.Fields["assigned"]
		require.True(t, exists)
		assert.Equal(t, "email", assignedField.Type)
		assert.False(t, assignedField.Required)
		assert.Equal(t, "Assigned user email address", assignedField.Description)

		priorityField, exists := config.Fields["priority"]
		require.True(t, exists)
		assert.Equal(t, "enum", priorityField.Type)
		assert.Equal(t, []string{"low", "medium", "high"}, priorityField.AllowedValues)
		assert.Equal(t, "medium", priorityField.Default)
		// Verify default case_sensitive is true when not set
		require.NotNil(t, priorityField.CaseSensitive, "CaseSensitive should be set to true by default")
		assert.True(t, *priorityField.CaseSensitive, "CaseSensitive should default to true")
	})

	t.Run("rejects configuration of hardcoded fields", func(t *testing.T) {
		hardcodedFields := []string{"id", "title", "status", "kind", "created"}
		for _, field := range hardcodedFields {
			testConfig := `version: "1.0"
fields:
  ` + field + `:
    type: string
    required: false
`
			require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))

			_, err := LoadConfig()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "cannot be configured and must use hardcoded validation")
			assert.Contains(t, err.Error(), fmt.Sprintf("field '%s'", field))

			_ = os.Remove("kira.yml")
		}
	})

	t.Run("rejects invalid field type", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  custom_field:
    type: invalid_type
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid type")
		assert.Contains(t, err.Error(), "string, date, email, url, number, array, enum")
	})

	t.Run("rejects enum type without allowed_values", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  priority:
    type: enum
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "enum type requires allowed_values")
	})

	t.Run("rejects array type without item_type", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  tags:
    type: array
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "array type requires item_type")
	})

	t.Run("rejects invalid regex format", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  epic:
    type: string
    format: "[invalid regex"
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid regex format")
	})

	t.Run("validates min/max length constraints", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  description:
    type: string
    min_length: 10
    max_length: 5
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "min_length")
		assert.Contains(t, err.Error(), "cannot be greater than max_length")
	})

	t.Run("rejects negative min_length", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  description:
    type: string
    min_length: -1
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "min_length")
		assert.Contains(t, err.Error(), "cannot be negative")
	})

	t.Run("rejects negative max_length", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  description:
    type: string
    max_length: -5
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max_length")
		assert.Contains(t, err.Error(), "cannot be negative")
	})

	t.Run("validates min/max value constraints", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  estimate:
    type: number
    min: 10
    max: 5
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "min")
		assert.Contains(t, err.Error(), "cannot be greater than max")
	})

	t.Run("accepts valid field configuration", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  assigned:
    type: email
    required: false
    description: "Assigned user email"
  due:
    type: date
    required: false
    format: "2006-01-02"
    min_date: "today"
  priority:
    type: enum
    required: false
    allowed_values: [low, medium, high, critical]
    default: medium
    case_sensitive: false
  tags:
    type: array
    required: false
    item_type: string
    unique: true
  estimate:
    type: number
    required: false
    min: 0
    max: 100
  url:
    type: url
    required: false
    schemes: [http, https]
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Fields)
		assert.Len(t, config.Fields, 6)

		// Verify that explicitly setting case_sensitive: false works
		priorityField, exists := config.Fields["priority"]
		require.True(t, exists)
		require.NotNil(t, priorityField.CaseSensitive, "CaseSensitive should be set")
		assert.False(t, *priorityField.CaseSensitive, "CaseSensitive should be false when explicitly set")
	})

	t.Run("defaults case_sensitive to true for enum fields when not set", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  priority:
    type: enum
    allowed_values: [low, medium, high]
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Fields)

		priorityField, exists := config.Fields["priority"]
		require.True(t, exists)
		require.NotNil(t, priorityField.CaseSensitive, "CaseSensitive should be set to true by default")
		assert.True(t, *priorityField.CaseSensitive, "CaseSensitive should default to true when not specified")
	})
}
