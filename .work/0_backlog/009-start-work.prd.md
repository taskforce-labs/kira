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
   - **Work item ID validation:** Validate that the work item ID matches the format specified in `cfg.Validation.IDFormat` (default: `"^\\d{3}$"` - three digits) using the same validation function as `kira new` command (`validateIDFormat` from `internal/validation/validator.go`). If validation fails, abort with error: `"Error: Invalid work item ID '{id}'. Work item ID format is invalid (expected format: {id-format}). Work item IDs must match the configured format."` (where `{id-format}` is the value from `cfg.Validation.IDFormat`). This validation occurs before any file system operations or worktree creation.
   - **Path traversal protection:** The work item ID validation also protects against path traversal attacks, as IDs that don't match the expected format (e.g., containing `../` or other invalid characters) will be rejected before being used in file paths.
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
   - **Validate current branch is trunk:** Check if the current branch (from `git rev-parse --abbrev-ref HEAD`) matches the determined trunk branch. If not, fail with error: `"Error: Start command can only be run from trunk branch. Please checkout the trunk branch and try again."` This validation occurs before any pull operations or worktree creation.
   - **Pull latest changes from origin** to ensure trunk branch is up-to-date before any operations:
     - **Remote name resolution:** See "Remote Name Resolution" section for priority order
       - For main repository: Use `git.remote` if configured, otherwise default to "origin"
       - For polyrepo projects: Use `project.remote` if configured, then `git.remote` if configured, then "origin" as final default
     - Check if remote exists using `git remote get-url <remote-name>`; if command fails (no remote configured):
       - Log warning: "Warning: No remote '{remote-name}' configured. Skipping pull step. Worktree will be created from local trunk branch."
       - Skip pull step, continue with worktree creation
     - For polyrepo: For each project, determine remote name using priority order (see "Remote Name Resolution" section)
       - Check each project repository independently using its configured remote name
       - Skip pull for projects without remote, log warning per project with the remote name that was checked
     - Check for uncommitted changes in trunk branch; if found, abort with error: "Error: Trunk branch has uncommitted changes. Cannot proceed with pull operation. Commit or stash changes before starting work."
     - **Note:** Trunk branch checkout is not needed here since validation ensures we're already on trunk branch (see trunk branch validation step above). The command must be run from the trunk branch.
     - Run `git fetch <remote-name> <trunk_branch>` to fetch latest changes (where remote-name is determined using priority order - see "Remote Name Resolution" section)
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
     - Ignore `ide.args` from config (flag value takes precedence)
     - **Shell expansion support:** Users can pass arguments via shell expansion (e.g., `--ide "cursor --new-window"`). The flag value can include both the command and its arguments, and will be executed as-is by the shell.
     - Execute: `{flag-command} {worktree-path}` (where `{flag-command}` may include arguments if provided via shell expansion)
   - **Config-based:** If neither flag provided, use `ide.command` from `kira.yml` - no auto-detection
   - If no IDE command found (flag or config): Skip IDE opening, log info message "Info: No IDE configured. Worktree created at {path}. Configure `ide.command` in kira.yml or use `--ide <command>` flag to automatically open IDE."
   - For standalone or monorepo: Open IDE in the worktree directory (single repository)
   - For polyrepo: Open IDE at the worktree root (`{worktree_root}/{work-item-id}-{kebab-case-work-item-title}`) with all project worktrees
   - IDE opens before setup commands run (allows user to start working while setup runs in background)

6. **Setup Commands/Scripts**
   - **Main project setup:** If `workspace.setup` is configured in `kira.yml`:
     - **Main project definition:** The main project is the repository where `kira.yml` is located
     - **Note:** The main project is never listed in `workspace.projects` (it's the repository containing `kira.yml`), but it always gets a worktree created
     - Execute setup command/script in main project worktree directory
     - For standalone/monorepo: Run in `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}`
     - For polyrepo: Run in main project worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/main/` (or `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{main-project-name}/`)
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
     - **Note:** This check only applies if `status_action` is not `none` (if `status_action` is `none`, skip this check entirely since no status update will occur)
     - This check happens after pulling latest changes to ensure we're checking the most up-to-date work item status
     - **Check logic:**
       - If work item status matches configured `start.move_to` (defaults to "doing") and `--skip-status-check` flag is not provided:
         - Fail with error: "Error: Work item {id} is already in '{move_to}' status. Work item status matches target status. Use --skip-status-check to restart work or review elsewhere." (where {move_to} is the configured value)
         - Abort command before worktree creation (step 7)
         - **Note:** This check applies regardless of `status_action` timing (whether status update happens at step 6 for `commit_only`/`commit_and_push`, or at step 8 for `commit_only_branch`), because the status will be updated eventually
       - If work item status matches configured `start.move_to` and `--skip-status-check` flag is provided:
         - Skip status update step (step 6 for `commit_only`/`commit_and_push`, or step 8 for `commit_only_branch`), proceed directly to worktree creation (step 7)
         - Allows resuming work or reviewing work item in different location
       - If work item status does not match configured `start.move_to`: Continue with normal status update flow (step 6 for `commit_only`/`commit_and_push`, or step 8 for `commit_only_branch`)
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
   - **Default behavior:** Creates separate worktrees for each repository, including the main project (repository where `kira.yml` is located)
   - **Main project worktree:** The main project always gets a worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/main/` (or `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{main-project-name}/` if project name can be derived)
   - Worktree per project: `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{project.mount}/` (see "Worktree Location" section for `worktree_root` defaults)
   - Branch: `{work-item-id}-{kebab-case-work-item-title}` (same branch name in each repo, including main project)
   - **Repo Root Grouping:** Projects sharing the same `repo_root` are grouped into a single worktree:
     - **Purpose 1 - Monorepo grouping:** When multiple projects share the same root directory (e.g., a monorepo), they share one worktree
     - **Purpose 2 - Nested folder structures:** When repos are configured in nested folder structures, `repo_root` specifies the common root directory
     - Create ONE worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{sanitized-repo-root}/`
       - `sanitized-repo-root` is created by: extracting directory name from `repo_root` using `filepath.Base(repo_root)`, then applying kebab-case sanitization (lowercase, replace spaces/underscores with hyphens) - see "Path Sanitization" section in "Worktree Location" for details
     - Create ONE branch in that worktree: `{work-item-id}-{kebab-case-work-item-title}`
     - All projects with the same `repo_root` share that worktree and branch
     - Skip worktree creation for subsequent projects with the same `repo_root`
   - IDE opens at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}` with all project worktrees (including main project)

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
        - `sanitized-repo-root` is created by: extracting directory name from `repo_root` using `filepath.Base(repo_root)`, then applying kebab-case sanitization (lowercase, replace spaces/underscores with hyphens) - see "Path Sanitization" section in "Worktree Location" for details
      - Projects without `repo_root`: Each gets own worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{project.mount}/`
        - `project.mount` values are also sanitized using kebab-case (same algorithm) if they contain spaces or special characters
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
    # Note: Main project is never listed in workspace.projects, but it always gets a worktree created
    #   For polyrepo, runs in main project worktree at {worktree_root}/{work-item-id}-{kebab-case-work-item-title}/main/
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
        remote: upstream               # optional: override remote name for this project (see "Remote Name Resolution" section for priority order)
        setup: ./scripts/setup.sh      # optional: command or script path for project-specific setup
   ```

   **Configuration Defaults:**
   - If `git` section is omitted: the `trunk_branch` config field defaults to auto-detect (check "main" first, then "master")
   - If `git.trunk_branch` is omitted or empty: auto-detect (check "main" first, then "master")
   - If both "main" and "master" branches exist: fail with error: "Error: Both 'main' and 'master' branches exist. Cannot auto-detect trunk branch. Configure `git.trunk_branch` explicitly in kira.yml to specify which branch to use."
   - If `start` section is omitted: `status_action` defaults to `commit_and_push` (meaning status is updated, committed, and pushed to trunk branch)
   - If `start.move_to` is omitted or empty: defaults to `"doing"`
   - **Validation:** After applying defaults, validate that `start.move_to` (or default "doing") is a valid status key in `cfg.StatusFolders`. If invalid, fail with error at config load time: `"Error: Invalid status '{invalid_status}'. Status must be one of the following: {valid_statuses}. Please check your configuration and try again."` (where `{valid_statuses}` is a comma-separated list of all keys in `cfg.StatusFolders`, sorted alphabetically)
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
   ├── main/              (worktree for main project - repository where kira.yml is located)
   ├── frontend/          (separate git worktree from ../frontend repo)
   ├── orders-service/    (separate git worktree from ../orders-service repo)
   └── ...
   Branch: {work-item-id}-{kebab-case-work-item-title} (same branch name created in each repo, including main project)
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
   ├── main/              (worktree for main project - repository where kira.yml is located)
   ├── monorepo/          (ONE worktree at repo_root, shared by frontend + backend)
   │   ├── frontend/      (component within monorepo worktree)
   │   └── backend/       (component within monorepo worktree)
   ├── orders-service/    (separate git worktree - standalone)
   └── ...
   Branch: {work-item-id}-{kebab-case-work-item-title} (one branch in main project worktree, one branch in monorepo worktree, one branch in orders-service worktree)
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
     # See "Remote Name Resolution" section for priority order
     remote: origin
   ```

3. **Status Management Configuration**

   The `start` configuration controls how work item status changes are handled when starting work.

   ```yaml
   start:
     # Optional: status folder to move work item to (defaults to "doing")
     # Specifies which status folder the work item should be moved to when starting work
     # Must be a valid status key from cfg.StatusFolders (e.g., "backlog", "todo", "doing", "review", "done", "archived")
     # Invalid values will cause config load to fail with error
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
     - Ignore `ide.args` from config (flag value takes precedence)
     - **Shell expansion support:** Users can pass arguments via shell expansion (e.g., `--ide "cursor --new-window"`). The flag value can include both the command and its arguments, and will be executed as-is by the shell.
     - Execute: `{flag-command} {worktree-path}` (where `{flag-command}` may include arguments if provided via shell expansion)
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
     - When using `--ide` flag: Arguments can be included in the flag value via shell expansion (e.g., `--ide "cursor --new-window"`). The command is executed as-is by the shell, allowing users to pass any arguments needed.
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
         - `sanitized-repo-root` is created by: extracting directory name from `repo_root` using `filepath.Base(repo_root)`, then applying kebab-case sanitization (lowercase, replace spaces/underscores with hyphens) - see "Path Sanitization" section above
       - Projects without `repo_root`: `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{project.mount}/`
         - `project.mount` values are also sanitized using kebab-case (same algorithm) if they contain spaces or special characters
     - **Example:** `../../monorepo_worktrees/123-fix-bug/monorepo/` (for grouped projects) or `../../monorepo_worktrees/123-fix-bug/frontend/` (for standalone project)

   **Override Mechanism:**

   You can override the default derivation by explicitly specifying `worktree_root` in your `kira.yml`:

   ```yaml
   workspace:
     worktree_root: ../my-custom-worktrees  # Overrides default derivation
   ```

   **Path Sanitization:**

   All path values (`worktree_root`, `repo_root`, `project.path`) must be sanitized and validated before use, following the same patterns as other file operations in the codebase (see `docs/security/golang-secure-coding.md`):

   - **Path cleaning:** Use `filepath.Clean()` to normalize paths and remove path traversal attempts (`..`, `.`, etc.)
   - **Absolute path resolution:** Resolve all paths to absolute paths using `filepath.Abs()` before validation
   - **Path traversal validation:** After cleaning, check that the resolved path doesn't contain `..` components (if `filepath.Clean()` still results in a path containing `..` after resolution, this indicates an invalid path)
   - **Directory name sanitization:** For directory names derived from `repo_root` values (used in `sanitized-repo-root`):
     - Extract the directory name using `filepath.Base(repo_root)` to get the final directory component
     - Apply kebab-case sanitization (same algorithm as title sanitization): convert to lowercase, replace spaces and underscores with hyphens
     - This ensures directory names are safe for filesystem use and consistent with other sanitized names in the codebase
   - **Invalid path characters:** Paths containing invalid filesystem characters (OS-specific) should be rejected with appropriate error messages
   - **Error handling:** If path sanitization or validation fails, abort with error: `"Error: Invalid path '{path}'. Path contains invalid characters or path traversal attempts. Please check your configuration and try again."`

   **Path Resolution:**

   - All paths are resolved relative to the repository root where `kira.yml` is located
   - Relative paths use `../` to go up directory levels
   - Absolute paths are supported but not recommended (reduces portability)
   - Paths are sanitized and normalized using `filepath.Clean()` before use (see "Path Sanitization" section above)

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

### Remote Name Resolution

**Remote Name Resolution Priority:**

Remote name resolution follows a consistent priority order across all operations (pull, push, etc.):

**For Main Repository (Standalone/Monorepo):**
1. Use `git.remote` if configured in `kira.yml`
2. Otherwise, default to `"origin"`

**For Polyrepo Projects:**
For each project in `workspace.projects`, determine remote name in order of precedence:
1. `project.remote` if configured (project-specific override)
2. `git.remote` if configured (workspace default)
3. `"origin"` as final default

**Examples:**
- Standalone with `git.remote: upstream` → uses `"upstream"`
- Standalone without `git.remote` → uses `"origin"`
- Polyrepo project with `project.remote: upstream` → uses `"upstream"` for that project
- Polyrepo project without `project.remote` but with `git.remote: github` → uses `"github"` for that project
- Polyrepo project without `project.remote` or `git.remote` → uses `"origin"` for that project

**Note:** This priority order applies consistently to all git operations that require a remote name (pull, push, etc.).

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
   - **Invalid `start.move_to` value:** Error: `"Error: Invalid status '{invalid_status}'. Status must be one of the following: {valid_statuses}. Please check your configuration and try again."` (where `{invalid_status}` is the configured `start.move_to` value, and `{valid_statuses}` is a comma-separated list of all keys in `cfg.StatusFolders`, sorted alphabetically). This error occurs at config load time, before command execution begins.
   - **Invalid work item ID:** Error: `"Error: Invalid work item ID '{id}'. Work item ID contains invalid characters or doesn't match expected format. Work item IDs must be valid identifiers."` (occurs before any sanitization or worktree creation)
   - **Empty sanitized title:** Error: `"Error: Work item '{id}' title sanitization resulted in empty string. Title cannot be sanitized to a valid name. Please update the work item title to include valid characters."` (occurs after kebab-case conversion if result is empty)
   - **Invalid path:** Error: `"Error: Invalid path '{path}'. Path contains invalid characters or path traversal attempts. Please check your configuration and try again."` (occurs when `worktree_root`, `repo_root`, or `project.path` values fail path sanitization or validation - see "Path Sanitization" section in "Worktree Location" for details)
   - **Not on trunk branch:** Error: `"Error: Start command can only be run from trunk branch. Please checkout the trunk branch and try again."` (occurs after trunk branch determination, before any pull operations or worktree creation)
   - **Work item not found:** Error: `"Error: Work item '{id}' not found. No work item file exists with that ID. Check the work item ID and try again."`
   - **Work item missing title:** Warning: `"Warning: Work item {id} has no title field. Using work item ID '{id}' for worktree directory and branch name."` Use just work item ID as fallback, continue execution
   - **Invalid git repository:** Error: `"Error: Not a git repository. Current directory is not a git repository. Run this command from within a git repository."`
   - **Worktree path already exists:**
     - **Standalone/Monorepo:** Check if target worktree path already exists
       - If path exists and is a valid git worktree:
         - Check if worktree is for the same work item (by checking if branch name matches `{work-item-id}-{kebab-case-work-item-title}` or if work item ID is in the path)
         - If same work item: Error: `"Error: Worktree already exists at {path} for work item {id}. A worktree for this work item already exists. Use \`--override\` to remove existing worktree and create a new one, or use the existing worktree."`
         - If different work item: Error: `"Error: Worktree path {path} already exists for a different work item. Path conflicts with existing worktree. Use \`--override\` to remove existing worktree, or choose a different work item."`
       - If path exists but is not a valid git worktree: Error: `"Error: Path {path} already exists but is not a valid git worktree. Cannot create worktree at this location. Remove it manually and try again, or use \`--override\` to remove it automatically."`
       - If path doesn't exist: Proceed with worktree creation
       - **With `--override` flag:** Remove existing worktree or directory before creating new one
         - If path is a valid git worktree: Remove using `git worktree remove <path>` (or `git worktree remove --force` if worktree has uncommitted changes)
         - If path is not a valid git worktree: Remove using `os.RemoveAll`
         - If removal fails: Abort with error: `"Error: Failed to remove existing worktree at {path}. Cannot proceed with --override. {error-details}. Resolve the issue and try again."` (where `{error-details}` includes the specific error from the removal operation)
         - Only proceed with worktree creation after successful removal
     - **Polyrepo:** Check all worktree paths before creating any (all-or-nothing approach):
       - **Pre-validation (before Phase 1 worktree creation):** For each worktree that will be created (main project and all projects in `workspace.projects`):
         - Check if target worktree path already exists
         - If path exists: Determine if it's a valid git worktree and check if it's for the same work item
         - Track path status per worktree: `pathStatus map[string]PathStatus` where key is worktree path and value indicates: `not_exists`, `valid_worktree_same_item`, `valid_worktree_different_item`, or `invalid_worktree`
       - **All-or-nothing validation:** After checking all paths:
         - If ANY path exists (regardless of type) and `--override` flag is NOT provided:
           - If path is valid git worktree for same work item: Error listing all affected paths: `"Error: Worktree already exists at one or more paths for work item {id}: {path-list}. A worktree for this work item already exists. Use \`--override\` to remove existing worktrees and create new ones, or use the existing worktrees."`
           - If path is valid git worktree for different work item: Error listing all affected paths: `"Error: Worktree path(s) already exist for different work items: {path-list}. Path conflicts with existing worktrees. Use \`--override\` to remove existing worktrees, or choose a different work item."`
           - If path is not a valid git worktree: Error listing all affected paths: `"Error: Path(s) already exist but are not valid git worktrees: {path-list}. Cannot create worktrees at these locations. Remove them manually and try again, or use \`--override\` to remove them automatically."`
         - If ANY path exists and `--override` flag IS provided: Remove all existing paths before creating worktrees
           - **All-or-nothing removal:** `--override` applies to all conflicting worktree paths for the work item (main project and all projects in `workspace.projects`). All paths must be successfully removed before proceeding.
           - **Removal method:** For each conflicting path:
             - If path is a valid git worktree: Remove using `git worktree remove <path>` (or `git worktree remove --force` if worktree has uncommitted changes)
             - If path is not a valid git worktree (invalid worktree or non-worktree directory): Remove using `os.RemoveAll`
           - **Failure handling:** If removal fails for any path (e.g., worktree is locked, permission denied, file system error), abort with error: `"Error: Failed to remove existing worktree at {path}. Cannot proceed with --override. {error-details}. Resolve the issue and try again."` (where `{error-details}` includes the specific error from the removal operation). Do not continue with worktree creation if any removal fails.
           - **Success:** Only after all conflicting paths are successfully removed, proceed with worktree creation (Phase 1)
         - If NO paths exist: Proceed with worktree creation
       - **Phase 1 worktree creation:** After validation passes (or `--override` removes existing paths), create all worktrees
   - **Branch already exists:**
     - **Standalone/Monorepo:** Check if branch `{work-item-id}-{kebab-case-work-item-title}` already exists in repository before creating worktree
       - If branch exists:
         - Check what commit the branch points to
         - If branch points to trunk branch commit (same commit): Branch exists but has no commits, likely from previous worktree that was removed
           - Error: `"Error: Branch {branch-name} already exists and points to trunk. Branch exists but has no commits. Use \`--reuse-branch\` to checkout existing branch in new worktree, or delete the branch first: \`git branch -d {branch-name}\`"`
           - With `--reuse-branch` flag: Create worktree without `-b` flag, then checkout existing branch: `git worktree add <path> <trunk-branch>` followed by `git checkout {branch-name}`
         - If branch points to different commit (has commits): Branch has work
           - Error: `"Error: Branch {branch-name} already exists and has commits. Branch contains work that would be lost. Delete the branch first if you want to start fresh: \`git branch -D {branch-name}\`, or use a different work item."`
       - If branch doesn't exist: Create new branch using `git worktree add <path> -b <branch-name> <trunk-branch>`
     - **Polyrepo:** Check branch existence in each repository independently before creating any worktrees (all-or-nothing approach):
       - **Pre-validation (before Phase 1 worktree creation):** For each repository (main project and all projects in `workspace.projects`):
         - Check if branch `{work-item-id}-{kebab-case-work-item-title}` exists using `git show-ref --verify --quiet refs/heads/{branch-name}` in that repository
         - If branch exists:
           - Get branch commit hash using `git rev-parse {branch-name}` in that repository
           - Get trunk branch commit hash using `git rev-parse <trunk-branch>` in that repository (using project-specific trunk branch if configured)
           - Compare commits:
             - If commits match (branch points to trunk): Branch exists but has no commits
             - If commits don't match (branch has commits): Branch has work
         - Track branch status per repository: `branchStatus map[string]BranchStatus` where key is repository path and value indicates: `not_exists`, `points_to_trunk`, or `has_commits`
       - **All-or-nothing validation:** After checking all repositories:
         - If ANY branch has commits: Abort with error listing all repositories where branch has commits: `"Error: Branch {branch-name} already exists and has commits in one or more repositories: {repo-list}. Branch contains work that would be lost. Delete the branches first if you want to start fresh, or use a different work item."`
         - If ALL branches point to trunk (or don't exist): Continue with worktree creation
           - If `--reuse-branch` flag is provided: Use existing branches in Phase 2 (checkout existing branches instead of creating new ones)
           - If `--reuse-branch` flag is NOT provided: Abort with error listing all repositories where branch exists: `"Error: Branch {branch-name} already exists and points to trunk in one or more repositories: {repo-list}. Branch exists but has no commits. Use \`--reuse-branch\` to checkout existing branches in new worktrees, or delete the branches first: \`git branch -d {branch-name}\` (run in each repository)"`
         - If SOME branches exist (point to trunk) and SOME don't exist: Treat as if all point to trunk (apply `--reuse-branch` logic consistently across all repositories)
       - **Phase 2 branch creation:** After all worktrees created successfully:
         - If `--reuse-branch` flag is provided: For each worktree where branch exists, checkout existing branch instead of creating new one
         - If `--reuse-branch` flag is NOT provided: Create new branches in all worktrees (branches that existed pointing to trunk will be overwritten)

2. **Git Errors**
   - **Trunk branch not found:** Error: `"Error: Trunk branch '{branch-name}' not found. Configured branch does not exist and auto-detection failed. Verify the branch name in \`git.trunk_branch\` configuration or ensure 'main' or 'master' branch exists."`
   - **Both "main" and "master" branches exist:** Error: `"Error: Both 'main' and 'master' branches exist. Cannot auto-detect trunk branch. Configure \`git.trunk_branch\` explicitly in kira.yml to specify which branch to use."`
   - **Remote not found:** Warning: `"Warning: No remote '{remote-name}' configured. Skipping pull step. Worktree will be created from local trunk branch."` (where remote-name is determined using priority order - see "Remote Name Resolution" section), skip pull step, continue with worktree creation
   - **For polyrepo:** Each project uses its configured remote (determined using priority order - see "Remote Name Resolution" section)
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
   - **Polyrepo worktree creation failure:** If any project's worktree creation fails, attempt to rollback all successfully created worktrees using `rollbackWorktrees()` helper function. If rollback succeeds, abort command with error: `"Error: Failed to create worktree for project '{project-name}'. Worktree creation failed for one or more projects. All worktrees have been rolled back. Check git output for details and resolve any issues."` If rollback fails, abort command with error: `"Error: Failed to create worktree for project '{project-name}'. Worktree creation failed for one or more projects. Rollback also failed: {rollback-error}. Some worktrees may remain. Check git output for details and resolve any issues."`
   - **Polyrepo branch creation/checkout failure:** If any project's branch creation/checkout fails, attempt to rollback all successfully created worktrees using `rollbackWorktrees()` helper function. If rollback succeeds, abort command with error: `"Error: Failed to create/checkout branch for project '{project-name}'. Branch creation/checkout failed for one or more projects. All worktrees have been rolled back. Check git output for details and resolve any issues."` If rollback fails, abort command with error: `"Error: Failed to create/checkout branch for project '{project-name}'. Branch creation/checkout failed for one or more projects. Rollback also failed: {rollback-error}. Some worktrees may remain. Check git output for details and resolve any issues."`
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

# Override IDE command (overrides ide.command from config, ignores ide.args)
# Arguments can be passed via shell expansion: --ide "cursor --new-window"
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
- **Validation:** Perform all validation steps (steps 1-3 and step 5 status check) as normal - these don't modify state
  - Validate work item exists
  - Infer workspace behavior
  - Determine trunk branch
  - **Status check (step 5):** Perform status check if `status_action` is not `none`
    - If work item is already in target status and `--skip-status-check` is NOT provided: Show error that would occur: "Error: Work item {id} is already in '{move_to}' status. Work item status matches target status. Use --skip-status-check to restart work or review elsewhere."
    - If work item is already in target status and `--skip-status-check` IS provided: Show that status check would be skipped and status update would be skipped
    - If work item is not in target status: Show that status would be updated
- **Preview generation:** After validation passes, generate structured preview without executing:
  - Show what worktrees would be created (paths and locations)
  - Show what branches would be created (names)
  - Show what status changes would be made (current status → target status from `start.move_to`, or "no change" if `--skip-status-check` is provided or `status_action` is `none`)
  - Show what git operations would be performed (pull, commit, push)
  - Show what setup commands would be executed (if configured)
  - Show what IDE would be opened (if configured)
- **Does NOT execute:**
  - No git operations (no pull, no worktree creation, no branch creation, no commits, no pushes)
  - No file moves or status updates
  - No IDE opening
  - No setup command execution
- **Output format:** Clear, structured preview showing all planned operations
- **Exit code:**
  - 0 (success) if preview can be generated successfully
  - Non-zero if validation errors occurred (e.g., work item not found, status check would fail without `--skip-status-check`)

**Command Execution Order:**

**Note:** If `--dry-run` flag is provided:
- Perform validation steps (steps 1-3 and step 5 status check) as normal - these don't modify state
- If validation fails (e.g., work item not found, status check would fail without `--skip-status-check`): Exit with non-zero code and show error
- If validation passes: Generate preview showing what would be done, then exit with code 0
- Skip all execution steps (steps 4, 6-10) - no git operations, no file moves, no IDE opening, no setup execution

1. Validate work item exists
2. **Infer workspace behavior** (see "Workspace Behavior Inference" section for detailed logic)
3. **Determine trunk branch and validate current branch:** (See "Git Operations" section in Implementation Notes for detailed algorithm and exact commands)
   - Determine trunk branch (configured or auto-detected)
   - Validate that current branch (from `git rev-parse --abbrev-ref HEAD`) matches trunk branch
   - If not on trunk branch, fail with error: "Error: Start command can only be run from trunk branch. Please checkout the trunk branch and try again."
   - Handle conflicts when both "main" and "master" exist
4. **Pull latest changes from remote:** (See "Git Operations" section in Implementation Notes for detailed pull strategy and exact command sequences)
   - Pull latest changes from remote on trunk branch (and all project repos for polyrepo)
   - Use `git fetch` + `git merge` approach (not `git pull`) for more control
   - Handle missing remotes, uncommitted changes, merge conflicts, and network errors
5. **Check work item status:** (Only if `status_action` is not `none` - if `status_action` is `none`, skip this step entirely)
   - Get configured status folder from `start.move_to` (defaults to "doing" if not configured)
   - **Note:** This check applies regardless of when status update occurs (step 6 for `commit_only`/`commit_and_push`, or step 8 for `commit_only_branch`), because the status will be updated eventually
   - If work item status already matches configured status and `--skip-status-check` flag is not provided:
     - Fail with error: "Error: Work item {id} is already in '{move_to}' status. Work item status matches target status. Use --skip-status-check to restart work or review elsewhere." (where {move_to} is the configured value)
   - If work item status matches configured status and `--skip-status-check` flag is provided:
     - Skip status update step (step 6 for `commit_only`/`commit_and_push`, or step 8 for `commit_only_branch`), proceed directly to worktree creation
     - Allows resuming work or reviewing work item in different location
   - If work item status does not match configured status: Continue with normal flow (status update will occur at step 6 for `commit_only`/`commit_and_push`, or step 8 for `commit_only_branch`)
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
     - **Main project definition:** The main project is the repository where `kira.yml` is located
     - **Note:** The main project is never listed in `workspace.projects` (it's the repository containing `kira.yml`), but it always gets a worktree created
     - Execute setup command/script in main project worktree directory
     - For standalone/monorepo: Run in `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}`
     - For polyrepo: Run in main project worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/main/` (or `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{main-project-name}/`)
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
   - [ ] Performs all validation steps (steps 1-3 and step 5 status check) as normal - these don't modify state
   - [ ] **Status check in dry-run:**
     - [ ] If `status_action` is not `none`: Performs status check (step 5) during dry-run
     - [ ] If work item is already in target status and `--skip-status-check` is NOT provided: Exits with non-zero code and shows error that would occur
     - [ ] If work item is already in target status and `--skip-status-check` IS provided: Shows in preview that status check would be skipped and status update would be skipped
     - [ ] If work item is not in target status: Shows in preview that status would be updated from current to target status
     - [ ] If `status_action` is `none`: Skips status check and shows "no status change" in preview
   - [ ] Shows what worktrees would be created (paths and locations)
   - [ ] Shows what branches would be created (names)
   - [ ] Shows what status changes would be made (current status → target status from `start.move_to`, or "no change" if `--skip-status-check` is provided or `status_action` is `none`)
   - [ ] Shows what git operations would be performed (pull, commit, push)
   - [ ] Shows what setup commands would be executed (if configured)
   - [ ] Shows what IDE would be opened (if configured)
   - [ ] Does NOT execute any git operations (no pull, no worktree creation, no branch creation, no commits, no pushes)
   - [ ] Does NOT move files or update status
   - [ ] Does NOT open IDE
   - [ ] Does NOT run setup commands
   - [ ] Output format is clear and structured
   - [ ] Exit code is 0 if preview can be generated successfully, non-zero if validation errors occurred (e.g., work item not found, status check would fail without `--skip-status-check`)

1. **Work Item Resolution**
   - [ ] Command accepts work item ID and locates the corresponding file
   - [ ] Command fails with clear error if work item ID not found
   - [ ] Command extracts title and status from work item metadata correctly
   - [ ] If title is missing or "unknown": Uses work item ID as fallback, logs warning "Warning: Work item {id} has no title field. Using work item ID '{id}' for worktree directory and branch name."
   - [ ] Status check only executes if `status_action` is not `none` (if `status_action` is `none`, skips status check entirely)
   - [ ] Command checks work item status after git pull (step 4) and before worktree creation (step 7) - executes as step 5
   - [ ] Status check applies regardless of when status update occurs (step 6 for `commit_only`/`commit_and_push`, or step 8 for `commit_only_branch`), because status will be updated eventually
   - [ ] Command fails with clear error if work item is already in configured status (`start.move_to`, defaults to "doing") (unless `--skip-status-check` flag)
   - [ ] `--skip-status-check` flag allows command to proceed when work item is in target status
   - [ ] When `--skip-status-check` is used: Skips status update step (step 6 for `commit_only`/`commit_and_push`, or step 8 for `commit_only_branch`), proceeds directly to worktree creation (step 7)

2. **Git Worktree Creation**
   - [ ] Determines trunk branch: uses `git.trunk_branch` if configured, otherwise auto-detects (main/master)
   - [ ] If both "main" and "master" branches exist: fails with clear error asking user to configure `git.trunk_branch`
   - [ ] Validates trunk branch exists, errors clearly if not found
   - [ ] Determines remote name using priority order (see "Remote Name Resolution" section)
   - [ ] Checks for remote existence using `git remote get-url <remote-name>` before pulling (where remote-name is determined using priority order)
   - [ ] If remote doesn't exist: Logs warning "Warning: No remote '{remote-name}' configured. Skipping pull step. Worktree will be created from local trunk branch." and skips pull step
   - [ ] Continues with worktree creation even if remote doesn't exist
   - [ ] For polyrepo: Determines remote name per project using priority order (see "Remote Name Resolution" section)
   - [ ] For polyrepo: Checks each project repository independently; skips pull for projects without remote (logs warning per project with the remote name that was checked)
   - [ ] Checks for uncommitted changes in trunk branch before pull; aborts with clear error if found
   - [ ] Validates that current branch is trunk branch before any operations (does not checkout trunk branch - command must be run from trunk branch)
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
   - [ ] **Standalone/Monorepo:** Checks if worktree path already exists before creation
   - [ ] **Standalone/Monorepo:** If worktree exists for same work item: Errors with message suggesting `--override` flag
   - [ ] **Standalone/Monorepo:** If worktree exists for different work item: Errors with message suggesting `--override` flag
   - [ ] **Standalone/Monorepo:** If path exists but is not valid git worktree: Errors with message suggesting manual removal or `--override`
   - [ ] **Standalone/Monorepo:** With `--override` flag: Removes existing worktree (using `git worktree remove` for valid worktrees, `os.RemoveAll` for invalid paths) before creating new one
   - [ ] **Standalone/Monorepo:** With `--override` flag: If removal fails, aborts with error and does not create worktree
   - [ ] **Polyrepo:** Checks all worktree paths before creating any (all-or-nothing approach)
   - [ ] **Polyrepo:** Validates all paths before Phase 1 worktree creation - if ANY path exists and `--override` is NOT provided: Aborts with error listing all affected paths
   - [ ] **Polyrepo:** If ANY path exists and `--override` IS provided: Removes all existing paths (valid worktrees using `git worktree remove`, invalid paths using `os.RemoveAll`) before creating worktrees
   - [ ] **Polyrepo with --override:** `--override` applies to all conflicting worktree paths for the work item (main project and all projects in `workspace.projects`)
   - [ ] **Polyrepo with --override:** All paths must be successfully removed before proceeding - if removal fails for any path, aborts with error and does not create any worktrees
   - [ ] **Polyrepo:** Error messages list all affected paths when multiple paths conflict
   - [ ] Command handles git repository detection correctly

3. **Branch Creation**
   - [ ] **Standalone/Monorepo:** Checks if branch `{work-item-id}-{kebab-case-work-item-title}` already exists in repository before creating worktree
   - [ ] **Polyrepo:** Checks branch existence in each repository independently before creating any worktrees (all-or-nothing approach)
   - [ ] **Polyrepo:** Validates all repositories before Phase 1 worktree creation - if ANY branch has commits, aborts with error listing all affected repositories
   - [ ] **Polyrepo:** If ALL branches point to trunk (or don't exist) and `--reuse-branch` is NOT provided: Aborts with error listing all repositories where branch exists
   - [ ] **Polyrepo:** If SOME branches exist and SOME don't: Treats consistently (applies `--reuse-branch` logic across all repositories)
   - [ ] Creates branch with format: `{work-item-id}-{kebab-case-work-item-title}`
   - [ ] Branch name is valid for git (no invalid characters)
   - [ ] Branch is checked out in the new worktree
   - [ ] If branch exists and points to trunk commit: Errors with message suggesting `--reuse-branch` flag or branch deletion
   - [ ] If branch exists and has commits: Errors with message requiring branch deletion or different work item
   - [ ] With `--reuse-branch` flag: Creates worktree without `-b` flag, changes to worktree directory, then checks out existing branch using `git checkout {branch-name}`
   - [ ] **Polyrepo with `--reuse-branch`:** For each worktree where branch exists, checks out existing branch instead of creating new one
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
   - [ ] Polyrepo: Always creates worktree for main project (repository where `kira.yml` is located) at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/main/`
   - [ ] Polyrepo: Groups projects by `repo_root` value
   - [ ] Polyrepo: Creates ONE worktree at `repo_root` for each unique root value
   - [ ] Polyrepo: Creates ONE branch per repo_root group (shared by all projects in group)
   - [ ] Polyrepo: Skips worktree creation for subsequent projects with the same `repo_root`
   - [ ] Polyrepo: Creates separate worktree for each project without `repo_root` (standalone)
   - [ ] Polyrepo: Uses consistent branch naming: `{work-item-id}-{kebab-case-work-item-title}` across all worktrees (including main project)
   - [ ] Polyrepo: Organizes worktrees at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{main/ or repo_root or project.mount}/`
   - [ ] Polyrepo: Validates all project repositories exist before creating any worktrees
   - [ ] Polyrepo: Errors clearly if project repository not found, aborts entire command (all-or-nothing)
   - [ ] Polyrepo: Checks branch existence in each repository independently before creating any worktrees (all-or-nothing approach)
   - [ ] Polyrepo: If ANY branch has commits: Aborts with error listing all repositories where branch has commits (all-or-nothing)
   - [ ] Polyrepo: If ALL branches point to trunk (or don't exist) and `--reuse-branch` is NOT provided: Aborts with error listing all repositories where branch exists
   - [ ] Polyrepo: If ALL branches point to trunk (or don't exist) and `--reuse-branch` IS provided: Continues with worktree creation, checks out existing branches in Phase 2
   - [ ] Polyrepo: If SOME branches exist and SOME don't: Treats consistently (applies `--reuse-branch` logic across all repositories)
   - [ ] Polyrepo: Creates all worktrees first (Phase 1), then creates branches (Phase 2) for easier rollback
   - [ ] Polyrepo: If any worktree creation fails: Rolls back all successfully created worktrees using `git worktree remove`, aborts command with error
   - [ ] Polyrepo: If any branch creation/checkout fails: Rolls back all successfully created worktrees using `git worktree remove`, aborts command with error
   - [ ] Polyrepo with `--reuse-branch`: For each worktree where branch exists, checks out existing branch instead of creating new one
   - [ ] Polyrepo: Tracks all successfully created worktrees for rollback purposes
   - [ ] Handles mixed setup: some projects grouped by repo_root, others standalone
   - [ ] Respects draft_pr configuration at workspace and project levels

5. **IDE Integration**
   - [ ] Checks `--no-ide` flag first (highest priority)
   - [ ] If `--no-ide` flag provided: Skips IDE opening entirely, no log messages (useful for agents)
   - [ ] If `--no-ide` not set: Checks `--ide` flag next
   - [ ] If `--ide <command>` flag provided: Uses flag value as IDE command (may include arguments via shell expansion), ignores `ide.args` from config, executes command as-is
   - [ ] If neither flag provided: Requires `ide.command` configuration in `kira.yml` - no auto-detection
   - [ ] If no IDE command found (flag or config): Skips IDE opening, logs info message, continues with worktree creation
   - [ ] Standalone/Monorepo: Opens IDE in worktree directory (if IDE command found)
   - [ ] Polyrepo: Opens IDE at worktree root (`{worktree_root}/{work-item-id}-{kebab-case-work-item-title}`) with all project worktrees (if IDE command found)
   - [ ] IDE opens with correct branch checked out (if IDE command found)
   - [ ] Command continues successfully even if IDE launch fails
   - [ ] When using config: Respects IDE configuration from `kira.yml` (`ide.command` and `ide.args`)
   - [ ] When using `--ide` flag: Overrides `ide.command` and ignores `ide.args` (arguments can be included in flag value via shell expansion, e.g., `--ide "cursor --new-window"`)
   - [ ] If IDE command not found: Logs warning and continues (worktree creation succeeds)
   - [ ] If IDE launch fails: Logs warning and continues (worktree creation succeeds)
   - [ ] Worktree creation succeeds regardless of IDE behavior
   - [ ] **Setup Commands/Scripts:**
     - [ ] If `workspace.setup` configured: Executes setup command/script in main project worktree directory (main project is repository where `kira.yml` is located)
     - [ ] Main project is never listed in `workspace.projects` (it's the repository containing `kira.yml`), but it always gets a worktree created
     - [ ] For standalone/monorepo: Runs `workspace.setup` in main project worktree
     - [ ] For polyrepo: Runs `workspace.setup` in main project worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/main/`
     - [ ] If `project.setup` configured: Executes setup command/script in project worktree directory (for polyrepo)
     - [ ] Setup runs after IDE opening (allows user to start working while setup runs in background)
     - [ ] For script paths: Executes with appropriate shell (bash for .sh, python for .py, etc.)
     - [ ] For commands: Uses centralized `executeCommand` function (or `os/exec` directly if centralized function not yet available)
     - [ ] Changes directory to worktree before execution
     - [ ] For polyrepo: Runs setups sequentially in order projects are defined
     - [ ] If setup fails: Logs error and aborts command (setup is critical for environment preparation)

6. **Status Management**
   - [ ] If `status_action: none`: Work item status is not changed, proceed to worktree creation (status check at step 5 is skipped)
   - [ ] Status check (step 5) applies for all `status_action` values except `none`, regardless of when status update occurs (step 6 for `commit_only`/`commit_and_push`, or step 8 for `commit_only_branch`)
   - [ ] If `status_action: commit_only`: Updates status to configured status folder (`start.move_to`, defaults to "doing") and commits on trunk branch before worktree creation (at step 6)
   - [ ] If `status_action: commit_and_push`: Updates status to configured status folder (`start.move_to`, defaults to "doing"), commits and pushes to trunk branch before worktree creation (at step 6)
   - [ ] If `status_action: commit_only_branch`: Updates status to configured status folder (`start.move_to`, defaults to "doing") and commits on new branch after worktree creation (at step 8)
   - [ ] When `--skip-status-check` is used with `commit_only_branch`: Skips status update at step 8, proceeds directly after worktree creation
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
   - [ ] Determines remote name using priority order (see "Remote Name Resolution" section)
   - [ ] Checks for remote existence using `git remote get-url <remote-name>` before pulling (where remote-name is determined using priority order)
   - [ ] If remote doesn't exist: Logs warning "Warning: No remote '{remote-name}' configured. Skipping pull step. Worktree will be created from local trunk branch." and skips pull step
   - [ ] Continues with worktree creation even if remote doesn't exist
   - [ ] For polyrepo: For each project, determines remote name using priority order (see "Remote Name Resolution" section)
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
   - [ ] Status check only executes if `status_action` is not `none` (if `status_action` is `none`, skips status check entirely)
   - [ ] Checks work item status after git pull (step 4) and before worktree creation (step 7) - executes as step 5
   - [ ] Status check applies regardless of when status update occurs (step 6 for `commit_only`/`commit_and_push`, or step 8 for `commit_only_branch`), because status will be updated eventually
   - [ ] If work item is already in configured status (`start.move_to`, defaults to "doing"): Fails unless `--skip-status-check` flag is provided
   - [ ] `--skip-status-check` flag allows restarting work or reviewing work item elsewhere
   - [ ] When `--skip-status-check` is used: Skips status update step (step 6 for `commit_only`/`commit_and_push`, or step 8 for `commit_only_branch`), proceeds directly to worktree creation (step 7)
   - [ ] Handles work items with special characters in title
   - [ ] Handles very long work item titles (truncates appropriately)
   - [ ] Handles work items with missing title field: Uses work item ID as fallback, logs warning "Warning: Work item {id} has no title field. Using work item ID '{id}' for worktree directory and branch name."
   - [ ] When title is missing: Worktree directory becomes `{work-item-id}` and branch becomes `{work-item-id}`
   - [ ] **Title sanitization uses same algorithm as `kira new`:** Converts to lowercase, replaces spaces and underscores with hyphens (same as `kebabCase()` function in `internal/commands/new.go`)
   - [ ] **Unicode characters:** Unicode characters are converted to lowercase as-is (no special normalization), same as `kira new`
   - [ ] **Invalid work item ID:** Command fails with error "Error: Invalid work item ID '{id}'. Work item ID contains invalid characters or doesn't match expected format. Work item IDs must be valid identifiers." if ID contains invalid characters or doesn't match format
   - [ ] **Empty sanitized title:** Command fails with error "Error: Work item '{id}' title sanitization resulted in empty string. Title cannot be sanitized to a valid name. Please update the work item title to include valid characters." if sanitization results in empty string
   - [ ] **Path sanitization:** All path values (`worktree_root`, `repo_root`, `project.path`) are sanitized using `filepath.Clean()` and validated before use (consistent with other file operations in codebase)
   - [ ] **Invalid path:** Command fails with error "Error: Invalid path '{path}'. Path contains invalid characters or path traversal attempts. Please check your configuration and try again." if path sanitization or validation fails
   - [ ] **Directory name sanitization:** `sanitized-repo-root` is created by extracting directory name from `repo_root` using `filepath.Base()` and applying kebab-case sanitization (lowercase, replace spaces/underscores with hyphens)
   - [ ] **Trunk branch validation:** Command validates that current branch (from `git rev-parse --abbrev-ref HEAD`) matches the determined trunk branch before any operations
   - [ ] **Not on trunk branch:** Command fails with error "Error: Start command can only be run from trunk branch. Please checkout the trunk branch and try again." if current branch is not the trunk branch

3. **File System**
   - [ ] Handles permission errors gracefully
   - [ ] Works with relative and absolute paths
   - [ ] Handles spaces in directory names
   - [ ] Works on Windows, macOS, and Linux
   - [ ] **Standalone/Monorepo:** Handles worktree path already exists for same work item (errors with suggestion to use `--override`)
   - [ ] **Standalone/Monorepo:** Handles worktree path already exists for different work item (errors with suggestion to use `--override`)
   - [ ] **Standalone/Monorepo:** Handles path exists but is not a valid git worktree (errors with suggestion to use `--override`)
   - [ ] **Standalone/Monorepo:** `--override` flag removes existing valid git worktree using `git worktree remove`
   - [ ] **Standalone/Monorepo:** `--override` flag removes existing invalid worktree directory using filesystem operations
   - [ ] **Standalone/Monorepo:** `--override` flag handles worktree with uncommitted changes (uses `git worktree remove --force`)
   - [ ] **Polyrepo:** Checks all worktree paths before creating any (all-or-nothing approach)
   - [ ] **Polyrepo:** If ANY path exists and `--override` is NOT provided: Aborts with error listing all affected paths
   - [ ] **Polyrepo:** If ANY path exists and `--override` IS provided: Removes all existing paths before creating worktrees (all-or-nothing: all must succeed or command aborts)
   - [ ] **Polyrepo:** Error messages list all affected paths when multiple paths conflict

4. **Setup Behavior**
   - [ ] If `workspace.setup` configured: Executes in main project worktree directory (main project is repository where `kira.yml` is located)
   - [ ] Main project is never listed in `workspace.projects` (it's the repository containing `kira.yml`), but it always gets a worktree created
   - [ ] For standalone/monorepo: Runs `workspace.setup` in main project worktree
   - [ ] For polyrepo: Runs `workspace.setup` in main project worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/main/`
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
   - [ ] If `--ide <command>` flag provided: Uses flag value (may include arguments via shell expansion), ignores `ide.args` from config, executes command as-is
   - [ ] If neither flag provided: Requires `ide.command` configuration - no auto-detection
   - [ ] If no IDE command found (flag or config): Skips IDE opening, logs info message, continues
   - [ ] All IDE behavior is controlled by `--ide` flag or `ide.command`/`ide.args` configuration (no hardcoded IDE logic)
   - [ ] When using config: User configures appropriate args for their IDE in `kira.yml` (e.g., `["--new-window"]` to open in new window)
   - [ ] When using `--ide` flag: Arguments can be included in flag value via shell expansion (e.g., `--ide "cursor --new-window"` or `--ide "$KIRA_IDE"`). Command is executed as-is by the shell.
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
   - [ ] Polyrepo: Rollback aborts immediately if any removal fails (does not continue attempting to remove other worktrees)
   - [ ] Polyrepo: If rollback fails, error message indicates which worktree failed to remove and that rollback was aborted: "Error: Failed to remove worktree at {path} during rollback. Rollback aborted. {error-details}. Resolve the issue and try again."
   - [ ] Polyrepo: Error messages clearly indicate which project failed and whether rollback succeeded or was aborted

7. **Error Scenarios** (Explicit test cases for all error conditions mentioned in Error Handling section)

   **Validation Errors:**
   - [ ] **Invalid `start.move_to` value:** Config load fails with error: "Error: Invalid status '{invalid_status}'. Status must be one of the following: {valid_statuses}. Please check your configuration and try again." (where `{invalid_status}` is the configured `start.move_to` value, and `{valid_statuses}` is a comma-separated list of all keys in `cfg.StatusFolders`, sorted alphabetically). Command does not execute.
   - [ ] **Invalid work item ID:** Command fails with error: "Error: Invalid work item ID '{id}'. Work item ID contains invalid characters or doesn't match expected format. Work item IDs must be valid identifiers." Command does not execute.
   - [ ] **Empty sanitized title:** Command fails with error: "Error: Work item '{id}' title sanitization resulted in empty string. Title cannot be sanitized to a valid name. Please update the work item title to include valid characters." Command does not execute.
   - [ ] **Invalid path:** Command fails with error: "Error: Invalid path '{path}'. Path contains invalid characters or path traversal attempts. Please check your configuration and try again." (when `worktree_root`, `repo_root`, or `project.path` values fail path sanitization or validation) Command does not execute.
   - [ ] **Not on trunk branch:** Command fails with error: "Error: Start command can only be run from trunk branch. Please checkout the trunk branch and try again." (occurs after trunk branch determination, before any pull operations or worktree creation) Command does not execute.
   - [ ] **Work item not found:** Command fails with error: "Error: Work item '{id}' not found. No work item file exists with that ID. Check the work item ID and try again."
   - [ ] **Work item missing title:** Command logs warning "Warning: Work item {id} has no title field. Using work item ID '{id}' for worktree directory and branch name." and continues execution using work item ID as fallback
   - [ ] **Invalid git repository:** Command fails with error: "Error: Not a git repository. Current directory is not a git repository. Run this command from within a git repository."
   - [ ] **Worktree path already exists (same work item):** Command fails with error: "Error: Worktree already exists at {path} for work item {id}. A worktree for this work item already exists. Use `--override` to remove existing worktree and create a new one, or use the existing worktree."
   - [ ] **Worktree path already exists (different work item):** Command fails with error: "Error: Worktree path {path} already exists for a different work item. Path conflicts with existing worktree. Use `--override` to remove existing worktree, or choose a different work item."
   - [ ] **Path exists but not valid git worktree:** Command fails with error: "Error: Path {path} already exists but is not a valid git worktree. Cannot create worktree at this location. Remove it manually and try again, or use `--override` to remove it automatically."
   - [ ] **Polyrepo worktree path already exists (same work item):** Command fails with error listing all affected paths: "Error: Worktree already exists at one or more paths for work item {id}: {path-list}. A worktree for this work item already exists. Use `--override` to remove existing worktrees and create new ones, or use the existing worktrees."
   - [ ] **Polyrepo worktree path already exists (different work item):** Command fails with error listing all affected paths: "Error: Worktree path(s) already exist for different work items: {path-list}. Path conflicts with existing worktrees. Use `--override` to remove existing worktrees, or choose a different work item."
   - [ ] **Polyrepo path exists but not valid git worktree:** Command fails with error listing all affected paths: "Error: Path(s) already exist but are not valid git worktrees: {path-list}. Cannot create worktrees at these locations. Remove them manually and try again, or use `--override` to remove them automatically."
   - [ ] **Polyrepo worktree path validation:** Checks all worktree paths before creating any (all-or-nothing approach)
   - [ ] **Polyrepo with --override:** Removes all existing paths (valid worktrees using `git worktree remove`, invalid paths using `os.RemoveAll`) before creating worktrees
   - [ ] **Polyrepo with --override:** `--override` applies to all conflicting worktree paths for the work item (main project and all projects in `workspace.projects`)
   - [ ] **Polyrepo with --override failure:** If removal fails for any path (e.g., worktree locked, permission denied), aborts with error: "Error: Failed to remove existing worktree at {path}. Cannot proceed with --override. {error-details}. Resolve the issue and try again." and does not create any worktrees
   - [ ] **Standalone/Monorepo with --override failure:** If removal fails, aborts with error: "Error: Failed to remove existing worktree at {path}. Cannot proceed with --override. {error-details}. Resolve the issue and try again." and does not create worktree
   - [ ] **Branch already exists (points to trunk):** Command fails with error: "Error: Branch {branch-name} already exists and points to trunk. Branch exists but has no commits. Use `--reuse-branch` to checkout existing branch in new worktree, or delete the branch first: `git branch -d {branch-name}`"
   - [ ] **Branch already exists (has commits):** Command fails with error: "Error: Branch {branch-name} already exists and has commits. Branch contains work that would be lost. Delete the branch first if you want to start fresh: `git branch -D {branch-name}`, or use a different work item."
   - [ ] **Polyrepo branch already exists (has commits):** Command fails with error listing all repositories: "Error: Branch {branch-name} already exists and has commits in one or more repositories: {repo-list}. Branch contains work that would be lost. Delete the branches first if you want to start fresh, or use a different work item."
   - [ ] **Polyrepo branch already exists (points to trunk, no --reuse-branch):** Command fails with error listing all repositories: "Error: Branch {branch-name} already exists and points to trunk in one or more repositories: {repo-list}. Branch exists but has no commits. Use `--reuse-branch` to checkout existing branches in new worktrees, or delete the branches first: `git branch -d {branch-name}` (run in each repository)"
   - [ ] **Polyrepo branch existence check:** Checks branch existence in each repository independently before creating any worktrees (all-or-nothing approach)
   - [ ] **Polyrepo with --reuse-branch:** For each worktree where branch exists, checks out existing branch instead of creating new one

   **Git Errors:**
   - [ ] **Trunk branch not found:** Command fails with error: "Error: Trunk branch '{branch-name}' not found. Configured branch does not exist and auto-detection failed. Verify the branch name in `git.trunk_branch` configuration or ensure 'main' or 'master' branch exists."
   - [ ] **Both 'main' and 'master' branches exist:** Command fails with error: "Error: Both 'main' and 'master' branches exist. Cannot auto-detect trunk branch. Configure `git.trunk_branch` explicitly in kira.yml to specify which branch to use."
   - [ ] **Repository without remote:** Command logs warning "Warning: No remote '{remote-name}' configured. Skipping pull step. Worktree will be created from local trunk branch." (where remote-name is determined using priority order - see "Remote Name Resolution" section) and continues with worktree creation
   - [ ] **Uncommitted changes in trunk branch (before pull):** Command fails with error: "Error: Trunk branch has uncommitted changes. Cannot proceed with pull operation. Commit or stash changes before starting work."
   - [ ] **Uncommitted changes in trunk branch (for commit_only/commit_and_push):** Command fails with error: "Error: Trunk branch has uncommitted changes. Cannot commit status change. Commit or stash changes before starting work."
   - [ ] **Pull merge conflicts:** Command fails with error: "Error: Failed to merge latest changes from {remote-name}/{trunk-branch}. Merge conflicts detected. Resolve conflicts manually and try again." (includes git output)
   - [ ] **Diverged branches:** Command fails with error: "Error: Trunk branch has diverged from {remote-name}/{trunk-branch}. Local and remote branches have different commits. Rebase or merge manually before starting work."
   - [ ] **Network errors during pull:** Command fails with error: "Error: Failed to fetch changes from {remote-name}. Network error occurred. Check network connection and try again." (includes git output)
   - [ ] **Invalid status_action value:** Command fails with error: "Error: Invalid status_action value '{value}'. Value is not recognized. Use one of: 'none', 'commit_only', 'commit_and_push', 'commit_only_branch'."
   - [ ] **Status commit failure:** Command fails with error: "Error: Failed to commit status change. Git commit operation failed. Check git output for details and resolve any issues." (includes git output, aborts worktree creation)
   - [ ] **Status push failure:** Command fails with error: "Error: Failed to push status change to {remote-name}/{trunk-branch}. Git push operation failed. Check git output for details and resolve any issues." (includes git output, aborts worktree creation)
   - [ ] **Worktree creation failure:** Command fails with error: "Error: Failed to create worktree at {path}. Git worktree creation failed. Check git output for details and resolve any issues." (includes git output)
   - [ ] **Polyrepo worktree creation failure:** Command attempts rollback. If rollback succeeds, fails with error: "Error: Failed to create worktree for project '{project-name}'. Worktree creation failed for one or more projects. All worktrees have been rolled back. Check git output for details and resolve any issues." If rollback fails, fails with error: "Error: Failed to create worktree for project '{project-name}'. Worktree creation failed for one or more projects. Rollback also failed: {rollback-error}. Some worktrees may remain. Check git output for details and resolve any issues."
   - [ ] **Polyrepo branch creation/checkout failure:** Command attempts rollback. If rollback succeeds, fails with error: "Error: Failed to create/checkout branch for project '{project-name}'. Branch creation/checkout failed for one or more projects. All worktrees have been rolled back. Check git output for details and resolve any issues." If rollback fails, fails with error: "Error: Failed to create/checkout branch for project '{project-name}'. Branch creation/checkout failed for one or more projects. Rollback also failed: {rollback-error}. Some worktrees may remain. Check git output for details and resolve any issues."
   - [ ] **Polyrepo rollback failure:** If rollback itself fails (e.g., worktree is locked, permission denied), command aborts immediately with error: "Error: Failed to remove worktree at {path} during rollback. Rollback aborted. {error-details}. Resolve the issue and try again." Does not continue attempting to remove other worktrees.
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

## Implementation Phases

This implementation uses a **horizontal slicing approach**, building complete layers of functionality from foundation to integration. Each phase implements an entire architectural layer before moving to the next, ensuring the foundation is rock-solid before adding complexity.

**Phase Boundaries = Commit Boundaries:**
- Each phase must complete with code that passes `make check` (linting, security, tests, coverage)
- No `#nosec` or other check-skipping comments allowed
- Manual commit after each phase before proceeding to next
- Each phase builds working, tested code on top of the previous phase

### Phase 0: Infrastructure Refactoring ⭐ **DO THIS FIRST**

**Goal:** Create centralized command execution utilities and establish consistent patterns before building new features.

**Why First:** Battle-testing the infrastructure on existing commands (that already have tests) is safer than introducing untested utilities alongside complex new features. This also gives us dry-run support across all commands immediately.

**Scope:**
- Create `executeCommand()` utility in `internal/commands/utils.go`
- Function signature: `func executeCommand(ctx context.Context, name string, args []string, dir string, dryRun bool) (string, error)`
- Refactor `move.go` to use `executeCommand()` for all git operations
- Refactor `save.go` to use `executeCommand()` for all git operations
- Add `--dry-run` flag to existing commands
- Implement consistent error handling patterns
- Write comprehensive tests for `executeCommand()`

**Success Criteria:**
- [ ] `executeCommand()` handles all command execution with consistent error handling
- [ ] If `dryRun=true`: Prints command without executing, returns nil
- [ ] If `dryRun=false`: Executes command using `exec.CommandContext` with proper error handling
- [ ] Returns both stdout and stderr for debugging
- [ ] `move.go` uses `executeCommand()` for all git commands
- [ ] `save.go` uses `executeCommand()` for all git commands
- [ ] `--dry-run` flag works on `move` command (shows what would happen)
- [ ] `--dry-run` flag works on `save` command (shows what would happen)
- [ ] All existing tests still pass (no regression)
- [ ] Integration tests verify refactored commands work identically
- [ ] **`make check` passes with no errors (linting, security, tests, coverage)**
- [ ] **No `#nosec` or other check-skipping comments added**
- [ ] **Code is commit-ready before moving to Phase 1**

**Deliverables:**
- `internal/commands/utils.go` with `executeCommand()` and helper functions
- Refactored `move.go` using centralized utilities
- Refactored `save.go` using centralized utilities
- Unit tests for `executeCommand()`
- Updated integration tests
- Documentation for command execution patterns

---

### Phase 1: Input Validation & Safety (Horizontal Layer)

**Goal:** Build bulletproof input validation and safety mechanisms before touching any git operations. This phase validates inputs for BOTH standalone AND polyrepo workflows.

**Scope:**
- Work item ID validation (using `cfg.Validation.IDFormat`, path traversal protection)
- Work item lookup and metadata extraction
- Title sanitization (kebab-case conversion, handle missing titles, empty checks, length limits)
- Path validation and sanitization (worktree_root, repo_root, project.path) - **for standalone AND polyrepo**
- Workspace behavior inference (standalone, monorepo, polyrepo detection)
- Worktree path collision detection (check if path exists, validate if it's a worktree) - **for standalone AND polyrepo**
- `--override` flag logic (detect conflicts, plan removal strategy) - **for standalone AND polyrepo**
- `--skip-status-check` flag logic (detect status matches)
- Configuration parsing and validation (Git, Start, Workspace configs)
- All validation error messages with consistent format
- **NO git operations** - just validation and planning

**Success Criteria:**
- [ ] Work item ID validated against `cfg.Validation.IDFormat` before any operations
- [ ] Path traversal attacks prevented (IDs with `../` rejected)
- [ ] Work item file located correctly in `.work/` directory structure
- [ ] Missing titles handled with fallback to ID (warning logged)
- [ ] Title sanitization uses kebab-case (same algorithm as `kira new`)
- [ ] Empty sanitized titles detected and rejected with clear error
- [ ] Long titles truncated with hash suffix for uniqueness
- [ ] All path values cleaned with `filepath.Clean()` and validated
- [ ] Worktree path existence detected correctly
- [ ] Valid git worktrees distinguished from invalid directories
- [ ] Work item ID in path/branch name detected (for same-item conflicts)
- [ ] `--override` flag: Plans removal of existing paths (doesn't execute yet)
- [ ] **Workspace behavior inference: Detects standalone (no workspace config)**
- [ ] **Workspace behavior inference: Detects monorepo (projects without `path` fields)**
- [ ] **Workspace behavior inference: Detects polyrepo (projects with `path` fields OR `repo_root`)**
- [ ] Configuration loads with all defaults applied correctly
- [ ] Invalid `start.move_to` rejected at config load time
- [ ] All validation errors follow format: "Error: {what}. {why}. {action}."
- [ ] Comprehensive unit tests for all validation logic (standalone AND polyrepo)
- [ ] Edge cases covered (special characters, Unicode, empty strings, long strings)
- [ ] **`make check` passes with no errors (linting, security, tests, coverage)**
- [ ] **No `#nosec` or other check-skipping comments added**
- [ ] **Code is commit-ready before moving to Phase 2**

**Configuration Required:**
```yaml
git:
  trunk_branch: main  # optional: defaults to auto-detect
  remote: origin      # optional: defaults to "origin"

start:
  move_to: doing      # optional: defaults to "doing" (validated against StatusFolders)
  status_action: none # Phase 1: only validate config, don't use yet

workspace:
  # Phase 1: Validate workspace config exists and is parseable
  # Behavior inference happens but no git operations yet
  projects:
    - name: frontend
      path: ../frontend  # polyrepo indicator
```

**Deliverables:**
- `internal/commands/start.go` with validation functions (no git operations)
- Work item metadata extraction (reuses `extractWorkItemMetadata` from `move.go`)
- Title sanitization function (same algorithm as `kira new`)
- Path validation and sanitization utilities (standalone AND polyrepo)
- Workspace behavior inference logic (standalone, monorepo, polyrepo detection)
- Worktree collision detection logic (standalone AND polyrepo)
- `internal/commands/start_test.go` with comprehensive validation tests
- Updated `internal/config/config.go` with Git, Start, and Workspace configuration structs
- Configuration validation tests
- Documentation for validation patterns

---

### Phase 2: Git Operations Layer (Horizontal Layer)

**Goal:** Implement all git operations with comprehensive error handling for BOTH standalone AND polyrepo workflows. This phase builds on validated inputs from Phase 1 and uses `executeCommand()` from Phase 0.

**Scope:**
- Trunk branch detection (configured or auto-detect main/master, handle conflicts) - **for standalone AND polyrepo**
- Per-project trunk branch override (`project.trunk_branch`) - **polyrepo only**
- Current branch validation (must be on trunk branch before any operations)
- Remote name resolution (priority order: project.remote > git.remote > "origin") - **for standalone AND polyrepo**
- Git pull operations (fetch + merge, handle uncommitted changes, conflicts, diverged branches, network errors) - **for standalone AND polyrepo**
- **Standalone git operations:**
  - Single worktree creation
  - Single branch creation/checkout
  - Worktree removal with `--override`
- **Polyrepo git operations:**
  - Main project worktree creation (always created)
  - Multiple worktree creation with repo_root grouping
  - Transaction-like behavior: Pre-validation → Phase 1 (all worktrees) → Phase 2 (all branches)
  - Rollback helper function (`rollbackWorktrees`) for all-or-nothing failure handling
  - All-or-nothing validation (paths, branches) before creating anything
  - `--override` for polyrepo: Remove all conflicting paths before creating worktrees
- Branch existence checks (detect if branch points to trunk or has commits) - **for standalone AND polyrepo**
- Branch creation (new branches) - **for standalone AND polyrepo**
- Branch checkout with `--reuse-branch` (checkout existing branches) - **for standalone AND polyrepo**
- All git error handling with consistent format
- Uses `executeCommand()` from Phase 0 for all git operations

**Success Criteria:**
- [ ] Trunk branch detection works (configured, auto-detect, conflict detection)
- [ ] Both "main" and "master" exist: Fails with clear error asking for config
- [ ] Configured trunk branch that doesn't exist: Clear error message
- [ ] Current branch validated (must be on trunk) before any operations
- [ ] Not on trunk branch: Fails with error (no auto-checkout)
- [ ] Remote existence checked before pull operations
- [ ] Missing remote: Warning logged, pull skipped, worktree creation continues
- [ ] Uncommitted changes detected: Fails with clear error before pull
- [ ] Git fetch + merge executes correctly (not `git pull`)
- [ ] Merge conflicts detected and reported with git output
- [ ] Diverged branches detected and reported with clear error
- [ ] Network errors during pull reported with git output
- [ ] Worktree created at correct path using `git worktree add`
- [ ] `--override` removes existing worktree using `git worktree remove` or `os.RemoveAll`
- [ ] `--override` handles locked worktrees and uncommitted changes (`--force`)
- [ ] Branch existence checked using `git show-ref`
- [ ] Branch pointing to trunk detected (compare commit hashes)
- [ ] Branch with commits detected: Fails with error requiring deletion
- [ ] New branch created with `git worktree add -b <branch> <trunk>`
- [ ] `--reuse-branch` creates worktree then checks out existing branch
- [ ] **Standalone: Single worktree and branch created correctly**
- [ ] **Polyrepo: Main project worktree created at `{worktree_root}/{work-item-id}-{title}/main/`**
- [ ] **Polyrepo: Projects with same `repo_root` grouped into single worktree**
- [ ] **Polyrepo: Standalone projects (no `repo_root`) get separate worktrees**
- [ ] **Polyrepo: Pre-validation checks all paths and branches before creating anything**
- [ ] **Polyrepo: All-or-nothing worktree creation (Phase 1: all worktrees, Phase 2: all branches)**
- [ ] **Polyrepo: Rollback removes all worktrees in reverse order on any failure**
- [ ] **Polyrepo: `--override` removes all conflicting paths (all-or-nothing) before creating worktrees**
- [ ] **Polyrepo: Per-project trunk_branch override works correctly**
- [ ] **Polyrepo: Per-project remote override works correctly**
- [ ] **Polyrepo: Consistent branch naming across all repos: `{work-item-id}-{title}`**
- [ ] All git operations use `executeCommand()` from Phase 0
- [ ] All error messages include git output for debugging
- [ ] Integration tests cover all git scenarios (standalone AND polyrepo, mocked git commands)
- [ ] **`make check` passes with no errors (linting, security, tests, coverage)**
- [ ] **No `#nosec` or other check-skipping comments added**
- [ ] **Code is commit-ready before moving to Phase 3**

**Configuration Required:**
```yaml
git:
  trunk_branch: main  # optional: defaults to auto-detect
  remote: origin      # optional: defaults to "origin"

start:
  move_to: doing      # optional: defaults to "doing"
  status_action: none # Phase 2: only git operations, no status management yet

workspace:
  # Phase 2: Uses workspace config for polyrepo git operations
  worktree_root: ../../my-worktrees  # optional: defaults to derived path
  projects:
    - name: frontend
      path: ../frontend
      trunk_branch: develop  # optional: per-project override
      remote: upstream       # optional: per-project override
```

**Deliverables:**
- Updated `internal/commands/start.go` with all git operations (standalone AND polyrepo)
- Git operation helper functions (trunk detection, pull, worktree, branch)
- Workspace behavior inference logic (standalone, monorepo, polyrepo)
- Rollback helper function (`rollbackWorktrees`) for polyrepo failure handling
- Pre-validation logic (paths, branches, repositories) for polyrepo
- Transaction-like execution (Phase 1 worktrees, Phase 2 branches) for polyrepo
- Error handling for all git scenarios (standalone AND polyrepo)
- Updated `internal/config/config.go` with workspace configuration
- Updated `internal/commands/start_test.go` with git operation tests (standalone AND polyrepo)
- Integration tests for git workflows (standalone AND polyrepo, mocked git commands)
- Documentation for git operation patterns

---

### Phase 3: Status Management Layer (Horizontal Layer)

**Goal:** Implement all status update functionality for BOTH standalone AND polyrepo workflows, building on validated inputs (Phase 1) and working git operations (Phase 2).

**Scope:**
- Status check after git pull (step 5 in execution order) - **for standalone AND polyrepo**
- `--skip-status-check` flag to bypass status check
- Status update using `moveWorkItem()` from `move.go` - **works for both workflows**
- All `status_action` values:
  - `none`: Skip status update
  - `commit_only`: Commit on trunk before worktree creation - **for standalone AND polyrepo**
  - `commit_and_push`: Commit and push on trunk before worktree creation - **for standalone AND polyrepo**
  - `commit_only_branch`: Commit on new branch after worktree creation - **for standalone AND polyrepo**
- Status commit message templates (with variable substitution)
- Integration with existing `moveWorkItem()` function
- Error handling for status operations

**Success Criteria:**
- [ ] Status check executes after git pull (step 5) if `status_action != none`
- [ ] Status check skipped entirely if `status_action == none`
- [ ] Work item already in target status: Fails unless `--skip-status-check`
- [ ] `--skip-status-check` allows restarting work (skips status check and update)
- [ ] `moveWorkItem(cfg, id, move_to, false)` called to move file (no auto-commit)
- [ ] `status_action: none` skips all status operations
- [ ] `status_action: commit_only` stages and commits on trunk before worktree creation
- [ ] `status_action: commit_and_push` stages, commits, and pushes on trunk before worktree creation
- [ ] `status_action: commit_only_branch` stages and commits on new branch after worktree creation
- [ ] Status commit message template supports variables: {type}, {id}, {title}, {move_to}
- [ ] Default commit message used if template not configured
- [ ] Uncommitted changes detected before status commit on trunk
- [ ] Status commit failure aborts worktree creation (for commit_only/commit_and_push)
- [ ] Status push failure aborts worktree creation (for commit_and_push)
- [ ] For `commit_only_branch`: Commit happens in worktree context on new branch
- [ ] Uses `executeCommand()` for all git operations
- [ ] Integration tests cover all status_action scenarios
- [ ] **`make check` passes with no errors (linting, security, tests, coverage)**
- [ ] **No `#nosec` or other check-skipping comments added**
- [ ] **Code is commit-ready before moving to Phase 4**

**Configuration Required:**
```yaml
git:
  trunk_branch: main
  remote: origin

start:
  move_to: doing                                    # optional: defaults to "doing"
  status_action: commit_and_push                    # options: none | commit_only | commit_and_push | commit_only_branch
  status_commit_message: "Start work on {type} {id}: {title}"  # optional: template
```

**Deliverables:**
- Updated `internal/commands/start.go` with status management
- Status check logic (step 5 in execution order)
- Status commit operations for all `status_action` values
- Commit message template engine (variable substitution)
- Updated `internal/commands/start_test.go` with status tests
- Integration tests for all status workflows
- Documentation for status management patterns

---

### Phase 4: Integration & Polish (Horizontal Layer)

**Goal:** Add IDE launching, setup commands, dry-run mode, and final polish.

**Scope:**
- IDE configuration (`ide.command` and `ide.args`)
- IDE launching with error handling
- `--no-ide` and `--ide <command>` flags
- Setup commands/scripts (`workspace.setup` and `project.setup`)
- `--dry-run` flag with comprehensive preview generation
- Flag overrides (`--trunk-branch`, `--status-action`)
- Enhanced error messages
- E2E tests
- Performance optimization
- Documentation and release notes

**Success Criteria:**
- [ ] `--no-ide` skips IDE opening silently (highest priority)
- [ ] `--ide <command>` overrides config (supports shell expansion)
- [ ] `ide.command` from config used if no flag (no auto-detection)
- [ ] IDE opens in worktree directory (standalone/monorepo)
- [ ] IDE opens at worktree root (polyrepo)
- [ ] IDE launch failure logged as warning (worktree creation succeeds)
- [ ] `workspace.setup` executes in main project worktree
- [ ] `project.setup` executes in project worktrees (polyrepo)
- [ ] Setup runs after IDE opening (allows background execution)
- [ ] Setup failure aborts command with clear error
- [ ] `--dry-run` validates all inputs (including status check)
- [ ] `--dry-run` shows comprehensive preview (worktrees, branches, status, git ops, setup, IDE)
- [ ] `--dry-run` exits 0 if validation passes, non-zero if fails
- [ ] `--dry-run` executes no git operations, no file moves, no IDE, no setup
- [ ] `--trunk-branch` flag overrides config
- [ ] `--status-action` flag overrides config
- [ ] All error messages follow format standard
- [ ] E2E tests cover complete workflows
- [ ] Performance < 5 seconds for standalone
- [ ] Documentation complete
- [ ] Release notes written
- [ ] **`make check` passes with no errors (linting, security, tests, coverage)**
- [ ] **No `#nosec` or other check-skipping comments added**
- [ ] **Code is ready for public release (v0.1.0)**

**Configuration Required:**
```yaml
git:
  trunk_branch: main
  remote: origin

start:
  move_to: doing
  status_action: commit_and_push
  status_commit_message: "Start work on {type} {id}: {title}"

ide:
  command: "cursor"
  args: ["--new-window"]

workspace:
  setup: docker compose up -d  # main project setup
  projects:
    - name: frontend
      setup: npm install
    - name: backend
      setup: ./scripts/setup.sh
```

**Deliverables:**
- Updated `internal/commands/start.go` with IDE and setup support
- IDE launching logic (with all flags)
- Setup command execution logic
- Dry-run mode with preview generation
- Flag override handling
- E2E tests in `kira_e2e_tests.sh`
- Complete documentation
- Release notes

---

### Testing Strategy Across Phases

**Phase 0: Infrastructure**
- Unit tests for `executeCommand()` function
- Test both dryRun=true and dryRun=false paths
- Integration tests verifying refactored commands work identically
- Regression tests ensuring no behavior changes

**Phase 1: Input Validation**
- Unit tests for all validation functions (90%+ coverage)
- Edge case tests (empty strings, Unicode, special characters, path traversal)
- Configuration parsing and default application tests
- Error message format tests
- NO integration tests yet (no git operations)

**Phase 2: Git Operations**
- Mock-based unit tests for git operation functions
- Integration tests with real git repositories (temporary test repos)
- Error scenario tests (conflicts, network errors, diverged branches)
- `--override` and `--reuse-branch` flag tests
- Remote name resolution tests
- **Standalone AND polyrepo:** Workspace behavior inference tests
- **Polyrepo:** Transaction rollback tests (simulate failures at different stages)
- **Polyrepo:** Pre-validation tests (all-or-nothing checking)
- **Polyrepo:** Per-project override tests

**Phase 3: Status Management**
- Unit tests for status check logic
- Integration tests for all `status_action` values (standalone AND polyrepo)
- Tests for `moveWorkItem()` integration
- Commit message template tests
- `--skip-status-check` flag tests

**Phase 4: Integration & Polish**
- IDE integration tests (mock IDE execution)
- Setup command execution tests
- Dry-run mode tests (comprehensive preview validation)
- E2E tests covering complete workflows
- Performance benchmarks

---

### Dependencies Between Phases

**Strict Linear Dependencies (Horizontal Layering):**

```
Phase 0 (Infrastructure)
    ↓
Phase 1 (Input Validation) ← Uses executeCommand() for any git checks
    ↓
Phase 2 (Git Operations) ← Uses validated inputs, uses executeCommand()
    ↓                    ← Handles standalone AND polyrepo
Phase 3 (Status Management) ← Uses validated inputs + git operations
    ↓                       ← Works for standalone AND polyrepo
Phase 4 (Integration) ← Adds IDE, setup, dry-run on top of complete foundation
```

**Why This Order:**
- **Phase 0 first:** Infrastructure must exist before anything else
- **Phase 1 next:** All subsequent phases rely on validated, sanitized inputs (standalone AND polyrepo)
- **Phase 2 builds on Phase 1:** Git operations need validated inputs to be safe. Handles BOTH standalone AND polyrepo in same layer
- **Phase 3 builds on Phase 1+2:** Status management needs validation + git operations. Works for BOTH standalone AND polyrepo
- **Phase 4 is final:** Integration features (IDE, setup, dry-run) go on top of complete foundation

**Cannot Skip or Reorder:** Each phase builds on the previous one's foundation.

---

### Rollout Plan

**Horizontal Implementation = Complete Foundation First**

Each phase builds a complete architectural layer before moving to the next. No user releases until the entire feature is complete and polished.

**Implementation Order:**

1. **Phase 0: Infrastructure**
   - Refactor existing commands to use `executeCommand()`
   - Add dry-run support to `move` and `save`
   - Internal testing and validation

2. **Phase 1: Input Validation**
   - All input validation and sanitization (standalone AND polyrepo)
   - Workspace behavior inference
   - Worktree collision detection
   - Path safety checks
   - Internal testing (no git operations yet)

3. **Phase 2: Git Operations**
   - All git operations (trunk, pull, worktree, branch)
   - **Standalone AND polyrepo** workflows
   - Transaction rollback for polyrepo
   - `--override` and `--reuse-branch` support
   - Internal testing with real git repositories (standalone AND polyrepo)
   - **Internal milestone: Git operations complete for all workflow types**

4. **Phase 3: Status Management**
   - Status check and update logic
   - All `status_action` values (standalone AND polyrepo)
   - Internal testing of complete workflows
   - **Internal milestone: Complete workflow functional (standalone AND polyrepo)**

5. **Phase 4: Integration & Polish**
   - IDE launching
   - Setup commands
   - Dry-run mode
   - E2E tests
   - Documentation

**Release Strategy:**
- No public releases until Phase 5 complete
- Internal testing after each phase
- First public release includes complete feature set
- Well-tested, documented, and polished

## Implementation Notes

### Concurrent Execution Safety

**Concurrent Execution Handling:**

The `kira start` command is designed to support concurrent execution for different work items, which is essential for agentic workflows where multiple tasks are worked on simultaneously.

- **Partial protection via work item state change:** Concurrent execution is partially handled by the change of state for the work item file. When `kira start` moves a work item to a different status folder (e.g., from "backlog" to "doing"), this file system operation provides some natural protection against concurrent modifications of the same work item.

- **No explicit file locking:** The `kira start` command does not implement explicit file locking mechanisms or additional race condition protections beyond the natural file system operations.

- **Concurrent execution for different work items:** Multiple `kira start` commands can run simultaneously for different work items without conflicts, as each work item operates on its own file and creates its own isolated worktree.

- **Same work item concurrent execution:** If two `kira start` commands run simultaneously for the same work item, the behavior depends on timing:
  - If both commands attempt to move the work item file at the same time, one will succeed and the other may fail with a "file not found" error or may detect the work item is already in the target status.
  - The status check (step 5) provides some protection: if one command successfully moves the work item, the second command will detect it's already in the target status and fail (unless `--skip-status-check` is provided).
  - Worktree creation conflicts are handled by the existing worktree path validation logic (see "Error Handling" section).

- **No additional safety measures:** Further measures of safety (such as file locking, mutexes, or transaction-like operations) are not a concern for the `kira start` command at this point. The command relies on:
  1. Natural file system operations (file moves are atomic on most filesystems)
  2. Git worktree validation (prevents duplicate worktree creation)
  3. Status check validation (prevents duplicate status updates)
  4. Branch existence checks (prevents duplicate branch creation)

- **Recommendation for users:** For maximum safety when running concurrent commands, users should ensure different work items are used for each concurrent `kira start` execution, or use appropriate coordination mechanisms at the workflow level if concurrent execution of the same work item is required.

### Code Quality Requirements

**Every phase must meet these requirements before moving to the next:**

- **`make check` passes:** All linting, security checks, unit tests, and code coverage requirements
- **No check-skipping:** Absolutely no `#nosec`, `.golangci.yml` modifications, or other check-skipping mechanisms
- **Security compliance:** Follow `docs/security/golang-secure-coding.md` for all file operations and command execution
- **Test coverage:** Unit tests for all new functions, integration tests for complete workflows
- **Documentation:** Update relevant docs (README, security docs, etc.) in the same phase

**Why this matters:**
- Horizontal slicing means each layer is foundation for the next
- Buggy validation layer = all subsequent layers are unsafe
- Phase boundaries = commit boundaries = must be production-ready
- No "we'll fix the tests later" - tests are part of the phase

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
   - **Validation:** Perform all validation steps as normal - these don't modify state:
     - Step 1: Validate work item exists
     - Step 2: Infer workspace behavior
     - Step 3: Determine trunk branch
     - Step 4: Pull latest changes (skip if `--dry-run` - this is a git operation, but we need to check for uncommitted changes)
     - Step 5: **Status check** (if `status_action` is not `none`):
       - Check current work item status
       - If work item is already in target status and `--skip-status-check` is NOT provided:
         - Exit with non-zero code and show error: "Error: Work item {id} is already in '{move_to}' status. Work item status matches target status. Use --skip-status-check to restart work or review elsewhere."
         - Do NOT generate preview (validation failed)
       - If work item is already in target status and `--skip-status-check` IS provided:
         - Show in preview that status check would be skipped and status update would be skipped
       - If work item is not in target status:
         - Show in preview that status would be updated from current status to target status
   - **Preview generation:** After validation passes, generate structured preview without executing:
     - **Worktrees:** List all worktrees that would be created with their full paths (e.g., `../my-project_worktrees/123-fix-bug/`)
     - **Branches:** List all branches that would be created with their names (e.g., `123-fix-bug`)
     - **Status changes:** Show current status → target status (from `start.move_to` config, defaults to "doing"), or "no change" if `--skip-status-check` is provided or `status_action` is `none`
     - **Git operations:** List what git commands would be executed:
       - `git fetch <remote> <trunk_branch>` + `git merge <remote>/<trunk_branch>` (if remote exists)
       - `git worktree add <path> -b <branch> <trunk>` (worktree creation)
       - `git commit` (if `status_action` requires commit and status check passes)
       - `git push` (if `status_action` is `commit_and_push` and status check passes)
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
       - **Validate current branch is trunk:** After determining trunk branch, check if current branch (from `git rev-parse --abbrev-ref HEAD`) matches the trunk branch. If not, fail with error: "Error: Start command can only be run from trunk branch. Please checkout the trunk branch and try again." This validation occurs before any pull operations or worktree creation.
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
     - **Remote Name Resolution:** See "Remote Name Resolution" section for priority order
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
       - Determine remote name using priority order (see "Remote Name Resolution" section): Use `git.remote` if configured, otherwise default to "origin"
       - **Note:** Trunk branch checkout is not needed since validation ensures we're already on trunk branch (see trunk branch validation step above). The command must be run from the trunk branch.
       - **Pull latest changes:** Use `git fetch` + `git merge` for more control (rather than `git pull`):
         - Run `git fetch <remote-name> <trunk_branch>` to fetch latest changes (where remote-name is determined using priority order)
         - Run `git merge <remote-name>/<trunk_branch>` to merge remote changes into local trunk branch
     - **Polyrepo pull sequence:** For each project:
       - Determine remote name using priority order (see "Remote Name Resolution" section): Use `project.remote` if configured, then `git.remote` if configured, then "origin" as final default
       - Check if remote exists: Run `git remote get-url <project-remote-name>` in each project's repository directory
       - If remote doesn't exist: Skip pull for this project (log warning: "Warning: No remote '{project-remote-name}' configured for project '{project-name}'. Skipping pull step.")
       - If remote exists:
         - **Note:** Trunk branch checkout is not needed for main repository since validation ensures we're already on trunk branch (see trunk branch validation step above). The command must be run from the trunk branch.
         - **Pull latest changes:** Use `git fetch` + `git merge` for more control:
           - Run `git fetch <project-remote-name> <trunk-branch>` to fetch latest changes
           - Run `git merge <project-remote-name>/<trunk-branch>` to merge remote changes into local trunk branch
     - If merge fails with conflicts: abort with error: "Error: Failed to merge latest changes from {remote-name}/{trunk-branch}. Merge conflicts detected. Resolve conflicts manually and try again." Include git output in error message
     - If merge fails because branches have diverged: abort with error: "Error: Trunk branch has diverged from {remote-name}/{trunk-branch}. Local and remote branches have different commits. Rebase or merge manually before starting work."
     - If network error occurs: abort with error: "Error: Failed to fetch changes from {remote-name}. Network error occurred. Check network connection and try again." Include git output in error message
     - For polyrepo: Perform above steps sequentially for all project repositories; abort entire command if any project's pull fails (all-or-nothing approach)
     - This ensures everything is up-to-date before any operations
   - **Work Item Status Check:** (Executes as step 5 in Command Execution Order, after git pull and before worktree creation)
     - **Note:** This check only applies if `status_action` is not `none` (if `status_action` is `none`, skip this check entirely since no status update will occur)
     - After git pull completes successfully (step 4), check current work item status
     - This ensures we're checking the most up-to-date work item status after pulling latest changes
     - Get configured status from `start.move_to` (defaults to "doing")
     - **Check logic:**
       - If status matches configured value and `--skip-status-check` flag is not provided:
         - Return error: "Error: Work item {id} is already in '{move_to}' status. Work item status matches target status. Use --skip-status-check to restart work or review elsewhere." (where {move_to} is the configured value)
         - Abort command before worktree creation (step 7)
         - **Note:** This check applies regardless of when status update occurs (step 6 for `commit_only`/`commit_and_push`, or step 8 for `commit_only_branch`), because the status will be updated eventually
       - If status matches configured value and `--skip-status-check` flag is provided:
         - Skip status update step (step 6 for `commit_only`/`commit_and_push`, or step 8 for `commit_only_branch` - don't move file or update status field)
         - Proceed directly to worktree creation (step 7)
         - Allows resuming work or reviewing work item in different location
       - If status does not match configured `start.move_to`: Continue with normal status management flow (status update will occur at step 6 for `commit_only`/`commit_and_push`, or step 8 for `commit_only_branch`)
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
     - **Standalone/Monorepo:** Check if target worktree path already exists
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
     - **Polyrepo:** Worktree path validation happens in pre-validation phase (before Phase 1 worktree creation) - see "Workspace Handling" section for details
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
         - **Branch creation logic:**
           - If `--reuse-branch` flag is provided AND branch exists in this repository (from pre-validation):
             - **Checkout existing branch:** Run `git checkout {branch-name}` (executed in worktree directory) - branch already exists, just checkout
           - If `--reuse-branch` flag is NOT provided OR branch doesn't exist:
             - **Create and checkout branch:** Run `git checkout -b <branch-name>` (executed in worktree directory)
         - If branch creation/checkout fails: Attempt to rollback all worktrees using `rollbackWorktrees()` helper function. If rollback succeeds, abort command with error indicating all worktrees were rolled back. If rollback fails, abort command with error indicating rollback failed and some worktrees may remain.
   - **Status Commit Operations:**
     - **For `commit_only` or `commit_and_push`:**
       - Stage work item file: `git add <work-item-path>` (file is at new location after `moveWorkItem` call)
       - Commit on trunk branch: `git commit -m "<commit-message>"` (where commit message is from template or default)
       - If `commit_and_push`: Push to remote: `git push <remote-name> <trunk-branch>` (where remote-name is determined using priority order - see "Remote Name Resolution" section)
       - **Execution context:** Commands run in main repository directory (trunk branch)
     - **For `commit_only_branch`:**
       - Stage work item file: `git add <work-item-path>` (file is at new location after `moveWorkItem` call)
       - Commit on new branch: `git commit -m "<commit-message>"` (where commit message is from template or default)
       - **Execution context:** Commands run in worktree directory (new branch)
       - **Note:** The work item file is in the original repository's `.work/` directory, but the commit happens in the worktree context

3. **Work Item Parsing**
   - Reuse `extractWorkItemMetadata` function from `move.go`
   - **Work item ID validation:** Validate that work item ID contains only valid characters and matches expected format (see "Title Sanitization" section for details)
   - Extract title and sanitize for directory/branch names (see "Title Sanitization" section below for detailed sanitization rules)
   - **Handle missing title:** If title is empty or "unknown":
     - Use just the work item ID as fallback
     - Log warning: "Warning: Work item {id} has no title field. Using work item ID '{id}' for worktree directory and branch name."
     - This ensures worktree directory and branch names are always valid: `{work-item-id}` when title is missing
   - **Empty sanitized title validation:** If title sanitization results in empty string, fail with error (see "Title Sanitization" section for details)
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
        Remote       string `yaml:"remote"`      // default: "" (see "Remote Name Resolution" section for priority order)
        TrunkBranch  string `yaml:"trunk_branch"` // optional: per-project trunk branch override (defaults to workspace.git.trunk_branch or auto-detected)
        Setup        string `yaml:"setup"`       // optional: command or script path for project-specific setup
    }
     ```
   - Default values should be applied during config loading:
     - If `git` is nil: use defaults (trunk_branch: "" for auto-detect)
     - If `git.trunk_branch` is omitted or empty: auto-detect (check "main" first, then "master")
     - If `start` is nil: use defaults (move_to: "doing", status_action: "commit_and_push")
     - If `start.move_to` is omitted or empty: default to "doing"
     - **Validation:** After applying defaults, validate that `start.move_to` (or default "doing") is a valid status key in `cfg.StatusFolders`. If invalid, fail with error at config load time: `"Error: Invalid status '{invalid_status}'. Status must be one of the following: {valid_statuses}. Please check your configuration and try again."` (where `{valid_statuses}` is a comma-separated list of all keys in `cfg.StatusFolders`, sorted alphabetically)
     - If `start.status_action` is omitted: default to "commit_and_push"
     - Validate `start.status_action` is one of: "none", "commit_only", "commit_and_push", "commit_only_branch"
     - If `start.status_commit_message` is empty: use default template "Move {type} {id} to {move_to}" (uses configured `move_to` value, or "doing" if not configured)
     - If `git.remote` is omitted or empty: default to "origin"
     - For polyrepo projects: If `project.remote` is omitted or empty: use priority order (see "Remote Name Resolution" section) - `git.remote` if configured, otherwise "origin"
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
   - **Kebab-case conversion algorithm:** Use the same algorithm as `kira new` command (defined in `internal/commands/new.go`):
     1. Convert entire string to lowercase using `strings.ToLower()`
     2. Replace all spaces (`" "`) with hyphens (`"-"`) using `strings.ReplaceAll()`
     3. Replace all underscores (`"_"`) with hyphens (`"-"`) using `strings.ReplaceAll()`
     - **Unicode handling:** Unicode characters are handled the same way as in `kira new`: converted to lowercase as-is (no special Unicode normalization). Unicode characters that are valid in git branch names and directory names are preserved.
   - **Work item ID validation:** Before sanitization, validate that the work item ID contains only valid characters:
     - Work item IDs must match the format specified in `cfg.Validation.IDFormat` (default: `"^\\d{3}$"` - three digits)
     - If work item ID contains invalid characters (e.g., path traversal characters like `../`, or characters that don't match the ID format), fail with error: `"Error: Invalid work item ID '{id}'. Work item ID contains invalid characters or doesn't match expected format. Work item IDs must be valid identifiers."`
     - This validation should occur before any sanitization or worktree creation
   - **Handle missing title:** If title is empty or "unknown" (from `extractWorkItemMetadata`):
     - Use just the work item ID as fallback (e.g., `"123"` → worktree directory `"123"`, branch `"123"`)
     - Log warning: "Warning: Work item {id} has no title field. Using work item ID '{id}' for worktree directory and branch name."
   - **Empty sanitized title validation:** After kebab-case conversion, if the sanitized title results in an empty string (e.g., title contained only special characters that were removed), fail with error: `"Error: Work item '{id}' title sanitization resulted in empty string. Title cannot be sanitized to a valid name. Please update the work item title to include valid characters."`
   - **Branch name length limit:**
     - Git allows branch names up to 255 characters, but very long names are unwieldy and hard to work with
     - **Recommended approach:** Truncate sanitized title to 100 characters
     - **Uniqueness handling:** If truncation occurred (original sanitized title > 100 chars), append `-{short-hash}` where `short-hash` is the first 6 characters of SHA256 hash of the full sanitized title
     - This ensures:
       - Branch names remain readable and manageable (100 chars is sufficient for most descriptive titles)
       - Uniqueness is preserved even when different long titles truncate to the same 100 characters
       - No need to check for existing branch collisions (hash ensures uniqueness)
   - **Final validation:** After all sanitization steps, ensure the final worktree directory name and branch name are valid:
     - Must not be empty (handled above)
     - Must not contain path traversal characters (`../`, `..\\`, etc.)
     - Must be valid for git branch names (no leading/trailing dots, no consecutive dots, no `.lock` suffix, etc.)

6. **IDE Detection**
   - **Configuration priority:** Check flags in order: `--no-ide` (highest), then `--ide <command>`, then `ide.command` from `kira.yml`
   - **`--no-ide` flag:** If `--no-ide` flag is provided:
     - Skip IDE opening entirely (useful for agents or CI/CD environments)
     - Ignore `--ide` flag and `ide.command` config (flag takes precedence)
     - No log messages about IDE (silently skip)
   - **Flag override:** If `--ide <command>` flag is provided (and `--no-ide` not set):
     - Use flag value as IDE command (overrides `ide.command` from config)
     - Ignore `ide.args` from config (flag value takes precedence)
     - **Shell expansion support:** Users can pass arguments via shell expansion (e.g., `--ide "cursor --new-window"`). The flag value can include both the command and its arguments, and will be executed as-is by the shell.
     - Execute: `{flag-command} {worktree-path}` (where `{flag-command}` may include arguments if provided via shell expansion)
   - **Config-based:** If neither flag provided, use `ide.command` from `kira.yml` - no auto-detection
   - If no IDE command found (flag or config): Skip IDE opening, log info message "Info: No IDE configured. Worktree created at {path}. Configure `ide.command` in kira.yml or use `--ide <command>` flag to automatically open IDE.", continue
   - **No hardcoded logic:** All IDE behavior comes from flags (`--no-ide`, `--ide`) or `ide.command`/`ide.args` configuration
   - **IDE Launch:**
     - Check if `--no-ide` flag is provided
     - If `--no-ide` provided: Skip IDE opening (no execution, no log messages)
     - If `--no-ide` not provided: Check if `--ide <command>` flag is provided
     - If `--ide` flag provided: Execute `{flag-command} {worktree-path}` (where `{flag-command}` may include arguments if provided via shell expansion)
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
     - **Main project definition:** The main project is the repository where `kira.yml` is located
     - **Note:** The main project is never listed in `workspace.projects` (it's the repository containing `kira.yml`), but it always gets a worktree created
     - Execute setup command/script in main project worktree directory
     - For standalone/monorepo: Run in `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}`
     - For polyrepo: Run in main project worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/main/` (or `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{main-project-name}/`)
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
       - **Worktree path existence check:** Check all worktree paths before creating any (all-or-nothing approach):
         - For each worktree that will be created (main project and all projects in `workspace.projects`):
           - Determine target worktree path (main project: `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/main/`, projects: based on `repo_root` grouping or `project.mount`)
           - Check if target worktree path already exists
           - If path exists: Determine if it's a valid git worktree and check if it's for the same work item (by checking branch name or work item ID in path)
           - Track path status per worktree: `pathStatus map[string]PathStatus` where key is worktree path and value indicates: `not_exists`, `valid_worktree_same_item`, `valid_worktree_different_item`, or `invalid_worktree`
         - **All-or-nothing path validation:** After checking all paths:
           - If ANY path exists (regardless of type) and `--override` flag is NOT provided:
             - If path is valid git worktree for same work item: Abort with error listing all affected paths
             - If path is valid git worktree for different work item: Abort with error listing all affected paths
             - If path is not a valid git worktree: Abort with error listing all affected paths
           - If ANY path exists and `--override` flag IS provided: Remove all existing paths before creating worktrees
             - **All-or-nothing removal:** `--override` applies to all conflicting worktree paths for the work item (main project and all projects in `workspace.projects`). All paths must be successfully removed before proceeding.
             - **Removal method:** For each conflicting path:
               - If path is a valid git worktree: Remove using `git worktree remove <path>` (or `git worktree remove --force` if worktree has uncommitted changes)
               - If path is not a valid git worktree (invalid worktree or non-worktree directory): Remove using `os.RemoveAll`
             - **Failure handling:** If removal fails for any path (e.g., worktree is locked, permission denied, file system error), abort with error: `"Error: Failed to remove existing worktree at {path}. Cannot proceed with --override. {error-details}. Resolve the issue and try again."` (where `{error-details}` includes the specific error from the removal operation). Do not continue with worktree creation if any removal fails.
             - **Success:** Only after all conflicting paths are successfully removed, proceed with worktree creation (Phase 1)
           - If NO paths exist: Proceed with worktree creation
       - **Branch existence check:** Check branch `{work-item-id}-{kebab-case-work-item-title}` existence in each repository independently (main project and all projects in `workspace.projects`):
         - For each repository: Check if branch exists using `git show-ref --verify --quiet refs/heads/{branch-name}` in that repository
         - If branch exists: Get branch commit hash and trunk branch commit hash, compare to determine if branch points to trunk or has commits
         - Track branch status per repository: `branchStatus map[string]BranchStatus`
       - **All-or-nothing branch validation:** After checking all repositories:
         - If ANY branch has commits: Abort with error listing all repositories where branch has commits (all-or-nothing)
         - If ALL branches point to trunk (or don't exist) and `--reuse-branch` is NOT provided: Abort with error listing all repositories where branch exists
         - If ALL branches point to trunk (or don't exist) and `--reuse-branch` IS provided: Continue with worktree creation, will checkout existing branches in Phase 2
         - If SOME branches exist and SOME don't: Treat consistently (apply `--reuse-branch` logic across all repositories)
     - **Phase 1 - Worktree Creation:** Create all worktrees first (makes rollback easier)
       - Initialize empty list to track successfully created worktrees: `createdWorktrees []string`
       - **Main project worktree:** Always create worktree for main project (repository where `kira.yml` is located) first
         - Create worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/main/` (or use project name if derivable)
         - If creation succeeds: Add worktree path to `createdWorktrees` list
         - If creation fails: Attempt to rollback all worktrees in `createdWorktrees` using `rollbackWorktrees()` helper function. If rollback succeeds, abort command with error indicating all worktrees were rolled back. If rollback fails, abort command with error indicating rollback failed and some worktrees may remain.
       - **Group projects by `repo_root`:**
         - Group projects that share the same `repo_root` value
         - **Purpose:** Groups projects sharing the same root directory (monorepo case) or handles nested folder structures
         - For each unique `repo_root`:
           - Create ONE worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{sanitized-repo-root}/`
             - `sanitized-repo-root` is created by: extracting directory name from `repo_root` using `filepath.Base(repo_root)`, then applying kebab-case sanitization (lowercase, replace spaces/underscores with hyphens) - see "Path Sanitization" section in "Worktree Location" for details
           - If creation succeeds: Add worktree path to `createdWorktrees` list
           - If creation fails: Attempt to rollback all worktrees in `createdWorktrees` using `rollbackWorktrees()` helper function. If rollback succeeds, abort command with error indicating all worktrees were rolled back. If rollback fails, abort command with error indicating rollback failed and some worktrees may remain.
           - Track which projects are in each group to skip duplicate worktree creation
       - **Standalone projects** (no `repo_root`):
         - For each project without `repo_root`:
           - Create separate worktree at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}/{project.mount}/`
           - If creation succeeds: Add worktree path to `createdWorktrees` list
           - If creation fails: Attempt to rollback all worktrees in `createdWorktrees` using `rollbackWorktrees()` helper function. If rollback succeeds, abort command with error indicating all worktrees were rolled back. If rollback fails, abort command with error indicating rollback failed and some worktrees may remain.
     - **Phase 2 - Branch Creation:** After all worktrees created successfully:
       - For each worktree in `createdWorktrees` (including main project worktree):
         - **Change to worktree directory:** Change directory to worktree path (use `os.Chdir()` or `exec.Command` with `Dir` field set to worktree path)
         - **Branch creation logic:**
           - If `--reuse-branch` flag is provided AND branch exists in this repository (from pre-validation `branchStatus` map):
             - **Checkout existing branch:** Run `git checkout {work-item-id}-{kebab-case-work-item-title}` (executed in worktree directory) - branch already exists, just checkout
           - If `--reuse-branch` flag is NOT provided OR branch doesn't exist:
             - **Create and checkout branch:** Run `git checkout -b {work-item-id}-{kebab-case-work-item-title}` (executed in worktree directory)
         - If branch creation/checkout fails: Attempt to rollback all worktrees in `createdWorktrees` using `rollbackWorktrees()` helper function. If rollback succeeds, abort command with error indicating all worktrees were rolled back. If rollback fails, abort command with error indicating rollback failed and some worktrees may remain.
     - Use consistent naming: `{work-item-id}-{kebab-case-work-item-title}` for both worktree directories and branches
     - IDE opens at `{worktree_root}/{work-item-id}-{kebab-case-work-item-title}` (worktree root)
   - **Rollback Helper Function:**
     - Create helper function `rollbackWorktrees(worktrees []string) error` that:
       - Iterates through list of worktree paths in reverse order
       - For each worktree: Run `git worktree remove <path>` (or `git worktree remove --force` if worktree has uncommitted changes)
       - Log each removal attempt
       - **Failure handling:** If any removal fails (e.g., worktree is locked, permission denied, file system error), abort rollback immediately and return error: `"Error: Failed to remove worktree at {path} during rollback. Rollback aborted. {error-details}. Resolve the issue and try again."` (where `{error-details}` includes the specific error from the removal operation). Do not continue attempting to remove other worktrees if any removal fails.
       - **Success:** Only return `nil` if all worktrees are successfully removed
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

