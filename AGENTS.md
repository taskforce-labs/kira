# Agents

See [.docs/agents/](.docs/agents/) for comprehensive agent documentation.

## Checking your work
Check your work by running `make check` after making changes.
This will run the following checks:
- Linting
- Security
- Testing (unit tests and e2e tests)
- Code coverage

If make check passes, run the following to verify the e2e tests pass:
`bash kira_e2e_tests.sh` will run the e2e tests and are worth running

## Golang

See [Go Secure Coding Practices](.docs/guides/security/golang-secure-coding.md) for comprehensive security guidelines covering:
- File path validation patterns
- File permissions
- Command execution security
- When to use `#nosec` comments

DON'T RELAX THESE RULES FOR TEST FILES DO NOT CHANGE .golangci.yml TO RELAX RULES UNDER ANY CIRCUMSTANCES UNLESS I TELL YOU TO DO SO EXPLICITLY.

## Slices (work item breakdown)
Work items can include a `## Slices` section with slices and tasks (e.g. `### 1. SliceName`, `- [ ] T001: description`). Use `kira slice` to manage them. Generated sections use numbered headings (`### 1. Name`, `### 2. Name`); the parser also accepts unnumbered headings (`### Name`). You can refer to a slice by **1-based number** or by name in commands (e.g. `kira slice show current 1`, `kira slice task add current 2 "desc"`).

**Selectors:** In `kira slice` commands, `current` and `completed` are **keywords** (argument placeholders), not separate subcommands. Omitting the work-item argument where allowed uses the same resolution as `current`. For commits, **`completed`** selects the slice you just finished (same as **`previous`**: the slice before the first slice that still has open tasks, or the last slice when all tasks are done).

**`kira slice commit`** (after a slice’s tasks are all checked off in the work item): `kira slice commit [current | <work-item-id>] [completed | <slice-number> | <slice-name>]`. Defaults: work item `current`, slice `completed`. Validates that every task in the target slice is done, then runs `git add -A` and `git commit` with a multi-line message (code and work item together): line 1 is `<id>:<slice-number>. <slice-name>`, then slug, optional `Message:`/`Commit:` line, numbered slice heading and task bullets; `-m` appends supplementary text after the task list. Flags: `--dry-run`, `-m` / `--message`, `--override-message`, `--commit-check` (default tag `commit`; `--commit-check-tags` overrides).

**Agent implementation loop (recommended):**
1. Get context: `kira slice show` or `kira slice task show` (omit the first arg to use work item from context—branch or single file in doing—or pass `current` or `<work-item-id>`). Use `--output json` on `slice task show` for machine-readable output.
2. Implement the current task (use task_id and description from step 1).
3. Mark task done: `kira slice task done current`. Use `kira slice task done current --next` to mark done and see the next task plus progress summary (e.g. 2/4 slices · 10/20 tasks · 1/3 in current slice). Done does **not** commit by default; commit with `kira slice commit` (or `git commit` after staging), or use `--commit`/`-c` on `slice task done` to commit the work item file in the same step (e.g. `kira slice task done current --commit`).
4. If you edited the Slices section markdown directly, run `kira slice lint` and fix any reported errors.
5. Repeat from step 1 for the next task, or stop if no open tasks.

Slice commands print a one-line progress summary by default. Use `--hide-summary` on any slice command to suppress it (e.g. `kira slice task show current --hide-summary`).

**Direct-edit + lint workflow:** When adding or changing many slices/tasks, edit the work item markdown directly (add or replace the `## Slices` section), then run `kira slice lint [current | <work-item-id>]` (or `--output json`). Fix any reported errors and re-run until clean.

## Instruction to Cursor
When editing markdown or specs, make direct edits to the file.
Do not propose patch-style changes or require accept/reject confirmation.
I will review changes using git diff after you finish.