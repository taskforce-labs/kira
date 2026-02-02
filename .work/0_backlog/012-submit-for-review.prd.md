---
id: 012
title: submit for review
status: backlog
kind: prd
assigned:
created: 2026-01-19
tags: [github, review, cli]
---

# submit for review

A command that moves work items to review status on the current branch, optionally updates the status on the main trunk branch to reflect the review state (treating trunk as source of truth), creates pull requests on GitHub or changes the status of a PR on GitHub from draft to ready for review.

## Context

In the Kira workflow, developers finish work on a feature branch and need to submit for review: move the work item to review status, ensure the branch is up to date with trunk, push the branch, and create or update a pull request on GitHub. Today this requires multiple manual steps: run `kira latest` to rebase, run `kira move <id> review` to move the work item, push the branch, then create or update the PR in the GitHub UI.

The `kira review` command today only checks slice/task readiness (warns if tasks are open). This PRD extends it into a **submit for review** flow that:

1. Optionally runs the same trunk-update and rebase logic as `kira latest` (update trunk from remote, then rebase current branch onto trunk), unless `--no-trunk-update` or `--no-rebase` is set
2. Moves the work item to review status (from doing to review folder) like `kira move` does
3. Pushes the current branch to the configured remote (e.g. `git push <remote> <current-branch> --force-with-lease`, where `<remote>` is from `git.remote` in kira.yml, default `origin`)
4. If a draft PR exists for the branch, updates it to ready for review using the GitHub API and KIRA_GITHUB_TOKEN (same library as `kira start` uses for draft PRs)
5. If no draft PR exists, creates a new PR (ready for review or draft per `--draft`) using the GitHub API and KIRA_GITHUB_TOKEN

Work item ID is derived from the current branch name (e.g. `012-submit-for-review` → `012`). The command cannot be run on the trunk branch or on non-kira branches.

## Requirements

### Core Functionality

#### Command Interface
- **Command**: `kira review [--reviewer <user>...]`
- **Behavior**: Derives work item ID from current branch name (must be on a kira-created branch; format `{id}-{kebab-title}`). Performs trunk update + rebase (unless skipped), move to review, push, then create/update PR on GitHub.
- **Restrictions**: Cannot be run on trunk branch or non-kira branches
- **Flags**:
  - `--reviewer <user>` - Specify reviewer (can be user number from `kira user` command or email address). Can be used multiple times. Passed to GitHub when creating/updating PR.
  - `--draft` (default: true) - Create or leave PR as draft when creating new PR; when updating an existing draft PR, `--no-draft` means update to ready for review
  - `--no-trunk-update` - Skip updating trunk branch from remote before rebasing (overrides config)
  - `--no-rebase` - Skip rebasing current branch after trunk update (overrides config)
  - `--dry-run` - Show what would be done without executing (validation, planned steps, no git/GitHub changes)

#### Validation and Branch Rules
- Resolve current branch; if it equals configured trunk branch, error with clear message (e.g. "cannot run review on trunk branch; checkout a feature branch first")
- Parse branch name to derive work item ID (e.g. first segment before `-`); validate against config ID format; if branch does not match kira naming or ID invalid, error with clear message
- Resolve work item file by ID (e.g. `findWorkItemFile`); if not found or work item not in doing status, error
- When slice/task checking is configured (e.g. from slices-and-tasks feature), run existing readiness check: warn or error if tasks are open (same behavior as current `kira review` slice check)

#### Trunk Update and Rebase
- When not `--no-trunk-update` and not `--no-rebase`: run same flow as `kira latest` for the current repo(s)—discover repos from work item and config, fetch, update trunk from remote when on trunk, rebase current branch onto trunk when on feature branch; respect stash/pop and conflict handling
- When `--no-trunk-update`: skip updating trunk; if not `--no-rebase`, rebase current branch onto current local trunk only
- When `--no-rebase`: skip rebase entirely
- Support polyrepo: apply trunk update and rebase per repository where the work item applies

#### Move to Review
- Move work item file from doing folder to review folder (same logic as `kira move <id> review`): rename/move file, update status in front matter
- Optionally commit the move when config or flag requests it (reuse move commit templates)

#### Push
- Push current branch to configured remote with `--force-with-lease` (e.g. `git push <remote> <current-branch> --force-with-lease`)
- For polyrepo, push the branch in each repo that has the work item branch

#### GitHub PR Create or Update
- Use KIRA_GITHUB_TOKEN; same client and base URL resolution as `kira start` (internal/git package, ParseGitHubOwnerRepo, NewClient)
- List open PRs for the repository with head = current branch (owner:branch)
- If an open draft PR exists for that head: update it to ready for review when `--no-draft` (or config); optionally set reviewers from `--reviewer`
- If no open PR exists: create a new PR (draft or ready per `--draft`), base = trunk branch, head = current branch, title/body from work item; optionally set reviewers
- On token unset when PR step would run: clear error suggesting KIRA_GITHUB_TOKEN or skip option; do not fail earlier steps (move, push) if only PR step needs token

### Configuration

#### kira.yml Integration
- Use `git.trunk_branch`, `git.remote` (default `origin`), `work` folder, `status_folders` (doing, review) from config
- Optional: `review.trunk_update` / `review.rebase` booleans to default running trunk update and rebase (flags override)
- Optional: `review.commit_move` to commit the move to review (align with move command)
- Workspace/base URL for GitHub from existing workspace config when present

### Error Handling

#### Validation Errors
- On trunk branch: "Cannot run review on trunk branch {name}. Checkout a feature branch first."
- Non-kira branch: "Branch '{name}' does not match kira branch format (expected {id}-{kebab-title}). Checkout a kira feature branch or use kira move for status changes."
- Work item not found: "Work item {id} not found."
- Work item not in doing: "Work item {id} is not in doing status (current: {status}). Only work items in doing can be submitted for review."

#### Git / GitHub Errors
- Push failure: clear message with remote and branch; do not roll back move
- GitHub token missing when required: "KIRA_GITHUB_TOKEN is not set. Set it to create or update PRs, or use --dry-run to skip."
- PR create/update failure: log or print warning; do not fail the command if move and push succeeded (degraded success)

### User Experience
- Clear progress or step messages (validating, rebasing, moving, pushing, creating/updating PR)
- Success message with PR URL when PR was created or updated
- Support for `--dry-run` to show planned steps without executing

## Acceptance Criteria

### Command and Validation
- [ ] `kira review` without args derives work item ID from current branch and fails with clear error when on trunk or non-kira branch
- [ ] When on a kira branch (e.g. `012-submit-for-review`), work item is resolved by ID; command fails if work item not found or not in doing
- [ ] `kira review --dry-run` runs validation and prints planned steps; no file move, no push, no GitHub calls

### Trunk Update and Rebase
- [ ] When neither `--no-trunk-update` nor `--no-rebase`: behavior matches `kira latest` (fetch, update trunk, rebase onto trunk) for the current repo(s)
- [ ] With `--no-rebase`, rebase is skipped; with `--no-trunk-update`, trunk is not updated from remote before rebase
- [ ] Polyrepo: trunk update and rebase run only in repos associated with the work item

### Move and Push
- [ ] Work item file is moved from doing folder to review folder and status field updated to review
- [ ] Current branch is pushed to configured remote with `--force-with-lease`
- [ ] Optional commit of the move when configured; same commit message behavior as `kira move`

### GitHub PR
- [ ] If an open draft PR exists for the current branch, it can be updated to ready for review (e.g. when `--no-draft`); reviewers can be set from `--reviewer`
- [ ] If no open PR exists, a new PR is created (draft or ready per `--draft`), with title/body from work item and optional reviewers
- [ ] KIRA_GITHUB_TOKEN required for PR create/update; clear error when unset and PR step would run
- [ ] PR create/update uses same GitHub client and base URL logic as `kira start` draft PR flow

### Slice Readiness (existing behavior)
- [ ] When slice/task checking is enabled, open tasks cause warning or error before proceeding (existing `kira review` slice check preserved or integrated)

## Slices

Slices are ordered; each slice is a committable unit of work. Tasks within a slice are implemented in order.

### Slice 1: Command, validation, and branch derivation

- [ ] T001: Add `kira review` Cobra command (or repurpose existing) with flags: `--reviewer`, `--draft`, `--no-trunk-update`, `--no-rebase`, `--dry-run`; long description updated for submit-for-review flow.
- [ ] T002: Implement branch derivation: get current branch, compare to trunk (error if on trunk); parse branch name to work item ID (e.g. `{id}-{kebab}`), validate ID format via config; error if not kira branch.
- [ ] T003: Resolve work item by ID (`findWorkItemFile`); validate work item is in doing status (derive current status from file location or front matter); error if not found or not in doing.
- [ ] T004: Integrate existing slice/task readiness check when configured (warn or error if tasks open); keep behavior consistent with current `kira review` slice check.
- [ ] T005: Implement `--dry-run`: run validation and print planned steps (trunk update, rebase, move, push, PR create/update); no side effects.

### Slice 2: Trunk update and rebase

- [ ] T006: Reuse or call `kira latest` discovery and repo resolution for the work item (repos from config/workspace and work item metadata); run in context of current branch and work item.
- [ ] T007: When not `--no-trunk-update` and not `--no-rebase`: execute fetch, trunk update (when on trunk), rebase onto trunk (when on feature branch); reuse stash/pop and conflict handling from latest.
- [ ] T008: When `--no-trunk-update`: skip trunk update; when `--no-rebase`: skip rebase; apply per-repo for polyrepo.
- [ ] T009: Add optional config for default trunk_update/rebase (e.g. `review.trunk_update`, `review.rebase`); flags override config.

### Slice 3: Move to review and push

- [ ] T010: Move work item file from doing folder to review folder (reuse move logic: rename, update status in front matter); optional commit of move when config/flag set.
- [ ] T011: Push current branch to configured remote with `--force-with-lease`; support polyrepo (push in each repo for the work item).
- [ ] T012: Progress/success messages for move and push; error handling without rolling back move on push failure.

### Slice 4: GitHub PR create or update

- [ ] T013: In internal/git: add ListPullRequestsByHead(ctx, client, owner, repo, head string) (or use existing List with Head filter); add UpdateDraftToReady(ctx, client, owner, repo, prNumber) and optionally SetReviewers; add CreatePR(ctx, client, owner, repo, base, head, title, body, draft bool) for non-draft or draft PR.
- [ ] T014: After push: get remote URL and base URL; parse owner/repo; create GitHub client from KIRA_GITHUB_TOKEN; list open PRs for head = current branch; if draft PR found and `--no-draft`, update to ready and optionally set reviewers; if no PR, create PR (draft or ready per `--draft`) with title/body from work item and optional reviewers.
- [ ] T015: When KIRA_GITHUB_TOKEN is unset and PR step would run: return clear error; do not fail move/push if they already succeeded (degraded success or fail before push—decide and document). Prefer failing before push with clear message when token required.
- [ ] T016: Use same base URL and client construction as start (workspace GitBaseURL when polyrepo); handle GitHub Enterprise.

### Slice 5: Tests and docs

- [ ] T017: Unit tests for branch derivation, validation (trunk, non-kira, work item not found, not in doing), and dry-run output.
- [ ] T018: Integration or e2e tests: run review on a kira branch (with/without --no-rebase, --no-trunk-update), verify move and push; mock or real GitHub for PR create/update where applicable.
- [ ] T019: Update README or user docs for `kira review` submit-for-review flow; document flags and config.

## Implementation Notes

- **Branch ID derivation**: Same convention as start (branch name `{id}-{kebab-title}`); extract ID as segment before first `-`; validate with `cfg.Validation.IDFormat`.
- **Reuse**: Prefer calling existing `latest` flow (discoverRepositories, performFetchAndRebase) parameterized by work item and current branch; reuse moveWorkItem for move to review; reuse git.ParseGitHubOwnerRepo, NewClient, and extend internal/git with List PRs by head, Update draft to ready, Create PR with draft flag and reviewers.
- **GitHub**: List open PRs with `Head: owner:branch`; update draft via PullRequests.Edit with `Draft: false`; create with Draft: true/false and RequestedReviewers if supported.

## Release Notes

- **`kira review` submit for review**: From a kira feature branch, `kira review` now performs trunk update and rebase (optional), moves the work item to review status, pushes the branch, and creates or updates a GitHub PR. Use `--no-trunk-update` or `--no-rebase` to skip update/rebase, and `--dry-run` to preview steps.

