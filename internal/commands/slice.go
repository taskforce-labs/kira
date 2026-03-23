// Package commands implements the CLI commands for the kira tool.
package commands

import (
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
Use slice show, progress, and task show to view; use slice add/remove
and slice task add/remove/edit/note to modify.`,
	SilenceUsage: false, // show usage when args are wrong
}

var sliceAddCmd = &cobra.Command{
	Use:          "add (current | <work-item-id>) <slice-name>",
	Short:        "Add a new slice to a work item",
	Args:         cobra.ExactArgs(2),
	RunE:         runSliceAdd,
	SilenceUsage: false, // show usage when args are wrong
}

var sliceRemoveCmd = &cobra.Command{
	Use:          "remove (current | <work-item-id>) (<slice-number> | <slice-name>)",
	Short:        "Remove a slice and all its tasks",
	Args:         cobra.ExactArgs(2),
	RunE:         runSliceRemove,
	SilenceUsage: false, // show usage when args are wrong
}

var sliceTaskCmd = &cobra.Command{
	Use:          "task",
	Short:        "Task operations (add, remove, edit, note, show, done)",
	SilenceUsage: false, // show usage when args are wrong
}

var sliceTaskAddCmd = &cobra.Command{
	Use:          "add (current | <work-item-id>) (<slice-number> | <slice-name>) <task-description>",
	Short:        "Add a task to a slice",
	Args:         cobra.MinimumNArgs(3),
	RunE:         runSliceTaskAdd,
	SilenceUsage: false, // show usage when args are wrong
}

var sliceTaskRemoveCmd = &cobra.Command{
	Use:          "remove (current | <work-item-id>) <task-id>",
	Short:        "Remove a task",
	Args:         cobra.ExactArgs(2),
	RunE:         runSliceTaskRemove,
	SilenceUsage: false, // show usage when args are wrong
}

var sliceTaskEditCmd = &cobra.Command{
	Use:          "edit (current | <work-item-id>) <task-id> <new-description>",
	Short:        "Update a task's description",
	Args:         cobra.MinimumNArgs(3),
	RunE:         runSliceTaskEdit,
	SilenceUsage: false, // show usage when args are wrong
}

var sliceTaskNoteCmd = &cobra.Command{
	Use:          "note (current | <work-item-id>) <task-id> <note>",
	Short:        "Add or update task notes",
	Args:         cobra.MinimumNArgs(3),
	RunE:         runSliceTaskNote,
	SilenceUsage: false, // show usage when args are wrong
}

var sliceTaskShowCmd = &cobra.Command{
	Use:          "show [current | <work-item-id>] [(<slice-number> | <slice-name>)]",
	Short:        "Show the current task",
	RunE:         runSliceTaskCurrent,
	SilenceUsage: false, // show usage when args are wrong
}

var sliceTaskDoneCmd = &cobra.Command{
	Use:          "done",
	Short:        "Mark current task done",
	Long:         "Mark the current (first open) task as done. Use 'done current' to resolve from context.",
	SilenceUsage: false, // show usage when args are wrong
}

var sliceTaskDoneCurrentCmd = &cobra.Command{
	Use:          "current [current | <work-item-id>]",
	Short:        "Mark the current task done and optionally show next",
	RunE:         runSliceTaskDoneCurrent,
	SilenceUsage: false, // show usage when args are wrong
}

var sliceShowCmd = &cobra.Command{
	Use:          "show [current | <work-item-id>] [all|current|<slice-number>|<slice-name>|<task-id>]",
	Short:        "Show slices and tasks",
	Long:         "With one arg: show current slice if work item is 'current', otherwise all slices. With two args: use second as 'all' (all slices), 'current' (current slice), slice number/name, or task-id. Omit first arg to use work item from context (branch or doing folder).",
	Args:         cobra.RangeArgs(0, 2),
	RunE:         runSliceShow,
	SilenceUsage: false, // show usage when args are wrong
}

var sliceProgressCmd = &cobra.Command{
	Use:          "progress [current | <work-item-id>]",
	Short:        "Show progress summary",
	Long:         "Omit the first argument to use the work item from context (branch or doing folder).",
	Args:         cobra.MaximumNArgs(1),
	RunE:         runSliceProgress,
	SilenceUsage: false, // show usage when args are wrong
}

var sliceLintCmd = &cobra.Command{
	Use:           "lint [current | <work-item-id>]",
	Short:         "Validate the Slices section",
	RunE:          runSliceLint,
	SilenceUsage:  false, // show usage when args are wrong
	SilenceErrors: true,  // main prints error once
}

func init() {
	sliceCmd.AddCommand(sliceAddCmd)
	sliceCmd.AddCommand(sliceRemoveCmd)
	sliceCmd.AddCommand(sliceTaskCmd)
	sliceCmd.AddCommand(sliceShowCmd)
	sliceCmd.AddCommand(sliceProgressCmd)
	sliceCmd.AddCommand(sliceLintCmd)
	sliceTaskCmd.AddCommand(sliceTaskAddCmd)
	sliceTaskCmd.AddCommand(sliceTaskRemoveCmd)
	sliceTaskCmd.AddCommand(sliceTaskEditCmd)
	sliceTaskCmd.AddCommand(sliceTaskNoteCmd)
	sliceTaskCmd.AddCommand(sliceTaskShowCmd)
	sliceTaskCmd.AddCommand(sliceTaskDoneCmd)
	sliceTaskDoneCmd.AddCommand(sliceTaskDoneCurrentCmd)

	sliceCmd.PersistentFlags().Bool("hide-summary", false, "Do not print the one-line slice/task progress summary")

	sliceAddCmd.Flags().BoolP("commit", "c", false, "Commit the work item change (default: no commit)")
	sliceRemoveCmd.Flags().BoolP("commit", "c", false, "Commit the work item change (default: no commit)")
	sliceRemoveCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	sliceTaskAddCmd.Flags().BoolP("commit", "c", false, "Commit the work item change (default: no commit)")
	sliceTaskRemoveCmd.Flags().BoolP("commit", "c", false, "Commit the work item change (default: no commit)")
	sliceTaskRemoveCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	sliceTaskEditCmd.Flags().BoolP("commit", "c", false, "Commit the work item change (default: no commit)")
	sliceTaskNoteCmd.Flags().BoolP("commit", "c", false, "Commit the work item change (default: no commit)")
	sliceTaskDoneCurrentCmd.Flags().BoolP("commit", "c", false, "Commit the work item change (default: no commit)")
	sliceTaskDoneCurrentCmd.Flags().Bool("next", false, "After marking done, show the next task and progress summary")
	sliceLintCmd.Flags().String("output", "", "Output format: json")
	sliceTaskShowCmd.Flags().String("output", "", "Output format: json")
	sliceShowCmd.Flags().String("output", "", "Output format: json")
}
