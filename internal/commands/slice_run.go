// Package commands implements slice command run logic.
package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"kira/internal/config"
)

const (
	sliceLintOutputJSON  = "json"
	taskStateOpen        = "open"
	sliceSelectorCurrent = "current"
)

// PrintSliceSummaryIfPresent prints a one-line slice/task summary if the work item has a Slices section.
// Used by kira start when moving to doing.
func PrintSliceSummaryIfPresent(path string, cfg *config.Config) {
	_, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil || len(slices) == 0 {
		return
	}
	var total, open int
	for _, s := range slices {
		for _, t := range s.Tasks {
			total++
			if !t.Done {
				open++
			}
		}
	}
	fmt.Printf("%s %d slices, %d tasks (%d open)\n", labelStyle("Slices:"), len(slices), total, open)
}

func getSlicesConfig(cfg *config.Config) *config.SlicesConfig {
	if cfg.Slices == nil {
		return &config.SlicesConfig{TaskIDFormat: "T%03d", DefaultState: "open"}
	}
	return cfg.Slices
}

// loadSlicesFromFile reads the work item file and parses the Slices section.
// Returns content (for later write), slices (nil if no section), and error.
func loadSlicesFromFile(path string, cfg *config.Config) (content []byte, slices []Slice, err error) {
	content, err = safeReadFile(path, cfg)
	if err != nil {
		return nil, nil, err
	}
	slices, err = ParseSlicesSection(content)
	if err != nil {
		return nil, nil, err
	}
	if slices == nil {
		slices = []Slice{}
	}
	return content, slices, nil
}

// writeSlicesToFile replaces the Slices section in content with generated section and writes the file.
func writeSlicesToFile(path string, content []byte, slices []Slice, cfg *config.Config) error {
	taskIDFormat := getSlicesConfig(cfg).TaskIDFormat
	newSection := GenerateSlicesSection(slices, taskIDFormat)
	newContent, err := ReplaceSlicesSection(content, newSection)
	if err != nil {
		return fmt.Errorf("failed to replace Slices section: %w", err)
	}
	return safeWriteFile(path, newContent, cfg)
}

// sliceCommitWorkItem stages the work item path and commits with the given message.
func sliceCommitWorkItem(path, message string, _ *config.Config) error {
	if err := validateStagedChanges([]string{path}); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()
	if _, err := executeCommand(ctx, "git", []string{"add", path}, "", false); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}
	commitCtx, commitCancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer commitCancel()
	if _, err := executeCommandCombinedOutput(commitCtx, "git", []string{"commit", "-m", message}, "", false); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return nil
}

func runSliceAdd(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	workItemID := args[0]
	sliceName := args[1]
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice add")
	if err != nil {
		return err
	}
	id := workItemIDFromPath(path, cfg)
	if id == "" {
		id = workItemID
	}
	content, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	// Check duplicate slice name
	for _, s := range slices {
		if strings.EqualFold(s.Name, sliceName) {
			return fmt.Errorf("slice named %q already exists", sliceName)
		}
	}
	slices = append(slices, Slice{Name: sliceName, Tasks: []Task{}})
	if err := writeSlicesToFile(path, content, slices, cfg); err != nil {
		return err
	}
	printSliceSummaryIf(cmd, path, cfg, "")
	fmt.Printf("Added slice %q to work item %s\n", sliceName, id)
	doCommit, _ := cmd.Flags().GetBool("commit")
	if doCommit {
		msg := fmt.Sprintf("Add slice %s to %s", sliceName, id)
		if err := sliceCommitWorkItem(path, msg, cfg); err != nil {
			return err
		}
		fmt.Println("Changes committed.")
	}
	return nil
}

func runSliceRemove(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	workItemID := args[0]
	sliceSelector := args[1]
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice remove")
	if err != nil {
		return err
	}
	id := workItemIDFromPath(path, cfg)
	if id == "" {
		id = workItemID
	}
	content, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	s, sliceIdx, err := resolveSliceSelector(slices, sliceSelector)
	if err != nil {
		return err
	}
	sliceName := s.Name
	var newSlices []Slice
	for i := range slices {
		if i+1 != sliceIdx {
			newSlices = append(newSlices, slices[i])
		}
	}
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		fmt.Printf("Remove slice %q and all its tasks? [y/N] ", sliceName)
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(line)) != "y" {
			return fmt.Errorf("aborted")
		}
	}
	if err := writeSlicesToFile(path, content, newSlices, cfg); err != nil {
		return err
	}
	printSliceSummaryIf(cmd, path, cfg, "")
	fmt.Printf("Removed slice %q from work item %s\n", sliceName, id)
	doCommit, _ := cmd.Flags().GetBool("commit")
	if doCommit {
		msg := fmt.Sprintf("Remove slice %s from %s", sliceName, id)
		if err := sliceCommitWorkItem(path, msg, cfg); err != nil {
			return err
		}
		fmt.Println("Changes committed.")
	}
	return nil
}

func runSliceTaskAdd(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	workItemID := args[0]
	sliceSelector := args[1]
	description := strings.Join(args[2:], " ")
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice task add")
	if err != nil {
		return err
	}
	id := workItemIDFromPath(path, cfg)
	if id == "" {
		id = workItemID
	}
	content, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	s, sliceIdx, err := resolveSliceSelector(slices, sliceSelector)
	if err != nil {
		return err
	}
	sliceIdx0 := sliceIdx - 1
	nextID, err := NextTaskID(slices, getSlicesConfig(cfg).TaskIDFormat)
	if err != nil {
		return err
	}
	defaultState := getSlicesConfig(cfg).DefaultState
	done := strings.EqualFold(defaultState, "done")
	slices[sliceIdx0].Tasks = append(slices[sliceIdx0].Tasks, Task{ID: nextID, Description: description, Done: done})
	if err := writeSlicesToFile(path, content, slices, cfg); err != nil {
		return err
	}
	printSliceSummaryIf(cmd, path, cfg, s.Name)
	fmt.Printf("Added task %s to slice %s in work item %s\n", taskIDStyle(nextID), sliceNameStyle(fmt.Sprintf("%d. %s", sliceIdx, s.Name)), id)
	doCommit, _ := cmd.Flags().GetBool("commit")
	if doCommit {
		msg := fmt.Sprintf("Add task %s to %s", nextID, id)
		if err := sliceCommitWorkItem(path, msg, cfg); err != nil {
			return err
		}
		fmt.Println("Changes committed.")
	}
	return nil
}

func findTaskByID(slices []Slice, taskID string) (sliceIdx, taskIdx int) {
	for i, s := range slices {
		for j, t := range s.Tasks {
			if t.ID == taskID {
				return i, j
			}
		}
	}
	return -1, -1
}

func runSliceTaskRemove(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	workItemID := args[0]
	taskID := args[1]
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice task remove")
	if err != nil {
		return err
	}
	id := workItemIDFromPath(path, cfg)
	if id == "" {
		id = workItemID
	}
	content, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	si, ti := findTaskByID(slices, taskID)
	if si < 0 {
		return fmt.Errorf("task %s not found", taskID)
	}
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		fmt.Printf("Remove task %s? [y/N] ", taskID)
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(line)) != "y" {
			return fmt.Errorf("aborted")
		}
	}
	tasks := slices[si].Tasks
	slices[si].Tasks = append(tasks[:ti], tasks[ti+1:]...)
	if err := writeSlicesToFile(path, content, slices, cfg); err != nil {
		return err
	}
	sliceIdx := si + 1
	sliceLabel := sliceNameStyle(fmt.Sprintf("%d. %s", sliceIdx, slices[si].Name))
	printSliceSummaryIf(cmd, path, cfg, "")
	fmt.Printf("Removed task %s from slice %s in work item %s\n", taskIDStyle(taskID), sliceLabel, id)
	doCommit, _ := cmd.Flags().GetBool("commit")
	if doCommit {
		msg := fmt.Sprintf("Remove task %s from %s", taskID, id)
		if err := sliceCommitWorkItem(path, msg, cfg); err != nil {
			return err
		}
		fmt.Println("Changes committed.")
	}
	return nil
}

func runSliceTaskEdit(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	workItemID := args[0]
	taskID := args[1]
	newDesc := strings.Join(args[2:], " ")
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice task edit")
	if err != nil {
		return err
	}
	id := workItemIDFromPath(path, cfg)
	if id == "" {
		id = workItemID
	}
	content, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	si, ti := findTaskByID(slices, taskID)
	if si < 0 {
		return fmt.Errorf("task %s not found", taskID)
	}
	slices[si].Tasks[ti].Description = newDesc
	if err := writeSlicesToFile(path, content, slices, cfg); err != nil {
		return err
	}
	sliceIdx := si + 1
	sliceLabel := sliceNameStyle(fmt.Sprintf("%d. %s", sliceIdx, slices[si].Name))
	printSliceSummaryIf(cmd, path, cfg, "")
	fmt.Printf("Updated task %s in slice %s (work item %s)\n", taskIDStyle(taskID), sliceLabel, taskIDStyle(id))
	doCommit, _ := cmd.Flags().GetBool("commit")
	if doCommit {
		msg := fmt.Sprintf("Edit task %s in %s", taskID, id)
		if err := sliceCommitWorkItem(path, msg, cfg); err != nil {
			return err
		}
		fmt.Println("Changes committed.")
	}
	return nil
}

func runSliceTaskNote(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	workItemID := args[0]
	taskID := args[1]
	note := strings.Join(args[2:], " ")
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice task note")
	if err != nil {
		return err
	}
	id := workItemIDFromPath(path, cfg)
	if id == "" {
		id = workItemID
	}
	content, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	si, ti := findTaskByID(slices, taskID)
	if si < 0 {
		return fmt.Errorf("task %s not found", taskID)
	}
	slices[si].Tasks[ti].Notes = note
	if err := writeSlicesToFile(path, content, slices, cfg); err != nil {
		return err
	}
	sliceIdx := si + 1
	sliceLabel := sliceNameStyle(fmt.Sprintf("%d. %s", sliceIdx, slices[si].Name))
	printSliceSummaryIf(cmd, path, cfg, "")
	fmt.Printf("Updated notes for task %s in slice %s (work item %s)\n", taskIDStyle(taskID), sliceLabel, id)
	doCommit, _ := cmd.Flags().GetBool("commit")
	if doCommit {
		msg := fmt.Sprintf("Note task %s in %s", taskID, id)
		if err := sliceCommitWorkItem(path, msg, cfg); err != nil {
			return err
		}
		fmt.Println("Changes committed.")
	}
	return nil
}

// printSliceTasks prints task lines for a slice. When spaceBetween is true, adds a blank line between each task.
// When showNotes is true, prints "    Notes: ..." under a task when present.
func printSliceTasks(tasks []Task, between, notes bool,
) {
	for i, t := range tasks {
		fmt.Printf("  %s %s: %s\n", taskBoxStyle(t.Done), taskIDStyle(t.ID), taskDescriptionStyle(t.Description, t.Done))
		if notes && t.Notes != "" {
			fmt.Printf("    Notes: %s\n", t.Notes)
		}
		if between && i < len(tasks)-1 {
			fmt.Println()
		}
	}
}

func runSliceShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	if len(args) == 0 {
		args = []string{"current"}
	}
	workItemID := args[0]
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice show")
	if err != nil {
		return err
	}
	_, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	id := workItemIDFromPath(path, cfg)
	if id == "" {
		id = workItemID
	}
	outputFormat, _ := cmd.Flags().GetString("output")
	if len(slices) == 0 {
		if outputFormat == sliceLintOutputJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(&SliceShowJSON{WorkItemID: id, Slices: []SliceShowSliceJSON{}})
		}
		fmt.Println("No slices in this work item.")
		return nil
	}
	if outputFormat == sliceLintOutputJSON {
		out, err := buildSliceShowJSON(id, slices, args)
		if err != nil {
			return err
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	return runSliceShowText(cmd, path, cfg, workItemID, slices, args)
}

// runSliceShowText performs the text-mode output for slice show.
func runSliceShowText(cmd *cobra.Command, path string, cfg *config.Config, workItemID string, slices []Slice, args []string) error {
	if len(args) == 1 {
		if workItemID == sliceSelectorCurrent {
			s, idx, err := resolveSliceSelector(slices, sliceSelectorCurrent)
			if err != nil {
				return err
			}
			printSliceProgressWorkItemLine(path, cfg, workItemID)
			printSliceSummaryBlock(cmd, slices, s.Name, sliceProgressPct(slices, s.Name))
			printSliceDetail(*s, idx)
			return nil
		}
		printSliceProgressWorkItemLine(path, cfg, workItemID)
		printSliceSummaryBlock(cmd, slices, "", sliceProgressPct(slices, ""))
		printAllSlices(slices)
		return nil
	}
	arg := args[1]
	currentName := sliceShowCurrentNameForSummary(workItemID, slices, arg)
	printSliceProgressWorkItemLine(path, cfg, workItemID)
	printSliceSummaryBlock(cmd, slices, currentName, sliceProgressPct(slices, currentName))
	if strings.EqualFold(arg, "all") {
		printAllSlices(slices)
		return nil
	}
	if s, idx, err := resolveSliceSelector(slices, arg); err == nil {
		printSliceDetail(*s, idx)
		return nil
	}
	if si, ti := findTaskByID(slices, arg); si >= 0 {
		printTaskDetail(slices[si].Tasks[ti], slices[si].Name)
		return nil
	}
	return fmt.Errorf("slice or task %q not found", arg)
}

func sliceShowCurrentNameForSummary(_ string, slices []Slice, secondArg string) string {
	if len(slices) == 0 {
		return ""
	}
	if strings.EqualFold(secondArg, "all") {
		return ""
	}
	if s, _, err := resolveSliceSelector(slices, secondArg); err == nil {
		return s.Name
	}
	if si, _ := findTaskByID(slices, secondArg); si >= 0 {
		return slices[si].Name
	}
	return ""
}

// buildSliceShowJSON builds the JSON for slice show --output json.
func buildSliceShowJSON(workItemID string, slices []Slice, args []string) (*SliceShowJSON, error) {
	out := &SliceShowJSON{WorkItemID: workItemID}
	if len(args) == 1 {
		if args[0] == sliceSelectorCurrent {
			s, idx, err := resolveSliceSelector(slices, sliceSelectorCurrent)
			if err != nil {
				return nil, err
			}
			out.Slices = []SliceShowSliceJSON{sliceToShowJSON(s)}
			out.SliceNumber = idx
			return out, nil
		}
		out.Slices = slicesToShowJSON(slices)
		return out, nil
	}
	arg := args[1]
	if strings.EqualFold(arg, "all") {
		out.Slices = slicesToShowJSON(slices)
		return out, nil
	}
	if s, idx, err := resolveSliceSelector(slices, arg); err == nil {
		out.Slices = []SliceShowSliceJSON{sliceToShowJSON(s)}
		out.SliceNumber = idx
		return out, nil
	}
	si, ti := findTaskByID(slices, arg)
	if si >= 0 {
		s := &slices[si]
		t := &s.Tasks[ti]
		out.Slices = []SliceShowSliceJSON{sliceToShowJSON(s)}
		out.SliceNumber = sliceIndex1Based(slices, s)
		out.Task = &SliceShowTaskDetailJSON{
			Slice:       s.Name,
			SliceNumber: sliceIndex1Based(slices, s),
			ID:          t.ID,
			Description: t.Description,
			Done:        t.Done,
			Notes:       t.Notes,
		}
		return out, nil
	}
	return nil, fmt.Errorf("slice or task %q not found", arg)
}

func sliceToShowJSON(s *Slice) SliceShowSliceJSON {
	tasks := make([]SliceShowTaskJSON, 0, len(s.Tasks))
	for i := range s.Tasks {
		tasks = append(tasks, SliceShowTaskJSON{
			ID:          s.Tasks[i].ID,
			Description: s.Tasks[i].Description,
			Done:        s.Tasks[i].Done,
			Notes:       s.Tasks[i].Notes,
		})
	}
	return SliceShowSliceJSON{
		Name:          s.Name,
		Description:   s.Description,
		CommitSummary: s.CommitSummary,
		Tasks:         tasks,
	}
}

func slicesToShowJSON(slices []Slice) []SliceShowSliceJSON {
	out := make([]SliceShowSliceJSON, 0, len(slices))
	for i := range slices {
		out = append(out, sliceToShowJSON(&slices[i]))
	}
	return out
}

func findSliceByName(slices []Slice, name string) *Slice {
	for i := range slices {
		if strings.EqualFold(slices[i].Name, name) {
			return &slices[i]
		}
	}
	return nil
}

// resolveSliceSelector resolves a slice by selector string. Returns the slice, its 1-based index, and an error.
// Selector can be: "current" (first with open tasks), "previous" (slice before current), a positive integer
// string ("1", "2", ...) for 1-based index, or a slice name.
func resolveSliceSelector(slices []Slice, selector string) (*Slice, int, error) {
	if len(slices) == 0 {
		return nil, 0, fmt.Errorf("no slices in work item")
	}
	sel := strings.TrimSpace(selector)
	if n, err := strconv.Atoi(sel); err == nil && n >= 1 {
		return resolveSliceByIndex(slices, n)
	}
	switch strings.ToLower(sel) {
	case sliceSelectorCurrent:
		return resolveSliceCurrent(slices)
	case "previous":
		return resolveSlicePrevious(slices)
	default:
		return resolveSliceByNameWithIndex(slices, selector)
	}
}

func resolveSliceByIndex(slices []Slice, n int) (*Slice, int, error) {
	if n > len(slices) {
		return nil, 0, fmt.Errorf("slice index %d out of range (have %d slice(s))", n, len(slices))
	}
	return &slices[n-1], n, nil
}

func resolveSliceCurrent(slices []Slice) (*Slice, int, error) {
	s := firstSliceWithOpenTasks(slices)
	if s == nil {
		idx := len(slices)
		return &slices[idx-1], idx, nil
	}
	for i := range slices {
		if &slices[i] == s {
			return s, i + 1, nil
		}
	}
	return s, 1, nil
}

func resolveSlicePrevious(slices []Slice) (*Slice, int, error) {
	curIdx := -1
	for i := range slices {
		for _, t := range slices[i].Tasks {
			if !t.Done {
				curIdx = i
				break
			}
		}
		if curIdx >= 0 {
			break
		}
	}
	if curIdx < 0 {
		idx := len(slices)
		return &slices[idx-1], idx, nil
	}
	if curIdx == 0 {
		return nil, 0, fmt.Errorf("no previous slice")
	}
	return &slices[curIdx-1], curIdx, nil
}

func resolveSliceByNameWithIndex(slices []Slice, name string) (*Slice, int, error) {
	s := findSliceByName(slices, name)
	if s == nil {
		return nil, 0, fmt.Errorf("slice %q not found", name)
	}
	for i := range slices {
		if &slices[i] == s {
			return s, i + 1, nil
		}
	}
	return s, 1, nil
}

func printAllSlices(slices []Slice) {
	fmt.Println(labelStyle("Slice:"))
	for i, s := range slices {
		fmt.Println(sliceNameStyle(fmt.Sprintf("%d. %s", i+1, s.Name)))
		printSliceTasks(s.Tasks, true, false)
		if i < len(slices)-1 {
			fmt.Println()
		}
	}
}

func printSliceDetail(s Slice, sliceIndex int) {
	heading := s.Name
	if sliceIndex >= 1 {
		heading = fmt.Sprintf("%d. %s", sliceIndex, s.Name)
	}
	fmt.Println(labelStyle("Slice:"))
	fmt.Println(sliceNameStyle(heading))
	printSliceTasks(s.Tasks, true, true)
}

func printTaskDetail(t Task, sliceName string) {
	stateStr := taskStateOpen
	if t.Done {
		stateStr = successStyle(defaultReleaseStatus)
	}
	fmt.Printf("%s %s\n", labelStyle("Task:"), taskIDStyle(t.ID))
	fmt.Printf("%s %s\n", labelStyle("Slice:"), sliceNameStyle(sliceName))
	fmt.Printf("%s %s\n", labelStyle("Description:"), t.Description)
	fmt.Printf("%s %s\n", labelStyle("State:"), stateStr)
	if t.Notes != "" {
		fmt.Printf("%s %s\n", labelStyle("Notes:"), t.Notes)
	}
}

// printSliceTaskShowCurrentOutput prints work item line, progress block, slice, current task, and notes for slice task show.
func printSliceTaskShowCurrentOutput(cmd *cobra.Command, path string, cfg *config.Config, workItemID string, slices []Slice, s *Slice, t *Task) {
	printSliceProgressWorkItemLine(path, cfg, workItemID)
	printSliceSummaryBlock(cmd, slices, s.Name, sliceProgressPct(slices, s.Name))
	sliceIdx := sliceIndex1Based(slices, s)
	fmt.Printf("%s %s\n", labelStyle("Slice:"), sliceNameStyle(fmt.Sprintf("%d. %s", sliceIdx, s.Name)))
	fmt.Printf("%s %s - %s\n", labelStyle("Current task:"), taskIDStyle(t.ID), t.Description)
	if t.Notes != "" {
		fmt.Printf("  %s %s\n", labelStyle("Notes:"), t.Notes)
	}
}

// workItemTypeLabel returns a display label for the work item kind (e.g. "issue" -> "Issue", "prd" -> "Prd"). Uses "Work item" when kind is empty or unknown.
func workItemTypeLabel(kind string) string {
	if kind == "" || kind == unknownValue {
		return "Work item"
	}
	return strings.ToUpper(kind[0:1]) + strings.ToLower(kind[1:])
}

// printSliceProgressWorkItemLine prints a leading blank line, then "<Type> <id>: <title>" (e.g. "Issue 047: No-commit on toggle") using metadata from path, then a blank line. Fallback id from workItemID if path has no id.
func printSliceProgressWorkItemLine(path string, cfg *config.Config, workItemID string) {
	fmt.Println()
	kind, metaID, title, _, _, _ := extractWorkItemMetadata(path, cfg)
	displayID := workItemIDFromPath(path, cfg)
	if displayID == "" {
		displayID = workItemID
	}
	if metaID != "" && metaID != unknownValue {
		displayID = metaID
	}
	label := workItemTypeLabel(kind)
	line := label + " " + taskIDStyle(displayID)
	if title != "" && title != unknownValue {
		line += ": " + title
	}
	fmt.Println(line)
	fmt.Println()
}

// printSliceProgressBreakdown prints the "Slices" label and per-slice "tasks X/Y" lines with green/orange styling. Percentage is shown in the summary line above, not here.
func printSliceProgressBreakdown(slices []Slice) {
	fmt.Println(labelStyle("Slices"))
	for i, s := range slices {
		sd := 0
		for _, t := range s.Tasks {
			if t.Done {
				sd++
			}
		}
		totalInSlice := len(s.Tasks)
		line := fmt.Sprintf("tasks %d/%d", sd, totalInSlice)
		if totalInSlice > 0 && sd == totalInSlice {
			line = summaryCompleteStyle(line)
		} else {
			line = summaryIncompleteStyle(line)
		}
		fmt.Printf("  %s: %s\n", sliceNameStyle(fmt.Sprintf("%d. %s", i+1, s.Name)), line)
	}
}

func runSliceProgress(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	hideSummary, _ := cmd.Flags().GetBool("hide-summary")
	if hideSummary {
		return nil
	}
	workItemID := sliceSelectorCurrent
	if len(args) > 0 {
		workItemID = args[0]
	}
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice progress")
	if err != nil {
		return err
	}
	_, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	printSliceProgressWorkItemLine(path, cfg, workItemID)
	printSliceSummaryBlock(cmd, slices, "", sliceProgressPct(slices, ""))
	printSliceProgressBreakdown(slices)
	return nil
}

// firstSliceWithOpenTasks returns the first slice (in order) that has at least one open task.
func firstSliceWithOpenTasks(slices []Slice) *Slice {
	for _, s := range slices {
		for _, t := range s.Tasks {
			if !t.Done {
				sCopy := s
				return &sCopy
			}
		}
	}
	return nil
}

// firstOpenTaskInSlice returns the first open task in the slice.
func firstOpenTaskInSlice(s *Slice) *Task {
	for i := range s.Tasks {
		if !s.Tasks[i].Done {
			return &s.Tasks[i]
		}
	}
	return nil
}

// sliceSummaryNumbers returns completedSlices, totalSlices, doneTasks, totalTasks, and the in-current-slice suffix (" · X/Y in current slice" or ""). Single source of truth for summary counts used by formatSliceSummary, formatSliceSummaryParts, and runSliceProgress.
func sliceSummaryNumbers(slices []Slice, currentSliceName string) (completedSlices, totalSlices, doneTasks, totalTasks int, inCurrentStr string) {
	if len(slices) == 0 {
		return 0, 0, 0, 0, ""
	}
	for _, s := range slices {
		sDone, sTotal := 0, len(s.Tasks)
		for _, t := range s.Tasks {
			totalTasks++
			if t.Done {
				doneTasks++
				sDone++
			}
		}
		if sTotal > 0 && sDone == sTotal {
			completedSlices++
		}
	}
	totalSlices = len(slices)
	curName := currentSliceName
	if curName == "" {
		if next := firstSliceWithOpenTasks(slices); next != nil {
			curName = next.Name
		}
	}
	if curName != "" {
		for _, s := range slices {
			if !strings.EqualFold(s.Name, curName) {
				continue
			}
			done, total := 0, len(s.Tasks)
			for _, t := range s.Tasks {
				if t.Done {
					done++
				}
			}
			inCurrentStr = fmt.Sprintf(" · %d/%d in current slice", done, total)
			break
		}
	}
	return completedSlices, totalSlices, doneTasks, totalTasks, inCurrentStr
}

// formatSliceSummary returns a one-line progress summary: "completedSlices/totalSlices slices · doneTasks/totalTasks tasks · doneInCurrent/totalInCurrent in current slice".
// currentSliceName is the name of the slice that contains the "current" (next open) task; if empty, the first slice with open tasks is used.
func formatSliceSummary(slices []Slice, currentSliceName string) string {
	completedSlices, totalSlices, doneTasks, totalTasks, inCurrent := sliceSummaryNumbers(slices, currentSliceName)
	if totalSlices == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d slices · %d/%d tasks%s", completedSlices, totalSlices, doneTasks, totalTasks, inCurrent)
}

// formatSliceSummaryParts returns the main part ("4/7 slices · 11/17 tasks") and the in-current-slice part (" · 3/3 in current slice" or ""). Used so each part can be styled separately.
func formatSliceSummaryParts(slices []Slice, currentSliceName string) (main, inCurrent string) {
	completedSlices, totalSlices, doneTasks, totalTasks, inCurrentStr := sliceSummaryNumbers(slices, currentSliceName)
	if totalSlices == 0 {
		return "", ""
	}
	main = fmt.Sprintf("%d/%d slices · %d/%d tasks", completedSlices, totalSlices, doneTasks, totalTasks)
	return main, inCurrentStr
}

// currentSliceAllDone returns true if the slice identified by currentSliceName (or first with open tasks when empty) has all tasks done. When there is no current slice (all work complete), returns true.
func currentSliceAllDone(slices []Slice, currentSliceName string) bool {
	curName := currentSliceName
	if curName == "" {
		if next := firstSliceWithOpenTasks(slices); next != nil {
			curName = next.Name
		}
	}
	if curName == "" {
		return true
	}
	for _, s := range slices {
		if !strings.EqualFold(s.Name, curName) {
			continue
		}
		total := len(s.Tasks)
		if total == 0 {
			return true
		}
		done := 0
		for _, t := range s.Tasks {
			if t.Done {
				done++
			}
		}
		return done == total
	}
	return true
}

// sliceProgressPct returns task completion percentage (0-100) for use in the progress line, or -1 if total is 0. Used so all commands show the same "… tasks (N%)" format.
func sliceProgressPct(slices []Slice, currentSliceName string) int {
	_, _, done, total, _ := sliceSummaryNumbers(slices, currentSliceName)
	if total <= 0 {
		return -1
	}
	return 100 * done / total
}

// printSliceSummaryBlock prints a progress block: "Progress" and styled summary on same line, then blank line.
// If percentagePct >= 0, embeds task completion percentage in the main part as "… tasks (N%)" so it reads with the task count, not the current-slice suffix. Main part (and percentage when present) use green when all tasks done, orange otherwise; " · X/Y in current slice" is styled separately.
// Skips when cmd has --hide-summary or output is json.
func printSliceSummaryBlock(cmd *cobra.Command, slices []Slice, currentSliceName string, percentagePct int) {
	hide, _ := cmd.Flags().GetBool("hide-summary")
	if hide {
		return
	}
	if cmd != nil && cmd.Flags().Lookup("output") != nil {
		if out, _ := cmd.Flags().GetString("output"); out == sliceLintOutputJSON {
			return
		}
	}
	if len(slices) == 0 {
		return
	}
	mainPart, inCurrentPart := formatSliceSummaryParts(slices, currentSliceName)
	if mainPart == "" {
		return
	}
	if percentagePct >= 0 {
		mainPart = fmt.Sprintf("%s (%d%%)", mainPart, percentagePct)
	}
	styledMain := summaryIncompleteStyle(mainPart)
	var styledInCurrent string
	if inCurrentPart != "" {
		if currentSliceAllDone(slices, currentSliceName) {
			styledInCurrent = summaryCompleteStyle(inCurrentPart)
		} else {
			styledInCurrent = summaryIncompleteStyle(inCurrentPart)
		}
	}
	_, _, done, total, _ := sliceSummaryNumbers(slices, currentSliceName)
	if percentagePct >= 0 && total > 0 && done == total {
		styledMain = summaryCompleteStyle(mainPart)
	}
	fmt.Printf("%s %s%s\n", labelStyle("Progress"), styledMain, styledInCurrent)
	fmt.Println()
}

// printSliceSummaryIf loads slices from path and prints the progress block (heading, blank, styled summary, blank) unless --hide-summary or json.
func printSliceSummaryIf(cmd *cobra.Command, path string, cfg *config.Config, currentSliceName string) {
	hide, _ := cmd.Flags().GetBool("hide-summary")
	if hide {
		return
	}
	if cmd != nil {
		if cmd.Flags().Lookup("output") != nil {
			if out, _ := cmd.Flags().GetString("output"); out == sliceLintOutputJSON {
				return
			}
		}
	}
	_, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil || len(slices) == 0 {
		return
	}
	printSliceProgressWorkItemLine(path, cfg, "")
	printSliceSummaryBlock(cmd, slices, currentSliceName, sliceProgressPct(slices, currentSliceName))
}

// sliceIndex1Based returns the 1-based index of s in slices, or 0 if not found.
func sliceIndex1Based(slices []Slice, s *Slice) int {
	for i := range slices {
		if &slices[i] == s {
			return i + 1
		}
	}
	return 0
}

// TaskRefJSON is a task reference for JSON output.
type TaskRefJSON struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// TaskCurrentJSON is the JSON output for slice task show --output json.
type TaskCurrentJSON struct {
	WorkItemID  string `json:"work_item_id"`
	Slice       string `json:"slice,omitempty"`
	SliceNumber int    `json:"slice_number,omitempty"`
	TaskID      string `json:"task_id,omitempty"`
	Description string `json:"description,omitempty"`
	Notes       string `json:"notes,omitempty"`
	AllComplete bool   `json:"all_complete,omitempty"`
	Summary     string `json:"summary,omitempty"`
}

// SliceShowTaskJSON is a task in slice show JSON output.
type SliceShowTaskJSON struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
	Notes       string `json:"notes,omitempty"`
}

// SliceShowSliceJSON is a slice in slice show JSON output.
type SliceShowSliceJSON struct {
	Name          string              `json:"name"`
	Description   string              `json:"description,omitempty"`
	CommitSummary string              `json:"commit_summary,omitempty"`
	Tasks         []SliceShowTaskJSON `json:"tasks"`
}

// SliceShowJSON is the JSON output for slice show --output json.
type SliceShowJSON struct {
	WorkItemID  string                   `json:"work_item_id"`
	Slices      []SliceShowSliceJSON     `json:"slices"`
	SliceNumber int                      `json:"slice_number,omitempty"` // 1-based when showing single slice
	Task        *SliceShowTaskDetailJSON `json:"task,omitempty"`         // set when showing single task
}

// SliceShowTaskDetailJSON is the task detail when slice show targets a task-id.
type SliceShowTaskDetailJSON struct {
	Slice       string `json:"slice"`
	SliceNumber int    `json:"slice_number"`
	ID          string `json:"id"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
	Notes       string `json:"notes,omitempty"`
}

func runSliceTaskCurrent(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	workItemID := sliceSelectorCurrent
	sliceSelector := ""
	if len(args) > 0 {
		workItemID = args[0]
	}
	if len(args) > 1 {
		sliceSelector = args[1]
	}
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice task show")
	if err != nil {
		return err
	}
	id := workItemIDFromPath(path, cfg)
	if id == "" {
		id = workItemID
	}
	_, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	var s *Slice
	if sliceSelector != "" {
		var resolveErr error
		s, _, resolveErr = resolveSliceSelector(slices, sliceSelector)
		if resolveErr != nil {
			return resolveErr
		}
	} else {
		s = firstSliceWithOpenTasks(slices)
	}
	if s == nil {
		// No open tasks: show status (same format as slice show) and exit 0.
		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == sliceLintOutputJSON {
			out := TaskCurrentJSON{
				WorkItemID:  id,
				AllComplete: true,
				Summary:     formatSliceSummary(slices, ""),
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}
		printSliceProgressWorkItemLine(path, cfg, workItemID)
		printSliceSummaryBlock(cmd, slices, "", sliceProgressPct(slices, ""))
		fmt.Println(summaryCompleteStyle("All tasks complete."))
		return nil
	}
	t := firstOpenTaskInSlice(s)
	if t == nil {
		// Slice has no open tasks (shouldn't happen if s from firstSliceWithOpenTasks): show status and exit 0.
		outputFormat, _ := cmd.Flags().GetString("output")
		if outputFormat == sliceLintOutputJSON {
			out := TaskCurrentJSON{
				WorkItemID:  id,
				AllComplete: true,
				Summary:     formatSliceSummary(slices, ""),
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}
		printSliceProgressWorkItemLine(path, cfg, workItemID)
		printSliceSummaryBlock(cmd, slices, "", sliceProgressPct(slices, ""))
		fmt.Println(summaryCompleteStyle("All tasks complete."))
		return nil
	}
	outputFormat, _ := cmd.Flags().GetString("output")
	if outputFormat == sliceLintOutputJSON {
		out := TaskCurrentJSON{
			WorkItemID:  id,
			Slice:       s.Name,
			SliceNumber: sliceIndex1Based(slices, s),
			TaskID:      t.ID,
			Description: t.Description,
			Notes:       t.Notes,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	printSliceTaskShowCurrentOutput(cmd, path, cfg, workItemID, slices, s, t)
	return nil
}

// printDoneCurrentNext prints the status block, slice, optional message, completed task line (if any), then either "All tasks complete" or the next task block (Next / Same slice | New slice / New task).
// messageAfterStatus is printed after the status header when nextSlice is nil (e.g. "No task marked done (all tasks already complete).").
func printDoneCurrentNext(cmd *cobra.Command, path string, cfg *config.Config, workItemID string, slices []Slice, completedSliceName, completedTaskID, completedTaskDesc, messageAfterStatus string) {
	hideSummary, _ := cmd.Flags().GetBool("hide-summary")
	if !hideSummary {
		printSliceProgressWorkItemLine(path, cfg, workItemID)
		printSliceSummaryBlock(cmd, slices, "", sliceProgressPct(slices, ""))
	}
	nextSlice := firstSliceWithOpenTasks(slices)
	var sliceForHeader *Slice
	if nextSlice != nil {
		sliceForHeader = nextSlice
	} else if completedSliceName != "" {
		sliceForHeader = findSliceByName(slices, completedSliceName)
	}
	if sliceForHeader != nil {
		idx := sliceIndex1Based(slices, sliceForHeader)
		fmt.Printf("%s %s\n", labelStyle("Slice:"), sliceNameStyle(fmt.Sprintf("%d. %s", idx, sliceForHeader.Name)))
	}
	if nextSlice == nil {
		if messageAfterStatus != "" {
			fmt.Println(messageAfterStatus)
			return
		}
		fmt.Println(summaryCompleteStyle("All tasks complete."))
		return
	}
	if completedTaskID != "" {
		fmt.Printf("%s %s - %s\n", labelStyle("Completed task:"), taskIDStyle(completedTaskID), completedTaskDesc)
	}
	nextTask := firstOpenTaskInSlice(nextSlice)
	if nextTask == nil {
		fmt.Println(summaryCompleteStyle("All tasks complete."))
		return
	}
	nextSliceIdx := sliceIndex1Based(slices, nextSlice)
	sliceLabel := sliceNameStyle(fmt.Sprintf("%d. %s", nextSliceIdx, nextSlice.Name))
	fmt.Println(labelStyle("\nNext"))
	if nextSlice.Name == completedSliceName {
		fmt.Printf("%s %s\n", labelStyle("Same slice:"), sliceLabel)
	} else {
		fmt.Printf("%s %s\n", labelStyle("New slice:"), sliceLabel)
	}
	fmt.Printf("%s %s - %s\n", labelStyle("New task:"), taskIDStyle(nextTask.ID), nextTask.Description)
}

func runSliceTaskDoneCurrent(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	workItemID := sliceSelectorCurrent
	if len(args) > 0 {
		workItemID = args[0]
	}
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice task done current")
	if err != nil {
		return err
	}
	content, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	s := firstSliceWithOpenTasks(slices)
	if s == nil {
		// All tasks already complete: show same status as after marking last task done (idempotent, exit 0).
		printDoneCurrentNext(cmd, path, cfg, workItemID, slices, "", "", "", "No task marked done (all tasks already complete).")
		return nil
	}
	t := firstOpenTaskInSlice(s)
	if t == nil {
		// Slice has no open tasks: same idempotent success with status.
		printDoneCurrentNext(cmd, path, cfg, workItemID, slices, "", "", "", "No task marked done (all tasks already complete).")
		return nil
	}
	si, ti := findTaskByID(slices, t.ID)
	if si < 0 {
		return fmt.Errorf("task %s not found", t.ID)
	}
	completedSliceName := s.Name
	completedTaskID := t.ID
	completedDesc := t.Description
	slices[si].Tasks[ti].Done = true
	if err := writeSlicesToFile(path, content, slices, cfg); err != nil {
		return err
	}
	doCommit, _ := cmd.Flags().GetBool("commit")
	if doCommit {
		msg := fmt.Sprintf("Toggle task %s to %s", completedTaskID, defaultReleaseStatus)
		if err := sliceCommitWorkItem(path, msg, cfg); err != nil {
			return err
		}
		fmt.Println("Changes committed.")
	}
	printDoneCurrentNext(cmd, path, cfg, workItemID, slices, completedSliceName, completedTaskID, completedDesc, "")
	return nil
}

func workItemIDFromPath(path string, cfg *config.Config) string {
	_, id, _, _, _, err := extractWorkItemMetadata(path, cfg)
	if err != nil || id == unknownValue {
		return ""
	}
	return id
}

// SliceLintError represents a single lint error with location, rule, message, and optional suggestion.
type SliceLintError struct {
	Location   string `json:"location"`
	Rule       string `json:"rule"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

func runSliceLint(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	outputFormat, _ := cmd.Flags().GetString("output")
	if len(args) > 0 {
		return runSliceLintOne(cmd, cfg, args[0], outputFormat)
	}
	return runSliceLintAll(cmd, cfg, outputFormat)
}

func runSliceLintOne(cmd *cobra.Command, cfg *config.Config, workItemID, outputFormat string) error {
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice lint")
	if err != nil {
		return err
	}
	errors := lintSlicesSection(path, cfg)
	if outputFormat == sliceLintOutputJSON {
		return outputSliceLintJSON(errors)
	}
	if len(errors) > 0 {
		if err := outputSliceLintHuman(path, errors); err != nil {
			return err
		}
		return nil
	}
	printSliceSummaryIf(cmd, path, cfg, "")
	fmt.Println(successStyle("Slices section is valid."))
	return nil
}

func runSliceLintAll(cmd *cobra.Command, cfg *config.Config, outputFormat string) error {
	paths, err := getDoingWorkItemPaths(cfg)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("slice lint: no work item in doing folder; specify work-item-id or start a work item (e.g. kira slice lint <work-item-id>)")
		}
		return fmt.Errorf("failed to read doing folder: %w", err)
	}
	if len(paths) == 0 {
		return fmt.Errorf("slice lint: no work item in doing folder; specify work-item-id or start a work item (e.g. kira slice lint <work-item-id>)")
	}
	var allErrs []SliceLintError
	for _, path := range paths {
		allErrs = append(allErrs, lintSlicesSection(path, cfg)...)
	}
	if outputFormat == sliceLintOutputJSON {
		return outputSliceLintJSON(allErrs)
	}
	anyFailed := false
	for i, path := range paths {
		if len(paths) > 1 && i > 0 {
			fmt.Println()
		}
		if len(paths) > 1 {
			fmt.Println(filepath.Base(path))
		}
		errs := lintSlicesSection(path, cfg)
		if len(errs) > 0 {
			if err := outputSliceLintHuman(path, errs); err != nil {
				anyFailed = true
			}
		} else {
			printSliceSummaryIf(cmd, path, cfg, "")
			fmt.Println(successStyle("Slices section is valid."))
		}
	}
	if anyFailed {
		return fmt.Errorf("slice lint found error(s)")
	}
	return nil
}

func lintSlicesSection(path string, cfg *config.Config) []SliceLintError {
	content, err := safeReadFile(path, cfg)
	if err != nil {
		return []SliceLintError{{Location: path, Rule: "read", Message: err.Error()}}
	}
	slices, err := ParseSlicesSection(content)
	if err != nil {
		return []SliceLintError{{Location: path, Rule: "parse", Message: err.Error()}}
	}
	if slices == nil {
		return []SliceLintError{{
			Location:   path,
			Rule:       "missing-section",
			Message:    "Slices section missing",
			Suggestion: "Add a ## Slices section with ### slice names and task list items",
		}}
	}
	var errs []SliceLintError
	seenTaskIDs := make(map[string]string) // id -> slice name
	seenSliceNames := make(map[string]bool)
	for _, s := range slices {
		if seenSliceNames[s.Name] {
			errs = append(errs, SliceLintError{
				Location:   path + " (slice: " + s.Name + ")",
				Rule:       "duplicate-slice-name",
				Message:    "Duplicate slice name: " + s.Name,
				Suggestion: "Use unique slice names",
			})
		}
		seenSliceNames[s.Name] = true
		for _, t := range s.Tasks {
			if prev, ok := seenTaskIDs[t.ID]; ok {
				errs = append(errs, SliceLintError{
					Location:   path + " (task: " + t.ID + ")",
					Rule:       "duplicate-task-id",
					Message:    "Task ID " + t.ID + " appears more than once",
					Suggestion: "Use unique task IDs (e.g. T001, T002, ...). Previously seen in slice: " + prev,
				})
			}
			seenTaskIDs[t.ID] = s.Name
			// State is always open or done from parser (checkbox [ ] or [x]); no other state possible
			// So we don't need to validate "invalid state" if parser only produces open/done
		}
	}
	return errs
}

func outputSliceLintHuman(path string, errs []SliceLintError) error {
	for _, e := range errs {
		line := path
		if e.Location != path {
			line = e.Location
		}
		fmt.Printf("%s [%s] %s", line, errorStyle(e.Rule), e.Message)
		if e.Suggestion != "" {
			fmt.Printf(" Suggestion: %s", e.Suggestion)
		}
		fmt.Println()
	}
	if len(errs) > 0 {
		return fmt.Errorf("slice lint found %d error(s)", len(errs))
	}
	fmt.Println(successStyle("Slices section is valid."))
	return nil
}

func outputSliceLintJSON(errs []SliceLintError) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(errs); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("slice lint found %d error(s)", len(errs))
	}
	return nil
}
