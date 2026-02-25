---
id: 045
title: start creates tracking
status: backlog
kind: prd
assigned:
estimate: 0
created: 2026-02-25
due: 2026-02-25
tags: []
---

# start creates tracking

when kira start runs and create the branch it should setup tracking so that you can use git operations without having to specify what branch or remote to pull/push etc

## Context

- **Current behavior**: `kira start` creates a worktree and branch via `createWorktreeWithBranch` (standalone) or `createBranchInWorktree` (polyrepo). When draft PR is enabled, `pushBranchesForDraftPR` calls `pushBranch(remoteName, branchName, worktreePath, false)`, which runs `git push <remote> <branch>` **without** `-u` / `--set-upstream`. The local branch is therefore not configured to track the remote branch.
- **Impact**: In the worktree, `git pull`, `git push`, and `git status` (ahead/behind) require specifying remote and branch (e.g. `git push origin 045-start-creates-tracking`), and default `git push` may push nothing or prompt for upstream.
- **Relevant code**: `internal/commands/start.go` — `pushBranch` (line ~2259, no `-u`), `pushBranchStandalone`, `pushBranchesPolyrepo`, `pushProjectBranchIfNeeded`. Remote name comes from `resolveRemoteName(ctx.Config, nil)` for main repo and from `p.Remote` for polyrepo projects.
- **Related**: `.work/IDEAS.md` item 36 — "Start sets upstream: when kira start creates the branch and pushes it should set the upstream so it's as easy as git pull for changes to the branch".

## Requirements

1. When `kira start` creates the branch **and** pushes it (draft PR path: GitHub remote, draft PR not skipped), set the branch's upstream to `<remote>/<branch>` in that worktree so `git pull`, `git push`, and branch status work without extra arguments.
2. Apply to both **standalone** (single repo) and **polyrepo** (main project + each project that gets pushed); each pushed worktree should have its current branch tracking the same branch on the appropriate remote.
3. Do not set tracking when no push occurs (e.g. `--no-draft-pr`, `--dry-run`, or when draft PR is disabled / non-GitHub). Optionally document that when the user later pushes manually, they can use `git push -u <remote> <branch>` once to establish tracking.

## Acceptance Criteria

- After `kira start <work-item-id>` with draft PR enabled (and push succeeding): in the created worktree, `git push` and `git pull` work without specifying remote or branch; `git status` shows correct ahead/behind vs the remote branch.
- Standalone: one worktree; its branch tracks `<resolveRemoteName>/<branchName>`.
- Polyrepo: main worktree and each project worktree that is pushed have their branch tracking the correct remote (main uses `resolveRemoteName`, each project uses its `project.Remote`) and branch name.
- `--no-draft-pr` or `--dry-run`: behavior unchanged; no tracking is set (no push, so no remote branch to track).
- Existing behavior (branch creation, draft PR creation, IDE, setup commands) unchanged except for the added upstream configuration after push.

## Slices

## Implementation Notes

- Change `pushBranch` to accept a `setUpstream bool`; when true, run `git push -u <remote> <branch>` instead of `git push <remote> <branch>`. Call from `pushBranchStandalone` and from `pushBranchesPolyrepo` / `pushProjectBranchIfNeeded` with `setUpstream: true` whenever a push is performed. Git sets upstream when push succeeds.
- Use the same remote name already resolved for the push: `resolveRemoteName(ctx.Config, nil)` for standalone/main, and `p.Remote` for polyrepo projects. Branch name is `ctx.BranchName` everywhere.
- Add unit tests for the new behavior (e.g. after start with draft PR, verify `git rev-parse --abbrev-ref --symbolic-full-name @{u}` in the worktree returns `<remote>/<branch>`). Consider e2e in `kira_e2e_tests.sh` if a scenario already runs start and push.
- Follow existing patterns: `executeCommand` with `gitCommandTimeout`, run from worktree directory for branch config; see `internal/commands/start.go` for similar git calls.

## Release Notes

- **start**: When `kira start` pushes the branch (draft PR flow), it now sets the branch's upstream to the remote branch. In the worktree you can use `git pull` and `git push` without specifying remote or branch, and `git status` shows ahead/behind correctly.
