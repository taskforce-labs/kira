---
id: 039
title: kira questions
status: backlog
kind: prd
assigned:
created: 2026-02-02
tags: []
---

# kira questions

`kira questions` will search the work and docs locations configured in kira.yml for **unanswered questions**. It returns a list of each question and the file that contains it.

- **Scope**: Only content under a `## Questions` heading in `.md` or `.qmd` files under the configured work folder and docs folder.
- **Unanswered**: A question is unanswered when, within that Questions section, there is no `### Options` subsection that contains at least one checked option (`[x]`). If there are Options and at least one is `[x]`, the question is considered answered.

## Context

- Work items (PRDs, spikes, tasks) and docs often include a "Questions to Answer" or "Questions" section. Open questions block clarity and completion.
- Users and agents need a single command to discover all such open questions across the repo without opening every file.
- Aligns with existing docs/work layout: work folder (e.g. `.work`) and docs folder (e.g. `.docs`) are already in config; `.qmd` is supported for docs (see 025).

## Requirements

- **Command**: `kira questions` (no required arguments).
- **Search paths**: Resolve work folder and docs folder from config (e.g. `config.GetWorkFolderAbsPath`, `config.DocsRoot`); search recursively under both for `.md` and `.qmd` files.
- **Section**: Only consider content under a level-2 heading exactly `## Questions` (or `## Questions to Answer` — define one canonical name or both; see Implementation Notes). Content before the next same-or-higher-level heading belongs to that section.
- **Question detection**: Within that section, treat each block of text (e.g. paragraph or list item) that looks like a question (e.g. ends with `?` or matches a simple pattern) as one question. Optional: treat each list item under a `### Options` as an option.
- **Unanswered**: A question is **unanswered** if, within the same `## Questions` section, there is no `### Options` subsection containing at least one checked item (`[x]` or `[x]`). If `### Options` exists and at least one option is checked, the question is **answered** and excluded from output.
- **Output**: By default, human-readable lines: one line per unanswered question with file path and question text (or path and line number). Optional: `--output json` for machine-readable output (array of `{ "file", "question" }` or similar).
- **Errors**: If config is missing or paths invalid, exit non-zero with a clear message; do not search.
- **Filters** (all optional; when omitted, search both work and docs with no stage/doc-type restriction):
  - **Location**: `--work` only search the work folder; `--docs` only search the docs folder. Default: search both.
  - **Work stage**: `--status <values>` restrict work results to files under the given status folder(s) (e.g. `backlog`, `doing`, `done`). Values are from config `status_folders` / `validation.status_values` (e.g. backlog, todo, doing, review, done, released, abandoned, archived). Multiple values: comma-separated or repeatable (e.g. `--status backlog --status doing`). Only applies when search includes work (default or `--work`). Invalid status values exit non-zero with a clear message.
  - **Docs kind**: `--doc-type <values>` restrict docs results to files under the given docs subfolder(s) (e.g. `agents`, `guides`, `architecture`, `product`, `reports`, `api`). Multiple values: comma-separated or repeatable. Only applies when search includes docs (default or `--docs`). Unknown doc-type is treated as “no match” (empty result for that type) or validated against a known list and exit non-zero; define in Implementation Notes.

## Acceptance Criteria

- [ ] `kira questions` exists and runs without required arguments.
- [ ] Search is limited to files under the configured work folder and docs folder (from kira.yml); default work folder `.work` and docs folder `.docs` when not set.
- [ ] Only `.md` and `.qmd` files are scanned.
- [ ] Only content under a `## Questions` (or agreed canonical) heading is considered; questions outside that heading are ignored.
- [ ] A question is reported as unanswered only when there is no `### Options` with at least one `[x]` in that same Questions section; if such an Options block exists with one or more `[x]`, the question is not listed.
- [ ] Output lists each unanswered question with the file path (relative to repo root or config dir) and the question text (or line reference).
- [ ] If config cannot be loaded or work/docs paths are invalid, the command exits non-zero with an explicit error message.
- [ ] Unit tests cover: no Questions section (no output); Questions with no Options (question listed); Questions with Options all unchecked (question listed); Questions with at least one Option checked (question not listed); multiple files and multiple questions per file.
- [ ] `--work` limits search to the work folder only; `--docs` limits search to the docs folder only; default (no flag) searches both.
- [ ] `--status <values>` limits work results to files under the given status folder(s); invalid status exits non-zero; multiple values supported (comma or repeatable).
- [ ] `--doc-type <values>` limits docs results to files under the given docs subfolder(s); multiple values supported; behavior for unknown doc-type is defined (validate and exit non-zero, or treat as no match).

## Slices

### Slice 1: Command and config
- [ ] T001: Add `questions` subcommand to root; load config and resolve work folder and docs folder; exit with error if not in a kira workspace or paths invalid.
- [ ] T002: Implement recursive file discovery for `.md` and `.qmd` under work and docs roots; skip non-files and unsupported extensions.

### Slice 2: Parse Questions and Options
- [ ] T003: Parse each file for `## Questions` (and optionally `## Questions to Answer`) section; extract content until next `##` or `#`; identify discrete questions (e.g. lines ending with `?` or list items).
- [ ] T004: Within each Questions section, detect `### Options`; parse checklist items and treat question as answered if at least one `[x]` is present.

### Slice 3: Filter and output
- [ ] T005: Filter to unanswered questions only; output human-readable lines (file path + question text).
- [ ] T006: Add `--output json` for machine-readable array of `{ "file", "question" }` (or equivalent).

### Slice 4: Location and scope filters
- [ ] T007: Add `--work` and `--docs` flags; when set, restrict file discovery to work folder only or docs folder only; when neither set, search both.
- [ ] T008: Add `--status <values>`; resolve status folders from config; restrict work results to files under those folders; support multiple values (comma or repeatable); invalid status exit non-zero.
- [ ] T009: Add `--doc-type <values>`; restrict docs results to files under docs subfolders matching the given names (e.g. agents, guides, architecture); support multiple values; define validation for unknown doc-type (fail vs no match).

## Implementation Notes

- **Heading names**: Decide whether to support both `## Questions` and `## Questions to Answer` (e.g. match either) or only `## Questions`; document in help or docs. Existing templates use "Questions to Answer" (e.g. `template.spike.md`); consider matching both for compatibility.
- **Checkbox format**: Match `[x]` (and optionally `[X]`) as checked; `[ ]` as unchecked. Keep parsing simple (no need to support every possible markdown variant initially).
- **Path resolution**: Reuse `config.GetWorkFolderAbsPath(cfg)` and `config.DocsRoot(cfg, cfg.ConfigDir)` (or equivalent); same validation as other commands (e.g. no `..` escape).
- **Performance**: For large trees, consider limiting depth or file count if needed; not required for MVP.
- **Work stage**: Derive stage from path: a file is “in” a status if it lives under `work_folder/<status_folder>/` (e.g. `.work/0_backlog/`, `.work/2_doing/`). Use `config.StatusFolders` and `config.Validation.StatusValues` for valid status values. If `--status X` and X is not a key in status_folders (or not in status_values), exit non-zero.
- **Docs kind**: Derive from first path segment under docs folder (e.g. `.docs/agents/`, `.docs/guides/security/` → kind `guides`). Allow any subfolder name for `--doc-type` (no strict allowlist) so new doc folders are included; or allowlist from a known set (agents, architecture, product, reports, guides, api) and invalid value exits non-zero. Recommend: allowlist and validate for clearer UX.

## Release Notes

- **Added** `kira questions` to list unanswered questions from work and docs: scans `.md` and `.qmd` under the configured work and docs folders, reports questions under `## Questions` that have no `### Options` with a checked `[x]`. Optional filters: `--work` / `--docs` (location), `--status` (work stage), `--doc-type` (docs subfolder).

