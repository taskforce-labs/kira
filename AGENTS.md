# Agents

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

See [Go Secure Coding Practices](docs/security/golang-secure-coding.md) for comprehensive security guidelines covering:
- File path validation patterns
- File permissions
- Command execution security
- When to use `#nosec` comments

DON'T RELAX THESE RULES FOR TEST FILES DO NOT CHANGE .golangci.yml TO RELAX RULES UNDER ANY CIRCUMSTANCES UNLESS I TELL YOU TO DO SO EXPLICITLY.

## Slices (work item breakdown)
Work items can include a `## Slices` section with slices and tasks (e.g. `### SliceName`, `- [ ] T001: description`). Use `kira slice` to manage them.

**Agent implementation loop (recommended):**
1. Get context: `kira slice current` or `kira slice task current` (omit work-item-id when one work item is in doing). Use `--output json` for machine-readable output.
2. Implement the current task (use task_id and description from step 1).
3. Mark task done: `kira slice task current toggle`, then optionally `kira slice commit`.
4. If you edited the Slices section markdown directly, run `kira slice lint` and fix any reported errors.
5. Repeat from step 1 for the next task, or stop if no open tasks.

**Direct-edit + lint workflow:** When adding or changing many slices/tasks, edit the work item markdown directly (add or replace the `## Slices` section), then run `kira slice lint [work-item-id]` (or `--output json`). Fix any reported errors and re-run until clean.

## Instruction to Cursor
When editing markdown or specs, make direct edits to the file.
Do not propose patch-style changes or require accept/reject confirmation.
I will review changes using git diff after you finish.