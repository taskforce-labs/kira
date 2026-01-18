---
id: 010
title: start draft pr
status: backlog
kind: prd
assigned:
estimate: 0
created: 2026-01-14
due: 2026-01-14
tags: []
---

# start draft pr

Extend the `kira start` command to create draft pull requests by default when starting work on a new task, with an option to skip PR creation.

```bash
kira start <work-item-id> [--no-draft-pr]
```

## Context

### Problem Statement

In modern development workflows, especially when working with multiple agents or parallel development streams, developers often need to create pull requests early in the development process to:

1. **Signal intent**: Let team members know work is in progress
2. **Enable early feedback**: Allow reviewers to see work-in-progress and provide guidance
3. **Facilitate collaboration**: Enable other developers to branch from the work-in-progress
4. **Track progress**: Provide visibility into active development streams

Currently, developers using Kira must manually create pull requests after starting work, which breaks the streamlined workflow that Kira provides. This manual step is particularly disruptive in agentic workflows where multiple tasks are being worked on simultaneously.

### Current State

The `kira start` command currently:
- Manages work item status transitions
- Creates isolated git worktrees for parallel development
- Creates feature branches for each work item
- Opens IDEs in the correct context
- Supports monorepo and polyrepo configurations

However, it does not integrate with GitHub/GitLab to create pull requests, requiring manual PR creation after work begins.

### Proposed Solution

Extend the `kira start` command to create draft pull requests by default:
- Creates worktree and branch locally
- Pushes the branch to the remote repository
- Creates a draft pull request automatically after the branch is pushed
- Uses the work item title and description as PR title/body
- Links the PR back to the work item for traceability
- Provides a `--no-draft-pr` flag to skip PR creation when needed
- Supports configuration-driven behavior (can be disabled per workspace/project)
- Handles authentication and error cases gracefully

This maintains Kira's philosophy of reducing friction in development workflows while extending its capabilities into the collaborative review process.

### Impact

- **For developers**: Zero-friction workflow from idea to visible draft PR
- **For teams**: Automatic visibility into all active work streams
- **For agents**: Native integration with collaborative development processes
- **For organizations**: Improved tracking and early feedback on work-in-progress

## Requirements

### Functional Requirements

#### FR1: Draft PR Creation
The `kira start` command SHALL create a draft pull request by default after successfully creating the worktree and branch, unless the `--no-draft-pr` flag is provided or `workspace.draft_pr` is explicitly disabled in kira.yml.

**Flag Priority:**
The `--no-draft-pr` flag SHALL have the highest priority and SHALL override all other configuration, including:
- Work item `repos` metadata field
- Project-level `draft_pr` settings
- Workspace-level `draft_pr` settings

When `--no-draft-pr` is present, no branches SHALL be pushed and no draft PRs SHALL be created for any repositories, regardless of any other configuration.

#### FR2: PR Content Generation
The draft PR SHALL use:
- **Title**: Work item title prefixed with work item ID (e.g., "010: Add user authentication")
- **Body**: Work item description/content, with work item metadata included
- **Branch**: The newly created feature branch
- **Base**: The trunk branch (main/master or configured)

#### FR3: Configuration Support
The feature SHALL support configuration via `kira.yml`:
```yaml
workspace:
  draft_pr: false  # Disable draft PR creation for this workspace

  # Git platform configuration
  git_platform: github  # Options: github, gitlab, auto (default: auto)
  git_base_url: https://github.example.com  # For GHE/GitLab self-hosted (optional, auto-detected if not set)
```

#### FR3b: Selective Operations for Polyrepo Projects
For polyrepo workspaces, the command SHALL support multiple mechanisms for selective PR creation:

**Mechanism 1: Work Item Metadata Override (Highest Priority)**
- Work items MAY include an optional `repos` field in YAML front matter listing affected repositories
- When present, this field SHALL override all project-level `draft_pr` configuration
- Only repositories listed in the `repos` field SHALL have draft PRs created
- Repository names in `repos` MUST match project names from `workspace.projects`

**Mechanism 2: Per-Project Configuration**
For polyrepo workspaces, the command SHALL support simple per-project opt-out to accommodate repositories that don't need PRs:

**Simple Per-Project Configuration:**
```yaml
workspace:
  projects:
    - name: frontend
      draft_pr: true   # Explicitly enable (default behavior)
      git_platform: gitlab
      git_base_url: https://gitlab.internal.company.com

    - name: infrastructure
      draft_pr: false  # Disable PR creation for this project
      # When draft_pr is false, branch pushing is also skipped (no point pushing without PR)

    - name: docs
      draft_pr: false  # Documentation repos might not need PRs
```

**Work Item Metadata Override:**
Work items MAY include an optional `repos` field in their YAML front matter to declare which repositories will be affected by the work:

```yaml
---
id: 010
title: start draft pr
status: backlog
kind: prd
repos:
  - frontend
  - backend
---
```

When `repos` is present in the work item metadata:
- **Overrides project-level `draft_pr` configuration** for that work item
- **Only creates draft PRs for repos listed** in the `repos` field
- **Skips PR creation for all other repos** in the polyrepo workspace
- **Worktrees still created for all repos** (local work is always available)

This makes the work item self-documenting - it declares which repos it affects, and Kira respects that declaration.

**Natural Behavior (Priority Order):**
1. **`--no-draft-pr` flag** (highest priority): If present, skip push and PR creation for ALL repos, regardless of any other configuration
2. **Work item `repos` field**: If present, only create PRs for listed repos
3. **Project-level `draft_pr: false`**: Skip push and PR creation for that project
4. **Project-level `draft_pr: true` or unset**: Push branch + create draft PR (default)
5. **Auto-detection**: If repo's remote isn't GitHub/GitLab, automatically skip PR creation
6. **Worktrees always created**: Local worktrees are always created regardless of PR settings

This approach lets teams:
- **Override with flag**: Use `--no-draft-pr` to skip all PR creation regardless of configuration
- **Declare scope in work items**: PRDs and tasks can specify which repos they affect
- **Opt-out specific repos**: Infrastructure repos can be configured to never create PRs
- **Maintain flexibility**: Work items can override config when needed, but flag always wins

#### FR4: Authentication Handling
The command SHALL support authentication for multiple Git platforms:

**GitHub Authentication**
1. **Interactive CLI login (preferred)**: `kira auth login` using a browser-based device flow, storing credentials securely in the OS credential store.
2. **Non-interactive token (fallback)**: `GITHUB_TOKEN` environment variable (recommended for CI and headless environments).

**GitLab Authentication**
1. **Interactive CLI login**: `kira auth login --platform gitlab` using OAuth device flow
2. **Non-interactive token**: `GITLAB_TOKEN` environment variable
3. **Personal Access Token**: Supports both OAuth tokens and Personal Access Tokens

**Cross-Platform Auth Storage**
Credentials SHALL be stored separately per platform:
- Service: `kira`
- Account: `github` or `gitlab-enterprise` or `gitlab` or `gitlab-self-hosted`

The command SHALL:
- Use `GITHUB_TOKEN` or `GITLAB_TOKEN` if present
- Look for stored credentials from `kira auth login`
- Fail gracefully with a clear error message if neither auth method is available or if credentials are invalid/expired
- Never log credentials or token values

#### FR4a: Auth Command
Kira SHALL provide an authentication command for developers:

```bash
kira auth login [--platform github|gitlab|auto]
kira auth status [--platform github|gitlab|auto]
kira auth logout [--platform github|gitlab|auto]
```

`kira auth login` SHALL:
- Auto-detect platform and domain from git remotes if `--platform auto` (default)
- For single-repo workspaces: authenticate against the main repository's remote
- For polyrepo workspaces: authenticate against all unique platform/domain combinations found in project remotes
- Initiate platform-appropriate browser-based device flow sign-in for each unique platform/domain
- Obtain tokens suitable for calling the respective platform APIs needed by `kira start`
- Store tokens securely using hierarchical OS credential store keys:
  - Service: `kira`
  - Account: `{platform}/{domain}` (e.g., `github/github.com`, `gitlab/gitlab.company.com`)
- Support custom base URLs for enterprise/self-hosted instances

`kira auth status` SHALL:
- Show status for all stored credentials when no `--platform` specified
- Show status for specific platform/domain when `--platform` specified
- Indicate whether valid stored credentials are present (without revealing secrets)
- Display which domains/platforms have valid tokens

`kira auth logout` SHALL:
- Remove all stored credentials when no `--platform` specified
- Remove credentials for specific platform/domain when `--platform` specified

#### FR4b: Auth Auto-Prompt on Start
When `kira start` attempts draft PR/MR creation and credentials are missing for required platforms/domains, it SHALL:

**For Single-Repo Workspaces:**
- If running interactively (TTY), prompt for authentication against the repository's platform/domain
- If running non-interactively (no TTY), fail with clear message to set appropriate token env var or disable draft PR creation

**For Polyrepo Workspaces:**
- Check credentials for all unique platform/domain combinations across all projects
- If running interactively, prompt sequentially for each missing platform/domain
- If running non-interactively, fail with clear message listing all required token environment variables

Authentication flow SHALL:
- Launch `kira auth login --platform auto` inline to authenticate against detected platforms/domains
- If user completes authentication successfully, continue and create draft PRs/MRs without requiring rerun
- If user declines authentication for any platform, skip draft PR/MR creation for repositories on that platform (equivalent to `--no-draft-pr` for those repos), with clear warning
- Continue with worktree/branch creation regardless of auth outcome

#### FR5: Branch Push Requirement
Before creating draft PRs, the command SHALL push the newly created branch to the remote repository. This ensures:
- The branch exists on the remote for PR creation
- Early visibility of work-in-progress branches
- Consistent state between local and remote repositories

**Selective Push Logic:**
- **If `draft_pr: true` or unset**: Push branch to remote (required for PR creation)
- **If `draft_pr: false`**: Skip branch push (no point pushing if no PR will be created)
- **Auto-detection**: If repo remote isn't GitHub/GitLab, skip push and PR creation automatically

#### FR6: Error Handling
If branch push fails, the command SHALL:
- Stop execution and display clear error message
- Not attempt PR creation without a pushed branch
- Worktree and branch remain available locally

If PR creation fails after successful push, the command SHALL:
- Continue with IDE opening and setup (branch remains pushed and available remotely)
- Display clear error message about PR creation failure
- Not fail the entire start operation

#### FR7: Push Failure Handling
If branch push fails, the command SHALL:
- Display clear error message explaining the push failure
- Suggest checking network connectivity, remote repository access, or authentication
- Allow user to manually push and create PR later
- Not automatically retry push operations

#### FR8: Platform Support
The implementation SHALL support multiple Git hosting platforms:

**GitHub (GitHub.com and GitHub Enterprise Server)**
- REST API for PR creation with `draft: true`
- GraphQL API for draft ↔ ready state transitions
- Base URL auto-detection: `github.com` for public, configurable for GHE

**GitLab (GitLab.com and self-hosted GitLab)**
- REST API for MR creation with `draft: true` (GitLab 14.0+)
- Work-in-progress (WIP) prefix for older GitLab versions
- Base URL auto-detection: `gitlab.com` for cloud, configurable for self-hosted

**Platform Detection**
The command SHALL automatically detect the platform based on:
1. Git remote URL patterns (`github.com`, `gitlab.com`, custom domains)
2. Configuration override in `kira.yml` for explicit platform specification
3. Fallback to GitHub.com if detection fails

### Non-Functional Requirements

#### NFR1: Performance
PR creation SHALL NOT significantly impact the overall `kira start` execution time. PR creation SHALL happen after worktree setup is complete and SHALL not block IDE opening. In non-interactive environments, the command SHALL NOT block on authentication prompts.

#### NFR2: Security
- Authentication credentials SHALL be handled securely
- No sensitive information SHALL be logged
- SHALL follow secure coding practices as defined in `docs/security/golang-secure-coding.md`

#### NFR3: Backward Compatibility
The feature SHALL be enabled by default but SHALL provide opt-out mechanisms. Existing `kira start` behavior changes to include draft PR creation, but users can disable this via `--no-draft-pr` flag or configuration.

#### NFR4: Observability
The command SHALL provide clear feedback about PR creation status, including:
- Success confirmation with PR URL
- Clear error messages for failures
- Progress indicators during PR creation

## Acceptance Criteria

### AC1: Default Draft PR Creation
**Given** a valid work item exists
**When** `kira start <work-item-id>` is executed
**Then** the branch is pushed to remote AND a draft PR is created with correct title, body, and branch targeting trunk

### AC2: Skip Draft PR Creation
**Given** a valid work item exists
**When** `kira start <work-item-id> --no-draft-pr` is executed
**Then** no draft PR is created, but worktree and branch are created successfully

### AC4: PR Content Accuracy
**Given** a work item with title "Add user authentication" and description
**When** a draft PR is created
**Then** PR title is "010: Add user authentication" and body contains work item content

### AC5: Error Resilience
**Given** PR creation fails due to network issues
**When** `kira start <work-item-id>` is executed
**Then** worktree and branch are still created successfully

### AC6: Authentication Handling (Stored Credentials)
**Given** stored credentials exist from `kira auth login`
**When** draft PR creation is attempted
**Then** the draft PR is created without requiring `GITHUB_TOKEN`

### AC6b: Authentication Handling (Fallback Token)
**Given** stored credentials do not exist OR are invalid/expired
**And** `GITHUB_TOKEN` is set and valid
**When** draft PR creation is attempted
**Then** the draft PR is created using `GITHUB_TOKEN`

### AC6c: Authentication Handling (No Auth)
**Given** stored credentials do not exist OR are invalid/expired
**And** `GITHUB_TOKEN` is missing or invalid
**When** draft PR creation is attempted
**Then** a clear error message guides the user to run `kira auth login` or set `GITHUB_TOKEN` (or disable draft PR creation)

### AC6d: Authentication Auto-Prompt (Interactive)
**Given** stored credentials do not exist OR are invalid/expired
**And** `GITHUB_TOKEN` is missing or invalid
**And** the command is running interactively (TTY)
**When** `kira start <work-item-id>` attempts draft PR creation
**Then** the user is prompted to run the `kira auth login` flow
**And** if the user authenticates successfully, the draft PR is created in the same run

### AC6e: Authentication No-Prompt (Non-Interactive)
**Given** stored credentials do not exist OR are invalid/expired
**And** `GITHUB_TOKEN` is missing or invalid
**And** the command is running non-interactively (no TTY)
**When** `kira start <work-item-id>` attempts draft PR creation
**Then** draft PR creation is skipped with a clear message to set `GITHUB_TOKEN` or disable draft PR creation

### AC7: Polyrepo Support
**Given** a polyrepo workspace with multiple projects
**When** `kira start <work-item-id>` is executed
**Then** worktrees are created for all projects
**And** branches are pushed and draft PRs are created only for projects where `draft_pr: true` or unset (default)

### AC11: Selective Polyrepo Operations
**Given** a polyrepo workspace with a project configured with `draft_pr: false`
**When** `kira start <work-item-id>` is executed
**Then** worktree and branch are created for that project
**And** branch is NOT pushed to remote
**And** no draft PR is created for that project
**And** other projects with `draft_pr: true` or unset still get PRs created normally

### AC12: Work Item Metadata Override
**Given** a polyrepo workspace with projects: `frontend`, `backend`, `infrastructure`, `docs`
**And** a work item with `repos: [frontend, backend]` in its YAML front matter
**When** `kira start <work-item-id>` is executed
**Then** worktrees and branches are created for ALL projects
**And** branches are pushed and draft PRs are created ONLY for `frontend` and `backend`
**And** branches are not pushed and no draft PRs are created for `infrastructure` or `docs`
**And** this behavior overrides any project-level `draft_pr` configuration

### AC13: Work Item Metadata Override Priority
**Given** a polyrepo workspace with `frontend` configured with `draft_pr: false`
**And** a work item with `repos: [frontend]` in its YAML front matter
**When** `kira start <work-item-id>` is executed
**Then** draft PR is created for `frontend` (work item metadata overrides project config)

### AC14: No Draft PR Flag Override
**Given** a polyrepo workspace with projects: `frontend`, `backend`
**And** a work item with `repos: [frontend, backend]` in its YAML front matter
**And** all projects configured with `draft_pr: true`
**When** `kira start <work-item-id> --no-draft-pr` is executed
**Then** worktrees and branches are created for ALL projects
**And** branches are NOT pushed to remote for any project
**And** no draft PRs are created for any project
**And** the `--no-draft-pr` flag overrides both work item metadata and project configuration

### AC8: Multi-Platform Auth in Polyrepo
**Given** a polyrepo workspace with projects on GitHub and GitLab
**When** `kira start <work-item-id>` is executed without stored credentials
**And** running interactively
**Then** the user is prompted to authenticate against both GitHub and GitLab
**And** draft PRs are created for repositories on platforms where authentication succeeded

### AC10: Branch Push Before PR Creation
**Given** a valid work item exists
**When** `kira start <work-item-id>` is executed
**Then** the newly created branch is pushed to the remote repository before draft PR creation
**And** if push fails, PR creation is skipped with appropriate error message

## Implementation Notes

### GitHub Integration Approach

#### Multi-Platform Git API Integration (Required)
Kira SHALL create draft pull requests using GitHub’s API directly from Go.

**GitHub Implementation**
- `github.com/google/go-github/v61/github` for REST operations (creating the PR with `draft: true`)
- GitHub GraphQL API for draft ↔ ready state transitions (future commands)
- `golang.org/x/oauth2` for authentication

**GitLab Implementation**
- `gitlab.com/gitlab-org/api/client-go` for REST operations (creating MR with `draft: true`)
- GraphQL API for advanced MR operations (future)
- `golang.org/x/oauth2` for authentication

**Platform Detection and Client Creation**
```go
type GitPlatform interface {
    CreateDraftPR(ctx context.Context, req DraftPRRequest) (*DraftPRResponse, error)
    ValidateCredentials(ctx context.Context) error
}

type CredentialKey struct {
    Platform string // "github", "gitlab"
    Domain   string // "github.com", "gitlab.company.com"
}

func detectPlatform(remoteURL string, config *Config) GitPlatform {
    // Auto-detect from remote URL patterns
    if strings.Contains(remoteURL, "github.com") || strings.Contains(remoteURL, "github.example.com") {
        return &GitHubPlatform{baseURL: config.GitBaseURL}
    }
    if strings.Contains(remoteURL, "gitlab.com") || strings.Contains(remoteURL, "gitlab.example.com") {
        return &GitLabPlatform{baseURL: config.GitBaseURL}
    }
    // Fallback to config or default
    return &GitHubPlatform{baseURL: "https://api.github.com"}
}

func detectRequiredCredentials(config *Config, projects []Project, workItemRepos []string, noDraftPRFlag bool) []CredentialKey {
    // If --no-draft-pr flag is set, no credentials needed
    if noDraftPRFlag {
        return []CredentialKey{}
    }

    creds := make(map[CredentialKey]bool)

    for _, project := range projects {
        // Only require credentials for projects that need PR creation
        if shouldCreateDraftPR(project, workItemRepos, noDraftPRFlag) {
            platform, domain := detectPlatformAndDomain(project.RepoURL)
            if platform != "" {
                creds[CredentialKey{Platform: platform, Domain: domain}] = true
            }
        }
    }

    var result []CredentialKey
    for key := range creds {
        result = append(result, key)
    }
    return result
}

func shouldCreateDraftPR(project Project, workItemRepos []string, noDraftPRFlag bool) bool {
    // Priority 1: --no-draft-pr flag (highest priority - overrides everything)
    if noDraftPRFlag {
        return false
    }

    // Priority 2: Work item metadata override
    if len(workItemRepos) > 0 {
        // If work item specifies repos, only create PRs for listed projects
        for _, repoName := range workItemRepos {
            if repoName == project.Name {
                // Project is in the list, check if it supports PRs
                return supportsPRs(project.RepoURL)
            }
        }
        // Project not in work item's repos list, skip PR creation
        return false
    }

    // Priority 3: Check explicit project config
    if project.DraftPR != nil {
        return *project.DraftPR
    }

    // Priority 4: Check workspace default
    if config.Workspace.DraftPR != nil {
        return *config.Workspace.DraftPR
    }

    // Priority 5: Default: enabled, but verify repo supports PRs
    return supportsPRs(project.RepoURL)
}
```

**Rationale**
- Removes dependency on external CLIs
- Works in headless/CI/agent environments
- Enables better error handling and retries
- Aligns with Kira’s goal of being self-contained

#### API Behavior Notes

**GitHub**
- Draft PR creation uses the REST API `PullRequests.Create` with `Draft: true`
- Converting Draft → Ready and Ready → Draft uses GraphQL mutations:
  - `markPullRequestReadyForReview`
  - `convertPullRequestToDraft`

**GitLab**
- Draft MR creation uses the REST API `MergeRequests.Create` with `Draft: true` (GitLab 14.0+)
- For older GitLab versions, uses title prefix: `Draft: ` or `WIP: `
- Converting Draft ↔ Ready uses `MergeRequests.Update` with draft flag
- GitLab GraphQL mutations available for advanced operations

**Branch Push Requirement**
- All platforms require the branch to exist on the remote before PR/MR creation
- Kira automatically pushes branches with `git push -u origin <branch-name>` after local creation
- This ensures remote branch availability and sets up tracking for future pushes

### Configuration Schema Updates

Add to `WorkspaceConfig`:
```go
type WorkspaceConfig struct {
    // ... existing fields ...
    DraftPR         *bool           `yaml:"draft_pr"`         // Optional: defaults to true
    GitPlatform     string          `yaml:"git_platform"`     // Optional: github, gitlab, auto (default: auto)
    GitBaseURL      string          `yaml:"git_base_url"`     // Optional: for GHE/GitLab self-hosted
    Projects        []ProjectConfig `yaml:"projects"`
}
```

Add to `ProjectConfig`:
```go
type ProjectConfig struct {
    // ... existing fields ...
    DraftPR         *bool           `yaml:"draft_pr"`         // Optional: false to skip PR creation for this project
    GitPlatform     string          `yaml:"git_platform"`     // Optional project platform override
    GitBaseURL      string          `yaml:"git_base_url"`     // Optional project base URL override
}
```

### Command Flow Updates

Modify `executeGitOperations` in `start.go`:
1. Existing worktree/branch creation (for ALL projects)
2. **NEW**: Check `--no-draft-pr` flag first - if present, skip all push/PR operations and proceed to step 4
3. **NEW**: Extract `repos` field from work item metadata (if present)
4. **NEW**: For each project, determine if draft PR should be created (priority order):
   - **Priority 1**: If `--no-draft-pr` flag is set: Skip push and PR creation for ALL projects (already checked in step 2)
   - **Priority 2**: If work item has `repos` field, only create PRs for projects listed there
   - **Priority 3**: If `draft_pr: false`: Skip push and PR creation (explicit opt-out)
   - **Priority 4**: If repo remote isn't GitHub/GitLab: Auto-skip push and PR creation (doesn't support PRs)
   - **Priority 5**: If `draft_pr: true` or unset (default) AND repo supports PRs: Push branch to remote, then create draft PR
5. Existing IDE/setup execution (for ALL projects)

**Work Item Metadata Extraction:**
```go
type WorkItemMetadata struct {
    // ... existing fields ...
    Repos []string `yaml:"repos"` // Optional: list of project names affected by this work
}

func extractWorkItemRepos(workItemPath string) ([]string, error) {
    // Parse YAML front matter and extract repos field
    workItem, err := parseWorkItemFile(workItemPath)
    if err != nil {
        return nil, err
    }

    // Check if repos field exists in Fields map
    if repos, ok := workItem.Fields["repos"].([]interface{}); ok {
        var result []string
        for _, r := range repos {
            if name, ok := r.(string); ok {
                result = append(result, name)
            }
        }
        return result, nil
    }
    return nil, nil // No repos field - use config defaults
}
```

**Auto-Detection Logic:**
- Check git remote URL to determine if repo supports PRs (GitHub/GitLab patterns)
- If unsupported platform detected, automatically skip push/PR without requiring explicit config
- This prevents errors when working with repos that don't have PR/MR functionality

### Error Handling Strategy

```go
func pushBranchAndCreateDraftPR(ctx *StartContext) error {
    // Check --no-draft-pr flag first (highest priority)
    if ctx.Flags.NoDraftPR {
        return nil // Skip all push/PR operations
    }

    if shouldSkipDraftPR(ctx) {
        return nil
    }

    // Step 1: Push branch to remote (required for PR creation)
    fmt.Printf("Pushing branch %s to remote...\n", ctx.BranchName)
    if err := pushBranchToRemote(ctx); err != nil {
        return fmt.Errorf("failed to push branch: %w", err)
    }
    fmt.Printf("Branch pushed successfully.\n")

    // Step 2: Attempt PR creation
    err := createGitHubDraftPR(ctx)
    if err != nil {
        fmt.Printf("Warning: Failed to create draft PR: %v\n", err)
        fmt.Println("Branch pushed successfully. PR can be created manually or with `kira pr create`.")
        return nil // Don't fail the start operation
    }

    fmt.Printf("Draft PR created: %s\n", prURL)
    return nil
}

func pushBranchToRemote(ctx *StartContext) error {
    // Use git push -u origin <branch-name> to set upstream
    cmd := exec.Command("git", "push", "-u", "origin", ctx.BranchName)
    cmd.Dir = ctx.WorktreeRoot // Run from worktree directory
    if output, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("git push failed: %s", string(output))
    }
    return nil
}
```

### Authentication Detection

```go
// Auth precedence by platform/domain:
// 1) Domain-specific token from `kira auth login` stored in OS credential store
// 2) Platform-specific env var (GITHUB_TOKEN or GITLAB_TOKEN)
// 3) If no auth and running interactively, offer inline device-flow login

// For polyrepo: check platform/domain combinations only for projects that need PRs
requiredCreds := detectRequiredCredentials(config, projects)
missingCreds := []CredentialKey{}

for _, key := range requiredCreds {
    token, source := loadStoredToken(key.Platform, key.Domain)
    if token == "" {
        // Try platform-specific env var as fallback
        envVar := getEnvVarForPlatform(key.Platform)
        token = os.Getenv(envVar)
        source = "env"
    }

    if token == "" {
        missingCreds = append(missingCreds, key)
    } else {
        validatedTokens[key] = TokenInfo{Token: token, Source: source}
    }
}

// Handle missing credentials
if len(missingCreds) > 0 {
    if isInteractiveTTY() {
        for _, key := range missingCreds {
            ok := promptYesNo("No credentials found for %s/%s. Run `kira auth login --platform %s` now?",
                            key.Platform, key.Domain, key.Platform)
            if ok {
                err := runAuthLoginForDomain(ctx, key.Platform, key.Domain)
                if err != nil {
                    return fmt.Errorf("auth login failed for %s/%s: %w", key.Platform, key.Domain, err)
                }
                // Reload token after successful auth
                token, source := loadStoredToken(key.Platform, key.Domain)
                validatedTokens[key] = TokenInfo{Token: token, Source: source}
            }
        }
    } else {
        // Non-interactive: list all required env vars
        var required []string
        for _, key := range missingCreds {
            envVar := getEnvVarForPlatform(key.Platform)
            required = append(required, envVar)
        }
        return fmt.Errorf("Missing credentials for draft PR creation. Set: %s (or use --no-draft-pr)",
                        strings.Join(required, ", "))
    }
}

ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
httpClient := oauth2.NewClient(ctx, ts)
gh := github.NewClient(httpClient)
_ = source // used for metrics/logging (never print token)
```

### Credential Storage & Management

Kira SHOULD store developer credentials using an OS-backed credential store:
- macOS: Keychain
- Windows: Credential Manager
- Linux: Secret Service (e.g., GNOME Keyring / KWallet), with a documented fallback strategy

The stored secret name SHOULD be platform-specific, e.g.:
- Service: `kira`
- Account: `github` or `gitlab` or `custom-domain`

Tokens SHOULD be rotated/re-authenticated when invalid/expired.

### PR Template Integration

Consider supporting PR templates by:
- Checking for `.github/PULL_REQUEST_TEMPLATE.md`
- Allowing work item content to be inserted into template variables
- Supporting custom PR templates in kira.yml

### Auth Command Behavior for Multiple Repos

**Auto-Detection Mode (`--platform auto`)**
- Scans all git remotes in the workspace (main repo + all polyrepo projects)
- Identifies unique platform/domain combinations
- Authenticates against each unique platform/domain found
- Stores credentials hierarchically by platform/domain

**Example Polyrepo Scenario**
```
Workspace with projects:
├── main (github.com/myorg/main)
├── frontend (github.com/myorg/frontend)
├── backend (gitlab.com/myorg/backend)
└── mobile (github.enterprise.com/mobile)

Result: Authenticates against github.com, gitlab.com, and github.enterprise.com
Stored as: kira/github/github.com, kira/gitlab/gitlab.com, kira/github/github.enterprise.com
```

### Future Extensions

- Custom PR templates
- Automatic reviewer assignment
- PR labels and milestones
- Integration with CI/CD pipelines
- Token rotation and renewal automation

### Testing Strategy

- Unit tests for PR/MR content generation
- Integration tests against mocked platform API servers (GitHub REST + GraphQL, GitLab REST)
- E2E tests with real repositories (requires `GITHUB_TOKEN` or `GITLAB_TOKEN`)
- Error scenario testing (network failures, auth issues, platform differences)
- Unit tests for credential load/precedence logic (stored vs env, platform/domain-specific)
- Integration tests for auth login/status/logout flows (with test doubles for credential store)
- Platform detection and auto-configuration tests
- Multi-repo credential detection and management tests (polyrepo scenarios)
- Cross-platform authentication flow tests
- Branch push testing (success/failure scenarios)
- Push-then-PR creation integration tests

## Implementation Plan: Commit Breakdown

This feature should be implemented incrementally across multiple commits to maintain code quality, testability, and reviewability.

### Commit 1: Configuration Schema and Validation
**Goal**: Add configuration fields without changing behavior

**Changes**:
- Add `DraftPR *bool` to `WorkspaceConfig` in `internal/config/config.go`
- Add `DraftPR *bool`, `GitPlatform string`, `GitBaseURL string` to `ProjectConfig`
- Update config loading/merging logic to handle new fields
- Add config validation for new fields
- Update `config_test.go` with tests for new fields
- Ensure backward compatibility (nil = default behavior)

**Why first**: Foundation for everything else. Can be tested independently.

**Testing**: Unit tests for config loading, validation, defaults

---

### Commit 2: Work Item Metadata Extraction
**Goal**: Extract `repos` field from work item YAML front matter

**Changes**:
- Extend `extractWorkItemMetadata()` or create `extractWorkItemRepos()` in `internal/commands/start.go`
- Update `workItemMetadata` struct to include `repos []string`
- Add parsing logic for `repos` field from YAML front matter
- Add unit tests for metadata extraction

**Why second**: Needed for decision logic, but doesn't require external dependencies.

**Testing**: Unit tests with various work item YAML structures

---

### Commit 3: Flag Parsing and Basic Skip Logic
**Goal**: Add `--no-draft-pr` flag and basic skip detection

**Changes**:
- Add `NoDraftPR bool` to `StartFlags` struct
- Add flag definition in `startCmd` init
- Create `shouldSkipDraftPR()` function that checks flag first
- Wire flag into `StartContext`
- Add basic skip logic that returns early (no-op for now)

**Why third**: Simple, testable, establishes the flag priority logic.

**Testing**: Unit tests for flag parsing, integration tests for flag behavior

---

### Commit 4: Platform Detection and Auto-Detection
**Goal**: Detect Git platform (GitHub/GitLab) from remote URLs

**Changes**:
- Create `detectPlatform()` function
- Create `detectPlatformAndDomain()` function
- Create `supportsPRs()` function to check if platform supports PRs
- Add platform detection to `StartContext` building
- Handle auto-detection logic (GitHub.com, GitLab.com, custom domains)

**Why fourth**: Needed for PR creation, but doesn't require API clients yet.

**Testing**: Unit tests with various remote URL patterns

---

### Commit 5: Decision Logic for PR Creation
**Goal**: Implement priority-based decision logic for which repos get PRs

**Changes**:
- Create `shouldCreateDraftPR()` function with full priority logic:
  1. `--no-draft-pr` flag check
  2. Work item `repos` field check
  3. Project-level `draft_pr` config
  4. Workspace-level `draft_pr` config
  5. Auto-detection (platform support)
- Create `getProjectOperations()` helper
- Wire into `StartContext` building
- Add comprehensive unit tests

**Why fifth**: Core logic that ties everything together, but still no external dependencies.

**Testing**: Extensive unit tests for all priority scenarios

---

### Commit 6: Branch Push Functionality
**Goal**: Push branches to remote before PR creation

**Changes**:
- Create `pushBranchToRemote()` function
- Add push logic to `executeGitOperations()` flow
- Handle push failures gracefully
- Add push status to output messages
- Only push for projects that need PRs (based on decision logic)

**Why sixth**: Required for PR creation, but uses existing git commands (no new dependencies).

**Testing**: Integration tests with git repos, unit tests with mocked git commands

---

### Commit 7: Authentication Infrastructure (Part 1 - Storage)
**Goal**: Secure credential storage using OS keychain

**Changes**:
- Add credential storage package (e.g., `internal/auth/storage.go`)
- Implement OS-specific storage backends (Keychain, Credential Manager, Secret Service)
- Create `storeToken()`, `loadToken()`, `deleteToken()` functions
- Add credential key structure (`platform/domain`)
- Add fallback to encrypted file storage

**Why seventh**: Needed for API calls, but can be tested independently.

**Testing**: Unit tests with test doubles for credential stores

---

### Commit 8: Authentication Infrastructure (Part 2 - Device Flow)
**Goal**: Browser-based OAuth device flow for GitHub/GitLab

**Changes**:
- Add `internal/auth/deviceflow.go` package
- Implement GitHub device flow
- Implement GitLab device flow
- Create `kira auth login` command
- Create `kira auth status` command
- Create `kira auth logout` command
- Add interactive prompts and browser opening

**Why eighth**: Complex but isolated feature. Can be tested with mocked OAuth servers.

**Testing**: Integration tests with OAuth test servers, unit tests for flow logic

---

### Commit 9: GitHub API Client and Draft PR Creation
**Goal**: Create draft PRs on GitHub

**Changes**:
- Add `github.com/google/go-github/v61/github` dependency
- Create `internal/git/github.go` package
- Implement `CreateDraftPR()` function
- Add PR content generation (title, body from work item)
- Wire into `executeGitOperations()` flow
- Handle GitHub API errors

**Why ninth**: First platform implementation. Can test with GitHub API mocks.

**Testing**: Unit tests with mocked GitHub client, integration tests with test repos

---

### Commit 10: GitLab API Client and Draft MR Creation
**Goal**: Create draft MRs on GitLab

**Changes**:
- Add `gitlab.com/gitlab-org/api/client-go` dependency
- Create `internal/git/gitlab.go` package
- Implement `CreateDraftMR()` function
- Handle GitLab-specific differences (MR vs PR, draft flag)
- Wire into platform detection logic
- Handle GitLab API errors

**Why tenth**: Second platform. Can reuse patterns from GitHub implementation.

**Testing**: Unit tests with mocked GitLab client, integration tests with test repos

---

### Commit 11: Multi-Platform Integration and Error Handling
**Goal**: Integrate all pieces with proper error handling

**Changes**:
- Create `GitPlatform` interface
- Refactor to use platform abstraction
- Add comprehensive error handling throughout
- Add retry logic for transient failures
- Improve user-facing error messages
- Add logging/metrics (without exposing secrets)

**Why eleventh**: Integration commit that ties everything together.

**Testing**: End-to-end tests, error scenario tests

---

### Commit 12: Polyrepo Support and Credential Detection
**Goal**: Handle multiple repos with different platforms/domains

**Changes**:
- Update `detectRequiredCredentials()` to use decision logic
- Handle multiple platform/domain combinations
- Update auth flow to handle polyrepo scenarios
- Add credential detection per project
- Handle partial failures (some repos succeed, some fail)

**Why twelfth**: Complex polyrepo scenarios. Builds on all previous work.

**Testing**: Polyrepo integration tests, multi-platform scenarios

---

### Commit 13: Documentation and Examples
**Goal**: Document the feature

**Changes**:
- Update README with new flag and configuration
- Add examples to kira.yml comments
- Update command help text
- Add migration guide (if needed)
- Update CHANGELOG

**Why last**: Documentation should reflect final implementation.

---

### Testing Strategy Per Commit

Each commit should:
1. Have unit tests for new functionality
2. Pass existing tests (no regressions)
3. Be reviewable independently
4. Have clear commit messages explaining the "why"

### Suggested First Commit

Start with **Commit 1: Configuration Schema and Validation** because:
- ✅ No external dependencies
- ✅ Can be tested in isolation
- ✅ Establishes foundation
- ✅ Backward compatible (doesn't break existing behavior)
- ✅ Easy to review

## Release Notes

### New Features
- **Draft PR/MR Creation**: `kira start` now creates draft pull requests/merge requests by default, automatically pushing branches first
- **Multi-Platform Support**: Native support for GitHub (including Enterprise) and GitLab (including self-hosted)
- **Developer Auth Flow**: `kira auth login/status/logout` for browser-based sign-in without manually managing tokens, with support for multiple platforms and domains
- **Skip Option**: Use `--no-draft-pr` flag to skip PR/MR creation when needed
- **Configuration Support**: Configure platform and disable draft PRs per workspace/project in kira.yml
- **Polyrepo Support**: Creates draft PRs/MRs across multiple repositories in polyrepo workspaces
- **Work Item Metadata Override**: Work items can declare affected repositories via `repos` field in YAML front matter, overriding project-level configuration

### Improvements
- **Streamlined Workflow**: Automatic PR creation eliminates manual step in development process
- **Error Resilience**: PR creation failures don't prevent worktree creation
- **Clear Feedback**: Better status messages during PR creation process

### Technical Changes
- Added native multi-platform Git API integration (Go) for PR/MR management
- Support for GitHub and GitLab APIs with platform-specific implementations
- Extended workspace and project configuration schema for platform settings
- No dependency on external CLIs for PR/MR functionality
- Added developer-friendly auth flow with device code login and secure credential storage, supporting multiple platforms/domains for polyrepo scenarios, with `GITHUB_TOKEN`/`GITLAB_TOKEN` fallback for CI/headless use cases
