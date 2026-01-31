---
id: 025
title: docs folder
status: todo
kind: prd
assigned:
estimate: 0
created: 2026-01-27
due: 2026-01-27
tags: []
---

# docs folder

A folder for storing long lived project docs like product definitions, architecture diagrams, ADRs amongst many other useful information. The docs path is **configurable** in `kira.yml`, created by **`kira init`**, and used by kira when **placing documentation artifacts** produced by processes.

## Context
Future features will use docs as inputs and outputs to complete specific tasks like agent guided product discovery, agent guided domain discovery, agent guided technical discovery, agent guided roadmap planning, agent guided work item elaboration, and agent guided RALF on work items.

To make things easier to manage and discover for humans and agents, the docs folder needs a structured place to store:
- Architecture Decision Records (ADRs)
- Product definitions and specifications
- Architecture diagrams and system designs
- Development guides and best practices
- Agent-specific documentation (as referenced in PRD 005)
- Security guidelines (under the docs folder, e.g. `.docs/guides/security/`, migrated from existing `docs/security/`)
- API documentation
- Integration guides
- Other reference materials that don't belong in README.md or AGENTS.md

The docs folder should serve as the canonical location for all project documentation that:
- Is not ephemeral (unlike work items in `.work/`)
- Needs to be version controlled alongside the code
- Should be discoverable and organized by topic
- May be referenced by both humans and agents
- Is the **target for documentation artifacts** when kira or other processes generate docs (e.g. from work items, release notes, ADRs). Kira config must know this path so artifacts are written to the right place.

**Default name: `.docs`** (like `.work`) so it appears at the top of the file tree. The path is **configurable** in `kira.yml` so users can use `docs`, `.docs`, or a custom path.

Documentation may use **Markdown (`.md`)** or **Quarto (`.qmd`)** so that when richer execution is desired, docs can embed code, run analyses, and be rendered like Jupyter notebooks (see [Quarto](https://quarto.org)). This supports capturing and filtering analysis directly in the docs (e.g. in reports or guides) without leaving the docs tree.

This aligns with PRD 005 which references creating docs under the configured docs path (e.g. `.docs/agents/using-kira.md`) for comprehensive agent-focused documentation.

## Requirements

### Configuration (`kira.yml`)

The docs folder path must be **configurable** so kira and processes know where to put documentation artifacts.

- **Key**: `docs_folder` (or equivalent) in `kira.yml`
- **Default**: `.docs` — so it appears at the top of the file tree like `.work`
- **Configurable**: Users may set `docs_folder: docs` or `docs_folder: .docs` or any relative path (e.g. `documentation`)
- **Usage**: All kira commands and processes that create or reference documentation artifacts must read this path from config (e.g. when generating ADRs, agent docs, release summaries, or other doc artifacts)

Example:

```yaml
# kira.yml
docs_folder: .docs   # default; use .docs or docs or custom path
```

### Init command

The docs folder must be created as part of **`kira init`**, analogous to `.work`.

- **When**: `kira init` creates the docs folder and its standard subfolders (see Folder Structure) in the target directory, using the configured `docs_folder` path (or default `.docs` if not set).
- **Flags**: Same behavior as `.work` for `--force` and `--fill-missing`: if the docs folder already exists, user can cancel, overwrite, or fill-missing (add missing subfolders/files only).
- **Fill-missing**: If `--fill-missing` is used and the docs folder exists, add any missing subfolders or index files; do not remove existing content.
- **Config first**: If `kira.yml` already exists in the target dir, init must use its `docs_folder` value when creating the docs tree; otherwise use default `.docs`.

### Artifact placement

When kira or other processes **create documentation artifacts** (e.g. from work items, release notes, ADRs, agent docs), they must use the **configured docs path** from `kira.yml`. No hardcoded `docs/` or `.docs/` in code; always resolve from config so artifact output goes to the right place.

### Folder Structure

The docs folder (default `.docs`) should support a hierarchical organization. All paths below are relative to the configured `docs_folder`.

```
.docs/                     # or docs/ or custom path from kira.yml
├── agents/                 # Agent-specific documentation
│   └── using-kira.md      # Referenced in PRD 005
├── architecture/          # Architecture Decision Records and diagrams
│   ├── README.md          # Index of ADRs
│   └── adr-*.md           # Individual ADR files
├── product/               # Product-related documents
│   ├── README.md          # Index of product docs
│   ├── personas.md        # User/persona definitions
│   ├── vision.md          # Product vision and strategy
│   ├── roadmap.md         # High-level product roadmap
│   ├── glossary.md        # Product/domain terminology
│   ├── commercials/       # (optional) Commercial strategy, pricing, GTM
│   └── features/          # (optional) Feature briefs or one-pagers
├── reports/               # Reports (release, metrics, audits)
│   ├── README.md          # Index of reports
│   └── *.md, *.qmd        # e.g. release reports; Quarto for runnable analysis
├── api/                   # API documentation
│   └── README.md          # API reference
├── guides/                # Development and usage guides
│   ├── README.md          # Index of guides
│   └── security/          # Security guidelines (e.g. secure coding)
│       └── golang-secure-coding.md
└── README.md              # Main docs index
```

### Documentation Standards

1. **File Naming**: Use kebab-case for file names (e.g., `using-kira.md`, `golang-secure-coding.md`)
2. **Index Files**: Each major subdirectory should have a `README.md` that serves as an index
3. **Main Index**: The root of the docs folder (e.g. `.docs/README.md`) should provide an overview and navigation to all documentation
4. **Document Formats**: Documentation may use:
   - **Markdown (`.md`)**: Standard markdown; universally readable and editable.
   - **Quarto (`.qmd`)**: [Quarto](https://quarto.org) format for executable, notebook-like docs. Use when a doc should run code (e.g. Python, R), render outputs (tables, plots), or support a Jupyter-like experience for analysis. Rendered outputs (HTML, PDF, etc.) and Quarto cache (e.g. `_freeze/`) may be gitignored or committed per project choice.
5. **Cross-References**: Documentation should link to related docs and to relevant work items when appropriate

### Key product documents (docs folder `/product/`)

Product-related documents that may live under the docs folder’s `product/` subfolder (e.g. `.docs/product/`) include:

- **Personas** – User/persona definitions (e.g. `personas.md`), roles, goals, and contexts
- **Product vision / strategy** – Vision, goals, and strategic direction (e.g. `vision.md`, `strategy.md`)
- **Roadmap** – High-level product roadmap (outcomes, themes, or timeline), distinct from backlog/PRDs
- **Glossary** – Product and domain terminology so humans and agents share the same language
- **Feature briefs / one-pagers** – Short specs for a feature area (e.g. under `docs/product/features/`)
- **Problem statements / opportunity briefs** – Problem framing, opportunity, and success criteria
- **User research summaries** – Synthesised findings, jobs-to-be-done, or key quotes
- **Success metrics / OKRs** – Product-level metrics, objectives, and key results
- **Release summaries** – Product-level release narrative and highlights (not replace PR/release notes)
- **Commercials** – Commercial strategy, pricing, business model, go-to-market, and packaging (e.g. under `docs/product/commercials/` or `docs/product/commercial-strategy.md`). Keeps product and commercial context in one place.

Not all of these are required; create only what the project needs. Each should be linked from the docs folder’s `product/README.md` (e.g. `.docs/product/README.md`).

### Reports (docs folder `/reports/`)

A `reports/` subfolder under the docs folder (e.g. `.docs/reports/`) holds periodic or one-off reports that are worth keeping in the repo:

- **Release reports** – Post-release summaries, rollout status, or launch reports
- **Metrics / health reports** – Usage, performance, or product metrics summaries
- **Audit or review reports** – Security, compliance, or quality review write-ups
- **Retrospectives** – Project or release retrospectives stored as markdown

Reports can be dated (e.g. `2026-01-release-report.md`) or named by theme. The docs folder’s `reports/README.md` (e.g. `.docs/reports/README.md`) should index available reports. Reports are a natural place for **Quarto (`.qmd`)** when they include runnable analysis (metrics, filtering, plots).

### Quarto and executable docs

When richer rendering or in-doc execution is desired, use **Quarto** (see [quarto.org](https://quarto.org)):

- **Formats**: `.qmd` (Quarto’s standard). Same syntax: YAML front matter, fenced code blocks, optional execution.
- **Use cases**: Reports with embedded analysis; guides with runnable examples; docs that need tables/plots generated from code; Jupyter-notebook-style narratives (run, filter, re-render).
- **Tooling**: Render with `quarto render` (CLI) or via IDE/CI. Use `freeze` for expensive runs so re-renders reuse cached results.
- **Location**: Any docs-folder subfolder (e.g. `.docs/reports/`, `.docs/guides/`) may mix `.md` and `.qmd`. Index files (README.md) should list both; readers can open source or rendered output as needed.
- **Generated artifacts**: Rendered output (HTML, PDF) and cache dirs (e.g. `_freeze/`) are optional in git; document in `.gitignore` or commit policy in the docs folder’s README (e.g. `.docs/README.md`).

### Content Guidelines

The docs folder (configured via `docs_folder` in `kira.yml`, default `.docs`) should contain:
- **Long-lived content**: Information that remains relevant over time
- **Reference material**: Guides, specifications, and decision records
- **Organized by topic**: Grouped into logical subdirectories
- **Version controlled**: All docs committed to git alongside code

The docs folder should NOT contain:
- Work items (those belong in `.work/`)
- Ephemeral notes or temporary documentation
- Generated documentation (unless it's the source of truth; Quarto rendered output may be committed or gitignored per project)
- External dependencies or binaries (Quarto and language runtimes are environment tooling, not stored in docs)

### Integration with Existing Documentation

- **README.md**: Should link to the docs folder’s README (e.g. `.docs/README.md`) for detailed documentation
- **AGENTS.md**: Should reference the docs folder’s `agents/` subfolder (e.g. `.docs/agents/`) for comprehensive agent docs
- **CONTRIBUTING.md**: Should reference relevant guides in the docs folder’s `guides/` (e.g. `.docs/guides/`)
- **Work Items**: PRDs and other work items can reference docs in the configured docs path for context

## Acceptance Criteria

### Configuration
- [ ] `kira.yml` supports a `docs_folder` (or equivalent) key
- [ ] Default value for `docs_folder` is `.docs` so it appears at the top of the file tree like `.work`
- [ ] Config is merged/validated so missing `docs_folder` uses default `.docs`
- [ ] All kira code that writes or resolves documentation paths uses the configured docs path (no hardcoded `docs/` or `.docs/`)

### Init command
- [ ] `kira init` creates the docs folder and its standard subfolders (agents, architecture, product, reports, guides, api, etc.) using the configured `docs_folder` path
- [ ] If docs folder already exists: init respects `--force` (overwrite) and `--fill-missing` (add missing subfolders/files only); without flags, user is prompted (cancel / overwrite / fill-missing)
- [ ] When writing `kira.yml` on init, `docs_folder` is included (e.g. default `.docs`) so the path is explicit and configurable

### Artifact placement
- [ ] When kira or processes create documentation artifacts (e.g. from work items, release notes, ADRs, agent docs), they resolve the target path from config (`docs_folder`) and write to that location

### Folder Structure
- [ ] Docs folder (default `.docs`) exists in the repository root (or configured path)
- [ ] Docs folder root has a `README.md` that provides an overview of all documentation
- [ ] `product/` exists with README.md and may contain product docs (e.g. personas, vision, roadmap, glossary, feature briefs, problem statements, research summaries, success metrics, commercials)
- [ ] `reports/` exists with README.md for release, metrics, audit, or retrospective reports
- [ ] `agents/` exists (for PRD 005)
- [ ] `architecture/` exists with a README.md
- [ ] `architecture/<id>-<name>.adr.[md|qmd]` – ADRs may be Markdown (`.md`) or Quarto (`.qmd`)
- [ ] `guides/` exists with a README.md index
- [ ] `api/` exists with a README.md index
- [ ] Security content lives under `guides/security/` (e.g. `golang-secure-coding.md`); existing `docs/security/` is migrated there and documented in the index

### Documentation Index
- [ ] Docs folder root `README.md` includes:
  - Overview of the documentation structure
  - Links to all major documentation sections
  - Brief description of what each section contains
  - Navigation to help users find relevant docs

### Integration
- [ ] `README.md` (root) links to the docs folder’s README (e.g. `.docs/README.md`) in an appropriate section
- [ ] `AGENTS.md` references the docs folder’s `agents/` (e.g. `.docs/agents/`) for detailed agent documentation
- [ ] `CONTRIBUTING.md` references relevant guides in the docs folder’s `guides/` (e.g. `.docs/guides/`) if applicable

### Documentation Standards
- [ ] All documentation files use kebab-case naming
- [ ] Docs may be Markdown (`.md`) or Quarto (`.qmd`); format is consistent per file
- [ ] All major subdirectories have README.md index files
- [ ] Documentation follows consistent Markdown/Quarto formatting
- [ ] Cross-references between docs work correctly
- [ ] Quarto docs (where used) render successfully (e.g. `quarto render`) or project documents render expectations

### Git Integration
- [ ] All documentation (`.md`, `.qmd`) is committed to git as source
- [ ] `.gitignore` may exclude Quarto cache/generated output (e.g. `_freeze/`, `*.html` in docs folder) per project; policy documented in docs or CONTRIBUTING
- [ ] Documentation changes are tracked in git history

## Implementation Notes

### Configuration

1. **Add `docs_folder` to config** (`internal/config/config.go`):
   - Add `DocsFolder string` to `Config` (e.g. `yaml:"docs_folder"`).
   - In `DefaultConfig`, set `DocsFolder: ".docs"`.
   - In `mergeWithDefaults`, if `config.DocsFolder == ""`, set to `".docs"`.
   - Validate: if set, path should be safe (no `..`, reasonable length). Optionally restrict to a single path segment or relative path under repo root.

2. **Resolve docs path everywhere**: Any command or helper that creates or references documentation artifacts must call a helper that returns the configured docs root (e.g. `filepath.Join(workDir, cfg.DocsFolder)`), not a hardcoded `docs` or `.docs`.

### Init command

1. **Create docs folder in `kira init`** (`internal/commands/init.go`):
   - After creating `.work` and its contents, read config (or use default) for `docs_folder`.
   - Create the docs folder (e.g. `.docs`) and standard subfolders: `agents`, `architecture`, `product`, `reports`, `guides`, `guides/security`, `api`.
   - Create `README.md` in the docs root and in each subfolder (index content as per PRD).
   - If the docs folder already exists: apply same decision flow as `.work` (prompt or `--force` / `--fill-missing`). For fill-missing, only add missing subfolders and index files; do not remove existing content.

2. **Ensure `kira.yml` written by init includes `docs_folder`** (e.g. `docs_folder: .docs`) so the path is explicit and configurable.

### Initial Setup (folder structure under configured path)

1. **Create folder structure** (relative to configured docs folder, e.g. `.docs`):
   ```bash
   DOCS_ROOT="${DOCS_FOLDER:-.docs}"   # from config
   mkdir -p "$DOCS_ROOT/agents"
   mkdir -p "$DOCS_ROOT/architecture"
   mkdir -p "$DOCS_ROOT/product"
   mkdir -p "$DOCS_ROOT/reports"
   mkdir -p "$DOCS_ROOT/guides/security"
   mkdir -p "$DOCS_ROOT/api"
   ```

2. **Create index files**: Each subdirectory should have a `README.md` that:
   - Explains the purpose of that documentation section
   - Lists available documents in that section
   - Provides brief descriptions

3. **Create main docs index** (e.g. `.docs/README.md`):
   - Overview of documentation organization
   - Links to all major sections
   - Quick navigation guide
   - Note about what belongs in the docs folder vs other locations

### Content Migration

1. **Security content**: Move existing `docs/security/` (e.g. `golang-secure-coding.md`) into the configured docs folder’s `guides/security/` (e.g. `.docs/guides/security/`). Update any references (e.g. AGENTS.md, README) to use the configured path. Document in the docs folder’s README and `guides/README.md`.
2. **Future content**: As new documentation is created (e.g. `.docs/agents/using-kira.md` from PRD 005), it should be added to the appropriate index; artifact-creation code must use the configured docs path.

### ADR Template (Optional)

Consider creating a template for Architecture Decision Records in the docs folder’s `architecture/` (e.g. `.docs/architecture/`):
- `template.adr.md` (or `template.adr.qmd` if using Quarto)
- Follow standard ADR format (context, decision, consequences)

### Quarto setup (optional)

When using Quarto in the docs folder:
- Install Quarto CLI ([quarto.org](https://quarto.org/docs/get-started/)); ensure required engines (e.g. Python, R) are available for executable chunks.
- Render from repo root using the configured docs path: e.g. `quarto render .docs/reports/` or per-file. Optionally add a script or CI job to render `.qmd` and publish or commit outputs.
- Document in the docs folder’s README (e.g. `.docs/README.md`) that some docs are Quarto and how to render them; list any generated output or cache policy (e.g. `_freeze/` in `.gitignore`).

### Documentation Maintenance

- Keep indexes up-to-date as new docs are added
- Review and organize docs periodically
- Remove or archive outdated documentation
- Ensure links remain valid

### Agent Considerations

Since agents will be reading and potentially creating documentation:
- Use clear, structured headings
- Include examples where helpful
- Cross-reference related documentation
- Make navigation intuitive

## Open Questions / Clarifications

Resolve these before or during implementation so the PRD is unambiguous:

1. **Init: one prompt or two?** When both `.work` and the docs folder already exist, should init prompt once (e.g. “workspace already exists: c/o/f” for both) or twice (once for `.work`, once for docs)? Recommendation: one combined prompt so the user chooses once for the whole workspace.

2. **`docs_folder` path shape.** Can the value be a multi-segment path (e.g. `internal/docs` or `content/documentation`) or only a single path segment (e.g. `docs`, `.docs`)? Recommendation: allow any relative path that does not contain `..`; init creates that path and subfolders under it. Document the choice in the Implementation Notes.

3. **Migration: init vs manual.** Should `kira init` automatically migrate an existing `docs/security/` (or other legacy docs) into the configured docs folder (e.g. copy into `.docs/guides/security/`), or is migration a one-time manual/script step for this repo only? Recommendation: init does not auto-migrate; migration is documented as a manual step (or separate task) so init stays simple and predictable.

4. **Existing repos with `docs/`.** If the user already has a top-level `docs/` folder and runs init with default `.docs`, we create `.docs/` and leave `docs/` as-is. Should we document that users can set `docs_folder: docs` in `kira.yml` to adopt their existing folder, and that init with `--fill-missing` will then add only missing subfolders under `docs/`? Recommendation: add one sentence to Integration or Implementation Notes.

5. **No current artifact writers.** There is no kira code today that writes to a docs path; the “use configured path” AC applies to future commands. Implementers should add a shared helper (e.g. `config.DocsRoot(targetDir string) (string, error)`) and use it in init and in any future feature that writes docs.

## Release Notes

- **Configurable docs folder**: `kira.yml` supports `docs_folder` (default `.docs`) so the docs path appears at the top of the file tree like `.work` and can be set to `docs` or a custom path
- **Init creates docs folder**: `kira init` creates the docs folder and standard subfolders (agents, architecture, product, reports, guides, api); respects `--force` and `--fill-missing` like `.work`
- **Artifact placement**: Kira and processes use the configured docs path when creating documentation artifacts (e.g. from work items, release notes, ADRs, agent docs)
- Established docs folder structure for long-lived project documentation
- Created organized subdirectories: `agents/`, `architecture/`, `product/`, `reports/`, `guides/`, `api/`
- `product/` for product docs (personas, vision, roadmap, glossary, feature briefs, commercials, and related artifacts)
- `reports/` for release reports, metrics summaries, audits, and retrospectives
- Added docs folder README as the main documentation index
- Security content lives under `guides/security/`; existing `docs/security/` is migrated there
- Integrated documentation references in README.md and AGENTS.md
- **Quarto (`.qmd`) supported** for executable, notebook-like docs: embed code, run analysis, and render richly when desired (see [quarto.org](https://quarto.org))
- Provides foundation for comprehensive project documentation including agent guides, ADRs, development references, and optional in-doc analysis

