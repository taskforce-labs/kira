package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestSlicesConfigDefaults(t *testing.T) {
	t.Run("applies default slices config when not specified", func(t *testing.T) {
		_ = os.Remove("kira.yml")
		_ = os.Remove(".work/kira.yml")

		cfg, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, cfg.Slices)
		assert.Equal(t, "T%03d", cfg.Slices.TaskIDFormat)
		assert.Equal(t, "open", cfg.Slices.DefaultState)
		assert.False(t, cfg.Slices.AutoUpdateStatus)
	})

	t.Run("preserves custom slices config", func(t *testing.T) {
		testConfig := `version: "1.0"
slices:
  auto_update_status: true
  task_id_format: "T%04d"
  default_state: "done"
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		cfg, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, cfg.Slices)
		assert.True(t, cfg.Slices.AutoUpdateStatus)
		assert.Equal(t, "T%04d", cfg.Slices.TaskIDFormat)
		assert.Equal(t, "done", cfg.Slices.DefaultState)
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
		require.NotNil(t, config.Workspace.DraftPR)
		assert.True(t, *config.Workspace.DraftPR)
		assert.Equal(t, "make setup", config.Workspace.Setup)
	})
}

func TestWorkspaceDraftPRAndGitPlatform(t *testing.T) {
	t.Run("workspace draft_pr nil defaults to true", func(t *testing.T) {
		testConfig := `version: "1.0"
workspace:
  description: "Test"
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Workspace)
		require.NotNil(t, config.Workspace.DraftPR)
		assert.True(t, *config.Workspace.DraftPR)
	})

	t.Run("workspace draft_pr true preserved", func(t *testing.T) {
		testConfig := `version: "1.0"
workspace:
  draft_pr: true
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Workspace)
		require.NotNil(t, config.Workspace.DraftPR)
		assert.True(t, *config.Workspace.DraftPR)
	})

	t.Run("workspace draft_pr false preserved", func(t *testing.T) {
		testConfig := `version: "1.0"
workspace:
  draft_pr: false
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Workspace)
		require.NotNil(t, config.Workspace.DraftPR)
		assert.False(t, *config.Workspace.DraftPR)
	})

	t.Run("workspace git_platform and git_base_url preserved", func(t *testing.T) {
		testConfig := `version: "1.0"
workspace:
  git_platform: github
  git_base_url: https://github.example.com
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Workspace)
		assert.Equal(t, "github", config.Workspace.GitPlatform)
		assert.Equal(t, "https://github.example.com", config.Workspace.GitBaseURL)
	})

	t.Run("rejects invalid workspace git_platform", func(t *testing.T) {
		testConfig := `version: "1.0"
workspace:
  git_platform: gitlab
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid workspace.git_platform")
		assert.Contains(t, err.Error(), "github")
		assert.Contains(t, err.Error(), "auto")
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

	t.Run("rejects empty field name", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  "":
    type: string
    required: false
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field name cannot be empty")
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

	t.Run("rejects invalid date format with no time components", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  due:
    type: date
    format: "invalid-date-format"
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid date format")
		assert.Contains(t, err.Error(), "field 'due'")
		assert.Contains(t, err.Error(), "does not contain time components")
	})

	t.Run("accepts valid date format", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  due:
    type: date
    format: "2006-01-02"
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Fields)
		dueField, exists := config.Fields["due"]
		require.True(t, exists)
		assert.Equal(t, "date", dueField.Type)
		assert.Equal(t, "2006-01-02", dueField.Format)
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

	t.Run("rejects negative max when min is not set", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  estimate:
    type: number
    max: -5
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max")
		assert.Contains(t, err.Error(), "cannot be negative")
	})

	t.Run("rejects negative max when min is non-negative", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  estimate:
    type: number
    min: 0
    max: -5
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max")
		assert.Contains(t, err.Error(), "cannot be negative")
	})

	t.Run("allows negative range when both min and max are negative", func(t *testing.T) {
		testConfig := `version: "1.0"
fields:
  offset:
    type: number
    min: -10
    max: -5
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config.Fields)
		offsetField, exists := config.Fields["offset"]
		require.True(t, exists)
		assert.NotNil(t, offsetField.MinValue)
		assert.NotNil(t, offsetField.MaxValue)
		assert.Equal(t, -10.0, *offsetField.MinValue)
		assert.Equal(t, -5.0, *offsetField.MaxValue)
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

func TestGetWorkFolderPath(t *testing.T) {
	t.Run("returns default when config is nil", func(t *testing.T) {
		assert.Equal(t, ".work", GetWorkFolderPath(nil))
	})

	t.Run("returns default when workspace is nil", func(t *testing.T) {
		cfg := &Config{Workspace: nil}
		assert.Equal(t, ".work", GetWorkFolderPath(cfg))
	})

	t.Run("returns default when work_folder is empty", func(t *testing.T) {
		cfg := &Config{Workspace: &WorkspaceConfig{WorkFolder: ""}}
		assert.Equal(t, ".work", GetWorkFolderPath(cfg))
	})

	t.Run("returns custom work_folder", func(t *testing.T) {
		cfg := &Config{Workspace: &WorkspaceConfig{WorkFolder: "work"}}
		assert.Equal(t, "work", GetWorkFolderPath(cfg))
	})

	t.Run("returns trimmed work_folder", func(t *testing.T) {
		cfg := &Config{Workspace: &WorkspaceConfig{WorkFolder: "  tasks  "}}
		assert.Equal(t, "tasks", GetWorkFolderPath(cfg))
	})
}

func TestGetWorkFolderAbsPath(t *testing.T) {
	t.Run("resolves relative path against ConfigDir", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &Config{ConfigDir: tmpDir, Workspace: &WorkspaceConfig{WorkFolder: "work"}}
		absPath, err := GetWorkFolderAbsPath(cfg)
		require.NoError(t, err)
		expected := filepath.Join(tmpDir, "work")
		assert.Equal(t, filepath.Clean(expected), filepath.Clean(absPath))
	})

	t.Run("resolves default .work when ConfigDir set", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &Config{ConfigDir: tmpDir}
		absPath, err := GetWorkFolderAbsPath(cfg)
		require.NoError(t, err)
		expected := filepath.Join(tmpDir, ".work")
		assert.Equal(t, filepath.Clean(expected), filepath.Clean(absPath))
	})

	t.Run("uses current dir when ConfigDir empty", func(t *testing.T) {
		cfg := &Config{ConfigDir: ""}
		absPath, err := GetWorkFolderAbsPath(cfg)
		require.NoError(t, err)
		cwd, err := filepath.Abs(".")
		require.NoError(t, err)
		expected := filepath.Join(cwd, ".work")
		assert.Equal(t, filepath.Clean(expected), filepath.Clean(absPath))
	})
}

func TestLoadConfigFromDir(t *testing.T) {
	t.Run("returns default config when no file in dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg, err := LoadConfigFromDir(tmpDir)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, ".work", GetWorkFolderPath(cfg))
		absDir, err := filepath.Abs(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, absDir, cfg.ConfigDir)
	})

	t.Run("loads config from dir when kira.yml exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		testConfig := `version: "1.0"
workspace:
  work_folder: work
`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "kira.yml"), []byte(testConfig), 0o600))
		cfg, err := LoadConfigFromDir(tmpDir)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "work", GetWorkFolderPath(cfg))
		absDir, err := filepath.Abs(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, absDir, cfg.ConfigDir)
	})
}

func TestWorkFolderValidation(t *testing.T) {
	t.Run("rejects work_folder that is only whitespace", func(t *testing.T) {
		testConfig := `version: "1.0"
workspace:
  work_folder: "   "
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()
		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "work_folder")
	})

	t.Run("rejects work_folder containing null byte", func(t *testing.T) {
		testConfig := "version: \"1.0\"\nworkspace:\n  work_folder: \"work\x00path\"\n"
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()
		_, err := LoadConfig()
		require.Error(t, err)
		// YAML parser may reject control characters before we validate, or we reject in validateConfig
		assert.True(t, strings.Contains(err.Error(), "null") || strings.Contains(err.Error(), "control characters"),
			"expected error to mention null or control characters, got: %s", err.Error())
	})
}

func TestGetDocsFolderPath(t *testing.T) {
	t.Run("returns default when config is nil", func(t *testing.T) {
		assert.Equal(t, ".docs", GetDocsFolderPath(nil))
	})

	t.Run("returns default when docs_folder is empty", func(t *testing.T) {
		cfg := &Config{DocsFolder: ""}
		assert.Equal(t, ".docs", GetDocsFolderPath(cfg))
	})

	t.Run("returns custom docs_folder", func(t *testing.T) {
		cfg := &Config{DocsFolder: "docs"}
		assert.Equal(t, "docs", GetDocsFolderPath(cfg))
	})

	t.Run("returns trimmed docs_folder", func(t *testing.T) {
		cfg := &Config{DocsFolder: "  .docs  "}
		assert.Equal(t, ".docs", GetDocsFolderPath(cfg))
	})
}

func TestDocsRoot(t *testing.T) {
	t.Run("resolves relative path against targetDir", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &Config{DocsFolder: "docs"}
		absPath, err := DocsRoot(cfg, tmpDir)
		require.NoError(t, err)
		expected := filepath.Join(tmpDir, "docs")
		assert.Equal(t, filepath.Clean(expected), filepath.Clean(absPath))
	})

	t.Run("resolves default .docs when ConfigDir set", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &Config{ConfigDir: tmpDir}
		absPath, err := DocsRoot(cfg, tmpDir)
		require.NoError(t, err)
		expected := filepath.Join(tmpDir, ".docs")
		assert.Equal(t, filepath.Clean(expected), filepath.Clean(absPath))
	})

	t.Run("rejects docs_folder with path traversal", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &Config{DocsFolder: "../other"}
		_, err := DocsRoot(cfg, tmpDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not escape")
	})
}

func TestDocsFolderConfig(t *testing.T) {
	t.Run("loads docs_folder from kira.yml", func(t *testing.T) {
		testConfig := `version: "1.0"
docs_folder: documentation
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		config, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, "documentation", config.DocsFolder)
		assert.Equal(t, "documentation", GetDocsFolderPath(config))
	})

	t.Run("defaults docs_folder to .docs when missing", func(t *testing.T) {
		_ = os.Remove("kira.yml")
		_ = os.Remove(".work/kira.yml")

		config, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, ".docs", config.DocsFolder)
		assert.Equal(t, ".docs", GetDocsFolderPath(config))
	})

	t.Run("rejects docs_folder with ..", func(t *testing.T) {
		testConfig := `version: "1.0"
docs_folder: "../elsewhere"
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "docs_folder")
		assert.Contains(t, err.Error(), "..")
	})

	t.Run("rejects docs_folder that is only whitespace", func(t *testing.T) {
		testConfig := `version: "1.0"
docs_folder: "   "
`
		require.NoError(t, os.WriteFile("kira.yml", []byte(testConfig), 0o600))
		defer func() { _ = os.Remove("kira.yml") }()

		_, err := LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "docs_folder")
	})
}
