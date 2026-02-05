// Package commands implements slice command run logic.
package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"kira/internal/config"
)

const (
	sliceLintOutputJSON = "json"
	taskStateOpen       = "open"
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
	path, err := findWorkItemFile(workItemID, cfg)
	if err != nil {
		return err
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
	fmt.Printf("Added slice %q to work item %s\n", sliceName, workItemID)
	noCommit, _ := cmd.Flags().GetBool("no-commit")
	if !noCommit {
		msg := fmt.Sprintf("Add slice %s to %s", sliceName, workItemID)
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
	sliceName := args[1]
	path, err := findWorkItemFile(workItemID, cfg)
	if err != nil {
		return err
	}
	content, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	var found bool
	var newSlices []Slice
	for _, s := range slices {
		if strings.EqualFold(s.Name, sliceName) {
			found = true
			continue
		}
		newSlices = append(newSlices, s)
	}
	if !found {
		return fmt.Errorf("slice named %q not found", sliceName)
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
	fmt.Printf("Removed slice %q from work item %s\n", sliceName, workItemID)
	noCommit, _ := cmd.Flags().GetBool("no-commit")
	if !noCommit {
		msg := fmt.Sprintf("Remove slice %s from %s", sliceName, workItemID)
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
	sliceName := args[1]
	description := strings.Join(args[2:], " ")
	path, err := findWorkItemFile(workItemID, cfg)
	if err != nil {
		return err
	}
	content, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	sliceIdx := -1
	for i, s := range slices {
		if strings.EqualFold(s.Name, sliceName) {
			sliceIdx = i
			break
		}
	}
	if sliceIdx < 0 {
		return fmt.Errorf("slice named %q not found", sliceName)
	}
	nextID, err := NextTaskID(slices, getSlicesConfig(cfg).TaskIDFormat)
	if err != nil {
		return err
	}
	defaultState := getSlicesConfig(cfg).DefaultState
	done := strings.EqualFold(defaultState, "done")
	slices[sliceIdx].Tasks = append(slices[sliceIdx].Tasks, Task{ID: nextID, Description: description, Done: done})
	if err := writeSlicesToFile(path, content, slices, cfg); err != nil {
		return err
	}
	fmt.Printf("Added task %s to slice %q in work item %s\n", nextID, sliceName, workItemID)
	noCommit, _ := cmd.Flags().GetBool("no-commit")
	if !noCommit {
		msg := fmt.Sprintf("Add task %s to %s", nextID, workItemID)
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
	path, err := findWorkItemFile(workItemID, cfg)
	if err != nil {
		return err
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
	fmt.Printf("Removed task %s from work item %s\n", taskID, workItemID)
	noCommit, _ := cmd.Flags().GetBool("no-commit")
	if !noCommit {
		msg := fmt.Sprintf("Remove task %s from %s", taskID, workItemID)
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
	path, err := findWorkItemFile(workItemID, cfg)
	if err != nil {
		return err
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
	fmt.Printf("Updated task %s in work item %s\n", taskIDStyle(taskID), taskIDStyle(workItemID))
	noCommit, _ := cmd.Flags().GetBool("no-commit")
	if !noCommit {
		msg := fmt.Sprintf("Edit task %s in %s", taskID, workItemID)
		if err := sliceCommitWorkItem(path, msg, cfg); err != nil {
			return err
		}
		fmt.Println("Changes committed.")
	}
	return nil
}

func runSliceTaskToggle(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	workItemID := args[0]
	taskID := args[1]
	path, err := findWorkItemFile(workItemID, cfg)
	if err != nil {
		return err
	}
	content, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	si, ti := findTaskByID(slices, taskID)
	if si < 0 {
		return fmt.Errorf("task %s not found", taskID)
	}
	slices[si].Tasks[ti].Done = !slices[si].Tasks[ti].Done
	if err := writeSlicesToFile(path, content, slices, cfg); err != nil {
		return err
	}
	state := taskStateOpen
	if slices[si].Tasks[ti].Done {
		state = defaultReleaseStatus
	}
	stateOut := state
	if state == defaultReleaseStatus {
		stateOut = successStyle(state)
	}
	fmt.Printf("Task %s set to %s in work item %s\n", taskIDStyle(taskID), stateOut, taskIDStyle(workItemID))
	noCommit, _ := cmd.Flags().GetBool("no-commit")
	if !noCommit {
		msg := fmt.Sprintf("Toggle task %s in %s", taskID, workItemID)
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
	path, err := findWorkItemFile(workItemID, cfg)
	if err != nil {
		return err
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
	fmt.Printf("Updated notes for task %s in work item %s\n", taskID, workItemID)
	noCommit, _ := cmd.Flags().GetBool("no-commit")
	if !noCommit {
		msg := fmt.Sprintf("Note task %s in %s", taskID, workItemID)
		if err := sliceCommitWorkItem(path, msg, cfg); err != nil {
			return err
		}
		fmt.Println("Changes committed.")
	}
	return nil
}

func runSliceShow(_ *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	workItemID := args[0]
	path, err := findWorkItemFile(workItemID, cfg)
	if err != nil {
		return err
	}
	_, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	if len(slices) == 0 {
		fmt.Println("No slices in this work item.")
		return nil
	}
	if len(args) == 1 {
		printAllSlices(slices)
		return nil
	}
	arg := args[1]
	if s := findSliceByName(slices, arg); s != nil {
		printSliceDetail(*s)
		return nil
	}
	if si, ti := findTaskByID(slices, arg); si >= 0 {
		printTaskDetail(slices[si].Tasks[ti], slices[si].Name)
		return nil
	}
	return fmt.Errorf("slice or task %q not found", arg)
}

func findSliceByName(slices []Slice, name string) *Slice {
	for i := range slices {
		if strings.EqualFold(slices[i].Name, name) {
			return &slices[i]
		}
	}
	return nil
}

func printAllSlices(slices []Slice) {
	for _, s := range slices {
		fmt.Println(sliceNameStyle(s.Name))
		for _, t := range s.Tasks {
			fmt.Printf("  %s %s: %s\n", taskBoxStyle(t.Done), taskIDStyle(t.ID), t.Description)
		}
		fmt.Println()
	}
}

func printSliceDetail(s Slice) {
	fmt.Println(sliceNameStyle(s.Name))
	for _, t := range s.Tasks {
		fmt.Printf("  %s %s: %s\n", taskBoxStyle(t.Done), taskIDStyle(t.ID), t.Description)
		if t.Notes != "" {
			fmt.Printf("    Notes: %s\n", t.Notes)
		}
	}
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

func runSliceProgress(_ *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	workItemID := args[0]
	path, err := findWorkItemFile(workItemID, cfg)
	if err != nil {
		return err
	}
	_, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	var total, done int
	for _, s := range slices {
		for _, t := range s.Tasks {
			total++
			if t.Done {
				done++
			}
		}
	}
	open := total - done
	fmt.Printf("Work item %s: %d tasks (%d done, %d open)\n", taskIDStyle(workItemID), total, done, open)
	if total > 0 {
		pct := 100 * done / total
		fmt.Printf("Progress: %d%%\n", pct)
	}
	for _, s := range slices {
		var sd, so int
		for _, t := range s.Tasks {
			if t.Done {
				sd++
			} else {
				so++
			}
		}
		fmt.Printf("  %s: %d done, %d open\n", sliceNameStyle(s.Name), sd, so)
	}
	return nil
}

// firstSliceWithOpenTasks returns the first slice (in order) that has at least one open task.
func firstSliceWithOpenTasks(slices []Slice) *Slice {
	for i := range slices {
		for _, t := range slices[i].Tasks {
			if !t.Done {
				return &slices[i]
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

// SliceCurrentJSON is the JSON output for slice current --output json.
type SliceCurrentJSON struct {
	WorkItemID    string        `json:"work_item_id"`
	Slice         string        `json:"slice"`
	OpenTaskCount int           `json:"open_task_count"`
	OpenTasks     []TaskRefJSON `json:"open_tasks"`
}

// TaskRefJSON is a task reference for JSON output.
type TaskRefJSON struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// TaskCurrentJSON is the JSON output for slice task current --output json.
type TaskCurrentJSON struct {
	WorkItemID  string `json:"work_item_id"`
	Slice       string `json:"slice"`
	TaskID      string `json:"task_id"`
	Description string `json:"description"`
	Notes       string `json:"notes"`
}

func runSliceCurrent(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	workItemID := ""
	if len(args) > 0 {
		workItemID = args[0]
	}
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice current")
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
	cur := firstSliceWithOpenTasks(slices)
	outputFormat, _ := cmd.Flags().GetString("output")
	if outputFormat == sliceLintOutputJSON {
		return outputSliceCurrentJSON(id, cur)
	}
	printSliceCurrentHuman(cur)
	return nil
}

func outputSliceCurrentJSON(workItemID string, cur *Slice) error {
	out := SliceCurrentJSON{WorkItemID: workItemID}
	if cur != nil {
		out.Slice = cur.Name
		for _, t := range cur.Tasks {
			if !t.Done {
				out.OpenTaskCount++
				out.OpenTasks = append(out.OpenTasks, TaskRefJSON{ID: t.ID, Description: t.Description})
			}
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func printSliceCurrentHuman(cur *Slice) {
	if cur == nil {
		fmt.Println(labelStyle("Current slice:") + " (none - all tasks done)")
		return
	}
	openCount := 0
	for _, t := range cur.Tasks {
		if !t.Done {
			openCount++
		}
	}
	fmt.Printf("%s %s (%d open tasks)\n", labelStyle("Current slice:"), sliceNameStyle(cur.Name), openCount)
	for _, t := range cur.Tasks {
		if !t.Done {
			fmt.Printf("  - %s: %s\n", taskIDStyle(t.ID), t.Description)
		}
	}
}

func runSliceTaskCurrent(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	workItemID := ""
	sliceName := ""
	if len(args) > 0 {
		workItemID = args[0]
	}
	if len(args) > 1 {
		if strings.EqualFold(args[1], "toggle") {
			return runSliceTaskCurrentToggle(cmd, cfg, workItemID)
		}
		sliceName = args[1]
	}
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice task current")
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
	if sliceName != "" {
		s = findSliceByName(slices, sliceName)
		if s == nil {
			return fmt.Errorf("slice %q not found", sliceName)
		}
	} else {
		s = firstSliceWithOpenTasks(slices)
	}
	if s == nil {
		return fmt.Errorf("no open tasks in work item")
	}
	t := firstOpenTaskInSlice(s)
	if t == nil {
		return fmt.Errorf("no open tasks in slice %q", s.Name)
	}
	outputFormat, _ := cmd.Flags().GetString("output")
	if outputFormat == sliceLintOutputJSON {
		out := TaskCurrentJSON{
			WorkItemID:  id,
			Slice:       s.Name,
			TaskID:      t.ID,
			Description: t.Description,
			Notes:       t.Notes,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	fmt.Printf("%s %s - %s\n", labelStyle("Current task:"), taskIDStyle(t.ID), t.Description)
	if t.Notes != "" {
		fmt.Printf("  %s %s\n", labelStyle("Notes:"), t.Notes)
	}
	return nil
}

func runSliceTaskCurrentToggle(cmd *cobra.Command, cfg *config.Config, workItemID string) error {
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice task current toggle")
	if err != nil {
		return err
	}
	content, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	s := firstSliceWithOpenTasks(slices)
	if s == nil {
		return fmt.Errorf("no open tasks in work item")
	}
	t := firstOpenTaskInSlice(s)
	if t == nil {
		return fmt.Errorf("no open tasks in slice %q", s.Name)
	}
	si, ti := findTaskByID(slices, t.ID)
	if si < 0 {
		return fmt.Errorf("task %s not found", t.ID)
	}
	slices[si].Tasks[ti].Done = !slices[si].Tasks[ti].Done
	if err := writeSlicesToFile(path, content, slices, cfg); err != nil {
		return err
	}
	state := taskStateOpen
	if slices[si].Tasks[ti].Done {
		state = defaultReleaseStatus
	}
	stateOut := state
	if state == defaultReleaseStatus {
		stateOut = successStyle(state)
	}
	fmt.Printf("Task %s set to %s\n", taskIDStyle(t.ID), stateOut)
	noCommit, _ := cmd.Flags().GetBool("no-commit")
	if !noCommit {
		msg := fmt.Sprintf("Toggle task %s to %s", t.ID, state)
		if err := sliceCommitWorkItem(path, msg, cfg); err != nil {
			return err
		}
		fmt.Println("Changes committed.")
	}
	return nil
}

func workItemIDFromPath(path string, cfg *config.Config) string {
	_, id, _, _, _, err := extractWorkItemMetadata(path, cfg)
	if err != nil || id == unknownValue {
		return ""
	}
	return id
}

// runSliceCommitNoSubcommand is run when "slice commit" is invoked without a valid subcommand (add, remove, generate). Returns error so exit code is non-zero.
func runSliceCommitNoSubcommand(_ *cobra.Command, _ []string) error {
	return fmt.Errorf("subcommand required: use 'add', 'remove', or 'generate'")
}

// runSliceCommitAdd adds a task to a slice. Args: 2 = slice-name + task-desc (doing folder); 3+ = work-item-id, slice-name, task-desc.
func runSliceCommitAdd(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	var workItemID, sliceName, description string
	if len(args) >= 3 {
		workItemID = args[0]
		sliceName = args[1]
		description = strings.Join(args[2:], " ")
	} else {
		sliceName = args[0]
		description = strings.Join(args[1:], " ")
	}
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice commit add")
	if err != nil {
		return err
	}
	id := workItemIDFromPath(path, cfg)
	if id == "" {
		id = workItemID
	}
	return runSliceTaskAdd(cmd, []string{id, sliceName, description})
}

// runSliceCommitRemove removes a slice. Args: 1 = slice-name (doing folder); 2 = work-item-id, slice-name.
func runSliceCommitRemove(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	var workItemID, sliceName string
	if len(args) >= 2 {
		workItemID = args[0]
		sliceName = args[1]
	} else {
		sliceName = args[0]
	}
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice commit remove")
	if err != nil {
		return err
	}
	id := workItemIDFromPath(path, cfg)
	if id == "" {
		id = workItemID
	}
	return runSliceRemove(cmd, []string{id, sliceName})
}

// runSliceCommitGenerate prints a structured commit message to stdout in the PRD format.
func runSliceCommitGenerate(_ *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	workItemID := ""
	selector := "current"
	if len(args) > 0 {
		workItemID = args[0]
	}
	if len(args) > 1 {
		selector = args[1]
	}
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice commit generate")
	if err != nil {
		return err
	}
	id := workItemIDFromPath(path, cfg)
	if id == "" {
		id = workItemID
	}
	out, err := formatGeneratedCommitMessage(path, cfg, id, selector)
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

// selectSliceBySelector returns the slice for generate: current (first with open tasks), previous, or by name.
func selectSliceBySelector(slices []Slice, selector string) (*Slice, error) {
	if len(slices) == 0 {
		return nil, fmt.Errorf("no slices in work item")
	}
	switch strings.ToLower(selector) {
	case "current":
		s := firstSliceWithOpenTasks(slices)
		if s == nil {
			return nil, fmt.Errorf("no slice with open tasks (all done)")
		}
		return s, nil
	case "previous":
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
		if curIdx <= 0 {
			return nil, fmt.Errorf("no previous slice")
		}
		return &slices[curIdx-1], nil
	default:
		s := findSliceByName(slices, selector)
		if s == nil {
			return nil, fmt.Errorf("slice %q not found", selector)
		}
		return s, nil
	}
}

// formatGeneratedCommitMessage builds the exact PRD format: line1 id+message, line2 id-kebab-title, line3 slice name, then task lines.
func formatGeneratedCommitMessage(path string, cfg *config.Config, workItemID, selector string) (string, error) {
	content, err := safeReadFile(path, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to read work item: %w", err)
	}
	slices, err := ParseSlicesSection(content)
	if err != nil || slices == nil {
		return "", fmt.Errorf("failed to parse Slices section: %w", err)
	}
	chosen, err := selectSliceBySelector(slices, selector)
	if err != nil {
		return "", err
	}
	fullMsg := generateSliceCommitMessage(path, cfg, workItemID)
	oneLine := fullMsg
	if idx := strings.Index(fullMsg, "\n"); idx >= 0 {
		oneLine = fullMsg[:idx]
	}
	_, _, title, _, _, _ := extractWorkItemMetadata(path, cfg)
	slug := workItemID + "-" + kebabCase(title)
	if title == "" || title == unknownValue {
		slug = workItemID
	}
	var b strings.Builder
	b.WriteString(workItemID + " " + strings.TrimSpace(oneLine) + "\n")
	b.WriteString("\n")
	b.WriteString(slug + "\n")
	b.WriteString("\n")
	b.WriteString(chosen.Name + ":\n")
	for _, t := range chosen.Tasks {
		b.WriteString("- " + t.ID + " " + t.Description + "\n")
	}
	return b.String(), nil
}

// generateSliceCommitMessage builds a commit message from task state changes, or fallback.
func generateSliceCommitMessage(path string, cfg *config.Config, workItemID string) string {
	content, err := safeReadFile(path, cfg)
	if err != nil {
		return fallbackSliceCommitMessage(path, cfg, workItemID)
	}
	current, err := ParseSlicesSection(content)
	if err != nil || current == nil {
		return fallbackSliceCommitMessage(path, cfg, workItemID)
	}
	previous := loadPreviousSlicesFromGit(path)
	completed, reopened, added := detectTaskChanges(previous, current)
	msg := formatSliceCommitParts(completed, reopened, added)
	if msg != "" {
		return msg
	}
	return fallbackSliceCommitMessage(path, cfg, workItemID)
}

func loadPreviousSlicesFromGit(path string) []Slice {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()
	output, err := executeCommand(ctx, "git", []string{"show", "HEAD:" + path}, "", false)
	if err != nil {
		return nil
	}
	previous, _ := ParseSlicesSection([]byte(output))
	return previous
}

func detectTaskChanges(previous, current []Slice) (completed, reopened, added []Task) {
	currentByID := make(map[string]Task)
	for _, s := range current {
		for _, t := range s.Tasks {
			currentByID[t.ID] = t
		}
	}
	if previous == nil {
		for _, t := range currentByID {
			added = append(added, t)
		}
		return completed, reopened, added
	}
	prevByID := make(map[string]bool)
	for _, s := range previous {
		for _, t := range s.Tasks {
			prevDone := t.Done
			cur, ok := currentByID[t.ID]
			if ok {
				if cur.Done && !prevDone {
					completed = append(completed, cur)
				}
				if !cur.Done && prevDone {
					reopened = append(reopened, cur)
				}
			}
			prevByID[t.ID] = true
		}
	}
	for id, t := range currentByID {
		if !prevByID[id] {
			added = append(added, t)
		}
	}
	return completed, reopened, added
}

func formatSliceCommitParts(completed, reopened, added []Task) string {
	var parts []string
	if len(completed) > 0 {
		ids := make([]string, 0, len(completed))
		descs := make([]string, 0, len(completed))
		for _, t := range completed {
			ids = append(ids, t.ID)
			descs = append(descs, t.Description)
		}
		parts = append(parts, fmt.Sprintf("Complete %s: %s", strings.Join(ids, ", "), strings.Join(descs, "; ")))
	}
	if len(reopened) > 0 {
		ids := make([]string, 0, len(reopened))
		for _, t := range reopened {
			ids = append(ids, t.ID)
		}
		parts = append(parts, fmt.Sprintf("Reopen %s", strings.Join(ids, ", ")))
	}
	if len(added) > 0 {
		ids := make([]string, 0, len(added))
		for _, t := range added {
			ids = append(ids, t.ID)
		}
		parts = append(parts, fmt.Sprintf("Add tasks %s", strings.Join(ids, ", ")))
	}
	if len(parts) > 0 {
		return strings.Join(parts, "\n")
	}
	return ""
}

func fallbackSliceCommitMessage(path string, cfg *config.Config, workItemID string) string {
	content, err := safeReadFile(path, cfg)
	if err != nil {
		return "Update slices for " + workItemID
	}
	slices, _ := ParseSlicesSection(content)
	if s := firstSliceWithOpenTasks(slices); s != nil {
		return s.Name
	}
	_, _, title, _, _, err := extractWorkItemMetadata(path, cfg)
	if err == nil && title != "" && title != unknownValue {
		return title
	}
	return "Update slices for " + workItemID
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
		return runSliceLintOne(cfg, args[0], outputFormat)
	}
	return runSliceLintAll(cfg, outputFormat)
}

func runSliceLintOne(cfg *config.Config, workItemID, outputFormat string) error {
	path, err := resolveSliceWorkItem(workItemID, cfg, "slice lint")
	if err != nil {
		return err
	}
	errors := lintSlicesSection(path, cfg)
	if outputFormat == sliceLintOutputJSON {
		return outputSliceLintJSON(errors)
	}
	return outputSliceLintHuman(path, errors)
}

func runSliceLintAll(cfg *config.Config, outputFormat string) error {
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
		if err := outputSliceLintHuman(path, errs); err != nil {
			anyFailed = true
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
