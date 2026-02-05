# RALF Work Item

## Overview
RALF (Read, Analyze, Learn, Fix) on work items: read state, analyze gaps, learn from feedback, fix and verify. Use when implementing or reviewing work items or slices (single or multiple).

## Steps

1. **Read Project Configuration**
   - Check `kira.yaml` for work folder and slice/status structure
   - Use `kira slice` and status folders as needed

2. **Read**
   - Read the work item(s) and current state (slices, tasks, acceptance criteria)
   - Understand scope and dependencies
   - Identify what is done and what is open

3. **Analyze**
   - Analyze gaps between current state and acceptance criteria
   - Identify failures, regressions, or missing behavior
   - Prioritize what to fix or implement next

4. **Learn**
   - Use test results, lint, and feedback to learn what is broken or missing
   - Incorporate feedback into the next steps
   - Update work item or tasks (e.g. `kira slice task toggle`, notes) as needed

5. **Fix**
   - Apply fixes or implement missing behavior
   - Include unit tests for any code changes
   - Get checks to run from project config: `kira config get checks`
   - Run those checks before each commit; ensure all checks (including e2e) pass by the end of the work item
   - Commit each slice when its task(s) are done (e.g. `kira slice task current toggle` then `kira slice commit`)
   - Coordinate with parallel work (merge, rebase, conflict resolution) when multiple items are in play

6. **Implementation loop (single work item)**
   - Get current slice/task: `kira slice current` or `kira slice task current` (omit work-item-id when one work item is in doing)
   - Implement the current task; add or update unit tests for any code changes
   - Run checks from config: `kira config get checks` — run the returned checks before committing
   - Mark task done: `kira slice task current toggle`, then `kira slice commit`
   - Repeat until no open tasks; by the end of the work item, all configured checks (including e2e) must pass

7. **Parallel execution (multiple work items)**
   - When multiple items can be worked in parallel, sequence RALF cycles to minimize conflicts
   - Use branches/worktrees and status folders to track progress
   - Resolve conflicts and integrate incrementally

8. **Intervention points**
   - When conflicts or blockers require human decisions, present options and guide the user
   - When scope or priority of fixes needs validation, ask
   - When parallel streams need coordination, escalate

## Output

Updated work items and code; all configured checks passing. If you edited the Slices section markdown directly, run `kira slice lint` and fix any reported errors.
