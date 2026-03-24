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

## Selectors: `current` and `completed` are not commands

In **`kira slice`**, **`current`** and **`completed`** are **keywords that stand in for a work-item id or slice identity** when the real value comes from **context** (doing folder, branch, worktree, etc.). They are **argument placeholders**, not separate user-facing “commands” or verbs. Omitting an optional selector, where the grammar allows, means the same as supplying **`current`** for the work item when that is the default.

The shipped CLI may spell a path such as **`kira slice task done current …`** because of how the binary groups subcommands; read the trailing tokens as **selectors and flags**, not as a literal instruction to “run current.”

## `kira slice task done`

Mark the first open task in the **current slice** done.

- **Work item:** Optional selector — **`current`**, **`<work-item-id>`**, or omit (resolve from context). Same resolution rules as other `slice` commands.
- **Flags:** **`--commit`** / **`-c`** — stage and commit the work item file in the same step (default: no commit). **`--next`** — after marking done, show the next task and whether it is in the same slice or the next slice. **`--hide-summary`** — suppress the one-line progress summary (e.g. with **`--next`**).

Typical output includes what was completed (e.g. **`Completed: T001 — …`**). With **`--next`**, also a one-line summary such as **`2/4 slices · 10/20 tasks · 1/3 in current slice`** unless hidden.

Slice commands that change or show slice state print that one-line summary by default unless **`--hide-summary`** is set (and unless output must stay machine-clean, e.g. **`--output json`**).

## `kira slice commit`

**Invocation:**

```text
kira slice commit [<work-item-selector>] [<slice-selector>] \
  [--dry-run] [-m | --message <message>] [--override-message <message>] [--commit-check [<tag> ...]]
```

**`-m` / `--message`:** optional **extra context** appended to the **default** line-1 summary (see below). It **does not replace** the default summary.

Where:

- **`<work-item-selector>`** — **`current`**, **`<work-item-id>`**, or omit → default **`current`** (context).
- **`<slice-selector>`** — **`completed`**, **`<slice-no>`** (1-based), **`<slice-name>`**, or omit → default **`completed`**.

So **`kira slice commit`**, **`kira slice commit current`**, and **`kira slice commit current completed`** are equivalent.

### Commit message template

**Implementation today:** `internal/commands/slice_run.go` only has **`sliceCommitWorkItem`**, which runs **`git commit -m <single-line message>`**. The multi-line layout below is **not** implemented in code yet; it is the **target** for **`kira slice commit`** (slice **9**), aligned with **`.work/4_done/041-slice-commit.prd.md`**.

Unless **`--override-message`** is set, the git commit body uses this **normative multi-line shape**:

| Part | Content |
|------|---------|
| **Line 1** | `<work-item-id>` + **`:`** + `<slice-number>` + **`. `** + **slice name** (1-based slice index and heading title, same as in the Slices section) |
| **Blank** | |
| **Line 2** | `<work-item-id>-<kebab-case-title>` (slug from work item title) |
| **Blank** | |
| **Slice message** | Optional **Message:** / **Commit:** line after the `###` heading (verbatim when present) |
| **Blank** | (omitted if no slice message) |
| **Slice heading** | `<slice-number>.` + space + **slice name** (matches generated `### N. Name` title) |
| **Following** | One line per task: **`- `** + `<task-id>` + space + plain description |
| **Optional** | **`-m` / `--message`**: supplementary paragraph **after** the task list (whitespace collapsed to one line unless documented otherwise) |

### Line 1

Line 1 is **`&lt;work-item-id&gt;:&lt;slice-number&gt;. &lt;slice-name&gt;`** (phase = slice number and name). There is no separate “base summary” chain on line 1; the slug line still uses the work item **title** for **`-&lt;kebab-case-title&gt;`**.

**`-m` / `--message` (supplementary):** optional. **Does not replace** the generated template. When set, append **after** the task bullets (see schematic below). Newlines in **`-m`** are normalized to spaces for a single trailing paragraph.

**`--override-message`:** replaces the **entire** commit body; template not used. **`--override-message`** wins over **`-m`** for the final body (or error if both are ambiguous — pick one rule and document it).

**Schematic:**

```text
<work-item-id>:<slice-number>. <slice-name>

<work-item-id>-<kebab-case-work-item-title>

<slice-message>

<slice-name>
- <task-id> <task-description>
- <task-id> <task-description>
…

[<supplementary from -m if any>]
```

**Example:**

```text
001:1 Auth Token Validation

001-authentication

Implement OIDC login flow and JWT validation

1. Auth Token Validation
- T001 Implement OIDC login flow
- T002 Add JWT token validation


Refreshed OpenAPI fixtures for error shapes
```

*(Without **`-m`**, there is no supplementary paragraph after the task list.)*

### Behavior

1. Resolve work item and target slice from selectors. **`completed`** picks the slice this commit is for (implementation must resolve unambiguously).
2. **Validation:** every task in the target slice must be **`- [x]`**. If any task is open, fail with a clear error listing open task ids.

   ```markdown
   ### 2. My task
   - [x] foo
   - [ ] bar
   ```

3. Build message per template unless **`--override-message`** (see schematic: line 1 = **`id:sliceNo. name`**; optional **`-m`** after tasks); stage work item per **`kira move`** conventions; **`git commit`** unless **`--dry-run`**.
4. **`--dry-run`:** print validation result, full final message, and intended git steps; no commit.
5. **`--commit-check`:** run **`kira check`** with tag filter; **default** **`kira check -t commit`**. Tags after the flag **replace** that default (e.g. **`--commit-check lint e2e`** → checks tagged `lint` or `e2e`).

## Acceptance criteria (`kira slice commit`)

- [ ] **AC1:** No args → same as explicit **`current`** + **`completed`** defaults.
- [ ] **AC2:** Fails if the target slice has any open task; succeeds only when all are done.
- [ ] **AC3:** Line 1 is **`&lt;id&gt;:&lt;slice-number&gt;. &lt;slice-name&gt;`**; **`-m` / `--message`** appends supplementary text **after** the task list (does not replace the template); **`--override-message`** replaces the full body.
- [ ] **AC4:** Non-override commits match the normative template (lines + blanks; task lines each prefixed with **`- `**).
- [ ] **AC5:** **`--dry-run`** shows validation, full message, and git intent; no commit.
- [ ] **AC6:** **`--commit-check`** defaults to **`commit`** tag; explicit tags override.
- [ ] **AC7:** Selectors documented as placeholders; help and AGENTS/plan-and-build match this spec.

---

The checklist below spans multiple iterations of the CLI; several behaviors and command shapes changed along the way. Treat **this document above the line** as the source of truth for how things work **now**; the slices preserve task history without re-explaining each change.

## Slices

### 1. Toggle default no-commit and --commit flag *(historical; referred to removed toggle CLI)*
Message: Default to no commit for slice task toggle (both forms); add --commit/-c opt-in aligned with kira move.
- [x] T001: Change default to no commit for `slice task toggle <work-item-id> <task-id>` and `slice task current [<work-item-id>] toggle` (update slice_run.go so commit only when flag set).
- [x] T002: Replace `--no-commit` with `--commit`/`-c` (default false) on sliceTaskToggleCmd and sliceTaskCurrentCmd; when set, stage work item file and commit with same message style (e.g. "Toggle task T001 to done").
- [x] T003: Add/update unit tests for toggle with and without --commit; ensure no commit by default, commit when --commit.

### 2. slice commit current (with validation)
Message: Add kira slice commit current: resolve work item from context, validate current slice has no open tasks, then generate and commit.
- [x] T004: Add subcommand `slice commit current [<work-item-id>]`; resolve work item from args or doing folder (same as other slice commit commands).
- [x] T005: Before committing: validate the slice to be committed (e.g. "previous" — the one just completed) has no open tasks; if any open tasks, fail with clear message listing task IDs.
- [x] T006: On success: run generate for that slice and execute `git commit -F -` (reuse sliceCommitWorkItem or equivalent); ensure git availability and single work-item staging checks consistent with move/generate.
- [x] T007: Add tests for `slice commit current`: success when current slice complete; failure when open tasks remain; work-item resolution from doing folder and explicit id.

### 3. Documentation
Message: Update AGENTS.md, kira-plan-and-build, and any loop docs to describe toggle no-commit default and slice commit current.
- [x] T008: Update AGENTS.md: recommend loop with toggle (no commit by default), optional `kira slice commit generate | git commit -F -` or `kira slice commit current`; document `--commit`/`-c` for toggle.
- [x] T009: Update .cursor/commands/kira-plan-and-build.md and internal/cursorassets/commands/kira-plan-and-build.md: toggle then commit via generate or `slice commit current`.
- [x] T010: Update any PRD/spec that describes the slice workflow to mention default no-commit on toggle and `kira slice commit current`.

### 4. slice task done current and --next (expanded scope)
Message: Add slice task done current; output completed task; --next shows next task and summary.
- [x] T011: Add `slice task done [current  | <work-item-id>]`: resolve work item, find current open task, mark done, write file, print "Completed: <id> - <description>". Flags: --commit/-c, --next, --hide-summary.
- [x] T012: With --next: after marking done, show next task and "same slice" vs "next slice: <name>"; print one-line summary (e.g. 2/4 slices · 10/20 tasks · 1/3 in current slice). Honor --hide-summary for the summary line. When all tasks done, print "All tasks complete" and full summary.
- [x] T013: Add helper formatSliceSummary(slices, currentSliceName) and printSliceSummaryIf(cmd, path, cfg, currentSliceName); unit tests for summary format and done/--next behavior.

### 5. --hide-summary and summary on slice commands (expanded scope)
Message: Add --hide-summary to slice commands; print one-line summary by default where applicable.
- [x] T014: Add --hide-summary (e.g. persistent on sliceCmd or per-command) to slice add/remove, slice task add/remove/edit/toggle/note/current/done, slice show, slice current, slice progress, slice commit add/remove/current, slice lint. Do not add summary to slice commit generate when stdout is commit message.
- [x] T015: At end of each runSlice* that modifies or displays state, call printSliceSummaryIf unless --hide-summary. For --output json, do not print human summary.

### 6. Toggle completion output (expanded scope)
Message: When slice task current ... toggle marks task done, print "Completed: <id> - <description>" for consistency.
- [x] T016: N/A — `slice task current toggle` removed; use `slice task done current` instead.

### 7. Documentation for expanded scope
- [x] T017: Update AGENTS.md and kira-plan-and-build: recommend `slice task done current` and `--next`; document `--hide-summary` for slice commands.

### 8. Remove kira commit commands
- [x] T018: remove commands

### 9. Add new kira slice commit command
Message: Implement **`kira slice commit`** per **Acceptance criteria (`kira slice commit`)** and the **`kira slice commit`** section above.
- [x] T019: Wire **`kira slice commit`** with work-item and slice **selectors** (defaults **`current`** / **`completed`**), flags **`--dry-run`**, **`-m` / `--message`** (append-only context), **`--override-message`**, **`--commit-check`** (optional trailing tags). Help text describes selectors as placeholders, not extra commands.
- [x] T020: Resolve work item and target slice from selectors; support **`completed`**, slice number, slice name.
- [x] T021: Validation: all tasks in target slice done; clear errors with open task ids (include mixed-checkbox case in tests).
- [x] T022: Build commit body from normative template (bulleted task lines); line 1 **base summary** from documented fallback chain; **`-m`** appends supplementary text (separator + rules for multiline); **`--override-message`** = full body; document interaction if both **`-m`** and **`--override-message`** are set.
- [x] T023: Stage and **`git commit`**; **`--dry-run`** prints validation + message + git intent only.
- [x] T024: **`--commit-check`:** default **`kira check -t commit`**; explicit tags override; fail closed on check failure.
- [x] T025: Tests: defaults, validation, dry-run, messages, commit-check.
- [x] T026: Update AGENTS.md and plan-and-build to match this issue (selectors, **`kira slice commit`**, task done flags).

