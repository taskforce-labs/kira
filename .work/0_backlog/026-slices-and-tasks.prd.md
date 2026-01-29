---
id: 026
title: slices and tasks
status: backlog
kind: prd
assigned:
estimate: 0
created: 2026-01-27
due: 2026-01-27
tags: []
---

# slices and tasks

A command and sections of a work item file for breaking down and tracking tasks organized by slices for better work organization and progress tracking.

## Overview

Slices and tasks provide a structured way to break down work items (especially PRDs and larger work items) into manageable pieces. A **slice** represents a logical grouping or phase of work, while **tasks** are the individual actionable items within each slice. This breakdown enables:

- Better progress tracking at a granular level
- Clear organization of related work
- Support for parallel execution of tasks within slices
- Integration with agent workflows that need structured task lists
- Visibility into what's done and remaining

The `kira slice` command manages slices and tasks within work items, and work items can include a "Slices" section to track this breakdown.

## Context

Large work items (especially PRDs) often need to be broken down into smaller, manageable pieces before implementation. Currently, Kira work items can be elaborated (see PRD 024's `elaborate-work-item`), but there's no structured way to:

1. **Organize tasks into logical groups**: Tasks often belong to phases, features, or workstreams that should be grouped together
2. **Track progress at multiple levels**: Need to see both slice-level and task-level progress
3. **Support parallel work**: Multiple agents or developers can work on different slices or tasks simultaneously
4. **Maintain task state**: Tasks have two states (open or done), toggled independently of the work item status
5. **Reference tasks in commits and PRs**: Need stable identifiers for tasks that can be referenced in git commits and pull requests

**Current Limitations:**
- Work items are atomic - you can't track sub-progress
- No way to organize related tasks together
- No structured format for task breakdowns in work items
- Agents can't easily see what tasks are available to work on
- No integration between task completion and work item status

**Design Philosophy:**
- Slices and tasks are stored **within** work item markdown files (not as separate files)
- Tasks are simple, actionable items (not full work items themselves)
- Slices provide logical grouping without adding unnecessary complexity
- **Order is sequential**: The order of slices in the document is the work sequence; the order of tasks within each slice is the work sequence. The "next" slice is the first (in order) that has open tasks; the "next" task is the first open task (in order) within that slice. No explicit dependency tracking—order implies sequence.
- The structure is human-readable and LLM-friendly (markdown lists with clear formatting)
- Tasks can be referenced by stable identifiers for git commits and PRs

**Direct Edit Workflow (LLM-friendly):**
- LLM skills for creating slices can **bypass the CLI commands for speed** by editing the work item markdown directly (e.g., adding a full `## Slices` section with slices and tasks in one edit) instead of making many tool calls (`kira slice add`, `kira slice task add`, etc.).
- After direct edits, the LLM runs `kira slice lint <work-item-id>` to validate the Slices section. Lint reports errors in a format the LLM can read and fix (location, rule, message, suggested fix).
- This keeps the markdown the source of truth and avoids round-trips for bulk creation.

## Requirements

### Functional Requirements

1. **Slice and Task Structure**
   - A **slice** is a named grouping of related tasks (e.g., "Authentication", "API Endpoints", "Frontend Components")
   - A **task** is an individual actionable item within a slice (e.g., "Implement OIDC login", "Add JWT validation")
   - Tasks have:
     - A unique identifier within the work item (e.g., `T001`, `T002`)
     - A description/title
     - A state: `open` or `done` (two-state toggle only)
     - Optional: notes
   - Slices have:
     - A name/title
     - A list of tasks
     - Optional: description, state (derived from task states: e.g. all done vs has open)

2. **Work Item Sections**
   - Work items can include a "Slices" section (or "## Slices" heading)
   - The section contains markdown-formatted slices and tasks
   - Format is human-readable and LLM-friendly
   - **Sequential order**: Slices appear in work order (first slice first); tasks within a slice appear in work order (first task first). Order implies sequence—no explicit dependency metadata.
   - Tasks are numbered sequentially within the work item (T001, T002, etc.)
   - Slices can be nested (optional, for future extensibility)

3. **`kira slice` Command**

   **Slice Operations:**
   - `kira slice add <work-item-id> <slice-name>` - Add a new slice to a work item
   - `kira slice remove <work-item-id> <slice-name>` - Remove a slice and all its tasks (with confirmation)
   - To rename a slice: remove it and add a new slice with the new name (tasks can be re-added to the new slice).

   **Task Operations (under `task` subcommand):**
   - `kira slice task add <work-item-id> <slice-name> <task-description>` - Add a task to a slice
   - `kira slice task remove <work-item-id> <task-id>` - Remove a task (with confirmation)
   - `kira slice task edit <work-item-id> <task-id> <new-description>` - Update a task's description
   - To move a task to another slice: remove it and add a new task to the target slice.
   - `kira slice task toggle <work-item-id> <task-id>` - Toggle task state (open ↔ done)
   - `kira slice task note <work-item-id> <task-id> <note>` - Add or update task notes

   **Viewing Operations:**
   - `kira slice show <work-item-id>` - Show all slices and tasks with their state (open/done); default view
   - `kira slice show <work-item-id> <slice-name>` - Show a specific slice with its tasks
   - `kira slice show <work-item-id> <task-id>` - Show details for a specific task
   - `kira slice progress <work-item-id>` - Show progress summary (total, done, open, percentage, per-slice)
   - `kira slice next <work-item-id>` - Show the next slice to work on (first slice in order that has open tasks)

   **Validation:**
   - `kira slice lint <work-item-id>` - Validate the Slices section and report errors for the LLM to read and fix (supports `--output json` for machine-readable errors)

   **Workflow Helpers:**
   - `kira slice task next <work-item-id> <slice-name>` - Show the next task in a slice (first open task in order)
   - `kira slice commit <work-item-id> [commit-message]` - Commit slice/task changes:
     - With message: Commit changes with provided message
     - Without message: Generate commit message from task state changes; if none detected, fall back to the name of the next slice (first with open tasks), or the work item title, or "Update slices for &lt;work-item-id&gt;"

   - Commands should:
     - Validate work item exists
     - Preserve existing work item content (only modify Slices section)
     - Auto-commit changes (unless `--no-commit` flag)
     - Generate appropriate task IDs automatically

4. **Task State (Two-State Toggle)**
   - Tasks have only two states: `open` or `done`
   - State is toggled via `kira slice task toggle <work-item-id> <task-id>` (no status argument)
   - State is displayed in slice listings
   - Work item status can optionally be derived from task state (all tasks done → work item can move to review)

5. **Slice and Task Format in Work Items**
   - Slices section uses markdown formatting:
     ```markdown
     ## Slices

     ### Authentication
     - [ ] T001: Implement OIDC login flow
     - [ ] T002: Add JWT token validation
     - [x] T003: Configure OIDC provider settings

     ### API Endpoints
     - [ ] T004: Create user registration endpoint
     - [ ] T005: Add password reset endpoint
     ```
   - Task checkboxes (`[ ]` or `[x]`) indicate state: `[ ]` = open, `[x]` = done
   - Task IDs (T001, T002) are stable identifiers
   - Optional explicit format: `[open]` or `[done]` for clarity (same two states)

6. **Task ID Generation**
   - Task IDs are sequential within a work item: T001, T002, T003, etc.
   - IDs are never reused (even if tasks are removed)
   - IDs are stable across edits (don't change when tasks are reordered)
   - IDs can be referenced in git commits: `git commit -m "Complete T001: Implement OIDC login"`

7. **Integration with Work Item Status**
   - Optionally, work item status can be auto-updated based on task completion
   - When all tasks are done, suggest moving work item to "review" status
   - Configuration option: `slices.auto_update_status: true/false` in `kira.yml`

8. **Progress Tracking**
   - `kira slice progress <work-item-id>` - Show progress summary:
     - Total tasks, done tasks, open tasks
     - Progress percentage
     - Per-slice breakdown
   - Progress can be displayed in work item summaries

9. **Task Metadata**
   - Tasks can have optional metadata (slices and tasks are not assignable):
     - Notes/description
   - Order implies sequence; no explicit dependency metadata. Metadata stored in task description or as YAML frontmatter within task.

10. **Workflow Helpers**
    - `kira slice next <work-item-id>` - Identifies and displays the next slice to work on:
      - First slice in document order that has open tasks
      - Shows slice name, task count, and first few open tasks
      - Useful for agents/developers to know where to focus next
    - `kira slice task next <work-item-id> <slice-name>` - Identifies and displays the next task in a slice:
      - First open task in list order within the specified slice
      - Order is sequential; no explicit dependency tracking
      - Shows task ID, description, and any metadata
      - Useful for agents/developers to know which specific task to work on
    - `kira slice commit <work-item-id> [commit-message]` - Commits slice/task changes:
      - With message: Commits changes with the provided message
      - Without message: Generates a commit message based on task state changes:
        - Analyzes which tasks were completed (state toggled to "done")
        - Analyzes which tasks were reopened (state toggled to "open")
        - Analyzes which tasks were added
        - Generates message like: "Complete T001, T002: Implement OIDC login and JWT validation"
        - Or: "Reopen T003: Configure OIDC provider settings"
        - Or: "Add tasks T004-T006: User registration endpoints"
        - Includes task IDs and descriptions in the message
        - **Fallback when no task changes detected:** Use the name of the next slice (first slice in order with open tasks), or the work item title, or "Update slices for &lt;work-item-id&gt;" so the commit always has a meaningful message
      - Only commits changes to `.work/` directory (slice/task updates)
      - Can be used by LLMs to provide context-aware commit messages

11. **`kira slice lint` and Direct Edit Workflow**
    - `kira slice lint <work-item-id>` - Validates the Slices section and reports errors:
      - **Purpose**: Allow LLM skills to edit the work item markdown directly (add/change slices and tasks in one or few edits) instead of many CLI calls, then validate with lint.
      - **Checks**: Same rules as Validation (section 12): unique task IDs, valid task states (open/done), no duplicate slice names.
      - **Output**: Errors in a format the LLM can read and fix:
        - Human-readable (default): one error per line with file, line (if known), rule, message, and optional suggested fix.
        - Machine-readable: `--output json` emits a JSON array of `{ "location": "file:line or slice/task", "rule": "rule_id", "message": "...", "suggestion": "..." }`.
      - **Exit code**: 0 if valid, non-zero if any errors (so scripts/LLMs can detect failure).
      - **No auto-fix**: Lint only reports; the LLM (or user) applies fixes.
    - **Direct edit workflow for LLMs**:
      - LLM adds or edits the `## Slices` section in the work item file (e.g., full section with multiple slices and tasks in one edit).
      - LLM runs `kira slice lint <work-item-id>` (optionally `--output json`).
      - If errors: LLM reads output and fixes the markdown, then re-runs lint until clean.
      - Skills/docs should describe this workflow so agents prefer direct edit + lint for bulk slice/task creation.

12. **Validation**
    - Validate task IDs are unique within work item
    - Validate task state is open or done only
    - Validate slice names don't conflict
    - Order is sequential; no dependency validation (order implies sequence)

### Technical Requirements

1. **Command Implementation**
   - Add `kira slice` command with subcommands:
     - **Slice operations**: `add`, `remove` (rename = remove + add new slice)
     - **Task operations** (nested under `task`): `task add`, `task remove`, `task edit`, `task toggle`, `task note`, `task next` (move = remove + add to target slice)
     - **Viewing operations**: `show`, `progress`, `next`
     - **Validation**: `lint` (with `--output json` for machine-readable errors)
     - **Workflow helpers**: `commit`
   - Use Cobra for command structure with nested subcommands for task operations
   - Parse work item markdown to find/update Slices section
   - Preserve all other work item content
   - Commit message generation: Analyze task state changes to generate meaningful commit messages
   - Lint: Same validation rules as other slice validation; output format suitable for LLM consumption (location, rule, message, suggestion)

2. **Markdown Parsing and Generation**
   - Parse existing Slices section from work item markdown
   - Generate/update Slices section with proper formatting
   - Handle missing Slices section (create it)
   - Preserve formatting and comments outside Slices section

3. **Task ID Management**
   - Track highest task ID used in work item
   - Generate next sequential ID
   - Store ID mapping to prevent conflicts

4. **Task State (Two-State Toggle)**
   - Parse task state from markdown (checkbox: `[ ]` = open, `[x]` = done)
   - Toggle updates checkbox in markdown
   - Support both checkbox format and optional explicit format (`[open]` / `[done]`)

5. **Configuration**
   - Add `slices` section to `kira.yml`:
     ```yaml
     slices:
       auto_update_status: false  # Auto-update work item status when all tasks done
       task_id_format: "T%03d"   # Format for task IDs (default: T001, T002, etc.)
       default_state: "open"     # Default state for new tasks (only open or done)
     ```

6. **File Format**
   - Slices section is inserted after "## Requirements" or "## Acceptance Criteria" (or at end if neither exists)
   - Section uses standard markdown heading: `## Slices`
   - Tasks use markdown list items with checkboxes (`[ ]` open, `[x]` done)

7. **Error Handling**
   - Handle missing work item
   - Handle invalid task IDs
   - Handle duplicate slice names (warn or error)
   - Handle malformed Slices section (try to repair or error)

8. **Integration Points**
   - `kira start` can optionally show slice/task breakdown
   - `kira review` can check if all tasks are done
   - Work item templates can include empty Slices section
   - Agents can read slices/tasks to understand work breakdown

## Acceptance Criteria

1. ✅ `kira slice add <work-item-id> <slice-name>` successfully adds a new slice to a work item
2. ✅ `kira slice remove <work-item-id> <slice-name>` removes a slice and all its tasks with confirmation
3. ✅ Rename slice = remove slice + add new slice with new name; move task = remove task + add task to target slice
4. ✅ `kira slice task add <work-item-id> <slice-name> <task-description>` adds a task with auto-generated ID (T001, T002, etc.)
5. ✅ `kira slice task edit <work-item-id> <task-id> <new-description>` updates a task's description
6. ✅ `kira slice task remove <work-item-id> <task-id>` removes task with confirmation
7. ✅ `kira slice task toggle <work-item-id> <task-id>` toggles task state (open ↔ done) in work item markdown
8. ✅ `kira slice task note <work-item-id> <task-id> <note>` adds or updates task notes
9. ✅ `kira slice show <work-item-id>` shows all slices and tasks with their state (open/done); single viewing command
10. ✅ `kira slice show <work-item-id> <slice-name>` shows a specific slice with its tasks
11. ✅ `kira slice show <work-item-id> <task-id>` shows details for a specific task
12. ✅ `kira slice progress <work-item-id>` shows progress summary (total, done, open, percentage, per-slice)
13. ✅ Order of slices and tasks is sequential (order implies work sequence); no explicit dependency tracking
14. ✅ Task IDs are sequential and never reused (T001, T002, T003...)
15. ✅ Slices section is properly formatted markdown and human-readable
16. ✅ Commands preserve all other work item content (only modify Slices section)
17. ✅ Task state is validated (open or done only)
18. ✅ Work item markdown includes properly formatted Slices section
19. ✅ Task IDs can be referenced in git commits and PRs
20. ✅ Configuration in `kira.yml` controls task ID format and default state
21. ✅ Commands auto-commit changes (unless `--no-commit` flag)
22. ✅ Progress tracking shows per-slice and overall progress
23. ✅ Integration with `kira start` optionally shows slice/task breakdown
24. ✅ Integration with `kira review` checks if all tasks are done (if configured)
25. ✅ Work item templates can include empty Slices section
26. ✅ Task metadata (notes) is properly stored and displayed; slices and tasks are not assignable; order is sequential
27. ✅ `kira slice next <work-item-id>` identifies the next slice (first in order with open tasks)
28. ✅ `kira slice task next <work-item-id> <slice-name>` identifies the next task (first open task in order within slice)
29. ✅ `kira slice commit <work-item-id> <commit-message>` commits changes with the provided message
30. ✅ `kira slice commit <work-item-id>` generates a commit message based on task state changes (completed, reopened, added tasks); when no task changes are detected, falls back to next slice name, work item title, or "Update slices for &lt;work-item-id&gt;"
31. ✅ Commit message generation includes task IDs and descriptions in a clear format
32. ✅ Commit only affects `.work/` directory changes (slice/task updates)
33. ✅ `kira slice lint <work-item-id>` validates the Slices section and reports errors (unique task IDs, valid state open/done, no duplicate slice names)
34. ✅ Lint output is readable by LLMs (location, rule, message, optional suggestion) and supports `--output json` for machine-readable errors
35. ✅ Lint exits with 0 when valid and non-zero when there are errors
36. ✅ LLM skills can create/update slices by editing the work item markdown directly and then run `kira slice lint` to validate; skills/docs describe this direct-edit + lint workflow for bulk creation

## Implementation Notes

### Architecture

```
kira slice <subcommand> <work-item-id> [args]
  ├── Work Item Parser
  │   ├── Read work item markdown
  │   ├── Parse Slices section
  │   ├── Extract slices and tasks
  │   └── Parse task IDs and state (open/done)
  ├── Slice Manager
  │   ├── Add/remove slices
  │   ├── Add/remove tasks
  │   ├── Toggle task state (open ↔ done)
  │   └── Generate task IDs
  ├── Markdown Generator
  │   ├── Format Slices section
  │   ├── Generate task list items
  │   ├── Update task state (checkbox)
  │   └── Preserve other content
  ├── Progress Calculator
  │   ├── Count tasks by state (open/done)
  │   ├── Calculate percentages
  │   └── Generate progress summary
  ├── Work Item Writer
  │   ├── Update Slices section
  │   ├── Preserve other sections
  │   └── Write updated markdown
  └── Lint Validator
      ├── Parse Slices section
      ├── Run all validation rules
      ├── Collect errors with location, rule, message, suggestion
      └── Output human-readable or JSON (--output json)
```

### Data Structures

```go
type Slice struct {
    Name        string
    Description string  // Optional
    Tasks       []Task
}

type Task struct {
    ID           string     // T001, T002, etc.
    Description  string
    Done         bool       // true = done, false = open (two-state only)
    Notes        string     // Optional
    // Order is sequential; no Dependencies field—order implies sequence
}
```

### Markdown Format

**Simple Format (Checkbox-based):**
```markdown
## Slices

### Authentication
- [ ] T001: Implement OIDC login flow
- [x] T002: Add JWT token validation
- [ ] T003: Configure OIDC provider settings

### API Endpoints
- [ ] T004: Create user registration endpoint
- [ ] T005: Add password reset endpoint
```

**With metadata (notes/deps in list or following lines):**
```markdown
## Slices

### Authentication
- [ ] T001: Implement OIDC login flow
- [x] T002: Add JWT token validation
  - Notes: Completed OIDC configuration for Auth0
- [ ] T003: Configure OIDC provider settings

### API Endpoints
- [ ] T004: Create user registration endpoint
- [ ] T005: Add password reset endpoint
  - Notes: Blocked on T002 completion
```

### Command Examples

```bash
# Slice management (rename = remove + add new slice)
kira slice add 001 "Authentication"
kira slice remove 001 "Old Slice Name"

# Task management (move = remove + add to target slice)
kira slice task add 001 "Authentication" "Implement OIDC login flow"
kira slice task add 001 "Authentication" "Add JWT token validation"
kira slice task edit 001 T001 "Implement OIDC login flow with PKCE"
kira slice task remove 001 T003

# Task state and metadata (slices and tasks are not assignable)
kira slice task toggle 001 T001   # open → done
kira slice task toggle 001 T002  # done → open (reopen)
kira slice task note 001 T002 "Need to coordinate with backend team"

# Viewing (show = single command; no list; progress kept)
kira slice show 001
kira slice show 001 "Authentication"
kira slice show 001 T002
kira slice progress 001

# Validation (e.g. after direct markdown edit by LLM)
kira slice lint 001
kira slice lint 001 --output json

# Workflow helpers
kira slice next 001
# Output: Next slice: Authentication (3 open tasks)
#   - T001: Implement OIDC login flow
#   - T004: Add refresh token support
#   - T005: Implement logout flow

kira slice task next 001 "Authentication"
# Output: Next task: T001 - Implement OIDC login flow

kira slice commit 001 "Complete T001, T002: Implement OIDC login and JWT validation"
# Commits changes with provided message

kira slice commit 001
# Generates and commits with message like:
# "Complete T001, T002: Implement OIDC login and JWT validation"
# If no task changes detected, falls back to next slice name (e.g. "Authentication"), work item title, or "Update slices for 001"


### Parsing Strategy

1. **Find Slices Section**: Search for `## Slices` heading in work item markdown
2. **Parse Slices**: Extract slice headings (`### Slice Name`)
3. **Parse Tasks**: Extract task list items under each slice
4. **Extract Task Info**: Parse task ID, description, state (checkbox or [open]/[done]) from list item
5. **Build Data Structure**: Create Slice and Task objects
6. **Generate IDs**: Track highest ID, generate next sequential ID

### Markdown Generation Strategy

1. **Preserve Content**: Keep all content before and after Slices section
2. **Generate Section**: Create properly formatted Slices section
3. **Format Tasks**: Generate markdown list items with checkboxes (`[ ]` open, `[x]` done)
4. **Maintain Order**: Preserve slice and task order (or sort if configured)

### Workflow Helper Implementation

**`kira slice next` Implementation:**
1. Parse all slices and tasks from work item in document order
2. Find the first slice (in order) that has at least one open task
3. Display that slice name, task count, and first few open tasks
4. Order is sequential—no dependency checks

**`kira slice task next` Implementation:**
1. Parse tasks from specified slice in list order
2. Find the first open task in order
3. Display task ID, description, and relevant metadata
4. If no open tasks, indicate slice is complete
5. Order is sequential—no explicit dependency tracking

**`kira slice commit` Implementation:**
1. **With message provided:**
   - Commit changes to `.work/` directory with provided message
   - Only commit slice/task related changes (work item file updates)

2. **Without message (generate):**
   - Compare current state with last committed state (or git diff)
   - Analyze task state changes:
     - Tasks toggled to done → "Complete T001, T002: ..."
     - Tasks toggled to open (reopened) → "Reopen T003: ..."
     - Tasks that were added → "Add tasks T004-T006: ..."
   - Generate commit message format:
     - Single task: "Complete T001: Task description"
     - Multiple tasks: "Complete T001, T002: Task1 and Task2"
     - Mixed changes: "Complete T001, T002: Task1 and Task2\nReopen T003: Task3"
   - Include task IDs and descriptions
   - Keep message concise but informative
   - **Fallback when no task changes:** If no task state changes are detected (or generated message would be empty), use in order: (1) the name of the next slice (first slice in order with open tasks), e.g. "Authentication"; (2) the work item title from the file; (3) "Update slices for &lt;work-item-id&gt;". So the commit always has a meaningful message.
   - Commit with generated or fallback message

3. **Message generation algorithm:**
   ```go
   func GenerateCommitMessage(changes TaskChanges, nextSliceName string, workItemTitle string, workItemID string) string {
       var parts []string

       if len(changes.Completed) > 0 {
           parts = append(parts, formatTasks("Complete", changes.Completed))
       }
       if len(changes.Reopened) > 0 {
           parts = append(parts, formatTasks("Reopen", changes.Reopened))
       }
       if len(changes.Added) > 0 {
           parts = append(parts, formatTasks("Add tasks", changes.Added))
       }

       if len(parts) > 0 {
           return strings.Join(parts, "\n")
       }
       // Fallback: slice name, work item title, or generic
       if nextSliceName != "" {
           return nextSliceName
       }
       if workItemTitle != "" {
           return workItemTitle
       }
       return "Update slices for " + workItemID
   }
   ```

### Lint and Direct Edit Workflow

**`kira slice lint` Implementation:**
1. Parse work item and locate Slices section (if missing, report single error: "Slices section missing").
2. Parse slices and tasks using same parser as other slice commands.
3. Run validation rules (same as Validation section):
   - Unique task IDs
   - Valid task state (open or done only)
   - No duplicate slice names
   - Order is sequential; no dependency validation
4. For each error, record: `location` (file path, optional line, slice name, task ID), `rule` (e.g. `duplicate-task-id`, `invalid-state`), `message` (human-readable), `suggestion` (optional fix).
5. Output:
   - **Default (human-readable)**: One line per error, e.g. `001-user-auth.prd.md:42 [duplicate-task-id] Task ID T001 appears more than once. Suggestion: Use unique IDs T001, T002, ...`
   - **`--output json`**: JSON array to stdout, e.g. `[{"location":".work/1_todo/001-user-auth.prd.md:42","rule":"duplicate-task-id","message":"Task ID T001 appears more than once","suggestion":"Use unique IDs T001, T002, ..."}]`
6. Exit code: 0 if no errors, 1 (or non-zero) if any errors. No auto-fix; caller (LLM or user) applies fixes.

**Direct Edit Workflow for LLM Skills:**
- Skills that create or update slices should prefer **direct markdown edit** over many CLI calls when adding multiple slices/tasks:
  1. Edit the work item file: add or replace the `## Slices` section with the full slice/task structure (e.g. multiple `### Slice Name` and task list items).
  2. Run `kira slice lint <work-item-id>` (or `kira slice lint <work-item-id> --output json` for structured errors).
  3. If lint reports errors: read location, rule, message, suggestion; apply fixes to the markdown; re-run lint until clean.
- Document this workflow in slice-creation skills and in docs so agents use direct edit + lint for bulk creation, and reserve CLI commands for single-item updates or scripting.

### Integration with Existing Commands

**`kira start` Integration:**
- Optionally display slice/task breakdown when starting work
- Show available tasks that can be worked on
- Suggest which slice to start with

**`kira review` Integration:**
- Check if all tasks are done before allowing review
- Warn if tasks are incomplete
- Optionally auto-update work item status if all tasks done

**Work Item Templates:**
- Include empty Slices section in PRD templates
- Provide example format in comments

### Error Handling

- **Missing Work Item**: Clear error message with work item ID
- **Invalid Task ID**: List available task IDs
- **Invalid Slice Name**: List available slice names
- **Duplicate Slice Name**: Warn or error (configurable)
- **Malformed Slices Section**: Attempt to repair or provide clear error
- **Missing Slice**: Error when adding task to non-existent slice, or when operating on non-existent slice
- **Invalid State**: Only open and done are valid; reject any other state

### Testing Strategy

1. **Unit Tests**:
   - Markdown parsing (extract slices/tasks)
   - Markdown generation (format slices/tasks)
   - Task ID generation
   - State validation (open/done)
   - Lint validation rules (each rule, human and JSON output)

2. **Integration Tests**:
   - Full command workflows
   - Work item preservation
   - Git commit integration

3. **E2E Tests**:
   - Create work item → add slices → add tasks → toggle task state → verify
   - Direct edit: add Slices section via file edit → `kira slice lint` → fix reported errors → lint passes

## Release Notes

### v1.0.0 (Initial Release)

- Add `kira slice` command with comprehensive subcommands for managing slices and tasks:
  - Slice operations: `add`, `remove` (rename = remove + add new slice)
  - Task operations (nested): `task add`, `task remove`, `task edit`, `task toggle`, `task note`, `task next` (move = remove + add to target slice)
  - Viewing operations: `show` (single command for list/detail), `progress`, `next`
  - Validation: `lint` (human-readable and `--output json` for LLM/script consumption)
  - Workflow helpers: `commit` (with automatic commit message generation)
- Support for organizing tasks into logical slices (groupings)
- Automatic task ID generation (T001, T002, etc.) with stable identifiers
- Task state: two-state toggle only (open or done)
- Task metadata (notes); slices and tasks are not assignable; order of slices and tasks is sequential
- Progress tracking with per-slice and overall progress
- Markdown-formatted Slices section in work items (human-readable and LLM-friendly)
- Integration with `kira start` to show task breakdown
- Integration with `kira review` to check task completion
- Configuration via `kira.yml` for task ID format, default state, and auto-update behavior
- Commands preserve all other work item content (only modify Slices section)
- Workflow helpers: `next` commands to identify next slice/task to work on, `commit` with automatic commit message generation based on task changes
- `kira slice lint`: validates Slices section and reports errors (location, rule, message, suggestion) for LLMs to read and fix; supports `--output json`
- Direct-edit workflow: LLM skills can add/update slices by editing the work item markdown directly (one or few edits) and then run `kira slice lint` to validate, avoiding many CLI calls for bulk creation
- Comprehensive error handling and validation

