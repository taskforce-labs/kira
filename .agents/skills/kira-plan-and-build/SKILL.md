---
name: kira-plan-and-build
description: "Plan the included work item and then build using the following approach"
disable-model-invocation: false
---

Create a plan to implement the included work item as it's described using the following approach

For each **slice** (not each task):

1. Get current slice/tasks: `kira slice show current` (no argument when work item is clear from context). To specify the work item, pass one argument: `current` or `<work-item-id>` — e.g. `kira slice show current` or `kira slice show 047`. You can refer to a slice by its 1-based number or name in other commands (e.g. `kira slice show current 1`, `kira slice task add 2 "desc"`). Same for `kira slice task show current`.
2. Implement:
   - all tasks in that slice;
   - add/update unit tests and other relevant tests
   - follow secure coding practices
3. Verify:: `kira check -t commit` before committing. If checks fail, fix and re-run; only commit when they pass.
4. Commit (one commit per slice):
   - The commit must include **both** the code changes for the slice **and** the work item with that slice’s tasks marked done.
   - Edit the work item to check the boxes for the tasks you just completed in the `## Slices` section, then run `kira slice lint [current | <work-item-id>]` and fix any errors.
  - Commit: use `kira slice commit` (defaults: work item `current`, slice `completed`) -`kira slice commit` validates the target slice has no open tasks, then runs **`git add -A`** and commits with the generated multi-line template (code and work item together). Use `--dry-run` to preview; `--commit-check` to run `kira check` (default tag `commit`) before committing.
   - Use `kira slice task done current --next` to mark a task done and see the next task plus progress summary - which can help determine what is left to do;
5. Move to the next slice and repeat.

When all slices are done, run `kira check -t done`.

Do not mark the work item as done or in review leave in todo.