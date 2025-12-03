---
id: 007
title: Commit flag for move command
status: review
kind: prd
assigned:
estimate: 0
created: 2025-11-30
due: 2025-11-30
tags: []
---

# Commit flag for move command

Add a `--commit` and `-c` flag to the `kira move` command to allow users to automatically commit work item moves to git, eliminating the need for manual commits and preventing accidental inclusion in subsequent commits.

## Context

Currently, when a user moves a work item to a different status using `kira move <work-item-id> [target-status]`, the file is moved and the status field is updated, but no git commit is created. Users must manually stage and commit these changes, which can be:

- **Inconvenient**: Requires an extra manual step after each move
- **Error-prone**: Moves may be accidentally included in unrelated commits
- **Inconsistent**: Some moves get committed immediately, others are forgotten

The `kira save` command demonstrates how git commits are handled in this codebase, including proper message sanitization, staging, and error handling. The move command should follow similar patterns when committing.

## Requirements

### Command Behavior

Add a `--commit` (or `-c`) boolean flag to the `kira move` command:

```bash
kira move <work-item-id> [target-status] [--commit|-c]
```

**When `--commit` flag is used**:
- After successfully moving the work item file and updating its status field, the command should:
  1. Stage only the moved work item file (both old and new locations)
  2. Create a git commit with a formatted commit message
  3. Display success message indicating the move and commit were completed and committed successfully

**When `--commit` flag is NOT used**:
- The command behaves as it currently does (move file, update status, no git operations)

### Commit Message Format

The commit message should follow git best practices with a concise subject line (50-72 characters) and optional body for additional details. The format should be configurable in `kira.yml` with sensible defaults.

**Default format** (if not configured):
```
Move <work-item-type> <work-item-id> to <target-status>

<title> (<current-status> -> <target-status>)
```

**Configuration in `kira.yml`**:
```yaml
commit:
  default_message: Update work items
  move_subject_template: "Move {type} {id} to {target_status}"
  move_body_template: "{title} ({current_status} -> {target_status})"
```

**Template variables**:
- `{type}` - Work item type (prd, issue, spike, task)
- `{id}` - Work item ID
- `{title}` - Work item title
- `{current_status}` - Status before move
- `{target_status}` - Status after move

**Examples** (using default format):
```
Move prd 007 to doing

Commit flag for move command (todo -> doing)
```

```
Move issue 001 to review

Add user authentication (doing -> review)
```

**Format requirements**:
- Subject line: Imperative mood, concise (ideally ≤72 chars)
- Body: Separated by blank line, includes title and status transition
- Use ASCII arrow `->` instead of Unicode `→` for better compatibility
- If title is very long, truncate in subject line and include full title in body
- Templates support the variables listed above
- If templates are not configured, use default format

**Message requirements**:
- Must include work item type and ID
- Must include work item title (extracted from front matter)
- Must include current status (before move)
- Must include target status (after move)
- Must be sanitized using the existing `sanitizeCommitMessage()` function
- Must follow the same security validation as `kira save` command

### Git Operations

**Staging behavior**:
- Only stage the specific work item file that was moved (not all `.work/` changes)
- Stage both the deletion of the old file location and the addition of the new file location
- Use `git add` with specific file paths, not `git add .work/`

**Commit behavior**:
- Before committing, check if there are already staged changes in the repository
- If other files are already staged:
  - **Fail with error**: `Error: other files are already staged. Commit or unstage them before using --commit flag, or use 'kira save' to commit all changes in .work/ directory together`
  - This prevents accidentally including unrelated changes in the move commit
  - **Important**: The file move still succeeds - only the git commit fails
  - User can then use `kira save` to commit all changes in .work/ directory together, or unstage and retry with `--commit`
- If no other files are staged (or only the moved file is staged):
  - Proceed with commit using `git commit -m "<message>"` with the sanitized commit message
- Follow the same timeout and context patterns as `save.go` (10 second timeout)
- Use `exec.CommandContext` with proper error handling

## Acceptance Criteria

- [ ] `kira move <id> <status> --commit` successfully moves the work item and creates a git commit
- [ ] `kira move <id> <status>` (without flag) moves the work item without creating a git commit
- [ ] Commit message includes work item type, ID, title, current status, and target status
- [ ] Commit message is properly sanitized and validated
- [ ] Only the moved work item file is staged (not all `.work/` changes)
- [ ] Command fails gracefully if git is not available or not a git repository
- [ ] Command fails gracefully if git commit fails (e.g., no changes to commit, git config issues)
- [ ] Command fails with clear error if other files are already staged
- [ ] Success message indicates both move and commit completion when flag is used
- [ ] Error messages are clear and actionable

## Implementation Notes

### Flag Implementation

Add the flag to `moveCmd` in `internal/commands/move.go`:
```go
var moveCmd = &cobra.Command{
    Use:   "move <work-item-id> [target-status]",
    Short: "Move a work item to a different status folder",
    Long:  `Moves the work item to the target status folder. Will display options if target status not provided.`,
    Args:  cobra.RangeArgs(1, 2),
    RunE: func(cmd *cobra.Command, args []string) error {
        // ... existing code ...
        commitFlag, _ := cmd.Flags().GetBool("commit")
        return moveWorkItem(cfg, workItemID, targetStatus, commitFlag)
    },
}

func init() {
    moveCmd.Flags().BoolP("commit", "c", false, "Commit the move to git")
}
```

### Work Item Metadata Extraction

Create a helper function to extract work item metadata (type, ID, title, current status) from the file:
- Read the work item file using `safeReadFile()`
- Parse front matter to extract `id`, `title`, and `status` fields
- Handle missing or malformed front matter gracefully

### Git Operations

**Reuse existing patterns from `save.go`**:
- Use `sanitizeCommitMessage()` for message validation
- Use `exec.CommandContext` with 10-second timeout
- Follow the same error handling patterns
- Reuse the pattern from `checkExternalChanges()` for git command execution (but use `git diff --cached` instead of `git status`)

**New functions needed**:
- `checkStagedChanges(excludePaths []string) (bool, error)`:
  - Check if there are any staged changes in git, excluding specified paths
  - Use `git diff --cached --name-only` to list only staged files (more precise than `git status --porcelain`)
  - Filter out excluded paths (the moved file paths)
  - Return `true` if other files are staged, `false` if only excluded files (or nothing) is staged
  - Return error if git command fails
  - Note: Similar pattern to `checkExternalChanges()` but checks only staged changes and excludes specific paths instead of checking for files outside `.work/`

- `commitMove(oldPath, newPath, subject, body string) error`:
  - Check for staged changes excluding the moved file paths using `checkStagedChanges()`
  - If other files are staged, return error: `Error: other files are already staged. Commit or unstage them before using --commit flag, or use 'kira save' to commit all changes together`
  - Stage the old file deletion: `git add <old-path>`
  - Stage the new file addition: `git add <new-path>`
  - Sanitize subject and body separately using `sanitizeCommitMessage()`
  - Commit with multi-line message: `git commit -m "<sanitized-subject>" -m "<sanitized-body>"`
  - If body is empty, use single `-m` flag: `git commit -m "<sanitized-subject>"`

### Error Handling

**Git-related errors**:
- If git is not available: Return error: `Error: git is not available. Install git to use --commit flag`
- If not a git repository: Return error: `Error: not a git repository. Initialize git to use --commit flag`
- If other files are already staged: Return error: `Error: other files are already staged. Commit or unstage them before using --commit flag, or use 'kira save' to commit all changes together`
- If commit fails: Return error with git output: `Error: failed to commit: <git-error>`
- If staging fails: Return error: `Error: failed to stage changes: <error>`

**Work item errors** (existing):
- Work item not found
- Invalid target status
- File move failures
- Status update failures

**Important**: If git operations fail, the file move should still succeed (don't rollback the move). The error should be reported but the move operation should be considered successful.

### Current Status Detection

The current status must be determined before the move:
- Extract from the work item file's front matter before moving
- Use this value in the commit message
- If status cannot be determined, use "unknown" as fallback

### Commit Message Construction

**Template-based approach**:
- Load `move_subject_template` and `move_body_template` from config (with defaults if not set)
- Replace template variables: `{type}`, `{id}`, `{title}`, `{current_status}`, `{target_status}`
- If body template is empty or not configured, use single-line format (subject only)
- Sanitize subject and body separately before passing to git

**Default templates** (if not configured):
- Subject: `"Move {type} {id} to {target_status}"`
- Body: `"{title} ({current_status} -> {target_status})"`

**Implementation**:
- Create helper function `buildCommitMessage(cfg *config.Config, type, id, title, currentStatus, targetStatus string) (subject, body string, err error)`
- Load templates from `cfg.Commit.MoveSubjectTemplate` and `cfg.Commit.MoveBodyTemplate`
- Use `strings.ReplaceAll()` or template package to replace variables
- Handle missing template variables gracefully (use empty string or "unknown")
- If subject + body would exceed reasonable length, truncate title in subject and include full title in body
- Sanitize subject and body separately using `sanitizeCommitMessage()` before passing to git

## Technical Notes

### Testing Requirements

**Unit tests should cover**:
- Move command with `--commit` flag (success case)
- Move command without `--commit` flag (no git operations)
- Commit message format and content (default templates)
- Commit message format with custom templates from config
- Commit message sanitization
- Template variable replacement
- Work item metadata extraction (ID, title, status)
- Git staging (both old and new file paths)
- Error handling: git not available
- Error handling: not a git repository
- Error handling: git commit failure
- Error handling: git staging failure
- Error handling: other files already staged (should fail)
- Checking for staged changes (excluding moved file paths)
- Current status extraction from front matter
- Handling missing front matter fields

**End-to-end tests should cover**:
- Full workflow: `kira move <id> <status> --commit` → verify git commit created
- Full workflow: `kira move <id> <status>` → verify no git commit
- Verify commit message format matches specification (default format)
- Verify custom commit message templates from config are used when configured
- Verify only moved file is staged (not other `.work/` changes)
- Verify commit includes both file deletion and addition
- Error case: Move without git initialized
- Error case: Move with git unavailable
- Error case: Move with `--commit` when other files are already staged (should fail with clear error)
- Success case: Move with `--commit` when no other files are staged (should succeed)
- Multiple moves: Verify each move creates separate commits when using `--commit`

### Code Organization

**New functions to create**:
- `extractWorkItemMetadata(filePath string) (type, id, title, status string, err error)` - Extract metadata from front matter (including type)
- `buildCommitMessage(cfg *config.Config, type, id, title, currentStatus, targetStatus string) (subject, body string, err error)` - Build commit message from templates
- `checkStagedChanges(excludePaths []string) (bool, error)` - Check if other files (excluding specified paths) are staged
- `commitMove(oldPath, newPath, subject, body string) error` - Stage and commit the move with subject and body

**Functions to modify**:
- `moveWorkItem()` - Add `commitFlag bool` parameter and call commit logic when flag is set
- `moveCmd` - Add flag definition and pass flag value to `moveWorkItem()`

**Config struct to update**:
- `CommitConfig` in `internal/config/config.go` - Add `MoveSubjectTemplate` and `MoveBodyTemplate` fields
- `DefaultConfig` - Add default values for move templates

**Functions to reuse**:
- `sanitizeCommitMessage()` from `save.go`
- `safeReadFile()` from `utils.go`

### Security Considerations

- **Commit message sanitization**: Reuse `sanitizeCommitMessage()` to prevent injection attacks
- **Path validation**: Ensure file paths are validated before passing to git commands
- **Command execution**: Use `exec.CommandContext` with timeouts (10 seconds) to prevent hanging
- **File path safety**: Use `validateWorkPath()` or ensure paths are within `.work/` directory
- **Git command safety**: Use `#nosec G204` comment with justification (sanitized message, validated paths)

### Edge Cases

- **Work item file missing front matter**: Use "unknown" for missing fields in commit message
- **Work item title contains special characters**: Sanitize in commit message (handled by `sanitizeCommitMessage()`)
- **Very long work item titles**: Commit message length limit (1000 chars) enforced by sanitization
- **Moving to same status**: Should this be prevented? (Out of scope - current behavior allows it)
- **Git repository with uncommitted changes**: Only stage the moved file, don't affect other uncommitted changes
- **Git repository with staged changes**: Command should fail with error if other files are already staged (prevents accidental inclusion of unrelated changes in commit)
- **Git repository with only moved file staged**: Commit proceeds normally (this is the expected state after staging the move)

## Release Notes

### User-Facing Changes

- **New flag**: `kira move` command now supports `--commit` (or `-c`) flag to automatically commit moves to git
- **Commit messages**: When using `--commit`, moves are committed with descriptive messages including work item type, ID, title, and status transition
- **Configurable format**: Commit message format can be customized via `move_subject_template` and `move_body_template` in `kira.yml`

### Usage Examples

```bash
# Move without committing (existing behavior)
kira move 007 doing

# Move and commit in one step
kira move 007 doing --commit
# or
kira move 007 doing -c
```

### Known Limitations

- The `--commit` flag requires git to be installed and the repository to be initialized
- Only the moved work item file is committed; other uncommitted changes in `.work/` are not affected
- If git operations fail, the move still succeeds (file is moved but not committed)

### Future Enhancements (Out of Scope)

- Option to commit all `.work/` changes when moving (like `kira save`)
- Automatic commit without flag (always commit moves)
- Integration with git hooks or CI/CD workflows
- Additional template variables (e.g., date, author, assigned user)
