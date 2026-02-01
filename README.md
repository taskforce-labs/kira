# Kira
Version: v0.1.0-alpha

A cli to manage your work-items and specs in git as markdown files so you can use them with LLMs in your IDE or Agent Workflows. No need for JIRA.

Develop multiple features in parallel with kira workflows that create worktrees, branches and pull requests based on the specs and work-items you create.


## Quick Start

For detailed documentation see [.docs/README.md](.docs/README.md).

0. Setup the project
```bash
kira init
# my-project
# ├── kira.yml
# ├── .work/
# │   ├── 0_backlog/
# │   ├── 1_todo/
# │   ├── 2_doing/
# │   ├── 3_review/
# │   ├── 4_done/
# │   ├── templates/
# │   ├── z_archive/
# │   └── IDEAS.md
```

1. Creates a new work-item
```bash
kira new prd todo "User Authentication" "Implement OIDC based user login system"
# .work/1_todo/001-user-authentication.prd.md
```

2. Start the work-item in an isolated workspace
```bash
kira start 001
# 1. Sets the work-item to doing status
# 2. Creates a new worktree
# my-project_worktree/
# └── 001-user-authentication
# 3. Creates a new branch in the worktree
# 001-user-authentication
# 4. Pushes branch and opens a draft pull request
# 5. Opens the IDE in the worktree
# 6. Runs setup commands
```

3. Submits the work-item for review and creates a pull request
```bash
kira review
# 1. Rebases the branch onto the trunk branch
# 2. Changes the work-item to review status
# 3. Changes the status of the pull request to ready for review
```

4. Merges the pull request and marks the work-item as done
```bash
kira done
# 1. Rebases the branch onto the trunk branch
# 2. Merges the pull request
# 3. Marks the work-item as done
```

5. Releases the work items in the done folder
```bash
kira release
# 1. Generates release notes from the done folder
# 2. Updates the releases file
# 3. Archives the work items in the done folder
# 4. Tags and pushes the release to trigger release workflow
```

### Draft pull requests

`kira start` can push the new branch and open a **draft pull request** on GitHub for the work item. This is enabled by default when your remote is GitHub.

- **Enable:** Set the `KIRA_GITHUB_TOKEN` environment variable (e.g. a GitHub personal access token with `repo` scope). Draft PRs are created only for GitHub remotes.
- **Skip:** Use `--no-draft-pr` to skip pushing and creating a draft PR.
- **Config:** In `kira.yml`, use `workspace.draft_pr: false` to disable for the workspace, or `projects[].draft_pr: false` in polyrepo setups. Use `workspace.git_base_url` for GitHub Enterprise.

Example `workspace` in `kira.yml`:

```yaml
workspace:
  draft_pr: true          # default: true (create draft PRs for GitHub)
  git_platform: auto      # github | auto (default)
  git_base_url: ""        # optional; for GitHub Enterprise (e.g. https://ghe.example.com)
  # projects[].draft_pr   # optional override per project (polyrepo)
```

## Overview

Kira uses a combination of plaintext markdown files, git, and a lightweight CLI to manage and coordinate work. This approach means clankers can directly read, write, and commit changes without complex APIs, while meatbags get full transparency into what the clankers are doing through git history.

Unlike Jira and other tools that are overly complicated and expensive, Kira keeps it simple there in your codebase. It's free and open source - Kira could be the Jira killer you've been waiting for.

## Installation

### Package Managers (Recommended)

**macOS (Homebrew):**
```bash
brew tap taskforce-labs/kira
brew install kira
```

**Windows (Scoop):**
```powershell
scoop bucket add taskforce-labs https://github.com/taskforce-labs/scoop-bucket.git
scoop install kira
```

### Download Pre-built Binaries

Pre-built binaries are available for Linux, macOS, and Windows on the [GitHub Releases](https://github.com/taskforce-labs/kira/releases) page.

**Linux/macOS:**
```bash
# Download and extract the appropriate archive for your platform
# Example for Linux amd64:
wget https://github.com/taskforce-labs/kira/releases/latest/download/kira_linux_amd64.tar.gz
tar -xzf kira_linux_amd64.tar.gz
sudo mv kira /usr/local/bin/

# Or for macOS arm64:
wget https://github.com/taskforce-labs/kira/releases/latest/download/kira_darwin_arm64.tar.gz
tar -xzf kira_darwin_arm64.tar.gz
sudo mv kira /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/taskforce-labs/kira/cmd/kira@latest
```

### Build from Source (Contributors)

If you want to build and test locally, see the contributor guide:

```bash
# Development setup, building, testing, and install options:
see CONTRIBUTING.md
```

## Commands

### `kira init [folder]`
Creates the files and folders used by kira in the specified directory. If a `.work/` directory already exists, you can choose how to proceed using flags or interactively.

```bash
kira init                              # Initialize in current directory
kira init ~/my-project                # Initialize in specific directory
kira init --fill-missing              # Add any missing files/folders, keep existing
kira init --force                     # Overwrite existing .work (fresh init)
```

Notes:
- Creates status folders and template files.
- Adds `.gitkeep` files to empty folders.
- Without flags, if `.work/` exists you'll be prompted to cancel, overwrite, or fill-missing.

### `kira new [template] [status] [title] [description]`
Creates a new work item from a template.

```bash
kira new                                              # Prompts for template selection
kira new prd doing "Fix login bug"                    # Status before title
kira new prd "Feature"                                # Status omitted → defaults to backlog
kira new prd backlog "Feature"                        # Explicit status
kira new prd "Feature" --interactive                  # Enable prompts for missing fields
kira new prd "Feature" -I                            # Shorthand for --interactive
kira new prd "Feature" --input assigned=me@acme.com  # Provide inputs (key=value)
```

Notes:
- By default, only provided values are filled; missing template fields use defaults. The `created` field is set automatically to today when creating work items with `kira new`.
- Use `--interactive` (or `-I`) to enable prompts for missing template fields

### `kira users`
Lists users discovered from git history and/or `kira.yml`, and assigns each user a **number** you can use with `kira assign`.

```bash
kira users                 # Table output (default)
kira users --format list   # Plain list
kira users --format json   # Machine-readable output

# Limit git history processing (0 = no limit)
kira users --limit 200
```

Workflow with `kira assign`:

```bash
# 1) List users and note the number you want (e.g. 5)
kira users

# 2) Assign using that number
kira assign 001 5
```

### `kira assign <work-item-id...> [user-identifier]`
Assigns one or more work items to a user by updating front matter (default field: `assigned`).

```bash
# Switch (replace) mode (default)
kira assign 001 5                         # Assign work item 001 to user #5 (from `kira users`)
kira assign 001 user@example.com          # Assign by email (case-insensitive)
kira assign 001 "Jane Doe"                # Assign by name (exact/partial match if unique)
kira assign 001 002 003 5                 # Batch assign multiple work items

# Append mode (build a list; avoids duplicates)
kira assign 001 5 --append
kira assign 001 5 -a

# Unassign (clears/removes the field)
kira assign 001 --unassign
kira assign 001 -u

# Custom field (defaults to `assigned`)
kira assign 001 5 --field reviewer
kira assign 001 5 -f reviewer

# Interactive selection (user identifier optional)
kira assign 001 --interactive
kira assign 001 -I

# Dry run (no changes written)
kira assign 001 5 --dry-run
```

### `kira move <work-item-id> [target-status]`
Moves a work item to a different status folder.

```bash
kira move 001              # Show status options
kira move 001 doing        # Move to doing folder
```

### `kira idea <description>`
Adds an idea to the IDEAS.md file.

```bash
kira idea "Add dark mode support"
```

### `kira lint`
Scans for issues in work items.

```bash
kira lint                    # Standard validation
kira lint --strict          # Enable strict mode (flag unknown fields)
```

Behavior:
- Validates all work items for format, required fields, and field configuration compliance
- Groups errors by category (Field Validation, Unknown Fields, Workflow, Duplicate IDs, Parse, Other)
- Shows error counts for each category
- With `--strict`: Flags fields not defined in configuration

### `kira doctor`
Validates work items, automatically fixes issues where possible, and reports issues requiring manual attention.

```bash
kira doctor                  # Standard mode
kira doctor --strict        # Enable strict mode (flag unknown fields)
```

Behavior:
1. **Validates all work items** (same as `kira lint`) and displays all issues grouped by category
2. **Automatically fixes**:
   - **Duplicate IDs**: Assigns new sequential IDs to duplicate work items
   - **Date format issues**: Converts invalid date formats (e.g., ISO 8601 timestamps) to `YYYY-MM-DD` format for the `created` field
   - **Field validation issues**: Fixes common field problems:
     - Date format conversions
     - Enum value case corrections (when case-insensitive)
     - Email trimming and lowercasing
   - **Missing required fields**: Adds missing required fields with default values
3. **Reports unfixable issues** that require manual intervention:
   - Workflow violations (e.g., multiple items in doing folder)
   - Invalid status values
   - Invalid ID formats
   - Missing required fields without defaults
   - Unknown fields (in strict mode)
   - Parse errors

**Note**: The `doctor` command preserves all markdown body content when fixing issues. Only the YAML front matter is modified.

With `--strict`: Also flags fields not defined in configuration as unfixable issues.

### `kira release [status|path] [subfolder]`
Generates release notes and archives completed work items.

```bash
kira release                    # Release from done folder
kira release done v2           # Release from done/v2 subfolder
kira release 4_done/v2         # Release from specific path
```

Behavior:
- Updates work item status to "released" before archival
- Archives to `.work/z_archive/{date}/{original-path}/`
- Prepends release notes to the configured `release.releases_file` (default `RELEASES.md`)
- Only items with a `# Release Notes` section are included in notes

### `kira abandon <work-item-id|path> [reason|subfolder]`
Archives work items and marks them as abandoned.

```bash
kira abandon 001                                    # Abandon single item
kira abandon 001 "Superseded by new approach"       # With reason
kira abandon done v2                                # Abandon folder
```

Behavior:
- Updates work item status to "abandoned" and archives the item(s)
- Archives to `.work/z_archive/{date}/{id}/` or `.work/z_archive/{date}/{original-path}/`
- Preserves folder structure for path/subfolder abandons
- Adds an "Abandonment" section with reason and timestamp when a reason is provided

### `kira save [commit-message]`
Updates work items and commits changes to git.

```bash
kira save                           # Use default message
kira save "Add user auth requirements"  # Custom message
```

Behavior:
- Validates all non-archived work items before staging; fails on validation errors
- Updates/creates the `updated:` timestamp in changed work items
- Stages only `.work/` changes; skips committing if external (non-.work) changes are detected
- Uses provided commit message or the configured default when none is given

### `kira latest`
Updates your branch with the latest trunk. Works on both trunk and feature branches.

```bash
kira latest                    # Stash (if needed), fetch, update; pop stash after
kira latest --no-pop-stash      # Stash but do not pop after successful update
kira latest --abort-on-conflict # On conflict, abort rebase/update and pop stash
```

Behavior:
- **On a feature branch**: Fetches and rebases your branch onto trunk.
- **On trunk**: Fetches and updates local trunk from remote (e.g. pull --rebase).
- Uncommitted changes are stashed before the update and popped after success (unless `--no-pop-stash`).
- In polyrepo setups, each repository is handled according to its own current branch.

### `kira version`
Prints version information embedded at build time (SemVer tag if present), commit, build date, and dirty state.

```bash
kira version
# Version: v0.1.0
# Commit: abc1234
# BuildDate: 2025-01-01T00:00:00Z
# State: clean
```

## Folder Structure

```
kira.yml          # Configuration
.work/
├── 0_backlog/    # Ideas being shaped
├── 1_todo/       # Ready to work on
├── 2_doing/      # Currently in progress
├── 3_review/     # Ready for review
├── 4_done/       # Completed work
├── templates/    # Work item templates
├── z_archive/    # Archived items
└── IDEAS.md      # Quick idea capture
```

## Work Item Types

- **PRD** (Product Requirements Document): Feature specifications
- **Issue**: Bug reports and problems
- **Spike**: Discovery and research tasks
- **Task**: Discrete implementation tasks

## Configuration

The `kira.yml` file controls the tool's behavior:

```yaml
version: "1.0"

templates:
  prd: "templates/template.prd.md"
  issue: "templates/template.issue.md"
  spike: "templates/template.spike.md"
  task: "templates/template.task.md"

status_folders:
  backlog: "0_backlog"
  todo: "1_todo"
  doing: "2_doing"
  review: "3_review"
  done: "4_done"
  archived: "z_archive"

  # Default status used when not specified in `kira new`
  default_status: "backlog"

validation:
  required_fields: ["id", "title", "status", "kind", "created"]
  id_format: "^\\d{3}$"
  status_values: ["backlog", "todo", "doing", "review", "done", "released", "abandoned", "archived"]
  strict: false  # If true, flag fields not defined in configuration

commit:
  default_message: "Update work items"

release:
  releases_file: "RELEASES.md"
  archive_date_format: "2006-01-02"

workspace:
  work_folder: ".work"  # optional; default is ".work"

### Custom work folder

By default, kira uses the `.work` directory for status folders, templates, and IDEAS.md. You can override this with `workspace.work_folder` in `kira.yml`. Examples: `work`, `tasks`, or a relative path like `../shared-work`. The path is resolved relative to the directory containing `kira.yml`. Existing repos that do not set `work_folder` continue to use `.work` (backward compatible).

# Field Configuration (optional)
# Default templates do not include due or estimate; add them here and in custom templates if needed.
# Define custom fields with validation rules, defaults, and metadata
fields:
  assigned:
    type: email
    required: false
    description: "Assigned user email address"
    default: ""

  priority:
    type: enum
    required: false
    allowed_values: [low, medium, high, critical]
    default: medium
    description: "Priority level"
    case_sensitive: false

  due:
    type: date
    required: false
    format: "2006-01-02"
    min_date: "today"  # Relative date: must be today or future
    description: "Due date"

  tags:
    type: array
    required: false
    item_type: string
    unique: true
    description: "Tags for categorization"

  estimate:
    type: number
    required: false
    min: 0
    max: 100
    description: "Estimate in days"

  epic:
    type: string
    required: false
    format: "^[A-Z]+-\\d+$"  # Regex pattern, e.g., EPIC-001
    description: "Epic identifier"

  url:
    type: url
    required: false
    schemes: [http, https]
    description: "Related URL"
```

## Field Configuration

Kira supports custom field definitions with validation rules, default values, and metadata. This allows teams to customize work items to match their workflow needs.

### Field Types

- **string**: Plain text with optional regex format validation
- **date**: Date field with format validation and min/max date constraints
- **email**: Email address validation
- **url**: URL validation with optional scheme restrictions
- **number**: Numeric values with min/max constraints
- **array**: Arrays of values with item type validation and uniqueness constraints
- **enum**: Predefined list of allowed values with optional case sensitivity

### Field Configuration Options

- **type**: Field data type (required)
- **required**: Whether the field is required (default: false)
- **default**: Default value for the field
- **format**: Regex pattern for strings or date format for dates
- **allowed_values**: List of allowed values for enum types
- **description**: Human-readable description
- **min/max**: Numeric or date range constraints
- **min_length/max_length**: String or array length constraints
- **item_type**: Type of array items (string, number, enum)
- **unique**: Whether array values must be unique
- **schemes**: Allowed URL schemes (http, https, etc.)
- **case_sensitive**: Whether enum values are case-sensitive (default: true)

### Strict Mode

Strict mode flags fields in work items that are not defined in the configuration. This helps maintain consistency and catch typos.

Enable strict mode:
- **In configuration**: Set `validation.strict: true` in `kira.yml`
- **Via CLI flag**: Use `--strict` flag with `kira lint` or `kira doctor`

```bash
# Enable strict mode via CLI
kira lint --strict

# Or set in kira.yml
validation:
  strict: true
```

**Note**: The fields `id`, `title`, `status`, `kind`, and `created` are hardcoded and cannot be configured. These fields are managed by other configuration systems.

### Example Field Configuration

```yaml
fields:
  assigned:
    type: email
    required: true
    description: "Assigned user email address"

  priority:
    type: enum
    allowed_values: [low, medium, high, critical]
    default: medium
    case_sensitive: false

  due:
    type: date
    format: "2006-01-02"
    min_date: "today"

  tags:
    type: array
    item_type: string
    unique: true
    min_length: 1
    max_length: 10
```

## Work Item Format

Work items are markdown files with YAML front matter. The default template includes `id`, `title`, `status`, `kind`, `created`, `assigned`, and `tags`. Optional fields such as `due` and `estimate` can be added via `kira.yml` `fields:` and custom templates.

```yaml
---
id: 001
title: User Authentication Feature
status: todo
kind: prd
assigned: user@example.com
priority: high
created: 2024-01-15
tags: [auth, security, frontend]
---

# User Authentication Feature

## Context
Background and rationale...

## Requirements
Functional requirements...

## Acceptance Criteria
- [ ] User can log in with email/password
- [ ] User can log out
- [ ] Session is maintained across page refreshes
```

## Git Integration

Kira is designed to work seamlessly with git:

- All work items are tracked in git
- The `kira save` command commits only `.work/` changes
- External changes are detected and prevent accidental commits
- Full transparency through git history

## License

MIT License - see LICENSE file for details.

## Contributing

See CONTRIBUTING.md for the full contributor workflow, development setup, and build/install details.

