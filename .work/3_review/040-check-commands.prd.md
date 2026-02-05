---
id: 040
title: check commands
status: review
kind: prd
assigned:
created: 2026-02-02
tags: [configuration, cli]
---

# check commands

A configurable list of commands that can be run to verify code quality, tests, and other checks. Enables a single `kira check` (or similar) entry point and supports agents/docs that say “run the project checks” without hardcoding `make check`.

**Example config:**

```yaml
checks:
  - name: lint
    command: make lint
    description: Run linter
  - name: security
    command: make security
    description: Run vulnerability scanner
  - name: test
    command: make test
    description: Run unit tests
```

## Context

- **Today**: AGENTS.md and docs tell users to run `make check`. That target is repo-specific (e.g. in this repo: lint, security, test). There is no kira-level notion of “the project’s check commands,” and no way to list or run them via kira.
- **Problem**: Scripts and agents need a stable way to “run the project’s checks” that works across repos. Hardcoding `make check` is brittle when repos use different targets or tooling.
- **Goal**: Allow repos to define a list of check commands in `kira.yml`. Provide a kira command to list and/or run those commands so agents and users have one entry point.
- **Relationship to 038**: PRD 038 (kira config get) explicitly leaves “check commands” out of scope because the config key does not exist yet. This PRD introduces the `checks` config and the CLI; 038 can later add support for reading it (e.g. `kira config get checks`) if desired.

## Requirements

### Config schema

- **Key**: `checks` (top-level in `kira.yml`).
- **Type**: List of objects. Each entry:
  - `name` (string, required): Short identifier (e.g. `lint`, `test`). Used in output and for optional filtering.
  - `command` (string, required): Shell command to run (e.g. `make lint`, `go test ./...`). Executed from the config directory (directory containing `kira.yml`).
  - `description` (string, optional): Human-readable description (e.g. “Run linter”).
- **Default**: If `checks` is missing or empty, the list is empty (no checks configured).
- **Validation**: Duplicate `name` values allowed but not required to be rejected in initial scope; optional lint rule later.

### Command interface

- **Primary**: `kira check` runs all configured checks in order.
  - Load config from current directory (`kira.yml` or `.work/kira.yml`).
  - For each entry in `checks`, run `command` in the config directory (same cwd semantics as other kira commands).
  - Stream output (stdout/stderr) from each check to the user.
  - If any check exits non-zero: stop running remaining checks, print which check failed, exit with non-zero (e.g. first failure exit code or fixed code like 1).
  - If no checks configured: exit 0 and print a short message (e.g. “no checks configured; add a `checks` section to kira.yml”).
- **List (optional subcommand or flag)**: `kira check list` (or `kira check --list`) prints configured checks (name and description, one per line or table). If none configured, print the same “no checks configured” message and exit 0.
- **Scope**: No `--project` in initial scope; checks are workspace-level only (run from main repo root where config lives).

### Exit codes and output

- **Success**: All checks pass → exit 0.
- **Failure**: A check exits non-zero → exit non-zero (e.g. 1); clear message indicating which check failed.
- **No config / no checks**: Not treated as error; exit 0 with informational message.
- **Errors**: Config load failure, invalid `checks` structure, or run failure (e.g. command not found) → exit non-zero, message on stderr.

### Security and trust

- Config is trusted (same as today for `ide.command`, `workspace.setup`, etc.). Document that `checks` are executed in the config directory; repo maintainers control `kira.yml`.

## Acceptance Criteria

- [ ] `checks` is supported in config: list of objects with `name`, `command`, and optional `description`. Loaded and merged with defaults (empty list when absent).
- [ ] `kira check` exists and runs each configured check in order from the config directory; streams output; exits non-zero on first failure and reports which check failed.
- [ ] When no checks are configured, `kira check` exits 0 and prints an informational message (e.g. “no checks configured”).
- [ ] A way to list configured checks exists (e.g. `kira check list` or `kira check --list`): prints name and description for each; exits 0 when none configured with same message as above.
- [ ] Config load or invalid `checks` structure produces a clear error and non-zero exit.
- [ ] Unit tests cover: empty checks, single check success/failure, multiple checks (first failure stops run), list output.

## Slices

### Slice 1: Config and data model
- Add `Checks` field to config struct and defaults (empty slice).
- Parse and merge `checks` from `kira.yml`; no validation beyond structure (name/command present).
- **Done**: Config load returns checks; tests for load/merge.

### Slice 2: `kira check` run
- Implement `kira check` (or `kira check run`) that runs each configured command in config directory.
- Stream output; exit on first failure with clear message.
- Handle “no checks configured” with message and exit 0.
- **Done**: `make check`-style workflow works via config; tests.

### Slice 3: List checks
- Implement list (subcommand or flag): print name and description for each check.
- Reuse “no checks configured” behavior.
- **Done**: Users and scripts can discover configured checks; tests.

### Slice 4: Docs and release
- Document `checks` in config docs (or README); document `kira check` and list in CLI help.
- Add release note.

## Implementation Notes

- **Execution**: Use `exec.Command` with shell for `command` (e.g. `sh -c "<command>"`) so pipelines and shell built-ins work; run in `Config.ConfigDir`. Follow project security guidelines (no unsanitized user input in commands; config is trusted).
- **Order**: Preserve order of `checks` as in YAML; run in that order.
- **Naming**: Prefer `kira check` as the main command; subcommands `kira check run` and `kira check list` if Cobra structure favors it, or a single command with `--list` for simplicity.
- **038 follow-up**: After this PRD, `kira config get checks` (or a key like `checks`) can be added in 038 to expose the list to scripts; format TBD (e.g. JSON array of names or full entries).

## Release Notes

- **Check commands**: New `checks` config and `kira check` command to define and run project check commands (e.g. lint, test, security) from a single entry point; supports agents and scripts that need “run the project’s checks.”
