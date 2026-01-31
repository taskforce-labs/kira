---
id: 033
title: gitlab auth token
status: backlog
kind: prd
assigned:
estimate: 0
created: 2026-01-31
due: 2026-01-31
tags: []
---

# gitlab auth token

Extend the start-draft-PR flow to support **GitLab** using the same token-based method as GitHub: create draft merge requests when the remote is GitLab, using the **`KIRA_GITLAB_TOKEN`** environment variable. No interactive login in this PRD (see 032 for GitLab interactive auth).

**Depends on:** 010 (start draft PR with KIRA_GITHUB_TOKEN) — this PRD adds GitLab as a second platform with the same token-only auth pattern.

## Context

### Problem Statement

PRD 010 adds draft pull request creation for GitHub using `KIRA_GITHUB_TOKEN`. Many teams use GitLab (cloud or self-hosted) and need the same workflow: `kira start` creates a branch, pushes it, and opens a draft merge request, using an environment token for authentication.

### Proposed Solution

- **Platform detection:** When the git remote is GitLab (e.g. `gitlab.com`, or configured custom domain), treat the repo as GitLab for push/PR logic.
- **Draft MR creation:** Use GitLab REST API to create a merge request with draft/WIP state (e.g. `draft: true` for GitLab 14+, or WIP prefix for older).
- **Authentication:** `KIRA_GITLAB_TOKEN` environment variable only (same pattern as `KIRA_GITHUB_TOKEN` for GitHub).
- **Configuration:** Reuse/align with 010’s config: `workspace.git_platform`, `workspace.git_base_url`, per-project overrides for GitLab URLs.
- **Polyrepo:** Repos with GitHub remotes use `KIRA_GITHUB_TOKEN`; repos with GitLab remotes use `KIRA_GITLAB_TOKEN`; if a token is missing for a given platform, clear error or skip draft PR for that repo (per 010 behavior).

### Scope

- **In scope:** GitLab.com and self-hosted GitLab; draft MR via API; `KIRA_GITLAB_TOKEN`; platform detection; polyrepo (GitHub + GitLab in same workspace).
- **Out of scope:** GitLab interactive login / credential storage (see 032).

## Requirements

### Functional Requirements

#### FR1: GitLab Remote Detection
The command SHALL detect GitLab remotes from:
- URL patterns (e.g. `gitlab.com`, or custom host)
- Optional config: `workspace.git_platform: gitlab`, `workspace.git_base_url`, per-project `git_base_url` for self-hosted

When the remote is GitLab, push and draft MR creation SHALL be attempted (subject to `draft_pr` and `--no-draft-pr` as in 010).

#### FR2: Draft MR Creation
For GitLab remotes, the command SHALL:
- Push the newly created branch to the remote (same as 010).
- Call GitLab REST API to create a merge request in draft/WIP state (e.g. `MergeRequests.Create` with `Draft: true` where supported, or title prefix for older GitLab).
- Use work item title and description for MR title and body (same content rules as 010 for PR).

#### FR3: Authentication
- **`KIRA_GITLAB_TOKEN`** environment variable only for GitLab API calls.
- If draft MR is requested for a GitLab repo and `KIRA_GITLAB_TOKEN` is missing or invalid: fail with clear message to set `KIRA_GITLAB_TOKEN` or use `--no-draft-pr` (or skip that repo in polyrepo with clear message).
- Never log token values.

#### FR4: Configuration
Reuse 010-style configuration for GitLab:
- `workspace.git_platform: gitlab` or `auto` (auto-detect from remote).
- `workspace.git_base_url` / per-project `git_base_url` for self-hosted GitLab.
- Same `draft_pr` and work item `repos` behavior: only create MRs for repos where draft MR is enabled and remote is GitLab.

#### FR5: Polyrepo (GitHub + GitLab)
In a workspace with both GitHub and GitLab projects:
- GitHub projects: use `KIRA_GITHUB_TOKEN` (and 010 behavior).
- GitLab projects: use `KIRA_GITLAB_TOKEN` (this PRD).
- Missing token for one platform: skip draft PR/MR for that platform only; clear message; worktrees and branches still created.

#### FR6: Error Handling
- Push failure: same as 010 (stop, clear message, no MR attempt).
- MR creation failure after push: do not fail the whole start; continue with IDE/setup; show clear error.

### Non-Functional Requirements

- **Security:** No logging of tokens; follow `docs/security/golang-secure-coding.md`.
- **Consistency:** Same user-facing behavior as 010 (flag, config, work item `repos`) applied to GitLab.

## Acceptance Criteria

### AC1: Draft MR with Token
**Given** a valid work item, remote is GitLab, and `KIRA_GITLAB_TOKEN` is set
**When** `kira start <work-item-id>` is executed
**Then** the branch is pushed and a draft MR is created with correct title and body

### AC2: No Token — Clear Error
**Given** remote is GitLab and `KIRA_GITLAB_TOKEN` is not set
**When** `kira start <work-item-id>` would create a draft MR
**Then** a clear error message instructs the user to set `KIRA_GITLAB_TOKEN` or use `--no-draft-pr`

### AC3: Skip Draft MR
**Given** remote is GitLab
**When** `kira start <work-item-id> --no-draft-pr` is executed
**Then** worktree and branch are created; branch may be pushed if needed for other reasons, but no MR is created (align with 010: no push when no PR/MR).

### AC4: Polyrepo GitHub + GitLab
**Given** a polyrepo with both GitHub and GitLab projects, and both `KIRA_GITHUB_TOKEN` and `KIRA_GITLAB_TOKEN` set
**When** `kira start <work-item-id>` is executed
**Then** draft PRs are created for GitHub projects and draft MRs for GitLab projects

### AC5: Polyrepo — One Token Missing
**Given** a polyrepo with GitHub and GitLab projects; only `KIRA_GITHUB_TOKEN` is set
**When** `kira start <work-item-id>` is executed
**Then** draft PRs are created for GitHub projects; draft MRs are skipped for GitLab with clear message; worktrees/branches created for all

## Implementation Notes

- **API client:** Use GitLab REST API (e.g. `gitlab.com/gitlab-org/api/client-go`) for creating merge requests. Use `Draft: true` where supported; fallback to WIP/Draft prefix for older GitLab if needed.
- **Auth:** `golang.org/x/oauth2` with `StaticTokenSource` from `KIRA_GITLAB_TOKEN` (same pattern as 010 for GitHub).
- **Platform abstraction:** 010 can be extended with a small platform interface (e.g. `CreateDraftPR` vs `CreateDraftMR`) or separate code paths for GitHub and GitLab; ensure config and decision logic (flag, `repos`, `draft_pr`) are shared.
- **Base URL:** For GitLab.com use default; for self-hosted use `git_base_url` (e.g. `https://gitlab.company.com`).

## Release Notes

- **GitLab draft MR:** `kira start` can create draft merge requests on GitLab when `KIRA_GITLAB_TOKEN` is set.
- **Same token pattern:** Authentication via `KIRA_GITLAB_TOKEN` (CI/headless friendly), mirroring `KIRA_GITHUB_TOKEN` for GitHub.
- **Polyrepo:** Workspaces can mix GitHub and GitLab; each platform uses its own token; missing token for one platform skips only that platform’s PR/MR creation.
- **Configuration:** GitLab supported in `git_platform` and `git_base_url`; same `draft_pr` and work item `repos` behavior as GitHub.
- **Interactive GitLab auth:** Out of scope here; see 032 (gitlab interactive auth).
