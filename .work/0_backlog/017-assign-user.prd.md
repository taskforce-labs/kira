---
id: 017
title: assign-user
status: backlog
kind: prd
assigned:
estimate: 0
created: 2026-01-21
due: 2026-01-21
tags: [cli, workflow, users]
---

# assign-user

A command that assigns work items to users by updating user-related fields in the work item's front matter. The command supports multiple input formats including numeric user identifiers from `kira users`, email addresses, and interactive selection. It can handle single or multiple work items, supports custom fields (defaulting to `assigned`), and provides switch/append modes for managing existing assignments.

## Context

Currently, work items in Kira support an `assigned` field that accepts an email address. However, assigning work items requires manually editing the markdown file or using the `kira new` command with `--input assigned=email@example.com`. The `kira users` command provides a convenient way to list all users with numeric identifiers, but there's no streamlined way to use those identifiers to assign work items.

This creates friction in the workflow, especially when:
- Assigning existing work items to team members
- Agents (LLMs) need to assign work items programmatically
- Quickly reassigning work items between team members
- Using the numeric identifiers from `kira users` for faster assignment

The `kira assign` command will bridge this gap by providing a simple, intuitive interface for assigning work items using:
- Numeric user identifiers (from `kira users`)
- Email addresses (direct or partial match)
- Interactive selection from available users
- Support for unassigning work items (clearing the field)
- Multiple work items in a single command (batch assignments)
- Custom field support (defaults to `assigned`, but can target fields like `reviewer`)
- Switch mode (default) to replace existing assignments, or append mode to add to existing assignments

This command integrates seamlessly with the existing `kira users` command and follows established Kira patterns for work item manipulation.

**Dependencies**: This command requires:
- The `kira users` command to be functional
- Work items to exist and be accessible
- Valid user identification (number, email, or name)

## Implementation Phases

This feature will be implemented in vertical slices (by technical layer) to enable incremental testing and small, focused commits:

### Phase 1: Command Structure & Input Validation
**Goal**: Set up command skeleton and validate all inputs

**Scope**:
- Create `kira assign` command structure (Cobra)
- Define command flags (`--field`, `--append`, `--unassign`, `--interactive`, `--dry-run`)
- Parse command arguments (work item IDs, user identifier)
- Validate work item ID format
- Validate flag combinations (e.g., can't use `--unassign` with user identifier)
- Basic error messages for invalid inputs

**Files to Create/Modify**:
- `internal/commands/assign.go` (new file)
- `internal/commands/root.go` (register command)

**Acceptance Criteria**:
- [ ] `kira assign` command exists and shows help
- [ ] Command validates work item ID format
- [ ] Command rejects invalid flag combinations
- [ ] Command shows clear error for missing required arguments
- [ ] All flags are defined and accessible
- [ ] Command structure follows existing Kira patterns

**Deliverable**: Command skeleton with input validation

---

### Phase 2: Work Item Discovery & Validation
**Goal**: Find and validate work items exist

**Scope**:
- Find work item by ID across all status folders
- Validate work item file exists and is readable
- Support single and multiple work item IDs
- Return clear errors when work items not found
- Handle work items in any status folder

**Files to Create/Modify**:
- `internal/commands/assign.go` (add work item discovery functions)

**Acceptance Criteria**:
- [ ] Finds work item by numeric ID across all folders
- [ ] Validates work item file exists
- [ ] Handles multiple work item IDs
- [ ] Returns clear error when work item not found
- [ ] Works with work items in any status folder
- [ ] Handles invalid work item paths gracefully

**Deliverable**: Work item lookup and validation working

---

### Phase 3: User Collection & Resolution
**Goal**: Collect users and resolve user identifiers

**Scope**:
- Reuse user collection logic from `kira users` command
- Resolve user by numeric identifier
- Resolve user by email address (exact match)
- Resolve user by name (exact match)
- Handle user not found errors
- Support partial matching (email, name)
- Handle multiple matches with selection

**Files to Create/Modify**:
- `internal/commands/assign.go` (add user resolution functions)
- Reuse functions from `internal/commands/users.go`

**Acceptance Criteria**:
- [ ] Collects users using same logic as `kira users`
- [ ] Resolves numeric user identifiers correctly
- [ ] Resolves email addresses (case-insensitive)
- [ ] Resolves names (case-insensitive)
- [ ] Handles partial email/name matches
- [ ] Shows clear error when user not found
- [ ] Handles multiple matches appropriately

**Deliverable**: User resolution working for all identifier types

---

### Phase 4: Front Matter Parsing & Field Access
**Goal**: Read and parse work item front matter

**Scope**:
- Read work item file
- Parse YAML front matter
- Access field values (default: `assigned`)
- Handle missing fields
- Handle empty fields
- Preserve front matter formatting
- Get current field value for display

**Files to Create/Modify**:
- `internal/commands/assign.go` (add front matter parsing)
- Reuse existing front matter utilities if available

**Acceptance Criteria**:
- [ ] Reads work item file successfully
- [ ] Parses YAML front matter correctly
- [ ] Accesses field values (default and custom)
- [ ] Handles missing fields gracefully
- [ ] Handles empty fields correctly
- [ ] Preserves front matter structure
- [ ] Handles malformed front matter with clear errors

**Deliverable**: Front matter reading and field access working

---

### Phase 5: Field Update Logic (Switch Mode)
**Goal**: Update field value in front matter (replace mode)

**Scope**:
- Update field value (switch/replace mode)
- Create field if it doesn't exist
- Update `updated` timestamp
- Preserve all other front matter fields
- Write updated front matter back to file
- Handle file write errors

**Files to Create/Modify**:
- `internal/commands/assign.go` (add field update functions)

**Acceptance Criteria**:
- [ ] Updates field value correctly
- [ ] Creates field if it doesn't exist
- [ ] Updates `updated` timestamp
- [ ] Preserves other front matter fields
- [ ] Writes file successfully
- [ ] Handles write errors gracefully
- [ ] Maintains front matter formatting

**Deliverable**: Basic field update (switch mode) working

---

### Phase 6: Append Mode Logic
**Goal**: Support appending to existing field values

**Scope**:
- Detect if field is single value or array
- Convert single value to array when appending
- Append to array (avoiding duplicates)
- Handle field creation in append mode
- Update `updated` timestamp

**Files to Create/Modify**:
- `internal/commands/assign.go` (add append mode logic)

**Acceptance Criteria**:
- [ ] Appends to array fields correctly
- [ ] Converts single value to array when needed
- [ ] Prevents duplicate entries in arrays
- [ ] Creates field if it doesn't exist (append mode)
- [ ] Updates timestamp correctly
- [ ] Preserves other front matter fields

**Deliverable**: Append mode working

---

### Phase 7: Unassign Logic
**Goal**: Clear field values

**Scope**:
- Clear field value (set to empty or remove)
- Support default and custom fields
- Update `updated` timestamp
- Handle fields that don't exist

**Files to Create/Modify**:
- `internal/commands/assign.go` (add unassign logic)

**Acceptance Criteria**:
- [ ] Clears field value correctly
- [ ] Works with default `assigned` field
- [ ] Works with custom fields via `--field`
- [ ] Updates timestamp
- [ ] Handles non-existent fields gracefully

**Deliverable**: Unassign functionality working

---

### Phase 8: Batch Processing & Progress
**Goal**: Handle multiple work items efficiently

**Scope**:
- Process multiple work items in sequence
- Validate all work items before processing
- Show progress for each work item
- Handle partial failures
- Provide summary of results

**Files to Create/Modify**:
- `internal/commands/assign.go` (add batch processing)

**Acceptance Criteria**:
- [ ] Processes multiple work items correctly
- [ ] Validates all work items before processing
- [ ] Shows progress for each item
- [ ] Handles partial failures gracefully
- [ ] Provides clear summary of results

**Deliverable**: Batch processing working

---

### Phase 9: Interactive Mode
**Goal**: User-friendly selection interface

**Scope**:
- Display users in numbered list
- Show current assignment if exists
- Get user selection from input
- Handle invalid selections
- Support unassign option in menu

**Files to Create/Modify**:
- `internal/commands/assign.go` (add interactive mode)

**Acceptance Criteria**:
- [ ] Displays users in consistent format
- [ ] Shows current assignment
- [ ] Gets valid user selection
- [ ] Handles invalid input gracefully
- [ ] Supports unassign option

**Deliverable**: Interactive mode working

---

### Phase 10: Output & Feedback
**Goal**: User-friendly messages and dry-run

**Scope**:
- Success messages for assignments
- Success messages for unassignments
- Error messages with actionable guidance
- Dry-run mode (preview without changes)
- Progress indicators
- Summary output for batch operations

**Files to Create/Modify**:
- `internal/commands/assign.go` (add output functions)

**Acceptance Criteria**:
- [ ] Shows clear success messages
- [ ] Shows helpful error messages
- [ ] Dry-run mode works correctly
- [ ] Progress indicators are clear
- [ ] Batch summaries are informative

**Deliverable**: Polished output and feedback

---

### Phase Summary

| Phase | Layer | Focus | Testability |
|-------|-------|-------|-------------|
| 1 | Input | Command structure, argument parsing, validation | Unit tests for validation |
| 2 | Discovery | Work item lookup and validation | Unit tests for file discovery |
| 3 | Resolution | User collection and identifier resolution | Unit tests for user resolution |
| 4 | Parsing | Front matter reading and field access | Unit tests for parsing |
| 5 | Update | Field update logic (switch mode) | Integration tests with files |
| 6 | Append | Append mode logic | Unit tests for array handling |
| 7 | Unassign | Field clearing logic | Integration tests |
| 8 | Batch | Multiple work item processing | Integration tests |
| 9 | Interactive | User selection interface | Manual/E2E tests |
| 10 | Output | Messages, dry-run, feedback | Unit tests for formatting |

**Implementation Strategy**:
- Each phase builds on previous layers
- Each phase can be tested independently
- Small, focused commits per phase
- Can test each layer in isolation before moving to next
- Final phase wires everything together

## Requirements

### Core Functionality

#### Command Interface
- **Command**: `kira assign <work-item-id...> [user-identifier] [--unassign] [--interactive] [--field <field-name>] [--append]`
- **Behavior**: Updates the specified field (default: `assigned`) in the work item's front matter
- **Work Item IDs**: Can specify one or more work items:
  - Single: `kira assign 001 5`
  - Multiple: `kira assign 001 002 003 5`
  - Can be numeric IDs (e.g., `001`) or full paths
- **User Identifier**: Optional when `--interactive` is used; required otherwise (unless `--unassign`)
- **Flags**:
  - `--field <field-name>` or `-f` - Target field name (default: `assigned`). Examples: `assigned`, `reviewer`, `owner`
  - `--append` or `-a` - Append user to existing assignment (default: switch/replace mode)
  - `--unassign` or `-u` - Clear the field (removes assignment)
  - `--interactive` or `-I` - Show interactive user selection menu
  - `--dry-run` - Preview what would be done without making changes

#### User Identification

The command supports multiple ways to identify users:

1. **Numeric Identifier** (from `kira users`):
   - `kira assign 001 5` - Assign work item 001 to user number 5
   - `kira assign 001 002 003 5` - Assign multiple work items to user number 5
   - Validates that the number corresponds to an existing user

2. **Email Address**:
   - `kira assign 001 user@example.com` - Assign by full email
   - `kira assign 001 @example.com` - Partial email match (if unique)
   - Case-insensitive matching

3. **Name or Display Name**:
   - `kira assign 001 "John Doe"` - Assign by name (if unique)
   - `kira assign 001 John` - Partial name match (if unique)
   - Case-insensitive matching

4. **Interactive Selection**:
   - `kira assign 001 --interactive` - Show numbered list of users
   - `kira assign 001 -I` - Shorthand for interactive mode
   - Displays users in same format as `kira users` command

#### Work Item Management

- Find work items by ID across all status folders
- Support single or multiple work items in one command
- Validate all work items exist and are accessible before processing
- Read work item front matter for each work item
- Update specified field (default: `assigned`) in front matter
- Preserve all other front matter fields
- Update `updated` timestamp when assignment changes
- Support work items in any status (backlog, todo, doing, review, done, etc.)
- Process all work items in batch, showing progress for each

#### Field Management

- **Default Field**: `assigned` (when `--field` not specified)
- **Custom Fields**: Support any field name via `--field` flag
  - `kira assign 001 5 --field reviewer` - Assign reviewer
  - `kira assign 001 5 --field owner` - Assign owner
  - `kira assign 001 5 -f reviewer` - Shorthand
- **Field Format**: Fields can be single values (string) or arrays (list)
- **Field Creation**: Create field if it doesn't exist in front matter

#### Assignment Modes

**Switch Mode (Default)**:
- Replaces existing assignment with new user
- `kira assign 001 5` - Replaces current assignment with user 5
- If field is an array, replaces entire array with single user
- Use when you want to change assignment

**Append Mode**:
- Adds user to existing assignment (if field supports multiple values)
- `kira assign 001 5 --append` - Adds user 5 to existing assignment
- `kira assign 001 5 -a` - Shorthand for append
- If field is single value, converts to array and appends
- If field is array, appends new user (avoiding duplicates)
- If field doesn't exist, creates it with the new user
- Use when building assignments (e.g., multiple reviewers)

#### User Resolution Logic

The command should resolve user identifiers in the following priority order:

1. **Numeric Identifier**: Direct lookup by user number from `kira users`
2. **Exact Email Match**: Case-insensitive exact email match
3. **Partial Email Match**: If unique, match by email domain or partial email
4. **Exact Name Match**: Case-insensitive exact name match
5. **Partial Name Match**: If unique, match by partial name
6. **Display Name Match**: Match against "Name <email>" format

**Error Handling**:
- Multiple matches: Show all matches and prompt for selection
- No matches: Clear error message with suggestions
- Invalid number: "User number {n} not found. Run 'kira users' to see available users."

#### Unassign Functionality

- `kira assign 001 --unassign` - Clear the default field (`assigned`)
- `kira assign 001 --unassign --field reviewer` - Clear the reviewer field
- `kira assign 001 -u` - Shorthand for unassign
- `kira assign 001 002 003 -u` - Unassign multiple work items
- Sets field to empty string or removes it (based on template requirements)
- Updates `updated` timestamp
- Works with any field specified via `--field`

### User Integration

#### Integration with `kira users`

- Use the same user collection logic as `kira users` command
- Respect `kira.yml` user configuration (saved users, ignored emails, etc.)
- Use the same sorting and numbering scheme
- Display users in the same format for consistency

#### User Display Format

When showing users (interactive mode or error messages):
- Use same format as `kira users`: "Name <email>" or email only
- Show user number for reference
- Maintain consistency with existing `kira users` output

### Configuration

#### kira.yml Extensions

No new configuration required. The command uses existing user configuration:
- `users.use_git_history` - Whether to use git history
- `users.commit_limit` - Limit commits processed
- `users.ignored_emails` - Emails to ignore
- `users.ignored_patterns` - Email patterns to ignore
- `users.saved_users` - Saved users from config

### Error Handling

#### Validation Errors
- Work item not found: "Work item {id} not found"
- Invalid work item ID format: "Invalid work item ID format: {id}"
- User not found: "User '{identifier}' not found. Run 'kira users' to see available users."
- Multiple user matches: List all matches and prompt for selection
- Invalid user number: "User number {n} not found. Available numbers: 1-{max}"

#### File Operations
- File read errors: "Failed to read work item {id}: {error}"
- File write errors: "Failed to update work item {id}: {error}"
- Front matter parsing errors: "Failed to parse work item {id}: {error}"
- Permission errors: Clear guidance on file permissions

#### User Resolution Errors
- Ambiguous identifier: Show all matches with numbers for selection
- No users available: "No users found. Configure users in kira.yml or ensure git history is available."

### Output and Feedback

#### Success Messages
- Single assignment: "Assigned work item {id} to {user display}"
- Multiple assignments: "Assigned {count} work items to {user display}"
- Append mode: "Added {user display} to {field} for work item {id}"
- Unassignment: "Unassigned work item {id}"
- Field-specific: "Assigned {field} for work item {id} to {user display}"
- Show current assignment if already assigned (switch mode): "Work item {id} is already assigned to {user}. Use --unassign to clear or specify a different user."
- Show current assignment (append mode): "Work item {id} currently has {field}: {current}. Adding {user display}."

#### Interactive Mode
- Display numbered list of users (same format as `kira users`)
- Prompt: "Select user (number): "
- Show current assignment if work item is already assigned
- Allow selection by number or "0" to unassign

#### Dry Run Mode
- Show what would be changed without making changes
- Display: "Would assign work item {id} to {user display}"
- Display: "Would unassign work item {id}"

## Acceptance Criteria

### Core Command Functionality
- [ ] `kira assign 001 5` assigns work item 001 to user number 5 (switch mode)
- [ ] `kira assign 001 002 003 5` assigns multiple work items to user 5
- [ ] `kira assign 001 user@example.com` assigns by email address
- [ ] `kira assign 001 "John Doe"` assigns by name (if unique)
- [ ] `kira assign 001 --interactive` shows interactive user selection
- [ ] `kira assign 001 --unassign` clears the assigned field
- [ ] `kira assign 001 -u` clears assignment (shorthand)
- [ ] `kira assign 001 5 --field reviewer` assigns to reviewer field
- [ ] `kira assign 001 5 --append` adds user to existing assignment
- [ ] `kira assign 001 5 -a` appends user (shorthand)
- [ ] Command updates specified field (default: `assigned`) in work item front matter
- [ ] Command updates `updated` timestamp when assignment changes
- [ ] Command preserves all other front matter fields
- [ ] Command works with work items in any status folder
- [ ] Command finds work items by numeric ID across all folders
- [ ] Command processes multiple work items in batch

### User Identification
- [ ] Numeric identifiers resolve to correct user from `kira users`
- [ ] Email addresses match case-insensitively
- [ ] Partial email matches work when unique (e.g., `@example.com`)
- [ ] Name matching works for exact matches
- [ ] Partial name matches work when unique
- [ ] Display name format "Name <email>" is recognized
- [ ] Multiple matches show list and prompt for selection
- [ ] Invalid user number shows helpful error with available range

### Interactive Mode
- [ ] Interactive mode displays users in same format as `kira users`
- [ ] Users are numbered consistently with `kira users` output
- [ ] Selection by number works correctly
- [ ] Option to unassign is available in interactive mode
- [ ] Current assignment is shown if work item is already assigned
- [ ] Invalid selection shows error and re-prompts

### Error Handling
- [ ] Work item not found shows clear error message
- [ ] Invalid work item ID format shows validation error
- [ ] User not found shows error with suggestion to run `kira users`
- [ ] Multiple user matches lists all options for selection
- [ ] File read/write errors are handled gracefully
- [ ] Front matter parsing errors provide clear feedback
- [ ] Permission errors provide guidance

### Integration
- [ ] Uses same user collection logic as `kira users`
- [ ] Respects user configuration from `kira.yml`
- [ ] Ignores emails/patterns as configured
- [ ] Includes saved users from config
- [ ] Maintains consistency with `kira users` numbering

### Field Management
- [ ] Default field is `assigned` when `--field` not specified
- [ ] Custom fields work with `--field` flag (e.g., `reviewer`, `owner`)
- [ ] Field is created if it doesn't exist in front matter
- [ ] Switch mode replaces existing assignment (default behavior)
- [ ] Append mode adds user to existing assignment
- [ ] Append mode converts single value to array when needed
- [ ] Append mode avoids duplicate users in array fields
- [ ] Unassign works with custom fields via `--field` flag

### Multiple Work Items
- [ ] Multiple work items can be assigned in single command
- [ ] All work items are validated before processing
- [ ] Progress is shown for each work item assignment
- [ ] Partial failures are reported clearly (which items succeeded/failed)
- [ ] Batch operations are atomic where possible

### Edge Cases
- [ ] Assigning to already-assigned work item updates assignment (switch mode)
- [ ] Append mode adds to existing assignment without replacing
- [ ] Unassigning unassigned work item shows appropriate message
- [ ] Work items with missing field can be assigned (field created)
- [ ] Work items with empty field can be assigned
- [ ] Command works with work items that have complex front matter
- [ ] Special characters in user names/emails are handled correctly
- [ ] Array fields handle append mode correctly
- [ ] Single value fields convert to arrays when appending

### Output and Feedback
- [ ] Success messages show clear assignment information
- [ ] Current assignment is shown when already assigned
- [ ] Dry run mode shows what would change without making changes
- [ ] Interactive mode provides clear prompts and feedback

## Implementation Notes

### Architecture

#### Command Structure
```
internal/commands/assign.go
├── assignCmd - Main cobra command
├── resolveUserIdentifier() - Resolve user by number, email, or name
├── findUserByNumber() - Lookup user by numeric identifier
├── findUserByEmail() - Find user by email (exact or partial)
├── findUserByName() - Find user by name (exact or partial)
├── showInteractiveSelection() - Display users and get selection
├── processWorkItems() - Process single or multiple work items
├── updateWorkItemField() - Update specified field in front matter
├── updateFieldValue() - Update field value (switch or append mode)
├── appendToField() - Append user to field (array handling)
├── unassignWorkItem() - Clear specified field
└── validateWorkItemIDs() - Validate and find work items
```

#### Dependencies
- **User Management**: Reuse logic from `internal/commands/users.go`
- **Work Item Parsing**: Use existing front matter parsing utilities
- **File Operations**: Standard file I/O for reading/writing markdown files

### User Resolution Implementation

#### Reuse User Collection Logic
```go
// Reuse the user collection and processing logic from users.go
func collectUsersForAssignment(cfg *config.Config) ([]UserInfo, error) {
    // Reuse collectUsers and processAndSortUsers logic
    // This ensures consistency with kira users command
}
```

#### User Lookup Functions
```go
func resolveUserIdentifier(identifier string, users []UserInfo) (*UserInfo, error) {
    // Try numeric identifier first
    if num, err := strconv.Atoi(identifier); err == nil {
        return findUserByNumber(num, users)
    }

    // Try email match (exact, then partial)
    if matches := findUsersByEmail(identifier, users); len(matches) == 1 {
        return matches[0], nil
    } else if len(matches) > 1 {
        return nil, fmt.Errorf("multiple users match '%s': %v", identifier, matches)
    }

    // Try name match (exact, then partial)
    if matches := findUsersByName(identifier, users); len(matches) == 1 {
        return matches[0], nil
    } else if len(matches) > 1 {
        return nil, fmt.Errorf("multiple users match '%s': %v", identifier, matches)
    }

    return nil, fmt.Errorf("user '%s' not found", identifier)
}
```

### Work Item Update Implementation

#### Front Matter Update
```go
func updateWorkItemField(workItemPath string, fieldName string, userEmail string, appendMode bool) error {
    // Read file
    content, err := os.ReadFile(workItemPath)
    if err != nil {
        return err
    }

    // Parse front matter
    frontMatter, body, err := parseFrontMatter(content)
    if err != nil {
        return err
    }

    // Update field (switch or append mode)
    if appendMode {
        appendToField(frontMatter, fieldName, userEmail)
    } else {
        frontMatter[fieldName] = userEmail
    }

    // Update updated timestamp
    frontMatter["updated"] = time.Now().Format(time.RFC3339)

    // Write back to file
    return writeWorkItemFile(workItemPath, frontMatter, body)
}

func appendToField(frontMatter map[string]interface{}, fieldName string, userEmail string) {
    currentValue := frontMatter[fieldName]

    // If field doesn't exist, create it
    if currentValue == nil {
        frontMatter[fieldName] = userEmail
        return
    }

    // If single value, convert to array
    if str, ok := currentValue.(string); ok {
        if str == "" {
            frontMatter[fieldName] = userEmail
        } else {
            frontMatter[fieldName] = []string{str, userEmail}
        }
        return
    }

    // If array, append (avoiding duplicates)
    if arr, ok := currentValue.([]string); ok {
        // Check for duplicates
        for _, existing := range arr {
            if existing == userEmail {
                return // Already exists
            }
        }
        frontMatter[fieldName] = append(arr, userEmail)
    }
}
```

#### Front Matter Parsing
- Reuse existing front matter parsing utilities if available
- Handle YAML front matter with proper formatting
- Preserve comments and formatting in front matter
- Handle edge cases (missing fields, empty values, etc.)

### Interactive Selection

#### User Display
```go
func showInteractiveSelection(users []UserInfo, currentAssignment string) (int, error) {
    // Display header
    fmt.Println("Available users:")
    fmt.Println(strings.Repeat("-", 50))

    // Show current assignment if exists
    if currentAssignment != "" {
        fmt.Printf("Current assignment: %s\n\n", currentAssignment)
    }

    // Display users (same format as kira users)
    for _, user := range users {
        display := formatUserDisplay(user) // Reuse from users.go
        fmt.Printf("%d. %s\n", user.Number, display)
    }

    // Add unassign option
    fmt.Println("0. Unassign")
    fmt.Println()

    // Get selection
    fmt.Print("Select user (number): ")
    var selection int
    _, err := fmt.Scanf("%d", &selection)
    if err != nil {
        return 0, err
    }

    return selection, nil
}
```

### Testing Strategy

#### Unit Tests
- Test user resolution by number, email, and name
- Test partial matching logic
- Test multiple match handling
- Test work item front matter parsing and updating
- Test timestamp updates
- Test unassign functionality
- Test switch mode (replace existing assignment)
- Test append mode (add to existing assignment)
- Test field creation for missing fields
- Test custom field handling
- Test array field handling in append mode
- Test single value to array conversion
- Test duplicate prevention in append mode
- Test multiple work item processing
- Test error scenarios (not found, invalid format, etc.)

#### Integration Tests
- Test full assignment workflow with real work items
- Test interactive mode with various user lists
- Test integration with `kira users` command
- Test with different work item statuses
- Test with complex front matter
- Test multiple work items in single command
- Test custom fields (reviewer, owner, etc.)
- Test switch vs append modes
- Test batch operations with partial failures

#### E2E Tests
- `kira assign` command in test environment
- Verify assignment updates work item correctly
- Test interactive selection flow
- Test unassign functionality
- Test multiple work item assignments
- Test custom field assignments (reviewer, etc.)
- Test switch and append modes
- Test error scenarios

### Security Considerations

#### Input Validation
- Sanitize work item IDs to prevent path traversal
- Validate user identifiers before processing
- Escape special characters in user names/emails when displaying
- Validate email format when provided directly

#### File Operations
- Use secure file path operations
- Validate file permissions before writing
- Handle file locking if needed
- Preserve file permissions when updating

#### User Data
- Handle user data securely (emails, names)
- Don't log sensitive user information unnecessarily
- Validate user input before processing

### Edge Cases and Error Scenarios

#### Work Item Not Found
- Search across all status folders
- Provide helpful error if ID format is invalid
- Suggest running `kira doctor` if duplicates suspected

#### User Resolution Ambiguity
- Show all matching users with their numbers
- Allow selection from matches
- Provide clear error if no matches found with suggestions

#### File System Issues
- Handle read-only files gracefully
- Handle concurrent modifications
- Provide clear error messages for permission issues

#### Front Matter Parsing
- Handle malformed front matter
- Preserve formatting and comments
- Handle missing or empty fields gracefully

## Release Notes

### New Features
- **Assign Command**: New `kira assign` command for assigning work items to users
- **Multiple Identification Methods**: Support for numeric IDs, email addresses, and names
- **Interactive Selection**: Interactive mode for easy user selection from available users
- **Unassign Support**: Ability to clear work item assignments
- **Smart User Resolution**: Automatic resolution of user identifiers with partial matching
- **Consistent Integration**: Seamless integration with `kira users` command
- **Multiple Work Items**: Assign multiple work items in a single command
- **Custom Fields**: Support for any field via `--field` flag (default: `assigned`)
- **Switch/Append Modes**: Switch mode (default) replaces assignments, append mode adds to existing assignments
- **Batch Operations**: Process multiple work items efficiently with progress feedback

### Improvements
- Streamlined work item assignment workflow
- Reduced friction when assigning work items to team members
- Support for both human and agent (LLM) workflows
- Clear error messages and user guidance
- Consistent user display format with `kira users` command
- Flexible assignment modes for different use cases (switch vs append)
- Support for custom fields beyond `assigned` (e.g., `reviewer`, `owner`)
- Efficient batch operations for assigning multiple work items at once

### Technical Changes
- New command structure following established Kira patterns
- Reuse of user collection logic from `kira users` command
- Integration with existing front matter parsing utilities
- Extended work item manipulation capabilities
