---
id: 008
title: List users
status: done
kind: prd
assigned:
estimate: 0
created: 2025-12-04
due: 2025-12-04
tags: []
---

# List users

List email addresses of all users from the git history and associate them with a number that can be used to assign work items to the correct user. These numbers will be in order of first commit to the repo, so new users will have a higher number than existing users. The kira config should allow for users to be ignored.

## Context

Currently, work items in kira support an `assigned` field that accepts an email address. However, there's no convenient way to:
- Discover which users have contributed to the repository
- Quickly reference users by a short identifier instead of typing full email addresses
- Maintain consistency in assignment (avoiding typos or variations in email format)

This feature will enable users and agents to:
- See all contributors to the repository at a glance
- Use short numeric identifiers when assigning work items
- Filter out automated commits, bots, or other non-human contributors
- Maintain a stable numbering system based on contribution history
- Include agents, future contributors, or external collaborators who haven't committed yet

The feature should integrate seamlessly with existing kira workflows, particularly when creating or updating work items that require assignment. This is especially important for agents (LLMs) that may be assigned work items before they've made any commits to the repository.

## Requirements

### Command Behavior

Add a new `kira users` command that:
- Lists all unique email addresses found in git commit history (when enabled)
- Includes saved users configured in `kira.yml` that aren't in git history
- When `use_git_history` is `false`, only shows saved users from `saved_users`
- Associates each email with a sequential number (starting from 1)
- Orders users by their first commit date (earliest contributor = 1, newest = highest number)
- Saved users (without git history) are numbered after git history users
- When git history is disabled, saved users are numbered sequentially in config order
- Saved users with emails matching git history users are deduplicated (saved user name takes precedence for display)
- Displays output in a readable format (table or list)
- Respects ignore list from configuration (only when git history is enabled)

**Command syntax:**
```bash
kira users [--format=table|list|json] [--limit=N]
```

**Command flags:**
- `--format`: Output format (table, list, or json)
- `--limit`: Optional limit on number of commits to process (useful for very large repositories)

**Output format options:**
- `table` (default): Human-readable table with columns: Number, User, First Commit Date, Source
  - User column displays in "Name <email>" format (e.g., "John Doe <john@example.com>")
  - If name is not available, displays email only (e.g., "user@example.com")
  - Names from git history use the author name from commits
  - Names from saved users use the configured name or email if name not provided
- `list`: Simple numbered list format suitable for quick reference
  - Each line: `1. Name <email>` or `1. email@example.com` (if no name)
  - Format: `{number}. {name} <{email}>` or `{number}. {email}` (when name missing)
- `json`: JSON output for programmatic use
  - Includes separate `name` and `email` fields for programmatic access
  - Also includes `display` field with formatted "Name <email>" string for display purposes

**Note on "Source" column:**
- Shows "git" for users found in git history
- Shows "config" for users added via configuration (saved_users)

**Display format:**
- Users are displayed in git-style format: `Name <email>` when name is available
- When name is missing or empty, only email is shown
- This format is consistent with git commit author format and improves readability

### Configuration

Add a new `users` section to `kira.yml` configuration:

```yaml
users:
  use_git_history: true  # Set to false to ignore git history and only use saved users
  commit_limit: 50000  # Optional: limit git log processing (default: no limit, ignored if use_git_history is false)
  ignored_emails:
    - noreply@github.com
    - bot@example.com
    - ci@example.com
  ignored_patterns:
    - "*bot*"
    - "*noreply*"
  saved_users:
    - email: agent@example.com
      name: AI Agent  # Optional: displays as "AI Agent <agent@example.com>"
    - email: future-contributor@example.com
      name: Future Contributor  # Optional: displays as "Future Contributor <future-contributor@example.com>"
    - email: user@example.com  # No name provided: displays as "user@example.com"
```

**Configuration fields:**
- `use_git_history`: Boolean flag to enable/disable git history extraction
  - Default: `true` (process git history)
  - When `false`: Only use users from `saved_users` config, ignore git history entirely
  - Useful for non-git environments, manual user management, or avoiding git performance issues
  - When `false`, `commit_limit` and ignore patterns are ignored (not applicable)
- `commit_limit`: Optional integer limit on number of commits to process from git history
  - Default: no limit (process all commits)
  - Only applies when `use_git_history` is `true`
  - Recommended for repositories with > 50,000 commits if performance is slow
  - When set, processes most recent N commits but still maintains chronological ordering for first commit detection
  - Command-line `--limit` flag overrides config value
- `ignored_emails`: Array of exact email addresses to exclude from the list
  - Only applies when `use_git_history` is `true`
  - Saved users in `saved_users` are never ignored
- `ignored_patterns`: Array of glob patterns to match against email addresses (case-insensitive)
  - Only applies when `use_git_history` is `true`
  - Saved users in `saved_users` are never ignored
- `saved_users`: Array of user objects to include even if they're not in git history
  - Each user object requires `email` field
  - `name` field is optional (if not provided, displays as email only)
  - Display format: `Name <email>` when name is provided, `email` when name is missing
  - Useful for agents, future contributors, or external collaborators
  - When `use_git_history` is `false`, these are the only users shown (numbered sequentially starting from 1)
  - **Duplicate handling**: If a saved user's email matches a git history user, merge the information (no duplicate shown)
    - Git history provides commit date and other git metadata
    - Saved user name takes precedence for display (if saved user has a name configured)
    - If saved user has no name, git history name is used for display
    - Only one entry is shown per email address (deduplicated)

**User numbering for saved users:**
- Saved users are numbered sequentially after all git history users
- **Duplicate prevention**: If a saved user's email matches a git history user, merge the information
  - Only one entry is shown per email address (deduplicated)
  - Git history provides commit date and other git metadata
  - **Saved user name takes precedence** for display (if saved user has a name configured)
  - If saved user has no name, git history name is used for display
  - This ensures each email address appears only once in the output
- Saved users maintain their order as specified in the config file
- Example: If git history has users 1-5, saved users become 6, 7, 8, etc. (unless their emails match git users)

**Default behavior:**
- If `users` section is not present in config, use empty ignore lists and no saved users (show only git users)
- If `saved_users` is not specified, only git history users are shown
- Saved users with duplicate emails (matching git history) are automatically deduplicated
- Common bot emails (like `noreply@github.com`) should be documented as recommended ignores but not hardcoded

### Git History Extraction

The command should:
- **When `use_git_history` is `true` (default):**
  - Use `git log --all --format="%ae|%an|%ai" --reverse` or similar to extract author email, name, and date
  - Parse commits in the repository history (respecting `--limit` flag or `users.commit_limit` config)
  - When limit is specified, use `git log --all --format="%ae|%an|%ai" --reverse -N` where N is the limit
  - Track the earliest commit date for each unique email address within the processed commits
  - **Note on limits:** When a limit is applied, only commits within that range are considered for "first commit" detection
    - This means users who first committed before the limit may not appear, or their first commit date may be inaccurate
    - Recommend documenting this behavior and suggesting users increase limit if they need complete history
  - Handle edge cases:
    - Empty repository (show helpful message)
    - No git repository (show error message)
    - Git not installed (show error message)
    - Commits with missing author information (skip or handle gracefully)
    - Invalid limit values (negative, zero, or non-numeric) should show error or use default
- **When `use_git_history` is `false`:**
  - Skip all git history processing
  - No git commands are executed
  - Works in non-git environments
  - Only processes users from `saved_users` configuration
  - If `saved_users` is empty, show message indicating no users configured

### User Numbering

- Numbers are assigned sequentially starting from 1
- **When `use_git_history` is `true` (default):**
  - **Git history users**: Ordering is based on first commit date (chronological order)
    - If two users have the same first commit date, order by email address (alphabetical)
  - **Saved users**: Numbered sequentially after all git history users
    - Maintain order as specified in `users.saved_users` array
    - **Duplicate prevention**: If a saved user's email matches a git history user, merge the information
      - Git entry provides commit date and other git metadata
      - **Saved user name takes precedence** for display (if saved user has a name configured)
      - If saved user has no name, git history name is used for display
      - Only one entry is shown per email address (no duplicate)
      - Email addresses are matched case-insensitively for duplicate detection
  - If a user is ignored via config, their number is skipped (e.g., if user 3 is ignored, next user is still 4)
  - Saved users are never ignored (they're explicitly added, but may be deduplicated if email matches git history)
- **When `use_git_history` is `false`:**
  - Only saved users from `saved_users` are shown
  - Users are numbered sequentially starting from 1 in the order they appear in config
  - No git history processing occurs (faster, no git dependency)
  - No duplicate detection needed (no git history to compare against)
- Numbers should remain stable across runs (same user = same number)

### Integration with Work Item Assignment

While this PRD focuses on listing users, the numbering system should be designed to support future enhancements:
- Using numbers in `assigned` field (e.g., `assigned: 5` instead of `assigned: user@example.com`)
- Validation that assigned numbers correspond to valid users
- Auto-completion or suggestions when assigning work items

For this initial implementation, the command should output the mapping, but work items will continue to use email addresses in the `assigned` field.

### Error Handling

The command should handle:
- **When `use_git_history` is `true`:**
  - **Not a git repository**: Display clear error message
  - **Git not installed**: Display clear error message with installation guidance
  - **Empty repository**: Display message indicating no users found
  - **No commits**: Display message indicating no users found
  - **Permission errors**: Display appropriate error message
  - **Invalid limit**: Display error for invalid `--limit` values (negative, zero, non-numeric)
  - **Performance warnings**: Optionally warn if processing takes > 5 seconds (suggest using `--limit`)
- **When `use_git_history` is `false`:**
  - **No saved users**: Display message indicating no users in `saved_users` config
  - Git-related errors are not relevant (git is not used)

### Performance Considerations

- For large repositories, the command should complete within reasonable time (< 5 seconds for typical repos)
- Git log performance is generally fast, but processing output can slow down with very large commit histories
- **Performance testing results:**
  - Small repos (< 100 commits): < 0.1 seconds
  - Medium repos (100-10,000 commits): < 1 second typically
  - Large repos (10,000-50,000 commits): 1-3 seconds typically
  - Very large repos (> 50,000 commits): May exceed 5 seconds, limit recommended
- **Commit limit option:** Add optional `--limit` flag or `users.commit_limit` config to limit git log processing
  - When limit is set, only process the most recent N commits (still maintains chronological ordering for first commit detection)
  - Useful for very large repositories where full history processing is slow
  - Default: no limit (process all commits)
- Consider caching results if needed (future enhancement, not required for initial implementation)
- Use efficient git log queries to minimize execution time
- The `--reverse` flag processes commits chronologically but may be slower on very large repos

## Acceptance Criteria

### Command Execution
- [ ] `kira users` command exists and is accessible
- [ ] Command displays list of users with numbers, user info (name and email), first commit dates, and source
- [ ] Users are displayed in "Name <email>" format when name is available
- [ ] Users are displayed as email only when name is not available
- [ ] Git history users use author name from commits for display
- [ ] Saved users use configured name for display (or email if name not provided)
- [ ] Git history users are ordered by first commit date (earliest = 1)
- [ ] Saved users appear after git history users with sequential numbering
- [ ] Saved users with emails matching git history users are deduplicated (only one entry shown)
- [ ] When duplicate exists, saved user name takes precedence for display (if saved user has name)
- [ ] When duplicate exists and saved user has no name, git history name is used for display
- [ ] Git history commit date is still used when duplicate exists
- [ ] Numbers are sequential and stable across multiple runs
- [ ] Default output format is a readable table

### Output Formats
- [ ] `--format=table` displays formatted table with columns (Number, User, First Commit Date, Source)
- [ ] Table User column displays "Name <email>" format when name available
- [ ] Table User column displays email only when name not available
- [ ] `--format=list` displays numbered list in format "1. Name <email>" or "1. email@example.com"
- [ ] List format uses "Name <email>" when name available, email only when name missing
- [ ] `--format=json` outputs valid JSON structure with separate name, email, and display fields
- [ ] JSON includes `display` field with formatted "Name <email>" string
- [ ] Invalid format option shows helpful error message
- [ ] Source column/field distinguishes "git" vs "config" users

### Configuration
- [ ] `kira.yml` accepts `users.use_git_history` boolean (defaults to `true`)
- [ ] `kira.yml` accepts `users.commit_limit` integer (optional, only applies when `use_git_history` is `true`)
- [ ] `kira.yml` accepts `users.ignored_emails` array (only applies when `use_git_history` is `true`)
- [ ] `kira.yml` accepts `users.ignored_patterns` array (only applies when `use_git_history` is `true`)
- [ ] `kira.yml` accepts `users.saved_users` array with email and optional name fields
- [ ] When `use_git_history` is `false`, only saved users are shown
- [ ] When `use_git_history` is `false`, saved users are numbered sequentially starting from 1
- [ ] When `use_git_history` is `false`, no git commands are executed
- [ ] `commit_limit` defaults to no limit when not specified (only when `use_git_history` is `true`)
- [ ] Exact email matches in `ignored_emails` are excluded from output (only when `use_git_history` is `true`)
- [ ] Email addresses matching patterns in `ignored_patterns` are excluded (only when `use_git_history` is `true`)
- [ ] Pattern matching is case-insensitive
- [ ] Empty or missing `users` section shows only git users (no filtering, no saved users, no limit, git enabled)
- [ ] Saved users appear in output even if they have no git history
- [ ] Saved users with matching git history emails are deduplicated (only one entry shown, no duplicate)
- [ ] When duplicate exists, saved user name is used for display (if provided)
- [ ] Duplicate detection is case-insensitive (email@example.com matches Email@Example.com)
- [ ] Saved users maintain order as specified in config (when not deduplicated)
- [ ] Saved users are numbered after git history users (when git enabled and not deduplicated)
- [ ] Saved users with missing name field display as email only (not "email <email>")
- [ ] Saved users with name field display as "Name <email>" format
- [ ] Git history users display as "Name <email>" using author name from commits

### Git Integration
- [ ] When `use_git_history` is `true`, command extracts email addresses from git commit history
- [ ] When `use_git_history` is `true`, command extracts author names from git commit history
- [ ] When `use_git_history` is `true`, command extracts commit dates from git commit history
- [ ] When `use_git_history` is `true`, first commit date per email is correctly identified
- [ ] When `use_git_history` is `true`, duplicate emails are deduplicated (same email = same number)
- [ ] When `use_git_history` is `true`, handles repositories with thousands of commits efficiently
- [ ] When `use_git_history` is `false`, no git commands are executed
- [ ] When `use_git_history` is `false`, works in non-git environments
- [ ] `--limit` flag limits git log processing when specified (only when git enabled)
- [ ] `users.commit_limit` config limits git log processing when specified (only when git enabled)
- [ ] Command-line `--limit` overrides config `commit_limit` value (only when git enabled)
- [ ] Invalid limit values are handled gracefully with error messages (only when git enabled)
- [ ] Performance is acceptable (< 5 seconds) for typical repositories (when git enabled)
- [ ] When git disabled, command executes instantly (no git processing)

### Error Handling
- [ ] Non-git directory shows appropriate error message
- [ ] Missing git installation shows appropriate error message
- [ ] Empty repository shows appropriate message
- [ ] Repository with no commits shows appropriate message
- [ ] Invalid config values are handled gracefully

### Edge Cases
- [ ] Users with same first commit date are ordered consistently (alphabetical by email, when git enabled)
- [ ] Commits with missing author info are handled gracefully (when git enabled)
- [ ] Special characters in email addresses are displayed correctly
- [ ] Unicode characters in author names are displayed correctly
- [ ] Very long email addresses are displayed without breaking table format
- [ ] Saved user email matching git history user doesn't create duplicate (when git enabled)
- [ ] When duplicate exists, saved user name takes precedence over git history name for display
- [ ] Saved users without name field display email as name
- [ ] Empty `saved_users` array doesn't cause errors (shows git users if git enabled, or empty message if git disabled)
- [ ] Invalid email format in `saved_users` is handled gracefully (validation or warning)
- [ ] Saved users with duplicate emails within `saved_users` array are handled (show only one, or show error)
- [ ] Limit of 1 still processes commits correctly (when git enabled)
- [ ] Limit larger than total commits processes all commits (no error, when git enabled)
- [ ] Negative limit values show appropriate error (when git enabled)
- [ ] Zero limit shows appropriate error or warning (when git enabled)
- [ ] Non-numeric limit values show appropriate error (when git enabled)
- [ ] When `use_git_history` is `false` and `saved_users` is empty, shows appropriate message
- [ ] When `use_git_history` is `false`, git-related config options are ignored (no errors)

### Documentation
- [ ] Command is documented in README.md
- [ ] Configuration options are documented with examples
- [ ] Usage examples are provided
- [ ] Recommended ignore patterns are documented
- [ ] Performance considerations and when to use `--limit` are documented
- [ ] `commit_limit` config option is documented with recommendations
- [ ] `use_git_history` config option is documented with use cases
- [ ] Config-only mode (no git) is documented as alternative for non-git environments

## Implementation Notes

### Git Command Strategy

Use git log with format string to extract all needed information in one pass:
```bash
git log --all --format="%ae|%an|%ai" --reverse
```

This provides:
- `%ae`: Author email
- `%an`: Author name
- `%ai`: Author date (ISO format)

The `--reverse` flag ensures commits are processed chronologically, making it easier to track first occurrence.

### Data Structure

Consider using a map to track first commit date per email:
```go
type UserInfo struct {
    Email      string
    Name       string
    FirstCommit time.Time
    Number     int
}
```

Process commits chronologically, tracking the earliest date for each email.

### Configuration Structure

Add to `config.Config`:
```go
type AdditionalUser struct {
    Email string `yaml:"email"`
    Name  string `yaml:"name,omitempty"`
}

type UsersConfig struct {
    UseGitHistory   bool             `yaml:"use_git_history,omitempty"` // Defaults to true
    CommitLimit     int              `yaml:"commit_limit,omitempty"`    // 0 means no limit, only when UseGitHistory is true
    IgnoredEmails   []string         `yaml:"ignored_emails"`           // Only when UseGitHistory is true
    IgnoredPatterns []string         `yaml:"ignored_patterns"`           // Only when UseGitHistory is true
    SavedUsers      []SavedUser      `yaml:"saved_users"`               // Previously "additional_users"
}

// Note: SavedUser is the same structure as AdditionalUser, renamed for clarity
type SavedUser struct {
    Email string `yaml:"email"`
    Name  string `yaml:"name,omitempty"`
}

type Config struct {
    // ... existing fields
    Users UsersConfig `yaml:"users"`
}
```

**Handling saved users:**
- Merge saved users with git history users
- **Duplicate detection**: Check for duplicates by email (case-insensitive matching)
  - If email matches, merge information from both sources
  - Git history provides commit date and other git metadata
  - **Saved user name takes precedence** for display (if saved user has a name configured)
  - If saved user has no name, git history name is used for display
  - Only one entry is shown per email address (deduplicated)
- Assign numbers: git users first (by commit date), then saved users (by config order, excluding duplicates)
- **Display format logic:**
  - If user has name: display as "Name <email>"
  - If user has no name (null/empty): display as email only
- **Name priority**: Saved user name > Git history name > Email only

### Pattern Matching

For `ignored_patterns`, implement glob-style matching:
- Use `filepath.Match` or similar glob matching library
- Convert patterns to lowercase for case-insensitive matching
- Test patterns against email addresses (also converted to lowercase)

### Command Location

Create new file: `internal/commands/users.go`
- Follow existing command patterns (see `save.go`, `move.go` for reference)
- Register command in `root.go`
- Use cobra command structure consistent with other commands
- Add `--limit` flag using cobra's `IntVarP` or similar (only applies when git enabled)
- Read `users.use_git_history` from config (default to `true` if not specified)
- Read `users.commit_limit` from config and use as default if `--limit` not provided (only when git enabled)
- Check `use_git_history` flag before executing any git commands
- When `use_git_history` is `false`, skip all git processing and only process `saved_users`
- Implement duplicate detection: compare saved user emails (case-insensitive) with git history emails
- When duplicate found, merge information: use git history for commit date, use saved user name for display (if provided)
- If saved user has no name, use git history name for display

### Output Formatting

For table format, consider using a library like `github.com/jedib0t/go-pretty/v6/table` or implement simple column formatting:
- Column widths should auto-adjust to content
- Long emails/names should be truncated with ellipsis if needed
- Ensure table is readable in terminal

For JSON output, use standard `encoding/json`:
```json
{
  "users": [
    {
      "number": 1,
      "email": "user@example.com",
      "name": "User Name",
      "first_commit": "2024-01-15T10:00:00Z",
      "source": "git"
    },
    {
      "number": 6,
      "email": "agent@example.com",
      "name": "AI Agent",
      "first_commit": null,
      "source": "config"
    }
  ]
}
```

**Note:** Saved users will have `first_commit` as `null` since they have no git history. However, if a saved user's email matches a git history user, the git entry is shown instead (with actual commit date).

### Testing Considerations

- Unit tests for user extraction logic
- Unit tests for ignore pattern matching
- Unit tests for saved users merging with git history
- Unit tests for duplicate detection (saved user email matching git history email)
  - Test case-insensitive email matching
  - Test saved user name takes precedence for display when duplicate exists
  - Test git history name is used when saved user has no name
  - Test git history commit date is used when duplicate exists
  - Test only one entry is shown per email address
  - Test saved user entry is shown when no duplicate exists
- Unit tests for commit limit handling
- Unit tests for `use_git_history` flag (both true and false)
- Unit tests for config-only mode (no git processing)
- Integration tests with test git repositories
- Test with various git repository states (empty, single commit, many commits)
- Test edge cases (missing authors, special characters, etc.)
- Test saved users without names (should display as email only)
- Test saved users with names (should display as "Name <email>")
- Test git history users display as "Name <email>" using commit author names
- Test saved users with duplicate emails within `saved_users` array (should handle gracefully)
- Test saved users matching ignored patterns (should still appear since they're explicitly added, unless deduplicated)
- Test duplicate detection between saved users and git history (case-insensitive)
- Test when duplicate exists, saved user name takes precedence for display (if saved user has name)
- Test when duplicate exists and saved user has no name, git history name is used for display
- Test git history commit date is preserved when duplicate exists
- Test when `use_git_history` is `false` - should work without git repository
- Test when `use_git_history` is `false` and `saved_users` is empty - should show appropriate message
- Test display format consistency across table, list, and JSON formats
- **Performance tests:**
  - Test with small repos (< 100 commits) - should be < 0.1s (when git enabled)
  - Test with medium repos (100-10,000 commits) - should be < 1s (when git enabled)
  - Test with large repos (10,000-50,000 commits) - should be < 3s (when git enabled)
  - Test limit flag reduces processing time on large repos (when git enabled)
  - Test that limit doesn't break functionality (still finds users correctly)
  - Test config-only mode is instant (no git processing overhead)

### Security Considerations

- Validate git command output to prevent injection
- Sanitize email addresses before display (though git output should be safe)
- Follow existing patterns from `save.go` and `move.go` for git command execution
- Use `exec.CommandContext` with timeouts (see existing code patterns)
- Validate file paths and prevent path traversal (though not directly applicable here)

### Performance Optimization

- Process git log output in a single pass
- Use efficient data structures (maps for O(1) lookups)
- **Implement commit limit:** Use `git log --reverse -N` when limit is specified
  - Limit should be applied at git level, not after processing all commits
  - This reduces both git execution time and memory usage
- **Performance monitoring:** Consider measuring execution time and warning if > 5 seconds
  - Suggest using `--limit` flag if performance is slow
  - Can be optional feature (not required for initial implementation)
- Cache results if needed (future enhancement, not required initially)

### Performance Testing

A test script (`test_git_performance.sh`) has been created to benchmark git log performance:
- Tests full history vs limited commits (100, 1000, 10000, etc.)
- Measures unique email extraction time
- Measures full processing time (extract, deduplicate, track first commit)
- **Findings:**
  - Small repos (< 100 commits): < 0.1 seconds
  - Medium repos (100-10,000 commits): < 1 second typically
  - Large repos (10,000-50,000 commits): 1-3 seconds typically
  - Very large repos (> 50,000 commits): May exceed 5 seconds
- **Recommendation:** Add `--limit` option for repos with > 50,000 commits or when performance is slow

## Release Notes

- Added `kira users` command to list all contributors from git history
- Users are numbered sequentially based on first commit date (earliest contributor = 1)
- Command supports multiple output formats: table (default), list, and JSON
- Added `--limit` flag to limit git log processing for very large repositories
- Added `users` configuration section to `kira.yml` for ignoring specific emails or patterns
- Added `users.use_git_history` config option to disable git history and use only saved users
- Config-only mode enables use in non-git environments or for manual user management
- Supports ignoring exact email addresses via `users.ignored_emails` (when git enabled)
- Supports ignoring emails matching glob patterns via `users.ignored_patterns` (when git enabled)
- Added `users.commit_limit` config option to set default commit processing limit (when git enabled)
- Added `users.saved_users` to include users not in git history (e.g., agents, future contributors)
- Saved users are numbered after git history users and maintain config order (when git enabled)
- When git disabled, saved users are numbered sequentially starting from 1
- **Duplicate prevention**: Saved users with emails matching git history are automatically deduplicated
  - Git history provides commit date and other git metadata
  - **Saved user name takes precedence** for display (if saved user has a name configured)
  - If saved user has no name, git history name is used for display
  - Only one entry is shown per email address (no duplicate)
  - Email matching is case-insensitive for duplicate detection
  - Ensures each email address appears only once in the output
- User numbers remain stable across command runs for consistent assignment reference
- Displays users in "Name <email>" format (git-style) for improved readability
- Shows author email, name, first commit date, and source (git/config) for each contributor
- Display format matches git commit author format for consistency
- Performance optimized for repositories of all sizes (< 5 seconds for typical repos)
- Config-only mode executes instantly with no git processing overhead
- Helps users discover repository contributors and maintain consistent work item assignments
- Enables assignment of work items to agents or collaborators before they've made commits
- Performance tested and optimized for repositories with up to 50,000+ commits
- Supports both git-based and config-only workflows for maximum flexibility

