package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kira/internal/config"
	"kira/internal/templates"

	"github.com/spf13/cobra"
)

// docsSubdirs are the standard subdirectories under the docs folder (relative paths).
var docsSubdirs = []string{
	"agents", "architecture", "product", "reports", "guides", "api",
	"guides/security",
}

var initCmd = &cobra.Command{
	Use:   "init [folder]",
	Short: "Initialize a kira workspace",
	Long:  `Creates the files and folders used by kira in the specified directory.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := "."
		if len(args) > 0 {
			targetDir = args[0]
		}

		cfg, err := config.LoadConfigFromDir(targetDir)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		force, _ := cmd.Flags().GetBool("force")
		fillMissing, _ := cmd.Flags().GetBool("fill-missing")
		workPath := filepath.Join(targetDir, config.GetWorkFolderPath(cfg))
		docsPath := filepath.Join(targetDir, config.GetDocsFolderPath(cfg))
		if err := ensureWorkspaceDecision(workPath, docsPath, force, fillMissing); err != nil {
			return err
		}

		return initializeWorkspace(targetDir, cfg)
	},
}

func init() {
	initCmd.Flags().Bool("force", false, "Overwrite existing work folder if present")
	initCmd.Flags().Bool("fill-missing", false, "Create any missing files/folders without overwriting existing ones")
}

func initializeWorkspace(targetDir string, cfg *config.Config) error {
	// Create work directory (configured work folder)
	workDir := filepath.Join(targetDir, config.GetWorkFolderPath(cfg))
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}

	// Create status folders and .gitkeep files
	for _, folder := range cfg.StatusFolders {
		folderPath := filepath.Join(workDir, folder)
		if err := os.MkdirAll(folderPath, 0o700); err != nil {
			return fmt.Errorf("failed to create folder %s: %w", folder, err)
		}
		if err := os.WriteFile(filepath.Join(folderPath, ".gitkeep"), []byte(""), 0o600); err != nil {
			return fmt.Errorf("failed to create .gitkeep in %s: %w", folder, err)
		}
	}

	// Create templates directory and default templates and .gitkeep
	if err := templates.CreateDefaultTemplates(workDir); err != nil {
		return fmt.Errorf("failed to create default templates: %w", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "templates", ".gitkeep"), []byte(""), 0o600); err != nil {
		return fmt.Errorf("failed to create .gitkeep in templates: %w", err)
	}

	// Create or preserve IDEAS.md file (prepend header if missing)
	ideasPath := filepath.Join(workDir, "IDEAS.md")
	header := `# Ideas

This file is for capturing quick ideas and thoughts that don't fit into formal work items yet.

## How to use
- Add ideas with timestamps using ` + "`kira idea add \"your idea here\"`" + `
- Or manually add entries below

## List

`
	if _, err := os.Stat(ideasPath); os.IsNotExist(err) {
		if err := os.WriteFile(ideasPath, []byte(header), 0o600); err != nil {
			return fmt.Errorf("failed to create IDEAS.md: %w", err)
		}
	} else {
		content, readErr := safeReadFile(ideasPath, cfg)
		if readErr != nil {
			return fmt.Errorf("failed to read IDEAS.md: %w", readErr)
		}
		if !strings.HasPrefix(string(content), "# Ideas") {
			newContent := header + string(content)
			if err := os.WriteFile(ideasPath, []byte(newContent), 0o600); err != nil {
				return fmt.Errorf("failed to update IDEAS.md: %w", err)
			}
		}
	}

	// Create docs folder and standard subdirs
	if err := initializeDocsFolder(targetDir, cfg); err != nil {
		return err
	}

	// Create kira.yml config file under the target directory
	if err := config.SaveConfigToDir(cfg, targetDir); err != nil {
		return fmt.Errorf("failed to create kira.yml: %w", err)
	}

	fmt.Printf("Initialized kira workspace in %s\n", targetDir)
	return nil
}

func initializeDocsFolder(targetDir string, cfg *config.Config) error {
	docsRoot := filepath.Join(targetDir, config.GetDocsFolderPath(cfg))
	if err := os.MkdirAll(docsRoot, 0o700); err != nil {
		return fmt.Errorf("failed to create docs directory: %w", err)
	}
	for _, sub := range docsSubdirs {
		subPath := filepath.Join(docsRoot, sub)
		if err := os.MkdirAll(subPath, 0o700); err != nil {
			return fmt.Errorf("failed to create docs subfolder %s: %w", sub, err)
		}
	}
	return writeDocsIndexFiles(docsRoot)
}

// docsIndexEntries defines relative path (under docs root) and README content. Empty path = docs root.
var docsIndexEntries = []struct {
	path    string
	content string
}{
	{"", `# Documentation

Overview of project documentation. Use this folder for long-lived reference material (ADRs, guides, product docs, reports). Work items and specs live in .work instead.

## Sections

- [Agents](agents/) – Agent-specific documentation (e.g. using kira)
- [Architecture](architecture/) – Architecture Decision Records and diagrams
- [Product](product/) – Product vision, roadmap, personas, glossary, feature briefs
- [Reports](reports/) – Release reports, metrics, audits, retrospectives
- [API](api/) – API reference
- [Guides](guides/) – Development and usage guides (including security)
`},
	{"agents", `# Agent documentation

Documentation for agents and tooling (e.g. [using-kira](using-kira.md)).
`},
	{"architecture", `# Architecture

Architecture Decision Records (ADRs) and system design documents.
`},
	{"product", `# Product

Product vision, roadmap, personas, glossary, feature briefs, and commercials.
`},
	{"reports", `# Reports

Release reports, metrics summaries, audits, and retrospectives.
`},
	{"api", `# API

API reference documentation.
`},
	{"guides", `# Guides

Development and usage guides. See [security/](security/) for security guidelines.
`},
	{"guides/security", `# Security

Security guidelines (e.g. [golang-secure-coding](golang-secure-coding.md)).
`},
}

func writeDocsIndexFiles(docsRoot string) error {
	for _, e := range docsIndexEntries {
		dir := docsRoot
		if e.path != "" {
			dir = filepath.Join(docsRoot, e.path)
		}
		readmePath := filepath.Join(dir, "README.md")
		if _, err := os.Stat(readmePath); err == nil {
			continue
		}
		if err := os.WriteFile(readmePath, []byte(e.content), 0o600); err != nil {
			return fmt.Errorf("failed to write %s: %w", readmePath, err)
		}
	}
	return nil
}

func removePathIfExists(path, kind string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove existing %s: %w", kind, err)
	}
	return nil
}

func ensureWorkspaceDecision(workPath, docsPath string, force, fillMissing bool) error {
	workExists := pathExists(workPath)
	docsExists := pathExists(docsPath)
	if !workExists && !docsExists {
		return nil
	}
	if force {
		_ = removePathIfExists(workPath, "work folder")
		_ = removePathIfExists(docsPath, "docs folder")
		return nil
	}
	if fillMissing {
		return nil
	}

	fmt.Printf("Workspace (.work and docs) already exists. Choose an option: [c]ancel, [o]verwrite, [f]ill-missing\n")
	fmt.Print("Enter choice (c/o/f): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	choice := strings.ToLower(strings.TrimSpace(input))
	if choice == "f" || choice == "fill-missing" {
		return nil
	}
	if choice == "o" || choice == choiceOverwrite {
		_ = removePathIfExists(workPath, "work folder")
		_ = removePathIfExists(docsPath, "docs folder")
		return nil
	}
	return fmt.Errorf("init cancelled")
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
