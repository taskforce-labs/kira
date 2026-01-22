---
id: 014
title: kira-done
status: backlog
kind: prd
assigned:
estimate:
created: 2026-01-21
due: 2026-01-21
tags: [github, merge, done, cli]
---

# kira-done

A command that completes the work item workflow by rebasing the current branch onto the trunk branch, merging the associated pull request on GitHub, and marking the work item as done. This command automates the final step in the Kira workflow, ensuring work items are properly closed and merged.

## Context

In the Kira workflow, work items progress through different statuses: backlog → todo → doing → review → done. After a work item has been reviewed and approved (via `kira review`), it needs to be merged and marked as complete. Currently, this requires manual git operations and GitHub PR merging, which creates friction in the development workflow.

The `kira done` command streamlines this final step by:

1. Automatically deriving the work item ID from the current branch name
2. Rebasing the current branch onto the trunk branch to ensure it's up-to-date
3. Merging the associated pull request on GitHub using the configured merge strategy
4. Updating the work item status to "done" on both the feature branch and trunk branch
5. Cleaning up the feature branch after successful merge (optional)
6. Supporting only individual work items

This approach ensures that completed work items are properly merged, tracked, and archived, maintaining a clean git history and accurate project state. The command treats the trunk branch as the authoritative source of truth for work item status, ensuring consistency across the repository.

**Dependencies**: This command requires:
- An existing pull request (created via `kira review`)
- GitHub API access for PR merging
- The current branch to follow kira naming conventions

## Requirements

### Core Functionality

#### Command Interface
- **Command**: `kira done [--merge-strategy <strategy>] [--no-cleanup] [--no-trunk-update]`
- **Behavior**: Automatically derives work item ID from current branch name (must be on a kira-created branch)
- **Restrictions**: Cannot be run on trunk branch or non-kira branches
- **Flags**:
  - `--merge-strategy <strategy>` - Specify merge strategy (merge, squash, rebase). Defaults to config or "rebase"
  - `--no-cleanup` - Skip deleting the feature branch after merge (overrides config)
  - `--no-trunk-update` - Skip updating trunk branch status (overrides config)
  - `--no-rebase` - Skip rebasing current branch before merge (overrides config)
  - `--force` - Force merge even if PR has failing checks (use with caution)
  - `--dry-run` - Preview what would be done without executing

#### Work Item Management

- Derive work item ID from current branch name (format: `{id}-{title}`)
- Validate current branch was created by `kira start` (not trunk branch)
- Validate work item exists and is accessible
- Support work items in "review" status (error for other statuses)
- Move derived work item from "review" to "done" on the current branch
- Optionally update the work item status to "done" on the trunk branch (when enabled)
- Update work item front matter with completion metadata (merged date, merge commit SHA, etc.)
- Validate that work item has required fields before completion
- Handle merge conflicts during rebase with user guidance

#### Git Operations

- Fetch latest changes from remote before operations
- Rebase the current branch onto the trunk branch (when enabled)
- Ensure branch is up-to-date before merging
- Handle rebase conflicts with clear user guidance
- Validate that all commits are pushed to remote before merge
- Support polyrepo workflows by coordinating operations across multiple repositories

#### GitHub Integration

- Find associated pull request using branch name or work item metadata
- Validate PR exists and is in mergeable state
- Check PR status (must be approved/ready, not draft)
- Optionally check for required status checks (unless `--force` is used)
- Merge pull request using specified strategy (merge, squash, or rebase)
- Use git.trunk_branch as the base branch (from existing kira.yml git configuration)
- Use git.remote for repository identification (from existing kira.yml git configuration)
- Update PR with merge information
- Optionally delete feature branch after successful merge (when cleanup is enabled)
- Handle GitHub API rate limiting gracefully

### Configuration

#### kira.yml Extensions
```yaml
done:
  update_trunk_status: true   # Update work item status on trunk branch as source of truth
  rebase_before_merge: true   # Rebase current branch before merging PR
  cleanup_branch: true        # Delete feature branch after successful merge
  merge_strategy: "rebase"    # Options: merge, squash, rebase
  require_checks: true        # Require all status checks to pass before merge
  merge_commit_message: "Merge {id}: {title}"
  squash_commit_message: "{id}: {title}"
```

### Branch and PR Management

#### Branch Requirements
- Branch must already exist (created by `kira start`)
- Branch must follow naming convention: `{id}-{kebab-case-title}`
- Branch must be pushed to remote
- Branch must be up-to-date with trunk (or rebased before merge)

#### PR Merge Logic
1. Verify branch exists on remote
2. Fetch latest changes from remote
3. Rebase branch onto trunk (when enabled)
4. Validate PR exists and is mergeable
5. Check PR status (approved, checks passing)
6. Merge PR using specified strategy
7. Update work item status to "done"
8. Optionally delete feature branch

### Error Handling

#### Validation Errors
- Work item not found: "Work item {id} not found"
- Invalid status: "Cannot mark as done: work item is in {current_status} status. Only 'review' status work items can be marked as done"
- PR not found: "No pull request found for branch {branch-name}"
- PR not mergeable: "Pull request #{number} is not in a mergeable state"
- Branch not on remote: "Branch {branch-name} must be pushed to remote before merging"
- GitHub token missing: "GitHub token required for PR merge"

#### Git Operations
- Remote not found: "GitHub remote '{remote}' not configured"
- Rebase conflicts: Guide user to resolve conflicts before retrying
- Merge failures: Provide clear error messages with resolution steps
- Uncommitted changes: "Uncommitted changes detected. Please commit or stash before running 'kira done'"

#### GitHub API Errors
- Rate limiting: Implement backoff and retry logic
- API errors: Parse and provide user-friendly messages
- Token issues: Clear guidance on token setup and permissions
- PR status checks: Clear messages about failing checks and how to proceed

### Polyrepo Support

When working with polyrepo configurations, the command must:
- Coordinate rebase operations across all related repositories
- Ensure all repositories are up-to-date before merging
- Merge PRs in the correct order based on dependencies
- Update work item status consistently across all repositories
- Handle failures gracefully with rollback capabilities

## Acceptance Criteria

### Core Command Functionality
- [ ] `kira done` automatically derives work item ID from current branch name
- [ ] Command fails if run on trunk branch
- [ ] Command fails if current branch doesn't follow kira naming convention
- [ ] Command fails if work item is not in "review" status
- [ ] `kira done` rebases current branch onto trunk branch when enabled
- [ ] `kira done --no-rebase` skips rebase before merge
- [ ] `kira done` finds and merges associated pull request
- [ ] `kira done` updates work item status to "done" on current branch
- [ ] `kira done` updates work item status to "done" on trunk branch when enabled
- [ ] `kira done --no-trunk-update` skips trunk status updates
- [ ] `kira done --merge-strategy squash` uses squash merge strategy
- [ ] `kira done --merge-strategy rebase` uses rebase merge strategy
- [ ] `kira done --merge-strategy merge` uses merge commit strategy (default)
- [ ] Command fails gracefully when GitHub token is missing
- [ ] Command shows helpful error messages for invalid work items
- [ ] Command validates PR is mergeable before attempting merge

### GitHub Integration
- [ ] Finds PR associated with current branch
- [ ] Validates PR exists and is accessible
- [ ] Checks PR status (approved, not draft)
- [ ] Validates PR is mergeable
- [ ] Optionally checks for passing status checks (unless --force)
- [ ] Merges PR using specified strategy
- [ ] Updates PR with merge information
- [ ] Optionally deletes feature branch after merge (when cleanup enabled)
- [ ] `kira done --no-cleanup` preserves feature branch after merge
- [ ] Handles GitHub API rate limiting gracefully

### Work Item Updates
- [ ] Work item status changes to "done"
- [ ] Front matter includes merge metadata (merged date, merge commit SHA, PR number)
- [ ] Updated timestamp reflects completion time
- [ ] Work item remains in git history with full traceability
- [ ] Trunk branch status updates work correctly when enabled

### Rebase and Merge Functionality
- [ ] Rebase operations complete successfully before merge
- [ ] Command handles rebase conflicts with clear user guidance
- [ ] Merge operations complete successfully
- [ ] Merge commit messages follow configured templates
- [ ] Squash merge creates single commit with work item title
- [ ] Rebase merge maintains linear history
- [ ] Trunk branch configuration respects custom branch names
- [ ] Disabling rebase via config works correctly
- [ ] Override flags (--no-trunk-update, --no-rebase, --no-cleanup) work correctly

### Error Scenarios
- [ ] Invalid work item ID shows clear error message
- [ ] Work item in wrong status shows appropriate error
- [ ] PR not found shows clear error with guidance
- [ ] PR not mergeable shows appropriate error message
- [ ] Rebase conflicts provide clear resolution instructions
- [ ] GitHub API errors are handled gracefully
- [ ] Network failures retry with exponential backoff
- [ ] Uncommitted changes prevent execution with clear message
- [ ] Failing status checks prevent merge (unless --force)

### Polyrepo Support
- [ ] Polyrepo workspace configuration is detected correctly
- [ ] Rebase operations are coordinated across all repositories
- [ ] PR merges are coordinated across all repositories
- [ ] Work item status updates are consistent across all repositories
- [ ] Failures in one repository are handled gracefully
- [ ] All repositories are validated before operations begin

## Implementation Notes

### Architecture

#### Command Structure
```
internal/commands/done.go
├── doneCmd - Main cobra command
├── deriveWorkItemFromBranch() - Extract work item ID from current branch name
├── validateBranchContext() - Ensure command is run on valid kira branch
├── validateWorkItemStatus() - Ensure work item is in review status
├── findPullRequest() - Find associated PR for current branch
├── validatePRStatus() - Check PR is mergeable
├── rebaseBranch() - Rebase current branch onto trunk
├── mergePullRequest() - GitHub API integration for PR merge
├── updateWorkItemStatus() - Update work item to done status
├── cleanupBranch() - Delete feature branch after merge
└── updateTrunkStatus() - Update work item status on trunk branch
```

#### Dependencies
- **GitHub**: `github.com/google/go-github/v58/github` for API client
- **Git**: Standard git operations for rebase and branch management
- **YAML**: Extended config parsing for done settings

### GitHub API Integration

#### Authentication
```go
ctx := context.Background()
ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
tc := oauth2.NewClient(ctx, ts)
client := github.NewClient(tc)
```

#### PR Finding
```go
// Find PR by branch name
prs, _, err := client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
    State: "open",
    Head: fmt.Sprintf("%s:%s", owner, branchName),
})
```

#### PR Merging
```go
mergeOpts := &github.PullRequestOptions{
    MergeMethod: "merge", // or "squash", "rebase"
    CommitTitle: commitTitle,
    CommitMessage: commitMessage,
}

result, _, err := client.PullRequests.Merge(ctx, owner, repo, prNumber, commitMessage, mergeOpts)
```

#### Branch Deletion
```go
_, err := client.Git.DeleteRef(ctx, owner, repo, fmt.Sprintf("heads/%s", branchName))
```

### Work Item Processing

#### Status Validation
```go
validStatuses := []string{"review"}
if !contains(validStatuses, currentStatus) {
    return fmt.Errorf("work item must be in review status")
}
```

#### Work Item Derivation
```go
func deriveWorkItemFromBranch(currentBranch string) (string, error) {
    // Branch format: {id}-{title} (e.g., "001-user-authentication")
    // Extract ID from beginning of branch name
    dashIndex := strings.Index(currentBranch, "-")
    if dashIndex == -1 {
        return "", fmt.Errorf("branch name '%s' does not follow kira naming convention", currentBranch)
    }

    workItemID := currentBranch[:dashIndex]
    if !regexp.MustCompile(`^\d{3}$`).MatchString(workItemID) {
        return "", fmt.Errorf("invalid work item ID '%s' in branch name", workItemID)
    }

    return workItemID, nil
}
```

#### Trunk Status Update Process
```go
func updateTrunkStatus(workItemID string, cfg *config.Config) error {
    // Get trunk branch from existing git configuration
    trunkBranch := cfg.Git.TrunkBranch

    // Stash any uncommitted changes
    stashOutput, _ := exec.Command("git", "stash").Output()

    // Switch to trunk branch
    if err := exec.Command("git", "checkout", trunkBranch).Run(); err != nil {
        return fmt.Errorf("failed to checkout trunk branch '%s': %w", trunkBranch, err)
    }

    // Update work item status
    if err := moveWorkItem(workItemID, "done", true); err != nil {
        return fmt.Errorf("failed to update trunk status: %w", err)
    }

    // Commit and push status change
    if err := commitAndPushStatusChange(workItemID); err != nil {
        return fmt.Errorf("failed to commit status change: %w", err)
    }

    // Switch back to original branch (if it still exists)
    exec.Command("git", "checkout", "-").Run()

    // Restore stashed changes if any
    if stashOutput != nil {
        exec.Command("git", "stash", "pop")
    }

    return nil
}
```

#### Rebase Process
```go
func rebaseBranch(branchName, trunkBranch string) error {
    // Fetch latest changes
    if err := exec.Command("git", "fetch", "origin", trunkBranch).Run(); err != nil {
        return fmt.Errorf("failed to fetch trunk branch: %w", err)
    }

    // Perform rebase
    cmd := exec.Command("git", "rebase", fmt.Sprintf("origin/%s", trunkBranch))
    if err := cmd.Run(); err != nil {
        // Check if rebase failed due to conflicts
        if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
            return fmt.Errorf("rebase failed due to conflicts. Please resolve conflicts and run 'git rebase --continue', then retry 'kira done'")
        }
        return fmt.Errorf("rebase failed: %w", err)
    }

    // Force push rebased branch
    if err := exec.Command("git", "push", "--force-with-lease", "origin", branchName).Run(); err != nil {
        return fmt.Errorf("failed to push rebased branch: %w", err)
    }

    return nil
}
```

#### Metadata Updates
```go
frontMatter := map[string]interface{}{
    "status": "done",
    "updated": time.Now().Format(time.RFC3339),
    "merged_at": time.Now().Format(time.RFC3339),
    "merge_commit_sha": mergeCommitSHA,
    "pr_number": prNumber,
    "merge_strategy": mergeStrategy,
}
```

### Implementation Strategy

The feature should be implemented incrementally with the following commit progression. Each phase includes unit tests integrated into the implementation, with integration and E2E tests as separate phases.

#### Phase 1. Refactor: Extract Shared Branch Utilities
```
refactor: extract branch derivation and validation utilities

- Extract deriveWorkItemFromBranch() from PRD spec into utils.go
  - Parse branch name format: {id}-{title}
  - Validate ID format (3 digits)
  - Return work item ID or error
- Extract validateBranchContext() into utils.go
  - Check branch is not trunk branch
  - Validate branch follows kira naming convention
  - Reusable for both kira review and kira done commands
- Add unit tests for branch derivation (valid/invalid formats, edge cases)
- Add unit tests for branch validation (trunk rejection, invalid names)
- Update existing code that could benefit from these utilities
```

**Refactoring Opportunity**: These utilities will be needed by both `kira review` (PRD 012) and `kira done` commands. Extract them first to avoid duplication.

#### Phase 2. Refactor: Extract Trunk Status Update Utilities
```
refactor: extract trunk status update utilities

- Extract updateTrunkStatus() pattern into utils.go
  - Stash uncommitted changes
  - Checkout trunk branch
  - Update work item status using moveWorkItem()
  - Commit and push status change
  - Restore original branch and stashed changes
- Make it reusable with parameters: workItemID, targetStatus, cfg
- Add unit tests for trunk status update flow
- Test error scenarios (checkout failures, commit failures, etc.)
- Test with dry-run mode
```

**Refactoring Opportunity**: The `kira review` PRD (012) also needs trunk status updates. Extract this pattern to share between commands.

#### Phase 3. Refactor: Extract Rebase Utilities
```
refactor: extract git rebase utilities

- Extract rebaseBranch() into utils.go
  - Fetch latest trunk branch
  - Perform rebase operation
  - Handle rebase conflicts with clear error messages
  - Force push rebased branch (with --force-with-lease)
- Add unit tests for rebase operations
- Test conflict detection and error messages
- Test force push behavior
- Integrate with existing executeCommand() utility
```

**Refactoring Opportunity**: Rebase logic will be shared between `kira review` and `kira done`. Extract to avoid duplication.

#### Phase 4. CLI Command Structure
```
feat: add kira done command skeleton

- Add basic command registration and CLI interface
- Implement command help and basic argument parsing
- Add placeholder for main command logic
- Update command routing in root.go
- Add command flags (--merge-strategy, --no-cleanup, --no-trunk-update, --no-rebase, --force, --dry-run)
- Add unit tests for command registration and flag parsing
- Test help output and flag validation
```

#### Phase 5. Work Item Derivation and Validation
```
feat: implement work item derivation and validation for kira done

- Use refactored deriveWorkItemFromBranch() utility
- Use refactored validateBranchContext() utility
- Validate work item exists and is accessible
- Implement validateWorkItemStatus() - only "review" status allowed
- Create validation error messages
- Add work item context building
- Add unit tests for work item validation (status checks, existence checks)
- Test error scenarios (invalid status, missing work item, wrong branch)
```

#### Phase 6. Configuration Schema Extension
```
feat: extend configuration schema for done command

- Add DoneConfig struct to config package
- Add done section to kira.yml schema
- Support merge_strategy, update_trunk_status, rebase_before_merge, cleanup_branch, require_checks
- Add merge commit message templates
- Add configuration validation
- Add unit tests for configuration parsing
- Test configuration defaults and overrides
- Test command-line flag precedence over config
```

#### Phase 7. GitHub API Client Setup
```
feat: add GitHub API client utilities

- Create github.go utility file for GitHub API operations
- Implement GitHub client initialization with OAuth2
- Add token validation and error handling
- Implement repository owner/repo extraction from git remote
- Add rate limiting and retry logic
- Add unit tests for client initialization
- Mock GitHub API for testing
- Test token validation and error scenarios
```

**Refactoring Opportunity**: This will be shared with `kira review` and future `kira start --draft-pr` features. Extract early.

#### Phase 8. Pull Request Finding and Validation
```
feat: implement PR finding and validation for kira done

- Implement findPullRequest() using GitHub API
  - Find PR by branch name
  - Validate PR exists and is accessible
- Implement validatePRStatus()
  - Check PR is not draft
  - Check PR is mergeable
  - Optionally check status checks (unless --force)
- Add clear error messages for invalid PR states
- Add unit tests for PR finding logic
- Mock GitHub API responses for different PR states
- Test error scenarios (PR not found, not mergeable, failing checks)
```

#### Phase 9. Rebase Before Merge
```
feat: implement rebase before merge for kira done

- Use refactored rebaseBranch() utility
- Check for uncommitted changes before rebase
- Fetch latest trunk branch
- Perform rebase operation
- Handle rebase conflicts with user guidance
- Force push rebased branch
- Add unit tests for rebase flow
- Test conflict handling
- Test --no-rebase flag behavior
```

#### Phase 10. Pull Request Merging
```
feat: implement GitHub PR merging for kira done

- Implement mergePullRequest() using GitHub API
- Support merge strategies: merge, squash, rebase
- Generate merge commit messages from templates
- Handle merge failures gracefully
- Extract merge commit SHA for work item metadata
- Add unit tests for PR merge operations
- Mock GitHub API for different merge strategies
- Test error scenarios (merge failures, API errors)
```

#### Phase 11. Work Item Status Update
```
feat: implement work item status update to done

- Update work item status to "done" on current branch
- Use refactored updateTrunkStatus() utility when enabled
- Update work item front matter with merge metadata:
  - merged_at timestamp
  - merge_commit_sha
  - pr_number
  - merge_strategy
- Commit status change on feature branch
- Add unit tests for work item updates
- Test metadata extraction and updates
- Test trunk status update integration
```

#### Phase 12. Branch Cleanup
```
feat: implement feature branch cleanup after merge

- Implement deleteBranch() using GitHub API
- Delete feature branch after successful merge (when cleanup enabled)
- Handle branch deletion failures gracefully
- Add unit tests for branch cleanup
- Test --no-cleanup flag behavior
- Test cleanup error scenarios
```

#### Phase 13. Error Handling and User Experience
```
feat: enhance error handling and UX for kira done

- Add comprehensive error messages for all failure scenarios
- Implement progress messages for each operation step
- Add success message with merge information
- Improve error messages with actionable guidance
- Add consistent output formatting
- Add unit tests for error message formatting
- Test progress message output
- Test error recovery scenarios
```

#### Phase 14. Integration Tests
```
test: add integration tests for kira done workflows

- Test full workflow with test GitHub repository
- Test PR merge with different strategies (merge, squash, rebase)
- Test branch cleanup verification
- Test trunk status update verification
- Test rebase conflict resolution flow
- Test error recovery scenarios
- Test with different PR states (approved, failing checks, etc.)
```

#### Phase 15. E2E Tests
```
test: add e2e tests for kira done command

- Test complete command workflow in test environment
- Verify PR merge and work item updates
- Test branch cleanup
- Test error scenarios (missing PR, wrong status, etc.)
- Test all command flags and configurations
- Test polyrepo scenarios (if applicable)
- Test with real GitHub repository (optional, may require test account)
```

### Testing Strategy

#### Unit Tests (Integrated into Each Phase)
Unit tests are written as part of each implementation phase, ensuring that:
- Each feature is tested immediately as it's implemented
- Tests and implementation are in the same atomic commit
- Test coverage grows incrementally with functionality
- Issues are caught early before moving to the next phase

Unit tests cover:
- Work item ID derivation from branch names
- Branch validation (trunk branch rejection, invalid formats)
- PR finding and validation logic
- Mock GitHub API responses for PR merge
- Trunk status update functionality
- Rebase operations and conflict handling
- Work item metadata updates
- Error scenarios and edge cases
- Configuration parsing and flag handling

#### Integration Tests (Phase 14)
- Full workflow testing with test GitHub repository
- PR merge testing with different strategies
- Branch cleanup verification
- Trunk status update verification
- Rebase conflict resolution flow

#### E2E Tests (Phase 15)
- `kira done` command in test environment
- Verify PR merge and work item updates
- Test branch cleanup
- Test error scenarios (missing PR, wrong status, etc.)
- Test all command flags and configurations

### Security Considerations

#### Token Management
- Never log or expose GitHub tokens
- Use environment variables for sensitive config
- Validate token permissions before operations

#### Input Validation
- Sanitize branch names
- Validate PR numbers
- Escape special characters in commit messages

#### API Rate Limiting
- Implement intelligent backoff for GitHub API calls
- Cache repository information where possible
- Provide clear error messages for rate limit issues

#### Merge Safety
- Validate PR is approved before merging
- Check for required status checks (unless --force)
- Prevent accidental merges of unapproved PRs
- Require explicit confirmation for destructive operations

## Release Notes

### New Features
- **Done Command**: New `kira done` command that automatically derives work item from current branch for streamlined PR merging
- **Smart Context Detection**: Automatically identifies work items from kira-created branch names
- **Branch Validation**: Prevents accidental execution on trunk branches or invalid branches
- **Trunk Status Updates**: Optional updates to trunk branch status for maintaining source of truth (configurable)
- **Automatic Rebasing**: Seamless rebase of feature branch before merging
- **GitHub Integration**: Seamless integration with GitHub for PR merging and branch cleanup
- **Merge Strategy Support**: Configurable merge strategies (merge, squash, rebase)

### Improvements
- Streamlined command interface - no need to specify work item IDs manually
- Smart context awareness - automatically detects work items from branch names
- Enhanced safety - prevents execution on trunk branches
- Enhanced work item metadata tracking for completion processes
- Configurable trunk branch status updates (can be disabled if not desired)
- Improved error messages and user guidance for merge workflows and rebasing
- Automatic branch cleanup after successful merge (configurable)

### Technical Changes
- Added GitHub API client dependency for PR merge management
- Extended configuration schema for done and merge settings
- New command structure following established Kira patterns
- Integration with existing git operations utilities
