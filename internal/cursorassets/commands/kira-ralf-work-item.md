# RALF Work Item

## Overview
RALF (Read, Analyze, Learn, Fix) on work items in parallel: read state, analyze gaps, learn from feedback, fix and verify.

## Steps

1. **Read**
   - Read the work item(s) and current state (slices, tasks, acceptance criteria)
   - Understand what is done and what is open

2. **Analyze**
   - Analyze gaps between current state and acceptance criteria
   - Identify failures, regressions, or missing behavior
   - Prioritize what to fix or implement next

3. **Learn**
   - Use test results, lint, and feedback to learn what is broken or missing
   - Incorporate feedback into the next steps
   - Update work item or tasks (e.g. `kira slice task toggle`, notes) as needed

4. **Fix**
   - Apply fixes or implement missing behavior
   - Run checks (e.g. `make check`, tests) and commit when ready
   - Coordinate with parallel work (merge, rebase, conflict resolution)

## Output

Updated work items and code; tests and checks passing. Use the ralf-work-items skill for parallel execution strategies and coordination.
