---
id: 016
title: roadmap-planning
status: backlog
kind: prd
assigned:
estimate: 0
created: 2026-01-21
due: 2026-01-21
tags: [planning, roadmap, visualization, polyrepo]
---

# roadmap-planning


## Context

Road mapping in Kira is intended to support early-stage planning and sequencing of work before tasks are fully elaborated into durable work items. The roadmap acts as a lightweight, text-first planning surface that is easy to edit, diff, and reason about by both humans and LLMs.

The core design goal is to separate:
- **Planning intent** (ordering, grouping, provisional dependencies)
- from **task truth** (the canonical work item files)

To achieve this, roadmaps are data-first and YAML-based, with visualisation treated as a derived concern. A roadmap may contain references to existing work items as well as ad-hoc items that do not yet have an ID or backing file.

Roadmaps also act as a *staging area* for proposed metadata changes (e.g. dependencies, targets, tags) that can later be applied to work items once planning stabilises.

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
- Groups support staged metadata just like individual entries.
- Groups can participate in dependency relationships (group-level dependencies).
- Groups enable hierarchical filtering and selection (e.g., "all items in workstream:auth").

### Staged Overrides
- Roadmap entries may contain a `stage` block.
- The `stage` block represents proposed or draft metadata changes.
- Staged data does not modify the underlying work item until explicitly applied.
- Supported staged fields are intentionally limited (e.g. dependencies, tags, targets, notes).

### Filtering and Selection
- Roadmap entries may include metadata such as `period`, `workstream`, or `stream`.
- Operations can be scoped using selectors (e.g. period=Q1-26, workstream=auth).
- Hierarchical selectors allow filtering by group path (e.g., "workstream:auth/epic:oauth").
- Selection can target specific levels of the hierarchy (e.g., "all groups at depth 2").

### Apply Operation
- **Command**: `kira roadmap apply [filters] [flags]`
- A roadmap apply operation must:
  - Select a subset of roadmap entries via filters.
  - Promote ad-hoc items in that subset into real work items.
  - Apply staged overrides to referenced work items.
  - Update the roadmap to replace promoted ad-hoc items with newly assigned IDs.

#### Command Syntax Examples
```bash
# Apply all staged changes in current roadmap
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
- Roadmaps can reference work items without duplicating their data.
- Ad-hoc roadmap items can exist without IDs.
- Staged overrides do not affect work items until an apply operation is run.
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
- `ROADMAP.md` is always the current/active roadmap.
- Draft roadmaps can be created with filtered items from the current roadmap.
- Draft roadmaps can be promoted to current, archiving the previous roadmap.

## Implementation Notes

### Example Roadmap Structure

#### Flat Structure (Simple Case)
```yaml
roadmap:
  - id: AUTH-001
    stage:
      depends_on+: [PLAT-002]

  - title: OAuth provider spike
    meta:
      period: Q1-26
      workstream: auth
    stage:
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
            stage:
              depends_on+: [PLAT-002]
          - title: OAuth provider spike
            stage:
              owner: wayne
              outcome: decision
          - title: OAuth token refresh
            stage:
              tags+: [security, oauth]

      - group: SSO Features
        meta:
          epic: sso
        items:
          - id: AUTH-002
          - title: SAML integration
            stage:
              depends_on+: [AUTH-001]

  - group: Platform Migration
    meta:
      period: Q2-26
      workstream: platform
    items:
      - id: PLAT-002
        stage:
          tags+: [migration, infrastructure]
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

#### Alternative: Inline Group Syntax
```yaml
roadmap:
  - title: Authentication Workstream
    type: group
    meta:
      period: Q1-26
      workstream: auth
    items:
      - title: OAuth Integration Epic
        type: group
        items:
          - id: AUTH-001
          - title: OAuth provider spike
```

**Note**: The exact syntax can be refined during implementation. The key is supporting nested groups that can contain both work item references and ad-hoc items, enabling tree-like organization that maps naturally to Gantt charts and roadmap visualizations.

### Promotion Rules
- When applying a roadmap:
  - Entries without an ID are promoted to new work item files.
  - ID generation may be based on workstream prefixes, group hierarchy, or repository conventions.
  - Staged fields are written into the new work item on creation.
  - Groups are not promoted to work items (they remain organizational structure).
  - Group metadata can be inherited by child items during promotion (e.g., workstream from parent group).
  - The hierarchical structure is preserved in the roadmap after promotion (groups remain, ad-hoc items become ID references).

### Merge Semantics
- `field:` replaces the existing value.
- `field+:` appends to list-based fields.
- `field-:` removes from list-based fields.
- Conflicts may optionally be detected using a stored base hash.

### Roadmap Versioning and Archiving
- **Current Roadmap**: `ROADMAP.md` is always the active/current roadmap file.
- **Draft Roadmaps**: Users can create draft roadmaps (e.g., `ROADMAP-draft.md`, `ROADMAP-Q2-26.md`) for planning future periods or major changes.
- **Command**: `kira roadmap draft <name> [filters]` - Create a new draft roadmap
  - Can optionally bring over outstanding items from the current roadmap
  - Supports filtering to select which items come over (e.g., only items from specific workstreams, periods, or status)
  - Example: `kira roadmap draft Q2-26 --period Q2-26 --status todo,doing`
- **Command**: `kira roadmap promote <draft-name>` - Promote a draft to current and archive the previous
  - Moves the draft roadmap to `ROADMAP.md` (becomes the current roadmap)
  - Archives the previous `ROADMAP.md` to `.work/{archived}/roadmap/ROADMAP-{timestamp}.md` where `{archived}` is the configured archived folder (default: `z_archive`)
  - Preserves git history through the rename operation
- **Outstanding Items**: When creating a draft, outstanding items (not in "done" or "released" status) can be automatically included based on filters
- **Filtering on Draft Creation**: Filters determine which items from the current roadmap are copied to the draft:
  - `--status <statuses>` - Only include items with specific statuses
  - `--period <period>` - Only include items from specific periods
  - `--workstream <workstream>` - Only include items from specific workstreams
  - `--exclude-done` - Exclude completed items (default behavior)
  - `--include-all` - Include all items regardless of status

### Safety and UX
- Apply operations support `--dry-run`.
- Roadmap linting validates references and staged fields.
- Visualisation (e.g. Mermaid, Gantt charts) is treated as a downstream render step and is not part of the core data model.
- The hierarchical structure naturally supports Gantt chart rendering (groups as summary tasks, items as tasks).
- Tree visualization can show the roadmap structure with expandable/collapsible groups.
- Flat views can be generated by flattening the hierarchy when needed (e.g., for simple lists or exports).
- Roadmap promotion includes confirmation prompt to prevent accidental overwrites.

## Release Notes

- Added roadmap planning system with hierarchical tree structure
- Roadmaps stored as YAML files (ROADMAP.md for current roadmap)
- Support for nested work items (work items can contain other work items)
- Support for groups and ad-hoc items at any nesting level
- Staged metadata changes that can be applied to work items
- `kira roadmap apply` command with filtering options (--period, --workstream, --owner, etc.)
- Draft roadmap creation with `kira roadmap draft` command
- Draft roadmaps can selectively bring over outstanding items from current roadmap
- Roadmap promotion with `kira roadmap promote` to make draft current and archive previous
- All roadmap operations support --dry-run for safety
