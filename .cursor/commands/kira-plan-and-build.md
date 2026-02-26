# Plan and Build Work Item

Build the given work item by planning the approach and systematically implementing each slice.

## Steps

1. **Read Work Item**
   - Read the work item and understand scope, acceptance criteria, and current state
   - Use `kira slice current` to see slices and tasks

2. **Implementation Loop**
   For each open task:
   - Get current task: `kira slice task current` (use `--output json` if needed)
   - Implement the task, add/update tests
   - Verify: `kira check -t commit` before committing
   - Learn: If checks fail, analyze root cause, adjust implementation, and document learnings. Only commit when checks pass.
   - Commit: `kira slice task current toggle`, then `kira slice commit generate | git commit -F -`
   - If editing Slices markdown directly, run `kira slice lint` and fix errors

3. **Complete**
   - When all tasks are done, mark work item complete
   - All checks `kira check -t done` must pass by completion

## Intervention Points

Stop and prompt the user when: implementation approach unclear, tests fail persistently, scope changes, blockers encountered, or architectural decisions needed.

Add questions to a **## Questions** section in the work item. Use the **clarifying-questions-format** skill for structure (checkboxes, options, suggested choice). Record decisions and assumptions in the work item.