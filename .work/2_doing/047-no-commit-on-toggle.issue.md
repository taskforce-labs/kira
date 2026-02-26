---
id: 047
title: no commit on toggle
status: doing
kind: issue
assigned:
estimate: 0
created: 2026-02-26
tags: []
---

# no commit on toggle

Toggle (slice task toggle and slice task toggle current) should **not** commit by default. Default should be no commit, matching `kira move` behavior. Keep an opt-in way to commit when desired. This ticket also adds **`kira slice commit current`** — a shorthand (work item from context: doing folder / work tree) that generates a commit message, validates the current slice has no open tasks, and commits (convenience wrapper with a validation gate).

## Command form (verified against code)

**In the codebase today:** Toggling the current task is already `kira slice task toggle current [<work-item-id>]`. Implemented in `internal/commands/slice.go`: `sliceTaskToggleCmd` has subcommand `sliceTaskToggleCurrentCmd` (Use `"current [<work-item-id>]"`). So the path is `slice task toggle` → `current` → optional work-item-id. Error messages in `slice_run.go` say `"slice task toggle current"`.

**Previous form (no longer in code):** The old form `kira slice task current [<work-item-id>] toggle` has been removed — `slice task current` now only shows the current task (Use `"current [<work-item-id>] [<slice-name>]"`) and does not accept `toggle` as an argument.

**This ticket** does not change the CLI shape; it only changes the default commit behavior for the existing commands `kira slice task toggle <work-item-id> <task-id>` and `kira slice task toggle current [<work-item-id>]`.

## Steps to Reproduce

1. Run `kira slice task toggle current` (or `kira slice task toggle <work-item-id> <task-id>`).
2. Observe: a git commit is created for the work item file change.

## Expected Behavior

- **Default**: Toggle only updates the work item markdown (task checkbox open ↔ done). No git commit.
- **Opt-in commit**: When the user wants to commit the toggle in the same step, they pass a flag (e.g. `--commit` or `-c`, consistent with `kira move`). So: `kira slice task toggle current --commit` would stage and commit the work item change.

Align with `kira move`: move uses `--commit` / `-c` (default false). Slice toggle should behave the same way — default no commit, `--commit` to commit.

## Actual Behavior

- Toggle commits by default (only the work item file).
- User must pass `--no-commit` to avoid committing. This is the opposite of move and encourages accidental small commits when the intent is just to mark a task done.

## Solution

1. **Change default to no commit** for:
   - `kira slice task toggle <work-item-id> <task-id>`
   - `kira slice task toggle current [<work-item-id>]`
2. **Replace `--no-commit` with `--commit`** (or `-c`) on these commands, default false, so behavior matches `kira move`. When `--commit` is set, stage the work item file and commit with the same message style as today (e.g. "Toggle task T001 to done").
3. **Docs**: Update AGENTS.md, .cursor/commands/kira-plan-and-build.md, and any PRD/spec that describes the recommended loop (toggle then optionally commit; or toggle --commit when you want the work-item-only commit). Command form in docs is already `slice task toggle current`.

4. **Slice commit current (with validation):** Add `kira slice commit current` (and optionally `kira slice commit <work-item-id>` for explicit id). The keyword **`current`** is a shorthand: the work item is resolved from context (the repo/work tree you're in — e.g. doing folder or branch) so you don't have to pass the work-item-id when you're already working on that item. Same context-awareness as other slice commands that accept optional work-item-id.
   - **Behavior:** Equivalent to `kira slice commit generate [<work-item-id>] | git commit -F -` as a single command. When you use `current`, work-item-id is resolved from context (doing folder: single work item; or from branch/work tree when applicable). When omitted or when using an explicit id, same resolution as existing slice commands.
   - **Validation:** Before generating and committing, check that the **current slice** has no open tasks (all tasks in the current slice are done). If any task is still open, fail with a clear message (e.g. "Current slice has open tasks: T002, T003. Complete or toggle them before committing."). Optionally extend to "all slices must have all tasks done" for a full work-item completion check.
   - **Rationale:** Encourages completing a slice before committing; convenience wrapper around generate + git commit with validation as a gate. No need to type the work item ID when you're in that work item's context.

## Release Notes

- `kira slice task toggle` and `kira slice task toggle current` no longer create a git commit by default. Use `--commit` / `-c` to commit the work item change (aligned with `kira move`).
- **New:** `kira slice commit current` — shorthand that uses context (doing folder / work tree) to resolve the work item so you don't have to pass the id. Generates a commit message, validates that the current slice has no open tasks, then runs the commit. Fails with a clear message if there are open tasks in the current slice.

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
- [ ] T008: Update AGENTS.md: recommend loop with toggle (no commit by default), optional `kira slice commit generate | git commit -F -` or `kira slice commit current`; document `--commit`/`-c` for toggle.
- [ ] T009: Update .cursor/commands/kira-plan-and-build.md and internal/cursorassets/commands/kira-plan-and-build.md: toggle then commit via generate or `slice commit current`.
- [ ] T010: Update any PRD/spec that describes the slice workflow to mention default no-commit on toggle and `kira slice commit current`.

