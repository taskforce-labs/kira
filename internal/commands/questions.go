package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"kira/internal/config"

	"github.com/spf13/cobra"
)

const questionsOutputText = "text"

// QuestionRecord is one row for JSON output.
type QuestionRecord struct {
	File     string `json:"file"`
	Question string `json:"question"`
}

var questionsCmd = &cobra.Command{
	Use:   "questions",
	Short: "List unanswered clarifying questions in work items and docs",
	Long: `Scans markdown (.md) and Quarto (.qmd) under the configured work and docs folders.

Only content under a level-2 heading exactly "## Questions" is considered. Each question uses:

  ### N. Short title
  Optional context, then:
  #### Options
  - [ ] Option A
  - [x] Option B

A question is unanswered if there is no "#### Options" block, or no line uses a checked checkbox (- [x] or - [X]).

Location and filename filters (--work, --docs, --status, --doc-type) are added in the next slice.`,
	Args:          cobra.NoArgs,
	RunE:          runQuestions,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	questionsCmd.Flags().String("output", questionsOutputText, "Output format: text or json")
	rootCmd.AddCommand(questionsCmd)
}

func runQuestions(cmd *cobra.Command, _ []string) error {
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

	outputFormat, _ := cmd.Flags().GetString("output")
	if outputFormat != questionsOutputText && outputFormat != sliceLintOutputJSON {
		return fmt.Errorf("invalid output format %q: use text or json", outputFormat)
	}

	files, err := discoverMarkdownFiles(workAbs, docsAbs)
	if err != nil {
		return err
	}

	records, err := buildQuestionRecords(cfg, files)
	if err != nil {
		return err
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].File != records[j].File {
			return records[i].File < records[j].File
		}
		return records[i].Question < records[j].Question
	})

	return writeQuestionsOutput(cmd.OutOrStdout(), outputFormat, records)
}

func buildQuestionRecords(cfg *config.Config, files []string) ([]QuestionRecord, error) {
	records := make([]QuestionRecord, 0)
	for _, p := range files {
		rel, err := filepath.Rel(cfg.ConfigDir, p)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("invalid path for display: %s", p)
		}
		// #nosec G304 - path has been validated by discoverMarkdownFiles
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		entries := ParseQuestionsFromMarkdown(data)
		for _, e := range UnansweredQuestions(entries) {
			records = append(records, QuestionRecord{File: rel, Question: QuestionDisplayText(e)})
		}
	}
	return records, nil
}

func writeQuestionsOutput(out io.Writer, format string, records []QuestionRecord) error {
	if format == sliceLintOutputJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(records)
	}
	for _, r := range records {
		if _, err := fmt.Fprintf(out, "%s: %s\n", r.File, r.Question); err != nil {
			return err
		}
	}
	return nil
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
