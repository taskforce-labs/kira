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

A question is unanswered if there is no "#### Options" block, or no line uses a checked checkbox (- [x] or - [X]). Other variants of "## Questions" (e.g. "## Questions to Answer") are ignored.

Optional filters:
  --work / --docs   Limit search to the work folder or docs folder (default: both).
  --status          Restrict work files to these status subfolders (e.g. doing, backlog).
  --doc-type        Include only typed filenames (*.<type>.md); matching is case-insensitive.
  --no-doc-type     Include only untyped filenames (basename has a single dot before .md/.qmd).
  --doc-type and --no-doc-type together include the union of both sets.`,
	Args:          cobra.NoArgs,
	RunE:          runQuestions,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	questionsCmd.Flags().Bool("work", false, "Search only the work folder")
	questionsCmd.Flags().Bool("docs", false, "Search only the docs folder")
	questionsCmd.Flags().StringSlice("status", nil, "Restrict work files to these status folders (comma-separated or repeatable; e.g. doing,backlog)")
	questionsCmd.Flags().StringSlice("doc-type", nil, "Include only files whose basename type matches (e.g. prd, adr); comma-separated or repeatable; case-insensitive")
	questionsCmd.Flags().Bool("no-doc-type", false, "Include only untyped files (basename like README.md, not *.type.md)")
	questionsCmd.Flags().String("output", questionsOutputText, "Output format: text or json")
	rootCmd.AddCommand(questionsCmd)
}

type questionsRunOpts struct {
	searchWork      bool
	searchDocs      bool
	statuses        []string
	docTypes        []string
	noDocType       bool
	docTypeFilterOn bool
	outputFormat    string
}

func resolveSearchRoots(workOnly, docsOnly bool) (searchWork, searchDocs bool) {
	switch {
	case workOnly && !docsOnly:
		return true, false
	case docsOnly && !workOnly:
		return false, true
	default:
		return true, true
	}
}

func parseQuestionsRunFlags(cmd *cobra.Command) (questionsRunOpts, error) {
	workOnly, _ := cmd.Flags().GetBool("work")
	docsOnly, _ := cmd.Flags().GetBool("docs")
	searchWork, searchDocs := resolveSearchRoots(workOnly, docsOnly)

	statusRaw, _ := cmd.Flags().GetStringSlice("status")
	statuses := expandCommaSeparated(statusRaw)
	if len(statuses) > 0 && !searchWork {
		statuses = nil
	}

	docTypeRaw, _ := cmd.Flags().GetStringSlice("doc-type")
	docTypes := expandCommaSeparated(docTypeRaw)
	noDocType, _ := cmd.Flags().GetBool("no-doc-type")
	docTypeFilterOn := len(docTypes) > 0 || noDocType

	outputFormat, _ := cmd.Flags().GetString("output")
	if outputFormat != questionsOutputText && outputFormat != sliceLintOutputJSON {
		return questionsRunOpts{}, fmt.Errorf("invalid output format %q: use text or json", outputFormat)
	}

	return questionsRunOpts{
		searchWork:      searchWork,
		searchDocs:      searchDocs,
		statuses:        statuses,
		docTypes:        docTypes,
		noDocType:       noDocType,
		docTypeFilterOn: docTypeFilterOn,
		outputFormat:    outputFormat,
	}, nil
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

	opts, err := parseQuestionsRunFlags(cmd)
	if err != nil {
		return err
	}

	if len(opts.statuses) > 0 && opts.searchWork {
		if err := validateStatusValues(cfg, opts.statuses); err != nil {
			return err
		}
	}

	files, err := collectQuestionFiles(cfg, workAbs, docsAbs, opts)
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

	return writeQuestionsOutput(cmd.OutOrStdout(), opts.outputFormat, records)
}

func buildQuestionRecords(cfg *config.Config, files []string) ([]QuestionRecord, error) {
	records := make([]QuestionRecord, 0)
	for _, p := range files {
		rel, err := filepath.Rel(cfg.ConfigDir, p)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("invalid path for display: %s", p)
		}
		// #nosec G304 - path has been validated by collectQuestionFiles
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

func validateStatusValues(cfg *config.Config, values []string) error {
	for _, v := range values {
		if _, ok := cfg.StatusFolders[v]; !ok {
			return fmt.Errorf("invalid status %q: not defined in status_folders", v)
		}
		if !containsString(cfg.Validation.StatusValues, v) {
			return fmt.Errorf("invalid status %q: not in validation.status_values", v)
		}
	}
	return nil
}

func containsString(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func expandCommaSeparated(ss []string) []string {
	var out []string
	for _, s := range ss {
		for _, part := range strings.Split(s, ",") {
			p := strings.TrimSpace(part)
			if p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

func collectQuestionFiles(cfg *config.Config, workAbs, docsAbs string, opts questionsRunOpts) ([]string, error) {
	var paths []string
	if opts.searchWork {
		if err := walkWorkMarkdown(cfg, workAbs, &paths, opts); err != nil {
			return nil, err
		}
	}
	if opts.searchDocs {
		p, err := walkDocsMarkdown(docsAbs, opts)
		if err != nil {
			return nil, err
		}
		paths = append(paths, p...)
	}
	return paths, nil
}

func walkWorkMarkdown(cfg *config.Config, workAbs string, paths *[]string, opts questionsRunOpts) error {
	return filepath.WalkDir(workAbs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isMarkdownFile(path) {
			return nil
		}
		if err := validatePathUnderRoot(workAbs, path); err != nil {
			return err
		}
		if len(opts.statuses) > 0 && !fileMatchesWorkStatus(path, workAbs, opts.statuses, cfg) {
			return nil
		}
		if !includeByDocType(filepath.Base(path), opts.docTypes, opts.noDocType, opts.docTypeFilterOn) {
			return nil
		}
		*paths = append(*paths, path)
		return nil
	})
}

func walkDocsMarkdown(docsAbs string, opts questionsRunOpts) ([]string, error) {
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
		if !includeByDocType(filepath.Base(path), opts.docTypes, opts.noDocType, opts.docTypeFilterOn) {
			return nil
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

func fileMatchesWorkStatus(pathAbs, workAbs string, statuses []string, cfg *config.Config) bool {
	rel, err := filepath.Rel(workAbs, pathAbs)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return false
	}
	first := strings.Split(filepath.ToSlash(rel), "/")[0]
	for _, st := range statuses {
		folder := cfg.StatusFolders[st]
		if folder == first {
			return true
		}
	}
	return false
}

// deriveDocType returns true if the basename is typed (*.<type>.md), and the type segment.
func deriveDocType(basename string) (typed bool, typ string) {
	ext := strings.ToLower(filepath.Ext(basename))
	if ext != ".md" && ext != ".qmd" {
		return false, ""
	}
	stem := strings.TrimSuffix(basename, ext)
	if !strings.Contains(stem, ".") {
		return false, ""
	}
	i := strings.LastIndex(stem, ".")
	return true, stem[i+1:]
}

func includeByDocType(basename string, docTypes []string, noDocType, filterOn bool) bool {
	if !filterOn {
		return true
	}
	typed, t := deriveDocType(basename)
	var matchTyped bool
	for _, dt := range docTypes {
		if strings.EqualFold(t, dt) {
			matchTyped = true
			break
		}
	}
	if len(docTypes) == 0 && noDocType {
		return !typed
	}
	if len(docTypes) > 0 && !noDocType {
		return typed && matchTyped
	}
	if len(docTypes) > 0 && noDocType {
		return !typed || matchTyped
	}
	return false
}
