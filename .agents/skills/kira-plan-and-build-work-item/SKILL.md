---
name: kira-plan-and-build-work-item
description: "Plan the included work item and then build using the following approach"
disable-model-invocation: false
---

Create a plan to implement the included work item as it's described using the following approach

### Non-negotiables (slice boundaries)

The usual failure mode is **doing multiple slices—or the whole work item—in one shot** instead of **one commit per slice**. These rules prevent that.

- **One slice, one commit.** Each slice produces **exactly one** git commit (`kira slice commit --commit-check`) before you start the next slice. Do not batch multiple slices into a single commit.
- **Implement only the current slice.** Write and test only what that slice’s tasks require. Do not add code, flags, or files that belong to a later slice “while you’re here.”
- **Do not implement the full work item and commit once.** Never finish the entire card in one change set and then split or fix history afterward unless the user explicitly asks for a history repair.
- **After each slice commit**, confirm progress (e.g. `git log -1`, `kira slice show current`) before continuing. The next slice starts on top of that commit.

### Baseline

- Before the **first** slice, run **`kira check -t commit`**. If it fails, stop, tell the user, and wait for direction.

For each **slice** (not each task):

1. Get current slice/tasks: `kira slice show current` (no argument when work item is clear from context). To specify the work item, pass one argument: `current` or `<work-item-id>` — e.g. `kira slice show current` or `kira slice show 047`. You can refer to a slice by its 1-based number or name in other commands (e.g. `kira slice show current 1`, `kira slice task add 2 "desc"`). Same for `kira slice task show current`.
2. Implement **only this slice**:
   - all tasks in that slice (not tasks from later slices);
   - add/update unit tests and other relevant tests for **this slice’s behaviour**;
   - follow secure coding practices
3. Commit (one commit per slice)—this is also verification: use **`kira slice commit --commit-check`** every time. That runs **`kira check`** with the configured commit tag(s) **before** the git commit, so project checks (including tests if configured) must pass for each slice. If checks fail, fix and re-run; do not drop **`--commit-check`**.
   - The commit must include **both** the code changes for the slice **and** the work item with that slice’s tasks marked done.
   - Edit the work item to check the boxes for the tasks you just completed in the `## Slices` section, then run `kira slice lint [current | <work-item-id>]` and fix any errors.
   - **`kira slice commit --commit-check`** (defaults: work item `current`, slice `completed`). Validates the slice has no open tasks, runs **`kira check`**, then **`git add -A`** and commits with the generated multi-line template. Use **`--dry-run`** to preview; use **`--commit-check-tags`** only if you need tags other than the default for `--commit-check`.
   - Use `kira slice task done current --next` to mark a task done and see the next task plus progress summary - which can help determine what is left to do;
4. Move to the next slice and repeat.

When all slices are done, run `kira check -t done`.

Do not mark the work item as done or in review leave in todo.
