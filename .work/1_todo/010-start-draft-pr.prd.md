---
id: 010
title: start draft pr
status: todo
kind: prd
assigned:
estimate: 0
created: 2026-01-14
due: 2026-01-14
tags: []
---

# start draft pr

Extend the `kira start` command to create draft pull requests by default when starting work on a new task, using **GitHub only** and **KIRA_GITHUB_TOKEN** for authentication. Interactive auth and GitLab support are separate work items.

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

However, it does not integrate with GitHub to create pull requests, requiring manual PR creation after work begins.

### Proposed Solution (this PRD)

Extend the `kira start` command to create draft pull requests by default for **GitHub** repositories:
- Creates worktree and branch locally
- Pushes the branch to the remote repository
- Creates a draft pull request automatically after the branch is pushed (GitHub REST API)
- Uses the work item title and description as PR title/body
- **Authentication**: `KIRA_GITHUB_TOKEN` environment variable only (no interactive login in this PRD)
- Provides a `--no-draft-pr` flag to skip PR creation when needed
- Supports configuration-driven behavior (can be disabled per workspace/project)
- If the remote is not GitHub, skip push and PR creation (no error)

**Out of scope / follow-on work items:**
- **031** (github interactive auth): `kira auth login/status/logout`, device flow, credential storage, auto-prompt on start for GitHub
- **033** (gitlab auth token): Draft MR creation using `KIRA_GITLAB_TOKEN` (same token method for GitLab)
- **032** (gitlab interactive auth): GitLab interactive auth and login commands

### Impact

- **For developers**: Zero-friction workflow from idea to visible draft PR when `KIRA_GITHUB_TOKEN` is set
- **For teams**: Automatic visibility into all active work streams on GitHub
- **For agents**: Native integration with GitHub; CI/headless can use env token

## Requirements

### Functional Requirements

#### FR1: Draft PR Creation
The `kira start` command SHALL create a draft pull request by default after successfully creating the worktree and branch **for GitHub remotes only**, unless the `--no-draft-pr` flag is provided or `workspace.draft_pr` is explicitly disabled in kira.yml.

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
  git_platform: github  # Options: github, auto (default: auto) — this PRD implements github only
  git_base_url: https://github.example.com  # For GHE (optional, auto-detected if not set)
```

#### FR3b: Selective Operations for Polyrepo Projects
For polyrepo workspaces, the command SHALL support multiple mechanisms for selective PR creation **for GitHub remotes**:

**Mechanism 1: Work Item Metadata Override (Highest Priority)**
- Work items MAY include an optional `repos` field in YAML front matter listing affected repositories
- When present, only repositories listed in the `repos` field SHALL have draft PRs created (for GitHub remotes only)
- Repository names in `repos` MUST match project names from `workspace.projects`

**Mechanism 2: Per-Project Configuration**
- Per-project `draft_pr: false` to skip push and PR creation for that project
- **Auto-detection**: If a repo's remote is not GitHub, skip push and PR creation for that repo (no error)

**Natural Behavior (Priority Order):**
1. **`--no-draft-pr` flag** (highest priority): If present, skip push and PR creation for ALL repos
2. **Work item `repos` field**: If present, only create PRs for listed repos (and only if GitHub)
3. **Project-level `draft_pr: false`**: Skip push and PR creation for that project
4. **Project-level `draft_pr: true` or unset**: Push branch + create draft PR for GitHub remotes (default)
5. **Non-GitHub remote**: Skip push and PR creation for that repo
6. **Worktrees always created**: Local worktrees are always created regardless of PR settings

#### FR4: Authentication (this PRD — token only)
The command SHALL support authentication for **GitHub** via:
- **`KIRA_GITHUB_TOKEN`** environment variable only

The command SHALL:
- Use `KIRA_GITHUB_TOKEN` when creating draft PRs for GitHub remotes
- Fail gracefully with a clear error message if `KIRA_GITHUB_TOKEN` is not set when draft PR creation is requested for a GitHub repo (or suggest `--no-draft-pr` to skip)
- Never log credentials or token values

Interactive login, credential storage, and `kira auth login/status/logout` are **out of scope** (see 031).

#### FR5: Branch Push Requirement
Before creating draft PRs, the command SHALL push the newly created branch to the remote repository. This ensures:
- The branch exists on the remote for PR creation
- Early visibility of work-in-progress branches

**Selective Push Logic:**
- **If `draft_pr: true` or unset** and remote is GitHub: Push branch to remote (required for PR creation)
- **If `draft_pr: false`**: Skip branch push
- **If remote is not GitHub**: Skip push and PR creation

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

#### FR8: Platform Support (this PRD)
The implementation SHALL support **GitHub only** (GitHub.com and GitHub Enterprise Server):
- REST API for PR creation with `draft: true`
- Base URL auto-detection: `github.com` for public, configurable for GHE
- If remote is not GitHub, skip push and PR creation (no error)

GitLab and other platforms are out of scope (see 033, 032).

### Non-Functional Requirements

#### NFR1: Performance
PR creation SHALL NOT significantly impact the overall `kira start` execution time. PR creation SHALL happen after worktree setup is complete and SHALL not block IDE opening.

#### NFR2: Security
- Token SHALL be read only from environment; never log token values
- SHALL follow secure coding practices as defined in `docs/security/golang-secure-coding.md`

#### NFR3: Backward Compatibility
The feature SHALL be enabled by default but SHALL provide opt-out mechanisms. Existing `kira start` behavior changes to include draft PR creation when token is set, but users can disable this via `--no-draft-pr` flag or configuration.

#### NFR4: Observability
The command SHALL provide clear feedback about PR creation status, including:
- Success confirmation with PR URL
- Clear error messages for failures (including missing `KIRA_GITHUB_TOKEN`)
- Progress indicators during PR creation

## Acceptance Criteria

### AC1: Default Draft PR Creation
**Given** a valid work item exists and the remote is GitHub and `KIRA_GITHUB_TOKEN` is set
**When** `kira start <work-item-id>` is executed
**Then** the branch is pushed to remote AND a draft PR is created with correct title, body, and branch targeting trunk

### AC2: Skip Draft PR Creation
**Given** a valid work item exists
**When** `kira start <work-item-id> --no-draft-pr` is executed
**Then** no draft PR is created, but worktree and branch are created successfully

### AC3: PR Content Accuracy
**Given** a work item with title "Add user authentication" and description
**When** a draft PR is created
**Then** PR title is "010: Add user authentication" and body contains work item content

### AC4: Error Resilience
**Given** PR creation fails due to network issues
**When** `kira start <work-item-id>` is executed
**Then** worktree and branch are still created successfully

### AC5: Authentication — Token
**Given** `KIRA_GITHUB_TOKEN` is set and valid
**When** draft PR creation is attempted for a GitHub remote
**Then** the draft PR is created using the token

### AC6: Authentication — No Token
**Given** `KIRA_GITHUB_TOKEN` is missing or invalid
**When** draft PR creation is attempted for a GitHub remote
**Then** a clear error message guides the user to set `KIRA_GITHUB_TOKEN` or use `--no-draft-pr` to disable draft PR creation

### AC7: Non-GitHub Remote
**Given** the repository remote is not GitHub (e.g. GitLab, or unknown)
**When** `kira start <work-item-id>` is executed
**Then** worktree and branch are created; push and PR creation are skipped for that repo (no error)

### AC8: Polyrepo Support (GitHub only)
**Given** a polyrepo workspace with multiple projects (GitHub remotes)
**When** `kira start <work-item-id>` is executed with `KIRA_GITHUB_TOKEN` set
**Then** worktrees are created for all projects
**And** branches are pushed and draft PRs are created only for projects where `draft_pr: true` or unset (default)

### AC9: Selective Polyrepo Operations
**Given** a polyrepo workspace with a project configured with `draft_pr: false`
**When** `kira start <work-item-id>` is executed
**Then** worktree and branch are created for that project
**And** branch is NOT pushed to remote
**And** no draft PR is created for that project
**And** other projects with `draft_pr: true` or unset still get PRs created normally

### AC10: Work Item Metadata Override
**Given** a polyrepo workspace with projects: `frontend`, `backend`, `infrastructure`, `docs` (all GitHub)
**And** a work item with `repos: [frontend, backend]` in its YAML front matter
**When** `kira start <work-item-id>` is executed
**Then** worktrees and branches are created for ALL projects
**And** branches are pushed and draft PRs are created ONLY for `frontend` and `backend`
**And** branches are not pushed and no draft PRs are created for `infrastructure` or `docs`

### AC11: No Draft PR Flag Override
**Given** a polyrepo workspace with projects: `frontend`, `backend`
**And** a work item with `repos: [frontend, backend]` in its YAML front matter
**And** all projects configured with `draft_pr: true`
**When** `kira start <work-item-id> --no-draft-pr` is executed
**Then** worktrees and branches are created for ALL projects
**And** branches are NOT pushed to remote for any project
**And** no draft PRs are created for any project

### AC12: Branch Push Before PR Creation
**Given** a valid work item exists and remote is GitHub and `KIRA_GITHUB_TOKEN` is set
**When** `kira start <work-item-id>` is executed
**Then** the newly created branch is pushed to the remote repository before draft PR creation
**And** if push fails, PR creation is skipped with appropriate error message

## Implementation Notes

### GitHub Integration
- Use `github.com/google/go-github/v61/github` for REST operations (creating the PR with `draft: true`)
- Use `golang.org/x/oauth2` with `StaticTokenSource` from `KIRA_GITHUB_TOKEN`
- Draft PR creation: `PullRequests.Create` with `Draft: true`
- Base URL: auto-detect from remote (`github.com` or config `git_base_url` for GHE)

### Configuration Schema
Add to `WorkspaceConfig`: `DraftPR *bool`, `GitPlatform string`, `GitBaseURL string`, `Projects`
Add to `ProjectConfig`: `DraftPR *bool`, `GitPlatform string`, `GitBaseURL string`

### Command Flow
1. Existing worktree/branch creation (for ALL projects)
2. Check `--no-draft-pr` flag — if present, skip all push/PR and go to IDE step
3. Extract `repos` from work item metadata (if present)
4. For each project: if GitHub remote and draft PR desired (per priority), push branch then create draft PR using `KIRA_GITHUB_TOKEN`
5. Existing IDE/setup execution

### Error Handling
- Push failure: stop, clear message, no PR attempt
- PR creation failure after push: continue (IDE etc.), clear message, do not fail start

### Testing Strategy
- Unit tests for PR content generation, config, decision logic, platform detection (GitHub vs non-GitHub)
- Integration tests with mocked GitHub API
- E2E tests with real GitHub repo (requires `KIRA_GITHUB_TOKEN`)
- Branch push success/failure scenarios

## Implementation Plan: Commit Breakdown

### Commit 1: Configuration Schema and Validation
Add `DraftPR`, `GitPlatform`, `GitBaseURL` to workspace/project config; validation; backward compatibility (nil = default).

### Commit 2: Work Item Metadata Extraction
Extract `repos` from work item YAML front matter; add to metadata struct; unit tests.

### Commit 3: Flag Parsing and Basic Skip Logic
Add `--no-draft-pr` flag; `shouldSkipDraftPR()`; wire into StartContext.

### Commit 4: GitHub Detection
Detect GitHub from remote URL; if not GitHub, skip push/PR (no error). No GitLab in this PRD.

### Commit 5: Decision Logic for PR Creation
`shouldCreateDraftPR()` with full priority order (flag, work item repos, project draft_pr, workspace draft_pr, GitHub-only).

### Commit 6: Branch Push Functionality
Push branch to remote for projects that need PRs; handle push failures.

### Commit 7: GitHub API Client and Draft PR Creation
Add go-github dependency; `internal/git/github.go`; `CreateDraftPR()`; PR content from work item; wire into start flow; use `KIRA_GITHUB_TOKEN` only.

### Commit 8: Error Handling and Polyrepo (GitHub only)
Error handling; polyrepo: only GitHub repos get push/PR; clear message when token missing.

### Commit 9: Documentation
README, kira.yml examples, command help, CHANGELOG.

## Release Notes

### New Features
- **Draft PR Creation**: `kira start` creates draft pull requests by default for GitHub remotes, after pushing the branch
- **Authentication**: Uses `KIRA_GITHUB_TOKEN` environment variable (CI/headless friendly)
- **Skip Option**: `--no-draft-pr` flag to skip PR creation
- **Configuration**: `workspace.draft_pr`, per-project `draft_pr`, `git_platform`, `git_base_url` in kira.yml
- **Polyrepo**: Draft PRs for GitHub projects; work item `repos` field to limit which repos get PRs
- **Non-GitHub remotes**: Skipped for push/PR (no error); GitLab support is a follow-on (033, 032)

### Improvements
- **Streamlined Workflow**: Automatic PR creation eliminates manual step when token is set
- **Error Resilience**: PR creation failures do not prevent worktree creation
- **Clear Feedback**: Status messages and clear error when token is missing

### Technical Changes
- GitHub REST API integration (Go) for draft PR creation
- No dependency on external CLIs for PR functionality
- Extended workspace/project config for draft_pr and platform settings
