---
id: 021
title: ignore work in ci
status: done
kind: task
assigned:
estimate: 0
created: 2026-01-24
tags: []
---

# ignore work in ci

the ci build should ignore updates to the .work folder so we don't run the build needlessly

## Problem

Currently, the CI workflow runs on all pushes and pull requests to the `master` branch. This means that when work items in the `.work/` directory are updated (e.g., status changes, new items created, items moved between folders), the entire CI pipeline runs unnecessarily. This wastes CI resources and time, especially since work item changes don't affect the actual codebase functionality.

## Solution

Configure the GitHub Actions CI workflow to skip execution when only files in the `.work/` directory have changed. The workflow should still run if:
- Changes are made to files outside `.work/`
- Changes include both `.work/` files and other files (mixed changes)

## Implementation Details

### Approach: GitHub Actions Path Filters

Use GitHub Actions `paths-ignore` or conditional job execution to skip the build when only `.work/` changes are detected.

**Option 1: Using `paths-ignore` (Recommended)**
- Add `paths-ignore` to the workflow trigger conditions
- Ignore patterns: `['.work/**']`
- This will skip the entire workflow when only `.work/` files change

**Option 2: Conditional Job Execution**
- Keep the workflow trigger as-is
- Add a step to check if only `.work/` files changed
- Use `git diff` to detect changes
- Conditionally skip subsequent steps using `if` conditions

### Edge Cases to Consider

1. **Mixed Changes**: If a PR includes both `.work/` changes and code changes, CI should run normally
2. **Deleted Files**: Ensure path filters handle deleted files correctly
3. **Renamed Files**: Path filters should work for renamed files within `.work/`
4. **Empty Commits**: Handle edge case where only `.work/` changes result in no actual file changes

### Testing

After implementation, verify:
- [ ] CI skips when only `.work/` files are changed
- [ ] CI runs when code files are changed
- [ ] CI runs when both `.work/` and code files are changed
- [ ] CI runs on pull requests with mixed changes
- [ ] CI runs on direct pushes to master with code changes

### Files to Modify

- `.github/workflows/ci.yml` - Add path filters or conditional logic

### Example Implementation

```yaml
on:
  push:
    branches: [ master ]
    paths-ignore:
      - '.work/**'
  pull_request:
    branches: [ master ]
    paths-ignore:
      - '.work/**'
```

**Note**: The above approach will skip the workflow entirely when only `.work/` changes. If you want the workflow to start but skip specific jobs, use conditional job execution instead.

## Release Notes

- CI workflow now skips execution when only work items (`.work/` directory) are changed, reducing unnecessary CI runs and improving efficiency

