---
id: 038
title: kira config get
status: doing
kind: prd
assigned:
created: 2026-02-02
tags: [configuration, cli]
---

# kira config get

A command that gets a value from the kira configuration. It enables scripts, agents, and automation to read effective config values without parsing `kira.yml` or invoking other commands.

## Context

Kira commands (e.g. `kira start`, `kira open`, `kira slice`) rely on configuration from `kira.yml` and runtime resolution (e.g. trunk branch auto-detect, worktree root). Today there is no supported way for scripts or agents to obtain a single config value (e.g. trunk branch, work folder path) except by parsing the file or reimplementing resolution logic.

A dedicated `kira config get <key>` command provides:
- A stable, documented way to read config values for scripting and automation
- Consistent resolution behavior (same as other commands: defaults, auto-detect where applicable)
- Single-value output suitable for shell substitution: `branch=$(kira config get trunk_branch)`

**Dependencies**: Uses existing `config.LoadConfig()` and config helpers (e.g. `GetWorkFolderPath`, `GetDocsFolderPath`). For `trunk_branch` resolution, reuses the same logic as `kira start` (config → auto-detect main/master).

## Requirements

### Command interface

- **Command**: `kira config get <key> [--project <name>]`
- **Behavior**: Load config (from current directory: `kira.yml` or `.work/kira.yml`), resolve the requested key (optionally in the context of a project), and print the value to stdout. No key name or extra output—value only, so scripts can use `var=$(kira config get <key>)`.
- **Flags**:
  - `--output format`: `text` (default) or `json`; for list values (e.g. `ide.args`, `project_names`), `text` outputs one value per line.
  - `--project <name>`: Resolve the key for the named project (polyrepo only). Only valid for keys that have per-project meaning: `trunk_branch`, `remote`, `project_path`. Requires `workspace.projects` with a project matching `<name>`; otherwise exit non-zero with clear error. Special value `*` or `all`: resolve the key for **every** project and output a list (see “All projects” below).
- **Exit codes**: 0 on success; non-zero on unknown key, missing config (if we require kira workspace), resolution error (e.g. trunk_branch when not in a git repo), invalid `--project` (no projects / unknown project name), or key that does not support `--project`.

### Supported keys

Keys are lowercase, with optional underscores. Mapping to config and resolution:

| Key | Source / resolution | With `--project`? | Notes |
|-----|---------------------|-------------------|--------|
| `trunk_branch` | `git.trunk_branch` if set; else auto-detect main/master in current repo | Yes: project’s trunk (`project.trunk_branch` > `git.trunk_branch` > auto-detect in project path) | Requires git repo for auto-detect. |
| `remote` | `git.remote` | Yes: project’s remote (`project.remote` > `git.remote` > `origin`) | Default `origin`. |
| `project_path` | — | Yes only | Resolved absolute path to the project repository (same resolution as `kira start`). Without `--project`, error. |
| `project_names` | `workspace.projects[].name` | No (ignored) | List of project names; empty if standalone/monorepo. Use to iterate: `for p in $(kira config get project_names); do kira config get trunk_branch --project "$p"; done` |
| `work_folder` | `workspace.work_folder` | No | Default `.work`. |
| `work_folder_abs` | Resolved from `ConfigDir` + `work_folder` | No | Absolute path; fails if config dir unresolved. |
| `docs_folder` | `docs_folder` | No | Default `.docs`. |
| `docs_folder_abs` | Resolved from `ConfigDir` + `docs_folder` | No | Absolute path; same validation as `DocsRoot`. |
| `config_dir` | `Config.ConfigDir` | No | Absolute path to directory containing `kira.yml`. |
| `ide.command` | `ide.command` | No | Empty if not set. |
| `ide.args` | `ide.args` | No | List; output one per line (text) or JSON array (json). |

- **When `--project <name>` is used** (single project): Only `trunk_branch`, `remote`, and `project_path` are valid. For any other key, exit non-zero with message that the key does not support `--project`.
- **When `--project '*'` or `--project all` is used (all projects)**: Same keys (`trunk_branch`, `remote`, `project_path`) are valid. Output is one entry per project so the same key can be seen for every repo. See “All projects” below.
- **When `--project` is used but config has no `workspace.projects`** (standalone): Exit non-zero, e.g. “no projects defined; --project is for polyrepo only”. Exception: `--project '*'` / `--project all` with no projects returns empty output (or empty JSON object) and exit 0.
- **When `--project <name>` is used and `<name>` is not in `workspace.projects`** (and not `*`/`all`): Exit non-zero, e.g. “unknown project: '<name>'” and list valid project names.
- Unknown key (without or with `--project`) must exit with a clear error and non-zero exit code.

**All projects** (`--project '*'` or `--project all`): When used with a project-scoped key, resolve that key for every project and output the list. Format: **text** — one line per project, `name: value` (e.g. `frontend: main`, `api: main`); **json** — single object `{"projectName": "value", ...}` (e.g. `{"frontend":"main","api":"main"}`). Order follows `workspace.projects`. If there are no projects, output is empty (text: no lines; json: `{}`) and exit 0.

### Path syntax (dot-path into config)

When the key contains a dot (`.`), it is treated as a **path** into the merged config (the in-memory config after `LoadConfig()` and defaults merge). Paths return **raw** values from config—no extra resolution (e.g. no auto-detect for trunk; use curated key `trunk_branch` for resolved value).

- **Format**: Dot-separated segments. Each segment is a YAML key name in the merged config (snake_case as in kira.yml). Examples: `git.trunk_branch`, `workspace.work_folder`, `ide.command`, `docs_folder`, `validation.required_fields`.
- **Arrays**: Use a numeric index for list elements. Examples: `workspace.projects.0.name`, `workspace.projects.0.remote`, `workspace.projects.1.trunk_branch`. Order is the same as in `workspace.projects`.
- **Lookup**: Walk the merged config by path. Missing segment or invalid index → exit non-zero with a clear error (e.g. “path not found: git.foo” or “invalid index in workspace.projects.99”).
- **Output**: Leaf scalar (string, number, bool) → value only, one line. List → one value per line (text) or JSON array (json). Nested object at path → output as JSON. Empty or nil value → empty string (or appropriate empty for type).
- **Interaction with `--project`**: Path syntax is workspace-level only; `--project` is ignored for path keys. Paths do not support per-project resolution; use curated keys with `--project` for that.
- **Curated vs path**: If the key is an exact match for a curated key (e.g. `trunk_branch`, `work_folder`), the curated getter is used (resolved value). If the key contains a dot, path lookup is used. So `trunk_branch` = resolved trunk; `git.trunk_branch` = raw value from config (may be empty). Path segments use YAML key names; runtime-only fields (e.g. `ConfigDir`, not in kira.yml) are not required to be addressable by path—use curated key `config_dir` for that.

### Output format

- **Scalar values**: Single line, value only, **with newline at end** (so `$(kira config get key)` trims one line and is safe).
- **List values** (e.g. `ide.args`): With `--output text`, one value per line; with `--output json`, a JSON array.
- **Stderr**: Errors and hints on stderr; stdout is value-only on success.

### Error handling

- Unknown or unsupported key: clear message (e.g. `unknown key: 'foo'`), list valid keys in help; exit non-zero.
- Not in a kira workspace: **Do not require** a work directory. Config is loaded from the current directory; if no `kira.yml` or `.work/kira.yml` exists, use defaults and return values where possible (e.g. `remote` → `origin`). Only fail when a key genuinely cannot be resolved (e.g. `trunk_branch` auto-detect when not in a git repo).
- `trunk_branch`: when auto-detect is needed, if not in a git repo or branch cannot be determined, exit non-zero with clear message.
- Path resolution errors (`work_folder_abs`, `docs_folder_abs`, `config_dir`): exit non-zero with message.

### Configuration

- Uses `config.LoadConfig()` (current directory). No `--config` flag in initial scope.
- Respects same lookup order as rest of kira: root `kira.yml` then `.work/kira.yml`; merged with defaults.

### Polyrepo and per-project config

- **Where config is loaded**: Config is loaded from the current directory (where `kira.yml` or `.work/kira.yml` is found). In polyrepo setups, kira is typically run from the **main** repo root (the repo that contains `kira.yml` and `workspace.projects`). Running `kira config get` from a project subdirectory may not find config unless that directory has its own kira.yml.
- **Workspace-level keys** (no `--project`): `work_folder`, `work_folder_abs`, `docs_folder`, `docs_folder_abs`, `config_dir`, `ide.command`, `ide.args` always return main/workspace values. `trunk_branch` and `remote` without `--project` return the main repo’s values.
- **Listing projects**: Use `kira config get project_names` to get the list of `workspace.projects[].name` (one per line or JSON). In standalone/monorepo this returns an empty list.
- **Per-project keys**: Use `kira config get <key> --project <name>` to get values for a specific repo in `workspace.projects`:
  - `trunk_branch` — `project.trunk_branch` > `git.trunk_branch` > auto-detect in that project’s path (same priority as `kira start`).
  - `remote` — `project.remote` > `git.remote` > `origin`.
  - `project_path` — resolved absolute path to the project repository (only valid with `--project`).
- **Validation**: If `--project` is used but there are no projects, or the project name is unknown, the command exits non-zero with a clear error and (for unknown name) lists valid project names.
- **All projects**: Use `kira config get <key> --project '*'` (or `--project all`) to get the same key for every project in one call. Output: text = one line per project `name: value`; json = object `{projectName: value}`. Example: `kira config get trunk_branch --project '*'` → `frontend: main`, `api: main` (or JSON). No projects → empty output, exit 0.

### Out of scope

- **“Check commands”** for the project or main repo: not present in current config.
- **`kira config set`** or any other config mutation.

## Acceptance criteria

- [ ] `kira config get <key>` exists and is registered under `kira config`.
- [ ] Supported keys return the correct effective value (including defaults and, for `trunk_branch`, auto-detect when applicable).
- [ ] Scalar output is value-only on stdout, one line, with newline; list keys with `--output text` print one value per line; `--output json` prints JSON for list keys.
- [ ] Unknown key prints a clear error to stderr and exits non-zero; help or error lists valid keys.
- [ ] When `trunk_branch` requires auto-detect and the current directory is not a git repo (or trunk cannot be determined), command exits non-zero with a clear message.
- [ ] `work_folder_abs`, `docs_folder_abs`, and `config_dir` return absolute paths consistent with `GetWorkFolderAbsPath`, `DocsRoot(cfg, ConfigDir)`, and `ConfigDir`.
- [ ] `project_names` returns the list of `workspace.projects[].name` (one per line or JSON); empty when no projects.
- [ ] `--project <name>`: for `trunk_branch`, `remote`, `project_path` returns the effective value for that project; same resolution order as `kira start`. Invalid or unknown project name exits non-zero with clear error; using `--project` with a key that does not support it exits non-zero.
- [ ] `--project '*'` / `--project all`: for project-scoped keys, outputs one entry per project (text: `name: value` per line; json: `{"name":"value",...}`). No projects → empty output, exit 0.
- [ ] Unit tests cover at least: each supported key (with and without config file), unknown key, trunk_branch with/without repo, project_names (standalone vs polyrepo), --project (valid name, unknown name, no projects, unsupported key), and --project '*' (all projects, empty projects).
- [ ] **Path syntax**: When key contains a dot, resolve as path into merged config (raw). Path segments = YAML key names; arrays = numeric index (e.g. `workspace.projects.0.name`). Scalar path → value; list path → one per line (text) or JSON array; object path → JSON. Missing path or invalid index exits non-zero. Curated keys (exact match, no path) take precedence; path lookup is used only when key contains a dot. `--project` is ignored for path keys.
- [ ] `make check` passes; e2e or smoke test optional for “config get” in a real repo.

## Slices

### Slice 1: Command scaffold and key dispatch

- [ ] T001: Add `config` subcommand and `config get <key>` in `internal/commands`; wire in `root.go`.
- [ ] T002: Implement key parsing and dispatch (switch or map) to placeholder getters; unknown key returns error and non-zero exit.
- [ ] T003: Add tests for unknown key and for help listing valid keys.

### Slice 2: Scalar keys (remote, work_folder, docs_folder, config_dir)

- [ ] T004: Implement getters for `remote`, `work_folder`, `docs_folder`, `config_dir` using existing config helpers.
- [ ] T005: Implement `work_folder_abs` and `docs_folder_abs` using `GetWorkFolderAbsPath` and `DocsRoot(cfg, cfg.ConfigDir)`; handle errors.
- [ ] T006: Add unit tests for these keys (with default config, with custom kira.yml).

### Slice 3: trunk_branch resolution

- [ ] T007: Implement `trunk_branch` using same logic as `kira start` (config then auto-detect); reuse or refactor `determineTrunkBranch` from start.go so config get stays in sync.
- [ ] T008: Add tests: configured trunk_branch, empty (auto-detect main/master), not a git repo, and invalid/missing branch.

### Slice 4: ide.command and ide.args output formats

- [ ] T009: Implement `ide.command` and `ide.args`; for `ide.args`, support `--output text` (one per line) and `--output json` (array).
- [ ] T010: Add tests for scalar vs list output and for missing ide config.

### Slice 5: Polyrepo and per-project keys

- [ ] T011: Add `project_names` key: return list of `workspace.projects[].name` (one per line or JSON); empty when no projects.
- [ ] T012: Add `--project <name>` flag; accept `*` or `all` as “all projects”; validate single project name against `workspace.projects`; error when no projects or unknown name (except `*`/all with no projects → empty output, exit 0).
- [ ] T013: Implement per-project resolution for `trunk_branch`, `remote`, and `project_path` when `--project <name>` is set (reuse same logic as `kira start`: resolvePolyrepoProjects / resolveTrunkBranch per project path).
- [ ] T014: Implement “all projects” mode: when `--project '*'` or `--project all`, resolve the key for every project; output text as `name: value` per line, json as `{"name":"value",...}`. No projects → empty output.
- [ ] T015: Error when `--project` is used with a key that does not support it; error when `project_path` is requested without `--project`.
- [ ] T016: Add unit tests for project_names (standalone, polyrepo), --project (valid, unknown, no projects), --project '*' (all projects, empty), and per-project trunk_branch/remote/project_path.

### Slice 6: Path syntax (dot-path into config)

- [ ] T017: Implement path detection: if key contains `.`, treat as path; otherwise treat as curated key. Path parser: split on `.`; segments = YAML key names; last segment may be numeric for array index (e.g. `workspace.projects.0`).
- [ ] T018: Implement path lookup on merged config: walk config by path segments (map struct fields by yaml tag; support numeric index for slices). Return raw value at path. Missing segment or invalid index → error, exit non-zero.
- [ ] T019: Path output: scalar (string, number, bool) → value only; list → one per line (text) or JSON array (json); nested object → JSON. Ignore `--project` when key is a path.
- [ ] T020: Add unit tests for path: valid paths (git.trunk_branch, workspace.projects.0.name), missing path, invalid index, list/object output formats.

### Slice 7: Documentation and polish

- [ ] T021: Document supported keys, path syntax (dot-path, array index), `--project`, and `--project '*'` in help text and, if present, in `.docs/` or README.
- [ ] T022: Run `kira slice lint` and fix any slice/task issues; run `make check` and e2e if applicable.

## Implementation notes

- Reuse `config.LoadConfig()`; do not require a work directory—config get works from any directory where config is found (or defaults if no config file).
- Trunk branch: prefer refactoring `determineTrunkBranch` (or equivalent) into a shared package or `internal/commands` helper so both `start` and `config get` use the same logic.
- Per-project resolution: reuse the same resolution as `kira start`—e.g. `resolvePolyrepoProjects` (or equivalent) to get project list and paths, then per-project trunk/remote with project.TrunkBranch > git.TrunkBranch > auto-detect in project path, and project.Remote > git.remote > "origin". Keep `config get` in sync with start’s behavior.
- Keep key set extensible (e.g. map or registry) so adding keys later is a single place change.
- Path lookup: walk the merged config (after LoadConfig + defaults) by dot-separated path. Map path segments to struct fields via yaml tags; support numeric segment for array index (e.g. `workspace.projects.0`). Implementation can use reflection on config structs or serialize config to a nested map and look up by path.

## Release notes

- **Added** `kira config get <key>` to read effective configuration values (e.g. `trunk_branch`, `remote`, `work_folder`, `docs_folder`, `config_dir`, `ide.command`, `ide.args`) for use in scripts and automation.
- **Path syntax**: Keys containing a dot are treated as a path into the merged config (raw values). Examples: `kira config get git.trunk_branch`, `kira config get workspace.projects.0.name`. Use numeric index for arrays (e.g. `workspace.projects.0.remote`). Curated keys (no dot) return resolved values; path keys return raw config.
- **Polyrepo**: `kira config get project_names` returns the list of projects from `workspace.projects`. Use `kira config get <key> --project <name>` to get per-project values for `trunk_branch`, `remote`, and `project_path`. Use `kira config get <key> --project '*'` to get the same key for every project in one call (output: `name: value` per line or JSON object).

