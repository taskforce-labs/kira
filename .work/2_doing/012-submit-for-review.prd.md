---
id: 012
title: submit for review
status: doing
kind: prd
assigned:
estimate: 3 days
created: 2026-01-19
due: 2026-01-19
tags: [github, review, cli]
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
5. Reviewer notifications are out of scope; GitHub handles PR review notifications
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
  - `--draft` - Create as draft PR (default comes from `review.draft_by_default` in kira.yml; use `--draft=false` to create ready-for-review, or `--draft=true` to force draft). Explicit flag overrides config.
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

#### Resilience
- GitHub API and network calls (PR create/update, find existing PR, request reviews, etc.) must use **retry with exponential backoff** on transient failures (e.g. 5xx, rate limit 429, network errors)
- Max retries: 3; backoff policy: exponential (e.g. 1s, 2s, 4s); only retryable errors are retried (do not retry on 4xx except 429)

### Configuration

#### kira.yml Extensions
```yaml
review:
  update_trunk_status: true   # Update work item status on trunk branch as source of truth (default: true)
  rebase_after_trunk_update: true # Rebase current branch after trunk status update (default: true)
  draft_by_default: true    # Create draft PRs by default (default: true)
  auto_request_reviews: true # Auto-request reviews from assigned reviewers (default: true)
  pr_title: "[{id}] {title}"  # Template variables: {id}, {title}, {kind}
  pr_description: "View detailed work item: [{id}-{title}]({work_item_url})"  # {work_item_url} points to trunk branch
```

GitHub token is not configured in kira.yml; the command uses the `KIRA_GITHUB_TOKEN` environment variable only.

### Branch and PR Management

#### Branch Requirements
- Branch must already exist (created by `kira start`)
- Branch must follow naming convention: `{id}-{kebab-case-title}`
- Branch should be pushed to remote before creating PR
- Command will push branch if not already on remote

#### PR Creation Logic
1. Check if PR already exists for this branch (via GitHub API)
   - If exists and is draft, update to ready-for-review (when `--draft=false`)
   - If exists and already ready, show message and exit successfully
   - If doesn't exist, proceed to create new PR
2. Verify branch exists on remote (push if needed)
   - Check via `git ls-remote --heads <remote> <branch>`
   - If not exists, push normally
   - If exists but diverged, fail with error (don't force push)
   - If exists and up-to-date, proceed
3. Create draft PR with generated content
4. Add labels based on work item tags (1:1 mapping, skip if label doesn't exist on GitHub)
5. Request reviews from specified reviewers

### Error Handling

#### Validation Errors
- Work item not found: "Work item {id} not found"
- Already in review: If work item already in 'review' status, show message and exit successfully (don't fail)
- Missing required fields: "Work item missing required fields: {field1}, {field2}. Update work item and try again." (uses `validation.required_fields` from config)
- GitHub token missing: "GitHub token required for PR creation. Set KIRA_GITHUB_TOKEN environment variable."
- GitHub token invalid: "GitHub token validation failed. Token must have 'repo' scope for PR creation."

#### Git Operations
- Uncommitted changes: "Uncommitted changes detected. Commit or stash changes before submitting for review."
- Branch not on remote: Push branch to remote before creating PR
- Branch diverged from remote: "Branch has diverged from remote. Pull latest changes or resolve conflicts before submitting for review."
- Push conflicts: Guide user to resolve conflicts
- Remote not found: "GitHub remote '{remote}' not configured"
- Trunk branch diverged: Pull latest trunk before updating status (similar to `kira start` pattern) similar to `kira latest` pattern

#### Notification Failures
- Notifications are out of scope; no notification failure handling

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
- [ ] GitHub API transient errors (e.g. rate limit 429, 5xx) are retried with exponential backoff

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
- **YAML**: Extended config parsing for review config

### GitHub API Integration

#### Repository URL Detection
```go
// Get remote URL from git config
remoteName := cfg.Git.Remote  // e.g., "origin"
remoteURL, err := exec.Command("git", "remote", "get-url", remoteName).Output()

// Parse owner/repo from URL (handle both SSH and HTTPS)
// SSH: git@github.com:owner/repo.git
// HTTPS: https://github.com/owner/repo.git
// GitHub Enterprise: https://github.example.com/owner/repo.git

// Validate it's a GitHub URL
if !isGitHubURL(remoteURL) {
    return fmt.Errorf("remote '%s' is not a GitHub repository", remoteName)
}

// Extract owner and repo
owner, repo := parseGitHubURL(remoteURL)
```

#### Authentication
Token is read from the `KIRA_GITHUB_TOKEN` environment variable only (no config key).

```go
ctx := context.Background()
ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
tc := oauth2.NewClient(ctx, ts)
client := github.NewClient(tc)

// Validate token has repo scope
if err := validateTokenScope(client); err != nil {
    return fmt.Errorf("GitHub token validation failed: %w", err)
}
```

#### PR Creation
```go
// Check if PR already exists for this branch
existingPR, err := findExistingPR(client, owner, repo, branchName)
if err == nil && existingPR != nil {
    // Update existing PR if needed
    if existingPR.Draft && !isDraft {
        // Update from draft to ready
    }
    return existingPR, nil
}

// Create new PR
pr := &github.NewPullRequest{
    Title: &title,
    Head: &branchName,
    Base: &baseBranch,
    Body: &description,
    Draft: &isDraft,
}
```

#### Error Handling
- Retry with exponential backoff on rate limit (429), server errors (5xx), and transient network failures; do not retry on 4xx (except 429)
- API errors: Parse and provide user-friendly messages
- Token issues: Clear guidance on token setup
- Rebase conflicts: "Rebase conflicts detected. Resolve conflicts manually, then run 'git rebase --continue' and re-run 'kira review'"
- Non-GitHub remote: "Remote '{remote}' is not a GitHub repository. This command only works with GitHub repositories."


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
    // For MVP: If kira user command not available, return clear error
    // For now, return placeholder error with fallback suggestion
    return "", fmt.Errorf("user number resolution requires 'kira user' command: user number '%s' cannot be resolved. Use email address or GitHub username directly, or install 'kira user' command", userNumber)
}
```

#### Trunk Status Update Process
```go
func updateTrunkStatus(workItemID string, cfg *config.Config) error {
    // Get trunk branch from existing git configuration
    trunkBranch := cfg.Git.TrunkBranch

    // Pull latest trunk before updating (similar to kira start pattern)
    if err := pullLatestTrunk(trunkBranch, cfg); err != nil {
        return fmt.Errorf("failed to pull latest trunk: %w", err)
    }

    // Stash any uncommitted changes (handle stash failures gracefully)
    stashOutput, _ := exec.Command("git", "stash").Output()

    // Switch to trunk branch
    if err := exec.Command("git", "checkout", trunkBranch).Run(); err != nil {
        return fmt.Errorf("failed to checkout trunk branch '%s': %w", trunkBranch, err)
    }

    // Check if work item exists on trunk, if not copy from feature branch
    if !workItemExistsOnTrunk(workItemID) {
        if err := copyWorkItemFromFeatureBranch(workItemID); err != nil {
            return fmt.Errorf("failed to copy work item to trunk: %w", err)
        }
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
            return fmt.Errorf("rebase conflicts detected. Resolve conflicts manually, then run 'git rebase --continue' and re-run 'kira review'")
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
    "reviewers": reviewers,  // YAML array format
    "reviewed_at": time.Now().Format(time.RFC3339),
}
```
- Preserve all existing front matter fields
- Update `status` and `updated` fields
- Add new fields: `review_pr_url` (string), `reviewers` (array), `reviewed_at` (RFC3339 timestamp)

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
- N/A – notifications out of scope

#### E2E Tests
- `kira review` command in test environment
- Verify PR creation and work item updates

## Implementation Phases

This section breaks down the `kira review` command implementation into small, atomic, testable commits.

### Phase 1: Command Structure & Input Parsing

**Goal**: Create the basic command skeleton with flag parsing

**Files to create/modify**:
- `internal/commands/review.go` (new file)
- `internal/commands/root.go` (add command registration)

**What to implement**:
- Basic cobra command structure (`reviewCmd`)
- Command registration in `root.go`
- Flag definitions:
  - `--reviewer` (string slice, can be used multiple times)
  - `--draft` (bool, default: true)
  - `--no-trunk-update` (bool)
  - `--no-rebase` (bool)
  - `--title` (string)
  - `--description` (string)
- Basic `RunE` function that prints help (placeholder)

**Tests**:
- Command is registered and appears in help
- Flags are parsed correctly
- Flag defaults are correct

**Commit message**: `feat: add review command skeleton with flag parsing`

---

### Phase 2: Branch Context Validation

**Goal**: Validate that command is run in correct context

**Files to modify**:
- `internal/commands/review.go`

**What to implement**:
- `getCurrentBranch()` - Get current git branch name
- `deriveWorkItemFromBranch(branchName string) (string, error)` - Extract work item ID from branch name
  - Parse format: `{id}-{title}` (e.g., "012-submit-for-review")
  - Validate ID format (3 digits)
  - Return error if format doesn't match
- `validateBranchContext(cfg *config.Config) error` - Ensure not on trunk branch
  - Get trunk branch from config
  - Compare with current branch
  - Return error if on trunk

**Tests**:
- `deriveWorkItemFromBranch` extracts ID correctly from valid branches
- `deriveWorkItemFromBranch` returns error for invalid formats
- `validateBranchContext` rejects trunk branch
- `validateBranchContext` accepts feature branches

**Commit message**: `feat(review): add branch validation and work item ID derivation`

---

### Phase 3: Configuration Structure

**Goal**: Add review configuration section to config

**Files to modify**:
- `internal/config/config.go`

**What to implement**:
- `ReviewConfig` struct with fields:
  - `UpdateTrunkStatus bool`
  - `RebaseAfterTrunkUpdate bool`
  - `DraftByDefault bool`
  - `AutoRequestReviews bool`
  - `GitHubToken string`
  - `PRTitle string`
  - `PRDescription string`
- Add `Review *ReviewConfig` field to `Config` struct
- Add defaults in `DefaultConfig`:
  - `UpdateTrunkStatus: true`
  - `RebaseAfterTrunkUpdate: true`
  - `DraftByDefault: true`
  - `AutoRequestReviews: true`
  - `PRTitle: "[{id}] {title}"`
  - `PRDescription: "View detailed work item: [{id}-{title}]({work_item_url})"`

**Tests**:
- Config loads with defaults when review section missing
- Config loads custom review values from kira.yml

**Commit message**: `feat(config): add review configuration section with defaults`

---

### Phase 4: Work Item Loading & Status Validation

**Goal**: Load work item and validate it can be moved to review

**Files to modify**:
- `internal/commands/review.go`
- Reuse existing work item loading utilities

**What to implement**:
- `loadWorkItem(workItemID string) (*WorkItem, error)` - Load work item from current branch
- `validateWorkItemStatus(workItem *WorkItem) error` - Check status is "todo" or "doing"
  - Return success (don't fail) if already in "review" status
  - Return error for other statuses
- `validateRequiredFields(workItem *WorkItem, cfg *config.Config) error` - Check required fields exist
  - Use `cfg.Validation.RequiredFields`
  - Return clear error listing missing fields

**Tests**:
- Load work item from current branch
- Validate status transitions (todo/doing → review)
- Validate already-in-review returns success
- Validate invalid status returns error
- Validate required fields check

**Commit message**: `feat(review): add work item loading and status validation`

---

### Phase 5: Uncommitted Changes Check

**Goal**: Ensure working directory is clean before operations

**Files to modify**:
- `internal/commands/review.go`

**What to implement**:
- `checkUncommittedChanges() error` - Check for uncommitted changes
  - Use `git status --porcelain`
  - Return error if changes exist
  - Clear error message: "Uncommitted changes detected. Commit or stash changes before submitting for review."

**Tests**:
- Detects uncommitted changes
- Allows clean working directory
- Error message is clear

**Commit message**: `feat(review): add uncommitted changes validation`

---

### Phase 6: Work Item Status Update (Current Branch)

**Goal**: Update work item status to "review" on current branch

**Files to modify**:
- `internal/commands/review.go`
- Reuse existing work item move/update utilities

**What to implement**:
- `updateWorkItemStatus(workItemID string, status string) error` - Move work item to review status
  - Use existing move logic (similar to `kira move`)
  - Update status field in front matter
  - Preserve all other front matter fields

**Tests**:
- Work item status updates correctly
- Front matter preserved
- File moves to correct status folder

**Commit message**: `feat(review): update work item status to review on current branch`

---

### Phase 7: Git Remote & Branch Validation

**Goal**: Validate remote exists and branch is pushed

**Files to modify**:
- `internal/commands/review.go`

**What to implement**:
- `validateRemoteExists(cfg *config.Config) error` - Check remote is configured
  - Use `git remote get-url <remote>`
  - Return error if remote not found
- `checkBranchOnRemote(branchName string, cfg *config.Config) (bool, error)` - Check if branch exists on remote
  - Use `git ls-remote --heads <remote> <branch>`
  - Return true if exists, false if not
- `checkBranchDiverged(branchName string, cfg *config.Config) (bool, error)` - Check if branch diverged
  - Compare local and remote branch
  - Return error if diverged (don't allow force push)

**Tests**:
- Validates remote exists
- Detects branch on remote
- Detects diverged branches
- Returns appropriate errors

**Commit message**: `feat(review): add git remote and branch validation`

---

### Phase 8: Branch Push Logic

**Goal**: Push branch to remote if not already there

**Files to modify**:
- `internal/commands/review.go`

**What to implement**:
- `pushBranchIfNeeded(branchName string, cfg *config.Config) error`
  - Check if branch exists on remote (reuse Phase 7 logic)
  - If not exists, push normally
  - If exists and up-to-date, skip
  - If diverged, return error (never force push)

**Tests**:
- Pushes branch when not on remote
- Skips push when already on remote and up-to-date
- Errors when branch diverged

**Commit message**: `feat(review): add branch push logic`

---

### Phase 9: GitHub URL Parsing

**Goal**: Parse GitHub repository URL from git remote

**Files to modify**:
- `internal/commands/review.go` (or new `internal/github/url.go`)

**What to implement**:
- `parseGitHubURL(remoteURL string) (owner string, repo string, err error)`
  - Handle SSH format: `git@github.com:owner/repo.git`
  - Handle HTTPS format: `https://github.com/owner/repo.git`
  - Handle GitHub Enterprise: `https://github.example.com/owner/repo.git`
  - Extract owner and repo name
- `isGitHubURL(url string) bool` - Validate URL is GitHub
- `getGitHubRepoInfo(cfg *config.Config) (owner string, repo string, err error)`
  - Get remote URL
  - Parse and validate

**Tests**:
- Parses SSH URLs correctly
- Parses HTTPS URLs correctly
- Parses GitHub Enterprise URLs
- Returns error for non-GitHub remotes

**Commit message**: `feat(review): add GitHub URL parsing utilities`

---

### Phase 10: GitHub Token Validation

**Goal**: Validate GitHub token exists and has correct permissions

**Files to modify**:
- `internal/commands/review.go` (or new `internal/github/client.go`)

**What to implement**:
- `getGitHubToken(cfg *config.Config) (string, error)` - Get token from `KIRA_GITHUB_TOKEN` environment variable only (no config key)
  - Return error if missing
- `validateGitHubToken(token string) error` - Validate token has repo scope
  - Create GitHub client
  - Make API call to check token permissions
  - Return error if invalid or missing repo scope

**Tests**:
- Reads token from KIRA_GITHUB_TOKEN environment variable
- Validates token permissions
- Returns clear errors for missing/invalid tokens

**Commit message**: `feat(review): add GitHub token validation`

---

### Phase 11: GitHub Client Setup

**Goal**: Create authenticated GitHub API client

**Files to modify**:
- `internal/commands/review.go` (or new `internal/github/client.go`)

**What to implement**:
- `createGitHubClient(token string) (*github.Client, error)` - Create authenticated client
  - Use `github.com/google/go-github/v58/github`
  - Use OAuth2 token source
  - Return configured client

**Tests**:
- Creates client with valid token
- Handles invalid token gracefully

**Commit message**: `feat(review): add GitHub API client setup`

---

### Phase 12: PR Title & Description Generation

**Goal**: Generate PR title and description from work item

**Files to modify**:
- `internal/commands/review.go`

**What to implement**:
- `generatePRTitle(workItem *WorkItem, cfg *config.Config) (string, error)`
  - Use template from `cfg.Review.PRTitle` (default: `"[{id}] {title}"`)
  - Replace variables: `{id}`, `{title}`, `{kind}`
  - Truncate if > 200 chars
  - Sanitize special characters
- `generatePRDescription(workItem *WorkItem, cfg *config.Config) (string, error)`
  - Use template from `cfg.Review.PRDescription`
  - Replace variables: `{id}`, `{title}`, `{work_item_url}`
  - Construct work item URL (trunk branch)
  - Make URL optional (don't fail if can't construct)

**Tests**:
- Generates title with template variables
- Generates description with template variables
- Truncates long titles
- Handles missing URL gracefully

**Commit message**: `feat(review): add PR title and description generation`

---

### Phase 13: Check Existing PR

**Goal**: Check if PR already exists for branch

**Files to modify**:
- `internal/commands/review.go`

**What to implement**:
- `findExistingPR(client *github.Client, owner string, repo string, branchName string) (*github.PullRequest, error)`
  - List PRs for repository
  - Filter by head branch matching current branch
  - Return PR if found, nil if not found
- Handle existing PR cases:
  - If draft and `--draft=false`, update to ready
  - If already ready, show message and exit successfully
  - If not found, proceed to create

**Tests**:
- Finds existing PR by branch
- Returns nil when no PR exists
- Handles draft PRs correctly

**Commit message**: `feat(review): add existing PR detection logic`

---

### Phase 14: Create GitHub PR

**Goal**: Create pull request on GitHub

**Files to modify**:
- `internal/commands/review.go`

**What to implement**:
- `createGitHubPR(client *github.Client, owner string, repo string, branchName string, baseBranch string, title string, description string, isDraft bool) (*github.PullRequest, error)`
  - Create new PR using GitHub API
  - Set head branch, base branch, title, description, draft status
  - Return created PR with URL

**Tests**:
- Creates draft PR successfully
- Creates ready PR successfully
- Returns PR with correct fields
- Handles API errors gracefully

**Commit message**: `feat(review): add GitHub PR creation`

---

### Phase 15: Add PR Labels

**Goal**: Add labels to PR based on work item tags

**Files to modify**:
- `internal/commands/review.go`

**What to implement**:
- `addPRLabels(client *github.Client, owner string, repo string, prNumber int, tags []string) error`
  - For each tag, try to add as label
  - If label doesn't exist on GitHub, log warning and skip (don't fail)
  - Use 1:1 mapping (tag → label)

**Tests**:
- Adds existing labels successfully
- Skips non-existent labels with warning
- Handles API errors gracefully

**Commit message**: `feat(review): add PR labels from work item tags`

---

### Phase 16: Reviewer Resolution

**Goal**: Resolve reviewer specifications to GitHub usernames

**Files to modify**:
- `internal/commands/review.go`

**What to implement**:
- `resolveReviewers(reviewerSpecs []string, cfg *config.Config) ([]string, error)`
  - Check if spec is user number (digits only)
    - If yes, resolve via `kira user` config (if available)
    - If not available, return error with helpful message
  - Check if spec is email address (contains "@")
    - Use as-is (GitHub will handle email lookup)
  - Otherwise, assume GitHub username
- `resolveUserByNumber(userNumber string, cfg *config.Config) (string, error)`
  - Load user mappings from config
  - Return GitHub username or email

**Tests**:
- Resolves user numbers to usernames
- Handles email addresses
- Handles GitHub usernames
- Returns error for unresolved user numbers

**Commit message**: `feat(review): add reviewer resolution logic`

---

### Phase 17: Request PR Reviews

**Goal**: Request reviews from specified reviewers

**Files to modify**:
- `internal/commands/review.go`

**What to implement**:
- `requestPRReviews(client *github.Client, owner string, repo string, prNumber int, reviewers []string) error`
  - Use GitHub API to request reviews
  - Only if `cfg.Review.AutoRequestReviews` is true
  - Handle API errors gracefully

**Tests**:
- Requests reviews successfully
- Respects auto_request_reviews config
- Handles invalid reviewers gracefully

**Commit message**: `feat(review): add PR review request functionality`

---

### Phase 18: Trunk Branch Status Update

**Goal**: Update work item status on trunk branch

**Files to modify**:
- `internal/commands/review.go`

**What to implement**:
- `updateTrunkStatus(workItemID string, cfg *config.Config) error`
  - Pull latest trunk (similar to `kira start` pattern)
  - Stash uncommitted changes (handle failures gracefully)
  - Checkout trunk branch
  - Check if work item exists on trunk
    - If not, copy from feature branch
  - Update work item status to "review"
  - Commit and push (if configured)
  - Switch back to original branch
  - Restore stashed changes

**Tests**:
- Updates trunk status successfully
- Copies work item if not on trunk
- Handles stash failures
- Switches branches correctly

**Commit message**: `feat(review): add trunk branch status update`

---

### Phase 19: Rebase After Trunk Update

**Goal**: Rebase current branch onto updated trunk

**Files to modify**:
- `internal/commands/review.go`

**What to implement**:
- `performRebase(cfg *config.Config) error`
  - Get trunk branch from config
  - Run `git rebase <trunk>`
  - Detect conflicts
  - Return clear error message with resolution instructions

**Tests**:
- Rebases successfully when no conflicts
- Detects and reports conflicts
- Error message is clear

**Commit message**: `feat(review): add rebase after trunk update`

---

### Phase 20: Work Item Metadata Updates

**Goal**: Update work item with PR URL and reviewer metadata

**Files to modify**:
- `internal/commands/review.go`

**What to implement**:
- `updateWorkItemMetadata(workItemID string, prURL string, reviewers []string) error`
  - Load work item
  - Add/update front matter fields:
    - `review_pr_url`: PR URL
    - `reviewers`: array of reviewers
    - `reviewed_at`: RFC3339 timestamp
  - Preserve all existing front matter
  - Update `updated` timestamp

**Tests**:
- Updates metadata correctly
- Preserves existing front matter
- Formats timestamps correctly

**Commit message**: `feat(review): add work item metadata updates`

---

### Phase 21: Command Integration & Error Handling

**Goal**: Wire everything together with proper error handling

**Files to modify**:
- `internal/commands/review.go`

**What to implement**:
- Complete `RunE` function that orchestrates all phases:
  1. Validate branch context
  2. Derive work item ID
  3. Load and validate work item
  4. Check uncommitted changes
  5. Update work item status on current branch
  6. Validate remote and branch
  7. Push branch if needed
  8. Get GitHub repo info
  9. Validate GitHub token
  10. Create GitHub client
  11. Check for existing PR
  12. Generate PR title/description
  13. Create/update PR
  14. Add labels
  15. Resolve reviewers
  16. Request reviews
  17. Update trunk status (if enabled)
  18. Rebase (if enabled)
  19. Update work item metadata
- Comprehensive error handling at each step
- Clear error messages
- Success messages

**Tests**:
- Full integration test
- Error handling at each step
- Success path works end-to-end

**Commit message**: `feat(review): integrate all components with error handling`

---

### Phase 22: E2E Tests & Documentation

**Goal**: Add end-to-end tests and update documentation

**Files to create/modify**:
- `internal/commands/review_test.go` (if not already comprehensive)
- `kira_e2e_tests.sh` (add review command tests)
- `README.md` (document new command)

**What to implement**:
- E2E test scenarios:
  - Create PR from feature branch
  - Update existing draft PR
  - Handle trunk updates
  - Handle rebase conflicts
  - Error scenarios
- Documentation:
  - Command usage
  - Configuration options
  - Examples
  - Troubleshooting

**Tests**:
- All E2E scenarios pass
- Documentation is accurate

**Commit message**: `test(review): add e2e tests and documentation`

---

### Implementation Phase Summary

**Total Phases**: 22

**Order of Implementation**:
1. Input/CLI structure (Phase 1)
2. Validation layers (Phases 2-5)
3. Configuration (Phase 3)
4. Basic work item operations (Phase 6)
5. Git operations (Phases 7-8)
6. GitHub infrastructure (Phases 9-11)
7. GitHub PR operations (Phases 12-17)
8. Trunk/rebase operations (Phases 18-19)
9. Metadata updates (Phase 20)
10. Integration (Phase 21)
11. Testing & docs (Phase 22)

Each phase is:
- **Atomic**: Can be committed independently
- **Testable**: Has clear test requirements
- **Incremental**: Builds on previous phases
- **Focused**: Single responsibility per phase

## Implementation Clarifications

### Edge Cases and Error Handling

#### Trunk Branch Status Update Edge Cases
- **Trunk diverged from remote**: Pull latest trunk before updating status (similar to `kira start` pattern)
- **Work item doesn't exist on trunk**: Copy work item from feature branch to trunk using same file name and location structure
- **Stash failures**: Handle gracefully - if stash fails, check for uncommitted changes and fail with clear error
- **Untracked files**: Don't auto-stash untracked files - require user to commit or stash explicitly

#### GitHub Repository URL Detection
- Use `git remote get-url <remote>` to get actual URL (not just remote name)
- Parse URL to extract owner/repo (handle both SSH `git@github.com:owner/repo.git` and HTTPS `https://github.com/owner/repo.git` formats)
- Validate it's a GitHub URL (github.com or GitHub Enterprise)
- Return error if not GitHub: "Remote '{remote}' is not a GitHub repository. This command only works with GitHub repositories."

#### PR Description Template Variables
- `{work_item_url}`: Points to trunk branch (source of truth)
  - Format: `https://github.com/{owner}/{repo}/blob/{trunk_branch}/{work_item_path}`
  - Example: `https://github.com/owner/repo/blob/main/.work/3_review/012-submit-for-review.prd.md`
  - If file doesn't exist on trunk, use feature branch URL as fallback
  - Make template variable optional (don't fail if can't construct URL)

#### PR Title Template
- Support variables: `{id}`, `{title}`, `{kind}`
- Truncate title if > 200 chars (leave room for ID prefix, GitHub limit is 255)
- Sanitize special characters that might break GitHub API

#### Labels from Work Item Tags
- For MVP: Use tags as-is (1:1 mapping to GitHub labels)
- If label doesn't exist on GitHub, log warning and skip (don't create labels automatically)
- Future enhancement: Add config mapping `review.label_mapping` for custom mappings
- Document that labels must exist on GitHub repo beforehand

#### Branch Push Behavior
- Check if branch exists on remote: `git ls-remote --heads <remote> <branch>`
- If not exists, push normally
- If exists but diverged, fail with error: "Branch has diverged from remote. Pull latest changes or resolve conflicts before submitting for review."
- If exists and up-to-date, proceed
- Never force push

#### Error Recovery Strategy
- Operations are not atomic (by design)
- If trunk update succeeds but rebase fails: Trunk status is already updated (acceptable state)
- If rebase succeeds but PR creation fails: Work item is already in review status (user can create PR manually)
- Always provide clear error messages explaining current state and next steps

#### Uncommitted Changes Handling
- Check for uncommitted changes before starting any operations
- If changes exist, fail with error: "Uncommitted changes detected. Commit or stash changes before submitting for review."
- Don't auto-stash (too risky, user should be explicit about their changes)

#### Work Item Status Validation
- Only allow transitions: `todo` → `review` or `doing` → `review`
- If already in `review`, show message and exit successfully (don't fail): "Work item is already in review status."
- If in other status, return error: "Cannot submit for review: work item is in {current_status} status. Only 'todo' or 'doing' status can be moved to review."

#### Required Fields Validation
- Use `cfg.Validation.RequiredFields` from config (default: id, title, status, kind, created)
- Validate before any operations
- Return clear error: "Work item missing required fields: {field1}, {field2}. Update work item and try again."

#### GitHub Token Configuration
- Token is read from the `KIRA_GITHUB_TOKEN` environment variable only
- Validate token has `repo` scope before operations
- Return clear error if missing or invalid: "GitHub token validation failed. Token must have 'repo' scope for PR creation."

#### Existing PR draft → ready
- When updating an existing PR, if the user passed `--draft=false` and the existing PR is draft, the implementation must set the PR to ready-for-review (e.g. via API) in addition to updating title/body.

#### Polyrepo Support
- **For MVP**: Only support standalone/monorepo (single repository)
- Document polyrepo support as future enhancement
- Work item always lives in main repo, PR created in main repo

### Configuration Structure

#### ReviewConfig Struct
The `review` config section needs to be added to `internal/config/config.go`:

```go
type ReviewConfig struct {
    UpdateTrunkStatus    bool   `yaml:"update_trunk_status"`     // default: true
    RebaseAfterTrunkUpdate bool `yaml:"rebase_after_trunk_update"` // default: true
    DraftByDefault       bool   `yaml:"draft_by_default"`       // default: true
    AutoRequestReviews   bool   `yaml:"auto_request_reviews"`    // default: true
    PRTitle              string `yaml:"pr_title"`                // default: "[{id}] {title}"
    PRDescription        string `yaml:"pr_description"`         // default template
}
```

#### Default Values
- `review.update_trunk_status`: `true`
- `review.rebase_after_trunk_update`: `true`
- `review.draft_by_default`: `true`
- `review.auto_request_reviews`: `true`
- `review.pr_title`: `"[{id}] {title}"`
- `review.pr_description`: `"View detailed work item: [{id}-{title}]({work_item_url})"`

### Security Considerations

#### Token Management
- Never log or expose GitHub tokens
- Token is supplied via KIRA_GITHUB_TOKEN environment variable only (no token in config files)
- Validate token permissions before operations

#### Input Validation
- Sanitize work item content for PR descriptions
- Validate email addresses for reviewer resolution
- Escape special characters in branch names

#### API Rate Limiting
- Implement retries with **exponential backoff** for GitHub API calls (e.g. base delay 1s, multiplier 2, max 3 attempts)
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
- Extended configuration schema for review settings
- New command structure following established Kira patterns

