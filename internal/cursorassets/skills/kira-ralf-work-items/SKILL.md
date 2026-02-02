---
name: ralf-work-items
description: RALF (Read, Analyze, Learn, Fix) on work items in parallel. Use parallel execution strategies and coordination and conflict resolution. Use when implementing or reviewing multiple work items or slices.
disable-model-invocation: false
---

# RALF on Work Items

Guide the user through RALF (Read, Analyze, Learn, Fix) on work items in parallel.

## When to Use

- Implementing multiple work items or slices in parallel
- Reviewing and fixing issues across work items
- Coordinating parallel streams and resolving conflicts
- Learning from feedback and applying fixes

## Instructions

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
   - Run checks (e.g. `make check`, tests) and commit when ready
   - Coordinate with parallel work (merge, rebase, conflict resolution)

6. **Parallel Execution**
   - When multiple items can be worked in parallel, sequence RALF cycles to minimize conflicts
   - Use branches/worktrees and status folders to track progress
   - Resolve conflicts and integrate incrementally

## Intervention Points

- When conflicts or blockers require human decisions
- When scope or priority of fixes needs validation
- When parallel streams need coordination

At each intervention point, present options and guide the user to make informed decisions.
