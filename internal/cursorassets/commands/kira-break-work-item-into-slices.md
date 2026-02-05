# Break Into Slices

## Overview
Break a work item into testable slices that can be committed to git, with clear tasks and acceptance criteria.

## Steps

1. **Read Work Item**
   - Read the work item (e.g. PRD) and acceptance criteria
   - Identify logical slices (independently testable/committable units)

2. **Define Slices (preferred: direct markdown)**
   - Add a `## Slices` section to the work item by editing the markdown directly (preferred over using the CLI).
   - Use this format:

```
## Slices
### <Slice name 1>
Commit: <Summary of the slice and tasks>
- [ ] T001: task description
- [ ] T002: task description

### <Slice name 2>
Commit: <Summary of the slice and tasks>
- [ ] T003: task description
- [ ] T004: task description
```

   - Name each slice and describe scope; ensure each slice is testable and committable.
   - Add tasks per slice; order by dependency; keep tasks actionable and testable.

3. **Validate**
   - Run `kira slice lint [work-item-id]` to check the format is correct. Fix any reported errors and re-run until clean.

## Output

Work item with Slices section populated in the preferred markdown format. Use the work-item-elaboration skill for detailed guidance.
