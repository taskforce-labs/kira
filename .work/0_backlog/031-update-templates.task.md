---
id: 031
title: update templates
status: backlog
kind: task
assigned:
created: 2026-01-30
tags: []
---

# update templates

Update template files in the project so they match the canonical defaults defined in code.

## Details

**Source of truth:** The defaults are the template contents returned by `getPRDTemplate()`, `getIssueTemplate()`, `getSpikeTemplate()`, and `getTaskTemplate()` in `internal/templates/templates.go`. These are what `kira init` writes into the work folder via `CreateDefaultTemplates()`.

**Locations to align:**

1. **Root `templates/`** (repo root)  
   Reference copies shipped with the repo. Currently they differ from the code defaults: simpler frontmatter (no inline placeholders for section body text), single `description` placeholder instead of granular placeholders (e.g. `problem`, `step1`, `criteria1`), and different section structure in some templates.

2. **`.work/templates/`** (work folder)  
   User-facing templates used at runtime. These currently include frontmatter fields not in the code defaults (`estimate`, and for PRD `due`), and different `created` label text ("Creation date" vs "Created (auto-set)"). Body structure also differs from code.

**Scope for this task:**

- **In scope:** Update the four files under **root `templates/`** so their content matches exactly what is in `internal/templates/templates.go` (the `get*Template()` strings). That keeps the repo’s reference templates in sync with what init creates.
- **Out of scope (unless we decide otherwise):** Changing the behavior of `CreateDefaultTemplates()` or adding/removing fields (e.g. `estimate`, `due`) in the code defaults; that would be a separate change. The `.work/templates/` in this repo can be refreshed by re-running init with `--force` after root templates are fixed, or left as a one-time manual sync.

**Concrete steps:**

1. For each of `template.prd.md`, `template.issue.md`, `template.spike.md`, `template.task.md` in **`templates/`** (repo root):
   - Replace the file content with the corresponding string from `getPRDTemplate()`, `getIssueTemplate()`, `getSpikeTemplate()`, or `getTaskTemplate()` in `internal/templates/templates.go`.
2. Confirm that `make check` passes and that existing template tests (e.g. in `internal/templates/templates_test.go` and commands that use templates) still pass.
3. Optionally run `kira init --force` in a test copy of the repo and diff `.work/templates/` against root `templates/` to confirm they are identical.

## Acceptance Criteria

- [ ] Root `templates/template.prd.md` matches `getPRDTemplate()` output.
- [ ] Root `templates/template.issue.md` matches `getIssueTemplate()` output.
- [ ] Root `templates/template.spike.md` matches `getSpikeTemplate()` output.
- [ ] Root `templates/template.task.md` matches `getTaskTemplate()` output.
- [ ] `make check` passes (lint, security, unit tests, coverage).
- [ ] E2E tests pass (`bash kira_e2e_tests.sh`).

## Release Notes

- Template files in the repository root (`templates/`) are now aligned with the default templates used by `kira init`.
