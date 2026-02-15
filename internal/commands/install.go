// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kira/internal/config"
	"kira/internal/cursorassets"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

const choiceOverwrite = "overwrite"

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Cursor skills and commands",
	Long:  `Install bundled Cursor Agent Skills and Commands to the configured path (default .agent/skills and .cursor/commands).`,
}

var installCursorSkillsCmd = &cobra.Command{
	Use:   "cursor-skills",
	Short: "Install Cursor Agent Skills",
	Long:  `Copy bundled Cursor skills to the configured path. Skills are installed as kira-<name>/ with SKILL.md and optional scripts/, references/, assets/.`,
	RunE:  runInstallCursorSkills,
}

var installCursorCommandsCmd = &cobra.Command{
	Use:   "cursor-commands",
	Short: "Install Cursor Commands",
	Long:  `Copy bundled Cursor commands to the configured path. Commands are installed as kira-<name>.md.`,
	RunE:  runInstallCursorCommands,
}

func init() {
	installCmd.AddCommand(installCursorSkillsCmd)
	installCmd.AddCommand(installCursorCommandsCmd)
	installCursorSkillsCmd.Flags().Bool("force", false, "Overwrite existing skills at the target path without prompting")
	installCursorCommandsCmd.Flags().Bool("force", false, "Overwrite existing commands at the target path without prompting")
}

func runInstallCursorSkills(cmd *cobra.Command, _ []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	force, _ := cmd.Flags().GetBool("force")
	return runInstallCursorSkillsWithOptions(cfg, force)
}

func runInstallCursorSkillsWithOptions(cfg *config.Config, force bool) error {
	return runInstallCursorSkillsWithOptionsAndSilent(cfg, force, false)
}

func runInstallCursorSkillsWithOptionsAndSilent(
	cfg *config.Config,
	force bool,
	silent bool,
) error {
	skillsPath, err := config.GetCursorSkillsPath(cfg)
	if err != nil {
		return fmt.Errorf("failed to resolve skills path: %w", err)
	}
	if err := ensureSkillsOverwriteDecision(skillsPath, force); err != nil {
		return err
	}
	if err := os.MkdirAll(skillsPath, 0o700); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}
	names, err := cursorassets.ListSkills()
	if err != nil {
		return fmt.Errorf("failed to list bundled skills: %w", err)
	}
	skillsPathAbs, err := filepath.Abs(skillsPath)
	if err != nil {
		return fmt.Errorf("failed to resolve skills path: %w", err)
	}
	for _, name := range names {
		skillContent, err := cursorassets.ReadSkillSKILL(name)
		if err != nil {
			return fmt.Errorf("skill %s: %w", name, err)
		}
		if err := validateSkillFrontmatter(name, skillContent); err != nil {
			return fmt.Errorf("skill %s: %w", name, err)
		}
		targetDir := filepath.Join(skillsPath, name)
		if err := validatePathUnder(skillsPathAbs, targetDir); err != nil {
			return fmt.Errorf("skill %s: %w", name, err)
		}
		if err := copySkillToPath(name, targetDir); err != nil {
			return fmt.Errorf("skill %s: %w", name, err)
		}
	}
	if !silent {
		fmt.Printf("%s %d skill(s) to %s:\n", successStyle("Installed"), len(names), pathStyle(skillsPath))
		for _, name := range names {
			fmt.Printf("  • %s\n", itemNameStyle(name))
		}
		fmt.Println()
	}
	return nil
}

func runInstallCursorCommands(cmd *cobra.Command, _ []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	force, _ := cmd.Flags().GetBool("force")
	return runInstallCursorCommandsWithOptions(cfg, force)
}

func runInstallCursorCommandsWithOptions(cfg *config.Config, force bool) error {
	return runInstallCursorCommandsWithOptionsAndSilent(cfg, force, false)
}

func runInstallCursorCommandsWithOptionsAndSilent(
	cfg *config.Config,
	force bool,
	silent bool,
) error {
	commandsPath, err := config.GetCursorCommandsPath(cfg)
	if err != nil {
		return fmt.Errorf("failed to resolve commands path: %w", err)
	}
	if err := ensureCommandsOverwriteDecision(commandsPath, force); err != nil {
		return err
	}
	if err := os.MkdirAll(commandsPath, 0o700); err != nil {
		return fmt.Errorf("failed to create commands directory: %w", err)
	}
	names, err := cursorassets.ListCommands()
	if err != nil {
		return fmt.Errorf("failed to list bundled commands: %w", err)
	}
	commandsPathAbs, err := filepath.Abs(commandsPath)
	if err != nil {
		return fmt.Errorf("failed to resolve commands path: %w", err)
	}
	for _, name := range names {
		content, err := cursorassets.ReadCommand(name)
		if err != nil {
			return fmt.Errorf("command %s: %w", name, err)
		}
		if err := validateCommandMarkdown(content); err != nil {
			return fmt.Errorf("command %s: %w", name, err)
		}
		destPath := filepath.Join(commandsPath, name+".md")
		if err := validatePathUnder(commandsPathAbs, destPath); err != nil {
			return fmt.Errorf("command %s: %w", name, err)
		}
		if err := os.WriteFile(destPath, content, 0o600); err != nil {
			return fmt.Errorf("command %s: %w", name, err)
		}
	}
	if !silent {
		fmt.Printf("%s %d command(s) to %s:\n", successStyle("Installed"), len(names), pathStyle(commandsPath))
		for _, name := range names {
			fmt.Printf("  • %s\n", itemNameStyle(name+".md"))
		}
		fmt.Println()
	}
	return nil
}

// EnsureCursorSkillsInstalled checks that all bundled skills exist at the configured path
// with valid SKILL.md files; if any are missing or corrupted, installs them automatically
// (no user confirmation).
func EnsureCursorSkillsInstalled(cfg *config.Config) error {
	skillsPath, err := config.GetCursorSkillsPath(cfg)
	if err != nil {
		return err
	}
	skillsPathAbs, err := filepath.Abs(skillsPath)
	if err != nil {
		return err
	}
	names, err := cursorassets.ListSkills()
	if err != nil {
		return err
	}
	for _, name := range names {
		dirPath := filepath.Join(skillsPath, name)
		info, err := os.Stat(dirPath)
		if err != nil || !info.IsDir() {
			return runInstallCursorSkillsWithOptionsAndSilent(cfg, true, true)
		}
		// Verify SKILL.md exists and has valid frontmatter
		skillMDPath := filepath.Join(dirPath, "SKILL.md")
		if err := validatePathUnder(skillsPathAbs, skillMDPath); err != nil {
			return runInstallCursorSkillsWithOptionsAndSilent(cfg, true, true)
		}
		// #nosec G304 - path validated by validatePathUnder above; components are from config and bundled assets
		content, err := os.ReadFile(skillMDPath)
		if err != nil || len(content) == 0 {
			return runInstallCursorSkillsWithOptionsAndSilent(cfg, true, true)
		}
		if err := validateSkillFrontmatter(name, content); err != nil {
			return runInstallCursorSkillsWithOptionsAndSilent(cfg, true, true)
		}
	}
	return nil
}

// EnsureCursorCommandsInstalled checks that all bundled commands exist at the configured path
// with valid non-empty content; if any are missing or corrupted, installs them automatically
// (no user confirmation).
func EnsureCursorCommandsInstalled(cfg *config.Config) error {
	commandsPath, err := config.GetCursorCommandsPath(cfg)
	if err != nil {
		return err
	}
	commandsPathAbs, err := filepath.Abs(commandsPath)
	if err != nil {
		return err
	}
	names, err := cursorassets.ListCommands()
	if err != nil {
		return err
	}
	for _, name := range names {
		cmdFilePath := filepath.Join(commandsPath, name+".md")
		info, err := os.Stat(cmdFilePath)
		if err != nil || info.IsDir() {
			return runInstallCursorCommandsWithOptionsAndSilent(cfg, true, true)
		}
		// Verify command file has valid content
		if err := validatePathUnder(commandsPathAbs, cmdFilePath); err != nil {
			return runInstallCursorCommandsWithOptionsAndSilent(cfg, true, true)
		}
		// #nosec G304 - path validated by validatePathUnder above; components are from config and bundled assets
		content, err := os.ReadFile(cmdFilePath)
		if err != nil {
			return runInstallCursorCommandsWithOptionsAndSilent(cfg, true, true)
		}
		if err := validateCommandMarkdown(content); err != nil {
			return runInstallCursorCommandsWithOptionsAndSilent(cfg, true, true)
		}
	}
	return nil
}

func ensureCommandsOverwriteDecision(commandsPath string, force bool) error {
	kiraFiles, err := listExistingKiraCommands(commandsPath)
	if err != nil {
		return err
	}
	if len(kiraFiles) == 0 {
		return nil
	}
	if force {
		return removeKiraCommandFiles(commandsPath, kiraFiles)
	}
	fmt.Printf("\n%s at %s:\n", warningStyle("Commands already exist"), pathStyle(commandsPath))
	for _, f := range kiraFiles {
		fmt.Printf("  • %s\n", itemNameStyle(f))
	}
	fmt.Printf("\nChoose: [o]verwrite, [c]ancel\n")
	fmt.Print(promptStyle("Enter choice (o/c): "))
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	choice := strings.ToLower(strings.TrimSpace(input))
	if choice == "o" || choice == choiceOverwrite {
		return removeKiraCommandFiles(commandsPath, kiraFiles)
	}
	return fmt.Errorf("install cancelled")
}

func listExistingKiraCommands(commandsPath string) ([]string, error) {
	bundledNames, err := cursorassets.ListCommands()
	if err != nil {
		return nil, fmt.Errorf("failed to list bundled commands: %w", err)
	}
	entries, err := os.ReadDir(commandsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read commands path: %w", err)
	}
	// Create a set of bundled command filenames for quick lookup
	bundledFiles := make(map[string]bool)
	for _, name := range bundledNames {
		bundledFiles[name+".md"] = true
	}
	var kiraFiles []string
	for _, e := range entries {
		if !e.IsDir() && bundledFiles[e.Name()] {
			kiraFiles = append(kiraFiles, e.Name())
		}
	}
	return kiraFiles, nil
}

func removeKiraCommandFiles(commandsPath string, files []string) error {
	for _, f := range files {
		filePath := filepath.Join(commandsPath, f)
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to remove existing command %s: %w", f, err)
		}
	}
	return nil
}

func validateCommandMarkdown(content []byte) error {
	if len(content) == 0 {
		return fmt.Errorf("command file is empty")
	}
	if strings.TrimSpace(string(content)) == "" {
		return fmt.Errorf("command file has no content")
	}
	return nil
}

func ensureSkillsOverwriteDecision(skillsPath string, force bool) error {
	kiraDirs, err := listExistingKiraSkills(skillsPath)
	if err != nil {
		return err
	}
	if len(kiraDirs) == 0 {
		return nil
	}
	if force {
		return removeKiraSkills(skillsPath, kiraDirs)
	}
	return promptAndRemoveSkills(skillsPath, kiraDirs)
}

func listExistingKiraSkills(skillsPath string) ([]string, error) {
	bundledNames, err := cursorassets.ListSkills()
	if err != nil {
		return nil, fmt.Errorf("failed to list bundled skills: %w", err)
	}
	entries, err := os.ReadDir(skillsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read skills path: %w", err)
	}
	// Create a set of bundled skill names for quick lookup
	bundledDirs := make(map[string]bool)
	for _, name := range bundledNames {
		bundledDirs[name] = true
	}
	var kiraDirs []string
	for _, e := range entries {
		if e.IsDir() && bundledDirs[e.Name()] {
			kiraDirs = append(kiraDirs, e.Name())
		}
	}
	return kiraDirs, nil
}

func removeKiraSkills(skillsPath string, dirs []string) error {
	for _, d := range dirs {
		dirPath := filepath.Join(skillsPath, d)
		if err := os.RemoveAll(dirPath); err != nil {
			return fmt.Errorf("failed to remove existing skill %s: %w", d, err)
		}
	}
	return nil
}

func promptAndRemoveSkills(skillsPath string, kiraDirs []string) error {
	fmt.Printf("\n%s at %s:\n", warningStyle("Skills already exist"), pathStyle(skillsPath))
	for _, d := range kiraDirs {
		fmt.Printf("  • %s\n", itemNameStyle(d))
	}
	fmt.Printf("\nChoose: [o]verwrite, [c]ancel\n")
	fmt.Print(promptStyle("Enter choice (o/c): "))
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	choice := strings.ToLower(strings.TrimSpace(input))
	if choice == "o" || choice == choiceOverwrite {
		return removeKiraSkills(skillsPath, kiraDirs)
	}
	return fmt.Errorf("install cancelled")
}

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func validateSkillFrontmatter(folderName string, content []byte) error {
	// Extract YAML between first --- and second ---
	lines := strings.Split(string(content), "\n")
	var inFront bool
	var yamlLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !inFront {
				inFront = true
				continue
			}
			break
		}
		if inFront {
			yamlLines = append(yamlLines, line)
		}
	}
	if len(yamlLines) == 0 {
		return fmt.Errorf("SKILL.md has no frontmatter")
	}
	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), &fm); err != nil {
		return fmt.Errorf("invalid SKILL.md frontmatter: %w", err)
	}
	if fm.Name == "" {
		return fmt.Errorf("SKILL.md frontmatter missing required 'name'")
	}
	if fm.Description == "" {
		return fmt.Errorf("SKILL.md frontmatter missing required 'description'")
	}
	// Folder name must be kira-<name> where name is the frontmatter name (with or without kira- prefix)
	expectedFolder := "kira-" + strings.TrimPrefix(fm.Name, "kira-")
	if folderName != expectedFolder {
		return fmt.Errorf("skill folder name %q does not match frontmatter name %q (expected folder %q)", folderName, fm.Name, expectedFolder)
	}
	return nil
}

func validatePathUnder(baseAbs, target string) error {
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	baseWithSep := baseAbs + string(filepath.Separator)
	if targetAbs != baseAbs && !strings.HasPrefix(targetAbs, baseWithSep) {
		return fmt.Errorf("path outside target directory")
	}
	return nil
}

func copySkillToPath(skillName, targetDir string) error {
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return err
	}
	paths, err := cursorassets.ListSkillFilePaths(skillName)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(targetDir)
	if err != nil {
		return err
	}
	for _, rel := range paths {
		if strings.Contains(rel, "..") || strings.Contains(rel, "\x00") {
			return fmt.Errorf("invalid relative path in bundle: %s", rel)
		}
		data, err := cursorassets.ReadSkillFile(skillName, rel)
		if err != nil {
			return err
		}
		destPath := filepath.Join(targetDir, filepath.FromSlash(rel))
		if err := validatePathUnder(targetAbs, destPath); err != nil {
			return fmt.Errorf("path %s: %w", rel, err)
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(destPath, data, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
	}
	return nil
}
