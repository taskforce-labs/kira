---
id: 032
title: gitlab interactive auth
status: backlog
kind: prd
assigned:
estimate: 0
created: 2026-01-31
due: 2026-01-31
tags: []
---

# gitlab interactive auth

Add interactive authentication and login commands for **GitLab** (GitLab.com and self-hosted), mirroring the GitHub flow in 031: sign in via browser (OAuth device flow), store credentials securely, and optionally prompt from `kira start` when draft MR creation needs auth and no token is available.

**Depends on:** 031 (GitHub interactive auth) for pattern and credential storage; 033 (GitLab auth token) so that draft MR creation and `KIRA_GITLAB_TOKEN` already exist — this PRD adds stored credentials and interactive login for GitLab.

## Context

### Problem Statement

033 adds draft MR creation for GitLab using `KIRA_GITLAB_TOKEN`. As with GitHub, developers at a workstation often prefer to sign in once via the browser and have Kira reuse stored credentials instead of managing GitLab tokens manually.

### Proposed Solution

- **`kira auth login --platform gitlab`:** Browser-based OAuth device flow for GitLab; store token in the same OS credential store used in 031, with account key distinguishing GitLab (e.g. `gitlab` or `gitlab/<domain>`).
- **`kira auth status` / `kira auth logout`:** Extend to show and clear stored GitLab credentials (without revealing secrets).
- **Auto-prompt on start:** When `kira start` would create a draft MR for GitLab but neither stored credentials nor `KIRA_GITLAB_TOKEN` is available, and the session is interactive (TTY), prompt to run `kira auth login --platform gitlab`; if the user completes login, continue and create the draft MR in the same run.

### Scope

- **In scope:** GitLab.com and self-hosted GitLab; device flow; credential storage keys for GitLab; auth commands extended for GitLab; integration with `kira start` for GitLab MR creation.
- **Out of scope:** Changes to GitHub auth (031) or to token-only GitLab behavior (033).

## Requirements

### Functional Requirements

#### FR1: Auth Commands (GitLab)
Extend auth commands to support GitLab:

```bash
kira auth login [--platform github|gitlab|auto]
kira auth status [--platform github|gitlab|auto]
kira auth logout [--platform github|gitlab|auto]
```

When `--platform gitlab` or `auto` (and workspace has GitLab remotes): perform GitLab login/status/logout. When `auto`, support both GitHub and GitLab in one workspace (e.g. login for each unique platform/domain).

#### FR2: kira auth login (GitLab)
`kira auth login` for GitLab SHALL:
- Support `--platform gitlab` and, when `auto`, detect GitLab remotes (and optionally domain) from workspace.
- Initiate GitLab OAuth device flow (browser-based).
- Obtain a token suitable for GitLab API (e.g. merge request creation).
- Store the token in the same OS credential store as 031, with account key e.g. `gitlab` or `gitlab/<domain>` for self-hosted (e.g. `gitlab/gitlab.company.com`).
- Support custom base URL for self-hosted GitLab (config: `workspace.git_base_url` or per-project).

#### FR3: kira auth status / logout (GitLab)
- `kira auth status`: When `--platform gitlab` or `auto`, show whether stored GitLab credentials exist (without revealing secrets); show domain(s) (e.g. gitlab.com, gitlab.company.com).
- `kira auth logout --platform gitlab`: Remove stored GitLab credentials (all domains or specified domain if supported).

#### FR4: Credential Storage
- Same storage backend as 031 (Keychain / Credential Manager / Secret Service).
- Keys: service `kira`, account `gitlab` or `gitlab/<domain>`.
- Never log or print token values.

#### FR5: Auth Precedence in kira start (GitLab)
When `kira start` attempts draft MR creation for a GitLab remote, auth resolution SHALL be:
1. **Stored credentials** from `kira auth login` (GitLab) if present and valid.
2. **`KIRA_GITLAB_TOKEN`** environment variable (fallback).
3. **Interactive prompt** (if TTY): offer to run `kira auth login --platform gitlab`; if user completes login, retry draft MR in the same run.
4. **Non-interactive:** fail with clear message to set `KIRA_GITLAB_TOKEN` or run `kira auth login` (or use `--no-draft-pr`).

#### FR6: Auto-Prompt on Start (GitLab, Interactive)
When `kira start` would create a draft MR for GitLab but neither stored credentials nor `KIRA_GITLAB_TOKEN` is available, and the session is interactive (TTY):
- Prompt the user to run `kira auth login --platform gitlab` (or equivalent).
- If user completes login, continue and create the draft MR in the same run.
- If user declines, skip draft MR creation with clear warning; worktree and branch still created.

#### FR7: Polyrepo (GitHub + GitLab)
In a workspace with both GitHub and GitLab projects and interactive start:
- Missing GitHub auth: prompt for GitHub login (per 031).
- Missing GitLab auth: prompt for GitLab login (this PRD).
- Each platform/domain can be prompted independently; stored credentials and env tokens per platform unchanged.

### Non-Functional Requirements

- **Security:** Follow `docs/security/golang-secure-coding.md`; never log tokens.
- **Consistency:** Same UX pattern as 031 (login → status → logout → auto-prompt on start), applied to GitLab.

## Acceptance Criteria

### AC1: Login and Use on Start (GitLab)
**Given** no stored GitLab credentials and no `KIRA_GITLAB_TOKEN`
**When** user runs `kira auth login --platform gitlab` and completes the browser flow
**Then** GitLab credentials are stored
**And** a subsequent `kira start <work-item-id>` (GitLab remote) creates a draft MR without requiring `KIRA_GITLAB_TOKEN`

### AC2: Status Shows Stored GitLab Credentials
**Given** user has run `kira auth login --platform gitlab`
**When** user runs `kira auth status --platform gitlab`
**Then** status indicates that GitLab credentials are present (without revealing the token)

### AC3: Logout Removes GitLab Credentials
**Given** stored GitLab credentials exist
**When** user runs `kira auth logout --platform gitlab`
**Then** stored GitLab credentials are removed
**And** `kira auth status` no longer shows GitLab credentials for that domain

### AC4: Token Fallback Unchanged
**Given** `KIRA_GITLAB_TOKEN` is set
**When** draft MR creation is attempted for GitLab
**Then** token is used if no stored credentials, or per precedence (stored first, then env)

### AC5: Auto-Prompt (Interactive, GitLab)
**Given** no stored GitLab credentials and no `KIRA_GITLAB_TOKEN`, and session is interactive (TTY)
**When** `kira start <work-item-id>` attempts draft MR creation for a GitLab remote
**Then** user is prompted to run `kira auth login` for GitLab
**And** if user completes login, the draft MR is created in the same run

### AC6: Polyrepo — Both Platforms Interactive
**Given** polyrepo with GitHub and GitLab; no stored credentials; no env tokens; interactive TTY
**When** `kira start <work-item-id>` runs
**Then** user can be prompted for GitHub and/or GitLab login as needed; draft PRs/MRs created for platforms where auth is completed

## Implementation Notes

- **Device flow:** Use GitLab’s OAuth device flow (or equivalent); support GitLab.com and self-hosted (custom base URL).
- **Credential keys:** Reuse same credential store as 031; account key `gitlab` or `gitlab/<host>` to support multiple GitLab instances.
- **Integration:** In start command, for GitLab remotes, resolve token: stored credential for GitLab domain → `KIRA_GITLAB_TOKEN` → if TTY prompt for login and retry.

## Release Notes

- **GitLab interactive auth:** `kira auth login --platform gitlab` signs in to GitLab via browser and stores credentials securely.
- **Auth commands:** `kira auth status` and `kira auth logout` support `--platform gitlab` for viewing and clearing stored GitLab credentials.
- **Auto-prompt:** When `kira start` needs GitLab auth and none is available interactively, users can run `kira auth login` for GitLab inline and continue in the same run.
- **Precedence:** Stored GitLab credentials from `kira auth login` are used first; `KIRA_GITLAB_TOKEN` remains supported for CI and headless use.
