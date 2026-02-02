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
	Slices        *SlicesConfig          `yaml:"slices"`
	Review        *ReviewConfig          `yaml:"review"`
	DocsFolder    string                 `yaml:"docs_folder"` // default: ".docs"
	// ConfigDir is the absolute path to the directory containing kira.yml (set at load time; not persisted).
	ConfigDir string `yaml:"-"`
}

// ReviewConfig contains settings for the review (submit-for-review) command.
type ReviewConfig struct {
	TrunkUpdate *bool `yaml:"trunk_update"` // default: true (nil = run trunk update)
	Rebase      *bool `yaml:"rebase"`       // default: true (nil = run rebase)
	CommitMove  *bool `yaml:"commit_move"`  // optional: commit the move to review (align with move command)
}

// SlicesConfig contains settings for slices and tasks in work items.
type SlicesConfig struct {
	AutoUpdateStatus bool   `yaml:"auto_update_status"` // default: false
	TaskIDFormat     string `yaml:"task_id_format"`     // default: "T%03d"
	DefaultState     string `yaml:"default_state"`      // default: "open" (only open or done)
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
	WorkFolder      string          `yaml:"work_folder"`      // default: ".work"
	ArchitectureDoc string          `yaml:"architecture_doc"` // optional path to architecture doc
	Description     string          `yaml:"description"`      // optional workspace description
	DraftPR         *bool           `yaml:"draft_pr"`         // default: true (nil = enabled)
	GitPlatform     string          `yaml:"git_platform"`     // github, auto (default: auto)
	GitBaseURL      string          `yaml:"git_base_url"`     // optional; for GHE
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
	GitPlatform string `yaml:"git_platform"` // optional: override workspace default
	GitBaseURL  string `yaml:"git_base_url"` // optional: for GHE
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
	DocsFolder: ".docs",
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
		configDir, err := filepath.Abs(".")
		if err != nil {
			return nil, fmt.Errorf("failed to resolve config directory: %w", err)
		}
		config.ConfigDir = configDir
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

	configDir, err := filepath.Abs(filepath.Dir(configPath))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config directory: %w", err)
	}
	config.ConfigDir = configDir

	return &config, nil
}

// LoadConfigFromDir loads configuration from the given directory (looks for kira.yml in dir, then dir/.work/kira.yml).
// ConfigDir is set to the absolute path of dir.
func LoadConfigFromDir(dir string) (*Config, error) {
	rootPath := filepath.Join(dir, "kira.yml")
	legacyPath := filepath.Join(dir, ".work", "kira.yml")

	configPath := ""
	if _, err := os.Stat(rootPath); err == nil {
		configPath = rootPath
	} else if _, err := os.Stat(legacyPath); err == nil {
		configPath = legacyPath
	} else {
		config := DefaultConfig
		mergeWithDefaults(&config)
		configDir, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve config directory: %w", err)
		}
		config.ConfigDir = configDir
		return &config, nil
	}

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

	mergeWithDefaults(&config)
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	configDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config directory: %w", err)
	}
	config.ConfigDir = configDir

	return &config, nil
}

// GetWorkFolderPath returns the configured work folder path, defaulting to ".work".
func GetWorkFolderPath(cfg *Config) string {
	if cfg != nil && cfg.Workspace != nil && cfg.Workspace.WorkFolder != "" {
		return strings.TrimSpace(cfg.Workspace.WorkFolder)
	}
	return ".work"
}

// GetWorkFolderAbsPath returns the absolute path to the work folder, resolved relative to ConfigDir.
func GetWorkFolderAbsPath(cfg *Config) (string, error) {
	workFolder := GetWorkFolderPath(cfg)
	baseDir := "."
	if cfg != nil && cfg.ConfigDir != "" {
		baseDir = cfg.ConfigDir
	}
	absPath, err := filepath.Abs(filepath.Join(baseDir, workFolder))
	if err != nil {
		return "", fmt.Errorf("failed to resolve work folder path: %w", err)
	}
	return absPath, nil
}

// GetDocsFolderPath returns the configured docs folder path, defaulting to ".docs".
func GetDocsFolderPath(cfg *Config) string {
	if cfg != nil && strings.TrimSpace(cfg.DocsFolder) != "" {
		return strings.TrimSpace(cfg.DocsFolder)
	}
	return ".docs"
}

// DocsRoot returns the absolute path to the docs folder resolved relative to targetDir.
// It validates that the result does not escape targetDir (e.g. no .. in path).
func DocsRoot(cfg *Config, targetDir string) (string, error) {
	baseDir := targetDir
	if baseDir == "" {
		baseDir = "."
	}
	docsPath := filepath.Join(baseDir, GetDocsFolderPath(cfg))
	absDocs, err := filepath.Abs(docsPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve docs folder path: %w", err)
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve target directory: %w", err)
	}
	rel, err := filepath.Rel(absBase, absDocs)
	if err != nil {
		return "", fmt.Errorf("docs_folder path invalid: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("docs_folder path must not escape target directory")
	}
	return absDocs, nil
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

	// Validate workspace settings
	if err := validateWorkspaceConfig(config); err != nil {
		return err
	}

	// Validate field configuration
	if err := validateFieldConfig(config); err != nil {
		return err
	}

	// Validate docs_folder
	if err := validateDocsFolder(config); err != nil {
		return err
	}

	return nil
}

const maxDocsFolderPathLen = 256

// validateDocsFolder validates docs_folder: no .., no null byte, reasonable length, non-empty after trim.
func validateDocsFolder(config *Config) error {
	df := strings.TrimSpace(config.DocsFolder)
	if df == "" {
		return fmt.Errorf("docs_folder cannot be empty or whitespace only")
	}
	if strings.Contains(config.DocsFolder, "..") {
		return fmt.Errorf("docs_folder cannot contain '..'")
	}
	if strings.Contains(config.DocsFolder, "\x00") {
		return fmt.Errorf("docs_folder cannot contain null byte")
	}
	if len(config.DocsFolder) > maxDocsFolderPathLen {
		return fmt.Errorf("docs_folder path too long (max %d)", maxDocsFolderPathLen)
	}
	return nil
}

// validateWorkspaceConfig validates workspace-level settings (work_folder, git_platform).
func validateWorkspaceConfig(config *Config) error {
	if config.Workspace == nil {
		return nil
	}
	if config.Workspace.WorkFolder != "" {
		wf := strings.TrimSpace(config.Workspace.WorkFolder)
		if wf == "" {
			return fmt.Errorf("workspace.work_folder cannot be empty or whitespace only")
		}
		if strings.Contains(config.Workspace.WorkFolder, "\x00") {
			return fmt.Errorf("workspace.work_folder cannot contain null byte")
		}
	}
	if config.Workspace.GitPlatform != "" {
		validPlatforms := []string{"github", "auto"}
		for _, p := range validPlatforms {
			if config.Workspace.GitPlatform == p {
				return nil
			}
		}
		return fmt.Errorf("invalid workspace.git_platform value '%s': use one of: %s",
			config.Workspace.GitPlatform, strings.Join(validPlatforms, ", "))
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
		// Reject empty field names as they are likely configuration errors
		if fieldName == "" {
			return fmt.Errorf("field name cannot be empty")
		}
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
		// Validate date format by checking if it produces different outputs for different times
		// If two different times produce the same formatted output, the format is invalid (just literal text)
		testTime1 := time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)
		testTime2 := time.Date(2007, 2, 3, 16, 5, 6, 0, time.UTC)
		formatted1 := testTime1.Format(fieldConfig.Format)
		formatted2 := testTime2.Format(fieldConfig.Format)

		// If both times produce the same output, the format is just literal text (invalid)
		if formatted1 == formatted2 {
			return fmt.Errorf("field '%s': invalid date format '%s': format does not contain time components", fieldName, fieldConfig.Format)
		}

		// Verify the format can round-trip by parsing the formatted output
		if _, err := time.Parse(fieldConfig.Format, formatted1); err != nil {
			return fmt.Errorf("field '%s': invalid date format '%s': %w", fieldName, fieldConfig.Format, err)
		}
	}
	return nil
}

func validateFieldConstraints(fieldName string, fieldConfig *FieldConfig) error {
	// Validate min_length is non-negative
	if fieldConfig.MinLength != nil && *fieldConfig.MinLength < 0 {
		return fmt.Errorf("field '%s': min_length (%d) cannot be negative", fieldName, *fieldConfig.MinLength)
	}
	// Validate max_length is non-negative
	if fieldConfig.MaxLength != nil && *fieldConfig.MaxLength < 0 {
		return fmt.Errorf("field '%s': max_length (%d) cannot be negative", fieldName, *fieldConfig.MaxLength)
	}
	// Validate min_length <= max_length
	if fieldConfig.MinLength != nil && fieldConfig.MaxLength != nil {
		if *fieldConfig.MinLength > *fieldConfig.MaxLength {
			return fmt.Errorf("field '%s': min_length (%d) cannot be greater than max_length (%d)", fieldName, *fieldConfig.MinLength, *fieldConfig.MaxLength)
		}
	}
	// Validate that max is not negative when min is not set or is non-negative
	// This prevents cases where max: -5 would reject all positive values
	// This check must come before the min > max check to provide a more specific error
	if fieldConfig.MaxValue != nil && *fieldConfig.MaxValue < 0 {
		if fieldConfig.MinValue == nil || *fieldConfig.MinValue >= 0 {
			return fmt.Errorf("field '%s': max (%v) cannot be negative when min is not set or is non-negative", fieldName, *fieldConfig.MaxValue)
		}
	}
	// Validate numeric min/max constraints
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

	if config.DocsFolder == "" {
		config.DocsFolder = ".docs"
	}

	mergeGitDefaults(config)
	mergeStartDefaults(config)
	mergeReviewDefaults(config)
	mergeWorkspaceDefaults(config)
	mergeUsersDefaults(config)
	mergeSlicesDefaults(config)
	mergeFieldDefaults(config)
}

func mergeSlicesDefaults(config *Config) {
	if config.Slices == nil {
		config.Slices = &SlicesConfig{}
	}
	if config.Slices.TaskIDFormat == "" {
		config.Slices.TaskIDFormat = "T%03d"
	}
	if config.Slices.DefaultState == "" {
		config.Slices.DefaultState = "open"
	}
	// AutoUpdateStatus defaults to false (zero value)
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

func mergeReviewDefaults(config *Config) {
	if config.Review == nil {
		config.Review = &ReviewConfig{}
	}
	// TrunkUpdate and Rebase default to true when nil (run update/rebase by default)
	// CommitMove defaults to false when nil (no commit of move by default)
}

func mergeWorkspaceDefaults(config *Config) {
	if config.Workspace == nil {
		// No workspace config = standalone mode, nothing to merge
		return
	}
	if config.Workspace.Root == "" {
		config.Workspace.Root = "../"
	}
	if config.Workspace.WorkFolder == "" {
		config.Workspace.WorkFolder = ".work"
	}
	// WorktreeRoot is derived at runtime if not set
	// DraftPR defaults to true when nil (draft PR enabled by default)
	if config.Workspace.DraftPR == nil {
		enabled := true
		config.Workspace.DraftPR = &enabled
	}

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
