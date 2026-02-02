---
id: 034
title: ci update pr details
status: backlog
kind: prd
assigned:
created: 2026-02-01
tags: []
---

# ci update pr details

When a pull request is opened or updated (e.g. when code is pushed to the PR branch), the CI pipeline updates the PR's title and body from the linked work item so the PR always reflects the current work item. The workflow uses the **pull_request** event (opened, synchronize, reopened)—not push + API discovery—so GitHub tells us which PR to update. The PR body SHALL contain the **entire content** of the work item file.

**Depends on:** 010 (start draft PR) — Draft PRs are created with work-item-derived title and description; this PRD keeps that content in sync when the branch is pushed.

## Context

### Problem Statement

PRD 010 (start draft PR) introduces creating a draft PR from `kira start`, with title and description derived from the work item. After the PR is created, the author or an agent may update the work item (e.g. elaborate the PRD, change acceptance criteria). Those changes are not reflected on the PR unless someone manually edits the PR or runs a custom script. As a result, reviewers see stale PR descriptions and the "source of truth" (the work item) and the PR drift apart.

### Proposed Solution

- Add a **CI job** that runs on the **pull_request** event (types: opened, synchronize, reopened) for PRs targeting the trunk branch from kira config (`git.trunk_branch`). Trunk is always from config; no hardcoding. We use this event—not push + API discovery—so the event payload includes the PR number and the job updates that PR directly.
- The job **derives the work item ID** from the PR head branch name (same convention as 010: `{id}-{kebab-title}`).
- A **`kira current`** command derives the work item from the current branch name, looks for the work item file in the work folders per kira config (searching all status folders), and supports **`--title`** and **`--body`** flags to output the PR title and body separately. The CI job uses `kira current --title` and `kira current --body` to get those values and then updates the PR via the GitHub API.

This keeps the PR and the work item aligned whenever the branch is pushed.

### Scope

- **In scope:** GitHub only; a **`kira current`** command that derives work item from current branch and outputs `--title` and `--body` per work item (searching work folders per kira config); CI job that runs on `pull_request` (opened, synchronize, reopened) and uses `kira current --title` and `kira current --body` to update PR title and body (PR body = entire work item content); **no user-configured tokens** (automatic `GITHUB_TOKEN` only); a **dedicated, standalone** workflow file (`.github/workflows/update-pr-details.yml`); creating that workflow file on `kira init` when the user indicates GitHub.
- **Out of scope:** GitLab CI (separate work item); editing PR labels or reviewers from work item; triggering on `push` and discovering PRs by API (we use `pull_request` only); requiring `KIRA_GITHUB_TOKEN`, PAT, or repo secrets; read-only mode (output suggested body without updating the PR)—we support automatic `GITHUB_TOKEN` only and the job SHALL update the PR via API; adding this job to existing ci.yml or other workflows (we use a dedicated standalone file so it does not interfere with other CI).

## Requirements

### Functional Requirements

#### FR1: CI Trigger

- The job SHALL run on the **pull_request** event with types `opened`, `synchronize`, and `reopened`, for PRs whose base is the trunk branch from kira config (`git.trunk_branch`). Trunk branch is always from config; no hardcoding.
- The job SHALL **never** run when the PR head branch is the configured trunk branch. The workflow SHALL skip the job when `github.head_ref` equals the trunk branch from kira config.
- The workflow SHALL NOT use `paths-ignore` for the configured work folder so that pushes that only change work items still trigger the job. Use the work folder path from kira config; do not hardcode.
- The job SHALL run in a **dedicated, standalone** workflow file (`.github/workflows/update-pr-details.yml`), not in existing ci.yml or other workflows, so it can easily be added to any project and does not interfere with other CI.

#### FR2: `kira current` command

- Kira SHALL provide a **`kira current`** command that:
  - Derives the work item ID from the **current branch name** (same convention as 010: `{id}-{kebab-case-title}`).
  - Looks for the work item file in the work folder from kira config, **searching all status folders** (e.g. backlog, todo, doing, review, done) until the file with matching `id` in front matter is found.
  - Supports **`--title`**: output the PR title (same format as kira start: work item title prefixed with work item ID).
  - Supports **`--body`**: output the entire content of the work item file (front matter + body).
  - When work item is not found, logs that it is not found and exits with non-zero (or exit 0 per FR5 when used in CI); when branch name does not match the pattern, exit without output.
- The CI job SHALL use `kira current --title` and `kira current --body` to obtain PR title and body, then PATCH the PR via the GitHub API.

#### FR3: PR Content From Work Item

- PR **title** SHALL use the same format as kira start (010): work item title prefixed with work item ID (e.g. `034: ci update pr details`). Derive title from the work item front matter `title` or first `#` heading; do not define a different format in this PRD.
- PR **body** SHALL be set to the **entire content** of the work item file (full file contents: front matter and body).

#### FR4: Auth—automatic token only, no read-only mode

- The job SHALL **not** require users to create, store, or configure any token or secret (no `KIRA_GITHUB_TOKEN`, no PAT, no repo secrets).
- The job SHALL use only the **automatic per-run token** that GitHub Actions provides (the default `GITHUB_TOKEN`). The workflow SHALL grant `contents: read` and `pull-requests: write`. The job SHALL PATCH the PR via the GitHub API.
- Read-only mode (output suggested PR body without calling the API) is **not** supported; the job SHALL always update the PR.

#### FR5: Idempotency and Safety

- Updating the PR SHALL be idempotent: re-running the job with the same work item content SHALL produce the same PR body.
- If no open PR exists for the head branch, the job SHALL exit successfully without error (no-op).
- If the work item file is not found, the job SHALL log that the work item is not found (e.g. "Work item not found, skipping PR update"), then exit 0 without failing the workflow.
- When the same branch has multiple open PRs, the job SHALL update only the PR that triggered the event; do not list or update other PRs for that branch.

### Configuration

- **Trunk branch:** Always from kira config (`git.trunk_branch`). No hardcoding in the workflow; the job/workflow SHALL use the trunk branch from kira.yml.
- **PR title format:** Same as kira start (010)—work item title prefixed with work item ID (e.g. `034: ci update pr details`). No separate format in this PRD.
- **Work folder:** Always from kira config. Do not hardcode paths that already have config; the job SHALL read the work folder path from kira.yml (same as the rest of kira).

### Error Handling

- **Head branch is trunk:** Do not run the job (workflow condition or job step exit 0). Never update PR details when the PR head is the trunk branch.
- **Branch name invalid:** Exit 0, no PR update.
- **Work item not found:** Log that the work item is not found (e.g. "Work item not found, skipping PR update"), then exit 0.
- **No open PR for branch:** Exit 0, no PR update.
- **GitHub API error (e.g. 403, 404):** Fail the job with a clear message so permissions can be fixed; no user-configured token is required.

## Acceptance Criteria

- [ ] CI job runs on `pull_request` types `opened`, `synchronize`, and `reopened` for PRs targeting trunk.
- [ ] CI job never runs when the PR head branch (or the branch being pushed) is the configured trunk branch.
- [ ] Work item ID is correctly derived from branch names of the form `{id}-{kebab-title}`.
- [ ] Job does not fail when branch name does not match (e.g. non-kira branch); it exits successfully without updating any PR.
- [ ] `kira current` derives work item ID from current branch name and finds the work item file by searching all status folders in the configured work folder.
- [ ] `kira current --title` outputs the PR title (same format as kira start); `kira current --body` outputs the entire work item file content.
- [ ] Work item file is found by resolving `id` in the configured work folder (searching all status folders).
- [ ] Job does not fail when work item file is missing; it logs that the work item is not found, then exits successfully without updating any PR.
- [ ] PR title uses the same format as kira start (010): work item ID prefix + work item title (e.g. `034: ci update pr details`).
- [ ] PR body is set to the entire content of the work item file.
- [ ] When multiple open PRs use the same head branch, the job updates only the PR that triggered the event.
- [ ] Job is idempotent: same work item content produces the same PR body.
- [ ] No user-configured token or secret is required; job uses only automatic `GITHUB_TOKEN` with `contents: read` and `pull-requests: write`.
- [ ] GitHub API errors (e.g. insufficient permissions) fail the job with a clear message.
- [ ] Workflow is a dedicated standalone file (`.github/workflows/update-pr-details.yml`); the job is not added to existing ci.yml or other workflows, so it can easily be added to any project and does not interfere with other CI.
- [ ] When the user runs `kira init` and `git_platform` is `github`, the workflow file `.github/workflows/update-pr-details.yml` is created if it does not exist; existing file is not overwritten.

## Implementation Notes

### What needs to happen for it to work every time a branch is made with a work item

1. **Trigger:** Use `pull_request` with types `opened`, `synchronize`, and `reopened`. Filter by base branch using the trunk branch from kira config (`git.trunk_branch`)—e.g. read kira.yml in the job or pass trunk as an input; no hardcoded branch name. Skip when `github.head_ref` equals the trunk branch from config.

2. **Dedicated standalone workflow:** Use a **dedicated, standalone** workflow file `.github/workflows/update-pr-details.yml`. Do not add this job to existing ci.yml or other workflows—standalone so it can easily be added to any project and does not interfere with other CI. Do not use `paths-ignore` for the configured work folder (from kira config) so pushes that only change work items still trigger the job.

3. **Work item on branch:** The job checks out the PR head ref and runs `kira current --title` and `kira current --body`. `kira current` derives the work item from the current branch name and looks in the work folder (all status folders) per kira config. The work item file must be committed on that branch.

4. **Auth:** Use only the automatic `GITHUB_TOKEN`. Grant the job `contents: read` and `pull-requests: write`. No user-configured token. No read-only mode.

5. **Multiple PRs from same branch:** Update only the PR that triggered the event. Do not list or update other PRs for the same branch.

### CI Workflow

- Use a **dedicated, standalone** workflow file `.github/workflows/update-pr-details.yml`. Do not add this job to existing ci.yml or other workflows—standalone so it can easily be added to any project and does not interfere with other CI.
- Trigger: `pull_request` types `opened`, `synchronize`, `reopened`. Base branch filter and job condition: use trunk branch from kira config (`git.trunk_branch`); no hardcoded branch. Skip when `github.head_ref` equals trunk from config.
- No `paths-ignore` for the configured work folder (use kira config; do not hardcode path).

### `kira current` and Work Item Resolution

- The workflow SHALL use the **`kira current`** command to obtain PR title and body:
  - Run `kira current --title` to get the PR title (work item title prefixed with id, same format as kira start).
  - Run `kira current --body` to get the entire work item file content for the PR body.
- **`kira current`** derives the work item ID from the current branch name (same parsing as 010: split on first `-`, validate id), then searches the work folder from kira config **across all status folders** for a markdown file whose front matter contains `id: <workItemID>` (reuse `findWorkItemFile`-style logic but search all configured status folders). Flags `--title` and `--body` output those values to stdout for use in CI.

### GitHub API

- `PATCH /repos/{owner}/{repo}/pulls/{pull_number}` with `title` and `body`.
- Use automatic `GITHUB_TOKEN`; workflow grants `contents: read` and `pull-requests: write`.

### Dependencies

- The workflow depends on the **`kira current`** command (implemented in this PRD): the job runs `kira current --title` and `kira current --body` to get PR title and body. Kira must be installed or available in the CI environment.

### Creating the workflow file on kira init

- When the user runs `kira init` and `git_platform` is `github` (in `kira.yml` or set by init), `kira init` SHALL create `.github/workflows/update-pr-details.yml` if it does not exist. If the file already exists, do not overwrite it.
- Implementation: during `initializeWorkspace`, if config has `git_platform: github`, create `.github/workflows/` if needed and write the workflow YAML from an embedded template; skip if the workflow file already exists.

## Release Notes

### New Features

- **`kira current`:** New command that derives the work item from the current branch name, searches the work folder (all status folders) per kira config, and supports `--title` and `--body` to output the PR title and full work item content. Used by CI to update PR details.
- **CI updates PR details:** When you push to a branch that has an open PR, CI runs `kira current --title` and `kira current --body` and updates the PR. The PR body is set to the **entire content** of the work item file so the PR stays in sync with the work item.

### Improvements

- PR title and body stay aligned with work item content after elaboration or changes.
- Single source of truth: work item file; PR body is a copy of that content.
