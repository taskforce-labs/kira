// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Slice represents a logical grouping of tasks within a work item.
type Slice struct {
	Name          string // slice name/title
	Description   string // optional
	CommitSummary string // optional: suggested commit message when slice is complete
	Tasks         []Task
}

// Task represents an individual actionable item within a slice.
type Task struct {
	ID          string // T001, T002, etc.
	Description string
	Done        bool   // true = done, false = open (two-state only)
	Notes       string // Optional
}

// sliceCmd is the parent command for slice operations.
var sliceCmd = &cobra.Command{
	Use:   "slice",
	Short: "Manage slices and tasks within work items",
	Long: `Manage slices and tasks within work items. Slices group related tasks;
tasks are individual actionable items with stable IDs (T001, T002, ...).
Use slice show, progress, current, and task current to view; use slice add/remove
and slice task add/remove/edit/toggle/note to modify.`,
	SilenceUsage: true,
}

var sliceAddCmd = &cobra.Command{
	Use:          "add <work-item-id> <slice-name>",
	Short:        "Add a new slice to a work item",
	Args:         cobra.ExactArgs(2),
	RunE:         runSliceAdd,
	SilenceUsage: true,
}

var sliceRemoveCmd = &cobra.Command{
	Use:          "remove <work-item-id> <slice-name>",
	Short:        "Remove a slice and all its tasks",
	Args:         cobra.ExactArgs(2),
	RunE:         runSliceRemove,
	SilenceUsage: true,
}

var sliceTaskCmd = &cobra.Command{
	Use:          "task",
	Short:        "Task operations (add, remove, edit, toggle, note, current)",
	SilenceUsage: true,
}

var sliceTaskAddCmd = &cobra.Command{
	Use:          "add <work-item-id> <slice-name> <task-description>",
	Short:        "Add a task to a slice",
	Args:         cobra.MinimumNArgs(3),
	RunE:         runSliceTaskAdd,
	SilenceUsage: true,
}

var sliceTaskRemoveCmd = &cobra.Command{
	Use:          "remove <work-item-id> <task-id>",
	Short:        "Remove a task",
	Args:         cobra.ExactArgs(2),
	RunE:         runSliceTaskRemove,
	SilenceUsage: true,
}

var sliceTaskEditCmd = &cobra.Command{
	Use:          "edit <work-item-id> <task-id> <new-description>",
	Short:        "Update a task's description",
	Args:         cobra.MinimumNArgs(3),
	RunE:         runSliceTaskEdit,
	SilenceUsage: true,
}

var sliceTaskToggleCmd = &cobra.Command{
	Use:          "toggle <work-item-id> <task-id>",
	Short:        "Toggle task state (open ↔ done)",
	Args:         cobra.ExactArgs(2),
	RunE:         runSliceTaskToggle,
	SilenceUsage: true,
}

var sliceTaskNoteCmd = &cobra.Command{
	Use:          "note <work-item-id> <task-id> <note>",
	Short:        "Add or update task notes",
	Args:         cobra.MinimumNArgs(3),
	RunE:         runSliceTaskNote,
	SilenceUsage: true,
}

var sliceTaskCurrentCmd = &cobra.Command{
	Use:          "current [<work-item-id>] [<slice-name>|toggle]",
	Short:        "Show or toggle the current task",
	RunE:         runSliceTaskCurrent,
	SilenceUsage: true,
}

var sliceShowCmd = &cobra.Command{
	Use:          "show <work-item-id> [slice-name|task-id]",
	Short:        "Show slices and tasks",
	Args:         cobra.MinimumNArgs(1),
	RunE:         runSliceShow,
	SilenceUsage: true,
}

var sliceProgressCmd = &cobra.Command{
	Use:          "progress <work-item-id>",
	Short:        "Show progress summary",
	Args:         cobra.ExactArgs(1),
	RunE:         runSliceProgress,
	SilenceUsage: true,
}

var sliceCurrentCmd = &cobra.Command{
	Use:          "current [<work-item-id>]",
	Short:        "Show the current slice (first with open tasks)",
	RunE:         runSliceCurrent,
	SilenceUsage: true,
}

var sliceLintCmd = &cobra.Command{
	Use:          "lint [<work-item-id>]",
	Short:        "Validate the Slices section",
	RunE:         runSliceLint,
	SilenceUsage: true,
}

var sliceCommitCmd = &cobra.Command{
	Use:          "commit [<work-item-id>] [commit-message]",
	Short:        "Commit slice/task changes",
	RunE:         runSliceCommit,
	SilenceUsage: true,
}

func init() {
	sliceCmd.AddCommand(sliceAddCmd)
	sliceCmd.AddCommand(sliceRemoveCmd)
	sliceCmd.AddCommand(sliceTaskCmd)
	sliceCmd.AddCommand(sliceShowCmd)
	sliceCmd.AddCommand(sliceProgressCmd)
	sliceCmd.AddCommand(sliceCurrentCmd)
	sliceCmd.AddCommand(sliceLintCmd)
	sliceCmd.AddCommand(sliceCommitCmd)

	sliceTaskCmd.AddCommand(sliceTaskAddCmd)
	sliceTaskCmd.AddCommand(sliceTaskRemoveCmd)
	sliceTaskCmd.AddCommand(sliceTaskEditCmd)
	sliceTaskCmd.AddCommand(sliceTaskToggleCmd)
	sliceTaskCmd.AddCommand(sliceTaskNoteCmd)
	sliceTaskCmd.AddCommand(sliceTaskCurrentCmd)
}

func runSliceAdd(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args
	return fmt.Errorf("slice add: not implemented yet")
}

func runSliceRemove(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args
	return fmt.Errorf("slice remove: not implemented yet")
}

func runSliceTaskAdd(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args
	return fmt.Errorf("slice task add: not implemented yet")
}

func runSliceTaskRemove(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args
	return fmt.Errorf("slice task remove: not implemented yet")
}

func runSliceTaskEdit(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args
	return fmt.Errorf("slice task edit: not implemented yet")
}

func runSliceTaskToggle(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args
	return fmt.Errorf("slice task toggle: not implemented yet")
}

func runSliceTaskNote(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args
	return fmt.Errorf("slice task note: not implemented yet")
}

func runSliceTaskCurrent(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args
	return fmt.Errorf("slice task current: not implemented yet")
}

func runSliceShow(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args
	return fmt.Errorf("slice show: not implemented yet")
}

func runSliceProgress(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args
	return fmt.Errorf("slice progress: not implemented yet")
}

func runSliceCurrent(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args
	return fmt.Errorf("slice current: not implemented yet")
}

func runSliceLint(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args
	return fmt.Errorf("slice lint: not implemented yet")
}

func runSliceCommit(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args
	return fmt.Errorf("slice commit: not implemented yet")
}
