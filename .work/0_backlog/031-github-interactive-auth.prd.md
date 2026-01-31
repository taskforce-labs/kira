---
id: 031
title: github interactive auth
status: backlog
kind: prd
assigned:
created: 2026-01-31
tags: []
---

# github interactive auth

Add interactive authentication and login commands for GitHub so developers can sign in via the browser (device flow) and have credentials stored securely, instead of relying only on `KIRA_GITHUB_TOKEN`. When `kira start` needs GitHub auth and no token is available, prompt to run login.

**Depends on:** 010 (start draft PR with KIRA_GITHUB_TOKEN) — this PRD extends auth for GitHub to interactive + stored credentials.

## Context

### Problem Statement

PRD 010 delivers draft PR creation using the `KIRA_GITHUB_TOKEN` environment variable. That works well for CI and headless use, but developers at a workstation often prefer to sign in once (e.g. via browser) and have Kira reuse stored credentials instead of managing tokens manually.

### Proposed Solution

- **`kira auth login`** (GitHub): Browser-based OAuth device flow; store token securely in OS credential store.
- **`kira auth status`** / **`kira auth logout`**: Show or clear stored GitHub credentials (without revealing secrets).
- **Auto-prompt on start**: When `kira start` would create a draft PR but neither stored credentials nor `KIRA_GITHUB_TOKEN` is available, and the session is interactive (TTY), prompt to run `kira auth login`; if the user completes login, continue and create the draft PR in the same run.

### Scope

- **In scope:** GitHub only (GitHub.com and GHE). Credential storage; device flow; auth commands; integration with `kira start` for GitHub.
- **Out of scope:** GitLab interactive auth (see 032).

## Requirements

### Functional Requirements

#### FR1: Auth Commands
Kira SHALL provide:

```bash
kira auth login [--platform github|auto]
kira auth status [--platform github|auto]
kira auth logout [--platform github|auto]
```

For this PRD, only `github` (and `auto` when the workspace remote is GitHub) need be implemented. GitLab is 032.

#### FR2: kira auth login (GitHub)
`kira auth login` for GitHub SHALL:
- Auto-detect GitHub domain from workspace git remotes when `--platform auto` (default), or use `--platform github` for default GitHub.com.
- Initiate GitHub OAuth device flow (browser-based).
- Obtain a token suitable for GitHub REST/API (e.g. repo scope for PR creation).
- Store the token securely using OS credential store:
  - Service: `kira`
  - Account: `github` or `github/<domain>` for GHE (e.g. `github/github.example.com`)
- Support custom base URL for GitHub Enterprise via config (`workspace.git_base_url` or project-level).

#### FR3: kira auth status
`kira auth status` SHALL:
- When no `--platform` or `--platform auto`: show status for all stored credentials (this PRD: GitHub only).
- When `--platform github`: show whether valid stored GitHub credentials exist (without revealing secrets).
- Display which domain(s) have valid tokens (e.g. github.com, github.example.com).

#### FR4: kira auth logout
`kira auth logout` SHALL:
- When no `--platform`: remove all stored credentials (this PRD: GitHub only).
- When `--platform github`: remove stored GitHub credentials.
- Optionally support domain-specific logout for GHE (e.g. remove only `github/github.example.com`).

#### FR5: Credential Storage
- Use OS-backed credential store: macOS Keychain, Windows Credential Manager, Linux Secret Service (e.g. GNOME Keyring / KWallet), with a documented fallback if needed.
- Keys: service `kira`, account `github` or `github/<domain>`.
- Never log or print token values.

#### FR6: Auth Precedence in kira start
When `kira start` attempts draft PR creation for a GitHub remote, auth resolution SHALL be:
1. **Stored credentials** from `kira auth login` (if present and valid).
2. **`KIRA_GITHUB_TOKEN`** environment variable (fallback).
3. **Interactive prompt** (if TTY): offer to run `kira auth login`; if user completes login, retry draft PR in the same run.
4. **Non-interactive**: fail with clear message to set `KIRA_GITHUB_TOKEN` or run `kira auth login` (or use `--no-draft-pr`).

#### FR7: Auto-Prompt on Start (Interactive)
When `kira start` would create a draft PR for GitHub but neither stored credentials nor `KIRA_GITHUB_TOKEN` is available, and the command is running interactively (TTY):
- Prompt the user to run `kira auth login` (e.g. “No GitHub credentials. Run `kira auth login` now?”).
- If user agrees, run the login flow inline; on success, continue and create the draft PR without requiring a second `kira start`.
- If user declines, skip draft PR creation for that run with a clear warning; worktree and branch still created.

#### FR8: Non-Interactive
When not a TTY, do not block on prompts; fail with a clear message listing `KIRA_GITHUB_TOKEN` or `--no-draft-pr`.

### Non-Functional Requirements

- **Security:** Follow `docs/security/golang-secure-coding.md`; never log tokens.
- **Backward compatibility:** Existing use of `KIRA_GITHUB_TOKEN` continues to work; stored credentials take precedence when present.

## Acceptance Criteria

### AC1: Login and Use on Start
**Given** no stored credentials and no `KIRA_GITHUB_TOKEN`
**When** user runs `kira auth login` and completes the browser flow
**Then** credentials are stored
**And** a subsequent `kira start <work-item-id>` (GitHub remote) creates a draft PR without requiring `KIRA_GITHUB_TOKEN`

### AC2: Status Shows Stored Credentials
**Given** user has run `kira auth login` for GitHub
**When** user runs `kira auth status`
**Then** status indicates that GitHub credentials are present (without revealing the token)

### AC3: Logout Removes Credentials
**Given** stored GitHub credentials exist
**When** user runs `kira auth logout --platform github`
**Then** stored GitHub credentials are removed
**And** `kira auth status` no longer shows GitHub credentials

### AC4: Token Fallback Unchanged
**Given** `KIRA_GITHUB_TOKEN` is set
**When** draft PR creation is attempted
**Then** token is used if no stored credentials, or per precedence (stored first, then env)

### AC5: Auto-Prompt (Interactive)
**Given** no stored credentials and no `KIRA_GITHUB_TOKEN`, and session is interactive (TTY)
**When** `kira start <work-item-id>` attempts draft PR creation for GitHub
**Then** user is prompted to run `kira auth login`
**And** if user completes login, the draft PR is created in the same run

### AC6: No Prompt (Non-Interactive)
**Given** no stored credentials and no `KIRA_GITHUB_TOKEN`, and session is non-interactive (no TTY)
**When** `kira start <work-item-id>` would create a draft PR for GitHub
**Then** draft PR creation is skipped with a clear message to set `KIRA_GITHUB_TOKEN` or use `--no-draft-pr`

## Implementation Notes

- **Device flow:** Use GitHub’s OAuth device flow (request device code, show user code + URL, poll for token). Use `golang.org/x/oauth2` where applicable.
- **Credential store:** Use a Go library that abstracts Keychain / Credential Manager / Secret Service (e.g. keyring or similar); key format `kira` / `github` or `github/<domain>`.
- **Integration point:** In start command, before calling GitHub API, resolve token: load stored credential for detected GitHub domain; else use `KIRA_GITHUB_TOKEN`; else if TTY prompt for login and retry.

## Release Notes

- **GitHub interactive auth:** `kira auth login` signs in to GitHub via browser and stores credentials securely.
- **Auth commands:** `kira auth status` and `kira auth logout` for viewing and clearing stored GitHub credentials.
- **Auto-prompt:** When `kira start` needs GitHub auth and none is available interactively, users can run `kira auth login` inline and continue in the same run.
- **Precedence:** Stored credentials from `kira auth login` are used first; `KIRA_GITHUB_TOKEN` remains supported for CI and headless use.
