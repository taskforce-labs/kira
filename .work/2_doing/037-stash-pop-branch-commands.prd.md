---
id: 037
title: stash pop branch commands
status: doing
kind: prd
assigned:
created: 2026-02-02
tags: []
---

# stash pop branch commands

Today kira latest supports stash/pop before and after a rebase.

This PRD extends stash/pop to other branch commands:
- start
- done
- save, move — out of scope for this PRD (no pull/branch ops today; document for future).


## Context

### Current behaviour by command

| Command | Pull / push / branch ops | Uncommitted changes today | Stash/pop today |
|--------|--------------------------|---------------------------|-----------------|
| **latest** | Fetch + rebase (feature) or pull (trunk) | Auto stash → rebase → pop (or `--no-pop-stash`) | Yes, full flow in `latest.go` |
| **start** | Pull trunk (fetch+merge) then create worktree + branch, optional push for draft PR | **Fails**: "trunk branch has uncommitted changes... Commit or stash changes before starting work" | No |
| **save** | None (stages work folder, commits) | Allows other changes; warns on external changes, skips commit if mixed | No |
| **move** | None (moves file, optional commit) | No pull; commit can fail if other staged changes | No |
| **done** | Pull trunk, then push trunk (after merge PR) | No stash; assumes clean or acceptable state | No |

### Where git helpers live

- **Uncommitted check**: `checkUncommittedChanges(dir, dryRun)` in `start.go`; `checkUncommittedChangesForLatest(dir)` in `latest.go` — same idea, different signatures (duplication).
- **Pull**: `pullLatestChanges(remote, branch, dir, dryRun)` in `start.go` (fetch + merge); `pullTrunk(ctx, repoRoot, remote, branch)` in `done.go` (git pull). Different strategies (merge vs pull).
- **Push**: `pushStatusChange`, `pushBranch`, `pushBranchForceWithLease` in `start.go`; `done.go` uses `executeCommand` for `git push` directly.
- **Stash/pop**: Only in `latest.go`: `stashChanges(repo RepositoryInfo)`, `popStash(repo RepositoryInfo)`. Message format: `"kira latest: auto-stash before rebase on %s"`. No shared helper used elsewhere.

So: stash/pop and “run something that needs a clean tree” are only implemented for **latest**; **start** requires a clean tree and fails otherwise; **save** and **move** don’t do pull/branch ops today.


## Requirements

1. **start**
   - When trunk has uncommitted changes, optionally **stash → pull → create worktree/branch → pop** (mirror latest’s behavior) instead of failing.
   - Support a **`--no-pop-stash`** (or equivalent) so the user can leave changes stashed after start.
   - Keep existing behavior when there are no uncommitted changes (no stash/pop).

2. **save**
   - No pull or branch switch today; stash/pop is only relevant if we add a “sync with remote” step later.
   - **Out of scope for initial implementation** unless we add such a step; document as future option.

3. **move**
   - No pull or branch switch today; stash/pop only relevant if we add e.g. “pull trunk before move” or “push after move”.
   - **Out of scope for initial implementation** unless we add such a step; document as future option.

4. **Consistency**
   - Stash message and pop behaviour should be consistent with `latest` (e.g. same message prefix `kira <command>: ...`).
   - Where we add stash/pop, support a **`--no-pop-stash`** flag for parity with latest.

5. **done**
   - When trunk has uncommitted changes, **done** uses `RunWithCleanTree` around pull+update so it doesn’t fail or behave oddly. Same “run with clean tree” invariant; add `--no-pop-stash` for parity with latest/start.


## Design options

This section explains the three ways we could structure the code, and what each implies for where logic lives and who maintains it.

---

### What we’re trying to share

Today **latest** already has:

1. **Check** — “does this repo have uncommitted changes?” (git status --porcelain)
2. **Stash** — run `git stash push -m "..."` and handle “nothing to stash”
3. **Do the operation** — fetch + rebase (or pull on trunk)
4. **Pop** — run `git stash pop` and handle “no stash” / “pop had conflicts”
5. **Restore on failure** — if step 3 fails, pop the stash before returning so the user isn’t left with both a failed rebase and a stash

**Start** will need the same pattern around **pull** (steps 1 → 2 → pull → 4, and 5 if pull fails). So we have to decide: do we share only the low-level primitives (check, stash, pop), or do we also share the “wrap an operation with stash/pop” pattern?

---

### Option A: Centralize low-level helpers only

**Idea:** Add a small shared layer (e.g. `internal/git/stash.go` or `internal/commands/stash.go`) with:

- `HasUncommitted(dir string, dryRun bool) (bool, error)` — one implementation for “is working tree dirty?”
- `Stash(dir, message string) error` — run stash push, treat “nothing to stash” as success
- `Pop(dir string) error` — run stash pop, treat “no stash” as success, report conflicts as error

**Who builds the message?** The **caller** (latest, start) builds the message, e.g. `Stash(repo.Path, "kira start: auto-stash before pull on "+repo.Name)`.

**Where does “stash → do X → pop” live?** Still in each command. **Latest** keeps its current flow in `latest.go` (check → stash → fetch+rebase → pop, plus restore-on-failure). **Start** adds a similar flow in `start.go` (check → stash → pull → pop, plus restore-on-failure). So we **remove duplication** of the git commands and the “nothing to stash / no stash to pop” handling, but we **keep duplication** of the “when to stash, when to pop, what to do on failure” logic in two places.

**Implications:**

- **Pros:** Smallest shared surface; each command stays in full control of its flow (important for latest’s polyrepo / multi-repo ordering and error handling). Easy to add a third command later that uses the same helpers with a different flow.
- **Cons:** If we change the “restore stash on failure” rule or add a new edge case (e.g. “don’t pop if rebase was aborted”), we have to remember to update both latest and start.

---

### Option B: Same as A, but described as “extract only shared helpers”

Option B in the original PRD is the **same as Option A** in practice: we extract `Stash` / `Pop` (and unify the uncommitted check). The only nuance was “keep the logic of when to stash in each command” — which is what Option A does. So **Option A and B are the same approach**; the PRD was redundant. The recommendation “Option B for first step” means: do the shared helpers (A/B), and don’t yet introduce a generic wrapper (C).

---

### Option C: Generic “run with clean tree” wrapper

**Idea:** One function that encodes the invariant **“this operation runs with a clean tree”**:

```go
// RunWithCleanTree runs fn() after ensuring a clean tree, stashing if needed.
// If noPopStash is false and we stashed, it pops after fn() succeeds.
// If fn() fails and we had stashed, it pops before returning (restore).
func RunWithCleanTree(dir, opName string, noPopStash bool, fn func() error) (hadStash bool, err error)
```

So **start** does: `RunWithCleanTree(repoRoot, "start", noPopStash, func() error { return pullLatestChanges(...) })`.  
**Latest** runs over multiple repos, so for **each repo** it calls: `RunWithCleanTree(repo.Path, "latest", noPopStash, func() error { return doFetchAndRebase(repo) })`. The wrapper handles one directory; latest’s loop and ordering stay in place, but the “check → stash → do → pop / restore” semantics live in one place.

**Where does the logic live?** The “run with a clean tree” pattern lives **once** inside `RunWithCleanTree`. Every command that needs a clean tree (latest, start, and any future command like save/move if they gain a pull step) uses this wrapper and only supplies “what to do” (the callback).

**Implications:**

- **Pros:** The invariant “these commands run with a clean tree” is explicit and centralized. One place for check → stash → run → pop and restore-on-failure. Any change to that behaviour (e.g. “don’t pop if there were conflicts”) is done once and applies to all commands. Fits the mental model: “we run with a clean tree.”
- **Cons:** Latest needs a small refactor so its per-repo work is a function that can be passed to `RunWithCleanTree` (instead of inlining the stash/pop flow). That’s a one-time reshape; after that, both latest and start share the same wrapper.

---

### Summary and decision

| Option | What’s shared | What stays in each command | Best when |
|--------|----------------|----------------------------|-----------|
| **A (or B)** | Uncommitted check, `Stash(dir, message)`, `Pop(dir)` | The full flow: when to stash, when to pop, what to run in between, restore on failure | You want minimal change; accept duplicated flow in two places. |
| **C** | **RunWithCleanTree** — check → stash → fn() → pop (and restore on failure). Commands pass “what to do” (callback). | Only “what is the operation?” (e.g. pull, or fetch+rebase for one repo) | You want one place that enforces “run with a clean tree” for all these commands. |

**Decision: Option C.** These commands (latest, start, and any future ones that need a clean tree) should all “run with a clean tree.” Option C makes that invariant a single function: `RunWithCleanTree`. Latest uses it per repo (callback = fetch+rebase for that repo); start uses it once (callback = pull). Shared helpers (`HasUncommitted`, `Stash`, `Pop`) live inside or beside `RunWithCleanTree`; callers only pass the operation to run.


### Does this work for done? Does it change the flows?

**Done:** Yes. **done** runs on trunk, then pulls trunk and updates the work item (move file, commit, push). It does not today check for uncommitted changes; if you have local changes, `pullTrunk` can fail or behave oddly. We wrap the “pull + update work item” part in `RunWithCleanTree(repoRoot, "done", noPopStash, func() error { pullTrunk(...); return updateWorkItemToDone(...) })`. Done gets the same invariant: run with a clean tree. Add `--no-pop-stash` to done for parity. **Done is in scope.**

**Do start and done change in a fundamental way?** No. The **steps and order** stay the same:

- **Start:** Validate work item → pull trunk → status check → status update → create worktrees → push branches → IDE → setup. The only change is: instead of **failing** when trunk has uncommitted changes, we **stash → pull → … → pop** (or leave stashed with `--no-pop-stash`). Same flow; we just allow uncommitted changes and handle them automatically.
- **Done:** Merge PR (or use metadata) → pull trunk → update work item to done (move, commit, push) → cleanup worktree/branch. If we add `RunWithCleanTree` around pull+update, the steps and order are unchanged; we just stash before that block and pop after (or on failure), so done works even when trunk has uncommitted changes.

So we’re not reordering or redefining what start or done do — we’re making the “clean tree” requirement explicit and automatic (stash/pop) instead of failing (start) or assuming clean (done).


## Refactoring (Option C: run with clean tree)

1. **Introduce `RunWithCleanTree`**
   - Add a shared function (e.g. in `internal/commands/stash.go` or `internal/commands/clean_tree.go`): `RunWithCleanTree(dir, opName string, noPopStash bool, fn func() error) (hadStash bool, err error)`.
   - Inside it: use a single `HasUncommitted(dir, false)`, then if dirty call `Stash(dir, "kira "+opName+": auto-stash before ...")`, run `fn()`, then pop (unless noPopStash or fn failed); on fn failure after stash, pop before returning.
   - Implement or extract `HasUncommitted` and `Stash`/`Pop` as helpers used by `RunWithCleanTree` (and optionally by callers that need them directly).

2. **Refactor latest to use `RunWithCleanTree` per repo**
   - For each repository in latest’s flow, the “do the operation” step (fetch + rebase, or pull on trunk) becomes a callback passed to `RunWithCleanTree(repo.Path, "latest", noPopStash, func() error { return doFetchAndRebase(repo) })` (or equivalent). Latest’s loop, ordering, conflict handling, and result aggregation stay in `latest.go`; only the per-repo “check → stash → do → pop / restore” is delegated to the wrapper. Handle latest-specific cases (e.g. “rebase had conflicts so keep stash”) inside the callback or by having the callback return a sentinel so latest can decide not to pop — document the chosen approach in Implementation Notes.

3. **Start command**
   - In `validateAndPullLatest` (or the place that currently fails on uncommitted changes): call `RunWithCleanTree(repoRoot, "start", noPopStash, func() error { return pullLatestChanges(...) })` for the main repo. For polyrepo, call `RunWithCleanTree` per project before pulling that project (same pattern as latest).
   - Add `--no-pop-stash` to start’s flags and wire it through.

4. **Done command**
   - Wrap the pull+update block in `RunWithCleanTree(ctx.RepoRoot, "done", noPopStash, func() error { pullTrunk(...); return updateWorkItemToDone(...) })`. Add `--no-pop-stash` to done for parity with latest/start.
   - Ensures done works when trunk has uncommitted changes and keeps the “run with clean tree” invariant for all branch commands that touch trunk.

5. **Pull/push**
   - No need to centralize pull/push in this PRD: start uses `pullLatestChanges` (fetch+merge), done uses `pullTrunk` (pull). Document the difference; consider unifying later if desired.


## Acceptance Criteria

- [ ] **start** with uncommitted changes on trunk: stashes, pulls, creates worktree/branch as today, then pops stash (unless `--no-pop-stash`).
- [ ] **start --no-pop-stash** with uncommitted changes: stashes, pulls, creates worktree; stash is left in place; user can run `git stash pop` later.
- [ ] **start** with no uncommitted changes: behavior unchanged (no stash/pop).
- [ ] If pull fails during start after a stash: stash is popped (restored) before returning the error.
- [ ] Stash message for start is distinct and consistent (e.g. `kira start: auto-stash before pull on <repo>`).
- [ ] **latest** behavior unchanged (including `--no-pop-stash` and conflict/rebase handling).
- [ ] **done** with uncommitted changes on trunk: stashes, pulls, updates work item, pushes, then pops stash (unless `--no-pop-stash`). If pull or update fails, stash is restored before returning.
- [ ] Uncommitted-check and/or stash/pop refactor: no duplication between start and latest for “has uncommitted” and “stash/pop”; tests updated.
- [ ] `make check` and e2e (`bash kira_e2e_tests.sh`) pass.


## Implementation Notes

- **RunWithCleanTree** should take `opName` and optionally a repo name for stash messages (e.g. `"kira start: auto-stash before pull on main"`), so polyrepo and standalone both get clear messages. Either pass a `repoName string` or build the message inside the wrapper from `opName` and `dir`.
- **Latest’s “rebase had conflicts, keep stash” case:** Today latest sometimes does not pop when rebase leaves conflicts (so the user can resolve and re-run). Options: (1) have `RunWithCleanTree` always pop on failure (restore working tree), and let latest detect “rebase conflicted” and avoid calling pop in that path by handling it inside the callback and returning a special error or flag; or (2) add a parameter or return from the callback like `RestoreStashOnFailure bool` so the wrapper can decide. Prefer the simplest design that preserves current latest behaviour; document the chosen approach.
- **save** / **move**: When we add pull/sync steps to these commands, they can call `RunWithCleanTree` too; document that in release notes or docs.


## Slices

### Shared clean-tree helpers
Commit: Add RunWithCleanTree and stash helpers (HasUncommitted, Stash, Pop) in a shared package; no command behavior change yet.
- [x] T001: Add internal package (e.g. internal/commands/stash.go or clean_tree.go) with HasUncommitted(dir, dryRun), Stash(dir, message), Pop(dir) and RunWithCleanTree(dir, opName, noPopStash, fn)
- [x] T002: Implement stash message format and restore-on-failure semantics inside RunWithCleanTree
- [x] T003: Add unit tests for HasUncommitted, Stash, Pop, and RunWithCleanTree

### Latest uses RunWithCleanTree
Commit: Refactor latest to use RunWithCleanTree per repo; preserve existing behavior, --no-pop-stash, and conflict/rebase handling.
- [x] T004: Refactor latest.go so per-repo "do the operation" (fetch+rebase or pull) is a callback passed to RunWithCleanTree
- [x] T005: Preserve latest-specific behavior (e.g. rebase conflicts / keep stash) per Implementation Notes; document chosen approach
- [x] T006: Update or add tests for latest; ensure make check and e2e pass

### Start: stash/pop and --no-pop-stash
Commit: Use RunWithCleanTree in start flow and add --no-pop-stash flag; start no longer fails on uncommitted trunk changes.
- [x] T007: In start flow (validateAndPullLatest or equivalent), call RunWithCleanTree(repoRoot, "start", noPopStash, pullCallback) for main repo; polyrepo: RunWithCleanTree per project before pull
- [x] T008: Add --no-pop-stash to start flags and wire through
- [x] T009: Add/update tests for start with uncommitted changes, start --no-pop-stash, and restore-on-failure when pull fails

### Done: stash/pop and --no-pop-stash
Commit: Use RunWithCleanTree in done flow and add --no-pop-stash flag; done works with uncommitted trunk changes.
- [ ] T010: Wrap pull+update block in done with RunWithCleanTree(ctx.RepoRoot, "done", noPopStash, fn); add --no-pop-stash flag
- [ ] T011: Add/update tests for done with uncommitted changes, done --no-pop-stash, and restore-on-failure
- [ ] T012: Run make check and bash kira_e2e_tests.sh; fix any failures

### Release notes and docs
Commit: Add release notes and document save/move future use of RunWithCleanTree.
- [ ] T013: Add release notes for start, done, and "run with clean tree" per PRD Release Notes section
- [ ] T014: Document that save/move can use RunWithCleanTree when pull/sync steps are added (release notes or docs)


## Release Notes

- **start**: When trunk has uncommitted changes, `kira start` now stashes them, pulls latest, creates the worktree and branch, then pops the stash (same workflow as `kira latest`). Use `--no-pop-stash` to leave changes stashed.
- **done**: When trunk has uncommitted changes, `kira done` stashes them before pull+update, then pops after (or use `--no-pop-stash`). Flow unchanged; only the “clean tree” requirement is now automatic.
- **Run with clean tree**: Branch commands that need a clean working tree (latest, start, done) share a single “run with clean tree” helper: they stash if needed, run the operation, then pop (or restore on failure). Behaviour is consistent and future commands (e.g. save/move with a pull step) can use the same helper.
- **save/move**: When pull or sync steps are added to `kira save` or `kira move` in the future, they can use the same RunWithCleanTree helper; no change in this release.


