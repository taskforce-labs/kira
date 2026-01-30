---
id: 012
title: submit for review
status: review
kind: prd
assigned:
estimate: 3 days
created: 2026-01-19
due: 2026-01-19
tags: [github, notifications, review, cli]
---

# submit for review

A command that moves work items to review status on the current branch, optionally updates the status on the main trunk branch to reflect the review state (treating trunk as source of truth), creates pull requests on GitHub or changes the status of a PR on GitHub from draft to ready for review.

## Context

In the Kira workflow, work items progress through different statuses: backlog → todo → doing → review → done. Currently, moving to review status requires manual `kira move` commands and separate GitHub PR creation. This creates friction in the development workflow, especially when working with teams or when agents need to coordinate code reviews.

The `kira review` command (note: renamed from "submit for review" to avoid confusion with existing "review" status) will streamline this process by:

1. Moving the work item to review status on the current feature branch
2. Optionally updating the work item status on the main trunk branch to maintain it as the source of truth
3. Rebasing the current branch onto the updated trunk branch
4. Automatically creating a draft PR on GitHub with proper branch naming and descriptions
5. Optionally notifying reviewers through configurable channels
6. Supporting only individual work items

This approach treats the trunk branch as the authoritative source of truth for work item status, ensuring that the project's current state is always reflected in the main branch, while avoiding merge conflicts through strategic rebasing.

**Dependencies**: Full reviewer number resolution requires the `kira user` command for managing user mappings. Without it, only email addresses and direct GitHub usernames will work.

## Requirements

### Core Functionality

#### Command Interface
- **Command**: `kira review [--reviewer <user>...]`
- **Behavior**: Automatically derives work item ID from current branch name (must be on a kira-created branch)
- **Restrictions**: Cannot be run on trunk branch or non-kira branches
- **Flags**:
  - `--reviewer <user>` - Specify reviewer (can be user number from `kira user` command or email address). Can be used multiple times
  - `--draft` (default: true) - Create as draft PR
  - `--no-trunk-update` - Skip updating trunk branch status (overrides config)
  - `--no-rebase` - Skip rebasing current branch after trunk update (overrides config)
  - `--title <title>` - Custom PR title (derived from work item if not provided)
  - `--description <description>` - Custom PR description (uses work item content if not provided)

#### Work Item Management

- Derive work item ID from current branch name (format: `{id}-{title}`)
- Validate current branch was created by `kira start` (not trunk branch)
- Move derived work item from current status to "review" on the current branch
- Optionally update the work item status to "review" on the trunk branch (when enabled)
- Rebase the current branch onto the updated trunk branch (when trunk updates are enabled)
- Update work item front matter with review metadata (reviewers, PR URL, etc.)
- Validate that work item has required fields before submission
- Support work items in "doing" or "todo" status (error for other statuses)
- Handle merge conflicts during rebase with user guidance

#### GitHub Integration
- Create draft pull request on git.remote (from existing kira.yml git configuration)
- Use git.trunk_branch as the base branch (from existing kira.yml git configuration)
- Extract PR title from work item title (branch name already follows `{id}-{title}` pattern)
- Generate concise PR description with link to full work item details
- Assign reviewers specified via `--reviewer` flag (supports user numbers from `kira user` command or email addresses)
- Push current branch if not already pushed to remote


### Configuration

#### kira.yml Extensions
```yaml
review:
  update_trunk_status: true   # Update work item status on trunk branch as source of truth
  rebase_after_trunk_update: true # Rebase current branch after trunk status update
  draft_by_default: true    # Create draft PRs by default
  auto_request_reviews: true # Auto-request reviews from assigned reviewers
  github_token: "${GITHUB_TOKEN}"  # Environment variable or direct value
  pr_title: "[{id}] {title}"
  pr_description: "View detailed work item: [{id}-{title}]({work_item_url})"
```

### Branch and PR Management

#### Branch Requirements
- Branch must already exist (created by `kira start`)
- Branch must follow naming convention: `{id}-{kebab-case-title}`
- Branch should be pushed to remote before creating PR
- Command will push branch if not already on remote

#### PR Creation Logic
1. Verify branch exists on remote (push if needed)
2. Create draft PR with generated content
3. Add labels based on work item tags
4. Request reviews from specified reviewers

### Error Handling

#### Validation Errors
- Work item not found: "Work item {id} not found"
- Invalid status: "Cannot submit for review: work item is in {current_status} status"
- Missing required fields: "Work item missing required field: {field}"
- GitHub token missing: "GitHub token required for PR creation"

#### Git Operations
- Branch not on remote: Push branch to remote before creating PR
- Push conflicts: Guide user to resolve conflicts
- Remote not found: "GitHub remote '{remote}' not configured"

#### Notification Failures
- Log notification errors but don't fail the command
- Retry logic for transient failures (network issues)
- Graceful degradation when notification services are unavailable

## Acceptance Criteria

### Core Command Functionality
- [ ] `kira review` automatically derives work item ID from current branch name
- [ ] Command fails if run on trunk branch
- [ ] Command fails if current branch doesn't follow kira naming convention
- [ ] Command fails if current branch doesn't exist on remote
- [ ] `kira review` moves derived work item to review status on current branch
- [ ] `kira review` updates derived work item to review status on trunk branch when enabled
- [ ] `kira review` rebases current branch onto trunk after trunk update when enabled
- [ ] `kira review --no-trunk-update` skips trunk status updates
- [ ] `kira review --no-rebase` skips rebase after trunk update
- [ ] `kira review --reviewer user1 --reviewer user2` specifies reviewers
- [ ] `kira review --reviewer 1 --reviewer 2` uses user numbers from kira user command
- [ ] `kira review --reviewer user@example.com` uses email addresses for reviewers
- [ ] `kira review --draft=false` creates ready-for-review PR
- [ ] Command fails gracefully when GitHub token is missing
- [ ] Command shows helpful error messages for invalid work items

### GitHub Integration
- [ ] Creates draft PR with proper title and description
- [ ] Auto-generates branch name following convention
- [ ] PR description includes work item details and requirements
- [ ] PR links back to work item file
- [ ] Handles existing branches appropriately
- [ ] Adds appropriate labels based on work item tags

### Work Item Updates
- [ ] Work item status changes to "review"
- [ ] Front matter includes PR URL and reviewer information
- [ ] Updated timestamp reflects review submission time
- [ ] Work item remains in git history with full traceability


### Trunk Update and Rebase Functionality
- [ ] Trunk branch status updates work correctly when enabled
- [ ] Rebase operations complete successfully after trunk updates
- [ ] Command handles rebase conflicts with clear user guidance
- [ ] Trunk branch configuration respects custom branch names
- [ ] Disabling trunk updates via config works correctly
- [ ] Override flags (--no-trunk-update, --no-rebase) work correctly

### Error Scenarios
- [ ] Invalid work item ID shows clear error message
- [ ] Work item already in review shows appropriate message
- [ ] Trunk branch not found shows appropriate error
- [ ] Rebase conflicts provide clear resolution instructions
- [ ] GitHub API errors are handled gracefully
- [ ] Network failures retry with exponential backoff


## Implementation Notes

### Architecture

#### Command Structure
```
internal/commands/review.go
├── reviewCmd - Main cobra command
├── deriveWorkItemFromBranch() - Extract work item ID from current branch name
├── validateBranchContext() - Ensure command is run on valid kira branch
├── reviewWorkItem() - Core review logic
├── updateTrunkStatus() - Update work item status on trunk branch
├── performRebase() - Rebase current branch onto trunk
├── createGitHubPR() - GitHub API integration
└── updateWorkItem() - Work item metadata updates
```

#### Dependencies
- **GitHub**: `github.com/google/go-github/v58/github` for API client
- **HTTP**: Standard library for webhook notifications
- **SMTP**: `net/smtp` for email notifications
- **YAML**: Extended config parsing for notification settings

### GitHub API Integration

#### Authentication
```go
ctx := context.Background()
ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
tc := oauth2.NewClient(ctx, ts)
client := github.NewClient(tc)
```

#### PR Creation
```go
pr := &github.NewPullRequest{
    Title: &title,
    Head: &branchName,
    Base: &baseBranch,
    Body: &description,
    Draft: &isDraft,
}
```

#### Error Handling
- Rate limiting: Implement backoff and retry logic
- API errors: Parse and provide user-friendly messages
- Token issues: Clear guidance on token setup


### Work Item Processing

#### Status Validation
```go
validStatuses := []string{"todo", "doing"}
if !contains(validStatuses, currentStatus) {
    return fmt.Errorf("work item must be in todo or doing status")
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

func validateBranchContext(currentBranch string, cfg *config.Config) error {
    trunkBranch := cfg.Git.TrunkBranch
    if currentBranch == trunkBranch {
        return fmt.Errorf("cannot run 'kira review' on trunk branch '%s'", trunkBranch)
    }

    // Additional validation could check if branch was created by kira start
    // by verifying work item exists and branch name matches expected pattern

    return nil
}
```

#### Reviewer Resolution
```go
func resolveReviewers(reviewerSpecs []string) ([]string, error) {
    var reviewers []string

    for _, spec := range reviewerSpecs {
        // Check if it's a user number (digits only)
        if regexp.MustCompile(`^\d+$`).MatchString(spec) {
            // Resolve user number to GitHub username/email via kira user config
            user, err := resolveUserByNumber(spec)
            if err != nil {
                return nil, fmt.Errorf("failed to resolve user number '%s': %w", spec, err)
            }
            reviewers = append(reviewers, user)
        } else if strings.Contains(spec, "@") {
            // Email address - use as-is
            reviewers = append(reviewers, spec)
        } else {
            // Assume GitHub username
            reviewers = append(reviewers, spec)
        }
    }

    return reviewers, nil
}

func resolveUserByNumber(userNumber string) (string, error) {
    // Load user mappings from kira user configuration
    // This would integrate with the future kira user command
    // For now, return placeholder error
    return "", fmt.Errorf("user management not yet implemented")
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
    if err := moveWorkItem(workItemID, "review", true); err != nil {
        return fmt.Errorf("failed to update trunk status: %w", err)
    }

    // Switch back to original branch
    if err := exec.Command("git", "checkout", "-").Run(); err != nil {
        return fmt.Errorf("failed to switch back to original branch: %w", err)
    }

    // Restore stashed changes if any
    if stashOutput != nil {
        exec.Command("git", "stash", "pop")
    }

    return nil
}
```

#### Rebase Process
```go
func performRebase(cfg *config.Config) error {
    // Get trunk branch from existing git configuration
    trunkBranch := cfg.Git.TrunkBranch

    // Perform rebase
    cmd := exec.Command("git", "rebase", trunkBranch)
    if err := cmd.Run(); err != nil {
        // Check if rebase failed due to conflicts
        if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
            return fmt.Errorf("rebase failed due to conflicts. Please resolve conflicts and run 'git rebase --continue'")
        }
        return fmt.Errorf("rebase failed: %w", err)
    }
    return nil
}
```

#### Metadata Updates
```go
frontMatter := map[string]interface{}{
    "status": "review",
    "updated": time.Now().Format(time.RFC3339),
    "review_pr_url": prURL,
    "reviewers": reviewers,
}
```

### Testing Strategy

#### Unit Tests
- Test work item ID derivation from branch names
- Test branch validation (trunk branch rejection, invalid formats)
- Test reviewer resolution (user numbers, emails, usernames)
- Mock GitHub API responses
- Test trunk status update functionality
- Test rebase operations and conflict handling
- Validate work item updates
- Error scenario coverage

#### Integration Tests
- Full workflow testing with test GitHub repo
- Notification webhook testing
- Email delivery verification

#### E2E Tests
- `kira review` command in test environment
- Verify PR creation and work item updates
- Test notification delivery

### Security Considerations

#### Token Management
- Never log or expose GitHub tokens
- Use environment variables for sensitive config
- Validate token permissions before operations

#### Input Validation
- Sanitize work item content for PR descriptions
- Validate email addresses and webhook URLs
- Escape special characters in branch names

#### API Rate Limiting
- Implement intelligent backoff for GitHub API calls
- Cache repository information where possible
- Provide clear error messages for rate limit issues

## Release Notes

### New Features
- **Review Command**: New `kira review` command that automatically derives work item from current branch for streamlined PR creation
- **Smart Context Detection**: Automatically identifies work items from kira-created branch names
- **Branch Validation**: Prevents accidental execution on trunk branches or invalid branches
- **Trunk Status Updates**: Optional updates to trunk branch status for maintaining source of truth (configurable)
- **Automatic Rebasing**: Seamless rebase of feature branch after trunk status updates
- **GitHub Integration**: Seamless integration with GitHub for draft PR creation and review management

### Improvements
- Streamlined command interface - no need to specify work item IDs manually
- Smart context awareness - automatically detects work items from branch names
- Enhanced safety - prevents execution on trunk branches
- Enhanced work item metadata tracking for review processes
- Configurable trunk branch status updates (can be disabled if not desired)
- Improved error messages and user guidance for review workflows and rebasing
- Clean PR descriptions with links to detailed work items (no duplication)

### Technical Changes
- Added GitHub API client dependency for PR management
- Extended configuration schema for review and notification settings
- New command structure following established Kira patterns

