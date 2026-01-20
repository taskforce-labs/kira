---
id: 013
title: kira open work item
status: backlog
kind: prd
assigned:
estimate: 0
created: 2026-01-20
due: 2026-01-20
tags: []
---

# kira open work item

A command that opens a work-item that is in doing or in review status, it will create a worktree, checkout the branch, and open the IDE in the worktree. Similar to the `kira start` command, but for work items that are already in progress it doesn't change the status or create a branch. It will create the worktree if it doesn't exist and checkout the branch from the remote repository. It won't open if a remote branch doesn't exist.

## Context

In the Kira workflow, developers and agents work on multiple work items simultaneously using git worktrees for isolation. The `kira start` command is used to begin new work items, but there's a gap when resuming work on items that are already in progress.

Currently, developers must manually:
1. Identify which work item they want to continue working on
2. Check if a worktree already exists for that work item
3. Navigate to the worktree directory
4. Ensure the correct branch is checked out
5. Open their IDE in the correct context
6. Handle cases where the worktree doesn't exist but the branch does on remote

This manual process is error-prone and time-consuming, especially when:
- Switching between multiple work items in progress
- Resuming work after a break or context switch
- Working with agents that need to resume work on existing branches
- Managing worktrees across multiple repositories in polyrepo setups

The `kira open` command addresses this by providing a simple way to resume work on existing work items. Unlike `kira start`, which creates new branches and changes work item status, `kira open` focuses on:
- Opening work items that are already in progress (doing or review status)
- Reusing existing branches from remote repositories
- Creating worktrees only when needed
- Opening the IDE in the correct context
- Not modifying work item status or creating new branches

This command is particularly useful for:
- Developers switching between multiple active work items
- Agents resuming work on previously started items
- Teams where work items are shared and need to be opened by different team members
- Workflows where branches exist on remote but local worktrees don't

**Dependencies**: This command relies on work items having branches that exist on the remote repository. It will not create new branches or change work item status, making it safe to use for resuming existing work.

## Requirements

### Core Functionality

#### Command Interface
- **Command**: `kira open <work-item-id>`
- **Behavior**: Opens a work item that is in "doing" or "review" status by creating/using a worktree and checking out the branch
- **Restrictions**: 
  - Only works with work items in "doing" or "review" status
  - Requires the branch to exist on the remote repository
  - Will not create new branches or change work item status
- **Flags**:
  - `--override` - Remove existing worktree if it exists and recreate it
  - `--no-ide` - Skip opening IDE (useful for agents or automated workflows)
  - `--ide <command>` - Override IDE command (e.g., `--ide cursor`)
  - `--trunk-branch <branch>` - Override trunk branch for validation
  - `--dry-run` - Preview what would be done without executing

#### Work Item Validation
- Validate work item exists and is accessible
- Check work item status is "doing" or "review" (error if other status)
- Derive branch name from work item ID and title (format: `{id}-{kebab-case-title}`)
- Verify branch exists on remote repository (error if not found)
- Support both standalone and polyrepo workspace configurations

#### Worktree Management
- Check if worktree already exists at expected location
- If worktree exists:
  - Verify it's pointing to the correct branch
  - Use existing worktree if valid
  - Optionally recreate if `--override` flag is provided
- If worktree doesn't exist:
  - Create new worktree from trunk branch
  - Checkout the remote branch in the worktree
  - Handle polyrepo configurations by creating worktrees for all related repositories

#### Branch Operations
- Fetch latest changes from remote before checking out
- Checkout branch from remote (format: `{remote}/{branch-name}`)
- Handle cases where branch exists locally but not on remote (error)
- Handle cases where branch exists on remote but not locally (checkout from remote)
- Validate branch name matches expected pattern from work item

#### IDE Integration
- Open IDE in the worktree directory (respects `ide.command` from kira.yml or `--ide` flag)
- Support polyrepo workspaces by opening IDE in the main project directory
- Skip IDE opening if `--no-ide` flag is provided or `ide.command` is not configured
- Use same IDE opening logic as `kira start` command for consistency

#### Polyrepo Support
- Detect polyrepo workspace configuration
- Create/use worktrees for all repositories defined in workspace configuration
- Coordinate branch checkout across all related repositories
- Open IDE in the main project directory of the polyrepo workspace
- Handle repository dependencies and ordering

### Configuration

#### kira.yml Integration
- Use `git.trunk_branch` from kira.yml (defaults to "main" or "master")
- Use `git.remote` from kira.yml (defaults to "origin")
- Use `ide.command` from kira.yml for IDE opening
- Use `worktree.root` from kira.yml for worktree location
- Respect `workspace` configuration for polyrepo support

#### Worktree Location
- Default location: `{worktree_root}/{branch-name}` (same as `kira start`)
- For polyrepo: `{worktree_root}/{branch-name}/main` (main project directory)
- Worktree root defaults to `{repo_root}_worktree` if not configured

### Error Handling

#### Validation Errors
- Work item not found: "Work item {id} not found"
- Invalid status: "Cannot open work item: work item is in {current_status} status. Only 'doing' or 'review' status work items can be opened"
- Branch not on remote: "Branch {branch-name} does not exist on remote {remote}. Cannot open work item without remote branch"
- Invalid branch name: "Branch name '{branch-name}' does not match expected pattern for work item {id}"

#### Git Operations
- Remote not configured: "GitHub remote '{remote}' not configured"
- Fetch failures: "Failed to fetch from remote: {error}"
- Checkout failures: "Failed to checkout branch {branch-name}: {error}"
- Worktree creation failures: "Failed to create worktree: {error}"
- Uncommitted changes: Handle gracefully with clear error message

#### Worktree Conflicts
- Existing worktree with different branch: "Worktree at {path} exists but is on branch {branch}. Use --override to recreate"
- Worktree path conflicts: "Path {path} already exists and is not a git worktree"

### User Experience
- Clear progress messages for each step (validating, fetching, creating worktree, checking out, opening IDE)
- Success message with worktree path and branch name
- Helpful error messages with actionable guidance
- Support for dry-run mode to preview operations
- Consistent behavior with `kira start` command where applicable

## Acceptance Criteria

### Core Command Functionality
- [ ] `kira open <work-item-id>` command exists and is executable
- [ ] Command validates work item exists before proceeding
- [ ] Command only works with work items in "doing" or "review" status
- [ ] Command shows clear error for work items in other statuses
- [ ] Command derives branch name from work item ID and title correctly
- [ ] Command validates branch exists on remote before proceeding
- [ ] Command shows clear error if branch doesn't exist on remote
- [ ] Command creates worktree if it doesn't exist
- [ ] Command reuses existing worktree if it exists and is valid
- [ ] Command checks out branch from remote in the worktree
- [ ] Command opens IDE in worktree directory when configured
- [ ] Command skips IDE opening when `--no-ide` flag is provided
- [ ] Command respects `--ide` flag to override IDE command
- [ ] Command respects `--override` flag to recreate existing worktree
- [ ] Command does not change work item status
- [ ] Command does not create new branches

### Worktree Management
- [ ] Worktree is created at expected location based on configuration
- [ ] Existing worktree is detected and validated correctly
- [ ] `--override` flag removes and recreates existing worktree
- [ ] Worktree points to correct branch after checkout
- [ ] Worktree is properly linked to main repository

### Branch Operations
- [ ] Branch is fetched from remote before checkout
- [ ] Branch is checked out from remote correctly
- [ ] Local branch tracking is set up correctly
- [ ] Branch name validation matches work item pattern
- [ ] Error handling for missing remote branches works correctly

### Polyrepo Support
- [ ] Polyrepo workspace configuration is detected correctly
- [ ] Worktrees are created for all repositories in workspace
- [ ] Branch checkout is coordinated across all repositories
- [ ] IDE opens in main project directory for polyrepo
- [ ] All repositories are validated before operations begin

### Error Scenarios
- [ ] Invalid work item ID shows clear error message
- [ ] Work item in wrong status shows appropriate error
- [ ] Missing remote branch shows clear error with guidance
- [ ] Remote not configured shows helpful error message
- [ ] Worktree path conflicts are handled gracefully
- [ ] Git operation failures show clear error messages
- [ ] Uncommitted changes in worktree are handled appropriately

### Configuration Integration
- [ ] Respects `git.trunk_branch` from kira.yml
- [ ] Respects `git.remote` from kira.yml
- [ ] Respects `ide.command` from kira.yml
- [ ] Respects `worktree.root` from kira.yml
- [ ] Respects `workspace` configuration for polyrepo
- [ ] Command-line flags override configuration correctly

### User Experience
- [ ] Progress messages are clear and informative
- [ ] Success message shows worktree path and branch name
- [ ] Error messages are actionable and helpful
- [ ] Dry-run mode shows what would be done without executing
- [ ] Command output is consistent with `kira start` where applicable

## Implementation Notes

### Architecture

#### Command Structure
```
internal/commands/open.go
├── openCmd - Main cobra command
├── validateWorkItem() - Validate work item exists and is in correct status
├── deriveBranchName() - Extract branch name from work item
├── checkRemoteBranch() - Verify branch exists on remote
├── handleWorktree() - Create or reuse worktree
├── checkoutBranch() - Checkout branch from remote
├── openIDE() - Launch IDE in worktree
└── handlePolyrepo() - Coordinate polyrepo operations
```

#### Dependencies
- Reuse existing worktree management logic from `kira start`
- Reuse IDE opening logic from `kira start`
- Reuse polyrepo coordination from `kira start`
- Use existing git operations utilities
- Leverage existing configuration parsing

### Work Item Validation

#### Status Check
```go
func validateWorkItemStatus(workItem *WorkItem) error {
    validStatuses := []string{"doing", "review"}
    if !contains(validStatuses, workItem.Status) {
        return fmt.Errorf(
            "cannot open work item: work item is in %s status. " +
            "Only 'doing' or 'review' status work items can be opened",
            workItem.Status,
        )
    }
    return nil
}
```

#### Branch Name Derivation
```go
func deriveBranchName(workItem *WorkItem) string {
    // Format: {id}-{kebab-case-title}
    // Reuse logic from kira start command
    title := strings.ToLower(workItem.Title)
    title = strings.ReplaceAll(title, " ", "-")
    // Remove special characters, keep alphanumeric and hyphens
    title = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(title, "")
    return fmt.Sprintf("%s-%s", workItem.ID, title)
}
```

### Remote Branch Validation

#### Check Remote Branch Exists
```go
func checkRemoteBranchExists(branchName, remoteName string) (bool, error) {
    // Fetch from remote first
    if err := fetchFromRemote(remoteName); err != nil {
        return false, fmt.Errorf("failed to fetch from remote: %w", err)
    }

    // Check if branch exists on remote
    remoteBranch := fmt.Sprintf("%s/%s", remoteName, branchName)
    cmd := exec.Command("git", "ls-remote", "--heads", remoteName, branchName)
    output, err := cmd.Output()
    if err != nil {
        return false, fmt.Errorf("failed to check remote branch: %w", err)
    }

    return len(output) > 0, nil
}
```

### Worktree Management

#### Create or Reuse Worktree
```go
func handleWorktree(workItemID, branchName, worktreePath string, override bool) error {
    // Check if worktree exists
    exists, err := worktreeExists(worktreePath)
    if err != nil {
        return err
    }

    if exists {
        if override {
            // Remove existing worktree
            if err := removeWorktree(worktreePath); err != nil {
                return fmt.Errorf("failed to remove existing worktree: %w", err)
            }
            // Create new worktree
            return createWorktree(worktreePath, branchName)
        } else {
            // Validate existing worktree
            currentBranch, err := getWorktreeBranch(worktreePath)
            if err != nil {
                return fmt.Errorf("failed to get worktree branch: %w", err)
            }
            if currentBranch != branchName {
                return fmt.Errorf(
                    "worktree at %s exists but is on branch %s. Use --override to recreate",
                    worktreePath, currentBranch,
                )
            }
            // Worktree is valid, use it
            return nil
        }
    } else {
        // Create new worktree
        return createWorktree(worktreePath, branchName)
    }
}
```

#### Checkout Branch from Remote
```go
func checkoutBranchFromRemote(worktreePath, branchName, remoteName string) error {
    // Change to worktree directory
    originalDir, err := os.Getwd()
    if err != nil {
        return err
    }
    defer os.Chdir(originalDir)

    if err := os.Chdir(worktreePath); err != nil {
        return fmt.Errorf("failed to change to worktree directory: %w", err)
    }

    // Fetch latest changes
    if err := exec.Command("git", "fetch", remoteName).Run(); err != nil {
        return fmt.Errorf("failed to fetch from remote: %w", err)
    }

    // Checkout branch from remote
    remoteBranch := fmt.Sprintf("%s/%s", remoteName, branchName)
    if err := exec.Command("git", "checkout", "-b", branchName, remoteBranch).Run(); err != nil {
        // Branch might already exist locally, try checking out existing branch
        if err := exec.Command("git", "checkout", branchName).Run(); err != nil {
            return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
        }
    }

    // Set upstream tracking
    if err := exec.Command("git", "branch", "--set-upstream-to", remoteBranch, branchName).Run(); err != nil {
        // Non-fatal, continue
    }

    return nil
}
```

### Integration with Existing Code

#### Reuse Start Command Logic
- Leverage `executeStandaloneStart()` and `executePolyrepoStart()` functions where applicable
- Reuse `launchIDE()` function from start command
- Reuse `executeSetupCommands()` if needed (or skip for open command)
- Use same worktree creation utilities
- Share polyrepo discovery logic

#### Differences from Start Command
- No status updates (work item status remains unchanged)
- No branch creation (branch must exist on remote)
- No PR creation
- Simpler validation (only doing/review status)
- Focus on resuming existing work rather than starting new work

## Implementation Strategy

The feature should be implemented incrementally with the following commit progression:

### Implementation Phases

#### Phase 1. CLI Command Structure
```
feat: add kira open command skeleton

- Add basic command registration and CLI interface
- Implement command help and basic argument parsing
- Add placeholder for main command logic
- Update command routing in root.go
- Add command flags (--override, --no-ide, --ide, --trunk-branch, --dry-run)
- Add unit tests for command registration and flag parsing
```

#### Phase 2. Work Item Validation
```
feat: implement work item validation for kira open

- Add work item loading and existence checking
- Implement status validation (only doing/review status allowed)
- Add branch name derivation from work item ID and title
- Create validation error messages
- Add work item context building
- Add unit tests for work item validation (status checks, existence checks)
- Add unit tests for branch name derivation from work items
- Test error scenarios (invalid status, missing work item)
```

#### Phase 3. Remote Branch Checking
```
feat: add remote branch validation for kira open

- Implement remote branch existence checking
- Add git fetch before branch checking
- Create clear error messages for missing remote branches
- Add remote configuration validation
- Handle network errors gracefully
- Add unit tests for remote branch existence checking
- Mock git operations for isolated testing
- Test error scenarios (missing remote, network failures)
```

#### Phase 4. Worktree Management (Standalone)
```
feat: implement worktree management for standalone repos

- Add worktree existence checking
- Implement worktree creation from trunk branch
- Add worktree reuse logic for existing worktrees
- Implement --override flag for worktree recreation
- Add worktree path validation
- Handle worktree path conflicts
- Add unit tests for worktree creation and reuse logic
- Test worktree path validation and conflict handling
- Test --override flag behavior
```

#### Phase 5. Branch Checkout from Remote
```
feat: add branch checkout from remote for kira open

- Implement branch checkout in worktree
- Add upstream tracking setup
- Handle local branch existence scenarios
- Add fetch before checkout
- Create clear error messages for checkout failures
- Add unit tests for branch checkout operations
- Test upstream tracking setup
- Test local branch existence scenarios
- Mock git checkout operations
```

#### Phase 6. IDE Integration
```
feat: integrate IDE opening for kira open

- Reuse launchIDE() function from start command
- Add --no-ide flag support
- Add --ide flag for IDE override
- Respect ide.command from kira.yml
- Open IDE in correct worktree directory
- Add unit tests for IDE flag handling
- Test IDE command resolution (config vs flags)
- Test --no-ide flag behavior
```

#### Phase 7. Polyrepo Support
```
feat: add polyrepo support for kira open

- Detect polyrepo workspace configuration
- Coordinate worktree creation across multiple repositories
- Add branch checkout coordination for all repos
- Open IDE in main project directory
- Handle repository dependencies and validation
- Add unit tests for polyrepo discovery
- Test worktree coordination across multiple repos
- Test branch checkout coordination
- Test repository dependency validation
```

#### Phase 8. Configuration Integration
```
feat: integrate kira configuration for kira open

- Respect git.trunk_branch from kira.yml
- Respect git.remote from kira.yml
- Respect ide.command from kira.yml
- Respect worktree.root from kira.yml
- Add command-line flag overrides
- Add configuration validation
- Add unit tests for configuration parsing
- Test configuration defaults and overrides
- Test command-line flag precedence
```

#### Phase 9. Error Handling & User Experience
```
feat: enhance error handling and UX for kira open

- Add comprehensive error messages for all failure scenarios
- Implement progress messages for each operation step
- Add success message with worktree path and branch
- Add dry-run mode support
- Improve error messages with actionable guidance
- Add consistent output formatting
- Add unit tests for error message formatting
- Test progress message output
- Test dry-run mode behavior
```

#### Phase 10. Integration Tests
```
test: add integration tests for kira open workflows

- Test full workflow with test git repository
- Test worktree creation and branch checkout
- Test polyrepo coordination
- Test IDE opening integration
- Test configuration integration
- Test error recovery scenarios
```

#### Phase 11. E2E Tests
```
test: add e2e tests for kira open command

- Test complete command workflow in test environment
- Verify worktree creation and branch checkout
- Verify IDE opening
- Test error scenarios (missing branch, wrong status, etc.)
- Test polyrepo scenarios
- Test all command flags
```

### Benefits of This Approach

- **Incremental Development**: Each commit adds working functionality
- **Easy Review**: Smaller, focused changes are easier to review
- **Quick Feedback**: Issues can be caught early in smaller chunks
- **Logical Dependencies**: Each commit builds on the previous ones
- **Test-Driven Development**: Unit tests are part of each atomic commit, ensuring features are tested as they're built
- **Revert Safety**: Issues can be isolated to specific commits
- **Reuse Existing Code**: Leverages worktree and IDE logic from `kira start` command
- **Atomic Commits**: Each commit includes both implementation and its corresponding tests

### Testing Strategy

#### Unit Tests (Integrated into Each Phase)
Unit tests are written as part of each implementation phase, ensuring that:
- Each feature is tested immediately as it's implemented
- Tests and implementation are in the same atomic commit
- Test coverage grows incrementally with functionality
- Issues are caught early before moving to the next phase

Unit tests cover:
- Work item validation (status checks, existence checks)
- Branch name derivation from work items
- Remote branch existence checking
- Worktree creation and reuse logic
- Branch checkout from remote
- Mock git operations for isolated testing
- Error scenarios and edge cases
- Configuration parsing and flag handling
- IDE integration logic
- Polyrepo coordination

#### Integration Tests
- Full workflow testing with test git repository
- Test worktree creation and branch checkout
- Test polyrepo coordination
- Test IDE opening integration
- Test configuration integration

#### E2E Tests
- `kira open` command in test environment
- Verify worktree creation and branch checkout
- Verify IDE opening
- Test error scenarios (missing branch, wrong status, etc.)
- Test polyrepo scenarios

### Security Considerations

#### Input Validation
- Validate work item ID format
- Sanitize branch names
- Validate file paths for worktree creation
- Check for path traversal vulnerabilities

#### Git Operations
- Validate remote URLs before operations
- Handle credentials securely
- Avoid command injection in git commands
- Validate worktree paths are within expected directories

## Release Notes

### New Features
- **Open Command**: New `kira open <work-item-id>` command for resuming work on existing work items
- **Smart Worktree Management**: Automatically creates or reuses worktrees for work items in progress
- **Remote Branch Integration**: Seamlessly checks out branches from remote repositories
- **Status-Aware**: Only works with work items in "doing" or "review" status for safety
- **Polyrepo Support**: Coordinates worktree creation and branch checkout across multiple repositories

### Improvements
- Streamlined workflow for resuming work on existing items
- No status changes - work item status remains unchanged when opening
- No branch creation - reuses existing branches from remote
- Consistent behavior with `kira start` for IDE opening and worktree management
- Clear error messages for common scenarios (missing branch, wrong status, etc.)

### Technical Changes
- New command structure following established Kira patterns
- Reuses existing worktree and IDE opening logic from `kira start`
- Extends git operations utilities for remote branch checking
- Integrates with existing configuration system

