---
id: 016
title: roadmap-planning
status: done
kind: prd
created: 2026-01-21T00:00:00Z
assigned: null
due: 2026-01-21T00:00:00Z
estimate: 0
merge_commit_sha: 2862c0e859de225438ba9554eafbc8be258f8c89
merge_strategy: rebase
merged_at: "2026-03-24T03:56:25Z"
pr_number: 27
tags: [planning, roadmap, visualization, polyrepo]
---

# roadmap-planning


## Context

Road mapping in Kira is intended to support early-stage planning and sequencing of work before tasks are fully elaborated into durable work items. The roadmap acts as a lightweight, text-first planning surface that is easy to edit, diff, and reason about by both humans and LLMs.

The core design goal is to separate:
- **Planning intent** (ordering, grouping, provisional dependencies)
- from **task truth** (the canonical work item files)

To support both human/LLM authoring and machine consumption, the system distinguishes:
- **PLAN.md** — Free-form, prose-first planning document. It captures intent extracted from use cases, PRDs, and other text; it guides an LLM and serves as the source of truth for *what we want to do* before structure is fixed.
- **ROADMAP.yml** — Structured, data-first representation derived from the plan. The canonical current roadmap file is always `ROADMAP.yml`. It is the canonical format for ordering, grouping, dependencies, and metadata; it can be generated or updated from the plan (e.g. via LLM extraction).

Workflow: *product overview / use cases / domain models / architecture / other docs → extraction → PLAN.md (free text) → generation → ROADMAP.yml (structured).*

To achieve this, roadmaps are data-first and YAML-based, with visualisation treated as a derived concern. A roadmap may contain references to existing work items as well as ad-hoc items that do not yet have an ID or backing file.

## Plan vs Roadmap

- **PLAN.md** is the free-form planning artifact:
  - Written in natural language (markdown). It describes goals, sequencing, workstreams, and rationale in prose.
  - Intended to guide an LLM and to be updated by humans or by extraction from existing artifacts.
  - **Extraction inputs**: PRODUCT.md high-level product idea and goals, Use cases, DOMAIN_MODELS.md, ARCHITECTURE.md, ADRs, notes, and other text in the repo can be summarized or distilled into PLAN.md (manually or via LLM) so that the “current plan” is captured in one place.
  - **Role**: Single, readable source for *what we’re planning* before committing to a structured format.

- **Roadmap (ROADMAP.yml)** is the structured counterpart:
  - YAML tree of entries (work item refs, ad-hoc items, groups), with ordering, dependencies, and metadata.
  - **Derived from the plan**: The roadmap can be generated or refreshed from PLAN.md (e.g. via an LLM or tool that parses the plan and emits/updates the YAML). This keeps the roadmap aligned with the narrative in the plan.
  - **Role**: Machine-readable format for apply, filters, and visualization; this file feeds agentic workflows so that multiple agents can work in parallel picking up work items and progressing them, asking for help where need be etc.

This separation allows: (1) editing and reasoning in prose first, (2) extraction from scattered docs into one plan, and (3) generating a consistent, structured roadmap from that plan.

## Requirements

### Roadmap Data Model
- Roadmaps are stored as YAML
- A roadmap consists of a hierarchical tree structure of entries.
- Each entry is one of:
  - A reference to an existing work item by ID.
  - An ad-hoc item identified only by a title.
  - A group/container that contains nested entries (supports arbitrary nesting depth).
- **Work items can be nested**: Work item references can appear at any level of the hierarchy, not just as leaf nodes. This allows work items to contain other work items, creating parent-child relationships (e.g., an epic work item containing feature work items).
- Groups can represent workstreams, epics, phases, releases, or any organizational structure.
- The hierarchical structure naturally maps to Gantt chart work breakdown structures and roadmap visualizations.

### Ad-hoc Items
- Ad-hoc items may include inline metadata (owner, notes, outcomes, etc.).
- Ad-hoc items do not have IDs until they are explicitly promoted.
- Ad-hoc items may participate in ordering and dependency relationships within the roadmap.
- Ad-hoc items can exist at any level of the hierarchy (as leaf nodes or as groups).

### Hierarchical Groups
- Groups are containers that organize related entries into a tree structure.
- Groups can be nested to arbitrary depth (e.g., workstream → epic → feature → task).
- Groups may have metadata (title, description, period, workstream, owner, etc.).
- Groups can contain a mix of:
  - References to existing work items (which themselves may contain nested work items)
  - Ad-hoc items
  - Nested groups
- Groups can participate in dependency relationships (group-level dependencies).
- Groups enable hierarchical filtering and selection (e.g., "all items in workstream:auth").

### Filtering and Selection
- Roadmap entries may include metadata such as `period`, `workstream`, etc. (use **workstream** consistently; do not use `stream`).
- Operations can be scoped using selectors (e.g. period=Q1-26, workstream=auth).
- **Hierarchical filter syntax:** Path is a sequence of segments separated by `/`. Each segment is `key:value` where `key` is a meta field name (e.g. `workstream`, `epic`, `period`) and `value` is a single token. Matching is **by meta only** (not group title): a group matches a segment if `meta[key] == value`. Include an entry (group or item) if its path from root to entry matches the filter path (prefix or full). Example: `workstream:auth/epic:oauth` includes all entries under a group with `meta.workstream == "auth"` that are under a child group with `meta.epic == "oauth"`. For values containing `:` or `/`, use **quoted values** (e.g. `workstream:"auth/sso"`); alternatively restrict v1 to alphanumeric plus hyphen/underscore and document that.
- Selection can target specific levels of the hierarchy (e.g., "all groups at depth 2").

### Apply Operation
- **Command**: `kira roadmap apply [filters] [flags]`
- A roadmap apply operation must:
  - Select a subset of roadmap entries via filters.
  - Promote ad-hoc items in that subset into real work items (creating work item files and assigning IDs).
  - Update the roadmap in place to replace promoted ad-hoc items with newly assigned IDs (see below for file and conflict behavior).

**Where promoted work items are created:** New work item files are created in the **backlog** folder (`status_folders.backlog`). Subfolders per phase (e.g. per period) may be used to organize promoted items; exact layout is implementation-defined.

**ID generation:** Use the existing **GetNextID()** convention (e.g. numeric `\d{3}` or as configured via `validation.id_format`). No workstream prefix or custom ID shape is required for v1.

**Failure and atomicity:** Apply is **best-effort**. If some promotions succeed and others fail (e.g. validation, file write), log each failure with a clear reason. The user can fix issues and re-run apply; already-promoted items are skipped (by ID), and the command brings across the remaining ad-hoc items. No rollback of partial writes is required.

**Roadmap file update and conflicts:** Apply **rewrites the current roadmap file (ROADMAP.yml) in place** after promotions—either after each successful promotion or in a single write at the end of a best-effort run, replacing ad-hoc entries with their new IDs. **Read the roadmap once at the start of apply.** If the file is modified on disk after that read but before write, **overwrite** with the in-memory version. Optional: before writing, detect changed mtime/size and **warn** that the file may have been edited elsewhere, with a prompt to continue or abort.

**Resolving work item IDs:** Resolution uses the same rule as the rest of Kira: **findWorkItemFile(workItemID, cfg)** (search work folder by `id` in front matter). When an ID in the roadmap has no file: **lint** — report a warning (or error if strict) and list broken IDs; **apply** — skip that entry (do not promote if it was ad-hoc); optionally support a "strict apply" mode that fails the whole apply with a clear message.

#### Command Syntax Examples
```bash
# Promote ad-hoc items in current roadmap
kira roadmap apply

# Apply only items matching specific filters
kira roadmap apply --period Q1-26 --workstream auth

# Dry-run to see what would change
kira roadmap apply --period Q1-26 --workstream auth --dry-run

# Apply with multiple filters
kira roadmap apply --period Q1-26 --workstream auth --owner wayne

# Apply only specific groups or items
kira roadmap apply --filter "workstream:auth/epic:oauth"
```

#### Filter Options
- `--period <period>` - Filter by period (e.g., Q1-26)
- `--workstream <workstream>` - Filter by workstream
- `--owner <owner>` - Filter by owner
- `--filter <path>` - Hierarchical path filter (e.g., "workstream:auth/epic:oauth")
- `--dry-run` - Show what would change without making changes

## Acceptance Criteria

- Roadmaps can be authored entirely in plain text.
- **PLAN.md** exists as the free-form planning artifact; roadmap YAML is the structured derivative that can be generated or updated from the plan.
- Roadmaps can reference work items without duplicating their data.
- Ad-hoc roadmap items can exist without IDs.
- The apply operation can be safely run with filters to limit scope.
- Applying a roadmap results in deterministic file creation and modification.
- A dry-run mode clearly reports intended changes before mutation.
- Roadmaps support hierarchical grouping with arbitrary nesting depth.
- Groups can contain a mix of work item references, ad-hoc items, and nested groups.
- Work items can be nested within other work items (parent-child relationships).
- Group metadata can be inherited by child items during promotion.
- The hierarchical structure maps naturally to Gantt chart work breakdown structures.
- Roadmap visualization can render the tree structure (expandable groups, indentation, etc.).
- Flat views can be generated by flattening the hierarchy when needed.
- **ROADMAP.yml** is always the current/active roadmap (canonical filename; content is YAML).
- Draft roadmaps can be created with filtered items from the current roadmap.
- Draft roadmaps can be promoted to current, archiving the previous roadmap.



## Implementation Notes

### PLAN.md and Roadmap Generation
- **Location**: PLAN.md lives under the docs folder, e.g. `.docs/PLAN.md` (when `docs_folder: .docs`). This keeps plan and roadmap-related docs together and makes tooling consistent.
- **Extraction**: Use cases, PRDs, and other text can be used (manually or via tooling/LLM) to populate or update PLAN.md so it reflects current intent.
- **Generation**: A run command (e.g. `kira run plan-to-roadmap` or LLM-assisted) can read PLAN.md and produce or update the structured roadmap YAML, so the roadmap stays aligned with the plan. Assume `kira run <llm-workflow-script>` will exist soon; plan→roadmap generation is in scope and can be implemented accordingly.

### Example Roadmap Structure

#### Flat Structure (Simple Case)
```yaml
roadmap:
  - id: AUTH-001

  - title: OAuth provider spike
    meta:
      period: Q1-26
      workstream: auth
      owner: wayne
      outcome: decision
```

#### Hierarchical Structure (Tree-like)
```yaml
roadmap:
  - group: Authentication Workstream
    meta:
      period: Q1-26
      workstream: auth
      owner: wayne
    items:
      - group: OAuth Integration Epic
        meta:
          epic: oauth
        items:
          - id: AUTH-001
          - title: OAuth provider spike
            meta:
              owner: wayne
              outcome: decision
          - title: OAuth token refresh
            meta:
              tags: [security, oauth]

      - group: SSO Features
        meta:
          epic: sso
        items:
          - id: AUTH-002
          - title: SAML integration
            meta:
              depends_on: [AUTH-001]

  - group: Platform Migration
    meta:
      period: Q2-26
      workstream: platform
    items:
      - id: PLAT-002
        meta:
          tags: [migration, infrastructure]
```

#### Nested Work Items Example
```yaml
roadmap:
  - id: EPIC-001  # Parent work item (epic)
    items:
      - id: FEAT-001  # Child work item (feature)
        items:
          - id: TASK-001  # Grandchild work item (task)
          - title: Ad-hoc task
      - id: FEAT-002
  - id: EPIC-002
    items:
      - title: New feature (ad-hoc, will be promoted)
```

**Canonical group form (v1):** Use **`group: <title>`** with nested `items:` as the only supported form. This is compact, clearly distinguishes groups from leaf entries (no `type` field), and matches the hierarchical examples above. Parsing, generation, and lint use this schema only. A `title` + `type: group` form may be considered in a later release for a unified entry shape.

### Promotion Rules
- When applying a roadmap:
  - Entries without an ID are promoted to new work item files (see Apply section for folder and ID generation).
  - Entry and group metadata (e.g. workstream, owner, period, tags) can be written into the new work item on creation.
  - Groups are not promoted to work items (they remain organizational structure).
  - **Group metadata inheritance:** All keys from a group's `meta` are inherited by child entries (items or nested groups) as **defaults**. Precedence: **child wins** — if the child has a value for a key (e.g. `meta.workstream`), that value is used; only missing keys are filled from the nearest parent group. When creating a work item from an ad-hoc entry, the effective metadata is the merged `meta` (parent chain with child wins) plus the entry's own fields. Document this behavior in command help and any promotion docs.
  - The hierarchical structure is preserved in the roadmap after promotion (groups remain, ad-hoc items become ID references).
  - **Stage blocks:** v1 does not support staged overrides on existing work items. Apply only promotes ad-hoc items to new work items; to change dependencies, tags, or owner on existing work items, users edit the work item file or use existing commands. Staged overrides may be reintroduced in a later release.

### Roadmap Versioning and Archiving
- **Current Roadmap**: **ROADMAP.yml** is always the active/current roadmap file.
- **Draft Roadmaps**: Users can create draft roadmaps (e.g., `ROADMAP-draft.yml`, `ROADMAP-Q2-26.yml`) for planning future periods or major changes.
- **Command**: `kira roadmap draft <name> [filters]` - Create a new draft roadmap
  - **Default when no filters are given:** Include **all outstanding items** from the current roadmap. "Outstanding" means status **not** in `done` or `released` (i.e. backlog, todo, doing, review, abandoned, archived are outstanding). Use **`--empty`** to create a draft with no items when starting from scratch.
  - Supports filtering to select which items come over (e.g., only items from specific workstreams, periods, or status).
  - Example: `kira roadmap draft Q2-26 --period Q2-26 --status todo,doing` or `kira roadmap draft Q2-26 --empty`.
- **Command**: `kira roadmap promote <draft-name>` - Promote a draft to current and archive the previous
  - Moves the draft roadmap to **ROADMAP.yml** (becomes the current roadmap). Uses **git mv** and commits the rename so git history is preserved; implementation performs the rename and commit (push is optional/configurable).
  - Archives the previous ROADMAP.yml to `.work/{archived}/roadmap/ROADMAP-{timestamp}.yml` where **{archived}** is the **folder name** from `status_folders.archived` (e.g. `z_archive`), not a separate config key.
  - Preserves git history through the rename operation (git mv + commit).
- **Filtering on Draft Creation**: Filters determine which items from the current roadmap are copied to the draft:
  - `--status <statuses>` - Only include items with specific statuses
  - `--period <period>` - Only include items from specific periods
  - `--workstream <workstream>` - Only include items from specific workstreams
  - `--exclude-done` - Exclude completed items (default behavior)
  - `--include-all` - Include all items regardless of status
  - `--empty` - Create draft with no items (overrides default of including outstanding items)

### Safety and UX
- Apply operations support `--dry-run`.
- **Roadmap linting:** **`kira roadmap lint`** (or `kira roadmap validate`) runs over the current roadmap file (ROADMAP.yml). It **validates:** (1) **References** — every `id:` references a work item that exists (via findWorkItemFile); (2) **Staged fields** — only allowlisted names appear under `stage` (if present); (3) **Schema** — required structure for entries (e.g. `id` or `title` or `group`/`items`, not empty); (4) **Dependencies** — optional: warn on unknown `depends_on` IDs and optionally detect cycles. Emit warnings for broken refs and unknown staged fields; exit non-zero if any errors (e.g. invalid schema). Do not modify the file. Document the command and checks in command help.
- **Ad-hoc items:** Ad-hoc items can remain in the roadmap until promoted. Provide a **flag on roadmap lint** (e.g. `--check-adhoc` or a dedicated report) that lists ad-hoc items so teams can work through them and ensure they are addressed or promoted.
- Visualisation (e.g. Mermaid, Gantt charts) is treated as a downstream render step and is not part of the core data model.
- The hierarchical structure naturally supports Gantt chart rendering (groups as summary tasks, items as tasks).
- Tree visualization can show the roadmap structure with expandable/collapsible groups.
- Flat views can be generated by flattening the hierarchy when needed (e.g., for simple lists or exports).
- Roadmap promotion includes confirmation prompt to prevent accidental overwrites.


## Slices

Slices are ordered; each slice is a committable unit of work. Tasks within a slice are implemented in order.

### Slice 1: Data model and YAML
- [x] T001: Define roadmap structs: entry types (id ref, title+meta ad-hoc, group+items), nested items; support arbitrary depth
- [x] T002: Implement YAML parse and serialize for ROADMAP.yml format (canonical group form: group: <title>, items:)
- [x] T003: Add schema validation: each entry has id or title or group+items; reject empty or invalid entries
- [x] T004: Add unit tests for parse, round-trip, and validation (valid flat and hierarchical examples)

### Slice 2: Work item resolution and roadmap lint
- [x] T005: Integrate findWorkItemFile(cfg) to resolve work item IDs to paths; use existing Kira config
- [x] T006: Implement `kira roadmap lint` (or validate): validate refs (every id: has file), schema, optional stage allowlist
- [x] T007: Add optional dependency checks: warn on unknown depends_on IDs; optional cycle detection
- [x] T008: Add --check-adhoc flag (or report) to list ad-hoc items; document in help
- [x] T009: Add tests for lint: valid roadmap, broken refs, schema errors, ad-hoc report

### Slice 3: Filtering
- [x] T010: Implement filter types: --period, --workstream, --owner matching entry/group meta
- [x] T011: Implement hierarchical path filter (key:value/key:value, quoted values for : or /)
- [x] T012: Implement selection: include entry if path from root matches filter (prefix or full); by meta only
- [x] T013: Add unit tests for filter combination and path matching

### Slice 4: Apply
- [x] T014: Implement `kira roadmap apply [filters]`: read ROADMAP.yml, select entries by filters
- [x] T015: Promote ad-hoc entries: create work item files in status_folders.backlog, GetNextID(); merge group meta (child wins)
- [x] T016: After promotions, rewrite ROADMAP.yml in place (replace ad-hoc with new IDs); handle file-modified warning optional
- [x] T017: Add --dry-run; best-effort on failure (log, skip already promoted); tests for apply and dry-run

### Slice 5: Draft and promote
- [x] T018: Implement `kira roadmap draft <name> [filters]`: default include outstanding items (status not done/released); --empty for empty draft
- [x] T019: Support draft filters: --status, --period, --workstream, --include-all
- [x] T020: Implement `kira roadmap promote <draft-name>`: mv draft to ROADMAP.yml, archive previous to .work/{archived}/roadmap/ROADMAP-{timestamp}.yml, git mv + commit
- [x] T021: Add confirmation prompt for promote; tests for draft and promote

### Slice 6: PLAN.md and docs
- [x] T022: Document PLAN.md location (e.g. .docs/PLAN.md from docs_folder); add to implementation notes or config if needed
- [x] T023: Document plan vs roadmap, extraction and generation workflow in command help or docs
- [x] T024: Optional: add run script or stub for plan-to-roadmap (read PLAN.md, emit/update ROADMAP.yml); minimal or no-op acceptable for v1

## Release Notes

- Added roadmap planning system with hierarchical tree structure
- **PLAN.md** as free-form planning doc; extraction from use cases/docs feeds the plan; roadmap YAML can be generated from the plan
- Roadmaps stored as YAML files (**ROADMAP.yml** for current roadmap)
- Support for nested work items (work items can contain other work items)
- Support for groups and ad-hoc items at any nesting level
- `kira roadmap apply` command (promotes ad-hoc items to work items) with filtering options (--period, --workstream, --owner, etc.)
- Draft roadmap creation with `kira roadmap draft` command
- Draft roadmaps can selectively bring over outstanding items from current roadmap
- Roadmap promotion with `kira roadmap promote` to make draft current and archive previous
- `kira roadmap lint` (or validate) for reference, schema, and optional dependency checks; optional flag to report ad-hoc items
- All roadmap operations support --dry-run for safety
