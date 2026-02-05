---
id: 014
title: kira-done
status: doing
kind: prd
assigned:
estimate:
created: 2026-01-21
due: 2026-01-21
tags: [github, merge, done, cli]
---

# kira-done

A command that completes the work item workflow by merging the associated pull request on GitHub (if still open), pulling the latest trunk, updating the work item to "done" on trunk, and optionally cleaning up the feature branch. It runs only on the trunk branch and is idempotent so it can be re-run after failures or when the work item is already done.

## Context

In the Kira workflow, work items progress through different statuses: backlog → todo → doing → review → done. After a work item has been reviewed and approved (via `kira review`), it needs to be merged and marked as complete. Currently, this requires manual git operations and GitHub PR merging, which creates friction in the development workflow.

The `kira done` command streamlines this final step by:

1. Running only when the current branch is the trunk branch (so the feature branch can be removed after merge)
2. Taking the work item ID as an argument (e.g. `kira done 014`) since there is no feature branch to derive it from
3. Finding the associated pull request and, if still open: ensuring all PR checks pass and open comments are resolved (when possible), then merging using the configured strategy
4. If the PR is already closed (merged), continuing with the remaining steps without failing
5. After merge (or when already merged): pulling the latest trunk from remote
6. Updating the work item status to "done" on the trunk branch and recording completion metadata
7. Optionally deleting the feature branch after successful merge
8. Being idempotent: safe to run again if already done or after correcting a prior failure

This approach ensures that completed work items are properly merged, tracked, and archived, and that branch cleanup is possible (you cannot delete the branch you are on, so running from trunk is required).

**Dependencies**: This command requires:
- An existing pull request (created via `kira review`), or an already-merged PR for the work item
- GitHub API access for PR merge and status/comment checks
- Current branch must be the configured trunk branch

## Requirements

### Core Functionality

#### Command Interface
- **Command**: `kira done <work-item-id> [--merge-strategy <strategy>] [--no-cleanup] [--force] [--dry-run]`
- **Behavior**: Completes the work item by merging its PR (if open), pulling trunk, updating status to "done" on trunk, and optionally cleaning up the feature branch. Work item ID is required (e.g. `kira done 014`).
- **Restrictions**: Must be run on the trunk branch only. Running on a feature branch would prevent deleting that branch after merge; trunk is required so cleanup can remove the feature branch.
- **Idempotency**: Safe to run again if the work item is already done or after fixing a prior failure. Completed steps are skipped; remaining steps (e.g. pull trunk, update status, cleanup) are performed as needed.
- **Flags**:
  - `--merge-strategy <strategy>` - Specify merge strategy (merge, squash, rebase). Defaults to config or "rebase"
  - `--no-cleanup` - Skip deleting the feature branch after merge (overrides config)
  - `--force` - Force merge even if PR has failing checks or unresolved comments (use with caution)
  - `--dry-run` - Preview what would be done without executing

#### Work Item Management

- Accept work item ID as required argument (e.g. `014`)
- Validate current branch is the configured trunk branch; fail with clear message if not
- Validate work item exists and is accessible
- Allow work items in "review" (merge PR then mark done) or already "done" (idempotent: skip merge, perform any missing steps such as pull trunk or status update)
- Update the work item status to "done" on the trunk branch and set completion metadata (merged date, merge commit SHA, etc.)
- Validate that work item has required fields before completion
- When idempotent run and already done: pull trunk if needed, ensure status/metadata are correct, optionally ensure branch cleanup; do not fail

#### Git Operations

- Validate that current branch is trunk; fail otherwise
- After PR is merged (or detected as already merged): pull latest from the trunk branch on remote so local trunk is up to date before status updates and cleanup
- Fetch latest changes from remote as needed for PR checks and merge
- Support polyrepo workflows by coordinating operations across multiple repositories

#### GitHub Integration

- Find associated pull request using work item ID (branch name pattern `{id}-*` or work item metadata)
- If PR is **open**: run all required status checks on the PR before merge; if possible via API, ensure all open review comments are resolved before merge (unless `--force`)
- If PR is **already closed (merged)**: do not fail; continue with next steps (pull trunk, update work item status, cleanup)
- Validate PR is in mergeable state when open
- Check PR status (must be approved/ready, not draft) when open
- Require all status checks to pass before merge (unless `--force`)
- Merge pull request using specified strategy when PR is open
- Use git.trunk_branch and git.remote from kira.yml
- Optionally delete feature branch after successful merge (when cleanup is enabled)
- Handle GitHub API rate limiting gracefully

### Configuration

#### kira.yml Extensions
```yaml
done:
  cleanup_branch: true        # Delete feature branch after successful merge
  cleanup_worktree: true      # Delete worktree after successful merge (when applicable)
  merge_strategy: "rebase"    # Options: merge, squash, rebase
  require_checks: true        # Require all PR status checks to pass before merge
  require_comments_resolved: true  # When possible via API, require open review comments resolved before merge
  merge_commit_message: "{id} merge: {title}"
  squash_commit_message: "{id}: {title}"
```

### Branch and PR Management

#### Branch Requirements
- **Current branch must be trunk**. Command fails with a clear message if run on a feature branch (so that cleanup can delete the feature branch after merge).
- Feature branch for the work item must exist on remote (or already be merged and deleted) so the PR can be found or merge state determined.

#### PR Merge Logic
1. Ensure current branch is trunk; fail if not.
2. Resolve work item ID from argument and find associated PR (by branch name pattern or metadata).
3. **If PR is already closed (merged)**: skip merge; continue to step 6.
4. **If PR is open**: run all required status checks on the PR; if possible via API, ensure all open review comments are resolved (unless `--force`). Validate PR is mergeable and approved.
5. Merge PR using specified strategy (when PR was open).
6. **Pull latest from trunk** on remote so local trunk is up to date.
7. Update work item status to "done" on trunk and set completion metadata.
8. Optionally delete feature branch (when cleanup is enabled).
9. Idempotent: if work item is already done or PR already merged, perform only any remaining steps (e.g. pull trunk, status update, cleanup).

### Error Handling

#### Validation Errors
- Not on trunk: "Cannot run 'kira done' on a feature branch. Check out the trunk branch ({trunk}) first so the feature branch can be removed after merge."
- Work item not found: "Work item {id} not found"
- PR not found (when not already merged): "No pull request found for work item {id}. Ensure the branch exists and a PR is open, or that the PR was already merged."
- PR not mergeable: "Pull request #{number} is not in a mergeable state"
- PR checks failing: "Required status checks have not passed on PR #{number}. Fix failures and run 'kira done' again, or use --force (use with caution)."
- PR has unresolved comments: "Pull request #{number} has unresolved review comments. Resolve them and run 'kira done' again, or use --force (use with caution)."
- GitHub token missing: "GitHub token required for PR merge"

#### Git Operations
- Remote not found: "GitHub remote '{remote}' not configured"
- Merge failures: Provide clear error messages with resolution steps
- Pull trunk failures: Clear message and guidance to retry after fixing (idempotent: user can fix and run again)

#### GitHub API Errors
- Rate limiting: Implement backoff and retry logic
- API errors: Parse and provide user-friendly messages
- Token issues: Clear guidance on token setup and permissions
- PR status/comment checks: Clear messages about failing checks or unresolved comments and how to proceed (or use --force)

### Polyrepo Support

When working with polyrepo configurations, the command must:
- Ensure trunk branch is current in all related repositories before and after merge
- Pull trunk in each repository after PRs are merged
- Merge PRs in the correct order based on dependencies (when open)
- Update work item status consistently across all repositories
- Handle failures gracefully with rollback capabilities

## Acceptance Criteria

### Core Command Functionality
- [ ] `kira done <work-item-id>` requires work item ID as argument (e.g. `kira done 014`)
- [ ] Command fails with clear message if not run on trunk branch (must run on trunk so feature branch can be removed)
- [ ] Command finds PR associated with work item (by branch name pattern or metadata)
- [ ] When PR is open: all required status checks are run and must pass before merge (unless `--force`)
- [ ] When possible via API: open review comments must be resolved before merge (unless `--force`)
- [ ] When PR is already closed (merged): command continues with next steps (pull trunk, update status, cleanup) without failing
- [ ] After merge (or when already merged): command pulls latest from trunk on remote so local trunk is up to date
- [ ] `kira done` updates work item status to "done" on trunk and sets completion metadata
- [ ] `kira done --merge-strategy squash` uses squash merge strategy (when PR is open)
- [ ] `kira done --merge-strategy rebase` uses rebase merge strategy (when PR is open)
- [ ] `kira done --merge-strategy merge` uses merge commit strategy (when PR is open)
- [ ] Command is idempotent: safe to run again when work item is already done or after fixing a prior failure; completed steps are skipped
- [ ] Command fails gracefully when GitHub token is missing
- [ ] Command shows helpful error messages for invalid work items and when not on trunk
- [ ] Command validates PR is mergeable before attempting merge (when PR is open)

### GitHub Integration
- [ ] Finds PR associated with work item ID (branch name pattern or metadata)
- [ ] When PR is open: validates PR exists, is mergeable, approved, not draft
- [ ] When PR is open: runs all required status checks and blocks merge until passing (unless `--force`)
- [ ] When possible via API: blocks merge until open review comments are resolved (unless `--force`)
- [ ] When PR is already merged: continues without error to pull trunk, update status, cleanup
- [ ] Merges PR using specified strategy when PR is open
- [ ] Optionally deletes feature branch after merge (when cleanup enabled)
- [ ] `kira done --no-cleanup` preserves feature branch after merge
- [ ] Handles GitHub API rate limiting gracefully

### Work Item Updates
- [ ] Work item status changes to "done" on trunk
- [ ] Front matter includes merge metadata (merged date, merge commit SHA, PR number)
- [ ] Updated timestamp reflects completion time
- [ ] Work item remains in git history with full traceability
- [ ] Idempotent run when already done: ensures status/metadata correct, pulls trunk if needed

### Merge and Pull Flow
- [ ] Merge operations complete successfully when PR is open
- [ ] After merge (or when PR already merged): pull from trunk on remote is performed
- [ ] Merge commit messages follow configured templates
- [ ] Squash merge creates single commit with work item title
- [ ] Rebase merge maintains linear history
- [ ] Trunk branch configuration respected; command only runs on trunk
- [ ] Override flag --no-cleanup works correctly

### Error Scenarios
- [ ] Not on trunk shows clear error: must checkout trunk first so branch can be removed
- [ ] Invalid work item ID shows clear error message
- [ ] PR not found shows clear error with guidance (or that PR may already be merged)
- [ ] PR not mergeable shows appropriate error message
- [ ] Failing status checks block merge with clear message; user can fix and run `kira done` again (unless --force)
- [ ] Unresolved review comments block merge with clear message when API supports it (unless --force)
- [ ] GitHub API errors are handled gracefully
- [ ] Network failures retry with exponential backoff
- [ ] Idempotent: after fixing any failure, re-running `kira done` completes remaining steps

### Polyrepo Support
- [ ] Polyrepo workspace configuration is detected correctly
- [ ] Trunk is validated and pull-trunk is coordinated across all repositories
- [ ] PR merges are coordinated across all repositories (when open)
- [ ] Work item status updates are consistent across all repositories
- [ ] Failures in one repository are handled gracefully
- [ ] All repositories are validated before operations begin

## Slices

### Slice 1: Trunk validation utility
Commit: refactor: add trunk-only validation for kira done
- [ ] T001: Add validateTrunkBranch(cfg) to ensure current branch is trunk; return clear error if not; make reusable where trunk-only execution is required
- [ ] T002: Add unit tests for trunk validation (on trunk vs on feature branch)

### Slice 2: CLI command skeleton
Commit: feat: add kira done command skeleton
- [ ] T003: Add command registration with required work-item-id argument, flags (--merge-strategy, --no-cleanup, --force, --dry-run), and validate trunk branch before any operations
- [ ] T004: Add unit tests for command registration, argument and flag parsing, help output

### Slice 3: Work item and PR resolution
Commit: feat: resolve work item and PR for kira done
- [ ] T005: Validate work item ID argument (format, existence); implement findPullRequest(workItemID) by branch name pattern or metadata; implement isPRClosedOrMerged() for idempotent path
- [ ] T006: Add unit tests for work item validation and PR finding (open vs closed)

### Slice 4: Configuration schema extension
Commit: feat: extend configuration schema for done command
- [ ] T007: Add DoneConfig struct and done section in kira.yml (merge_strategy, cleanup_branch, cleanup_worktree, require_checks, require_comments_resolved, merge commit message templates)
- [ ] T008: Add unit tests for config parsing and flag precedence

### Slice 5: GitHub API client setup
Commit: feat: add GitHub API client utilities
- [ ] T009: Create github.go for GitHub API (OAuth2, owner/repo from remote, token validation, rate limiting, retry logic)
- [ ] T010: Add unit tests and mocks for client initialization (shared with kira review where applicable)

### Slice 6: PR checks and comments resolution
Commit: feat: run PR checks and enforce resolved comments before merge
- [ ] T011: Implement runPRChecks() — run required status checks and block merge until passing; when possible via API check open review comments and block until resolved (unless --force); clear error messages
- [ ] T012: Add unit tests with mocked API (checks passing/failing, comments resolved/unresolved)

### Slice 7: PR merge and pull trunk
Commit: feat: merge PR when open and pull trunk after merge
- [ ] T013: Implement mergePullRequest() when PR is open (merge/squash/rebase per config); skip merge when PR already closed/merged; implement pullTrunk() after merge or when already merged
- [ ] T014: Add unit tests for merge and pull flow; idempotent path when PR already merged

### Slice 8: Work item status update on trunk
Commit: feat: update work item to done on trunk
- [ ] T015: Update work item status to "done" on trunk and set completion metadata (merged_at, merge_commit_sha, pr_number, merge_strategy); commit and push; idempotent when already "done"
- [ ] T016: Add unit tests for status and metadata updates

### Slice 9: Branch cleanup
Commit: feat: implement feature branch cleanup after merge
- [ ] T017: Implement deleteBranch() via GitHub API when cleanup enabled; handle already-deleted branch (idempotent)
- [ ] T018: Add unit tests for cleanup and --no-cleanup behavior

### Slice 10: Idempotent flow and error handling
Commit: feat: idempotent flow and UX for kira done
- [ ] T019: Implement idempotent flow (skip completed steps; complete remaining); progress messages and clear errors with guidance to fix and re-run
- [ ] T020: Add unit tests for idempotent re-run and error recovery scenarios

### Slice 11: Integration tests
Commit: test: add integration tests for kira done workflows
- [ ] T021: Test full workflow on trunk (open PR → merge → pull trunk → status update → cleanup) and idempotent path (already merged PR → pull trunk → status → cleanup)
- [ ] T022: Test PR checks and comments blocking merge (unless --force); test different merge strategies and config

### Slice 12: E2E tests
Commit: test: add e2e tests for kira done command
- [ ] T023: Test complete workflow in test environment (trunk only, work item ID arg); verify PR merge, pull trunk, work item update, cleanup
- [ ] T024: Test not-on-trunk failure and idempotent re-run after failure; test all flags and configurations

## Implementation Notes

### Architecture

#### Command Structure
```
internal/commands/done.go
├── doneCmd - Main cobra command (accepts work-item-id argument)
├── validateTrunkBranch() - Ensure command is run on trunk branch only
├── findPullRequest() - Find associated PR for work item (branch name pattern or metadata)
├── isPRClosedOrMerged() - Detect if PR already merged (idempotent path)
├── runPRChecks() - Run required status checks; ensure open comments resolved when possible
├── mergePullRequest() - GitHub API integration for PR merge (when open)
├── pullTrunk() - Pull latest from trunk on remote after merge
├── updateWorkItemStatus() - Update work item to done on trunk (current branch)
├── cleanupBranch() - Delete feature branch after merge
└── idempotentSkipOrComplete() - Skip completed steps; complete any remaining when re-run
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

#### Work Item and PR State (Idempotent)
- Work item ID comes from the required argument (e.g. `kira done 014`).
- When work item is in "review": merge PR (after checks), pull trunk, update to "done", cleanup.
- When work item is already "done" or PR is already merged: skip merge; pull trunk if needed, ensure status/metadata correct, perform cleanup if still needed. Do not fail.

#### Trunk-Only and Pull-Trunk Flow
`kira done` runs only on trunk. After PR is merged (or detected as already merged), pull latest trunk so local is up to date, then update work item on trunk.

```go
func validateTrunkBranch(cfg *config.Config) error {
    currentBranch := getCurrentBranch()
    if currentBranch != cfg.Git.TrunkBranch {
        return fmt.Errorf("cannot run 'kira done' on a feature branch. Check out the trunk branch (%s) first so the feature branch can be removed after merge", cfg.Git.TrunkBranch)
    }
    return nil
}

func pullTrunk(trunkBranch string) error {
    if err := exec.Command("git", "pull", "origin", trunkBranch).Run(); err != nil {
        return fmt.Errorf("failed to pull trunk: %w. Fix and run 'kira done' again", err)
    }
    return nil
}

// Update work item on trunk (caller has already validated we are on trunk)
func updateWorkItemStatusOnTrunk(workItemID string, metadata map[string]interface{}) error {
    if err := moveWorkItem(workItemID, "done", true); err != nil {
        return fmt.Errorf("failed to update work item status: %w", err)
    }
    if err := commitAndPushStatusChange(workItemID); err != nil {
        return fmt.Errorf("failed to commit status change: %w", err)
    }
    return nil
}
```

#### Idempotent Flow
When re-running after a failure or when work item is already done: skip steps already completed (e.g. PR already merged, status already "done"); perform only remaining steps (pull trunk, ensure status/metadata correct, cleanup).

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

Implement in the order given in the **Slices** section above. Each slice corresponds to one commit; unit tests are included in the same slice. Integration and E2E tests are Slices 11 and 12.

### Testing Strategy

#### Unit Tests (Integrated into Each Phase)
Unit tests are written as part of each implementation phase, ensuring that:
- Each feature is tested immediately as it's implemented
- Tests and implementation are in the same atomic commit
- Test coverage grows incrementally with functionality
- Issues are caught early before moving to the next phase

Unit tests cover:
- Trunk-only validation (reject when not on trunk)
- Work item ID argument validation and PR resolution (open vs closed)
- PR checks and comments resolution (block merge until passing/resolved unless --force)
- Merge when open; skip merge when PR already closed; pull trunk after merge
- Work item status update on trunk and completion metadata
- Idempotent flow (already done or PR already merged)
- Configuration parsing and flag handling
- Error scenarios and edge cases

#### Integration Tests (Phase 11)
- Full workflow on trunk: open PR → checks → merge → pull trunk → status → cleanup
- Idempotent: already merged PR → pull trunk → status → cleanup
- PR checks and comments blocking merge (unless --force)

#### E2E Tests (Phase 12)
- `kira done <work-item-id>` on trunk; verify merge, pull trunk, status, cleanup
- Not-on-trunk failure; idempotent re-run after failure
- All flags and configurations

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
- Run only on trunk branch so feature branch can be removed after merge
- Validate PR is approved and mergeable before merging (when PR is open)
- Require all status checks to pass and, when possible via API, all open review comments resolved (unless --force)
- Prevent accidental merges of unapproved or failing PRs
- Require explicit confirmation for destructive operations where appropriate

## Release Notes

### New Features
- **Done Command**: New `kira done <work-item-id>` command (e.g. `kira done 014`) run on trunk to complete a work item: merge PR (if open), pull trunk, update status to "done", and optionally delete the feature branch
- **Trunk-Only Execution**: Must be run on the trunk branch so the feature branch can be removed after merge; clear error when run on a feature branch
- **PR Checks Before Merge**: All required status checks must pass on the PR before merge (unless `--force`); when possible via API, all open review comments must be resolved
- **Already-Merged PR Handling**: If the PR is already closed (merged), command continues with next steps (pull trunk, update status, cleanup) without failing
- **Pull Trunk After Merge**: Once the PR is merged on the remote, pulls latest from trunk so local trunk is up to date before status updates and cleanup
- **Idempotent**: Safe to run again if the work item is already done or after fixing a prior failure; completed steps are skipped, remaining steps are performed
- **GitHub Integration**: PR merge, status/comment checks, and branch cleanup via GitHub API
- **Merge Strategy Support**: Configurable merge strategies (merge, squash, rebase)

### Improvements
- Clear workflow: run on trunk with work item ID; PR checks and comments enforced before merge
- Idempotent re-runs allow correcting failures and re-running `kira done` without starting over
- Trunk is updated from remote after merge so status and cleanup operate on latest state
- Automatic branch cleanup after successful merge (configurable via `--no-cleanup`)

### Technical Changes
- Added GitHub API client dependency for PR merge, status checks, and comment resolution
- Extended configuration schema for done (require_checks, require_comments_resolved, cleanup, merge_strategy)
- Command structure: work-item-id argument, trunk validation, pull-trunk step, idempotent flow
