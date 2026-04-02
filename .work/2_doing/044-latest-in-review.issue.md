---
id: 044
title: latest in review
status: doing
kind: issue
assigned: 
estimate: 0
created: 2026-02-25
tags: []
---

# latest in review

`kira latest` should not depend on work item status or which folder the work item is in. We only care whether each repo is on the trunk branch or not (on trunk: update from remote; on feature branch: rebase onto trunk).

## Steps to Reproduce

1. Start a work item and create a feature branch (e.g. `kira start 044`).
2. Submit for review so the work item moves to review (e.g. `kira review`); the work item file moves from `.work/2_doing/` to `.work/3_review/`.
3. On that same feature branch, run `kira latest` (e.g. to rebase onto latest trunk after review feedback).

## Expected Behavior

`kira latest` runs successfully: it discovers repositories from workspace behavior, and for each repo performs trunk update (when on trunk) or rebase onto trunk (when on a feature branch). Work item status and folder are irrelevant.

## Actual Behavior

`kira latest` fails with:

```
$ kira latest
Error: no work item found in doing folder (.work/2_doing): start a work item first
no work item found in doing folder (.work/2_doing): start a work item first
```

The command currently requires a work item in the doing folder before discovering repositories, so when the work item is in review (or in any other folder, or absent) it fails.

## Solution

Drop the work-item check entirely for `kira latest`. Do not use work item status or folder; repository discovery does not use the work item (the work item ID is already ignored in `resolveRepositoriesForLatest`). Trunk vs feature-branch behavior is determined by the current git branch per repo. Discover repos from workspace behavior only (standalone/monorepo = current dir, polyrepo = all projects); for each repo, if on trunk then update from remote, else rebase onto trunk. Remove the call to `findCurrentWorkItem` (and the work-item path/metadata steps) from `discoverRepositories`.

## Slices

### 1. Latest discovery without work item

Commit: `discoverRepositories` uses workspace behavior only; no doing-folder work item; clearer empty-repo error; drop dead discovery helpers if unused.

- [x] T001: Refactor `discoverRepositories` in `internal/commands/latest.go` to skip `findCurrentWorkItem` and `extractWorkItemMetadataForLatest`; run `detectWorkspaceBehavior` then `resolveRepositoriesForLatest` (same behavior as today aside from the removed gate).
- [x] T002: Remove the unused work-item ID argument from `resolveRepositoriesForLatest` (and update `discoverRepositories`, `discoverRepositoriesFromPath`, and call sites) if it is only passed for legacy reasons.
- [x] T003: Remove `findCurrentWorkItem` if nothing references it after T001; otherwise keep only where still required.
- [x] T004: Update `runLatest` empty-repository error text so it does not mention a work item (e.g. workspace-focused wording).

**Acceptance:** With a valid git workspace and an empty or missing `.work/2_doing/` (no work item in doing), `kira latest` discovers repos and proceeds like today (subject to repo state), matching expected behavior in the issue.

### 2. Tests and regression coverage

Commit: Unit tests assert no doing work item is required; existing latest flows stay covered.

- [x] T005: Replace or extend `TestDiscoverRepositories` (and any tests that expect “no work item in doing folder”) so an empty doing folder no longer fails discovery when the cwd is a valid standalone/polyrepo workspace; keep or adjust parity between `discoverRepositories` and `discoverRepositoriesFromPath` as appropriate.
- [x] T006: Remove or rewrite `TestFindCurrentWorkItem` (or equivalent) if `findCurrentWorkItem` is deleted in slice 1.
- [x] T007: Run `make check`; if e2e scripts still assume a doing work item for every `kira latest` scenario, add or adjust one scenario that runs `kira latest` with no markdown file in doing (e.g. work item only in review) and assert success.

**Acceptance:** `make check` passes; new or updated tests document the regression (latest while work item not in doing).

## Release Notes

- `kira latest` no longer depends on work item status or folder. It runs based only on workspace type and whether each repo is on trunk or a feature branch (on trunk: update from remote; on feature branch: rebase onto trunk).

