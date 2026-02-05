---
id: 041
title: slice commit
status: backlog
kind: prd
assigned:
created: 2026-02-05
tags: []
---

# slice commit

Kira slice commit is implemented wrong
## How it works now
Currently Slice commit does the following:
`kira slice commit <work-item-id> <commit-message>` commits changes with the provided message
`kira slice commit <work-item-id>` generates a commit message based on task state changes (completed, reopened, added tasks); when no task changes are detected, falls back to current slice name, work item title, or "Update slices for &lt;work-item-id&gt;"

## How it should work
delete `kira slice commit <work-item-id> <commit-message>`
delete `kira slice commit <work-item-id>`


`kira slice commit add <work-item-id> <slice-name> <task-description>` adds a task to the slice

`kira slice commit remove <work-item-id> <slice-name>` removes the slice from the work item

`kira slice commit generate <work-item-id> [<slice-name>|current|previous]` generates a commit message based on task state changes (completed, reopened, added tasks); when no task changes are detected, falls back to current slice name, work item title, or "Update slices for &lt;work-item-id&gt;"

Generated commit message format:
```
<work-item-id> <commit-message>

<work-item-id>-<kebab-case-work-item-title>

<slice-name>
<task-ids> <task-descriptions>
```
**example:**
```
001 Implement OIDC login flow and JWT validation

001-authentication

Auth Token Validation
T001 Implement OIDC login flow
T002 Add JWT token validation
```

## Context

The current `kira slice commit` behavior (commit with optional message or generate-and-commit) is being replaced because:

- The CLI conflates "generate a commit message" with "perform a git commit," and the desired workflow is to separate message generation from committing.
- The generated message format does not match the desired standard (work-item-id + slug + slice name + task lines).
- Slice/task mutations (add task, remove slice) are desired under the `slice commit` namespace for a single workflow entry point.

Existing behaviour in 026: `kira slice add`, `kira slice remove`, `kira slice task add` already provide add/remove. This PRD adds the same operations under `kira slice commit add` and `kira slice commit remove` (convenience or alternative surface) and replaces the old commit behaviour with `slice commit generate` that outputs the new format only (no git commit).

## Requirements

1. **Remove current slice commit behaviour**
   - Remove `kira slice commit [<work-item-id>] [commit-message]` (both forms: with message and without).
   - No command shall perform a git commit as part of `kira slice commit` unless explicitly added in a later requirement.

2. **`kira slice commit add`**
   - `kira slice commit add <work-item-id> <slice-name> <task-description>` adds a task to the named slice (same effect as `kira slice task add`).
   - When `<work-item-id>` is omitted, resolve from the configured doing folder (one work item required); clear error if zero or multiple.

3. **`kira slice commit remove`**
   - `kira slice commit remove <work-item-id> <slice-name>` removes the slice and all its tasks from the work item (same effect as `kira slice remove`).
   - When `<work-item-id>` is omitted, resolve from the configured doing folder; clear error if zero or multiple.

4. **`kira slice commit generate`**
   - `kira slice commit generate <work-item-id> [<slice-name>|current|previous]` outputs a commit message to stdout only (no git commit).
   - **Slice selector**: `current` = first slice (in order) with open tasks; `previous` = slice immediately before current in order; `<slice-name>` = that slice by name. Default when omitted: `current`.
   - **Message content**: Based on task state changes (completed, reopened, added) compared to `HEAD` for the work item path; when no task changes, use fallback (slice name, work item title, or "Update slices for &lt;work-item-id&gt;").
   - **Output format** (exactly):
     - Line 1: `<work-item-id> <commit-message>` (commit message is the one-line summary, e.g. "Implement OIDC login flow and JWT validation").
     - Line 2: `<work-item-id>-<kebab-case-work-item-title>` (slug).
     - Line 3: `<slice-name>`.
     - Following lines: `<task-id> <task-description>` for each task in the slice (or for the tasks that changed, per product decision; PRD example shows slice’s tasks).
   - When `<work-item-id>` is omitted, resolve from the configured doing folder; clear error if zero or multiple.

## Acceptance Criteria

- **AC1** Invoking `kira slice commit` with no subcommand (or unknown subcommand) prints usage and exits non-zero; no git commit is performed.
- **AC2** `kira slice commit add [<work-item-id>] <slice-name> <task-description>` adds a task to the slice; behaviour matches `kira slice task add` (including task ID assignment, markdown update). With no work-item-id, resolution from doing folder works; clear error when zero or multiple work items in doing.
- **AC3** `kira slice commit remove [<work-item-id>] <slice-name>` removes the slice and all its tasks; behaviour matches `kira slice remove` (including confirmation unless `--yes`). With no work-item-id, resolution from doing folder works; clear error when zero or multiple.
- **AC4** `kira slice commit generate [<work-item-id>] [current|previous|<slice-name>]` prints only to stdout a commit message in the specified format (line 1: id + message; line 2: id-kebab-title; line 3: slice name; then task lines). No git operations. Slice selector `current`/`previous`/named works; default is `current`. With no work-item-id, resolution from doing folder works; clear error when zero or multiple.
- **AC5** Generated message uses task state changes (completed, reopened, added) when available; fallback to slice name, work item title, or "Update slices for &lt;id&gt;" when no changes. Kebab-case uses the same algorithm as elsewhere (e.g. `kira new`).
- **AC6** Existing `kira slice add`, `kira slice remove`, `kira slice task add` continue to work unchanged.
- **AC7** Unit and/or integration tests cover add, remove, generate (format and slice selector), optional work-item-id, and doing-folder resolution errors.

## Slices

### Replace commit with subcommands
Commit: Replace slice commit with subcommands; add add, remove, generate
- [ ] T001: Remove old slice commit behaviour: drop `commit [work-item-id] [message]`; require a subcommand (add, remove, generate); print usage and exit non-zero when no subcommand
- [ ] T002: Add `slice commit add` and `slice commit remove` subcommands; wire to existing slice task add and slice remove logic (or delegate); support optional work-item-id and doing-folder resolution
- [ ] T003: Add `slice commit generate` subcommand (stub or full); support optional work-item-id and doing-folder resolution

### Commit add and remove
Commit: Complete slice commit add and remove: delegate to task add and slice remove
- [ ] T004: Implement `slice commit add [<work-item-id>] <slice-name> <task-description>`; delegate to runSliceTaskAdd; optional work-item-id from doing folder
- [ ] T005: Implement `slice commit remove [<work-item-id>] <slice-name>`; delegate to runSliceRemove; optional work-item-id from doing folder; same confirmation/--yes as slice remove

### Generate message format and slice selector
Commit: Complete slice commit generate: new message format, current/previous/slice-name selector
- [ ] T006: Implement generate slice selector (current | previous | <slice-name>); default current; use existing task-change detection and fallbacks for message content
- [ ] T007: Format output exactly: line 1 <work-item-id> <message>, line 2 <work-item-id>-<kebab-title>, line 3 <slice-name>, then <task-id> <description> per task (for chosen slice or changed tasks per product decision)

### Tests and docs
Commit: Complete slice commit tests and docs: unit/integration tests, AGENTS.md and docs
- [ ] T008: Add unit and/or integration tests for add, remove, generate (format and slice selector), optional work-item-id, doing-folder resolution errors
- [ ] T009: Update AGENTS.md and any docs that reference `kira slice commit` to describe new subcommands and workflow

## Implementation Notes

- **Message format**: The example in the PRD shows the slice name and then all tasks in that slice (T001, T002). Confirm whether generate should list only *changed* tasks (completed/reopened/added) or all tasks in the selected slice; the current code uses changed tasks for the one-line summary. The specified format (slice name + task lines) suggests listing tasks for the slice.
- **Reuse**: `runSliceTaskAdd`, `runSliceRemove`, `generateSliceCommitMessage` / `fallbackSliceCommitMessage`, `resolveSliceWorkItem`, and `extractWorkItemMetadata` in `slice_run.go` and `slice.go` should be reused or refactored; kebab-case lives in `new.go` (`kebabCase`).
- **Optional work-item-id**: Same pattern as `slice current`, `slice task current`, `slice lint`: when work-item-id is omitted, resolve from doing folder; require exactly one work item; clear errors for zero or multiple.
- **Breaking change**: Scripts or agents using `kira slice commit &lt;id&gt;` or `kira slice commit &lt;id&gt; "message"` will break; release notes should call this out and document the new subcommands.

## Release Notes

- **Breaking**: `kira slice commit [<work-item-id>] [commit-message]` has been removed. Use subcommands: `kira slice commit add`, `kira slice commit remove`, `kira slice commit generate`. Generate only prints a message; to commit, use the generated output with `git commit -F -` or equivalent.
- New: `kira slice commit add [<work-item-id>] <slice-name> <task-description>` — add a task to a slice.
- New: `kira slice commit remove [<work-item-id>] <slice-name>` — remove a slice (and its tasks).
- New: `kira slice commit generate [<work-item-id>] [current|previous|<slice-name>]` — print a structured commit message to stdout in the standard format (work-item-id line, slug, slice name, task lines).

