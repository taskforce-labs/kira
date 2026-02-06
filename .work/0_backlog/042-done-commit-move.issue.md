---
id: 042
title: done commit move
status: backlog
kind: issue
assigned:
estimate: 0
created: 2026-02-06
tags: []
---

# done commit move
kira done moved the file to done but for some reason it didn't commit the removal of the file from the original status which was the review folder.

Output of `kira done 014`:
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

