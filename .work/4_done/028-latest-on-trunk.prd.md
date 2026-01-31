---
id: 028
title: latest-on-trunk
status: done
kind: prd
assigned:
estimate: 0
created: 2026-01-29
due: 2026-01-29
tags: []
---

# latest-on-trunk

Support `kira latest` when the current branch is trunk: update local trunk from remote (fetch + pull/rebase), with the same stash-before / unstash-after behavior as on feature branches.

## Overview

Today `kira latest` only works on a feature branch. It fetches from remote, rebases the current branch onto trunk, and optionally stashes uncommitted changes before rebase and pops them after. When the user is **on trunk**, the command errors with "already on trunk branch ..., cannot rebase onto itself". This PRD extends `kira latest` so that when the current branch is trunk, it performs an "update trunk from remote" flow: fetch, then update local trunk (e.g. pull --rebase), with the same stash/unstash behavior. Each repository is handled according to its own current branch (trunk vs feature), so in a polyrepo setup some repos can be on trunk and others on feature branches.

## Context

### Problem Statement

Developers and agents sometimes run `kira latest` while on trunk—for example to sync trunk with remote before creating a branch, or to pull the latest changes while a work item is being refined on trunk. Currently the command fails with "already on trunk branch ..., cannot rebase onto itself", forcing manual `git fetch` / `git pull --rebase` and manual stash handling. That breaks the single-command workflow and is inconsistent with the feature-branch experience.

### Current State

- `kira latest` requires a work item in the doing folder and discovers repositories from config/workspace.
- For each repo it: checks state (conflicts, dirty, in-rebase, etc.), then fetch + rebase onto `remote/trunk`.
- `rebaseOntoTrunk` explicitly returns an error when `currentBranch == repo.TrunkBranch`.
- Stash/pop and conflict handling are implemented for the rebase path only.

### Proposed Solution

- When the current branch **equals** the resolved trunk branch for that repo, treat the repo as "on trunk".
- For repos on trunk: after fetch, update local trunk from remote (e.g. `git pull --rebase <remote> <trunk>`) instead of rebasing onto trunk.
- Reuse the same stash-before / pop-after behavior for the trunk-update path; respect `--no-pop-stash` and existing conflict/abort semantics where applicable.
- For repos on a feature branch, behavior remains unchanged (rebase onto trunk).
- Polyrepo: each repo is processed according to its own current branch; mixed trunk + feature-branch repos in one run are supported.

### Design Philosophy

- One command: `kira latest` works on both trunk and feature branches; no new subcommand.
- Same safety: stash uncommitted changes before update, pop after (unless `--no-pop-stash`).
- Branch-aware: operation per repo depends only on current branch (trunk vs not trunk).

## Requirements

### Functional Requirements

1. **Branch detection**
   - For each repository, resolve the trunk branch (existing logic: project > git config > auto-detect) and obtain the current branch.
   - If current branch equals trunk branch, treat that repo as "on trunk" for this run.

2. **On-trunk update**
   - For repos on trunk, after fetch perform "update trunk from remote":
     - Prefer rebase semantics: e.g. `git pull --rebase <remote> <trunk>` (or equivalent fetch + rebase onto `remote/trunk`) so local trunk is linear with remote.
   - Do not rebase trunk "onto itself"; only update local trunk from remote.

3. **Stash / pop on trunk**
   - If the repo has uncommitted changes and is on trunk: stash before the update, then pop after successful update (same as feature-branch path).
   - Honor `--no-pop-stash`: do not pop stash after trunk update when flag is set.
   - On failure (e.g. conflict during pull/rebase): leave repo in conflicted state; do not pop stash until user resolves and re-runs or pops manually (consistent with current rebase conflict behavior).

4. **Conflict and abort behavior**
   - When trunk update fails due to conflicts: leave repo in conflicted state; display conflicts; do not pop stash.
   - When `--abort-on-conflict` is set and trunk update conflicts: abort the rebase (or equivalent) and restore pre-update state; then pop stash if not `--no-pop-stash`.
   - Reuse existing conflict display and recovery messaging where applicable.

5. **Feature-branch behavior unchanged**
   - When current branch is not trunk, behavior remains exactly as today: fetch + rebase onto trunk, with existing stash/pop and flags.

6. **Polyrepo**
   - Each repository is processed independently based on its current branch. A run may include both repos on trunk (trunk-update) and repos on feature branches (rebase onto trunk).

### Non-Functional Requirements

- No new config required; trunk branch resolution uses existing config.
- No breaking changes to existing `kira latest` usage on feature branches.

## Acceptance Criteria

- **AC1** When on trunk (current branch equals configured trunk), `kira latest` does not error with "cannot rebase onto itself"; it fetches and updates local trunk from remote.
- **AC2** With uncommitted changes on trunk, `kira latest` stashes before update and pops after successful update (unless `--no-pop-stash`).
- **AC3** With `--no-pop-stash`, after a successful trunk update, stashed changes remain in stash.
- **AC4** If trunk update fails due to conflicts, repo is left in conflicted state; stash is not popped; re-running `kira latest` continues (e.g. after user resolves and continues rebase) or user can pop stash manually.
- **AC5** When `--abort-on-conflict` is set and trunk update conflicts, the update is aborted and stash is popped (if not `--no-pop-stash`).
- **AC6** When on a feature branch, behavior is unchanged: fetch + rebase onto trunk, same stash/pop and flags.
- **AC7** Polyrepo: repos on trunk get trunk-update; repos on feature branches get rebase onto trunk; both can occur in one `kira latest` run.
- **AC8** Existing `kira latest` tests remain passing; new tests cover on-trunk path (update, stash/pop, conflict, abort).

## Implementation Notes

- **Detection**: Reuse `getCurrentBranch(repo.Path)` and compare to `repo.TrunkBranch` before choosing rebase vs trunk-update.
- **Trunk update**: After `fetchFromRemote(repo)`, if on trunk call a new helper (e.g. `updateTrunkFromRemote(repo)`) that runs the equivalent of `git pull --rebase <remote> <trunk>` (or `git rebase <remote>/<trunk>` after fetch). Use same timeout and env (e.g. `GIT_EDITOR=true`) as `continueRebase` where relevant.
- **Stash**: The existing stash/pop flow in `performFetchAndRebase` (and the parallel path) can be extended: after fetch, branch on "on trunk" → `updateTrunkFromRemote` else → `rebaseOntoTrunk`; stash/pop and error handling stay unified.
- **Display**: Discovery output can optionally indicate "on trunk" vs "on feature branch" per repo; state summary and conflict display remain as today.
- **Tests**: Add cases in `latest_test.go` for on-trunk (with/without stash, conflict, abort); ensure feature-branch tests unchanged.

## Release Notes

- **`kira latest` on trunk**: When the current branch is trunk, `kira latest` now updates local trunk from remote (fetch + pull/rebase) instead of failing. Stash-before and pop-after behavior applies the same as on feature branches. Use `--no-pop-stash` to keep changes stashed after a successful update.

## Slices

Slices are ordered; each slice is a committable unit of work. Tasks within a slice are implemented in order.

### Slice 1: Detect on-trunk and implement trunk update

- [ ] T001: Add helper to determine if repo current branch is trunk (compare current branch to `repo.TrunkBranch`).
- [ ] T002: Implement `updateTrunkFromRemote(repo)`: after fetch, run equivalent of `git pull --rebase <remote> <trunk>` (or rebase onto `remote/trunk`) so local trunk is updated from remote; use same timeout and non-interactive env as existing rebase.
- [ ] T003: Wire trunk-update into `performFetchAndRebase` / `processRepositoryUpdate`: when on trunk, call `updateTrunkFromRemote` instead of `rebaseOntoTrunk`; keep single stash/pop path for both branches.

**Commit scope**: On-trunk detection and trunk-update implementation; `kira latest` on trunk no longer errors with "cannot rebase onto itself".

### Slice 2: Stash and pop for trunk update

- [ ] T004: Ensure stash-before runs for repos on trunk (same as feature branch) when uncommitted changes exist.
- [ ] T005: Ensure pop-after runs for trunk-update success unless `--no-pop-stash`; on trunk-update failure do not pop (same as rebase conflict behavior).
- [ ] T006: When trunk-update fails with conflicts, leave repo in conflicted state and do not pop stash; reuse existing conflict display and recovery messaging for the repo.

**Commit scope**: Stash/pop and conflict behavior for trunk path; parity with feature-branch path.

### Slice 3: Abort-on-conflict and polish

- [ ] T007: When `--abort-on-conflict` is set and trunk-update fails due to conflicts, abort the in-progress rebase (or pull --rebase) and restore pre-update state; then pop stash if not `--no-pop-stash`.
- [ ] T008: Optionally show in discovery or state summary that a repo is "on trunk" vs "on feature branch" for clarity.
- [ ] T009: Update `kira latest` short/long description and help to mention that it works on both trunk and feature branches.

**Commit scope**: Abort-on-conflict for trunk, small UX improvements, and help text.

### Slice 4: Tests and docs

- [ ] T010: Add unit/integration tests for on-trunk path: update success, stash/pop, conflict, abort-on-conflict; ensure existing feature-branch tests unchanged.
- [ ] T011: Run e2e with on-trunk scenario if applicable; update README or user docs to describe `kira latest` on trunk.

**Commit scope**: Test coverage and documentation.
