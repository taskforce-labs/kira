---
id: 042
title: done commit move
status: doing
kind: issue
assigned:
estimate: 0
created: 2026-02-06
tags: []
---

# done commit move
kira done moved the file to done but for some reason it didn't commit the removal of the file from the original status which was the review folder.

## Problem Summary

When `kira done` moves a work item from review (or any other status) to done, it creates a commit that adds the file to the done folder but fails to stage and commit the deletion of the file from the original location. This results in two separate commits:
1. A commit adding the file to `.work/4_done/`
2. A separate commit removing the file from `.work/3_review/` (or original status folder)

## Evidence

From git history for work item 014:
- Commit `416dc7a`: "014 move: to done - prd kira-done" - Only added `.work/4_done/014-kira-done.prd.md` (status `A`)
- Commit `8d402bb`: "014 remove from review" - Manually removed `.work/3_review/014-kira-done.prd.md` (status `D`)

The first commit should have included both the addition and deletion in a single move operation.

## Root Cause Analysis

The issue is in the `stageFileChanges` function in `internal/commands/move.go` (lines 207-223):

```go
func stageFileChanges(ctx context.Context, oldPath, newPath string, dryRun bool) error {
	// Stage the old file deletion first using git rm --cached
	_, cmdErr := executeCommand(ctx, "git", []string{"rm", "--cached", oldPath}, "", dryRun)
	if cmdErr != nil && !dryRun {
		// If git rm fails (file wasn't tracked), try git add -u to stage deletions
		oldDir := filepath.Dir(oldPath)
		_, _ = executeCommand(ctx, "git", []string{"add", "-u", oldDir}, "", dryRun)
	}

	// Stage the new file addition
	_, cmdErr = executeCommand(ctx, "git", []string{"add", newPath}, "", dryRun)
	if cmdErr != nil && !dryRun {
		return fmt.Errorf("failed to stage changes: %w", cmdErr)
	}
	return nil
}
```

**Problems identified:**

1. **Silent error handling**: When `git rm --cached oldPath` fails, the fallback `git add -u oldDir` is attempted, but its error is ignored (`_, _`). If both commands fail, the deletion is never staged, but the function doesn't return an error.

2. **Error handling logic**: The function only checks for errors when staging the new file addition (line 219), but silently continues if the deletion staging fails.

3. **Path resolution**: After `moveWorkItemWithoutCommit` moves the file on disk, `workItemPath` still points to the old location. If `git rm --cached` fails for any reason (wrong path, file not tracked, etc.), the fallback may not work correctly.

## Code Flow

1. `updateWorkItemToDone` (done.go:741) calls `moveWorkItemWithoutCommit` which moves the file on disk
2. `commitWorkItemUpdate` (done.go:634) is called with `workItemPath` (old path) and `targetPath` (new path)
3. `commitMove` (move.go:226) calls `stageFileChanges` to stage both deletion and addition
4. `stageFileChanges` attempts to stage deletion but silently fails if both `git rm --cached` and `git add -u` fail
5. Only the new file addition gets staged and committed

## Expected Behavior

When `kira done` moves a work item, it should create a single commit that:
- Stages the deletion of the file from the old location (e.g., `.work/3_review/`)
- Stages the addition of the file to the new location (e.g., `.work/4_done/`)
- Commits both changes together as a move operation

## Related Code

The `commitStatusChange` function in `internal/commands/start.go` (lines 2192-2225) has similar logic and may have the same issue. However, it's used by the `kira start` command which may operate in a different context. This should be investigated to ensure consistency across all move operations.

## Potential Fix Approaches

1. **Improve error handling**: Check if the deletion was successfully staged before proceeding. Verify staged changes after `git rm --cached` and `git add -u` to ensure the deletion is actually staged.

2. **Use git mv**: Consider using `git mv` instead of `os.Rename` + manual staging, which would handle the move atomically.

3. **Verify staging**: After attempting to stage the deletion, verify it was actually staged using `git diff --cached` before proceeding to stage the addition.

4. **Better error reporting**: Return an error if both `git rm --cached` and `git add -u` fail, rather than silently continuing.

## Output of `kira done 014`

```text
$ kira done 014

Completing work item 014
  Running PR checks for #19...
  ✓ PR checks passed
  Merging pull request #19 (rebase)...
  ✓ PR merged
  Pulling trunk (master)...
  ✓ Trunk up to date
  Updating work item to done...
  ✓ Work item marked done and pushed
  Deleting local branch 014-kira-done...
  ⚠ Local branch not found (may already be deleted)
  Deleting remote branch 014-kira-done...
  ✓ Remote branch deleted

✓ Work item 014 completed
```

## Slices

### Fix error handling in stageFileChanges
Commit: Fix silent error handling in stageFileChanges to ensure deletion is staged
- [x] T001: Fix error handling in stageFileChanges - return error if git rm --cached fails and git add -u fallback also fails
- [ ] T002: Check error return from git add -u fallback instead of ignoring it with `_, _`
- [ ] T003: Return descriptive error message if deletion staging fails

### Add verification of staged deletion
Commit: Add verification step to ensure deletion was actually staged before proceeding
- [ ] T004: Add function to verify deletion is staged using git diff --cached
- [ ] T005: Call verification after staging deletion in stageFileChanges
- [ ] T006: Return error if verification fails (deletion not staged)

### Fix similar issue in commitStatusChange
Commit: Apply same error handling fixes to commitStatusChange in start.go for consistency
- [ ] T007: Review commitStatusChange function in internal/commands/start.go
- [ ] T008: Apply same error handling improvements to commitStatusChange
- [ ] T009: Add verification step to commitStatusChange if needed

### Add tests for stageFileChanges fix
Commit: Add unit and integration tests to verify stageFileChanges correctly stages deletions
- [ ] T010: Add unit tests for stageFileChanges error handling scenarios
- [ ] T011: Add integration test for kira done move operation to verify single commit contains both deletion and addition
- [ ] T012: Add test case for when git rm --cached fails but git add -u succeeds

