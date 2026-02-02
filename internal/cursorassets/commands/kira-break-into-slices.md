# Break Into Slices

## Overview
Break work items into testable slices that can be committed to git, with clear tasks and acceptance criteria.

## Steps

1. **Read Work Item**
   - Read the work item (e.g. PRD) and acceptance criteria
   - Identify logical slices (independently testable/committable units)

2. **Define Slices**
   - Name each slice and describe scope
   - Ensure each slice is testable and committable
   - Use `kira slice add` to add slices to the work item

3. **Define Tasks**
   - Add tasks per slice with `kira slice task add`
   - Order tasks by dependency
   - Ensure tasks are actionable and testable

4. **Update Work Item**
   - Ensure Slices section in the work item reflects slices and tasks
   - Run `kira slice lint` to validate structure

## Output

Work item with Slices section populated; slices and tasks added via kira slice. Use the work-item-elaboration skill for detailed guidance.
