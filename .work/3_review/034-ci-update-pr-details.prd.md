---
id: 034
title: ci update pr details
status: review
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
- **Polyrepo support:** In polyrepo workspaces, only the main repo (where `kira.yml` is located) runs the CI workflow. The workflow uses `kira current prs` to discover all related PRs across all repos in the workspace (using the same logic as `kira start` to resolve projects and their GitHub remotes and if the repo requires a PR to be created), then updates all discovered PRs via the GitHub API. This ensures all PRs stay in sync with the work item, which exists only in the main repo.

This keeps the PR and the work item aligned whenever the branch is pushed, including in multi-repo environments.

### Scope

- **In scope:** GitHub only; a **`kira current`** command that derives work item from current branch and outputs `--title` and `--body` per work item (searching work folders per kira config); **`kira current prs`** command that discovers all related PRs in polyrepo workspaces (reuses `kira start` logic to resolve projects and GitHub remotes); CI job that runs on `pull_request` (opened, synchronize, reopened) and uses `kira current --title`, `kira current --body`, and `kira current prs` to update PR title and body across all repos (PR body = entire work item content); **no user-configured tokens** (automatic `GITHUB_TOKEN` only); a **dedicated, standalone** workflow file (`.github/workflows/update-pr-details.yml`); creating that workflow file on `kira init` when the user indicates GitHub; permissions documentation and meaningful error messages for cross-repo access.
- **Out of scope:** GitLab CI (separate work item); editing PR labels or reviewers from work item; triggering on `push` and discovering PRs by API (we use `pull_request` only); requiring `KIRA_GITHUB_TOKEN`, PAT, or repo secrets; read-only mode (output suggested body without updating the PR)—we support automatic `GITHUB_TOKEN` only and the job SHALL update the PR via API; adding this job to existing ci.yml or other workflows (we use a dedicated standalone file so it does not interfere with other CI).

## Requirements

### Functional Requirements

#### FR1: CI Trigger

- The job SHALL run on the **pull_request** event with types `opened`, `synchronize`, and `reopened`, for PRs whose base is the trunk branch from kira config (`git.trunk_branch`). Trunk branch is always from config; no hardcoding.
- The workflow SHALL filter the `pull_request` trigger by base branch using the trunk branch from kira config (e.g., `branches: [master]` or dynamically read from config). This ensures the job only runs for PRs targeting trunk.
- The job SHALL **never** run when the PR head branch is the configured trunk branch. The workflow SHALL skip the job when `github.head_ref` equals the trunk branch from kira config.
- The workflow SHALL NOT use `paths-ignore` for the configured work folder so that pushes that only change work items still trigger the job. Use the work folder path from kira config; do not hardcode. The workflow may use `paths-ignore` for other paths (e.g., documentation, unrelated files) but SHALL NOT ignore the configured work folder path.
- The job SHALL run in a **dedicated, standalone** workflow file (`.github/workflows/update-pr-details.yml`), not in existing ci.yml or other workflows, so it can easily be added to any project and does not interfere with other CI.

#### FR2: `kira current` command

- Kira SHALL provide a **`kira current`** command that:
  - Derives the work item ID from the **current branch name** (same convention as 010: `{id}-{kebab-case-title}`).
  - Looks for the work item file in the work folder from kira config, **searching all status folders** (e.g. backlog, todo, doing, review, done) until the file with matching `id` in front matter is found. The work item file SHALL be found in the checked-out branch (PR head ref) where the workflow runs. In polyrepo workspaces, the work item exists only in the main repo, so the workflow checks out the PR head ref from the main repo.
  - Supports **`--title`**: output the PR title (same format as kira start: work item title prefixed with work item ID).
  - Supports **`--body`**: output the entire content of the work item file (front matter + body).
  - Supports **`prs`** subcommand: discovers all related PRs across all repos (including main repo). Outputs JSON array of PR information: `[{"owner": "org", "repo": "repo-name", "pr_number": 123, "branch": "034-ci-update-pr-details"}, ...]`. Always includes the main repo (where `kira.yml` is located) if it's a GitHub remote. In polyrepo workspaces, also includes all projects from `workspace.projects` that are GitHub remotes. Uses the same logic as `kira start` to resolve the main repo remote and `workspace.projects` with their GitHub remotes. Only includes repos that are GitHub remotes. When not in polyrepo workspace, outputs array with only the main repo PR (if found). When work item is not found or branch name invalid, outputs empty array and exits 0.
  - **Error behavior for `--title` and `--body` flags:** When work item is not found, logs that it is not found and exits with non-zero exit code. When branch name does not match the pattern (invalid branch name), exits without output (non-zero exit code). The CI job SHALL catch these non-zero exits and handle them gracefully per FR5 and Error Handling sections.
  - **Error behavior for `prs` subcommand:** When work item is not found or branch name is invalid, outputs empty array `[]` and exits 0. The CI job SHALL handle empty arrays gracefully (no PRs to update).
- The CI job SHALL use `kira current --title` and `kira current --body` to obtain PR title and body.
- The CI job SHALL use `kira current prs` to discover all related PRs (including main repo), then update each discovered PR via the GitHub API. This works for both standalone/monorepo (returns main repo PR) and polyrepo (returns main repo + all project repos).

#### FR3: PR Content From Work Item

- PR **title** SHALL use the same format as kira start (010): work item title prefixed with work item ID (e.g. `034: ci update pr details`). Derive title from the work item front matter `title` or first `#` heading; do not define a different format in this PRD.
- PR **body** SHALL be set to the **entire content** of the work item file (full file contents: front matter and body).

#### FR4: Auth—automatic token only, no read-only mode

- The job SHALL **not** require users to create, store, or configure any token or secret (no `KIRA_GITHUB_TOKEN`, no PAT, no repo secrets).
- The job SHALL use only the **automatic per-run token** that GitHub Actions provides (the default `GITHUB_TOKEN`). The workflow SHALL grant `contents: read` and `pull-requests: write`. The job SHALL PATCH the PR via the GitHub API.
- **Polyrepo permissions:** For polyrepo workspaces, the `GITHUB_TOKEN` in the main repo may not have write access to other repos. The workflow SHALL attempt to update all discovered PRs, but SHALL gracefully handle permission errors (403 Forbidden) by logging a clear error message indicating which repos could not be updated and why, then continue updating other repos. The workflow SHALL fail only if the main repo's PR cannot be updated (since that's where the workflow runs).
- Read-only mode (output suggested PR body without calling the API) is **not** supported; the job SHALL always update the PR.

#### FR5: Idempotency and Safety

- Updating the PR SHALL be idempotent: re-running the job with the same work item content SHALL produce the same PR body.
- If no open PR exists for the head branch, the job SHALL exit successfully without error (no-op).
- **Command failures:** If `kira current --title` or `kira current --body` exits with non-zero (work item not found or invalid branch name), the job SHALL catch the error, log an appropriate message (e.g. "Work item not found, skipping PR update" or "Branch name does not match kira format, skipping PR update"), then exit 0 without failing the workflow. The job SHALL NOT fail when `kira current` commands fail—these are expected scenarios for non-kira branches or missing work items.
- If `kira current prs` returns an empty array (work item not found or invalid branch), the job SHALL exit 0 without updating any PR.
- When the same branch has multiple open PRs, the job SHALL update only the PR that triggered the event; do not list or update other PRs for that branch.
- **GitHub API rate limiting:** When updating multiple PRs in polyrepo scenarios, the workflow SHALL handle GitHub API rate limits gracefully. If rate limited (HTTP 429), the workflow SHALL wait and retry with exponential backoff, or log a clear warning and exit 0 (non-fatal). The workflow SHALL prioritize updating the main repo PR first, then other repos.
- **PR body size limits:** GitHub PR bodies have a size limit (~65KB). If the work item file exceeds this limit, the workflow SHALL truncate the body with a clear indicator (e.g., "... (truncated due to size limit)") and log a warning, or fail with a clear error message instructing the user to reduce the work item size.

### Configuration

- **Trunk branch:** Always from kira config (`git.trunk_branch`). No hardcoding in the workflow; the job/workflow SHALL use the trunk branch from kira.yml.
- **PR title format:** Same as kira start (010)—work item title prefixed with work item ID (e.g. `034: ci update pr details`). No separate format in this PRD.
- **Work folder:** Always from kira config. Do not hardcode paths that already have config; the job SHALL read the work folder path from kira.yml (same as the rest of kira).

### Error Handling

- **Head branch is trunk:** Do not run the job (workflow condition or job step exit 0). Never update PR details when the PR head is the trunk branch.
- **Branch name invalid:** If `kira current --title` or `kira current --body` fails due to invalid branch name (non-zero exit), the job SHALL log that the branch name is invalid (e.g. "Branch name does not match kira format, skipping PR update"), then exit 0 without updating any PR.
- **Work item not found:** If `kira current --title` or `kira current --body` fails because the work item file is not found (non-zero exit), the job SHALL log that the work item is not found (e.g. "Work item not found, skipping PR update"), then exit 0 without updating any PR. If `kira current prs` returns an empty array (work item not found or invalid branch), the job SHALL exit 0 without updating any PR.
- **No open PR for branch:** Exit 0, no PR update.
- **GitHub API error for main repo PR (e.g. 403, 404):** Fail the job with a clear message so permissions can be fixed. Include instructions: "Failed to update PR in main repo. Ensure the workflow has `pull-requests: write` permission. See [permissions documentation](#permissions) for details."
- **GitHub API error for other repo PRs (polyrepo, 403 Forbidden):** Log a clear warning message indicating which repo could not be updated and why: "Warning: Could not update PR in {owner}/{repo}: GITHUB_TOKEN does not have write access to this repository. To enable cross-repo PR updates, ensure the workflow runs with a token that has access to all repos, or configure repository permissions. Skipping this repo." Continue updating other repos. Do not fail the workflow.
- **GitHub API error for other repo PRs (404 Not Found):** Log a clear message: "Info: No open PR found for branch {branch} in {owner}/{repo}. This is expected if the PR hasn't been created yet or was closed. Skipping this repo." Continue updating other repos. Do not fail the workflow.
- **Invalid JSON from `kira current prs`:** Log error and fail the workflow with clear message: "Failed to parse PR list from `kira current prs`. Output: {output}"
- **GitHub API rate limiting (429 Too Many Requests):** Log a warning message indicating rate limit was hit, wait with exponential backoff, and retry. If retries are exhausted, log a clear warning and exit 0 (non-fatal) for other repo PRs, but fail if main repo PR cannot be updated due to rate limits.
- **PR body exceeds size limit:** If the work item file content exceeds GitHub's PR body size limit (~65KB), log an error and fail the workflow with a clear message: "Work item file exceeds GitHub PR body size limit (~65KB). Please reduce the work item size or split it into multiple work items."

### Permissions

For **standalone/monorepo** workspaces:
- The workflow requires `contents: read` and `pull-requests: write` permissions on the repository where it runs.
- The automatic `GITHUB_TOKEN` has these permissions by default for the repository where the workflow runs.

For **polyrepo** workspaces:
- The workflow runs only in the **main repo** (where `kira.yml` is located).
- The workflow requires `contents: read` and `pull-requests: write` permissions on the main repo (automatic `GITHUB_TOKEN` provides this).
- To update PRs in **other repos**, the `GITHUB_TOKEN` must have write access to those repositories. By default, `GITHUB_TOKEN` only has access to the repository where the workflow runs.
- **Options for cross-repo access:**
  1. **Organization-level permissions (recommended):** If all repos are in the same GitHub organization, configure the organization to allow workflows to access other repositories. See [GitHub documentation on workflow permissions](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#permissions).
  2. **Repository secrets:** If cross-repo access is not available, the workflow will gracefully skip updating PRs in repos it cannot access, logging clear warning messages. Only the main repo's PR will be updated.
  3. **Fine-grained personal access token:** As a future enhancement, users could configure a PAT with access to all repos, but this is out of scope for this PRD (automatic `GITHUB_TOKEN` only).

**Error messages:** When the workflow cannot update a PR in another repo due to permissions, it logs a clear warning message indicating which repo was skipped and why, then continues updating other repos. The workflow only fails if it cannot update the main repo's PR.

## Acceptance Criteria

- [ ] CI job runs on `pull_request` types `opened`, `synchronize`, and `reopened` for PRs targeting trunk.
- [ ] CI job only runs for PRs whose base branch is the trunk branch from kira config (workflow filters by base branch).
- [ ] CI job never runs when the PR head branch (or the branch being pushed) is the configured trunk branch.
- [ ] Work item ID is correctly derived from branch names of the form `{id}-{kebab-title}`.
- [ ] Job does not fail when branch name does not match (e.g. non-kira branch); it exits successfully without updating any PR.
- [ ] `kira current` derives work item ID from current branch name and finds the work item file by searching all status folders in the configured work folder.
- [ ] `kira current --title` outputs the PR title (same format as kira start); `kira current --body` outputs the entire work item file content.
- [ ] `kira current prs` discovers all related PRs (main repo + project repos in polyrepo), outputs JSON array with owner, repo, pr_number, and branch for each PR.
- [ ] `kira current prs` always includes the main repo (where `kira.yml` is located) if it's a GitHub remote.
- [ ] `kira current prs` in polyrepo workspaces also includes all projects from `workspace.projects` that are GitHub remotes, using the same logic as `kira start` to resolve projects.
- [ ] `kira current prs` only includes GitHub remotes (skips non-GitHub repos gracefully).
- [ ] `kira current prs` in standalone/monorepo returns array with only the main repo PR (if found).
- [ ] Work item file is found by resolving `id` in the configured work folder (searching all status folders).
- [ ] Job does not fail when work item file is missing; it logs that the work item is not found, then exits successfully without updating any PR.
- [ ] PR title uses the same format as kira start (010): work item ID prefix + work item title (e.g. `034: ci update pr details`).
- [ ] PR body is set to the entire content of the work item file.
- [ ] In all workspaces: the job uses `kira current prs` to discover all related PRs and updates each discovered PR. In standalone/monorepo, this is just the main repo PR. In polyrepo, this includes main repo + all project repos.
- [ ] Job is idempotent: same work item content produces the same PR body.
- [ ] No user-configured token or secret is required; job uses only automatic `GITHUB_TOKEN` with `contents: read` and `pull-requests: write`.
- [ ] GitHub API errors for main repo PR (e.g. insufficient permissions) fail the job with a clear message including permissions documentation link.
- [ ] GitHub API errors for other repo PRs (403 Forbidden) log clear warning messages indicating which repo was skipped and why, then continue updating other repos without failing the workflow.
- [ ] GitHub API errors for other repo PRs (404 Not Found) log clear info messages and continue without failing the workflow.
- [ ] Workflow is a dedicated standalone file (`.github/workflows/update-pr-details.yml`); the job is not added to existing ci.yml or other workflows, so it can easily be added to any project and does not interfere with other CI.
- [ ] When the user runs `kira init` and `git_platform` is `github`, the workflow file `.github/workflows/update-pr-details.yml` is created if it does not exist; existing file is not overwritten.
- [ ] Workflow handles GitHub API rate limiting (429) gracefully with retries and exponential backoff, prioritizing main repo PR updates.
- [ ] Workflow detects and handles PR body size limits, failing with a clear error message if work item exceeds GitHub's limit (~65KB).
- [ ] Work item file is found in the checked-out PR head branch (main repo in polyrepo scenarios).

## Slices

### Slice 1: `kira current` command
Commit: Implement `kira current` command that derives work item from branch name and outputs PR title and body. Supports --title, --body flags and prs subcommand for polyrepo discovery. Searches all status folders for work item file.
- [x] T001: Add `kira current` command structure with cobra command and flags (--title, --body) and prs subcommand
- [x] T002: Implement branch name parsing to derive work item ID (reuse parseWorkItemIDFromBranch logic)
- [x] T003: Implement work item file search across all status folders (modify findWorkItemFile or create new function that searches all status folders)
- [x] T004: Implement --title flag: output PR title format (work item ID + title, same as kira start)
- [x] T005: Implement --body flag: output entire work item file content (front matter + body)
- [x] T006: Implement `prs` subcommand: always include main repo (get remote from git.remote config or "origin", query GitHub API for PRs), detect polyrepo workspace, resolve workspace.projects using same logic as kira start (resolvePolyrepoProjects), get GitHub remotes for each project, query GitHub API to find open PRs with matching branch name, combine main repo + project repos into single JSON array
- [x] T007: Handle error cases: for `--title`/`--body` flags: invalid branch name (exit without output, non-zero), work item not found (log and exit non-zero); for `prs` subcommand: invalid branch name or work item not found (output empty array, exit 0); polyrepo detection failures (output empty array, exit 0)

### Slice 2: CI workflow file
Commit: Create dedicated GitHub Actions workflow file that triggers on pull_request events, uses kira current to get PR title/body and discover related PRs in polyrepo, and updates PRs via GitHub API. Includes proper triggers, permissions, and error handling.
- [x] T008: Create workflow YAML template structure for `.github/workflows/update-pr-details.yml`
- [x] T009: Configure pull_request trigger with types: opened, synchronize, reopened, and filter by base branch using trunk branch from kira config
- [x] T010: Read trunk branch from kira.yml using YAML parser (yq) or shell parsing with proper error handling (handle quoted values, comments, etc.)
- [x] T011: Add job condition to skip when github.head_ref equals trunk branch from config, and verify base branch matches trunk
- [x] T012: Configure permissions: contents: read, pull-requests: write (automatic GITHUB_TOKEN)
- [x] T013: Add steps to build kira from source: (1) Set up Go (using `actions/setup-go@v5`), (2) Cache Go modules (using `actions/cache@v4` like existing CI workflow), (3) Build kira using `make build` (reuse existing Makefile target) since `kira current` command hasn't been released yet. Updating to download released binary will be handled in a separate work item.
- [x] T014: Implement job steps: checkout PR head ref (main repo in polyrepo), run `kira current --title` and `kira current --body`, run `kira current prs` to get all PRs (main repo + project repos), parse JSON output with validation, for each discovered PR update via GitHub API PATCH with proper error handling (403 = warning, 404 = info, 429 = retry with backoff, continue; fail only on main repo errors), validate PR body size before updating
- [x] T016: Add error handling: catch non-zero exits from `kira current --title`/`--body` (work item not found or invalid branch) and handle gracefully (log and exit 0), handle empty array from `kira current prs` (exit 0), validate JSON structure from `kira current prs` before parsing, no PR exists (exit 0), GitHub API errors for main repo (fail with clear message and permissions link), GitHub API errors for other repos (log warning/info and continue), handle rate limiting (429) with exponential backoff retries, validate PR body size limit (~65KB) and fail with clear error if exceeded
- [x] T017: Test workflow locally using `./kdev current` (or build from source) to verify workflow logic before release
- [x] T018: Test workflow end-to-end in CI (requires kira release with `kira current` command from Slice 1, or temporarily build from source in workflow)
- [x] T019: Test polyrepo workflow: verify it discovers and updates PRs in multiple repos, handles permission errors gracefully

### Slice 3: kira init integration
Commit: Integrate workflow file creation into kira init command. When git_platform is github, create .github/workflows/update-pr-details.yml from template if it doesn't exist.
- [x] T020: Detect git_platform: github in kira config during initializeWorkspace
- [x] T021: Create `.github/workflows/` directory if it doesn't exist
- [x] T022: Create `update-pr-details.yml` from embedded template if file doesn't exist
- [x] T023: Skip creation if workflow file already exists (do not overwrite)

## Implementation Notes

### What needs to happen for it to work every time a branch is made with a work item

1. **Trigger:** Use `pull_request` with types `opened`, `synchronize`, and `reopened`. Filter by base branch using the trunk branch from kira config (`git.trunk_branch`) in the workflow trigger (e.g., `branches: [master]` or dynamically read from config). Additionally, read kira.yml in the job to verify base branch matches trunk (defense in depth). Skip when `github.head_ref` equals the trunk branch from config.

2. **Dedicated standalone workflow:** Use a **dedicated, standalone** workflow file `.github/workflows/update-pr-details.yml`. Do not add this job to existing ci.yml or other workflows—standalone so it can easily be added to any project and does not interfere with other CI. Do not use `paths-ignore` for the configured work folder (from kira config) so pushes that only change work items still trigger the job.

3. **Work item on branch:** The job checks out the PR head ref and runs `kira current --title` and `kira current --body`. `kira current` derives the work item from the current branch name and looks in the work folder (all status folders) per kira config. The work item file must be committed on that branch.

4. **Auth:** Use only the automatic `GITHUB_TOKEN`. Grant the job `contents: read` and `pull-requests: write`. No user-configured token. No read-only mode.

5. **All workspaces:** Use `kira current prs` to discover all related PRs (main repo + project repos in polyrepo), then update each discovered PR via GitHub API. In standalone/monorepo, `kira current prs` returns only the main repo PR. In polyrepo, it returns main repo + all project repos. Handle permission errors gracefully: log warnings for repos that cannot be updated, continue updating other repos, fail only if main repo PR cannot be updated (since that's where the workflow runs and where the work item exists).

### CI Workflow

- Use a **dedicated, standalone** workflow file `.github/workflows/update-pr-details.yml`. Do not add this job to existing ci.yml or other workflows—standalone so it can easily be added to any project and does not interfere with other CI.
- Trigger: `pull_request` types `opened`, `synchronize`, `reopened`. **Base branch filter:** The workflow SHALL filter by base branch using the trunk branch from kira config (e.g., `branches: [master]` in the trigger, or dynamically read from config in a job step and skip if base doesn't match). Job condition: skip when `github.head_ref` equals trunk from config.
- No `paths-ignore` for the configured work folder (use kira config; do not hardcode path).
- **Read trunk branch from config:** The workflow SHALL read the trunk branch from `kira.yml` using a YAML parser (e.g., `yq` or a small Go script) or shell parsing with proper error handling. Shell parsing is acceptable but should handle edge cases (quoted values, comments, etc.). As a future enhancement, `kira config get trunk_branch` could be used once available.
- **Install kira:** The workflow SHALL build kira from source before running `kira current` (since `kira current` hasn't been released yet). Reuse existing Makefile and CI patterns: (1) Set up Go using `actions/setup-go@v5`, (2) Cache Go modules using `actions/cache@v4` (same as existing CI workflow), (3) Build using `make build` (reuse existing Makefile target). This matches the existing CI workflow pattern but only builds instead of running checks. Updating to download released binary will be handled in a separate work item.
- **Work item file location:** The workflow checks out the PR head ref and looks for the work item file in that branch. In polyrepo workspaces, the work item exists only in the main repo, so the workflow checks out the main repo's PR head ref.

### `kira current` and Work Item Resolution

- The workflow SHALL use the **`kira current`** command to obtain PR title and body:
  - Run `kira current --title` to get the PR title (work item title prefixed with id, same format as kira start).
  - Run `kira current --body` to get the entire work item file content for the PR body.
- **`kira current`** derives the work item ID from the current branch name (same parsing as 010: split on first `-`, validate id), then searches the work folder from kira config **across all status folders** for a markdown file whose front matter contains `id: <workItemID>` (reuse `findWorkItemFile`-style logic but search all configured status folders). Flags `--title` and `--body` output those values to stdout for use in CI.
- **Work item file location:** The work item file SHALL be found in the checked-out branch (PR head ref). In polyrepo workspaces, the work item exists only in the main repo, so the workflow checks out the main repo's PR head ref where the work item file is located.
- **File encoding:** Work item files SHALL be UTF-8 encoded. The workflow SHALL handle UTF-8 content correctly when reading and updating PR bodies.

### `kira current prs` and PR Discovery

- The workflow SHALL use **`kira current prs`** to discover all related PRs (works for both standalone/monorepo and polyrepo):
  - Run `kira current prs` to get JSON array of PR information: `[{"owner": "org", "repo": "repo-name", "pr_number": 123, "branch": "034-ci-update-pr-details"}, ...]`
  - **`kira current prs`** always includes the main repo (where `kira.yml` is located):
    - Gets the main repo's GitHub remote URL (from `git.remote` config or "origin")
    - Queries GitHub API: `GET /repos/{owner}/{repo}/pulls?head={owner}:{branch}&state=open`
    - Includes main repo PR in the returned array if found
  - **In polyrepo workspaces**, also includes all projects from `workspace.projects`:
    - Uses the same logic as `kira start` to resolve projects: calls `resolvePolyrepoProjects` to get all projects with their paths and remotes
    - For each project, gets the GitHub remote URL (skips non-GitHub remotes)
    - For each GitHub remote, queries GitHub API: `GET /repos/{owner}/{repo}/pulls?head={owner}:{branch}&state=open`
    - Includes each found project repo PR in the returned array
  - **In standalone/monorepo**, returns array with only the main repo PR (if found)
  - The workflow then updates each discovered PR via `PATCH /repos/{owner}/{repo}/pulls/{pr_number}` with the work item title and body
  - **JSON parsing:** The workflow SHALL validate the JSON structure from `kira current prs` before parsing. Use `jq` with proper error handling, or validate JSON structure (array, required fields: owner, repo, pr_number, branch) before processing. Handle special characters in PR titles/bodies correctly when constructing JSON payloads for GitHub API.
  - If a repo cannot be accessed (403 Forbidden), log a clear warning and continue with other repos
  - If no PR is found for a repo (404 Not Found), log an info message and continue
  - Only fail the workflow if the main repo's PR cannot be updated (since that's where the workflow runs and where the work item exists)

### GitHub API

- **All PRs:** For each PR discovered via `kira current prs` (includes main repo + all project repos in polyrepo), `PATCH /repos/{owner}/{repo}/pulls/{pr_number}` with `title` and `body`.
- Use automatic `GITHUB_TOKEN`; workflow grants `contents: read` and `pull-requests: write`.
- **Cross-repo access:** The `GITHUB_TOKEN` may not have write access to other repos (in polyrepo). Handle 403 Forbidden errors gracefully by logging clear warnings and continuing with other repos. Only fail if main repo PR cannot be updated (since that's where the workflow runs and where the work item exists).
- **Rate limiting:** Handle GitHub API rate limits (HTTP 429) with exponential backoff retries. Prioritize updating the main repo PR first, then other repos. If rate limits persist after retries, log warnings for other repos (non-fatal) but fail if main repo PR cannot be updated.
- **PR body size:** Validate that work item content does not exceed GitHub's PR body size limit (~65KB). If exceeded, fail with a clear error message instructing the user to reduce the work item size.

### Dependencies

- The workflow depends on the **`kira current`** command (implemented in this PRD): the job runs `kira current --title` and `kira current --body` to get PR title and body. Kira must be installed or available in the CI environment.
- **Testing strategy:**
  - **Slice 1** can be tested independently using `./kdev current` locally (unit tests, integration tests, e2e tests)
  - **Slice 2, T015:** Test workflow logic locally using `./kdev current` to verify commands work before CI testing
  - **Slice 2, T016:** For end-to-end CI testing, build kira from source in the workflow (since `kira current` hasn't been released yet). This work item focuses on making the repo update PRs; updating to use released binary will be a separate work item.

### Release Strategy

**This work item scope:** This PRD focuses solely on making the repo update PR details. The workflow will build kira from source (using `make build` or `go build`) since the `kira current` command hasn't been released yet. This is similar to using `./kdev current` locally - both run from source code.

**Future work item:** Updating the workflow to download a stable released version of kira (instead of building from source) will be handled in a separate work item. That future work item will:
- Update the workflow template to download kira binary from GitHub releases
- Add logic to select/download stable versions of kira
- Update the workflow created by `kira init` to use the released binary approach

**This work item release:**
- Release kira with `kira current` command (Slice 1)
- Release workflow template that builds kira from source (Slice 2)
- Release `kira init` integration that creates the workflow file (Slice 3)

### Creating the workflow file on kira init

- When the user runs `kira init` and `git_platform` is `github` (in `kira.yml` or set by init), `kira init` SHALL create `.github/workflows/update-pr-details.yml` if it does not exist. If the file already exists, do not overwrite it.
- Implementation: during `initializeWorkspace`, if config has `git_platform: github`, create `.github/workflows/` if needed and write the workflow YAML from an embedded template; skip if the workflow file already exists.
- **Note:** The workflow file may already exist in the repository (e.g., manually created or from a previous implementation). The `kira init` integration SHALL check for existing files and skip creation if present, avoiding overwrites.

## Release Notes

### New Features

- **`kira current`:** New command that derives the work item from the current branch name, searches the work folder (all status folders) per kira config, and supports `--title` and `--body` to output the PR title and full work item content. Used by CI to update PR details.
- **`kira current prs`:** New subcommand that discovers all related PRs in polyrepo workspaces. Uses the same logic as `kira start` to resolve projects and their GitHub remotes, then queries GitHub API to find open PRs with matching branch names. Outputs JSON array for use by CI workflows.
- **CI updates PR details:** When you push to a branch that has an open PR, CI runs `kira current --title` and `kira current --body` and updates the PR. The PR body is set to the **entire content** of the work item file so the PR stays in sync with the work item.
- **Polyrepo support:** In polyrepo workspaces, the main repo's CI workflow automatically discovers and updates PRs in all related repos. The workflow handles permission errors gracefully, logging clear warnings for repos it cannot access while continuing to update others.

### Improvements

- PR title and body stay aligned with work item content after elaboration or changes.
- Single source of truth: work item file; PR body is a copy of that content.
