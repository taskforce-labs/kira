---
id: 035
title: wip limits
status: backlog
kind: prd
assigned:
estimate: 0
created: 2026-02-02
due: 2026-02-02
tags: []
---

# wip limits

Work-in-progress (WIP) limits constrain how many work items may sit in a given status folder (e.g. "doing"). Today, when the doing folder has more than one work item, `kira lint` can fail because the limit is effectively hardcoded (e.g. 1). This PRD adds configurable WIP limits per folder in `kira.yml` so teams can enforce limits (e.g. "at most 1 in doing", "at most 3 in review") and adjust them without code changes.

## Context

### Problem Statement

Kira uses status folders (backlog, todo, doing, review, done, etc.) to represent workflow. To avoid context-switching and unfinished work piling up, many teams want to cap how many items can be in "active" folders (notably doing, and sometimes review). Currently any such limit is hardcoded (e.g. 1 for doing). That causes `kira lint` to fail when the doing folder has multiple items—which is correct behavior, but the limit value and which folders are limited cannot be configured. Teams cannot relax the limit (e.g. allow 2 in doing) or add limits for other folders (e.g. review) without changing code.

### Proposed Solution

- Add a **WIP limits** section in `kira.yml` (e.g. under `validation` or a new top-level key) that maps **status names** (as used in `status_folders`) to **maximum count** of work items allowed in that folder.
- **`kira lint`** (via validation) SHALL enforce these limits: for each configured status, count work items in the corresponding folder; if the count exceeds the configured limit, report a workflow error.
- If no limit is configured for a status, or the limit is 0, no cap is enforced for that folder (unlimited).
- Default for WIP limits is 0 (no restrictions): when a limit is 0 or omitted, no check is performed for that folder.

### Scope

- **In scope:** Configuration of WIP limits in `kira.yml`; validation in `kira lint` that counts items per status folder and fails when a limit is exceeded; clear error messages naming the folder and the limit.
- **Out of scope:** Blocking `kira move` or `kira start` when a move would exceed a limit (could be a follow-up); per-kind or per-user limits; UI or dashboard for limits.

## Requirements

### Functional Requirements

#### FR1: Configuration Schema

- WIP limits SHALL be configurable in `kira.yml`.
- Limits SHALL be keyed by **status name** (the keys in `status_folders`, e.g. `doing`, `review`), not by folder path.
- Each value SHALL be a non-negative integer: 0 means no restrictions (unlimited); a positive value is the maximum number of work items allowed in that status folder.
- Only statuses that exist in `status_folders` SHALL be allowed in the WIP limits map; invalid keys SHALL be rejected at config load time (or validation time) with a clear error.

#### FR2: Validation Behaviour

- When `kira lint` runs, the validator SHALL, for each status that has a WIP limit configured:
  1. Resolve the folder path from `status_folders` (and the work folder).
  2. Count work item files (e.g. `*.md`, excluding templates and IDEAS.md) in that folder.
  3. If the configured limit is greater than 0 and count > limit, add a **workflow** validation error (so it appears under "Workflow Errors" in lint output).
- The error message SHALL include the status name, the folder path (or name), the current count, and the configured limit (e.g. "doing folder has 3 work items, exceeds WIP limit of 1").

#### FR3: Default 0 Means No Restrictions

- If a status is not mentioned in the WIP limits configuration, no check SHALL be performed for that folder (unlimited).
- A value of 0 SHALL mean no restrictions: no check is performed for that folder. Default for WIP limits is 0 (no restrictions).

#### FR4: Integration with Existing Lint

- WIP limit errors SHALL be reported via the same mechanism as other workflow errors (e.g. `result.AddError("workflow", ...)` in `validateWorkflowRules`), so they appear in the "Workflow Errors" section of `kira lint` output and cause lint to exit with failure when present.

### Configuration

#### Suggested kira.yml shape

```yaml
validation:
  # ... existing validation options ...
  wip_limits:
    doing: 1
    review: 3
```

- Keys MUST be status names from `status_folders`. Values MUST be non-negative integers (0 = no restrictions).
- Config load (or validate) SHALL reject unknown status keys and invalid values (e.g. negative integers).

### Defaults

- Default for WIP limits is **0** (no restrictions). When a limit is 0 or the key is omitted, no check is performed for that folder. Teams explicitly set a positive limit (e.g. `doing: 1`) only where they want to enforce a cap.

## Acceptance Criteria

- [ ] `kira.yml` can specify `wip_limits` under `validation`  as a map from status name to non-negative integer 0 = no restrictions and is the default if not specified.
- [ ] Invalid keys (non-existent status) or invalid values (negative, non-integer) are rejected with a clear error when loading or validating config.
- [ ] `kira lint` counts work items per configured status folder and reports a workflow error when a folder exceeds its limit.
- [ ] Error message includes status/folder, current count, and configured limit.
- [ ] When a WIP limit is 0 or not configured, `kira lint` does not run a check for that folder (default 0 = no restrictions).
- [ ] Unit tests cover: config parsing, validation logic (under/over limit), and error message content; integration or e2e test runs `kira lint` with a doing folder over limit and asserts failure.

## Implementation Notes

- **Config:** Extend `ValidationConfig` in `internal/config/config.go` with e.g. `WIPLimits map[string]int \`yaml:"wip_limits"\``. Ensure only status keys from `StatusFolders` are allowed; validate in `validateConfig` or a dedicated validator (reject negative values). Default 0 = no restrictions; no need to merge defaults for 0.
- **Validation:** In `internal/validation/validator.go`, implement `validateWorkflowRules(cfg *config.Config)` to:
  1. If `cfg.Validation.WIPLimits` is nil or empty, return nil.
  2. For each status → limit in `cfg.Validation.WIPLimits`, skip if limit is 0 (no restrictions). Otherwise get folder name from `cfg.StatusFolders[status]`, build path under work folder, count `*.md` files (excluding templates and IDEAS.md), and if count > limit, accumulate an error (e.g. one error per status that exceeds).
  3. Return a single error that describes all exceeded limits, or call `result.AddError("workflow", ...)` per limit in the caller so multiple workflow errors can be reported.
- **File counting:** Reuse the same rules as elsewhere: markdown files in the folder, excluding filenames containing "template" and IDEAS.md. Optionally reuse or share with `getWorkItemFiles`-style logic restricted to one folder.
- **Lint output:** Workflow errors are already categorised in `lint.go`; no change needed if errors use `result.AddError("workflow", msg)`.

## Slices

### Config: WIP limits schema and validation
- [ ] T001: Add `WIPLimits map[string]int` to `ValidationConfig` in `internal/config/config.go` with YAML tag `wip_limits`; ensure mergeWithDefaults does not overwrite (0 = no restrictions, no default merge needed).
- [ ] T002: In `validateConfig`, validate `wip_limits`: keys must exist in `StatusFolders`; values must be non-negative integers. Reject with clear error for unknown status or negative value.
- [ ] T003: Add config unit tests: parse valid `wip_limits` (doing: 1, review: 3); reject unknown status key; reject negative value; omit/empty leaves WIPLimits nil or empty.

### Validation: WIP limit check in lint
- [ ] T004: In `internal/validation/validator.go`, implement `validateWorkflowRules(cfg)`: if `cfg.Validation.WIPLimits` is nil or empty return nil; for each status→limit with limit > 0, resolve folder from StatusFolders + work folder, count work item `.md` files (exclude template, IDEAS.md), if count > limit return error describing status/folder, count, and limit.
- [ ] T005: Add validator unit tests: no error when WIPLimits nil/empty; no error when limit 0; no error when count ≤ limit; workflow error when count > limit; error message includes status, count, limit.

### E2E: Lint with WIP limit exceeded
- [ ] T006: Add integration or e2e test: set `validation.wip_limits.doing: 1` in test config, create two work items in doing folder, run `kira lint`, assert failure and that output contains workflow error with "doing", "exceeds WIP limit", and limit 1.

## Release Notes

- **Configurable WIP limits:** You can set `validation.wip_limits` in `kira.yml` (e.g. `doing: 1`, `review: 3`) to enforce maximum work items per status folder. `kira lint` reports a workflow error when a folder exceeds its limit. Default is 0 (no restrictions) for any folder.

