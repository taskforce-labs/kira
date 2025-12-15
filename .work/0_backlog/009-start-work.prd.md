---
id: 009
title: start work
status: backlog
kind: prd
assigned:
estimate: 0
created: 2025-12-04
due: 2025-12-04
tags: []
---

# start work

```bash
kira start <work-item-id>
```
- Pulls latest changes from origin on trunk branch (configured or auto-detected: main/master) (and all project repos for polyrepo)
- Optionally changes the work item status to the configured status folder (defaults to "doing", configurable via `start.move_to`). Behavior is controlled by `start.status_action` configuration (defaults to `commit_and_push`, which updates status, commits, and pushes to trunk branch)
- Creates a git worktree from trunk branch `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}` (see "Worktree Location" section for default derivation)
- If workspace configuration exists in kira.yml, creates coordinated worktrees for all projects in the workspace
- Creates a branch from the worktree: `{work-item-id}-{kebab-case-work-item-title}`
- Opens the IDE for the worktree(s) with the correct branch checked out (unless `--no-ide` flag is provided, or if `ide.command` is configured in kira.yml or `--ide <command>` flag is provided)


## Context

Agentic workflows have fundamentally changed how development teams work. Developers and AI agents now collaborate on multiple tasks simultaneously, requiring isolated workspaces for each work item to prevent conflicts and maintain clean separation of concerns.

Currently, developers must manually:
1. Create git worktrees for parallel work
2. Create branches with appropriate naming
3. Navigate to the correct directory
4. Open their IDE in the right context
5. Repeat this process for each related project in a monorepo or polyrepo setup

This manual process is error-prone, time-consuming, and doesn't scale well when working with multiple agents or parallel tasks. The `kira start` command automates this workflow, allowing developers and agents to quickly spin up isolated work environments with a single command.

Git worktrees enable multiple working directories attached to the same repository, each checked out to different branches. This is ideal for agentic workflows where:
- Multiple agents work on different tasks simultaneously
- Developers need to context-switch between tasks quickly
- Code reviews require separate checkouts
- Parallel development streams need isolation

When working with multi-project workspaces (monorepo or polyrepo configurations), the command creates coordinated worktrees across all projects, ensuring consistent branch naming and workspace organization. The workspace type determines how projects are organized, but the core workflow remains the same regardless of type.

## Requirements

### Core Functionality

1. **Work Item Lookup**
   - Accept a work item ID as a required argument
   - Locate the work item file in the `.work/` directory structure
   - Extract work item metadata (ID, title, status)
   - **Handle missing title:** If title is empty or "unknown", use work item ID as fallback and log warning
   - Validate that the work item exists and is accessible

2. **Git Worktree Creation**
   - Detect the current git repository root
   - Determine the trunk branch:
     - Use `git.trunk_branch` if configured
     - Otherwise auto-detect: check for "main" first, then "master"
     - If both "main" and "master" branches exist: fail with error: "Error: Both 'main' and 'master' branches exist. Cannot auto-detect trunk branch. Configure `git.trunk_branch` explicitly in kira.yml to specify which branch to use."
     - Error if trunk branch not found: "Error: Trunk branch '{branch-name}' not found. Configured branch does not exist and auto-detection failed. Verify the branch name in `git.trunk_branch` configuration or ensure 'main' or 'master' branch exists."
   - **Pull latest changes from origin** to ensure trunk branch is up-to-date before any operations:
     - Determine remote name: Use `git.remote` if configured, otherwise default to "origin"
     - Check if remote exists using `git remote get-url <remote-name>`; if command fails (no remote configured):
       - Log warning: "Warning: No remote '{remote-name}' configured. Skipping pull step. Worktree will be created from local trunk branch."
       - Skip pull step, continue with worktree creation
     - For polyrepo: For each project, determine remote name: Use `project.remote` if configured, otherwise use `git.remote` (or "origin" if `git.remote` is also omitted)
       - Check each project repository independently using its configured remote name
       - Skip pull for projects without remote, log warning per project with the remote name that was checked
     - Check for uncommitted changes in trunk branch; if found, abort with error: "Error: Trunk branch has uncommitted changes. Cannot proceed with pull operation. Commit or stash changes before starting work."
     - Checkout trunk branch (configured or auto-detected)
     - Run `git fetch <remote-name> <trunk_branch>` to fetch latest changes (where remote-name is from `git.remote` or "origin" default)
     - Run `git merge <remote-name>/<trunk_branch>` to merge remote changes into local trunk branch
     - If merge fails due to conflicts: abort with error showing git output
     - If merge fails due to diverged branches (local commits not on remote): abort with error: "Error: Trunk branch has diverged from {remote-name}/{trunk-branch}. Local and remote branches have different commits. Rebase or merge manually before starting work."
     - If network error occurs: abort with error showing git output
     - For polyrepo: perform above steps for all project repositories sequentially; abort entire command if any project's pull fails
   - Create a new git worktree from the trunk branch
   - Use consistent naming convention for worktree directory: `{work-item-id}-{kebab-case-work-item-title}`
   - Place worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}` (see "Worktree Location" section for defaults)
   - Abort if worktree creation fails (e.g., invalid path, permissions, etc.)

3. **Branch Creation**
   - Create a new branch in the worktree using sanitized work item title
   - Use consistent naming convention: `{work-item-id}-{kebab-case-work-item-title}`
   - Ensure branch name is valid for git (see "Title Sanitization" section in Implementation Notes for details on sanitization and length limits)

4. **Multi-Project Workspace Support**
   - Read `kira.yml` configuration for workspace configuration
   - Automatically infer workspace behavior (see "Workspace Behavior Inference" section for details)
   - Support three workflow types:
     - **Standalone:** Single repository (no workspace config)
     - **Monorepo:** Single repository with projects list for LLM context
     - **Polyrepo:** Multiple repositories with optional `repo_root` grouping (for monorepos or nested folder structures)
   - Create worktrees and branches based on inferred behavior (see "Implementation Notes" for details)

5. **IDE Integration**
   - **Configuration priority:** Check flags in order: `--no-ide` (highest), then `--ide <command>`, then `ide.command` from `kira.yml`
   - **`--no-ide` flag:** If `--no-ide` flag is provided:
     - Skip IDE opening entirely (useful for agents or CI/CD environments)
     - Ignore `--ide` flag and `ide.command` config (flag takes precedence)
     - No log messages about IDE (silently skip)
   - **`--ide` flag override:** If `--ide <command>` flag is provided (and `--no-ide` not set):
     - Use the flag value as IDE command (overrides `ide.command` from config)
     - Ignore `ide.args` from config (no args are passed to the IDE command)
     - Execute: `{flag-command} {worktree-path}` (no args)
   - **Config-based:** If neither flag provided, use `ide.command` from `kira.yml` - no auto-detection
   - If no IDE command found (flag or config): Skip IDE opening, log info message "Info: No IDE configured. Worktree created at {path}. Configure `ide.command` in kira.yml or use `--ide <command>` flag to automatically open IDE."
   - For standalone or monorepo: Open IDE in the worktree directory (single repository)
   - For polyrepo: Open IDE at the worktree root (`{worktree_root}/{work-item-id}-{kebab-case-work-item-title}`) with all project worktrees
   - IDE opens before setup commands run (allows user to start working while setup runs in background)

6. **Setup Commands/Scripts**
   - **Main project setup:** If `workspace.setup` is configured in `kira.yml`:
     - Execute setup command/script in main project worktree directory
     - For standalone/monorepo: Run in `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}`
     - For polyrepo: Run in main project worktree (if main project has its own worktree)
   - **Project-specific setup:** For each project in polyrepo with `project.setup` configured:
     - Execute setup command/script in that project's worktree directory
     - Run in the worktree path where the project is located
   - **Execution details:**
     - If setup is a script path: Execute with shell (e.g., `bash ./scripts/setup.sh`)
     - If setup is a command: Execute directly (e.g., `docker compose up -d`)
     - Run in the worktree directory context (change directory before execution)
     - If setup fails: Log error and abort command (setup is critical for environment preparation)
     - For polyrepo: Run setups sequentially in order projects are defined
     - Setup runs after IDE opening (allows user to start working while setup runs)
   - Ensure the correct branch is checked out in all worktrees when IDE opens
   - **Handle IDE already open:** Behavior is controlled by `ide.args` configuration (when using config, not flag):
     - Use configured `ide.args` from `kira.yml` (e.g., `["--new-window"]` to open in new window)
     - If IDE launch fails: Log warning "Warning: IDE launch failed. Worktree created successfully. You can manually open the IDE at {path}."
     - All IDE-specific behavior is configured via `ide.command` and `ide.args` - no hardcoded IDE logic

6. **Status Management**
   - **Status Check:** Executes as step 5 in Command Execution Order, after git pull (step 4) and before worktree creation (step 7)
     - This check happens after pulling latest changes to ensure we're checking the most up-to-date work item status
     - If work item status matches configured `start.move_to` (defaults to "doing") and `--skip-status-check` flag is not provided:
       - Fail with error: "Error: Work item {id} is already in '{move_to}' status. Work item status matches target status. Use --skip-status-check to restart work or review elsewhere." (where {move_to} is the configured value)
       - Abort command before worktree creation (step 7)
     - If work item status matches configured `start.move_to` and `--skip-status-check` flag is provided:
       - Skip status update step (step 6), proceed directly to worktree creation (step 7)
       - Allows resuming work or reviewing work item in different location
     - If work item status does not match configured `start.move_to`: Continue with normal status update flow (step 6)
   - **Status Update:** Optionally move work item to configured status folder (`start.move_to`, defaults to "doing") (controlled by `start.status_action`)
     - **File Location:** Reuses the `moveWorkItem` function from `move.go` (with `commitFlag=false`) to ensure consistency:
       - The function handles finding the work item file, validating target status, moving the file to `.work/{status_folder}/` directory, and updating the status field in frontmatter
       - Since `kira start` has different commit behavior (commit_only, commit_and_push, commit_only_branch), we call `moveWorkItem` with `commitFlag=false` and handle commits separately
       - This ensures consistency with the `kira move` command behavior and avoids code duplication
     - **Important - File Location for `commit_only_branch`:**
       - **Status update happens in the original repository:** The work item file is physically moved within the original repository's `.work/` directory structure (e.g., from `.work/0_backlog/` to `.work/1_doing/`)
       - **File remains in original `.work/` directory:** The work item file stays in the original repository's `.work/` directory structure, not in the worktree directory
       - **Worktree is a separate checkout:** The worktree is a separate checkout of the same repository, so it sees the same `.work/` directory structure (worktrees don't have their own separate `.work/` directory)
       - **Commit happens in worktree context:** For `commit_only_branch`, the commit is made in the worktree (on the new branch), but the file being committed is the same file in the original repository's `.work/` directory that the worktree can see
   - If `status_action` is not `none`: Update status and optionally commit/push
   - Configuration controls whether to update status and how to persist the change

### Configuration Requirements

1. **Workspace Configuration**

   The workspace configuration is optional. If not present, kira assumes a **standalone** workspace (single repository). All configuration values default to standalone behavior when omitted.

   **Default Behavior (Standalone):**
   - Single repository workflow
   - Worktree created at: `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}` (see "Worktree Location" section for defaults)
   - Branch created: `{work-item-id}-{kebab-case-work-item-title}`
   - IDE opens in worktree directory
   - No draft PR configuration (defaults to false)

   **Monorepo**
   - Multiple projects in a single repository
   - **Worktree and branching behavior identical to Standalone**
   - Detected by the presence of a `workspace.projects` list in kira.yml with no `repo_root` field on any project
   - Worktree: `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}` (same as standalone, see "Worktree Location" section for defaults)
   - Branch: `{work-item-id}-{kebab-case-work-item-title}` (same as standalone)
   - IDE opens in worktree directory (same as standalone)
   - **Purpose:** The monorepo configuration (`projects` list) is primarily for LLM context to understand how the monorepo is structured and help with planning. It does not change worktree or branching behavior.

   **c) Polyrepo**
   - Multiple separate repositories
   - **Default behavior:** Creates separate worktrees for each repository
   - Worktree per project: `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{project.mount}/` (see "Worktree Location" section for `worktree_root` defaults)
   - Branch: `{work-item-id}-{kebab-case-work-item-title}` (same branch name in each repo)
   - **Repo Root Grouping:** Projects sharing the same `repo_root` are grouped into a single worktree:
     - **Purpose 1 - Monorepo grouping:** When multiple projects share the same root directory (e.g., a monorepo), they share one worktree
     - **Purpose 2 - Nested folder structures:** When repos are configured in nested folder structures, `repo_root` specifies the common root directory
     - Create ONE worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{sanitized-repo-root}/`
     - Create ONE branch in that worktree: `{work-item-id}-{kebab-case-work-item-title}`
     - All projects with the same `repo_root` share that worktree and branch
     - Skip worktree creation for subsequent projects with the same `repo_root`
   - IDE opens at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}` with all project worktrees

   **Workspace Behavior Inference:**

   Kira automatically infers workspace behavior from the configuration - no `type` field needed:

   1. **Standalone (default):**
      - No `workspace` section OR no `workspace.projects` list
      - Behavior: Single repository workflow
      - Creates: 1 worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}` (see "Worktree Location" section for defaults)
      - IDE opens: In worktree directory

   2. **Monorepo (LLM context only):**
      - `workspace.projects` exists but projects have NO `path` fields
      - OR `workspace.projects` exists with `path` fields pointing to subdirectories within the current repository
      - Behavior: Single repository workflow (same as standalone)
      - Creates: 1 worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}` (see "Worktree Location" section for defaults)
      - IDE opens: In worktree directory
      - Purpose: Projects list provides LLM context about monorepo structure
      - **Detection:** Check if paths are subdirectories of current repo (not separate git repositories)

   3. **Polyrepo (multi-repository):**
      - `workspace.projects` exists with ANY project having `repo_root` field → **Polyrepo** (immediate detection)
      - OR `workspace.projects` exists with `path` fields pointing to separate git repositories
      - Behavior: Multi-repository workflow
      - Creates: Multiple worktrees at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/` (grouped by `repo_root` if present)
      - IDE opens: At `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}` (worktree root)
      - Projects sharing `repo_root`: Grouped into single worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{sanitized-repo-root}/`
      - Projects without `repo_root`: Each gets own worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{project.mount}/`
      - See "Worktree Location" section for `worktree_root` defaults
      - **Detection:**
        - First check: If ANY project has `repo_root` → polyrepo (immediate)
        - Otherwise: Check if each path is a separate git repository (contains `.git` directory)

   **Configuration Schema:**
   ```yaml
   workspace:
     # Optional: defaults to "../" relative to current repo
     root: ../                          # where canonical repos live

    # Optional: defaults derived from project structure if omitted (see "Worktree Location" section for detailed algorithm and examples)
    # You can override the default derivation by explicitly specifying a custom path:
    worktree_root: ../my-project_worktrees       # where generated worktrees go (overrides default derivation)

     # Optional: workspace-level architecture documentation
     architecture_doc: ./ARCHITECTURE.md

     # Optional: workspace description for LLM context
     description: >
       Defines how services in this workspace relate and provides
       structured metadata for tooling and AI agents.

    # Optional: defaults to false (no draft PRs)
    draft_pr: false                    # workspace-level default for draft PRs

    # Optional: setup command/script for main project (repository where kira.yml is located)
    # Can be a command string (e.g., "docker compose up -d") or path to script (e.g., "./scripts/setup.sh")
    # Executed in the worktree directory after IDE opening (allows user to start working while setup runs)
    setup: docker compose up -d       # optional: command or script path for main project setup

    # Optional: projects list
     # - If projects have `path` fields → polyrepo behavior (multi-repo)
     # - If projects have no `path` fields → monorepo behavior (LLM context only)
     # - If no projects list → standalone behavior (default)
     projects:
       - name: frontend                 # project identifier
         path: ../monorepo/frontend     # path to project repository
         mount: frontend                # folder name in worktree (defaults to name)
         repo_root: ../monorepo         # optional: shared root directory (groups projects sharing same root, handles nested folder structures)
         kind: app                      # app | service | library | infra (optional)
         description: >                 # optional: for LLM context
           Customer-facing application responsible for UI and orchestration.
        draft_pr: true                 # optional: override workspace default
        setup: npm install              # optional: command or script path for project-specific setup

      - name: backend
         path: ../monorepo/backend
         mount: backend
         repo_root: ../monorepo     # same root as frontend = grouped repo root
         kind: service
         draft_pr: false

       - name: orders-service
         path: ../orders-service
         mount: orders                  # different mount name than project name
         # No repo_root = standalone repository (gets own worktree)
         kind: service
        draft_pr: false                # explicit override (uses workspace default)
        remote: upstream               # optional: override remote name for this project (defaults to "origin" or git.remote)
        setup: ./scripts/setup.sh      # optional: command or script path for project-specific setup
   ```

   **Configuration Defaults:**
   - If `git` section is omitted: the `trunk_branch` config field defaults to auto-detect (check "main" first, then "master")
   - If `git.trunk_branch` is omitted or empty: auto-detect (check "main" first, then "master")
   - If both "main" and "master" branches exist: fail with error: "Error: Both 'main' and 'master' branches exist. Cannot auto-detect trunk branch. Configure `git.trunk_branch` explicitly in kira.yml to specify which branch to use."
   - If `start` section is omitted: `status_action` defaults to `commit_and_push` (meaning status is updated, committed, and pushed to trunk branch)
   - If `start.move_to` is omitted or empty: defaults to `"doing"`
   - If `start.status_action` is omitted: defaults to `commit_and_push` (meaning status is updated, committed, and pushed to trunk branch)
   - If `start.status_commit_message` is omitted: defaults to `"Move {type} {id} to {move_to}"` (uses configured `move_to` value, or "doing" if not configured)
   - If `workspace` section is omitted: treated as standalone (default behavior)
   - **Workspace behavior inference:**
     - If `workspace.projects` is omitted: standalone behavior
     - If `workspace.projects` exists:
       - If ANY project has `repo_root`: polyrepo behavior (multi-repo, immediate detection)
       - Otherwise, check if projects point to separate git repositories:
         - If projects have `path` fields pointing to separate git repositories: polyrepo behavior (multi-repo)
         - If projects have no `path` fields OR paths point to subdirectories within current repo: monorepo behavior (LLM context)
  - If `root` is omitted: defaults to `../` (parent directory)
  - If `worktree_root` is omitted: defaults are derived from project structure (see "Worktree Location" section for details; can be overridden by specifying `worktree_root`)
   - If `draft_pr` is omitted at workspace level: defaults to `false`
   - If `draft_pr` is omitted at project level: uses workspace-level default
   - If `mount` is omitted: defaults to project `name`
   - If `repo_root` is omitted: project is standalone (gets own worktree)
   - If `repo_root` is present: project is grouped with others sharing the same root (monorepo grouping) or handles nested folder structures
   - If `path` is omitted in project: project describes current repo structure (monorepo context)
   - If `kind` is omitted: no special behavior applied
   - If `description` is omitted: no LLM context provided

   **How Workspace Behavior is Determined:**

   **Standalone (no config or no projects list):**
   ```
   Current repo: /path/to/my-project
   Worktree:     /path/to/{work-item-id}-{kebab-case-work-item-title}
   Branch:       {work-item-id}-{kebab-case-work-item-title}
   IDE opens:    /path/to/{work-item-id}-{kebab-case-work-item-title}

   Inference: No workspace.projects → standalone behavior
   ```

   **Monorepo (projects list describing current repo structure):**
   ```yaml
   workspace:
     projects:
       - name: frontend
         path: ./apps/frontend        # subdirectory within current repo
         kind: app
         description: Customer-facing UI
       - name: backend
         path: ./services/backend     # subdirectory within current repo
         kind: service
         description: API service layer
   ```
   ```
   Current repo: /path/to/monorepo
   Worktree:     /path/to/{work-item-id}-{kebab-case-work-item-title}  (same as standalone)
   Branch:       {work-item-id}-{kebab-case-work-item-title}          (same as standalone)
   IDE opens:    /path/to/{work-item-id}-{kebab-case-work-item-title}  (same as standalone)

   Inference: Projects have `path` fields pointing to subdirectories within current repo → monorepo behavior (LLM context)
   Detection: Check if paths are subdirectories of current repo (not separate git repositories)
   Note: The projects configuration provides LLM context about monorepo structure:
   - projects[].name: identifies subdirectories/components
   - projects[].path: identifies subdirectory location (optional)
   - projects[].kind: categorizes components (app/service/library)
   - projects[].description: explains component purpose
   This helps LLMs understand the monorepo layout for planning and context.
   ```

   **Polyrepo (projects list with path fields):**
   ```yaml
   workspace:
     projects:
       - name: frontend
         path: ../frontend
       - name: orders-service
         path: ../orders-service
   ```
   ```
   Projects: ../frontend, ../orders-service
   Common parent: /path/to (detected from project paths)
   Worktree root: /path/to/proj_worktrees/{work-item-id}-{kebab-case-work-item-title}
   ├── frontend/          (separate git worktree from ../frontend repo)
   ├── orders-service/    (separate git worktree from ../orders-service repo)
   └── ...
   Branch: {work-item-id}-{kebab-case-work-item-title} (same branch name created in each separate repo)
   IDE opens: /path/to/proj_worktrees/{work-item-id}-{kebab-case-work-item-title}

   Inference: Projects have `path` fields pointing to separate git repositories → polyrepo behavior
   Detection: Check if each path is a separate git repository (contains `.git` directory)
   Note: See "Worktree Location" section for worktree_root defaults
   ```

   **Polyrepo with Repo Root Grouping:**
   ```yaml
   workspace:
     projects:
       - name: frontend
         path: ../monorepo/frontend
         repo_root: ../monorepo
       - name: backend
         path: ../monorepo/backend
         repo_root: ../monorepo
       - name: orders-service
         path: ../orders-service
         # No repo_root
   ```
   ```
   Projects: ../monorepo/frontend, ../monorepo/backend, ../orders-service
   Common parent: /path/to (detected from project paths)
   Worktree root: /path/to/proj_worktrees/{work-item-id}-{kebab-case-work-item-title}
   ├── monorepo/          (ONE worktree at repo_root, shared by frontend + backend)
   │   ├── frontend/      (component within monorepo worktree)
   │   └── backend/       (component within monorepo worktree)
   ├── orders-service/    (separate git worktree - standalone)
   └── ...
   Branch: {work-item-id}-{kebab-case-work-item-title} (one branch in monorepo worktree, one branch in orders-service worktree)
   IDE opens: /path/to/proj_worktrees/{work-item-id}-{kebab-case-work-item-title}

   Inference: Projects have `path` fields → polyrepo behavior
   Grouping: Projects sharing `repo_root` → grouped into single worktree
   Note: See "Worktree Location" section for worktree_root defaults
   ```

   **Draft PR Configuration:**
   - `draft_pr` at workspace level sets default for all projects
   - `draft_pr` at project level overrides workspace default
   - Used when creating pull requests (future enhancement)
   - Defaults to `false` if not specified

2. **Git Configuration**

   The `git` configuration controls repository-level git settings used across kira commands.

   ```yaml
   git:
     # Optional: trunk branch name (defaults to auto-detect: "main" first, then "master")
     # Used for pulling, committing status changes, creating worktrees, and other git operations
     # Can be set to any branch name (e.g., "develop", "trunk", "production")
     trunk_branch: main

     # Optional: remote name (defaults to "origin")
     # Used for pulling latest changes and pushing commits
     # Useful when repository uses a non-standard remote name (e.g., "upstream", "github", "gitlab")
     remote: origin
   ```

3. **Status Management Configuration**

   The `start` configuration controls how work item status changes are handled when starting work.

   ```yaml
   start:
     # Optional: status folder to move work item to (defaults to "doing")
     # Specifies which status folder the work item should be moved to when starting work
     move_to: doing  # e.g., "doing", "in-progress", "active", "working", etc.

     # Optional: defaults to "commit_and_push" (meaning status is updated, committed, and pushed to trunk branch)
     # Options: none | commit_only | commit_and_push | commit_only_branch
     # - none: Don't update work item status
     # - commit_only: Update status to configured status (move_to) and commit on trunk branch, don't push
     # - commit_and_push: Update status to configured status (move_to), commit on trunk branch and push to origin
     # - commit_only_branch: Update status to configured status (move_to) and commit on the new branch (after worktree creation)
     # Commit happens BEFORE worktree creation (except commit_only_branch)
     status_action: commit_and_push

     # Optional: custom commit message template for status update
     # Default: "Move {type} {id} to {move_to}" (uses configured move_to value)
     status_commit_message: "Start work on {type} {id}: {title}"
   ```

   **Behavior:**
   - If `status_action: none`: Work item status is not changed, proceed directly to worktree creation
   - If `status_action: commit_only`:
     1. Call `moveWorkItem(cfg, workItemID, move_to, false)` from `move.go` (with `commitFlag=false`) to move file to `.work/{status_folder}/` and update status field
     2. Commit change on trunk branch (configured or auto-detected)
     3. Then create worktree and branch
   - If `status_action: commit_and_push`:
     1. Call `moveWorkItem(cfg, workItemID, move_to, false)` from `move.go` (with `commitFlag=false`) to move file to `.work/{status_folder}/` and update status field
     2. Commit change on trunk branch (configured or auto-detected)
     3. Push to origin
     4. Then create worktree and branch
   - If `status_action: commit_only_branch`:
     1. Create worktree and branch
     2. Call `moveWorkItem(cfg, workItemID, move_to, false)` from `move.go` (with `commitFlag=false`) to move file to `.work/{status_folder}/` and update status field
     3. Commit change on the new branch (not trunk branch)
     - **Important - File Location Clarification:**
       - **Status update happens in the original repository:** The work item file is physically moved within the original repository's `.work/` directory structure (e.g., from `.work/0_backlog/` to `.work/1_doing/`)
       - **File remains in original `.work/` directory:** The work item file stays in the original repository's `.work/` directory structure, not in the worktree
       - **Worktree is a separate checkout:** The worktree is a separate checkout of the same repository, so it sees the same `.work/` directory structure (worktrees don't have their own separate `.work/` directory)
       - **Commit happens in worktree context:** The commit is made in the worktree (on the new branch), but the file being committed is the same file in the original repository's `.work/` directory that the worktree can see

4. **Pull Request Configuration**

   The `draft_pr` configuration controls whether pull requests created for branches should be marked as draft. This is a workspace-level default that can be overridden per project.

   ```yaml
   workspace:
     draft_pr: false              # workspace default
     projects:
       - name: frontend
         draft_pr: true            # override: always create draft PRs
       - name: backend
         # uses workspace default (false)
   ```

5. **IDE Detection**
   - **Configuration priority:** Check flags in order: `--no-ide` (highest), then `--ide <command>`, then `ide.command` from `kira.yml`
   - **`--no-ide` flag:** If `--no-ide` flag is provided:
     - Skip IDE opening entirely (useful for agents or CI/CD environments)
     - Ignore `--ide` flag and `ide.command` config (flag takes precedence)
     - No log messages about IDE (silently skip)
   - **`--ide` flag override:** If `--ide <command>` flag is provided (and `--no-ide` not set):
     - Use flag value as IDE command (overrides `ide.command` from config)
     - Ignore `ide.args` from config (execute command with no args)
     - Execute: `{flag-command} {worktree-path}` (no args passed)
   - **Config-based:** If neither flag provided, `ide.command` must be configured in `kira.yml` - no auto-detection
   - If no IDE command found (flag or config): Skip IDE opening, log info message, continue with worktree creation
   - **Configuration-driven:** All IDE behavior is controlled via `kira.yml`:
     ```yaml
     ide:
       command: "cursor"              # IDE command name (required if --ide flag not used)
       args: ["--new-window"]         # Arguments to pass to IDE command (ignored if --ide flag used)
     ```
   - **IDE Already Open Behavior:**
     - When using config: Behavior is entirely controlled by `ide.args` configuration
     - When using `--ide` flag: No args are passed (user can rely on IDE's default behavior or shell expansion)
     - User configures appropriate args for their IDE (e.g., `["--new-window"]` to open in new window, or IDE-specific flags)
     - If IDE launch fails: Log warning and continue (worktree creation succeeds)
     - No hardcoded IDE-specific logic - all behavior comes from configuration or flag

6. **Worktree Location**

   Worktree location is configured via `workspace.worktree_root`. This is **always used** for all workspace types (standalone, monorepo, and polyrepo).

   **Worktree Location Defaults:**

   If `workspace.worktree_root` is not specified in `kira.yml`, defaults are automatically derived based on workspace type:

   - **Standalone/Monorepo:**
     - **Algorithm:**
       1. Determine `repoRoot`: Absolute path to the repository root directory where `kira.yml` is located (obtained via `git rev-parse --show-toplevel` or equivalent)
       2. Extract project name: `projectName = filepath.Base(repoRoot)`
       3. Construct worktree root: `worktreeRoot = filepath.Join(filepath.Dir(repoRoot), projectName + "_worktrees")`
       4. Convert to relative path: If `repoRoot` is `/Users/dev/my-project/`, result is `../my-project_worktrees/` (relative to repo root)
     - **Result:** `../{project_name}_worktrees` where `{project_name}` is the basename of the repository root directory
     - **Path:** One level up from repository root
     - **Example:**
       - Repository root: `/Users/dev/my-project/`
       - `filepath.Base("/Users/dev/my-project/")` = `"my-project"`
       - Default worktree root: `/Users/dev/my-project_worktrees/` (absolute) or `../my-project_worktrees/` (relative)
     - **Worktree path:** `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}`
     - **Example:** `../my-project_worktrees/123-fix-bug/`

   - **Polyrepo:**
     - **Algorithm:**
       1. Collect all project paths: Extract all `project.path` values from configuration (resolve relative paths relative to `kira.yml` location)
       2. Convert to absolute paths: Resolve each project path to its absolute path
       3. **Common prefix detection:**
          - Start with the first project's absolute path as the initial candidate prefix
          - For each subsequent project path:
            - Compare character by character until paths diverge
            - Update candidate prefix to the common portion up to the last directory separator (`/`)
            - Continue until all project paths are processed
          - Result: `commonPrefix` = longest common path prefix ending at a directory boundary
       4. **Determine worktree root:**
          - If `commonPrefix` is non-empty and valid:
            - Extract parent directory name: `parentDir = filepath.Base(commonPrefix)`
            - Construct worktree root: `filepath.Join(filepath.Dir(commonPrefix), parentDir + "_worktrees")`
          - **Edge case:** If no common prefix exists (projects in completely different directory trees):
            - Use first project's parent directory: `firstProjectPath = absolutePath(projects[0].path)`
            - Extract parent: `parentDir = filepath.Base(filepath.Dir(firstProjectPath))`
            - Construct worktree root: `filepath.Join(filepath.Dir(filepath.Dir(firstProjectPath)), parentDir + "_worktrees")`
       5. Convert to relative path: Resolve relative to repository root where `kira.yml` is located
     - **Result:** `../../{parent_dir}_worktrees` where `{parent_dir}` is the basename of the common parent directory containing all projects
     - **Path:** Two levels up from repository root (or from common parent if projects share a parent)
     - **Common prefix detection examples:**
       - Projects: `/Users/dev/monorepo/frontend/`, `/Users/dev/monorepo/backend/`
       - Common prefix: `/Users/dev/monorepo/`
       - `filepath.Base("/Users/dev/monorepo/")` = `"monorepo"`
       - Default worktree root: `/Users/dev/monorepo_worktrees/` (absolute) or `../../monorepo_worktrees/` (relative)
     - **Edge case example:** If projects are at `/Users/dev/frontend/` and `/opt/backend/`:
       - No common prefix exists (paths share no common ancestor)
       - Use first project's parent directory: `filepath.Dir("/Users/dev/frontend/")` = `/Users/dev/`
       - Extract parent directory name: `filepath.Base("/Users/dev/")` = `"dev"`
       - Construct worktree root: `filepath.Join(filepath.Dir("/Users/dev/"), "dev_worktrees")` = `/Users/dev_worktrees/`
       - Default worktree root: `/Users/dev_worktrees/` (absolute) or `../../dev_worktrees/` (relative to repo root)
       - **Note:** In this edge case, worktrees are placed at the parent of the first project's parent directory, which may not be ideal. Consider explicitly setting `worktree_root` in configuration for projects in completely different directory trees.
     - **Worktree paths:**
       - Projects with same `repo_root`: `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{sanitized-repo-root}/`
       - Projects without `repo_root`: `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{project.mount}/`
     - **Example:** `../../monorepo_worktrees/123-fix-bug/monorepo/` (for grouped projects) or `../../monorepo_worktrees/123-fix-bug/frontend/` (for standalone project)

   **Override Mechanism:**

   You can override the default derivation by explicitly specifying `worktree_root` in your `kira.yml`:

   ```yaml
   workspace:
     worktree_root: ../my-custom-worktrees  # Overrides default derivation
   ```

   **Path Resolution:**

   - All paths are resolved relative to the repository root where `kira.yml` is located
   - Relative paths use `../` to go up directory levels
   - Absolute paths are supported but not recommended (reduces portability)
   - Paths are normalized using `filepath.Clean()` before use

   **Examples by Workspace Type:**

   - **Standalone:** Repository at `/Users/dev/my-app/`
     - Default: `../my-app_worktrees/`
     - Worktree: `../my-app_worktrees/123-fix-bug/`

   - **Monorepo:** Repository at `/Users/dev/monorepo/` (projects list for LLM context only)
     - Default: `../monorepo_worktrees/`
     - Worktree: `../monorepo_worktrees/123-fix-bug/` (same as standalone)

   - **Polyrepo:** Projects at `../services/frontend/` and `../services/backend/`
     - Common prefix: `../services/`
     - Default: `../../services_worktrees/`
     - Worktrees: `../../services_worktrees/123-fix-bug/frontend/` and `../../services_worktrees/123-fix-bug/backend/`

### Error Handling

**Error Message Format Standard:**

All error messages should follow a consistent format for clarity and actionability:

- **Format:** `"Error: {what}. {why}. {action}."`
- **Components:**
  - **{what}:** Clear description of what went wrong
  - **{why}:** Brief explanation of why it failed (if helpful)
  - **{action}:** Specific action the user can take to resolve the issue
- **Examples:**
  - `"Error: Work item '123' not found. No work item file exists with that ID. Check the work item ID and try again."`
  - `"Error: Both 'main' and 'master' branches exist. Cannot auto-detect trunk branch. Configure \`git.trunk_branch\` explicitly in kira.yml to specify which branch to use."`
  - `"Error: Trunk branch has uncommitted changes. Cannot proceed with worktree creation. Commit or stash changes before starting work."`

**Warning Message Format:**

Warning messages follow a similar format but use "Warning:" prefix and typically don't abort execution:

- **Format:** `"Warning: {what}. {why}. {action}."`
- **Examples:**
  - `"Warning: No remote 'origin' configured. Skipping pull step. Worktree will be created from local trunk branch."`
  - `"Warning: Work item {id} has no title field. Using work item ID '{id}' for worktree directory and branch name."`

1. **Validation Errors**
   - **Work item not found:** Error: `"Error: Work item '{id}' not found. No work item file exists with that ID. Check the work item ID and try again."`
   - **Work item missing title:** Warning: `"Warning: Work item {id} has no title field. Using work item ID '{id}' for worktree directory and branch name."` Use just work item ID as fallback, continue execution
   - **Invalid git repository:** Error: `"Error: Not a git repository. Current directory is not a git repository. Run this command from within a git repository."`
   - **Worktree path already exists:**
     - Check if target worktree path already exists
     - If path exists and is a valid git worktree:
       - Check if worktree is for the same work item (by checking if branch name matches `{work-item-id}-{kebab-case-work-item-title}` or if work item ID is in the path)
       - If same work item: Error: `"Error: Worktree already exists at {path} for work item {id}. A worktree for this work item already exists. Use \`--override\` to remove existing worktree and create a new one, or use the existing worktree."`
       - If different work item: Error: `"Error: Worktree path {path} already exists for a different work item. Path conflicts with existing worktree. Use \`--override\` to remove existing worktree, or choose a different work item."`
     - If path exists but is not a valid git worktree: Error: `"Error: Path {path} already exists but is not a valid git worktree. Cannot create worktree at this location. Remove it manually and try again, or use \`--override\` to remove it automatically."`
     - If path doesn't exist: Proceed with worktree creation
     - **With `--override` flag:** Remove existing worktree (using `git worktree remove` if valid git worktree) or directory before creating new one
   - **Branch already exists:**
     - Check if branch `{work-item-id}-{kebab-case-work-item-title}` already exists in repository before creating worktree
     - If branch exists:
       - Check what commit the branch points to
       - If branch points to trunk branch commit (same commit): Branch exists but has no commits, likely from previous worktree that was removed
         - Error: `"Error: Branch {branch-name} already exists and points to trunk. Branch exists but has no commits. Use \`--reuse-branch\` to checkout existing branch in new worktree, or delete the branch first: \`git branch -d {branch-name}\`"`
         - With `--reuse-branch` flag: Create worktree without `-b` flag, then checkout existing branch: `git worktree add <path> <trunk-branch>` followed by `git checkout {branch-name}`
       - If branch points to different commit (has commits): Branch has work
         - Error: `"Error: Branch {branch-name} already exists and has commits. Branch contains work that would be lost. Delete the branch first if you want to start fresh: \`git branch -D {branch-name}\`, or use a different work item."`
     - If branch doesn't exist: Create new branch using `git worktree add <path> -b <branch-name> <trunk-branch>`

2. **Git Errors**
   - **Trunk branch not found:** Error: `"Error: Trunk branch '{branch-name}' not found. Configured branch does not exist and auto-detection failed. Verify the branch name in \`git.trunk_branch\` configuration or ensure 'main' or 'master' branch exists."`
   - **Both "main" and "master" branches exist:** Error: `"Error: Both 'main' and 'master' branches exist. Cannot auto-detect trunk branch. Configure \`git.trunk_branch\` explicitly in kira.yml to specify which branch to use."`
   - **Remote not found:** Warning: `"Warning: No remote '{remote-name}' configured. Skipping pull step. Worktree will be created from local trunk branch."` (where remote-name is from `git.remote` or "origin" default), skip pull step, continue with worktree creation
   - **For polyrepo:** Each project uses its configured remote (`project.remote` or `git.remote` or "origin" default)
   - **For polyrepo:** If some projects have remote and others don't: Skip pull for projects without remote (log warning per project: `"Warning: No remote '{project-remote-name}' configured for project '{project-name}'. Skipping pull step."`), pull for projects with remote, continue with worktree creation
   - **Uncommitted changes in trunk branch (before pull):** Error: `"Error: Trunk branch has uncommitted changes. Cannot proceed with pull operation. Commit or stash changes before starting work."`
   - **Pull merge conflicts:** Error: `"Error: Failed to merge latest changes from {remote-name}/{trunk-branch}. Merge conflicts detected. Resolve conflicts manually and try again."` Include git output in error message
   - **Diverged branches (local commits not on remote):** Error: `"Error: Trunk branch has diverged from {remote-name}/{trunk-branch}. Local and remote branches have different commits. Rebase or merge manually before starting work."`
   - **Network errors during pull:** Error: `"Error: Failed to fetch changes from {remote-name}. Network error occurred. Check network connection and try again."` Include git output in error message
   - **Uncommitted changes in trunk branch (for commit_only/commit_and_push):** Error: `"Error: Trunk branch has uncommitted changes. Cannot commit status change. Commit or stash changes before starting work."`
   - **Invalid `status_action` value:** Error: `"Error: Invalid status_action value '{value}'. Value is not recognized. Use one of: 'none', 'commit_only', 'commit_and_push', 'commit_only_branch'."`
   - **Status commit failure:** Error: `"Error: Failed to commit status change. Git commit operation failed. Check git output for details and resolve any issues."` Include git output, abort worktree creation (for commit_only/commit_and_push)
   - **Status push failure:** Error: `"Error: Failed to push status change to {remote-name}/{trunk-branch}. Git push operation failed. Check git output for details and resolve any issues."` Include git output, abort worktree creation (for commit_and_push)
   - **Worktree creation failure:** Error: `"Error: Failed to create worktree at {path}. Git worktree creation failed. Check git output for details and resolve any issues."` Include git output
   - **Polyrepo worktree creation failure:** If any project's worktree creation fails, rollback all successfully created worktrees using `git worktree remove`, abort command with error: `"Error: Failed to create worktree for project '{project-name}'. Worktree creation failed for one or more projects. All worktrees have been rolled back. Check git output for details and resolve any issues."`
   - **Polyrepo branch creation failure:** If any project's branch creation fails, rollback all successfully created worktrees using `git worktree remove`, abort command with error: `"Error: Failed to create branch for project '{project-name}'. Branch creation failed for one or more projects. All worktrees have been rolled back. Check git output for details and resolve any issues."`
   - **Branch checkout failure (with `--reuse-branch`):** Error: `"Error: Failed to checkout branch '{branch-name}' in worktree. Branch checkout operation failed. Check git output for details and resolve any issues."` Include git output
   - **Missing project repository (polyrepo):** Error: `"Error: Project repository not found at {path}. Path does not exist or is not a git repository. Verify path exists and is a git repository, or update project configuration in kira.yml."` Abort entire command (all-or-nothing)

3. **Work Item Status Errors**
   - **Work item already in configured status:** Error: `"Error: Work item {id} is already in '{move_to}' status. Work item status matches target status. Use --skip-status-check to restart work or review elsewhere."` (where {move_to} is the configured value from `start.move_to`, defaults to "doing")
   - Error occurs during status check (step 5), after git pull (step 4) and before worktree creation (step 7)

4. **Setup Errors**
   - **Setup command/script not found:** Error: `"Error: Setup command/script not found: {setup}. File or command does not exist. Verify \`workspace.setup\` or \`project.setup\` configuration in kira.yml."`
   - **Setup execution failure:** Error: `"Error: Setup command failed: {error}. Setup command exited with error. Check command output for details and resolve any issues."` Include command output in error message
   - **Setup script execution permission denied:** Error: `"Error: Setup script is not executable: {script}. Script lacks execute permissions. Make script executable (\`chmod +x {script}\`) or use appropriate interpreter."`
   - **Setup timeout (if applicable):** Error: `"Error: Setup command timed out. Command did not complete within timeout period. Verify setup command completes successfully or increase timeout if needed."`

5. **IDE Errors** (Note: IDE issues are warnings, not errors - worktree creation succeeds regardless)
   - **`--no-ide` flag:** If `--no-ide` flag is provided, skip IDE opening silently (no errors, no warnings, no info messages)
   - **IDE not configured:** Info: `"Info: No IDE configured. Worktree created at {path}. Configure \`ide.command\` in kira.yml, use \`--ide <command>\` flag, or use \`--no-ide\` to skip IDE opening."` Continue without opening IDE
   - **IDE command not found:** Warning: `"Warning: IDE command '{command}' not found. Command does not exist in PATH. Verify \`--ide\` flag value or \`ide.command\` in kira.yml, or manually open IDE at {path}."` Continue without opening IDE
   - **IDE launch failure:** Warning: `"Warning: Failed to launch IDE. IDE command execution failed. Worktree created successfully. You can manually open the IDE at {path}."` Worktree still created
   - **IDE behavior:** All IDE behavior is controlled by flags (`--no-ide`, `--ide`) or `ide.command`/`ide.args` configuration - no automatic flag appending or IDE-specific logic
   - Worktree creation succeeds regardless of IDE behavior

### Command Syntax

```bash
# Basic usage (uses config defaults)
kira start <work-item-id>

# Override status action behavior (overrides config)
kira start <work-item-id> --status-action none              # Don't update status
kira start <work-item-id> --status-action commit_only      # Update status, commit only
kira start <work-item-id> --status-action commit_and_push  # Update status, commit and push
kira start <work-item-id> --status-action commit_only_branch # Update status, commit on branch

# Specify trunk branch (overrides git.trunk_branch config)
kira start <work-item-id> --trunk-branch develop

# Skip IDE opening (useful for agents or CI/CD environments - takes precedence over all IDE config)
kira start <work-item-id> --no-ide

# Override IDE command (overrides ide.command from config, ignores ide.args - no args passed)
kira start <work-item-id> --ide <command>
# Example: kira start 009 --ide cursor
# Example: kira start 009 --ide "$KIRA_IDE"  (shell expands $KIRA_IDE before passing to kira)

# Skip status check (allow starting work item already in "doing" status)
kira start <work-item-id> --skip-status-check

# Override existing worktree (remove existing worktree if it exists before creating new one)
kira start <work-item-id> --override

# Reuse existing branch (checkout existing branch in new worktree if branch exists and points to trunk)
kira start <work-item-id> --reuse-branch

# Dry run mode (preview what would be done without executing)
kira start <work-item-id> --dry-run
```

**Dry Run Mode (`--dry-run` flag):**

If `--dry-run` flag is provided, the command will:
- Preview what would be done without executing any operations
- Show what worktrees would be created (paths and locations)
- Show what branches would be created (names)
- Show what status changes would be made (current status → target status)
- Show what git operations would be performed (pull, commit, push)
- Show what setup commands would be executed
- Show what IDE would be opened (if configured)
- **Does NOT execute:**
  - No git operations (no pull, no worktree creation, no branch creation, no commits, no pushes)
  - No file moves or status updates
  - No IDE opening
  - No setup command execution
- Output format: Clear, structured preview showing all planned operations
- Exit code: 0 (success) if preview can be generated, non-zero if there are errors that would prevent execution

**Command Execution Order:**

**Note:** If `--dry-run` flag is provided, skip all execution steps (steps 4-10) and only perform validation (steps 1-3) and preview generation. Show what would be done without executing any operations.

1. Validate work item exists
2. **Infer workspace behavior** (see "Workspace Behavior Inference" section for detailed logic)
3. **Determine trunk branch:** (See "Git Operations" section in Implementation Notes for detailed algorithm and exact commands)
   - Detect trunk branch using configured value or auto-detection
   - Handle conflicts when both "main" and "master" exist
4. **Pull latest changes from remote:** (See "Git Operations" section in Implementation Notes for detailed pull strategy and exact command sequences)
   - Pull latest changes from remote on trunk branch (and all project repos for polyrepo)
   - Use `git fetch` + `git merge` approach (not `git pull`) for more control
   - Handle missing remotes, uncommitted changes, merge conflicts, and network errors
5. **Check work item status:**
   - Get configured status folder from `start.move_to` (defaults to "doing" if not configured)
   - If work item status already matches configured status and `--skip-status-check` flag is not provided:
     - Fail with error: "Work item {id} is already in '{move_to}' status. Use --skip-status-check to restart work or review elsewhere." (where {move_to} is the configured value)
   - If work item status matches configured status and `--skip-status-check` flag is provided:
     - Skip status update step, proceed directly to worktree creation
     - Allows resuming work or reviewing work item in different location
   - If work item status does not match configured status: Continue with normal flow
6. **If `status_action` is not `none`:** (See "Git Operations" section in Implementation Notes for detailed status management and commit operations)
   - Move work item to configured status folder (`start.move_to`, defaults to "doing")
   - If `status_action` is `commit_only` or `commit_and_push`: Commit (and optionally push) status change on trunk branch before worktree creation
7. **Create worktree(s) and branch(es):** (See "Git Operations" section in Implementation Notes for detailed worktree creation, branch existence checks, and exact command sequences)
   - For standalone/monorepo: Create single worktree and branch
   - For polyrepo: Transaction-like behavior - create all worktrees first (Phase 1), then create branches (Phase 2) for easier rollback
   - Handle existing worktrees and branches (with `--override` and `--reuse-branch` flags)
8. **If `status_action` is `commit_only_branch`:** (See "Git Operations" section in Implementation Notes for detailed status commit operations)
   - Move work item to configured status folder (`start.move_to`, defaults to "doing")
   - Commit status change on new branch (in worktree context)
9. Open IDE
   - Standalone/monorepo: Open in worktree directory (`{worktree_root}/{work-item-id}-{kebab-case-work-item-title}`, see "Worktree Location" section for defaults)
   - Polyrepo: Open at worktree root (`{worktree_root}/{work-item-id}-{kebab-case-work-item-title}`, see "Worktree Location" section for defaults)
10. Run setup commands/scripts (if configured):
   - **Main project setup:** If `workspace.setup` is configured:
     - Execute setup command/script in main project worktree directory
     - For standalone/monorepo: Run in `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}`
     - For polyrepo: Run in main project worktree (if main project has its own worktree)
   - **Project-specific setup:** For each project in polyrepo with `project.setup` configured:
     - Execute setup command/script in that project's worktree directory
     - Run in the worktree path where the project is located
   - **Execution details:**
     - If setup is a script path: Execute with shell (e.g., `bash ./scripts/setup.sh`)
     - If setup is a command: Execute directly (e.g., `docker compose up -d`)
     - Run in the worktree directory context (change directory before execution)
     - If setup fails: Log error and abort command (setup is critical for environment preparation)
     - For polyrepo: Run setups sequentially in order projects are defined
     - Setup runs after IDE opening (allows user to start working while setup runs)

## Acceptance Criteria

### Core Functionality

0. **Dry Run Mode**
   - [ ] `--dry-run` flag shows preview of what would be done without executing
   - [ ] Shows what worktrees would be created (paths and locations)
   - [ ] Shows what branches would be created (names)
   - [ ] Shows what status changes would be made (current status → target status from `start.move_to`)
   - [ ] Shows what git operations would be performed (pull, commit, push)
   - [ ] Shows what setup commands would be executed (if configured)
   - [ ] Shows what IDE would be opened (if configured)
   - [ ] Does NOT execute any git operations (no pull, no worktree creation, no branch creation, no commits, no pushes)
   - [ ] Does NOT move files or update status
   - [ ] Does NOT open IDE
   - [ ] Does NOT run setup commands
   - [ ] Output format is clear and structured
   - [ ] Exit code is 0 if preview can be generated, non-zero if there are errors that would prevent execution

1. **Work Item Resolution**
   - [ ] Command accepts work item ID and locates the corresponding file
   - [ ] Command fails with clear error if work item ID not found
   - [ ] Command extracts title and status from work item metadata correctly
   - [ ] If title is missing or "unknown": Uses work item ID as fallback, logs warning "Warning: Work item {id} has no title field. Using work item ID '{id}' for worktree directory and branch name."
   - [ ] Command checks work item status after git pull (step 4) and before worktree creation (step 7) - executes as step 5
   - [ ] Command fails with clear error if work item is already in configured status (`start.move_to`, defaults to "doing") (unless `--skip-status-check` flag)
   - [ ] `--skip-status-check` flag allows command to proceed when work item is in "doing" status
   - [ ] When `--skip-status-check` is used: Skips status update step (step 6), proceeds directly to worktree creation (step 7)

2. **Git Worktree Creation**
   - [ ] Determines trunk branch: uses `git.trunk_branch` if configured, otherwise auto-detects (main/master)
   - [ ] If both "main" and "master" branches exist: fails with clear error asking user to configure `git.trunk_branch`
   - [ ] Validates trunk branch exists, errors clearly if not found
   - [ ] Checks for remote 'origin' existence using `git remote get-url origin` before pulling
   - [ ] If remote 'origin' doesn't exist: Logs warning "Warning: No remote 'origin' configured. Skipping pull step. Worktree will be created from local trunk branch." and skips pull step
   - [ ] Continues with worktree creation even if remote doesn't exist
   - [ ] For polyrepo: Checks each project repository independently; skips pull for projects without remote (logs warning per project)
   - [ ] Checks for uncommitted changes in trunk branch before pull; aborts with clear error if found
   - [ ] Explicitly checks out trunk branch using `git checkout <trunk_branch>` before pulling
   - [ ] Pulls latest changes using `git fetch` + `git merge` (not `git pull`) for more control: `git fetch <remote-name> <trunk_branch>` followed by `git merge <remote-name>/<trunk_branch>`
   - [ ] Handles pull merge conflicts: aborts with clear error showing git output
   - [ ] Handles diverged branches: aborts with clear error message
   - [ ] Handles network errors during pull: aborts with clear error showing git output
   - [ ] For polyrepo: Determines trunk branch per project using priority: `project.trunk_branch` (if configured) > `git.trunk_branch` (workspace default) > auto-detect per project
   - [ ] For polyrepo: Supports per-project trunk branch override via `project.trunk_branch` config field
   - [ ] For polyrepo: Pulls from all project repositories sequentially before creating worktrees
   - [ ] For polyrepo: Aborts entire command if any project's pull fails (all-or-nothing)
   - [ ] Creates git worktree from trunk branch
   - [ ] Worktree directory name uses sanitized work item title
   - [ ] Worktree is created in appropriate location (configurable)
   - [ ] Checks if worktree path already exists before creation
   - [ ] If worktree exists for same work item: Errors with message suggesting `--override` flag
   - [ ] If worktree exists for different work item: Errors with message suggesting `--override` flag
   - [ ] If path exists but is not valid git worktree: Errors with message suggesting manual removal or `--override`
   - [ ] With `--override` flag: Removes existing worktree (using `git worktree remove`) or directory before creating new one
   - [ ] Command handles git repository detection correctly

3. **Branch Creation**
   - [ ] Checks if branch `{work-item-id}-{kebab-case-work-item-title}` already exists in repository before creating worktree
   - [ ] Creates branch with format: `{work-item-id}-{kebab-case-work-item-title}`
   - [ ] Branch name is valid for git (no invalid characters)
   - [ ] Branch is checked out in the new worktree
   - [ ] If branch exists and points to trunk commit: Errors with message suggesting `--reuse-branch` flag or branch deletion
   - [ ] If branch exists and has commits: Errors with message requiring branch deletion or different work item
   - [ ] With `--reuse-branch` flag: Creates worktree without `-b` flag, changes to worktree directory, then checks out existing branch using `git checkout {branch-name}`
   - [ ] Branch creation explicitly changes to worktree directory before running `git checkout -b` command
   - [ ] If branch doesn't exist: Creates new branch using `git worktree add <path> -b <branch-name> <trunk-branch>`

4. **Multi-Project Workspace**
   - [ ] Works correctly without workspace config (standalone mode)
   - [ ] Infers workspace behavior from configuration:
     - [ ] No `workspace.projects` → standalone behavior
     - [ ] Checks for `repo_root` first: If ANY project has `repo_root` → polyrepo behavior (immediate detection)
     - [ ] Otherwise checks if project paths point to separate git repositories
     - [ ] `workspace.projects` with paths pointing to subdirectories within current repo → monorepo behavior (LLM context)
     - [ ] `workspace.projects` with paths pointing to separate git repositories → polyrepo behavior (multi-repo)
     - [ ] `workspace.projects` without `path` fields → monorepo behavior (LLM context)
   - [ ] Reads workspace configuration from `kira.yml` when present
   - [ ] Standalone and Monorepo: Creates single worktree and branch (same behavior)
   - [ ] Monorepo: Projects configuration provides LLM context only (does not affect worktree/branch)
   - [ ] Polyrepo: Groups projects by `repo_root` value
   - [ ] Polyrepo: Creates ONE worktree at `repo_root` for each unique root value
   - [ ] Polyrepo: Creates ONE branch per repo_root group (shared by all projects in group)
   - [ ] Polyrepo: Skips worktree creation for subsequent projects with the same `repo_root`
   - [ ] Polyrepo: Creates separate worktree for each project without `repo_root` (standalone)
   - [ ] Polyrepo: Uses consistent branch naming: `{work-item-id}-{kebab-case-work-item-title}` across all worktrees
   - [ ] Polyrepo: Organizes worktrees at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{repo_root or project.mount}/`
   - [ ] Polyrepo: Validates all project repositories exist before creating any worktrees
   - [ ] Polyrepo: Errors clearly if project repository not found, aborts entire command (all-or-nothing)
   - [ ] Polyrepo: Creates all worktrees first (Phase 1), then creates branches (Phase 2) for easier rollback
   - [ ] Polyrepo: If any worktree creation fails: Rolls back all successfully created worktrees using `git worktree remove`, aborts command with error
   - [ ] Polyrepo: If any branch creation fails: Rolls back all successfully created worktrees using `git worktree remove`, aborts command with error
   - [ ] Polyrepo: Tracks all successfully created worktrees for rollback purposes
   - [ ] Handles mixed setup: some projects grouped by repo_root, others standalone
   - [ ] Respects draft_pr configuration at workspace and project levels

5. **IDE Integration**
   - [ ] Checks `--no-ide` flag first (highest priority)
   - [ ] If `--no-ide` flag provided: Skips IDE opening entirely, no log messages (useful for agents)
   - [ ] If `--no-ide` not set: Checks `--ide` flag next
   - [ ] If `--ide <command>` flag provided: Uses flag value as IDE command, ignores `ide.args` from config, executes with no args
   - [ ] If neither flag provided: Requires `ide.command` configuration in `kira.yml` - no auto-detection
   - [ ] If no IDE command found (flag or config): Skips IDE opening, logs info message, continues with worktree creation
   - [ ] Standalone/Monorepo: Opens IDE in worktree directory (if IDE command found)
   - [ ] Polyrepo: Opens IDE at worktree root (`{worktree_root}/{work-item-id}-{kebab-case-work-item-title}`) with all project worktrees (if IDE command found)
   - [ ] IDE opens with correct branch checked out (if IDE command found)
   - [ ] Command continues successfully even if IDE launch fails
   - [ ] When using config: Respects IDE configuration from `kira.yml` (`ide.command` and `ide.args`)
   - [ ] When using `--ide` flag: Overrides `ide.command` and ignores `ide.args` (no hardcoded IDE logic)
   - [ ] If IDE command not found: Logs warning and continues (worktree creation succeeds)
   - [ ] If IDE launch fails: Logs warning and continues (worktree creation succeeds)
   - [ ] Worktree creation succeeds regardless of IDE behavior
   - [ ] **Setup Commands/Scripts:**
     - [ ] If `workspace.setup` configured: Executes setup command/script in main project worktree directory
     - [ ] If `project.setup` configured: Executes setup command/script in project worktree directory (for polyrepo)
     - [ ] Setup runs after IDE opening (allows user to start working while setup runs in background)
     - [ ] For script paths: Executes with appropriate shell (bash for .sh, python for .py, etc.)
     - [ ] For commands: Uses centralized `executeCommand` function (or `os/exec` directly if centralized function not yet available)
     - [ ] Changes directory to worktree before execution
     - [ ] For polyrepo: Runs setups sequentially in order projects are defined
     - [ ] If setup fails: Logs error and aborts command (setup is critical for environment preparation)

6. **Status Management**
   - [ ] If `status_action: none`: Work item status is not changed, proceed to worktree creation
   - [ ] If `status_action: commit_only`: Updates status to configured status folder (`start.move_to`, defaults to "doing") and commits on trunk branch before worktree creation
   - [ ] If `status_action: commit_and_push`: Updates status to configured status folder (`start.move_to`, defaults to "doing"), commits and pushes to trunk branch before worktree creation
   - [ ] If `status_action: commit_only_branch`: Updates status to configured status folder (`start.move_to`, defaults to "doing") and commits on new branch after worktree creation
   - [ ] Updates work item status field correctly when status is changed
   - [ ] Validates work item before status update (when status is changed)
   - [ ] Respects `--status-action` flag (overrides config with: none|commit_only|commit_and_push|commit_only_branch)
   - [ ] Uses configured commit message template or default (when committing)
   - [ ] Aborts worktree creation if commit/push fails (for commit_only/commit_and_push)
   - [ ] Handles uncommitted changes in trunk branch appropriately (for commit_only/commit_and_push)

### Edge Cases

1. **Git Repository States**
   - [ ] Uses configured trunk branch if specified in `git.trunk_branch`
   - [ ] Auto-detects trunk branch (main first, then master) if not configured
   - [ ] If both "main" and "master" branches exist: fails with clear error asking user to configure `git.trunk_branch`
   - [ ] Handles custom trunk branch names (develop, trunk, production, etc.)
   - [ ] Errors clearly if trunk branch doesn't exist
   - [ ] Determines remote name: Uses `git.remote` if configured, otherwise defaults to "origin"
   - [ ] Checks for remote existence using `git remote get-url <remote-name>` before pulling
   - [ ] If remote doesn't exist: Logs warning "Warning: No remote '{remote-name}' configured. Skipping pull step. Worktree will be created from local trunk branch." and skips pull step
   - [ ] Continues with worktree creation even if remote doesn't exist
   - [ ] For polyrepo: For each project, determines remote name: Uses `project.remote` if configured, otherwise uses `git.remote` (or "origin" if `git.remote` is also omitted)
   - [ ] For polyrepo: Checks each project repository independently using its configured remote name; skips pull for projects without remote (logs warning per project with the remote name checked), pulls for projects with remote
   - [ ] Checks for uncommitted changes in trunk branch before pulling
   - [ ] Aborts with clear error if uncommitted changes found: "Trunk branch has uncommitted changes. Commit or stash changes before starting work."
   - [ ] Pulls from origin using `git fetch` + `git merge` before any operations
   - [ ] Handles pull merge conflicts gracefully (abort with clear error showing git output)
   - [ ] Handles diverged branches (abort with clear error: "Trunk branch has diverged from origin. Rebase or merge manually before starting work.")
   - [ ] Handles network errors during pull (abort with clear error showing git output)
   - [ ] Handles detached HEAD state appropriately
   - [ ] Works with repositories that use "main" vs "master" vs custom branch names
   - [ ] Handles repositories with no commits
   - [ ] Works with shallow clones
   - [ ] Handles branch already exists pointing to trunk (errors with suggestion to use `--reuse-branch` or delete branch)
   - [ ] Handles branch already exists with commits (errors requiring branch deletion or different work item)
   - [ ] `--reuse-branch` flag creates worktree and checks out existing branch that points to trunk

2. **Work Item States**
   - [ ] Works with work items in any status folder
   - [ ] Checks work item status after git pull (step 4) and before worktree creation (step 7) - executes as step 5
   - [ ] If work item is already in configured status (`start.move_to`, defaults to "doing"): Fails unless `--skip-status-check` flag is provided
   - [ ] `--skip-status-check` flag allows restarting work or reviewing work item elsewhere
   - [ ] When `--skip-status-check` is used: Skips status update step (step 6), proceeds directly to worktree creation (step 7)
   - [ ] Handles work items with special characters in title
   - [ ] Handles very long work item titles (truncates appropriately)
   - [ ] Handles work items with missing title field: Uses work item ID as fallback, logs warning "Warning: Work item {id} has no title field. Using work item ID '{id}' for worktree directory and branch name."
   - [ ] When title is missing: Worktree directory becomes `{work-item-id}-{work-item-id}` and branch becomes `{work-item-id}-{work-item-id}`

3. **File System**
   - [ ] Handles permission errors gracefully
   - [ ] Works with relative and absolute paths
   - [ ] Handles spaces in directory names
   - [ ] Works on Windows, macOS, and Linux
   - [ ] Handles worktree path already exists for same work item (errors with suggestion to use `--override`)
   - [ ] Handles worktree path already exists for different work item (errors with suggestion to use `--override`)
   - [ ] Handles path exists but is not a valid git worktree (errors with suggestion to use `--override`)
   - [ ] `--override` flag removes existing valid git worktree using `git worktree remove`
   - [ ] `--override` flag removes existing invalid worktree directory using filesystem operations
   - [ ] `--override` flag handles worktree with uncommitted changes (uses `git worktree remove --force`)

4. **Setup Behavior**
   - [ ] If `workspace.setup` configured: Executes in main project worktree directory
   - [ ] If `project.setup` configured: Executes in project worktree directory (for polyrepo)
   - [ ] Setup runs after IDE opening (allows user to start working while setup runs in background)
   - [ ] Handles script paths (e.g., `./scripts/setup.sh`) by detecting and using appropriate shell
   - [ ] Handles command strings (e.g., `docker compose up -d`) by executing directly
   - [ ] Changes directory to worktree before execution
   - [ ] For polyrepo: Runs setups sequentially in order projects are defined
   - [ ] If setup fails: Aborts command with clear error message
   - [ ] Handles setup commands that require interactive input (may timeout or fail appropriately)

5. **IDE Behavior**
   - [ ] Checks `--no-ide` flag first (highest priority)
   - [ ] If `--no-ide` flag provided: Skips IDE opening entirely, no log messages (useful for agents)
   - [ ] If `--no-ide` not set: Checks `--ide` flag next
   - [ ] If `--ide <command>` flag provided: Uses flag value, ignores `ide.args` from config, executes with no args
   - [ ] If neither flag provided: Requires `ide.command` configuration - no auto-detection
   - [ ] If no IDE command found (flag or config): Skips IDE opening, logs info message, continues
   - [ ] All IDE behavior is controlled by `--ide` flag or `ide.command`/`ide.args` configuration (no hardcoded IDE logic)
   - [ ] When using config: User configures appropriate args for their IDE in `kira.yml` (e.g., `["--new-window"]` to open in new window)
   - [ ] When using `--ide` flag: No args are passed (user can rely on shell expansion like `--ide "$KIRA_IDE"`)
   - [ ] If IDE command not found: Logs warning and continues (worktree creation succeeds)
   - [ ] If IDE launch fails: Logs warning "Warning: Failed to launch IDE. Worktree created successfully. You can manually open the IDE at {path}."
   - [ ] Worktree creation succeeds regardless of IDE behavior
   - [ ] Respects `ide.command` and `ide.args` configuration in `kira.yml` (when flag not used)

6. **Polyrepo Partial Failures**
   - [ ] Polyrepo: Validates all project repositories exist before creating any worktrees (all-or-nothing)
   - [ ] Polyrepo: If any project's pull fails, aborts entire command before creating worktrees (all-or-nothing)
   - [ ] Polyrepo: Creates all worktrees first (Phase 1), then creates branches (Phase 2)
   - [ ] Polyrepo: If any worktree creation fails, rolls back all successfully created worktrees
   - [ ] Polyrepo: If any branch creation fails, rolls back all successfully created worktrees
   - [ ] Polyrepo: Rollback removes worktrees in reverse order of creation
   - [ ] Polyrepo: Rollback handles worktrees with uncommitted changes (uses `git worktree remove --force`)
   - [ ] Polyrepo: Rollback continues attempting to remove other worktrees even if one removal fails
   - [ ] Polyrepo: Error messages clearly indicate which project failed and that rollback was performed

7. **Error Scenarios** (Explicit test cases for all error conditions mentioned in Error Handling section)

   **Validation Errors:**
   - [ ] **Work item not found:** Command fails with error: "Error: Work item '{id}' not found. No work item file exists with that ID. Check the work item ID and try again."
   - [ ] **Work item missing title:** Command logs warning "Warning: Work item {id} has no title field. Using work item ID '{id}' for worktree directory and branch name." and continues execution using work item ID as fallback
   - [ ] **Invalid git repository:** Command fails with error: "Error: Not a git repository. Current directory is not a git repository. Run this command from within a git repository."
   - [ ] **Worktree path already exists (same work item):** Command fails with error: "Error: Worktree already exists at {path} for work item {id}. A worktree for this work item already exists. Use `--override` to remove existing worktree and create a new one, or use the existing worktree."
   - [ ] **Worktree path already exists (different work item):** Command fails with error: "Error: Worktree path {path} already exists for a different work item. Path conflicts with existing worktree. Use `--override` to remove existing worktree, or choose a different work item."
   - [ ] **Path exists but not valid git worktree:** Command fails with error: "Error: Path {path} already exists but is not a valid git worktree. Cannot create worktree at this location. Remove it manually and try again, or use `--override` to remove it automatically."
   - [ ] **Branch already exists (points to trunk):** Command fails with error: "Error: Branch {branch-name} already exists and points to trunk. Branch exists but has no commits. Use `--reuse-branch` to checkout existing branch in new worktree, or delete the branch first: `git branch -d {branch-name}`"
   - [ ] **Branch already exists (has commits):** Command fails with error: "Error: Branch {branch-name} already exists and has commits. Branch contains work that would be lost. Delete the branch first if you want to start fresh: `git branch -D {branch-name}`, or use a different work item."

   **Git Errors:**
   - [ ] **Trunk branch not found:** Command fails with error: "Error: Trunk branch '{branch-name}' not found. Configured branch does not exist and auto-detection failed. Verify the branch name in `git.trunk_branch` configuration or ensure 'main' or 'master' branch exists."
   - [ ] **Both 'main' and 'master' branches exist:** Command fails with error: "Error: Both 'main' and 'master' branches exist. Cannot auto-detect trunk branch. Configure `git.trunk_branch` explicitly in kira.yml to specify which branch to use."
   - [ ] **Repository without remote origin:** Command logs warning "Warning: No remote '{remote-name}' configured. Skipping pull step. Worktree will be created from local trunk branch." and continues with worktree creation
   - [ ] **Uncommitted changes in trunk branch (before pull):** Command fails with error: "Error: Trunk branch has uncommitted changes. Cannot proceed with pull operation. Commit or stash changes before starting work."
   - [ ] **Uncommitted changes in trunk branch (for commit_only/commit_and_push):** Command fails with error: "Error: Trunk branch has uncommitted changes. Cannot commit status change. Commit or stash changes before starting work."
   - [ ] **Pull merge conflicts:** Command fails with error: "Error: Failed to merge latest changes from {remote-name}/{trunk-branch}. Merge conflicts detected. Resolve conflicts manually and try again." (includes git output)
   - [ ] **Diverged branches:** Command fails with error: "Error: Trunk branch has diverged from {remote-name}/{trunk-branch}. Local and remote branches have different commits. Rebase or merge manually before starting work."
   - [ ] **Network errors during pull:** Command fails with error: "Error: Failed to fetch changes from {remote-name}. Network error occurred. Check network connection and try again." (includes git output)
   - [ ] **Invalid status_action value:** Command fails with error: "Error: Invalid status_action value '{value}'. Value is not recognized. Use one of: 'none', 'commit_only', 'commit_and_push', 'commit_only_branch'."
   - [ ] **Status commit failure:** Command fails with error: "Error: Failed to commit status change. Git commit operation failed. Check git output for details and resolve any issues." (includes git output, aborts worktree creation)
   - [ ] **Status push failure:** Command fails with error: "Error: Failed to push status change to {remote-name}/{trunk-branch}. Git push operation failed. Check git output for details and resolve any issues." (includes git output, aborts worktree creation)
   - [ ] **Worktree creation failure:** Command fails with error: "Error: Failed to create worktree at {path}. Git worktree creation failed. Check git output for details and resolve any issues." (includes git output)
   - [ ] **Polyrepo worktree creation failure:** Command fails with error: "Error: Failed to create worktree for project '{project-name}'. Worktree creation failed for one or more projects. All worktrees have been rolled back. Check git output for details and resolve any issues."
   - [ ] **Polyrepo branch creation failure:** Command fails with error: "Error: Failed to create branch for project '{project-name}'. Branch creation failed for one or more projects. All worktrees have been rolled back. Check git output for details and resolve any issues."
   - [ ] **Branch checkout failure (with --reuse-branch):** Command fails with error: "Error: Failed to checkout branch '{branch-name}' in worktree. Branch checkout operation failed. Check git output for details and resolve any issues." (includes git output)
   - [ ] **Missing project repository (polyrepo):** Command fails with error: "Error: Project repository not found at {path}. Path does not exist or is not a git repository. Verify path exists and is a git repository, or update project configuration in kira.yml." (aborts entire command)

   **Work Item Status Errors:**
   - [ ] **Work item already in configured status:** Command fails with error: "Error: Work item {id} is already in '{move_to}' status. Work item status matches target status. Use --skip-status-check to restart work or review elsewhere." (unless `--skip-status-check` flag is provided)

   **Setup Errors:**
   - [ ] **Setup command/script not found:** Command fails with error: "Error: Setup command/script not found: {setup}. File or command does not exist. Verify `workspace.setup` or `project.setup` configuration in kira.yml."
   - [ ] **Setup execution failure:** Command fails with error: "Error: Setup command failed: {error}. Setup command exited with error. Check command output for details and resolve any issues." (includes command output)
   - [ ] **Setup script execution permission denied:** Command fails with error: "Error: Setup script is not executable: {script}. Script lacks execute permissions. Make script executable (`chmod +x {script}`) or use appropriate interpreter."
   - [ ] **Setup timeout:** Command fails with error: "Error: Setup command timed out. Command did not complete within timeout period. Verify setup command completes successfully or increase timeout if needed."

4. **Configuration**
   - [ ] Works without `kira.yml` (defaults to standalone)
   - [ ] Works without workspace config (defaults to standalone)
   - [ ] All workspace values default to standalone behavior when omitted
   - [ ] Validates configuration syntax
   - [ ] Provides clear errors for invalid configuration
   - [ ] draft_pr defaults to false at all levels when omitted

### Integration

1. **Git Integration**
   - [ ] Worktrees are properly linked to main repository
   - [ ] Branch appears in `git branch -a` output
   - [ ] Worktree can be removed with `git worktree remove`

2. **Kira Integration**
   - [ ] Works with existing `kira move` command
   - [ ] Works with `kira save` command
   - [ ] Status updates are tracked in git history

3. **IDE Integration**
   - [ ] IDE opens with correct workspace/project path
   - [ ] Command-line editors work correctly

## Implementation Notes

### Architecture

**Architectural Recommendation: Centralized Command Execution**

Currently, Kira executes commands directly using `exec.CommandContext` in multiple places (`move.go`, `save.go`, etc.). For better consistency, maintainability, and to support dry-run functionality across all commands, it is recommended to create a centralized command execution function:

- **Location:** Add `executeCommand` function to `internal/commands/utils.go`
- **Signature:** `func executeCommand(ctx context.Context, name string, args []string, dir string, dryRun bool) error`
- **Behavior:**
  - If `dryRun` is `true`: Print the command that would be executed (e.g., `[DRY RUN] git fetch origin main`) and return `nil` without executing
  - If `dryRun` is `false`: Execute the command using `exec.CommandContext` with proper error handling
- **Usage:** All git operations, setup commands, and IDE launching in `kira start` should use this centralized function
- **Benefits:**
  - Consistent error handling across all command executions
  - Unified dry-run support for all commands
  - Easier to add features like command logging, timeouts, or retry logic in the future
  - Reduces code duplication
- **Implementation Strategy:**
  - **Recommended approach:** Before implementing `kira start`, refactor existing commands (`move.go`, `save.go`, etc.) to use the centralized `executeCommand` function
  - This ensures the centralized function is battle-tested and working correctly before being used in new code
  - Refactoring existing commands first also provides immediate benefits (consistent error handling, dry-run support) across the entire codebase
  - After refactoring existing commands, `kira start` can be implemented using the centralized function from the start
- **Alternative approach:** If refactoring existing commands first is not feasible, `kira start` can initially use direct `os/exec` calls (as documented in "Git Operations" section below), but should migrate to the centralized function when it becomes available

0. **Dry Run Mode Implementation**
   - **Flag detection:** Check for `--dry-run` flag early in command execution (before any state-modifying operations)
   - **Validation:** Perform all validation steps (work item exists, trunk branch detection, status check) as normal - these don't modify state
   - **Preview generation:** After validation, generate structured preview without executing:
     - **Worktrees:** List all worktrees that would be created with their full paths (e.g., `../my-project_worktrees/123-fix-bug/`)
     - **Branches:** List all branches that would be created with their names (e.g., `123-fix-bug`)
     - **Status changes:** Show current status → target status (from `start.move_to` config, defaults to "doing")
     - **Git operations:** List what git commands would be executed:
       - `git fetch <remote> <trunk_branch>` + `git merge <remote>/<trunk_branch>` (if remote exists)
       - `git worktree add <path> -b <branch> <trunk>` (worktree creation)
       - `git commit` (if `status_action` requires commit)
       - `git push` (if `status_action` is `commit_and_push`)
     - **Setup commands:** List what setup commands/scripts would be executed (if `workspace.setup` or `project.setup` configured)
     - **IDE:** Show what IDE command would be executed (if `ide.command` configured or `--ide` flag provided)
   - **Output format:** Use structured output with clear sections and indentation for readability
   - **Early exit:** After preview generation, exit with code 0 (success) if preview generated successfully, or non-zero if validation errors occurred
   - **No execution:** Skip all steps that would modify state:
     - Skip git operations (no pull, no worktree creation, no branch creation, no commits, no pushes)
     - Skip file moves and status updates
     - Skip IDE opening
     - Skip setup command execution

1. **Command Structure**
   - Create `internal/commands/start.go` following existing command patterns
   - Use Cobra for CLI argument parsing
   - Follow error handling patterns from `move.go` and `save.go`
   - Add `--dry-run` flag to command flags

2. **Git Operations**
   - **Preferred approach:** Use the centralized `executeCommand` function (see architectural recommendation above) for all git commands, passing `dryRun` flag appropriately
   - **Initial implementation:** If centralized function doesn't exist yet, use `os/exec` directly (similar to `save.go`) but plan to migrate to centralized function
   - **Trunk Branch Detection:**
     - **Standalone/Monorepo:**
       - Use `git.trunk_branch` if configured
       - Otherwise auto-detect: check for "main" first, then "master"
       - Check if both "main" and "master" branches exist using `git show-ref --verify --quiet refs/heads/main` and `git show-ref --verify --quiet refs/heads/master`
       - If both branches exist: fail with error: "Both 'main' and 'master' branches exist. Configure `git.trunk_branch` explicitly in kira.yml to specify which branch to use."
       - Validate trunk branch exists, error if not found
     - **Polyrepo:** For each project:
       - **Priority order:** Check `project.trunk_branch` first, then `git.trunk_branch`, then auto-detect
       - **Per-project override:** If `project.trunk_branch` is configured, use that value for this project
       - **Workspace default:** If `project.trunk_branch` not set but `git.trunk_branch` is configured, use workspace default for this project
       - **Auto-detection:** If neither configured, auto-detect per project repository (check for "main" first, then "master" in each project's repo)
       - **Conflict handling:** If both "main" and "master" exist in a project's repo:
         - If `project.trunk_branch` or `git.trunk_branch` is configured: use configured value
         - Otherwise: fail with error: "Both 'main' and 'master' branches exist in {project.path}. Configure `git.trunk_branch` (workspace-level) or `trunk_branch` in project config to specify which branch to use."
       - **Validation:** Validate trunk branch exists in each project's repository, error if not found
   - **Pull Latest Changes:**
     - **Remote Name Resolution:**
       - For main repository: Use `git.remote` if configured, otherwise default to "origin"
       - For polyrepo projects: For each project, determine remote name in order of precedence:
         1. `project.remote` if configured
         2. `git.remote` if configured
         3. "origin" as final default
     - Check if remote exists using `git remote get-url <remote-name>`; if command fails (no remote configured):
       - Log warning: "Warning: No remote '{remote-name}' configured. Skipping pull step. Worktree will be created from local trunk branch."
       - Skip pull step, continue with worktree creation
     - Check for uncommitted changes in trunk branch using `git status --porcelain`; if output is non-empty, abort with error: "Error: Trunk branch has uncommitted changes. Cannot proceed with pull operation. Commit or stash changes before starting work."
     - **Standalone/Monorepo pull sequence:**
       - Determine remote name: Use `git.remote` if configured, otherwise default to "origin"
       - **Ensure we're on trunk branch:** Explicitly checkout trunk branch (configured or auto-detected) using `git checkout <trunk_branch>`
       - **Pull latest changes:** Use `git fetch` + `git merge` for more control (rather than `git pull`):
         - Run `git fetch <remote-name> <trunk_branch>` to fetch latest changes (where remote-name is from `git.remote` or "origin" default)
         - Run `git merge <remote-name>/<trunk_branch>` to merge remote changes into local trunk branch
     - **Polyrepo pull sequence:** For each project:
       - Determine remote name: Use `project.remote` if configured, otherwise `git.remote` (or "origin" if `git.remote` is also omitted)
       - Check if remote exists: Run `git remote get-url <project-remote-name>` in each project's repository directory
       - If remote doesn't exist: Skip pull for this project (log warning: "Warning: No remote '{project-remote-name}' configured for project '{project-name}'. Skipping pull step.")
       - If remote exists:
         - **Ensure we're on trunk branch:** Explicitly checkout trunk branch using `git checkout <trunk-branch>` (using project-specific trunk branch if `project.trunk_branch` is configured)
         - **Pull latest changes:** Use `git fetch` + `git merge` for more control:
           - Run `git fetch <project-remote-name> <trunk-branch>` to fetch latest changes
           - Run `git merge <project-remote-name>/<trunk-branch>` to merge remote changes into local trunk branch
     - If merge fails with conflicts: abort with error: "Error: Failed to merge latest changes from {remote-name}/{trunk-branch}. Merge conflicts detected. Resolve conflicts manually and try again." Include git output in error message
     - If merge fails because branches have diverged: abort with error: "Error: Trunk branch has diverged from {remote-name}/{trunk-branch}. Local and remote branches have different commits. Rebase or merge manually before starting work."
     - If network error occurs: abort with error: "Error: Failed to fetch changes from {remote-name}. Network error occurred. Check network connection and try again." Include git output in error message
     - For polyrepo: Perform above steps sequentially for all project repositories; abort entire command if any project's pull fails (all-or-nothing approach)
     - This ensures everything is up-to-date before any operations
   - **Work Item Status Check:** (Executes as step 5 in Command Execution Order, after git pull and before worktree creation)
     - After git pull completes successfully (step 4), check current work item status
     - This ensures we're checking the most up-to-date work item status after pulling latest changes
     - Get configured status from `start.move_to` (defaults to "doing")
     - If status matches configured value and `--skip-status-check` flag is not provided:
       - Return error: "Error: Work item {id} is already in '{move_to}' status. Work item status matches target status. Use --skip-status-check to restart work or review elsewhere." (where {move_to} is the configured value)
       - Abort command before worktree creation (step 7)
     - If status is "doing" and `--skip-status-check` flag is provided:
       - Skip status update step (don't move file or update status field)
       - Proceed directly to worktree creation (step 7)
       - Allows resuming work or reviewing work item in different location
     - If status does not match configured `start.move_to`: Continue with normal status management flow (step 6)
   - **Status Management:**
     - If `status_action` is not `none`:
       - Call `moveWorkItem(cfg, workItemID, move_to, false)` from `move.go` (with `commitFlag=false`) to move file and update status
       - This reuses the existing function to ensure consistency with `kira move` command
       - If `status_action: commit_only` or `status_action: commit_and_push`:
         - Stage work item file change (file is now at new location)
         - Commit with configured message template (or default) on trunk branch
         - If `commit_and_push`: Push to origin/<trunk_branch>
         - Abort worktree creation if commit/push fails
       - If `status_action: commit_only_branch`:
         - Create worktree and branch first
         - Call `moveWorkItem(cfg, workItemID, move_to, false)` from `move.go` (with `commitFlag=false`) to move file and update status
         - Then stage work item file change (file is now at new location)
         - Commit with configured message template (or default) on new branch
     - If `status_action: none`: Skip status update, proceed directly to worktree creation
   - **Worktree Creation:**
     - Check if target worktree path already exists
     - If path exists:
       - Check if `--override` flag is provided
       - If `--override` flag is provided:
         - If path is a valid git worktree: Remove it using `git worktree remove <path>` (or `git worktree remove --force` if worktree has uncommitted changes)
         - If path is not a valid git worktree: Remove directory using `os.RemoveAll`
         - Then proceed with worktree creation
       - If `--override` flag is not provided:
         - Check if path is a valid git worktree (check for `.git` file pointing to main repo)
         - If valid git worktree:
           - Check if branch name matches `{work-item-id}-{kebab-case-work-item-title}` or if work item ID is in path
           - If same work item: Error: `"Error: Worktree already exists at {path} for work item {id}. A worktree for this work item already exists. Use \`--override\` to remove existing worktree and create a new one, or use the existing worktree."`
           - If different work item: Error: `"Error: Worktree path {path} already exists for a different work item. Path conflicts with existing worktree. Use \`--override\` to remove existing worktree, or choose a different work item."`
         - If not valid git worktree: Error: `"Error: Path {path} already exists but is not a valid git worktree. Cannot create worktree at this location. Remove it manually and try again, or use \`--override\` to remove it automatically."`
     - Use trunk branch (configured or auto-detected)
     - **Branch Existence Check:**
       - Before creating worktree, check if branch `{work-item-id}-{kebab-case-work-item-title}` exists using `git show-ref --verify --quiet refs/heads/{branch-name}`
       - If branch exists:
         - Get branch commit hash using `git rev-parse {branch-name}`
         - Get trunk branch commit hash using `git rev-parse <trunk-branch>`
         - Compare commits:
           - If commits match (branch points to trunk): Branch exists but has no commits
             - If `--reuse-branch` flag is provided:
               - Create worktree without `-b` flag: `git worktree add <path> <trunk-branch>`
               - **Change to worktree directory:** `cd <worktree-path>` (or use `exec.Command` with `Dir` field set to worktree path)
               - **Checkout existing branch:** `git checkout {branch-name}` (executed in worktree directory)
             - If `--reuse-branch` flag is not provided: Error: `"Error: Branch {branch-name} already exists and points to trunk. Branch exists but has no commits. Use \`--reuse-branch\` to checkout existing branch in new worktree, or delete the branch first: \`git branch -d {branch-name}\`"`
           - If commits don't match (branch has commits): Error: `"Error: Branch {branch-name} already exists and has commits. Branch contains work that would be lost. Delete the branch first if you want to start fresh: \`git branch -D {branch-name}\`, or use a different work item."`
       - If branch doesn't exist: Create worktree and branch in one command: `git worktree add <path> -b <branch-name> <trunk-branch>` (this automatically creates branch in the worktree directory)
   - **Branch Creation:**
     - **When branch doesn't exist (new branch):**
       - Create worktree and branch in one command: `git worktree add <path> -b <branch-name> <trunk-branch>`
       - This automatically creates the branch in the worktree directory
       - **Command details:** `<path>` is the worktree path, `<branch-name>` is `{work-item-id}-{kebab-case-work-item-title}`, `<trunk-branch>` is the configured or auto-detected trunk branch
     - **When branch exists and points to trunk (`--reuse-branch` flag):**
       - Create worktree without `-b` flag: `git worktree add <path> <trunk-branch>`
       - **Change to worktree directory:** Change directory to worktree path (use `os.Chdir()` or `exec.Command` with `Dir` field set to worktree path)
       - **Checkout existing branch:** Run `git checkout <branch-name>` (executed in worktree directory)
     - **For polyrepo (Phase 2 branch creation):**
       - After all worktrees are created successfully:
       - For each worktree:
         - **Change to worktree directory:** Change directory to worktree path (use `os.Chdir()` or `exec.Command` with `Dir` field set to worktree path)
         - **Create and checkout branch:** Run `git checkout -b <branch-name>` (executed in worktree directory)
         - If branch creation fails: Rollback all worktrees using `git worktree remove <path>` for each, abort command with error: `"Error: Failed to create branch for project '{project-name}'. Branch creation failed for one or more projects. All worktrees have been rolled back. Check git output for details and resolve any issues."`
   - **Status Commit Operations:**
     - **For `commit_only` or `commit_and_push`:**
       - Stage work item file: `git add <work-item-path>` (file is at new location after `moveWorkItem` call)
       - Commit on trunk branch: `git commit -m "<commit-message>"` (where commit message is from template or default)
       - If `commit_and_push`: Push to remote: `git push <remote-name> <trunk-branch>` (where remote-name is from `git.remote` or "origin" default)
       - **Execution context:** Commands run in main repository directory (trunk branch)
     - **For `commit_only_branch`:**
       - Stage work item file: `git add <work-item-path>` (file is at new location after `moveWorkItem` call)
       - Commit on new branch: `git commit -m "<commit-message>"` (where commit message is from template or default)
       - **Execution context:** Commands run in worktree directory (new branch)
       - **Note:** The work item file is in the original repository's `.work/` directory, but the commit happens in the worktree context

3. **Work Item Parsing**
   - Reuse `extractWorkItemMetadata` function from `move.go`
   - Extract title and sanitize for directory/branch names (see "Title Sanitization" section below for detailed sanitization rules)
   - **Handle missing title:** If title is empty or "unknown":
     - Use just the work item ID as fallback
     - Log warning: "Warning: Work item {id} has no title field. Using work item ID '{id}' for worktree directory and branch name."
     - This ensures worktree directory and branch names are always valid: `{work-item-id}-{work-item-id}` when title is missing
   - Validate work item exists using `findWorkItemFile` from `utils.go`

4. **Configuration Extension**
   - Extend `internal/config/config.go` with new struct fields:
     ```go
     type Config struct {
         // ... existing fields
         Workspace *WorkspaceConfig `yaml:"workspace"`
         IDE       *IDEConfig       `yaml:"ide"`
         Git       *GitConfig        `yaml:"git"`
         Start     *StartConfig      `yaml:"start"`
     }

     type GitConfig struct {
         TrunkBranch string `yaml:"trunk_branch"` // default: "" (auto-detect)
         Remote      string `yaml:"remote"`       // default: "origin"
     }

    type StartConfig struct {
        MoveTo              string `yaml:"move_to"`               // default: "doing"
        StatusAction        string `yaml:"status_action"`         // default: "commit_and_push"
        // Valid values: "none" | "commit_only" | "commit_and_push" | "commit_only_branch"
        StatusCommitMessage string `yaml:"status_commit_message"` // optional template (default uses move_to value)
    }

     type IDEConfig struct {
         Command string   `yaml:"command"` // IDE command name (required, no auto-detection) (e.g., "cursor", "code", "idea", "vim")
         Args    []string `yaml:"args"`   // Arguments to pass to IDE command (optional) (e.g., ["--new-window"] to open in new window)
     }

    type WorkspaceConfig struct {
        // Type is inferred from projects configuration - no field needed
        Root           string         `yaml:"root"`           // default: "../"
        WorktreeRoot   string         `yaml:"worktree_root"`  // default: derived from project structure (see "Worktree Location" section for detailed algorithm)
        ArchitectureDoc string        `yaml:"architecture_doc"`
        Description    string         `yaml:"description"`
        DraftPR        bool           `yaml:"draft_pr"`      // default: false
        Setup          string         `yaml:"setup"`          // optional: command or script path for main project setup
        Projects       []ProjectConfig `yaml:"projects"`
    }

    type ProjectConfig struct {
        Name         string `yaml:"name"`
        Path         string `yaml:"path"`
        Mount        string `yaml:"mount"`        // default: name
        RepoRoot string `yaml:"repo_root"` // optional: groups projects sharing same root (monorepo grouping) or handles nested folder structures
        Kind         string `yaml:"kind"`         // app|service|library|infra
        Description  string `yaml:"description"`
        DraftPR      *bool  `yaml:"draft_pr"`     // nil = use workspace default
        Remote       string `yaml:"remote"`      // default: "" (use git.remote or "origin")
        TrunkBranch  string `yaml:"trunk_branch"` // optional: per-project trunk branch override (defaults to workspace.git.trunk_branch or auto-detected)
        Setup        string `yaml:"setup"`       // optional: command or script path for project-specific setup
    }
     ```
   - Default values should be applied during config loading:
     - If `git` is nil: use defaults (trunk_branch: "" for auto-detect)
     - If `git.trunk_branch` is omitted or empty: auto-detect (check "main" first, then "master")
     - If `start` is nil: use defaults (move_to: "doing", status_action: "commit_and_push")
     - If `start.move_to` is omitted or empty: default to "doing"
     - If `start.status_action` is omitted: default to "commit_and_push"
     - Validate `start.status_action` is one of: "none", "commit_only", "commit_and_push", "commit_only_branch"
     - If `start.status_commit_message` is empty: use default template "Move {type} {id} to {move_to}" (uses configured `move_to` value, or "doing" if not configured)
     - If `git.remote` is omitted or empty: default to "origin"
     - For polyrepo projects: If `project.remote` is omitted or empty: use `git.remote` (or "origin" if `git.remote` is also omitted)
     - For polyrepo projects: If `project.trunk_branch` is omitted or empty: use `git.trunk_branch` (or auto-detect if `git.trunk_branch` is also omitted)
     - If `workspace` is nil: treat as standalone
     - **Infer workspace behavior:**
       - If `workspace.projects` is nil or empty: standalone behavior
       - If `workspace.projects` exists but no projects have `path` fields: monorepo behavior (LLM context)
       - If `workspace.projects` exists with `path` fields: polyrepo behavior (multi-repo)
     - If `workspace.root` is empty: default to "../"
     - If `workspace.worktree_root` is empty: derive default based on workspace type (see "Worktree Location" section for details)
     - If `workspace.draft_pr` is false/omitted: default to false
     - If `project.draft_pr` is nil: use workspace default
     - If `project.mount` is empty: default to project.name
     - If `project.path` is empty: project describes current repo (monorepo context)
     - If `project.repo_root` is empty: project is standalone (gets own worktree)

5. **Title Sanitization**
   - **Handle missing title:** If title is empty or "unknown" (from `extractWorkItemMetadata`):
     - Use just the work item ID as fallback
     - Log warning: "Warning: Work item {id} has no title field. Using work item ID '{id}' for worktree directory and branch name."
   - Convert title to kebab-case
   - Remove special characters invalid for git
   - **Branch name length limit:**
     - Git allows branch names up to 255 characters, but very long names are unwieldy and hard to work with
     - **Recommended approach:** Truncate sanitized title to 100 characters
     - **Uniqueness handling:** If truncation occurred (original sanitized title > 100 chars), append `-{short-hash}` where `short-hash` is the first 6 characters of SHA256 hash of the full sanitized title
     - This ensures:
       - Branch names remain readable and manageable (100 chars is sufficient for most descriptive titles)
       - Uniqueness is preserved even when different long titles truncate to the same 100 characters
       - No need to check for existing branch collisions (hash ensures uniqueness)
   - Handle edge cases (all special chars, etc.)

6. **IDE Detection**
   - **Configuration priority:** Check flags in order: `--no-ide` (highest), then `--ide <command>`, then `ide.command` from `kira.yml`
   - **`--no-ide` flag:** If `--no-ide` flag is provided:
     - Skip IDE opening entirely (useful for agents or CI/CD environments)
     - Ignore `--ide` flag and `ide.command` config (flag takes precedence)
     - No log messages about IDE (silently skip)
   - **Flag override:** If `--ide <command>` flag is provided (and `--no-ide` not set):
     - Use flag value as IDE command (overrides `ide.command` from config)
     - Ignore `ide.args` from config (no args passed)
     - Execute: `{flag-command} {worktree-path}` (no args)
   - **Config-based:** If neither flag provided, use `ide.command` from `kira.yml` - no auto-detection
   - If no IDE command found (flag or config): Skip IDE opening, log info message "Info: No IDE configured. Worktree created at {path}. Configure `ide.command` in kira.yml or use `--ide <command>` flag to automatically open IDE.", continue
   - **No hardcoded logic:** All IDE behavior comes from flags (`--no-ide`, `--ide`) or `ide.command`/`ide.args` configuration
   - **IDE Launch:**
     - Check if `--no-ide` flag is provided
     - If `--no-ide` provided: Skip IDE opening (no execution, no log messages)
     - If `--no-ide` not provided: Check if `--ide <command>` flag is provided
     - If `--ide` flag provided: Execute `{flag-command} {worktree-path}` (no args)
     - If `--ide` flag not provided: Check if `ide.command` is configured
     - If configured: Execute `{ide.command} {ide.args...} {worktree-path}` (with args from config)
     - Use configured args exactly as specified - no automatic flag appending
     - **Preferred approach:** Use the centralized `executeCommand` function (see architectural recommendation above) for IDE launching, passing `dryRun` flag appropriately
     - **Initial implementation:** If centralized function doesn't exist yet, use `os/exec` directly but plan to migrate to centralized function
     - If command not found: Log warning "Warning: IDE command '{command}' not found. Verify `--ide` flag value or `ide.command` in kira.yml.", continue
     - If launch fails: Log warning "Warning: Failed to launch IDE. Worktree created successfully. You can manually open the IDE at {path}.", continue
   - IDE opens before setup commands run (allows user to start working while setup runs in background)

7. **Setup Commands/Scripts**
   - **Main project setup:** If `workspace.setup` is configured:
     - Execute setup command/script in main project worktree directory
     - For standalone/monorepo: Run in `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}`
     - For polyrepo: Run in main project worktree (if main project has its own worktree)
   - **Project-specific setup:** For each project in polyrepo with `project.setup` configured:
     - Execute setup command/script in that project's worktree directory
     - Run in the worktree path where the project is located
   - **Execution details:**
     - **Preferred approach:** Use the centralized `executeCommand` function (see architectural recommendation above) for all setup commands, passing `dryRun` flag appropriately
     - **Initial implementation:** If centralized function doesn't exist yet:
       - If setup is a script path (contains `/` or starts with `./`): Execute with shell (e.g., `bash ./scripts/setup.sh`)
       - If setup is a command: Execute directly using `os/exec` (e.g., `docker compose up -d`)
     - Change directory to worktree before execution (use `os.Chdir` or `exec.Command` with `Dir` field)
     - If setup fails: Log error "Error: Setup command failed: {error}. Aborting." and abort command
     - For polyrepo: Run setups sequentially in order projects are defined in config
     - Setup runs after IDE opening (allows user to start working while setup runs in background)
     - **Script execution:** For script paths, determine shell based on file extension (.sh → bash, .py → python, etc.) or use default shell
     - Worktree creation succeeds regardless of IDE behavior

7. **Workspace Handling**
   - If no workspace config: use standalone defaults
   - Parse workspace configuration when present
     - **Infer workspace behavior from configuration:**
     - If `workspace.projects` is nil or empty: **standalone** behavior
     - If `workspace.projects` exists:
       - **First check:** If ANY project has `repo_root` field → **polyrepo** behavior (immediate detection)
       - **Otherwise:** Check if projects point to separate git repositories
         - For each project with a `path` field:
           - Resolve path relative to current repository
           - Check if path contains a `.git` directory (separate git repository)
         - If ANY project path is a separate git repository: **polyrepo** behavior (multi-repo)
         - If ALL project paths are subdirectories within current repo (or no paths): **monorepo** behavior (LLM context)
   - **Standalone/Monorepo Behavior:** Same implementation - create single worktree and branch
     - Detect project root directory name from current repository path
     - Create single worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}` (see "Worktree Location" section for defaults)
     - Create single branch: `{work-item-id}-{kebab-case-work-item-title}`
     - IDE opens in worktree directory
   - **Polyrepo Behavior:** Different implementation - create worktrees based on grouping with transaction-like rollback
     - **Pre-validation:** Before creating any worktrees:
       - Validate all project paths exist and are git repositories
       - If any project path is invalid or not a git repository: Abort with error listing all invalid projects (all-or-nothing)
     - **Phase 1 - Worktree Creation:** Create all worktrees first (makes rollback easier)
       - Initialize empty list to track successfully created worktrees: `createdWorktrees []string`
       - **Group projects by `repo_root`:**
         - Group projects that share the same `repo_root` value
         - **Purpose:** Groups projects sharing the same root directory (monorepo case) or handles nested folder structures
         - For each unique `repo_root`:
           - Create ONE worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{sanitized-repo-root}/`
           - If creation succeeds: Add worktree path to `createdWorktrees` list
           - If creation fails: Rollback all worktrees in `createdWorktrees` using `git worktree remove <path>` for each, abort command with error
           - Track which projects are in each group to skip duplicate worktree creation
       - **Standalone projects** (no `repo_root`):
         - For each project without `repo_root`:
           - Create separate worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{project.mount}/`
           - If creation succeeds: Add worktree path to `createdWorktrees` list
           - If creation fails: Rollback all worktrees in `createdWorktrees` using `git worktree remove <path>` for each, abort command with error
     - **Phase 2 - Branch Creation:** After all worktrees created successfully:
       - For each worktree in `createdWorktrees`:
         - **Change to worktree directory:** Change directory to worktree path (use `os.Chdir()` or `exec.Command` with `Dir` field set to worktree path)
         - **Create and checkout branch:** Run `git checkout -b {work-item-id}-{kebab-case-work-item-title}` (executed in worktree directory)
         - If branch creation fails: Rollback all worktrees in `createdWorktrees` using `git worktree remove <path>` for each, abort command with error
     - Use consistent naming: `{work-item-id}-{kebab-case-work-item-title}` for both worktree directories and branches
     - IDE opens at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}` (worktree root)
   - **Rollback Helper Function:**
     - Create helper function `rollbackWorktrees(worktrees []string) error` that:
       - Iterates through list of worktree paths in reverse order
       - For each worktree: Run `git worktree remove <path>` (or `git worktree remove --force` if worktree has uncommitted changes)
       - Log each removal attempt
       - Return error if any removal fails (but continue attempting to remove others)
   - Apply defaults for missing configuration values
   - Store draft_pr configuration for future PR creation

8. **Status Commit and Push**
   - **Status Update Implementation:**
     - **Reuse existing function:** Call `moveWorkItem` function from `move.go` with `commitFlag=false`:
       - This function handles: finding work item file, validating target status against `cfg.StatusFolders`, moving file to `.work/{status_folder}/` directory, and updating status field in frontmatter
       - Since `kira start` has different commit behavior (commit_only, commit_and_push, commit_only_branch), we call `moveWorkItem` with `commitFlag=false` and handle commits separately
       - This ensures consistency with the `kira move` command behavior and avoids code duplication
     - **Commit handling:** After `moveWorkItem` completes, handle commits based on `status_action`:
       - Reuse `commitMove` pattern from `move.go` for committing status change (when `status_action` requires commit)
       - For `commit_only_branch`: Commit happens in worktree context (different from `move.go` which commits in main repo)
   - Handle different `status_action` values:
     - `none`: Don't update status, proceed directly to worktree creation
     - `commit_only`: Commit on trunk branch before worktree creation
       - Call `moveWorkItem(cfg, workItemID, move_to, false)` from `move.go` (with `commitFlag=false`) to move file and update status
       - Check for uncommitted changes in trunk branch before committing
       - Stage only the work item file change (file is now at new location)
       - Build commit message from template (support variables: {type}, {id}, {title}, {move_to})
       - Commit on trunk branch (configured or auto-detected)
       - Abort worktree creation if commit fails
     - `commit_and_push`: Commit and push on trunk branch before worktree creation
       - Call `moveWorkItem(cfg, workItemID, move_to, false)` from `move.go` (with `commitFlag=false`) to move file and update status
       - Same commit steps as `commit_only` plus push to origin/<trunk_branch>
       - Abort worktree creation if commit or push fails
     - `commit_only_branch`: Commit on new branch after worktree creation
       - Create worktree and branch first
       - Call `moveWorkItem(cfg, workItemID, move_to, false)` from `move.go` (with `commitFlag=false`) to move file and update status
       - Then stage work item file change (file is now at new location)
       - Build commit message from template
       - Commit on the new branch (not trunk branch)
       - **Important - File Location Clarification:**
         - **Status update happens in the original repository:** The work item file is physically moved within the original repository's `.work/` directory structure (e.g., from `.work/0_backlog/` to `.work/1_doing/`)
         - **File remains in original `.work/` directory:** The work item file stays in the original repository's `.work/` directory structure, not in the worktree directory
         - **Worktree is a separate checkout:** The worktree is a separate checkout of the same repository, so it sees the same `.work/` directory structure (worktrees don't have their own separate `.work/` directory)
         - **Commit happens in worktree context:** The commit is made in the worktree (on the new branch), but the file being committed is the same file in the original repository's `.work/` directory that the worktree can see
         - **Why this works:** Since worktrees share the same repository, they all see the same `.work/` directory structure. When we commit in the worktree, we're committing changes to files in the shared repository, including the work item file that was moved in the original repository's `.work/` directory

### Security Considerations

1. **Path Validation**
   - Use `filepath.Clean` and validate paths (similar to `safeReadFile`)
   - Prevent path traversal attacks
   - Validate worktree locations are within expected directories

2. **Command Execution**
   - Use centralized `executeCommand` function (see architectural recommendation) which internally uses `exec.CommandContext` with timeouts
   - Sanitize work item titles before using in git commands
   - Validate git command outputs before proceeding

3. **File Permissions**
   - Set appropriate permissions on created directories
   - Follow existing patterns from `utils.go` (0o600 for files, 0o700 for dirs)

### Testing Strategy

1. **Unit Tests**
   - Test title sanitization function
   - Test work item metadata extraction
   - Test configuration parsing
   - Test IDE detection logic

2. **Integration Tests**
   - Test git worktree creation in temporary repository
   - Test branch creation and checkout
   - Test sibling projects workflow
   - Test error handling scenarios

3. **E2E Tests**
   - Add to `kira_e2e_tests.sh`
   - Test full workflow: start → work → save → move
   - Test with sibling projects configuration
   - Test IDE opening (mock or skip in CI)

### Dependencies

- No new external dependencies required
- Use existing `os/exec` for git commands
- Use existing `gopkg.in/yaml.v3` for configuration

### Performance Considerations

- Worktree creation is I/O bound, should complete in < 5 seconds
- Standalone/Monorepo: Single worktree creation (fast)
- Polyrepo: Multiple worktrees processed sequentially to avoid resource contention
- IDE opening is asynchronous (don't wait for IDE to fully load)

### Future Enhancements

1. **Worktree Management**
   - `kira list-worktrees` command to show active worktrees
   - `kira cleanup-worktrees` command to remove stale worktrees
   - Automatic cleanup of worktrees for completed work items

2. **Advanced IDE Integration**
   - Pre-configure IDE settings per work item
   - Open specific files from work item
   - Set up debug configurations

3. **Multi-Branch Support**
   - Support creating multiple branches per work item
   - Branch naming strategies (feature/, fix/, etc.)

## Release Notes

### New Feature: `kira start` Command

The `kira start` command automates the creation of isolated git worktrees for parallel development work. This is especially useful for agentic workflows where multiple tasks are worked on simultaneously.

**Key Features:**
- Automatically creates git worktree and branch from work item
- **Automatic workspace behavior inference** - no `type` field needed:
  - No `workspace.projects` → standalone (single worktree)
  - `workspace.projects` without `path` fields → monorepo (LLM context, single worktree)
  - `workspace.projects` with `path` fields → polyrepo (multiple worktrees)
  - Projects sharing `repo_root` → grouped into single worktree
- Integrates with any IDE via configuration (`ide.command` and `ide.args`)
- Optionally moves work item to configured status folder (`start.move_to`, defaults to "doing"). Behavior is controlled by `start.status_action` configuration (defaults to `commit_and_push`, which updates status, commits, and pushes to trunk branch)
- Flexible commit options: none, commit_only, commit_and_push, or commit_only_branch
- Configurable draft PR settings per workspace and project

**Usage:**
```bash
kira start <work-item-id>
```

**Configuration:**

Standalone (default - no config needed):
```yaml
# No workspace config = standalone mode
# Status update happens locally by default
ide:
  command: "cursor"
```

With status action enabled:
```yaml
git:
  trunk_branch: develop           # optional: defaults to auto-detect (main/master)

start:
  status_action: commit_and_push # options: none | commit_only | commit_and_push | commit_only_branch
  status_commit_message: "Start work on {type} {id}: {title}"

ide:
  command: "cursor"
```

Custom trunk branch:
```yaml
git:
  trunk_branch: production       # use "production" instead of main/master
```

Examples of different status actions:
```yaml
# Commit only (no push)
start:
  status_action: commit_only

# Commit and push to trunk branch before worktree creation
start:
  status_action: commit_and_push

# Commit on the new branch after worktree creation
start:
  status_action: commit_only_branch
```

Monorepo (projects config provides LLM context only):
```yaml
workspace:
  architecture_doc: ./ARCHITECTURE.md
  projects:
    - name: frontend
      kind: app
      description: Customer-facing UI application
      # No path field = describes current repo structure
    - name: backend
      kind: service
      description: API service layer
      # No path field = describes current repo structure

ide:
  command: "cursor"
```
Note: Behavior inferred from projects without `path` fields → monorepo (LLM context)

Polyrepo (creates multiple worktrees):
```yaml
workspace:
  # worktree_root omitted - defaults to ../../{parent_dir}_worktrees (derived from common parent of all projects)
  draft_pr: false                    # workspace default
  projects:
    - name: frontend
      path: ../frontend              # path field = polyrepo behavior
      draft_pr: true                 # override: always draft PRs
      trunk_branch: develop          # optional: per-project trunk branch override (uses "develop" instead of workspace default)
    - name: backend
      path: ../backend               # path field = polyrepo behavior
      # uses workspace default (draft_pr: false)
      # uses workspace trunk branch (or auto-detected if not configured)

ide:
  command: "cursor"
```
Note: Behavior inferred from projects with `path` fields → polyrepo (multi-repo)
Note: See "Worktree Location" section for worktree_root defaults

Polyrepo with repo_root grouping (shared worktree for grouped projects - monorepo or nested folder structures):
```yaml
workspace:
  # worktree_root omitted - defaults to ../../{parent_dir}_worktrees (derived from common parent of all projects)
  projects:
    - name: frontend
      path: ../monorepo/frontend     # path field = polyrepo behavior
      repo_root: ../monorepo     # Groups with backend
    - name: backend
      path: ../monorepo/backend      # path field = polyrepo behavior
      repo_root: ../monorepo     # Same root = shares worktree with frontend
    - name: orders-service
      path: ../orders-service        # path field = polyrepo behavior
      # No repo_root = gets own worktree

ide:
  command: "cursor"
```
Note: Behavior inferred from projects with `path` fields → polyrepo (multi-repo)
Grouping: Projects sharing `repo_root` → grouped into single worktree
Note: See "Worktree Location" section for worktree_root defaults

**Breaking Changes:**
None - this is a new command with no impact on existing functionality.

