// Package config provides configuration management for the kira tool.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"
)

// Config represents the kira configuration structure.
type Config struct {
	Version       string                 `yaml:"version"`
	Templates     map[string]string      `yaml:"templates"`
	StatusFolders map[string]string      `yaml:"status_folders"`
	Validation    ValidationConfig       `yaml:"validation"`
	Commit        CommitConfig           `yaml:"commit"`
	Release       ReleaseConfig          `yaml:"release"`
	DefaultStatus string                 `yaml:"default_status"`
	Git           *GitConfig             `yaml:"git"`
	Start         *StartConfig           `yaml:"start"`
	IDE           *IDEConfig             `yaml:"ide"`
	Workspace     *WorkspaceConfig       `yaml:"workspace"`
	Users         UsersConfig            `yaml:"users"`
	Fields        map[string]FieldConfig `yaml:"fields"`
}

// GitConfig contains git-related settings.
type GitConfig struct {
	TrunkBranch string `yaml:"trunk_branch"` // default: "" (auto-detect main/master)
	Remote      string `yaml:"remote"`       // default: "origin"
}

// StartConfig contains settings for the start command.
type StartConfig struct {
	MoveTo              string `yaml:"move_to"`               // default: "doing"
	StatusAction        string `yaml:"status_action"`         // default: "commit_and_push"
	StatusCommitMessage string `yaml:"status_commit_message"` // optional template
}

// IDEConfig contains IDE-related settings.
type IDEConfig struct {
	Command string   `yaml:"command"` // IDE command name (e.g., "cursor", "code")
	Args    []string `yaml:"args"`    // Arguments to pass to IDE command
}

// WorkspaceConfig contains workspace-related settings.
type WorkspaceConfig struct {
	Root            string          `yaml:"root"`             // default: "../"
	WorktreeRoot    string          `yaml:"worktree_root"`    // derived if not set
	ArchitectureDoc string          `yaml:"architecture_doc"` // optional path to architecture doc
	Description     string          `yaml:"description"`      // optional workspace description
	DraftPR         bool            `yaml:"draft_pr"`         // default: false
	Setup           string          `yaml:"setup"`            // optional setup command/script
	Projects        []ProjectConfig `yaml:"projects"`         // optional list of projects
}

// ProjectConfig contains project-specific settings for polyrepo workspaces.
type ProjectConfig struct {
	Name        string `yaml:"name"`         // project identifier
	Path        string `yaml:"path"`         // path to project repository
	Mount       string `yaml:"mount"`        // folder name in worktree (defaults to name)
	RepoRoot    string `yaml:"repo_root"`    // optional: groups projects sharing same root
	Kind        string `yaml:"kind"`         // app | service | library | infra
	Description string `yaml:"description"`  // optional: for LLM context
	DraftPR     *bool  `yaml:"draft_pr"`     // optional: override workspace default
	Remote      string `yaml:"remote"`       // optional: override remote name
	TrunkBranch string `yaml:"trunk_branch"` // optional: per-project trunk branch override
	Setup       string `yaml:"setup"`        // optional: project-specific setup command
}

// ValidationConfig contains validation settings for work items.
type ValidationConfig struct {
	RequiredFields []string `yaml:"required_fields"`
	IDFormat       string   `yaml:"id_format"`
	StatusValues   []string `yaml:"status_values"`
	Strict         bool     `yaml:"strict"` // If true, flag fields not in configuration
}

// CommitConfig contains git commit settings.
type CommitConfig struct {
	DefaultMessage      string `yaml:"default_message"`
	MoveSubjectTemplate string `yaml:"move_subject_template"`
	MoveBodyTemplate    string `yaml:"move_body_template"`
}

// ReleaseConfig contains release-related settings.
type ReleaseConfig struct {
	ReleasesFile      string `yaml:"releases_file"`
	ArchiveDateFormat string `yaml:"archive_date_format"`
}

// SavedUser represents a user saved in configuration.
type SavedUser struct {
	Email string `yaml:"email"`
	Name  string `yaml:"name,omitempty"`
}

// UsersConfig contains user-related settings.
type UsersConfig struct {
	UseGitHistory   *bool       `yaml:"use_git_history,omitempty"` // Defaults to true if nil
	CommitLimit     int         `yaml:"commit_limit,omitempty"`    // 0 means no limit, only when UseGitHistory is true
	IgnoredEmails   []string    `yaml:"ignored_emails"`            // Only when UseGitHistory is true
	IgnoredPatterns []string    `yaml:"ignored_patterns"`          // Only when UseGitHistory is true
	SavedUsers      []SavedUser `yaml:"saved_users"`               // Users added via configuration
}

// FieldConfig represents configuration for a custom field in work items.
type FieldConfig struct {
	Type          string      `yaml:"type"`           // string, date, email, url, number, array, enum
	Required      bool        `yaml:"required"`       // whether the field is required
	Default       interface{} `yaml:"default"`        // default value for the field
	Format        string      `yaml:"format"`         // regex pattern or date format
	AllowedValues []string    `yaml:"allowed_values"` // for enum type
	Description   string      `yaml:"description"`    // human-readable description
	DisplayName   string      `yaml:"display_name"`   // alternative display name
	Category      string      `yaml:"category"`       // optional grouping
	Deprecated    bool        `yaml:"deprecated"`     // mark field as deprecated
	MinLength     *int        `yaml:"min_length"`     // for strings/arrays
	MaxLength     *int        `yaml:"max_length"`     // for strings/arrays
	MinValue      *float64    `yaml:"min"`            // for numbers
	MaxValue      *float64    `yaml:"max"`            // for numbers
	MinDate       string      `yaml:"min_date"`       // for dates (absolute or relative)
	MaxDate       string      `yaml:"max_date"`       // for dates (absolute or relative)
	ItemType      string      `yaml:"item_type"`      // for arrays
	Unique        bool        `yaml:"unique"`         // for arrays
	Schemes       []string    `yaml:"schemes"`        // for URLs
	CaseSensitive *bool       `yaml:"case_sensitive"` // for enums (default: true if nil)
}

// HardcodedFields is a list of fields that cannot be configured.
var HardcodedFields = []string{"id", "title", "status", "kind", "created"}

const fieldTypeEnum = "enum"

// DefaultConfig provides default configuration values.
var DefaultConfig = Config{
	Version: "1.0",
	Templates: map[string]string{
		"prd":   "templates/template.prd.md",
		"issue": "templates/template.issue.md",
		"spike": "templates/template.spike.md",
		"task":  "templates/template.task.md",
	},
	StatusFolders: map[string]string{
		"backlog":  "0_backlog",
		"todo":     "1_todo",
		"doing":    "2_doing",
		"review":   "3_review",
		"done":     "4_done",
		"archived": "z_archive",
	},
	DefaultStatus: "backlog",
	Validation: ValidationConfig{
		RequiredFields: []string{"id", "title", "status", "kind", "created"},
		IDFormat:       "^\\d{3}$",
		StatusValues:   []string{"backlog", "todo", "doing", "review", "done", "released", "abandoned", "archived"},
		Strict:         false,
	},
	Commit: CommitConfig{
		DefaultMessage:      "Update work items",
		MoveSubjectTemplate: "Move {type} {id} to {target_status}",
		MoveBodyTemplate:    "{title} ({current_status} -> {target_status})",
	},
	Release: ReleaseConfig{
		ReleasesFile:      "RELEASES.md",
		ArchiveDateFormat: "2006-01-02",
	},
}

// LoadConfig loads the configuration from kira.yml file or returns defaults.
func LoadConfig() (*Config, error) {
	// Prefer root-level kira.yml; fall back to legacy .work/kira.yml if present
	rootPath := "kira.yml"
	legacyPath := filepath.Join(".work", "kira.yml")

	configPath := ""
	if _, err := os.Stat(rootPath); err == nil {
		configPath = rootPath
	} else if _, err := os.Stat(legacyPath); err == nil {
		configPath = legacyPath
	} else {
		// No config file - return a copy of defaults with all defaults applied
		config := DefaultConfig
		mergeWithDefaults(&config)
		return &config, nil
	}

	// Validate config path is safe (no path traversal)
	cleanPath := filepath.Clean(configPath)
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("invalid config path: %s", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Merge with defaults for missing fields
	mergeWithDefaults(&config)

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// ValidStatusActions defines the valid values for start.status_action
var ValidStatusActions = []string{"none", "commit_only", "commit_and_push", "commit_only_branch"}

func validateConfig(config *Config) error {
	// Validate start.move_to is a valid status key
	if config.Start != nil && config.Start.MoveTo != "" {
		if _, exists := config.StatusFolders[config.Start.MoveTo]; !exists {
			validStatuses := make([]string, 0, len(config.StatusFolders))
			for status := range config.StatusFolders {
				validStatuses = append(validStatuses, status)
			}
			sort.Strings(validStatuses)
			return fmt.Errorf("invalid status '%s': status must be one of: %s",
				config.Start.MoveTo, strings.Join(validStatuses, ", "))
		}
	}

	// Validate start.status_action is a valid value
	if config.Start != nil && config.Start.StatusAction != "" {
		valid := false
		for _, action := range ValidStatusActions {
			if config.Start.StatusAction == action {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid status_action value '%s': use one of: %s",
				config.Start.StatusAction, strings.Join(ValidStatusActions, ", "))
		}
	}

	// Validate field configuration
	if err := validateFieldConfig(config); err != nil {
		return err
	}

	return nil
}

// validateFieldConfig validates the field configuration.
func validateFieldConfig(config *Config) error {
	if config.Fields == nil {
		return nil
	}

	if err := validateNoHardcodedFields(config.Fields); err != nil {
		return err
	}

	for fieldName, fieldConfig := range config.Fields {
		if err := validateSingleFieldConfig(fieldName, &fieldConfig); err != nil {
			return err
		}
	}

	return nil
}

func validateNoHardcodedFields(fields map[string]FieldConfig) error {
	for fieldName := range fields {
		for _, hardcoded := range HardcodedFields {
			if fieldName == hardcoded {
				return fmt.Errorf("field '%s' cannot be configured and must use hardcoded validation", fieldName)
			}
		}
	}
	return nil
}

func validateSingleFieldConfig(fieldName string, fieldConfig *FieldConfig) error {
	if err := validateFieldType(fieldName, fieldConfig); err != nil {
		return err
	}
	if err := validateFieldTypeSpecifics(fieldName, fieldConfig); err != nil {
		return err
	}
	if err := validateFieldFormat(fieldName, fieldConfig); err != nil {
		return err
	}
	return validateFieldConstraints(fieldName, fieldConfig)
}

func validateFieldType(fieldName string, fieldConfig *FieldConfig) error {
	if fieldConfig.Type == "" {
		return fmt.Errorf("field '%s': type is required", fieldName)
	}
	validTypes := map[string]bool{
		"string":      true,
		"date":        true,
		"email":       true,
		"url":         true,
		"number":      true,
		"array":       true,
		fieldTypeEnum: true,
	}
	if !validTypes[fieldConfig.Type] {
		return fmt.Errorf("field '%s': invalid type '%s'. Valid types: string, date, email, url, number, array, enum", fieldName, fieldConfig.Type)
	}
	return nil
}

func validateFieldTypeSpecifics(fieldName string, fieldConfig *FieldConfig) error {
	if fieldConfig.Type == fieldTypeEnum {
		if len(fieldConfig.AllowedValues) == 0 {
			return fmt.Errorf("field '%s': enum type requires allowed_values", fieldName)
		}
	}
	if fieldConfig.Type == "array" {
		return validateArrayFieldConfig(fieldName, fieldConfig)
	}
	return nil
}

func validateArrayFieldConfig(fieldName string, fieldConfig *FieldConfig) error {
	if fieldConfig.ItemType == "" {
		return fmt.Errorf("field '%s': array type requires item_type", fieldName)
	}
	validItemTypes := map[string]bool{
		"string":      true,
		"number":      true,
		fieldTypeEnum: true,
	}
	if !validItemTypes[fieldConfig.ItemType] {
		return fmt.Errorf("field '%s': invalid item_type '%s' for array. Valid item types: string, number, enum", fieldName, fieldConfig.ItemType)
	}
	if fieldConfig.ItemType == fieldTypeEnum && len(fieldConfig.AllowedValues) == 0 {
		return fmt.Errorf("field '%s': array with enum item_type requires allowed_values", fieldName)
	}
	return nil
}

func validateFieldFormat(fieldName string, fieldConfig *FieldConfig) error {
	if fieldConfig.Format == "" {
		return nil
	}
	if fieldConfig.Type == "string" {
		if _, err := regexp.Compile(fieldConfig.Format); err != nil {
			return fmt.Errorf("field '%s': invalid regex format '%s': %w", fieldName, fieldConfig.Format, err)
		}
	}
	if fieldConfig.Type == "date" {
		// Date format validation is best-effort
		testDate := "2006-01-02"
		if _, err := time.Parse(fieldConfig.Format, testDate); err != nil {
			// Try parsing with the format itself as a test
			_, _ = time.Parse(fieldConfig.Format, fieldConfig.Format)
		}
	}
	return nil
}

func validateFieldConstraints(fieldName string, fieldConfig *FieldConfig) error {
	if fieldConfig.MinLength != nil && fieldConfig.MaxLength != nil {
		if *fieldConfig.MinLength > *fieldConfig.MaxLength {
			return fmt.Errorf("field '%s': min_length (%d) cannot be greater than max_length (%d)", fieldName, *fieldConfig.MinLength, *fieldConfig.MaxLength)
		}
	}
	if fieldConfig.MinValue != nil && fieldConfig.MaxValue != nil {
		if *fieldConfig.MinValue > *fieldConfig.MaxValue {
			return fmt.Errorf("field '%s': min (%v) cannot be greater than max (%v)", fieldName, *fieldConfig.MinValue, *fieldConfig.MaxValue)
		}
	}
	return nil
}

func mergeCommitDefaults(commit *CommitConfig) {
	if commit.DefaultMessage == "" {
		commit.DefaultMessage = DefaultConfig.Commit.DefaultMessage
	}
	if commit.MoveSubjectTemplate == "" {
		commit.MoveSubjectTemplate = DefaultConfig.Commit.MoveSubjectTemplate
	}
	if commit.MoveBodyTemplate == "" {
		commit.MoveBodyTemplate = DefaultConfig.Commit.MoveBodyTemplate
	}
}

func mergeWithDefaults(config *Config) {
	if config.Templates == nil {
		config.Templates = make(map[string]string)
	}
	for k, v := range DefaultConfig.Templates {
		if _, exists := config.Templates[k]; !exists {
			config.Templates[k] = v
		}
	}

	if config.StatusFolders == nil {
		config.StatusFolders = make(map[string]string)
	}
	for k, v := range DefaultConfig.StatusFolders {
		if _, exists := config.StatusFolders[k]; !exists {
			config.StatusFolders[k] = v
		}
	}

	if config.Validation.RequiredFields == nil {
		config.Validation.RequiredFields = DefaultConfig.Validation.RequiredFields
	}
	if config.Validation.IDFormat == "" {
		config.Validation.IDFormat = DefaultConfig.Validation.IDFormat
	}
	if config.Validation.StatusValues == nil {
		config.Validation.StatusValues = DefaultConfig.Validation.StatusValues
	}

	mergeCommitDefaults(&config.Commit)

	if config.Release.ReleasesFile == "" {
		config.Release.ReleasesFile = DefaultConfig.Release.ReleasesFile
	}
	if config.Release.ArchiveDateFormat == "" {
		config.Release.ArchiveDateFormat = DefaultConfig.Release.ArchiveDateFormat
	}

	if config.DefaultStatus == "" {
		config.DefaultStatus = DefaultConfig.DefaultStatus
	}

	mergeGitDefaults(config)
	mergeStartDefaults(config)
	mergeWorkspaceDefaults(config)
	mergeUsersDefaults(config)
	mergeFieldDefaults(config)
}

func mergeFieldDefaults(config *Config) {
	if config.Fields == nil {
		config.Fields = make(map[string]FieldConfig)
		return
	}

	// Set default CaseSensitive to true for enum fields when not explicitly set
	for fieldName := range config.Fields {
		fieldConfig := config.Fields[fieldName]
		if fieldConfig.Type == fieldTypeEnum && fieldConfig.CaseSensitive == nil {
			caseSensitive := true
			fieldConfig.CaseSensitive = &caseSensitive
			config.Fields[fieldName] = fieldConfig
		}
	}
}

func mergeGitDefaults(config *Config) {
	if config.Git == nil {
		config.Git = &GitConfig{}
	}
	// TrunkBranch defaults to "" which means auto-detect (main/master)
	// Remote defaults to "origin"
	if config.Git.Remote == "" {
		config.Git.Remote = "origin"
	}
}

func mergeStartDefaults(config *Config) {
	if config.Start == nil {
		config.Start = &StartConfig{}
	}
	if config.Start.MoveTo == "" {
		config.Start.MoveTo = "doing"
	}
	if config.Start.StatusAction == "" {
		config.Start.StatusAction = "commit_and_push"
	}
	// StatusCommitMessage defaults to empty, which will use default template at runtime
}

func mergeWorkspaceDefaults(config *Config) {
	if config.Workspace == nil {
		// No workspace config = standalone mode, nothing to merge
		return
	}
	if config.Workspace.Root == "" {
		config.Workspace.Root = "../"
	}
	// WorktreeRoot is derived at runtime if not set
	// DraftPR defaults to false (zero value)

	// Apply project defaults
	for i := range config.Workspace.Projects {
		project := &config.Workspace.Projects[i]
		// Mount defaults to project name
		if project.Mount == "" {
			project.Mount = project.Name
		}
	}
}

func mergeUsersDefaults(config *Config) {
	// UseGitHistory defaults to true if nil (not set)
	if config.Users.UseGitHistory == nil {
		useGitHistory := true
		config.Users.UseGitHistory = &useGitHistory
	}
	// CommitLimit defaults to 0 (no limit) - already zero value
	// IgnoredEmails defaults to empty slice - already zero value
	// IgnoredPatterns defaults to empty slice - already zero value
	// SavedUsers defaults to empty slice - already zero value
}

// SaveConfig saves the configuration to kira.yml in the current directory.
func SaveConfig(config *Config) error {
	return SaveConfigToDir(config, ".")
}

// SaveConfigToDir saves the config to the specified target directory under .work/kira.yml
func SaveConfigToDir(config *Config, targetDir string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to root-level kira.yml in the target directory
	configPath := filepath.Join(targetDir, "kira.yml")
	// Ensure targetDir exists
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return fmt.Errorf("failed to ensure target directory: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
