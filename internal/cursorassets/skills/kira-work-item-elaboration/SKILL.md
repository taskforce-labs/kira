---
name: work-item-elaboration
description: Elaborate work items into small testable phases. Define acceptance criteria and phase sequencing and dependencies. Use when refining a PRD or task before implementation.
disable-model-invocation: false
---

# Work Item Elaboration

Guide the user through elaborating work items into testable phases.

## When to Use

- Refining a PRD or task before implementation
- Defining acceptance criteria
- Breaking work into phases or slices
- Sequencing phases and dependencies

## Instructions

1. **Read Project Configuration**
   - Check `kira.yaml` for work folder and templates
   - Use `.work/` structure and `kira slice` if the project uses slices

2. **Elaborate Work Item**
   - Read the work item (e.g. PRD) and identify phases or slices
   - Define clear acceptance criteria per phase
   - Ensure each phase is testable and committable

3. **Phase Sequencing**
   - Order phases by dependency and risk
   - Document dependencies between phases
   - Use `kira slice` to add slices and tasks if the project uses them

4. **Acceptance Criteria**
   - Write concrete, testable criteria for each phase/slice
   - Align with Definition of Done
   - Store in the work item (e.g. in Slices section or Acceptance Criteria)

5. **Create Artifacts**
   - Updated work item with phases/slices and criteria
   - Use project templates and locations

## Intervention Points

- When acceptance criteria are ambiguous
- When phase order or dependencies need validation
- When scope of a phase needs adjustment

At each intervention point, present options and guide the user to make informed decisions.
