---
id: 036
title: adrs
status: backlog
kind: prd
assigned:
created: 2026-02-02
tags: [documentation, adrs]
---

# adrs

Kira commands that create or update Architecture Decision Records (ADRs) SHALL use the `kira adr` command. This PRD defines the `kira adr` subcommand so ADRs are created and updated in a consistent location, format, and numbering scheme—using the configured docs folder and `architecture/` subfolder as defined in PRD 025 (docs folder).

## Context

### Prior Art / Established ADR CLIs

There are established CLI tools for ADRs that Kira SHOULD align with so users get a familiar experience and conventions are consistent across the ecosystem.

- **adr-tools (npryce)** — [github.com/npryce/adr-tools](https://github.com/npryce/adr-tools) — Shell scripts, ~5k stars, widely used. Commands: `adr init <path>` (create ADR dir, default `doc/adr`); `adr new <title>` (create numbered ADR, open in `$EDITOR`); `adr new -s <n> <title>` (create ADR that **supersedes** ADR n, mark old as superseded). Numbered Markdown files; opens new ADR in editor after create.
- **corani/adr** — [github.com/corani/adr](https://github.com/corani/adr) — Go-based. Commands: `adr init [path]`, `adr new <title>`, `adr show <id>`, `adr edit <id>`, `adr list`, `adr update <id> <status>` (status: `proposed`, `accepted`, `deprecated`, `superseded`). Uses YAML front matter and template.

**Alignment:** Kira’s `kira adr` SHOULD borrow from these conventions where practical: subcommand names (`init`, `new`, `list`, `update`); `new -s <id> <title>` (or equivalent) for superseding an ADR and marking the old one superseded; numbered filenames; optional “open in `$EDITOR`” after create/edit. This allows users familiar with adr-tools or corani/adr to use `kira adr` without relearning. Kira’s path is config-driven (`docs_folder/architecture/`) rather than adr-tools’ default `doc/adr`, but the mental model (init path, new with title, supersede flag) remains the same.

### Problem Statement

Teams need a standard way to capture and maintain Architecture Decision Records (ADRs): why decisions were made, what the decision was, and what the consequences are. PRD 025 defines a docs folder with an `architecture/` subfolder and naming convention (`<id>-<name>.adr.[md|qmd]`), but there is no Kira command to create or update ADRs. Without a dedicated command:

- ADRs may be created ad hoc in inconsistent locations or formats.
- Numbering and naming are not standardized.
- Any future Kira feature or agent that produces ADRs would need to duplicate logic for path resolution, template application, and file naming.

### Proposed Solution

- Introduce a **`kira adr`** subcommand that creates, lists, updates, and migrates ADRs.
- **Create:** `kira adr new <title>` (or equivalent) creates a new ADR file under the configured docs folder’s `architecture/` subfolder, using the next available numeric ID and a kebab-case slug from the title. Content is generated from a template (e.g. context, decision, consequences). **List**, **update**, and **migrate** actions are also in scope (see Scope and FRs).
- **Placement:** All ADR files SHALL be written under `docs_folder/architecture/` (e.g. `.docs/architecture/`), resolved from `kira.yml` as in PRD 025. No hardcoded docs path.
- **Template:** ADR content SHALL be generated from a template. The template may live in the docs folder (e.g. `architecture/template.adr.md`) or in `.docs/templates/`; if missing, use a built-in default that follows the standard ADR structure (context, decision, consequences).
- **Naming:** File names SHALL follow the pattern `<id>-<slug>.adr.md` (or `.adr.qmd` if configured), where `<id>` is a zero-padded number (e.g. 001, 002) and `<slug>` is kebab-case derived from the title.

### Scope

- **In scope:** `kira adr` subcommand with create (e.g. `kira adr new <title>`), **list** (e.g. `kira adr list`), and **update** (e.g. `kira adr update`); resolution of docs path and `architecture/` from config; template-based content generation; deterministic ID assignment (next number); kebab-case slug from title; standard ADR sections (context, decision, consequences); **Quarto-specific workflows** (e.g. `.adr.qmd` creation and config); **editing existing ADR content from the CLI** (update command or equivalent); **migration of existing ad-hoc ADR files** into the new structure (e.g. move/rename into `architecture/` with standard naming).
- **Out of scope:** None; all of the above are in scope for this PRD.

## Requirements

### Functional Requirements

#### FR1: Command and Subcommands

- A **`kira adr`** top-level subcommand SHALL exist.
- The following actions SHALL be supported: **create** (e.g. `kira adr new <title>`), **list** (e.g. `kira adr list`), **update** (e.g. `kira adr update <id|path>`), and **migrate** (e.g. `kira adr migrate [paths]`). Exact subcommand names are an implementation choice.
- Any future Kira code or agent that creates or updates ADRs SHALL use `kira adr` (or the same underlying logic) so there is a single place for path resolution, ID assignment, and templating.

#### FR2: Placement and Configuration

- ADR files SHALL be written under the **configured docs folder** (e.g. `docs_folder` in `kira.yml`), in the **`architecture/`** subfolder. Path SHALL be resolved via config (e.g. `filepath.Join(workDir, cfg.DocsFolder, "architecture")`); no hardcoded `docs/` or `.docs/` in code.
- If the docs folder or `architecture/` does not exist, the command MAY create it (consistent with PRD 025) or fail with a clear message (e.g. “run kira init first” or “docs folder missing”). Recommendation: create `architecture/` if docs folder exists; if docs folder is missing, direct user to run `kira init` or set `docs_folder`.

#### FR3: File Naming and ID Assignment

- File name pattern: **`<id>-<slug>.adr.md`** (or `.adr.qmd` if the project supports Quarto ADRs). Example: `001-use-plaintext-work-items.adr.md`.
- **`<id>`:** Zero-padded numeric ID (e.g. 001, 002). The next ID SHALL be determined by scanning existing files in `architecture/` matching the pattern `*-*.adr.md` (and optionally `*.adr.qmd`), parsing the leading number, and choosing max(existing) + 1. If no ADRs exist, start at 001.
- **`<slug>`:** Derived from the user-provided title: kebab-case, lowercase, non-alphanumeric characters removed or replaced by hyphens, no leading/trailing hyphens. Example: “Use plaintext work items” → `use-plaintext-work-items`.

#### FR4: Template and Content

- New ADR content SHALL be generated from a **template**. Template resolution order (recommendation): (1) project template in docs folder, e.g. `docs_folder/architecture/template.adr.md`; (2) project template in `.work/templates/`, e.g. `template.adr.md`; (3) built-in default template.
- The template SHALL support at least: **title** (or equivalent), **date** (e.g. YYYY-MM-DD), and **id** (the chosen ADR number). Standard ADR sections: **Context**, **Decision**, **Consequences** (or equivalent). Placeholders (e.g. `{{.Title}}`, `{{.Date}}`, `{{.ID}}`) are implementation-defined.
- The generated file SHALL be valid Markdown (or Quarto) and ready for the user to edit.

#### FR5: Integration with Docs Folder (PRD 025)

- This feature depends on PRD 025 (docs folder): `docs_folder` in `kira.yml` and the `architecture/` subfolder. If PRD 025 is not yet implemented, `kira adr` MAY assume a default docs path (e.g. `.docs`) and create `architecture/` when creating the first ADR, so long as the eventual implementation aligns with 025 (config-driven path, no hardcoded docs root).

#### FR6: List ADRs

- **`kira adr list`** (or equivalent) SHALL list ADRs in the configured `architecture/` folder. Output SHALL include at least: ADR id, title (parsed from file or filename), and path. Format (table, one-per-line, JSON) is implementation-defined; default SHOULD be human-readable (e.g. id, title, filename).

#### FR7: Update ADR and Supersede

- **`kira adr update <id|path> [options]`** (or equivalent) SHALL support updating an existing ADR: e.g. set status to one of `proposed`, `accepted`, `deprecated`, `superseded` (align with corani/adr and common ADR practice); append a section; or edit specific fields (title, date). The command SHALL identify the ADR by id (e.g. 001), path, or title slug and apply updates without overwriting unrelated content.
- **Supersede (align with adr-tools):** **`kira adr new -s <id> <title>`** (or `--supersedes <id>`) SHALL create a new ADR that supersedes the given ADR: the new ADR is created with a reference to the superseded one, and the superseded ADR’s status SHALL be set to `superseded` (and optionally link to the new ADR). This matches adr-tools’ `adr new -s 9 <title>` behaviour.

#### FR8: Quarto Workflows

- **Quarto support:** When creating or updating ADRs, Kira SHALL support **`.adr.qmd`** as well as `.adr.md`. Configuration (e.g. `adr_extension: .adr.qmd` in `kira.yml` or infer from project template extension) SHALL determine which extension to use. Template resolution and content generation SHALL work for both `.md` and `.qmd`; Quarto-specific content (e.g. executable chunks) MAY be included in the built-in or project template for `.qmd`.

#### FR9: Migration of Existing ADR Files

- **`kira adr migrate`** (or equivalent) SHALL support migrating existing ad-hoc ADR files into the new structure. Input: path(s) to existing ADR files (or a directory to scan). Behaviour: copy or move files into `docs_folder/architecture/`, assign next available id if missing, rename to `<id>-<slug>.adr.[md|qmd]` using existing title or filename for slug. Conflicting ids or filenames SHALL be handled (e.g. renumber, or prompt user). Option to dry-run (list planned renames) is recommended.

### Configuration

- **Docs path:** Taken from `kira.yml` `docs_folder` (see PRD 025). No separate `adr_folder` required; ADRs always live under `docs_folder/architecture/`.
- **Optional:** `kira.yml` MAY support an optional key (e.g. `adr_template`) to override the template path (e.g. `.work/templates/my-adr.md`). If not specified, use the resolution order in FR4.

### Defaults

- Default extension: `.adr.md`. Optional support for `.adr.qmd` can be configurable or inferred from an existing template extension.
- Default template: built-in minimal ADR (title, date, Context, Decision, Consequences) when no project template is found.
- **Open in editor (align with adr-tools / corani/adr):** After `kira adr new` (and optionally after `kira adr update` or `kira adr edit`), Kira MAY open the ADR file in the user’s editor (from `$EDITOR` or `$VISUAL`). This is optional but recommended so behaviour matches adr-tools (“opens it in your editor of choice”) and corani/adr (“open it in your $EDITOR”).

## Acceptance Criteria

- [ ] `kira adr` subcommand exists and is registered under the root command.
- [ ] A create action (e.g. `kira adr new <title>`) creates a new ADR file under `docs_folder/architecture/` (path resolved from config; no hardcoded docs path).
- [ ] File name follows `<id>-<slug>.adr.md` (or `.adr.qmd` when configured); ID is the next available number (001, 002, …) based on existing ADR files in `architecture/`; slug is kebab-case from the title.
- [ ] ADR content is generated from a template (project or built-in); at least title, date, and id are filled; standard sections (Context, Decision, Consequences) are present.
- [ ] If docs folder or `architecture/` is missing, the command either creates `architecture/` under the configured docs path or fails with a clear message (e.g. run `kira init` or set `docs_folder`).
- [ ] **List:** `kira adr list` lists ADRs in `architecture/` with at least id, title (or filename), and path; output is human-readable by default.
- [ ] **Update:** `kira adr update <id|path>` (or equivalent) updates an existing ADR (e.g. status: proposed/accepted/deprecated/superseded, append section, or edit fields) without overwriting unrelated content.
- [ ] **Supersede:** `kira adr new -s <id> <title>` (or equivalent) creates a new ADR that supersedes the given ADR and sets the old ADR’s status to superseded (and optionally links to the new one).
- [ ] **Editor (optional):** After `kira adr new` (and optionally after edit/update), the ADR file may be opened in `$EDITOR`/`$VISUAL` for parity with adr-tools and corani/adr.
- [ ] **Quarto:** Creating or updating ADRs supports `.adr.qmd` when configured; template and naming work for both `.md` and `.qmd`.
- [ ] **Migration:** `kira adr migrate` (or equivalent) moves/renames existing ad-hoc ADR files into `docs_folder/architecture/` with standard `<id>-<slug>.adr.[md|qmd]` naming; conflicts are handled (e.g. renumber or prompt); dry-run option recommended.
- [ ] Unit tests cover: ID selection (next number), slug generation from title, path resolution from config, template application, list output, update behaviour, migration renames.
- [ ] Integration or e2e test: run `kira adr new "Some Decision"` and assert file exists under docs_folder/architecture/ with correct name and structure; run `kira adr list` and assert ADR appears; run `kira adr migrate` with a test file and assert it is placed under architecture/ with standard name.

## Implementation Notes

- **Config:** Use existing `DocsFolder` from `internal/config/config.go` (PRD 025). Optionally add `AdrTemplate string` and `AdrExtension string` (e.g. `".adr.md"` or `".adr.qmd"`) for custom template path and Quarto. Resolve docs root with `config.GetDocsFolderPath(cfg)` or equivalent (must be implemented for 025; reuse here).
- **Command:** Add `adrCmd` in `internal/commands/` (e.g. `adr.go`) with subcommands: `new` (or `create`), `list`, `update`, `migrate`. Optionally add `init` for parity with adr-tools (ensure `docs_folder/architecture/` and template exist; may be no-op if `kira init` already created them). Register `adrCmd` in `root.go`.
- **Path:** ADR directory = `filepath.Join(workDir, cfg.DocsFolder, "architecture")`. Ensure path validation (no `..`, safe segment length) per project security guidelines.
- **ID:** List files in ADR directory matching `[0-9]*-*.adr.md` and `*.adr.qmd`; parse leading integer; next ID = max + 1; format as zero-padded 3 digits (e.g. 001). If directory is empty or missing, start at 001.
- **Slug:** From title: lowercase, replace non-alphanumeric with `-`, collapse multiple hyphens, trim leading/trailing hyphens. Handle empty slug (e.g. fallback to `adr`).
- **Template:** Use `internal/templates` or a small adr-specific helper. Built-in default: YAML front matter (optional) + title, date, ## Context, ## Decision, ## Consequences. If project template exists at `docs_folder/architecture/template.adr.md` or `template.adr.qmd`, use it with same placeholders. Support both `.md` and `.qmd` templates when Quarto is configured.
- **Writing:** Create the file with safe permissions (e.g. 0644); do not overwrite existing file (if `<id>-<slug>.adr.md` already exists, either increment ID or fail with clear message).
- **List:** Scan `architecture/` for `*.adr.md` and `*.adr.qmd`; parse id and title (from filename or front matter); output table or lines (id, title, path). Reuse path resolution and file glob.
- **Update:** Resolve ADR by id (e.g. `001`) or path; read file; apply updates (e.g. set status in front matter to proposed/accepted/deprecated/superseded, append section); write back with safe permissions. Preserve existing content; only modify specified fields or sections.
- **Supersede:** For `kira adr new -s <id> <title>`: create new ADR as usual; in new ADR front matter or body, add reference to superseded ADR (e.g. “Supersedes: 001” or link); load superseded ADR by id, set its status to `superseded` and optionally add “Superseded by: 002” (or link); save both files. Reuse adr-tools’ mental model.
- **Editor:** After creating (or editing) an ADR, exec user’s `os.LookupEnv("EDITOR")` or `VISUAL` with the file path; non-blocking or blocking is implementation choice (adr-tools blocks until editor closes). Validate editor path per project security guidelines (no shell injection).
- **Quarto:** When `adr_extension` or template is `.qmd`, create/update `.adr.qmd` files; built-in or project template may include Quarto chunks for `.qmd`. Document in README or docs how to render (e.g. `quarto render`).
- **Migration:** Accept file paths or directory; for each file, infer or prompt for id and slug; determine target path `architecture/<id>-<slug>.adr.<ext>`; copy or move; if id collision, renumber or prompt. Support `--dry-run` to print planned moves. Validate paths (no `..`, within repo).

## Release Notes

- **`kira adr` command:** Create, list, update, and migrate Architecture Decision Records, aligned with established CLIs (adr-tools, corani/adr). **Create:** `kira adr new <title>` creates a new ADR under the configured docs folder’s `architecture/` subfolder with automatic numbering (001, 002, …) and kebab-case filenames; content from a project or built-in template (Context, Decision, Consequences). **Supersede:** `kira adr new -s <id> <title>` creates an ADR that supersedes the given one and marks the old one superseded (like adr-tools). **List:** `kira adr list` lists ADRs with id, title, and path. **Update:** `kira adr update <id|path>` updates an existing ADR (e.g. status: proposed/accepted/deprecated/superseded, append section). **Migrate:** `kira adr migrate` moves ad-hoc ADR files into `architecture/` with standard naming. **Quarto:** Support for `.adr.qmd` when configured. Optional: open new/edited ADR in `$EDITOR`/`$VISUAL` for parity with adr-tools and corani/adr. Any Kira feature that creates or updates ADRs uses this command or the same logic for consistent placement and format.
