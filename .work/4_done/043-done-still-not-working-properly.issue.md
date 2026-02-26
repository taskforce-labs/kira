---
id: 043
title: done still not working properly
status: done
kind: issue
created: 2026-02-24T00:00:00Z
assigned: null
estimate: 0
merge_commit_sha: 9245a66b7b7a42e2542c0635a26859060ca7fb9c
merge_strategy: rebase
merged_at: "2026-02-26T01:29:33Z"
pr_number: 23
tags: []
---

# done still not working properly

After the fix in issue #042, `kira done` now properly verifies that file deletions are staged before committing. However, the staging itself is still failing in certain scenarios, causing the command to fail with an error.

## Steps to Reproduce

1. Run `kira done 034 --force` (or any work item ID)
2. The command successfully:
   - Runs PR checks
   - Merges the pull request
   - Pulls trunk (master)
3. When attempting to update the work item to done, it fails with:
   ```
   Error: update work item to done: deletion was not staged: file .work/2_doing/034-ci-update-pr-details.prd.md deletion not found in staged changes
   ```
   Note: The actual work item was in review (`.work/3_review/`), not doing (`.work/2_doing/`), so this is a move from review to done.

## Expected Behavior

`kira done` should successfully:
1. Move the work item file from its current status folder (e.g., `.work/3_review/` for work items in review) to the done folder (`.work/4_done/`)
2. Stage both the deletion of the old file and the addition of the new file
3. Commit the move operation
4. Push the changes

## Actual Behavior

The command fails during the staging step. The verification added in issue #042 correctly detects that the deletion was not staged, but the root cause is that the staging commands (`git rm --cached` and the fallback `git add -u`) are failing silently or not working as expected.

## Root Cause Analysis

### Primary Issue: Stale Path Used for Commit After Pull

**The fundamental problem is that `updateWorkItemToDone` receives `workItemPath` which was determined BEFORE pulling trunk.** After the pull, if the file was already moved to done in trunk, the file is now at the target location, but the code still uses the stale path when committing.

**The exact problem sequence:**

1. **Before pull**: `resolveDoneWorkItemAndPR` calls `findWorkItemFile` and finds file at `.work/3_review/034-...`, stores in `ctx.WorkItemPath`

2. **Pull trunk**: Brings in commits that already moved the file to `.work/4_done/034-...` in the working directory

3. **Move attempt**: `moveWorkItemWithoutCommit` calls `findWorkItemFile` internally and finds file at `.work/4_done/...` (current location). It tries to rename to the same location (no-op), or the file is already there.

4. **Commit with stale path**: `commitWorkItemUpdate` is called with `workItemPath` parameter which is still `.work/3_review/034-...` (the stale path from step 1)

5. **Staging failure**: `commitMove` → `stageFileChanges` tries to stage deletion of `.work/3_review/034-...`, but git's index doesn't have that path (it was already moved in the pulled commit), so staging fails

**Why this happens:**
- `moveWorkItemWithoutCommit` finds the file at its current location (correct)
- But `commitWorkItemUpdate` uses the stale `workItemPath` parameter passed to `updateWorkItemToDone`
- The stale path points to a location that no longer exists in git's index

### Secondary Issue: Code Duplication

**Additionally, `kira done` is NOT using the existing `moveWorkItem` function** that already handles moves correctly. Instead, it's using `moveWorkItemWithoutCommit` + manual commit logic, which creates a separate code path that duplicates logic.

**However, even `moveWorkItem` would need an idempotent check** - if the file is already at the target location, we shouldn't try to move/commit it. We should only update metadata if needed.

**Will `moveWorkItem` fix this?**

Using `moveWorkItem` with `commitFlag=true` would help because:
- It calls `findWorkItemFile` internally, finding the file at its current location after the pull
- It uses the current paths (not stale ones) when committing

However, `moveWorkItem` still needs an **idempotent check**:
- If the file is already at the target location (`.work/4_done/...`), we shouldn't try to move/commit it
- We should detect this case and skip the move, only updating metadata if needed
- Otherwise, `moveWorkItem` would try to stage deletion and addition of the same file path, which would fail

3. **Insufficient error handling in fallback**: The fallback `git add -u` might succeed (return no error) but not actually stage anything if there are no changes to stage. The verification catches this, but we should handle it more gracefully.

4. **Missing check for file state**: Before attempting to stage the deletion, we should check if the file actually exists in the git index at the old path. If it doesn't (because it was already moved in a previous commit), we should skip the deletion staging step.

## Code Flow

1. `executeDone` (done.go:93) calls `pullTrunkAndUpdateWorkItem`
2. `pullTrunkAndUpdateWorkItem` (done.go:165):
   - Calls `pullTrunk` to pull latest trunk
   - Calls `updateWorkItemToDone` with `ctx.WorkItemPath` (determined before pull)
3. `updateWorkItemToDone` (done.go:741):
   - Calls `moveWorkItemWithoutCommit` which moves the file on disk using `os.Rename`
   - Calls `commitWorkItemUpdate` with `workItemPath` (old path) and `targetPath` (new path)
4. `commitWorkItemUpdate` (done.go:634) calls `commitMove` (doesn't pass `repoRoot`)
5. `commitMove` (move.go:286) calls `stageFileChanges`
6. `stageFileChanges` (move.go:254):
   - Tries `git rm --cached oldPath` - fails if file not in index
   - Falls back to `git add -u oldDir` - may succeed but stage nothing
   - Calls `verifyDeletionStaged` which detects deletion wasn't staged
   - Returns error: "deletion was not staged: file ... deletion not found in staged changes"

## Potential Solutions

### Option 1: Use Existing `moveWorkItem` Function (RECOMMENDED - Addresses Root Cause)

**Refactor `updateWorkItemToDone` to use the existing `moveWorkItem` function:**

1. **Extend `moveWorkItem` to accept additional frontmatter fields**: Since `moveWorkItem` already updates the `status` field in frontmatter, extend it to accept optional additional frontmatter fields to update:
   ```go
   // Add optional parameter for additional frontmatter fields
   func moveWorkItem(cfg *config.Config, workItemID, targetStatus string, commitFlag, dryRun bool, additionalFields map[string]interface{}) error
   ```
   - When `additionalFields` is provided (non-nil), update those fields in the frontmatter along with the status field
   - Use the existing `parseWorkItemFrontMatter` and `writeWorkItemFrontMatter` functions (same approach as `updateWorkItemDoneMetadata`)
   - Update fields after moving the file but before committing (if `commitFlag=true`)

2. **Use extended `moveWorkItem` in `done`**: 
   ```go
   // Instead of:
   moveWorkItemWithoutCommit(cfg, workItemID, defaultReleaseStatus)
   updateWorkItemDoneMetadata(...)
   commitWorkItemUpdate(...)
   
   // Use:
   additionalFields := map[string]interface{}{
       "merged_at": mergedAt,
       "merge_commit_sha": mergeCommitSHA,
       "pr_number": prNumber,
       "merge_strategy": mergeStrategy,
   }
   moveWorkItem(cfg, workItemID, defaultReleaseStatus, true, false, additionalFields)
   ```
   This approach:
   - Uses the existing `moveWorkItem` function (eliminates code duplication)
   - Updates both status and completion metadata in a single operation
   - Commits everything together in one commit
   - Uses the tested staging logic from `moveWorkItem` (with fixes from issue #042)
   - Cleaner API - all frontmatter updates happen in one place

3. **Add idempotent check**: Before calling `moveWorkItem`, check if the file is already at the target location:
   - Use `findWorkItemFile` to locate the file (finds it regardless of current location after pull)
   - If already at target location, skip the move but still update metadata if needed
   - This handles the case where the file was already moved in a commit that was pulled

**Benefits:**
- Eliminates code duplication
- Uses tested, working code path
- Automatically benefits from future fixes to `moveWorkItem`
- Simpler, more maintainable code

**Note:** The `moveWorkItem` function already includes the verification logic from issue #042, so using it would automatically fix the staging issue.

## Related Issues

- Issue #042: Fixed silent error handling and added verification, but didn't address the root cause of why staging fails after pulling trunk

## Investigation Needed

1. **Extend `moveWorkItem` signature**: Add optional `additionalFields map[string]interface{}` parameter to allow updating additional frontmatter fields:
   - Update function signature and all call sites
   - Modify `executeMoveWorkItem` to accept and use `additionalFields`
   - Use `parseWorkItemFrontMatter` and `writeWorkItemFrontMatter` to update fields (same approach as `updateWorkItemDoneMetadata`)
   - Ensure fields are updated after the file move but before committing

2. **Check idempotent behavior**: After pulling trunk, verify:
   - Does `findWorkItemFile` correctly find the file if it's already in done folder?
   - Should we check file location before attempting move?
   - How to handle case where file is already at target (skip move, only update metadata)?

3. **Metadata update integration**: With the extended `moveWorkItem`, metadata updates will be integrated into the move operation:
   - `moveWorkItem` will update both `status` and any `additionalFields` in frontmatter
   - All frontmatter updates happen in one operation (cleaner than separate calls)
   - If `commitFlag=true`, everything is committed together in one commit
   - This maintains the desired behavior (move + metadata in single commit) while eliminating code duplication

4. **Test existing `moveWorkItem` behavior**: Verify that `moveWorkItem` with `commitFlag=true` correctly handles:
   - Files that don't exist in git index at old path (after pull)
   - Idempotent moves (file already at target)
   - Error cases that might occur in `done` context

## Slices

### Extend moveWorkItem with additionalFields
Commit: Add optional additionalFields to moveWorkItem for completion metadata; update frontmatter and call sites
- [x] T001: Add optional parameter additionalFields map[string]interface{} to moveWorkItem and executeMoveWorkItem; update all call sites (move command, tests)
- [x] T002: When additionalFields is non-nil, update those frontmatter fields after moving file (and status) and before committing; reuse parse/write frontmatter logic (e.g. from updateWorkItemDoneMetadata)
- [x] T003: Add or adapt unit tests for moveWorkItem with additionalFields

### Idempotent check in moveWorkItem
Commit: If work item already at target path skip move and staging; only update frontmatter and commit if needed
- [x] T004: After findWorkItemFile, if workItemPath equals targetPath (already in target status folder), skip os.Rename and move staging; only update frontmatter (status + additionalFields) and if commitFlag commit metadata
- [x] T005: Add unit tests for idempotent case (file already at target)

### Refactor done to use moveWorkItem
Commit: Replace moveWorkItemWithoutCommit + metadata + commitWorkItemUpdate with single moveWorkItem call
- [x] T006: In updateWorkItemToDone build additionalFields from mergedAt mergeCommitSHA prNumber mergeStrategy and call moveWorkItem(cfg workItemID defaultReleaseStatus true false additionalFields); keep push and trunk resolution in done
- [x] T007: Remove or simplify redundant move/commit logic and ensure kira done e2e or manual flow passes

### Tests and release notes
Commit: E2e for done after pull; unit tests; release notes
- [x] T008: Add e2e or integration test for kira done when work item file already moved on trunk (or simulate post-pull)
- [x] T009: Update Release Notes in work item (and docs if needed) for reliable kira done after pull and idempotent behavior

## Release Notes

Fixed. `kira done` now works reliably even when:
- The work item file was already moved in a previous commit (e.g. on trunk after pull)
- Trunk was pulled and contains commits that affect the work item file (path is resolved by ID after pull, so no stale path)
- The command is run multiple times (idempotent: if file already at target, only frontmatter is updated and committed when changed)

Implementation: `updateWorkItemToDone` now calls the shared `moveWorkItem` with completion metadata as `additionalFields`. Move path resolves the work item by ID (current location), handles already-at-target idempotently, and stages/commits using the same logic as `kira move`.

