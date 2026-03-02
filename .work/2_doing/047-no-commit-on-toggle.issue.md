---
id: 047
title: no commit on toggle + slice task done and summary
status: doing
kind: issue
assigned:
estimate: 0
created: 2026-02-26
tags: []
---

# no commit on toggle + slice task done and summary

**Original scope (done):** Toggle and slice commit current — toggle should not commit by default; add `kira slice commit current` with validation. All original slices below are complete.

**Expanded direction:** This work item has been extended to improve the slice workflow for agents and LLMs: add a dedicated **`kira slice task done current`** command that marks the current task done and outputs what was completed; support **`--next`** to show the next task and whether it’s in the same or next slice, with a one-line progress summary; and add a **consistent one-line summary** across slice commands, toggleable with **`--hide-summary`**.

## Command form (verified against code)

**Update:** `kira slice task current toggle` has been removed; use `kira slice task done current` instead.

**Toggle by task ID (unchanged):** `kira slice task toggle <work-item-id> <task-id>` still toggles a task by explicit ID. The variant `slice task current ... toggle` (second arg literal `toggle`) has been removed.

**This ticket (original):** Changed default commit behavior for `kira slice task toggle <work-item-id> <task-id>` and `kira slice task current [<work-item-id>] toggle`; did not change the CLI shape.

**This ticket (expansion):** Adds new command `kira slice task done current [<work-item-id>]` and `--hide-summary` on slice commands; optional `--next` on `slice task done current`.

## Original issue: Steps to Reproduce

1. Run `kira slice task current toggle` (or `kira slice task toggle <work-item-id> <task-id>`).
2. Observe: a git commit is created for the work item file change.

## Original issue: Expected Behavior

- **Default**: Toggle only updates the work item markdown (task checkbox open ↔ done). No git commit.
- **Opt-in commit**: When the user wants to commit the toggle in the same step, they pass a flag (e.g. `--commit` or `-c`, consistent with `kira move`). So: `kira slice task current toggle --commit` would stage and commit the work item change.

Align with `kira move`: move uses `--commit` / `-c` (default false). Slice toggle should behave the same way — default no commit, `--commit` to commit.

## Original issue: Actual Behavior

- Toggle commits by default (only the work item file).
- User must pass `--no-commit` to avoid committing. This is the opposite of move and encourages accidental small commits when the intent is just to mark a task done.

## Original issue: Solution (implemented)

1. **Change default to no commit** for:
   - `kira slice task toggle <work-item-id> <task-id>`
   - `kira slice task current [<work-item-id>] toggle`
2. **Replace `--no-commit` with `--commit`** (or `-c`) on these commands, default false, so behavior matches `kira move`. When `--commit` is set, stage the work item file and commit with the same message style as today (e.g. "Toggle task T001 to done").
3. **Docs**: Update AGENTS.md, .cursor/commands/kira-plan-and-build.md, and any PRD/spec that describes the recommended loop (toggle then optionally commit; or toggle --commit when you want the work-item-only commit). Command form in docs is already `slice task current ... toggle`.

4. **Slice commit current (with validation):** Add `kira slice commit current` (and optionally `kira slice commit <work-item-id>` for explicit id). The keyword **`current`** is a shorthand: the work item is resolved from context (the repo/work tree you're in — e.g. doing folder or branch) so you don't have to pass the work-item-id when you're already working on that item. Same context-awareness as other slice commands that accept optional work-item-id.
   - **Behavior:** Equivalent to `kira slice commit generate [<work-item-id>] | git commit -F -` as a single command. When you use `current`, work-item-id is resolved from context (doing folder: single work item; or from branch/work tree when applicable). When omitted or when using an explicit id, same resolution as existing slice commands.
   - **Validation:** Before generating and committing, check that the **current slice** has no open tasks (all tasks in the current slice are done). If any task is still open, fail with a clear message (e.g. "Current slice has open tasks: T002, T003. Complete or toggle them before committing."). Optionally extend to "all slices must have all tasks done" for a full work-item completion check.
   - **Rationale:** Encourages completing a slice before committing; convenience wrapper around generate + git commit with validation as a gate. No need to type the work item ID when you're in that work item's context.

## Expanded scope: slice task done and summary

- **`kira slice task done current [<work-item-id>]`** — Mark the current (first open) task as done. Always output what was completed (e.g. `Completed: T001 - Implement OIDC login flow`). Flags: `--commit`/`-c` (same as toggle), `--next` (see below), `--hide-summary` (suppress summary when used with `--next`).
- **`kira slice task done current --next`** — After marking done, show the next task and whether it is in the **same slice** or **next slice** (so the LLM knows the slice is complete when moving to a new slice). Print a one-line progress summary, e.g. `2/4 slices · 10/20 tasks · 1/3 in current slice` (completed slices / total slices; done tasks / total tasks; done/total in the current slice). Use `--hide-summary` to show the next task but omit the summary line.
- **Summary on slice commands** — Each slice command that modifies or displays slice state prints a one-line progress summary by default (same format as above). Add **`--hide-summary`** (e.g. on `slice` so all subcommands inherit, or per-command) to suppress it. Commands in scope: slice add/remove, slice task add/remove/edit/toggle/note/current/done, slice show, slice current, slice progress (or only suppress progress output when `--hide-summary`), slice commit add/remove/current, slice lint. Do not add summary to `slice commit generate` when stdout is used for the commit message (pipe).
- **Toggle output** — When `slice task current ... toggle` marks a task **done**, also print the completed task (e.g. `Completed: T001 - ...`) for consistency with `slice task done current`.

## Release Notes

- **Original (done):** `kira slice task toggle` and `kira slice task current ... toggle` no longer create a git commit by default. Use `--commit` / `-c` to commit the work item change (aligned with `kira move`). **New:** `kira slice commit current` — shorthand that uses context (doing folder / work tree) to resolve the work item; validates that the current slice has no open tasks, then runs the commit.
- **Expanded:** **New** `kira slice task done current [<work-item-id>]` — mark current task done and print what was completed. Use `--next` to show the next task (same slice vs next slice) and a one-line summary (e.g. 2/4 slices · 10/20 tasks · 1/3 in current slice). Use `--hide-summary` to suppress the summary. Slice commands now print a one-line progress summary by default; use `--hide-summary` to suppress.

## Slices

### Toggle default no-commit and --commit flag
Commit: Default to no commit for slice task toggle (both forms); add --commit/-c opt-in aligned with kira move.
- [x] T001: Change default to no commit for `slice task toggle <work-item-id> <task-id>` and `slice task current [<work-item-id>] toggle` (update slice_run.go so commit only when flag set).
- [x] T002: Replace `--no-commit` with `--commit`/`-c` (default false) on sliceTaskToggleCmd and sliceTaskCurrentCmd; when set, stage work item file and commit with same message style (e.g. "Toggle task T001 to done").
- [x] T003: Add/update unit tests for toggle with and without --commit; ensure no commit by default, commit when --commit.

### slice commit current (with validation)
Commit: Add kira slice commit current: resolve work item from context, validate current slice has no open tasks, then generate and commit.
- [x] T004: Add subcommand `slice commit current [<work-item-id>]`; resolve work item from args or doing folder (same as other slice commit commands).
- [x] T005: Before committing: validate the slice to be committed (e.g. "previous" — the one just completed) has no open tasks; if any open tasks, fail with clear message listing task IDs.
- [x] T006: On success: run generate for that slice and execute `git commit -F -` (reuse sliceCommitWorkItem or equivalent); ensure git availability and single work-item staging checks consistent with move/generate.
- [x] T007: Add tests for `slice commit current`: success when current slice complete; failure when open tasks remain; work-item resolution from doing folder and explicit id.

### Documentation
Commit: Update AGENTS.md, kira-plan-and-build, and any loop docs to describe toggle no-commit default and slice commit current.
- [x] T008: Update AGENTS.md: recommend loop with toggle (no commit by default), optional `kira slice commit generate | git commit -F -` or `kira slice commit current`; document `--commit`/`-c` for toggle.
- [x] T009: Update .cursor/commands/kira-plan-and-build.md and internal/cursorassets/commands/kira-plan-and-build.md: toggle then commit via generate or `slice commit current`.
- [x] T010: Update any PRD/spec that describes the slice workflow to mention default no-commit on toggle and `kira slice commit current`.

### slice task done current and --next (expanded scope)
Commit: Add slice task done current; output completed task; --next shows next task and summary.
- [ ] T011: Add `slice task done [current  | <work-item-id>]`: resolve work item, find current open task, mark done, write file, print "Completed: <id> - <description>". Flags: --commit/-c, --next, --hide-summary.
- [ ] T012: With --next: after marking done, show next task and "same slice" vs "next slice: <name>"; print one-line summary (e.g. 2/4 slices · 10/20 tasks · 1/3 in current slice). Honor --hide-summary for the summary line. When all tasks done, print "All tasks complete" and full summary.
- [ ] T013: Add helper formatSliceSummary(slices, currentSliceName) and printSliceSummaryIf(cmd, path, cfg, currentSliceName); unit tests for summary format and done/--next behavior.

### --hide-summary and summary on slice commands (expanded scope)
Commit: Add --hide-summary to slice commands; print one-line summary by default where applicable.
- [ ] T014: Add --hide-summary (e.g. persistent on sliceCmd or per-command) to slice add/remove, slice task add/remove/edit/toggle/note/current/done, slice show, slice current, slice progress, slice commit add/remove/current, slice lint. Do not add summary to slice commit generate when stdout is commit message.
- [ ] T015: At end of each runSlice* that modifies or displays state, call printSliceSummaryIf unless --hide-summary. For --output json, do not print human summary.

### Toggle completion output (expanded scope)
Commit: When slice task current ... toggle marks task done, print "Completed: <id> - <description>" for consistency.
- [x] T016: N/A — `slice task current toggle` removed; use `slice task done current` instead.

### Documentation for expanded scope
- [ ] T017: Update AGENTS.md and kira-plan-and-build: recommend `slice task done current` and `--next`; document `--hide-summary` for slice commands.

