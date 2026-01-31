// Package commands implements slice command run logic.
package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"kira/internal/config"
)

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
	fmt.Printf("Updated task %s in work item %s\n", taskID, workItemID)
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
	state := "open"
	if slices[si].Tasks[ti].Done {
		state = "done"
	}
	fmt.Printf("Task %s set to %s in work item %s\n", taskID, state, workItemID)
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

func taskBox(done bool) string {
	if done {
		return "[x]"
	}
	return "[ ]"
}

func printAllSlices(slices []Slice) {
	for _, s := range slices {
		fmt.Printf("### %s\n", s.Name)
		for _, t := range s.Tasks {
			fmt.Printf("  %s %s: %s\n", taskBox(t.Done), t.ID, t.Description)
		}
		fmt.Println()
	}
}

func printSliceDetail(s Slice) {
	fmt.Printf("### %s\n", s.Name)
	for _, t := range s.Tasks {
		fmt.Printf("  %s %s: %s\n", taskBox(t.Done), t.ID, t.Description)
		if t.Notes != "" {
			fmt.Printf("    Notes: %s\n", t.Notes)
		}
	}
}

func printTaskDetail(t Task, sliceName string) {
	fmt.Printf("Task: %s\n", t.ID)
	fmt.Printf("Slice: %s\n", sliceName)
	fmt.Printf("Description: %s\n", t.Description)
	fmt.Printf("State: %s\n", map[bool]string{false: "open", true: "done"}[t.Done])
	if t.Notes != "" {
		fmt.Printf("Notes: %s\n", t.Notes)
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
	fmt.Printf("Work item %s: %d tasks (%d done, %d open)\n", workItemID, total, done, open)
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
		fmt.Printf("  %s: %d done, %d open\n", s.Name, sd, so)
	}
	return nil
}

func runSliceTaskCurrent(_ *cobra.Command, _ []string) error {
	return fmt.Errorf("slice task current: not implemented yet")
}

func runSliceCurrent(_ *cobra.Command, _ []string) error {
	return fmt.Errorf("slice current: not implemented yet")
}

func runSliceLint(_ *cobra.Command, _ []string) error {
	return fmt.Errorf("slice lint: not implemented yet")
}

func runSliceCommit(_ *cobra.Command, _ []string) error {
	return fmt.Errorf("slice commit: not implemented yet")
}
