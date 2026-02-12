// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"kira/internal/config"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage kira configuration",
	Long:  `Manage and query kira configuration values.`,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long: `Get an effective configuration value by key.

Supported keys:
  - trunk_branch: Git trunk branch (resolved with auto-detect)
  - remote: Git remote name
  - work_folder: Work folder path (default: .work)
  - work_folder_abs: Absolute path to work folder
  - docs_folder: Docs folder path (default: .docs)
  - docs_folder_abs: Absolute path to docs folder
  - config_dir: Absolute path to directory containing kira.yml
  - ide.command: IDE command name
  - ide.args: IDE arguments (list)
  - project_names: List of project names (polyrepo only)
  - project_path: Absolute path to project repository (requires --project)

Path syntax:
  Keys containing a dot (.) are treated as paths into the merged config.
  Examples: git.trunk_branch, workspace.work_folder, ide.command
  Use numeric index for arrays: workspace.projects.0.name

Polyrepo:
  Use --project <name> to get per-project values for trunk_branch, remote, or project_path.
  Use --project '*' or --project all to get values for all projects.`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

func init() {
	configGetCmd.Flags().String("output", "text", "Output format: text or json")
	configGetCmd.Flags().String("project", "", "Project name (for polyrepo). Use '*' or 'all' for all projects.")
	configCmd.AddCommand(configGetCmd)
}

// curatedKeys is a set of keys that should be treated as curated keys even if they contain dots
var curatedKeys = map[string]bool{
	"ide.command": true,
	"ide.args":    true,
}

// Constants for key names
const (
	keyTrunkBranch  = "trunk_branch"
	keyRemote       = "remote"
	keyProjectPath  = "project_path"
	keyWorkFolder   = "work_folder"
	keyDocsFolder   = "docs_folder"
	keyConfigDir    = "config_dir"
	keyIDECommand   = "ide.command"
	keyIDEArgs      = "ide.args"
	keyProjectNames = "project_names"
)

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	outputFormat, _ := cmd.Flags().GetString("output")
	projectName, _ := cmd.Flags().GetString("project")

	// Validate output format
	if outputFormat != "text" && outputFormat != sliceLintOutputJSON {
		return fmt.Errorf("invalid output format '%s': use 'text' or 'json'", outputFormat)
	}

	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if key is a curated key (exact match)
	if curatedKeys[key] {
		// Handle as curated key
		return handleCuratedKey(cfg, key, projectName, outputFormat)
	}

	// Check if key contains a dot - treat as path syntax
	if strings.Contains(key, ".") {
		// Handle as path syntax
		return handlePathKey(cfg, key, outputFormat)
	}

	// Check if it's a single-segment path (e.g., "git", "workspace") - try as path first
	// If path lookup fails, fall back to curated key handling
	value, err := getPathValue(cfg, key)
	if err == nil {
		// Successfully found as path - output it
		return outputValue(value, outputFormat, os.Stdout)
	}

	// Not a path, handle as curated key
	return handleCuratedKey(cfg, key, projectName, outputFormat)
}

func handlePathKey(cfg *config.Config, path, outputFormat string) error {
	value, err := getPathValue(cfg, path)
	if err != nil {
		return err
	}

	return outputValue(value, outputFormat, os.Stdout)
}

func handleCuratedKey(cfg *config.Config, key, projectName, outputFormat string) error {
	// Check if project flag is used
	if projectName != "" {
		// Handle "all projects" mode
		if projectName == "*" || projectName == "all" {
			return handleAllProjects(cfg, key, outputFormat)
		}
		// Handle single project
		return handleSingleProject(cfg, key, projectName, outputFormat)
	}

	// Handle workspace-level keys
	return handleWorkspaceKey(cfg, key, outputFormat)
}

func handleWorkspaceKey(cfg *config.Config, key, outputFormat string) error {
	value, err := getWorkspaceKeyValue(cfg, key)
	if err != nil {
		return err
	}

	return outputValue(value, outputFormat, os.Stdout)
}

func getWorkspaceKeyValue(cfg *config.Config, key string) (interface{}, error) {
	switch key {
	case keyTrunkBranch:
		return getTrunkBranch(cfg, "")
	case keyRemote:
		return resolveRemoteName(cfg, nil), nil
	case keyWorkFolder:
		return config.GetWorkFolderPath(cfg), nil
	case "work_folder_abs":
		return config.GetWorkFolderAbsPath(cfg)
	case keyDocsFolder:
		return config.GetDocsFolderPath(cfg), nil
	case "docs_folder_abs":
		return config.DocsRoot(cfg, cfg.ConfigDir)
	case keyConfigDir:
		return getConfigDir(cfg)
	case keyIDECommand:
		return getIDECommand(cfg), nil
	case keyIDEArgs:
		return getIDEArgs(cfg), nil
	case keyProjectNames:
		return getProjectNames(cfg), nil
	case keyProjectPath:
		return nil, fmt.Errorf("project_path requires --project flag")
	default:
		return nil, fmt.Errorf("unknown key: '%s'\n\nValid keys: trunk_branch, remote, work_folder, work_folder_abs, docs_folder, docs_folder_abs, config_dir, ide.command, ide.args, project_names, project_path", key)
	}
}

func getConfigDir(cfg *config.Config) (interface{}, error) {
	if cfg.ConfigDir == "" {
		return nil, fmt.Errorf("config_dir not available: config directory could not be resolved")
	}
	return cfg.ConfigDir, nil
}

func getIDECommand(cfg *config.Config) string {
	if cfg.IDE != nil {
		return cfg.IDE.Command
	}
	return ""
}

func getIDEArgs(cfg *config.Config) []string {
	if cfg.IDE != nil && len(cfg.IDE.Args) > 0 {
		return cfg.IDE.Args
	}
	return []string{}
}

func handleSingleProject(cfg *config.Config, key, projectName, outputFormat string) error {
	// Validate project exists
	if cfg.Workspace == nil || len(cfg.Workspace.Projects) == 0 {
		return fmt.Errorf("no projects defined; --project is for polyrepo only")
	}

	// Find project
	var projectConfig *config.ProjectConfig
	for i := range cfg.Workspace.Projects {
		if cfg.Workspace.Projects[i].Name == projectName {
			projectConfig = &cfg.Workspace.Projects[i]
			break
		}
	}

	if projectConfig == nil {
		validNames := make([]string, len(cfg.Workspace.Projects))
		for i, p := range cfg.Workspace.Projects {
			validNames[i] = p.Name
		}
		return fmt.Errorf("unknown project: '%s'\n\nValid project names: %s", projectName, strings.Join(validNames, ", "))
	}

	// Check if key supports --project
	switch key {
	case keyTrunkBranch:
		value, err := getTrunkBranchForProject(cfg, projectConfig)
		if err != nil {
			return err
		}
		return outputValue(value, outputFormat, os.Stdout)
	case keyRemote:
		value := resolveRemoteName(cfg, projectConfig)
		return outputValue(value, outputFormat, os.Stdout)
	case keyProjectPath:
		value, err := getProjectPath(projectConfig)
		if err != nil {
			return err
		}
		return outputValue(value, outputFormat, os.Stdout)
	default:
		return fmt.Errorf("key '%s' does not support --project flag\n\nKeys that support --project: trunk_branch, remote, project_path", key)
	}
}

func handleAllProjects(cfg *config.Config, key, outputFormat string) error {
	// Check if key supports --project
	if key != keyTrunkBranch && key != keyRemote && key != keyProjectPath {
		return fmt.Errorf("key '%s' does not support --project flag\n\nKeys that support --project: trunk_branch, remote, project_path", key)
	}

	if cfg.Workspace == nil || len(cfg.Workspace.Projects) == 0 {
		// Empty output for all projects when no projects exist
		if outputFormat == sliceLintOutputJSON {
			_, _ = fmt.Println("{}")
		}
		return nil
	}

	results, err := getAllProjectsValues(cfg, key)
	if err != nil {
		return err
	}

	return outputAllProjectsResults(results, outputFormat)
}

func getAllProjectsValues(cfg *config.Config, key string) (map[string]interface{}, error) {
	repoRoot, err := getRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository root: %w", err)
	}

	projects, err := resolvePolyrepoProjects(cfg, repoRoot)
	if err != nil {
		return nil, err
	}

	results := make(map[string]interface{})
	for _, p := range projects {
		if p.Path == "" {
			continue
		}

		value, err := getProjectKeyValue(cfg, key, p)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %s for project %s: %w", key, p.Name, err)
		}

		results[p.Name] = value
	}

	return results, nil
}

func getProjectKeyValue(cfg *config.Config, key string, p PolyrepoProject) (interface{}, error) {
	switch key {
	case keyTrunkBranch:
		projectConfig := findProjectConfig(cfg, p.Name)
		return getTrunkBranchForProject(cfg, projectConfig)
	case keyRemote:
		projectConfig := findProjectConfig(cfg, p.Name)
		return resolveRemoteName(cfg, projectConfig), nil
	case keyProjectPath:
		return p.Path, nil
	default:
		return nil, fmt.Errorf("unsupported key: %s", key)
	}
}

func outputAllProjectsResults(results map[string]interface{}, outputFormat string) error {
	if outputFormat == sliceLintOutputJSON {
		jsonBytes, err := json.Marshal(results)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		_, err = fmt.Println(string(jsonBytes))
		return err
	}

	for name, val := range results {
		if _, err := fmt.Printf("%s: %v\n", name, val); err != nil {
			return err
		}
	}

	return nil
}

// getTrunkBranch determines the trunk branch using priority order:
// 1. git.trunk_branch config
// 2. Auto-detect: check "main" first, then "master"
// Returns error if both main and master exist (ambiguous) or neither exists
func getTrunkBranch(cfg *config.Config, dir string) (string, error) {
	// Priority 1: Config value
	if cfg.Git != nil && cfg.Git.TrunkBranch != "" {
		exists, err := branchExists(cfg.Git.TrunkBranch, dir, false)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("trunk branch '%s' not found: configured branch does not exist and auto-detection failed. Verify the branch name in `git.trunk_branch` configuration or ensure 'main' or 'master' branch exists", cfg.Git.TrunkBranch)
		}
		return cfg.Git.TrunkBranch, nil
	}

	// Priority 2: Auto-detect
	return autoDetectTrunkBranch(dir, false)
}

func getTrunkBranchForProject(cfg *config.Config, project *config.ProjectConfig) (string, error) {
	if project == nil {
		return "", fmt.Errorf("project config is nil")
	}

	repoRoot, err := getRepoRoot()
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}

	// Resolve project path
	projectPath := project.Path
	if projectPath == "" {
		return "", fmt.Errorf("project '%s' has no path configured", project.Name)
	}

	if !filepath.IsAbs(projectPath) {
		projectPath = filepath.Join(repoRoot, projectPath)
	}
	projectPath = filepath.Clean(projectPath)

	// Priority 1: Project-level trunk_branch
	if project.TrunkBranch != "" {
		exists, err := branchExists(project.TrunkBranch, projectPath, false)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("trunk branch '%s' not found for project '%s'", project.TrunkBranch, project.Name)
		}
		return project.TrunkBranch, nil
	}

	// Priority 2: Global git.trunk_branch
	if cfg.Git != nil && cfg.Git.TrunkBranch != "" {
		exists, err := branchExists(cfg.Git.TrunkBranch, projectPath, false)
		if err != nil {
			return "", err
		}
		if !exists {
			return "", fmt.Errorf("trunk branch '%s' not found for project '%s'", cfg.Git.TrunkBranch, project.Name)
		}
		return cfg.Git.TrunkBranch, nil
	}

	// Priority 3: Auto-detect in project path
	return autoDetectTrunkBranch(projectPath, false)
}

func getProjectPath(project *config.ProjectConfig) (string, error) {
	if project == nil {
		return "", fmt.Errorf("project config is nil")
	}

	if project.Path == "" {
		return "", fmt.Errorf("project '%s' has no path configured", project.Name)
	}

	repoRoot, err := getRepoRoot()
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}

	// Resolve absolute path
	projectPath := project.Path
	if !filepath.IsAbs(projectPath) {
		projectPath = filepath.Join(repoRoot, projectPath)
	}
	projectPath = filepath.Clean(projectPath)

	return projectPath, nil
}

func getProjectNames(cfg *config.Config) []string {
	if cfg.Workspace == nil || len(cfg.Workspace.Projects) == 0 {
		return []string{}
	}
	names := make([]string, len(cfg.Workspace.Projects))
	for i, p := range cfg.Workspace.Projects {
		names[i] = p.Name
	}
	return names
}

func getPathValue(cfg *config.Config, pathStr string) (interface{}, error) {
	// Serialize config to map for easier path traversal
	configBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize config: %w", err)
	}

	var configMap map[string]interface{}
	if err := yaml.Unmarshal(configBytes, &configMap); err != nil {
		return nil, fmt.Errorf("failed to deserialize config: %w", err)
	}

	// Parse path segments
	segments := strings.Split(pathStr, ".")
	if len(segments) == 0 {
		return nil, fmt.Errorf("invalid path: empty path")
	}

	// Walk the map by path segments
	current := interface{}(configMap)
	for i, segment := range segments {
		// Handle array index (e.g., workspace.projects.0.name)
		if idx, err := strconv.Atoi(segment); err == nil {
			// Segment is a numeric index - current must be a slice
			slice, ok := current.([]interface{})
			if !ok {
				return nil, fmt.Errorf("path not found: '%s' (segment '%s' is not an array)", pathStr, strings.Join(segments[:i], "."))
			}
			if idx < 0 || idx >= len(slice) {
				return nil, fmt.Errorf("invalid index in path '%s': index %d out of range (array has %d elements)", pathStr, idx, len(slice))
			}
			current = slice[idx]
		} else {
			// Segment is a map key
			mp, ok := current.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("path not found: '%s' (segment '%s' is not a map)", pathStr, strings.Join(segments[:i], "."))
			}
			value, exists := mp[segment]
			if !exists {
				return nil, fmt.Errorf("path not found: '%s' (key '%s' not found)", pathStr, strings.Join(segments[:i+1], "."))
			}
			current = value
		}
	}

	return current, nil
}

func outputValue(value interface{}, format string, w *os.File) error {
	switch format {
	case sliceLintOutputJSON:
		return outputJSON(value, w)
	case "text":
		return outputText(value, w)
	default:
		return fmt.Errorf("unknown output format: %s", format)
	}
}

func outputJSON(value interface{}, w *os.File) error {
	var jsonBytes []byte
	var err error

	switch v := value.(type) {
	case []string:
		jsonBytes, err = json.Marshal(v)
	case []interface{}:
		jsonBytes, err = json.Marshal(v)
	case map[string]interface{}:
		jsonBytes, err = json.Marshal(v)
	default:
		// For scalars, output as JSON value
		jsonBytes, err = json.Marshal(v)
	}

	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(w, string(jsonBytes))
	return err
}

func outputText(value interface{}, w *os.File) error {
	switch v := value.(type) {
	case []string:
		for _, item := range v {
			if _, err := fmt.Fprintln(w, item); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, item := range v {
			if _, err := fmt.Fprintln(w, item); err != nil {
				return err
			}
		}
	case map[string]interface{}:
		// For objects, output as JSON in text mode
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, string(jsonBytes))
		return err
	default:
		// For scalars, output value with newline
		_, err := fmt.Fprintln(w, v)
		return err
	}
	return nil
}
