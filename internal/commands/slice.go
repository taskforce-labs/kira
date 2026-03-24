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

var sliceCommitCmd = &cobra.Command{
	Use:   "commit [current | <work-item-id>] [completed | <slice-number> | <slice-name>]",
	Short: "Commit the work item with a generated message for a completed slice",
	Long: `Stage the work item file and create a git commit with a multi-line message for the target slice.

The first optional argument is the work item selector: "current", a work-item id, or omit for current (same resolution as other slice commands). These are placeholders, not separate commands.

The second optional argument selects the slice: "completed" (default), a 1-based slice number, or slice name. "completed" means the slice you just finished—the slice before the first slice that still has open tasks, or the last slice when all tasks are done (same as "previous").

The commit body layout is: line 1 as <work-item-id>:<slice-number>. <slice-name>; then the slug line (<id>-<kebab-title>); optional Message or Commit line from the work item (line after ###); then a numbered slice heading (N. name) and task bullets. Optional -m/--message text is appended after the task list. The slice Message or Commit line is copied as in the file when present. Slice name and task lines use plain text (inline markdown stripped) except the Message/Commit line, which is copied verbatim.

Staging runs git add -A (all changes in the repository), then git commit, so code and work item are committed together.`,
	Args:         cobra.MaximumNArgs(2),
	RunE:         runSliceCommit,
	SilenceUsage: false,
}

func init() {
	sliceCmd.AddCommand(sliceAddCmd)
	sliceCmd.AddCommand(sliceRemoveCmd)
	sliceCmd.AddCommand(sliceTaskCmd)
	sliceCmd.AddCommand(sliceShowCmd)
	sliceCmd.AddCommand(sliceProgressCmd)
	sliceCmd.AddCommand(sliceLintCmd)
	sliceCmd.AddCommand(sliceCommitCmd)
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

	sliceCommitCmd.Flags().Bool("dry-run", false, "Print validation, commit message, and intended git steps; do not stage or commit")
	sliceCommitCmd.Flags().StringP("message", "m", "", "Supplementary text appended after the task list (does not replace the generated template)")
	sliceCommitCmd.Flags().String("override-message", "", "Use this as the full commit body instead of the generated template (cannot be combined with -m/--message)")
	sliceCommitCmd.Flags().Bool("commit-check", false, "Run kira check with tag filter before committing")
	sliceCommitCmd.Flags().StringSlice("commit-check-tags", nil, "Tags for --commit-check (default when flag omitted: commit). Replaces the default when set with --commit-check")
}
