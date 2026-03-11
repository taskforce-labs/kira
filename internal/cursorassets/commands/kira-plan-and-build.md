# Plan and Build Work Item

Build the given work item by planning the approach and systematically implementing each slice.

## Steps

1. **Read Work Item**
   - Read the work item and understand scope, acceptance criteria, and current state
   - Use `kira slice current` to see slices and tasks (slices are shown as 1. Name, 2. Name; you can use the number or name in other slice commands)

2. **Implementation Loop**
   For each open task:
   - Get current task: `kira slice task current` (use `--output json` if needed)
   - Implement the task, add/update tests
   - Verify: `kira check -t commit` before committing
   - Learn: If checks fail, analyze root cause, adjust implementation, and document learnings. Only commit when checks pass.
   - Mark done: `kira slice task done current` (prints what was completed). Use `--next` to show the next task and progress summary (e.g. 2/4 slices · 10/20 tasks · 1/3 in current slice); use `--hide-summary` to omit the summary line.
   - Commit: `kira slice commit generate | git commit -F -` or `kira slice commit current` to commit. Use `kira slice task done current --commit` to commit the work-item change in the same step.
   - If editing Slices markdown directly, run `kira slice lint` and fix errors

3. **Complete**
   - When all tasks are done, mark work item complete
   - All checks `kira check -t done` must pass by completion

## Intervention Points

Stop and prompt the user when: implementation approach unclear, tests fail persistently, scope changes, blockers encountered, or architectural decisions needed.

Add questions to a **## Questions** section in the work item. Use the **clarifying-questions-format** skill for structure (checkboxes, options, suggested choice). Record decisions and assumptions in the work item.