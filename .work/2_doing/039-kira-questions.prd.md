---
id: 039
title: kira questions
status: doing
kind: prd
assigned:
created: 2026-02-02
tags: []
---

# kira questions

`kira questions` will search the work and docs locations configured in kira.yml for both answered and unanswered questions. It returns a list of each question and the file that contains it depending on the flags supplied.

- **Scope**: Only content under a `## Questions` heading in `.md` or `.qmd` files under the configured work folder and docs folder.
- **Question shape**: Under `## Questions`, each question is a level-3 heading `### <number>. <question-title-text>` (e.g. `### 1. API shape`). A `#### Options` block under that heading applies only to that question.
- **Unanswered**: A question is unanswered when its subsection has no `#### Options`, or `#### Options` exists but no line is checked (`[x]`). If at least one option is `[x]`, the question is answered and excluded from output.

## Context

- Work items (PRDs, spikes, tasks, issues) and docs often include a `## Questions` section with numbered sub-questions. Open questions block clarity and completion.
- Users and agents need a single command to discover all such open questions across the repo without opening every file.
- Aligns with existing docs/work layout: work folder (e.g. `.work`) and docs folder (e.g. `.docs`) are already in config; `.qmd` is supported for docs (see 025).

## Requirements

- **Command**: `kira questions` (no required arguments).
- **Search paths**: Resolve work folder and docs folder from config (e.g. `config.GetWorkFolderAbsPath`, `config.DocsRoot`); search recursively under both for `.md` and `.qmd` files.
- **Section**: Only consider content under a level-2 heading exactly `## Questions`. Do not treat `## Questions to Answer` or other variants as the Questions section (templates should use `## Questions`).
- **Question detection**: Within that section, each level-3 heading matching `### <number>. <question-title-text>` starts one question; content belongs to that question until the next `###` or `##` (or end of file). Optional body text under the heading is part of the question text for display.
- **Options**: Under each question subsection, a `#### Options` block holds checklist items. List items use `- [ ]` / `- [x]` (see Implementation Notes).
- **Unanswered**: A question is **unanswered** if its subsection has no `#### Options`, or `#### Options` has no checked item (`[x]`). If at least one option in that question’s `#### Options` is checked, the question is **answered** and excluded from output.
- **Output**: By default, human-readable lines: one line per unanswered question with file path and question text (or path and line number). Optional: `--output json` for machine-readable output (array of `{ "file", "question" }` or similar).
- **Errors**: If config is missing or paths invalid, exit non-zero with a clear message; do not search.
- **Filters** (all optional; when omitted, search both work and docs with no stage/doc-type restriction):
  - **Location**: `--work` only search the work folder; `--docs` only search the docs folder. Default: search both.
  - **Work stage**: `--status <values>` restrict work results to files under the given status folder(s) (e.g. `backlog`, `doing`, `done`). Values are from config `status_folders` / `validation.status_values` (e.g. backlog, todo, doing, review, done, released, abandoned, archived). Multiple values: comma-separated or repeatable (e.g. `--status backlog --status doing`). Only applies when search includes work (default or `--work`). Invalid status values exit non-zero with a clear message.
  - **Doc type**: Derived from the basename only (not folder path). **Typed** files: `*.<type>.md` or `*.<type>.qmd`—the segment before `.md`/`.qmd` is the type (e.g. `039-kira-questions.prd.md` → `prd`, `x.adr.md` → `adr`). **Untyped**: basename has a single dot before `.md`/`.qmd` (e.g. `README.md`, `report.qmd`). **`--doc-type <values>`** (comma-separated or repeatable; work and docs): include typed files whose type matches any value. **`--no-doc-type`**: include untyped files. **Combined** (`--doc-type` and `--no-doc-type`): include the **union**—files that match any given type **or** are untyped (e.g. `--doc-type adr --no-doc-type` → `*.adr.md` / `*.adr.qmd` plus untyped files). **`--doc-type` only**: matching typed files only (untyped excluded); no matching typed files → empty output, not an error. **`--no-doc-type` only**: untyped files only. **Neither flag**: typed and untyped.

## Acceptance Criteria

- [ ] `kira questions` exists and runs without required arguments.
- [ ] Search is limited to files under the configured work folder and docs folder (from kira.yml); default work folder `.work` and docs folder `.docs` when not set.
- [ ] Only `.md` and `.qmd` files are scanned.
- [ ] Only content under a level-2 heading exactly `## Questions` is considered; other headings (including `## Questions to Answer`) are ignored.
- [ ] Each `### <number>. <question-title>` under `## Questions` is one question; `#### Options` under that subsection applies only to that question. A question is listed as unanswered only when it has no `#### Options` or no `[x]` in that block; if at least one option is checked, the question is not listed.
- [ ] Output lists each unanswered question with the file path (relative to repo root or config dir) and the question text (or line reference).
- [ ] If config cannot be loaded or work/docs paths are invalid, the command exits non-zero with an explicit error message.
- [ ] Unit tests cover: no `## Questions` section (no output); question subsection with no `#### Options` (listed); `#### Options` all unchecked (listed); at least one `[x]` in `#### Options` (not listed); multiple files and multiple `### N.` questions per file.
- [ ] `--work` limits search to the work folder only; `--docs` limits search to the docs folder only; default (no flag) searches both.
- [ ] `--status <values>` limits work results to files under the given status folder(s); invalid status exits non-zero; multiple values supported (comma or repeatable).
- [ ] `--doc-type` / `--no-doc-type` behave as in Requirements (including union when both are set); with neither flag, typed and untyped are included.
- [ ] `--doc-type` matching (e.g. case-insensitivity) is documented in command help.

## Slices

### Slice 1: Command and config
- [x] T001: Add `questions` subcommand to root; load config and resolve work folder and docs folder; exit with error if not in a kira workspace or paths invalid.
- [x] T002: Implement recursive file discovery for `.md` and `.qmd` under work and docs roots; skip non-files and unsupported extensions.

### Slice 2: Parse Questions and Options
- [x] T003: Parse each file for a `## Questions` section only; within it, treat each `### <number>. <title>` heading as one question until the next `###` or `##`.
- [x] T004: Under each question subsection, detect `#### Options`; parse checklist items and treat the question as answered if at least one `[x]` is present.

### Slice 3: Filter and output
- [x] T005: Filter to unanswered questions only; output human-readable lines (file path + question text).
- [x] T006: Add `--output json` for machine-readable array of `{ "file", "question" }` (or equivalent).

### Slice 4: Location and scope filters
- [ ] T007: Add `--work` and `--docs` flags; when set, restrict file discovery to work folder only or docs folder only; when neither set, search both.
- [ ] T008: Add `--status <values>`; resolve status folders from config; restrict work results to files under those folders; support multiple values (comma or repeatable); invalid status exit non-zero.
- [ ] T009: Add `--doc-type <values>`; derive doc type from each scanned file basename (`*.<type>.md` / `*.<type>.qmd`); apply to work and docs; support multiple values; combinable with `--no-doc-type` (union).
- [ ] T010: Add `--no-doc-type` for untyped files; implement union when both `--doc-type` and `--no-doc-type` are set; default when neither flag: include typed and untyped files.

## Implementation Notes

- **Heading names**: Recognize **only** `## Questions` as the L2 section to scan. Legacy templates that used `## Questions to Answer` should be updated to `## Questions` over time.
- **Question / options layout**: Align with the clarifying-questions pattern: `### N. Short title` per question, then optional context, then `#### Options` with `- [ ]` / `- [x]` lines. Document this in command help.
- **Checkbox format**: Match `[x]` (and optionally `[X]`) as checked; `[ ]` as unchecked. Keep parsing simple (no need to support every possible markdown variant initially).
- **Path resolution**: Reuse `config.GetWorkFolderAbsPath(cfg)` and `config.DocsRoot(cfg, cfg.ConfigDir)` (or equivalent); same validation as other commands (e.g. no `..` escape).
- **Performance**: For large trees, consider limiting depth or file count if needed; not required for MVP.
- **Work stage**: Derive stage from path: a file is “in” a status if it lives under `work_folder/<status_folder>/` (e.g. `.work/0_backlog/`, `.work/2_doing/`). Use `config.StatusFolders` and `config.Validation.StatusValues` for valid status values. If `--status X` and X is not a key in status_folders (or not in status_values), exit non-zero.
- **Doc type**: Match `--doc-type` to the derived type case-insensitively (state in help). **Only `--doc-type`**: matching typed files, untyped excluded. **Only `--no-doc-type`**: untyped only. **Both**: union of matching typed files and untyped files. **Neither**: all files. Folder path does not define type; which roots are scanned follows `--work` / `--docs`.

## Release Notes

- **Added** `kira questions` to list unanswered questions from work and docs: scans `.md` and `.qmd` under the configured work and docs folders; under `## Questions`, each `### N. Title` is one question, answered only if its `#### Options` block contains at least one `[x]`. Optional filters: `--work` / `--docs`, `--status`, `--doc-type` / `--no-doc-type` (filename-derived type; see Requirements).

