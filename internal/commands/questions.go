package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kira/internal/config"

	"github.com/spf13/cobra"
)

var questionsCmd = &cobra.Command{
	Use:   "questions",
	Short: "List unanswered clarifying questions in work items and docs",
	Long: `Scans markdown (.md) and Quarto (.qmd) under the configured work and docs folders.

Further behaviour (parsing, filters, and listing questions) is added in subsequent slices.`,
	Args:          cobra.NoArgs,
	RunE:          runQuestions,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(questionsCmd)
}

func runQuestions(*cobra.Command, []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	workAbs, err := config.GetWorkFolderAbsPath(cfg)
	if err != nil {
		return fmt.Errorf("failed to resolve work folder: %w", err)
	}
	docsAbs, err := config.DocsRoot(cfg, cfg.ConfigDir)
	if err != nil {
		return fmt.Errorf("failed to resolve docs folder: %w", err)
	}
	_, err = discoverMarkdownFiles(workAbs, docsAbs)
	return err
}

// discoverMarkdownFiles returns all .md and .qmd file paths under work and docs roots (validated under each root).
func discoverMarkdownFiles(workAbs, docsAbs string) ([]string, error) {
	var paths []string
	if err := walkMarkdownRoot(workAbs, &paths); err != nil {
		return nil, err
	}
	docsPaths, err := walkDocsMarkdownRoot(docsAbs)
	if err != nil {
		return nil, err
	}
	paths = append(paths, docsPaths...)
	return paths, nil
}

func walkMarkdownRoot(rootAbs string, paths *[]string) error {
	return filepath.WalkDir(rootAbs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isMarkdownFile(path) {
			return nil
		}
		if err := validatePathUnderRoot(rootAbs, path); err != nil {
			return err
		}
		*paths = append(*paths, path)
		return nil
	})
}

func walkDocsMarkdownRoot(docsAbs string) ([]string, error) {
	st, err := os.Stat(docsAbs)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("docs folder: %w", err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("docs path is not a directory: %s", docsAbs)
	}
	var paths []string
	err = filepath.WalkDir(docsAbs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isMarkdownFile(path) {
			return nil
		}
		if err := validatePathUnderRoot(docsAbs, path); err != nil {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	return paths, err
}

func validatePathUnderRoot(rootAbs, filePath string) error {
	cleanPath := filepath.Clean(filePath)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	rootWithSep := rootAbs + string(filepath.Separator)
	if !strings.HasPrefix(absPath+string(filepath.Separator), rootWithSep) && absPath != rootAbs {
		return fmt.Errorf("path outside allowed root: %s", filePath)
	}
	return nil
}

func isMarkdownFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".qmd"
}
