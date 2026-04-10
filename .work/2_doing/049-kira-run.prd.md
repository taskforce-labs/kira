---
id: 049
title: kira run
status: doing
kind: prd
assigned:
estimate: 0
created: 2026-03-24
due: 2026-03-24
tags: []
---

# kira run

Run Go scripts with [Yaegi](https://github.com/traefik/yaegi) so users can trigger both deterministic workflows and non-deterministic agent based workflows.

This will enable [product dev loop triggers](.work/2_doing/024-product-dev-loop-triggers.prd.md), especially around resuming jobs.

## Requirements

**CLI**

- `kira run <script-or-workflow> [args]` — resolve by path or name under `workflows` in `kira.yml` (default: `<project-root>/.workflows/`).
- `--resume <run-id>` — continue a run; required when a session file already exists (runner fails fast otherwise). Exception: `--auto-retry` may reuse the same run id in one process without an extra `--resume`.
- `--auto-retry` — loop until `Run` returns nil or the user interrupts; same run id; each loop iteration bumps attempt (written before `Run`).
- `--ignore-attempt-limit` — this invocation only: do not treat the script’s own attempt cap as fatal (e.g. after `maxAttempts`); exposed on context (e.g. `ctx.Run.IgnoreAttemptLimit()`) so `Run` can skip an `Attempt() > maxAttempts` early return. The persisted attempt counter is unchanged.
- **New run id:** when not using `--resume`, the runner derives `run-id` as `<script-name>-<timestamp>` (script name from the resolved workflow identity, e.g. mapped name or script basename without extension), prints it (so the user can `--resume` later), and uses the same value for the session filename `.workflows/sessions/<run-id>.yml`. If a session file for that id already exists, fail fast with a clear error (treat rare random-id collisions like any other existing session).

**Script**

- Entrypoint: `Run(ctx *kirarun.Context, step *kirarun.Step, agents kirarun.Agents) error` — the third parameter is the agent execution surface; workflows that do not invoke agents may ignore it with `_`. The concrete `kirarun.Agents` type is defined by the host for this kira version. Idempotent steps use the package function `kirarun.Do(step, name, fn)` (Go has no generic methods).
- **Validate before run:** the workflow source must load under Yaegi (parse/compile), expose `Run` with that exact signature (package `main`, correct parameter types and arity), and only use the `kirarun` API the host registers for this kira version. If the script is invalid, the signature is wrong, or it calls types/methods not provided by the host, fail with a clear error — do not create or advance a session.
- `ctx` is **read-only** from the workflow: workspace paths, logging, read-only view of skills/commands, plus `ctx.Run` (e.g. `Attempt()` 1-based, `IsResume()` when `--resume` was passed, `IgnoreAttemptLimit()` when `--ignore-attempt-limit` was passed). Do not mutate host or agent state through `ctx`. Non-deterministic agent work goes through the third parameter, not side effects on `ctx`. Retry/backoff and caps live in the script via `Attempt()` and normal Go code (optional `const maxAttempts`).
- `kirarun.Do(step, name, fn)` — idempotent steps: persist successful output as **JSON** (one object per step name in the session file); skip `fn` if that step already completed for this run; `fn` error marks the step failed / resumable. **How return types work** (compiled vs Yaegi, coercion): see **Step outputs** below.
- **Step outputs (return types and typing rules)**
  - **What is stored:** `steps[].data` is always a JSON object. In code, represent that with a **struct** and **`json` tags** on each field, unless the payload has **dynamic keys** (then `map[string]any` is acceptable).
  - **Compiled Go** (packages built with `go build`, including tests and host code): use **`kirarun.Do[T]`** with a callback **`func(kirarun.StepContext) (T, error)`**. The **compiler** knows `T` for the return value and for resume decoding inside `Do`.
  - **Yaegi workflows** (scripts under `.workflows/`): the host registers **`kirarun.Do[any]`** only (one concrete symbol via `interp.Use`). Callbacks use **`func(kirarun.StepContext) (any, error)`** — you may still **return a struct value** inside that `any`. The **compiler does not** check that the value matches the struct type you use in the rest of the script; that limitation comes from `any`, not from missing helpers.
  - **Unmarshal after `Do` in scripts:** call **`kirarun.UnmarshalStepData(raw, &out)`** where `out` is your tagged struct variable and **`&out` is its address** (same rule as `json.Unmarshal`: the unmarshaler must receive a pointer so it can write fields into your variable). That performs a **runtime** JSON round-trip so the same code works whether `raw` is struct-backed (common on first run) or **`map[string]any`**-backed (common after `--resume`). If JSON does not fit `out`, you get an error **when unmarshaling**, not from `go build`.
  - **Compiled-only sugar:** **`kirarun.UnmarshalStepDataAs[T](raw)`** — same behavior, returns a `T`; convenient when you are not under Yaegi. Scripts use **`UnmarshalStepData`** (registered for the interpreter).
- Interpreter: no broad `unsafe` / syscall surface by default.

**Session file**

- One YAML per run: `.workflows/sessions/<run-id>.yml` (not a directory). Create/update **before** calling `Run` with at least current `attempt` so panics on the first line still leave a record. Successful steps under `steps`; failures under `attempts`. Remove the file when the run completes successfully. Session create failure → do not run the workflow.
- **Validate on load:** if the file exists but is not valid YAML, or does not match the expected session schema (required fields, types, version if versioned), fail immediately with an error that identifies the file and what is wrong — do not run the workflow or partially apply state.
- Same session holds run-level attempt counter and per-step outputs; later attempts or `--resume` still skip completed `step.Do` work.

**Session file format (YAML)**

Shape and semantics (field names and nesting are normative; value placeholders describe intent):

```yaml
path: <absolute-path-to-script>           # written on first run
kira-version: <kira-version>              # written on first run
run-id: <script-name>-<timestamp>         # written on first run; matches session filename stem
attempt: <int>                            # written / incremented before running
attempts:
  - attempt: <int>                        # written on failure
    name: <run | step-name>
    started_at: <timestamp>
    failed_at: <timestamp>
    error:
      <specific-to-error>
steps:
  - name: <step-name>                     # written on success of step
    attempt: <int>
    started_at: <timestamp>
    finished_at: <timestamp>
    data:
      <specific-to-step>
```

- `path` and `kira-version` are set when the session is first created for a run.
- `run-id` in the file must match the session file stem and the id printed for `--resume`.
- `attempts[].name` is `run` for a top-level `Run` failure (or equivalent), or the `step.Do` step name for a step failure.
- `steps[].data` holds the JSON object written for each successful step; typing and unmarshal rules are under **Step outputs** in **Script**.

**Other**

- Concurrent runs for the same run id: lock session (or sibling lock) and fail fast if busy.

## Invoking

```bash
kira run hello_world
kira run .workflows/hello_world.go

# new runs print a derived run-id (<script-name>-<timestamp>) for later resume
# resume
kira run foo_bar --resume run-abc123
kira run .workflows/foo_bar.go --resume run-abc123

# retry loop (same run id; attempt bumped before each Run)
kira run foo_bar --auto-retry

# keep going even when the script would stop on maxAttempts (counter unchanged; script checks IgnoreAttemptLimit)
kira run foo_bar --resume run-abc123 --ignore-attempt-limit

kira run examples/fizz-buzz
```

## Examples

### A — deterministic

```go
package main

import (
	"fmt"
	"path/filepath"

	"kira/kirarun"
)

func Run(ctx *kirarun.Context, _ *kirarun.Step, _ kirarun.Agents) error {
	root := ctx.Workspace.Root()
	doing := filepath.Join(root, ".work", "2_doing")
	ctx.Log.Info(fmt.Sprintf("Scanning %s", doing))
	return nil
}
```

### Hello world — three steps (fetch → construct → say)

Canonical script: `.workflows/hello_world.go`. Matches **Step outputs** in **Requirements** (Yaegi + `Do[any]` + `UnmarshalStepData`). Three idempotent steps: **`get_greeting`** (stub **phrase** + optional **style**; later LLM via **`kirarun.Agents`**), **`construct_greeting`** (`phrase + " world"`), **`say_greeting`** (stdout).

**Shape:** `runStepOneGetGreeting` / `runStepTwoConstructGreeting` / `runStepThreeSayGreeting` each wrap `kirarun.Do` + `UnmarshalStepData` with the step body inlined in the callback. `Run` is a short pipeline.

```go
package main

import (
	"fmt"

	"kira/kirarun"
)

type getGreetingOut struct {
	Phrase string `json:"phrase"`
	Style  string `json:"style"`
}

type constructGreetingOut struct {
	Message string `json:"message"`
}

type sayGreetingOut struct {
	Printed bool `json:"printed"`
}

func runStepOneGetGreeting(step *kirarun.Step, ctx *kirarun.Context, agents kirarun.Agents) (getGreetingOut, error) {
	raw, err := kirarun.Do(step, "get_greeting", func(_ kirarun.StepContext) (any, error) {
		_ = ctx
		_ = agents
		return getGreetingOut{Phrase: "G'day", Style: "australia"}, nil
	})
	if err != nil {
		return getGreetingOut{}, err
	}
	var out getGreetingOut
	if err := kirarun.UnmarshalStepData(raw, &out); err != nil {
		return getGreetingOut{}, fmt.Errorf("get_greeting: %w", err)
	}
	return out, nil
}

func runStepTwoConstructGreeting(step *kirarun.Step, in getGreetingOut) (constructGreetingOut, error) {
	raw, err := kirarun.Do(step, "construct_greeting", func(_ kirarun.StepContext) (any, error) {
		return constructGreetingOut{Message: in.Phrase + " world"}, nil
	})
	if err != nil {
		return constructGreetingOut{}, err
	}
	var out constructGreetingOut
	if err := kirarun.UnmarshalStepData(raw, &out); err != nil {
		return constructGreetingOut{}, fmt.Errorf("construct_greeting: %w", err)
	}
	return out, nil
}

func runStepThreeSayGreeting(step *kirarun.Step, in constructGreetingOut) (sayGreetingOut, error) {
	raw, err := kirarun.Do(step, "say_greeting", func(_ kirarun.StepContext) (any, error) {
		fmt.Println(in.Message)
		return sayGreetingOut{Printed: true}, nil
	})
	if err != nil {
		return sayGreetingOut{}, err
	}
	var out sayGreetingOut
	if err := kirarun.UnmarshalStepData(raw, &out); err != nil {
		return sayGreetingOut{}, fmt.Errorf("say_greeting: %w", err)
	}
	return out, nil
}

func Run(ctx *kirarun.Context, step *kirarun.Step, agents kirarun.Agents) error {
	fetched, err := runStepOneGetGreeting(step, ctx, agents)
	if err != nil {
		return err
	}
	built, err := runStepTwoConstructGreeting(step, fetched)
	if err != nil {
		return err
	}
	_, err = runStepThreeSayGreeting(step, built)
	return err
}
```

### B — two typed steps chained (minimal pattern)

Same **Step outputs** rules as **Hello world** (structs + tags, `(any, error)` under Yaegi, `UnmarshalStepData` between steps); this snippet is only two steps for brevity — the canonical pipeline is three steps in `.workflows/hello_world.go`.

```go
package main

import (
	"fmt"

	"kira/kirarun"
)

type step1Out struct {
	Foo   string `json:"foo"`
	Model string `json:"model"`
}

type step2Out struct {
	Summary string `json:"summary"`
}

func Run(ctx *kirarun.Context, step *kirarun.Step, _ kirarun.Agents) error {
	raw1, err := kirarun.Do(step, "step_1", func(_ kirarun.StepContext) (any, error) {
		return step1Out{Foo: "bar", Model: "gpt-5"}, nil
	})
	if err != nil {
		return err
	}
	var s1 step1Out
	if err := kirarun.UnmarshalStepData(raw1, &s1); err != nil {
		return fmt.Errorf("step_1: %w", err)
	}

	raw2, err := kirarun.Do(step, "step_2", func(_ kirarun.StepContext) (any, error) {
		return step2Out{Summary: fmt.Sprintf("foo=%s model=%s", s1.Foo, s1.Model)}, nil
	})
	if err != nil {
		return err
	}
	var s2 step2Out
	if err := kirarun.UnmarshalStepData(raw2, &s2); err != nil {
		return fmt.Errorf("step_2: %w", err)
	}
	_ = s2 // use s2.Summary in later steps or logging
	return nil
}
```

### C — attempt / backoff (illustrative)

```go
package main

import (
	"errors"
	"time"

	"kira/kirarun"
)

const maxAttempts = 3

func Run(ctx *kirarun.Context, step *kirarun.Step, _ kirarun.Agents) error {
	if !ctx.Run.IgnoreAttemptLimit() && ctx.Run.Attempt() > maxAttempts {
		return errors.New("max attempts exceeded")
	}
	if ctx.Run.Attempt() > 1 {
		time.Sleep(time.Duration(1<<uint(ctx.Run.Attempt()-2)) * time.Second)
	}
	// ... step.Do / main work ...
	return nil
}
```

## Acceptance Criteria

- [x] `kira run` runs Yaegi workflows with `Run(ctx, step, agents)` and the flags above; invalid scripts or wrong `Run` signature / mismatched `kirarun` usage fail before execution with a clear error.
- [x] Session file lifecycle (pre-invoke write, steps/attempts, delete on success, strict decode) matches this spec; malformed or schema-invalid session files error clearly and do not run the workflow.
- [x] `kirarun.Do` persists and skips completed steps across resume and retries; `ctx.Run.Attempt()` is 1-based; `--ignore-attempt-limit` sets `ctx.Run.IgnoreAttemptLimit()` without faking the counter.
- [x] `ctx` is read-only to workflows; agent invocation uses the third `Run` parameter (`kirarun.Agents`). Provider wiring (e.g. Cursor first) targets that parameter. [PRD 024](.work/2_doing/024-product-dev-loop-triggers.prd.md) is background and may lag this spec.
- [x] Concurrent same-run-id execution is guarded.
- [x] `make check` passes.
- [x] Step data unmarshal: `kirarun.UnmarshalStepData` / `UnmarshalStepDataAs` implemented and covered by tests; Yaegi export includes `UnmarshalStepData`; `.workflows/hello_world.go` uses it (slice 7).

## Slices

### 1. Session file schema, I/O, and per-run locking
Commit: Persistent session model at `.workflows/sessions/<run-id>.yml`, strict validation on load, create/update before any workflow invocation, delete on success, and exclusive lock for a given run id.
- [x] T001: Define versioned session YAML schema matching **Session file format (YAML)** (`path`, `kira-version`, `run-id`, `attempt`, `attempts`, `steps`); reject malformed YAML and schema-invalid files with errors that name the file and the problem.
- [x] T002: Implement load/save helpers: ensure parent dirs exist; write at least `attempt` before the workflow runs; remove the session file when the run completes successfully; surface create/write failures without running the workflow.
- [x] T003: Add a same-run-id lock (session-adjacent lock file or equivalent): concurrent second invocation fails fast with a clear message.
- [x] T004: Unit tests covering happy path, bad YAML, wrong schema, and lock contention.

### 2. `kirarun` API and `step.Do` (host-registered surface)
Commit: Read-only `Context`, `ctx.Run` (1-based `Attempt()`, `IsResume()`, `IgnoreAttemptLimit()`), `Step` / `StepContext`, generic `kirarun.Do` with JSON persistence and resume semantics, and a concrete `Agents` type placeholder for the host.
- [x] T005: Implement `kirarun` types matching the PRD examples: workflows cannot mutate host state via `ctx`; `Agents` is the extension point for agent execution.
- [x] T006: Implement `kirarun.Do[T]`: persist successful `T` per step name, skip `fn` when already succeeded, mark failures/resumable state; on resume, fail with a clear step-scoped error if stored JSON does not decode into `T`.
- [x] T007: Unit tests for `step.Do` and `ctx.Run` behavior without Yaegi (in-memory or test session backend wired to slice 1 types).

### 3. Yaegi load, symbol registry, and `Run` validation
Commit: Narrow interpreter surface via `interp.Use`, parse/compile workflow source, require `package main` and `Run(ctx *kirarun.Context, step *kirarun.Step, agents kirarun.Agents) error`; invalid scripts fail before any session is created or advanced.
- [x] T008: Register only the `kirarun` API intended for this kira version; exclude broad `unsafe` / syscall by default.
- [x] T009: Validate `Run` exists with the exact signature; return clear errors for wrong arity, types, or disallowed calls.
- [x] T010: Tests with valid and invalid sample workflow sources.

### 4. Runner orchestration (invoke, panic recovery, flags)
Commit: End-to-end runner: derive/accept run id, session pre-write, invoke `Run`, recover panics into errors where compatible with session updates, `--resume`, `--auto-retry` (attempt bumped before each `Run`), `--ignore-attempt-limit` on context only (persisted attempt unchanged), collision handling for new run ids.
- [x] T011: New run id as `<script-name>-<UTC-timestamp>` (second precision); print it for operator use; fail fast if a session file already exists for that id (including same-second collision).
- [x] T012: `--resume` required when a session already exists for the target run (except `--auto-retry` single-process reuse per PRD); wire `--ignore-attempt-limit` to `ctx.Run.IgnoreAttemptLimit()` without changing stored attempt.
- [x] T013: `--auto-retry`: loop until `Run` returns nil or interrupt; same run id; bump attempt in the session before each `Run` entry.
- [x] T014: Panic recovery in the runner with session/recording behavior aligned with pre-invoke writes.
- [x] T015: Integration tests covering resume, retry loop, and ignore-attempt-limit guards.

### 5. `kira run` CLI and workflow resolution
Commit: `kira run <script-or-workflow> [args]` resolving by path or name under `workflows` in `kira.yml` with default root `.workflows/`, forwarding script args, and exposing all runner flags.
- [x] T016: Parse flags and positional workflow selector; resolve workflow file from `kira.yml` and repo layout.
- [x] T017: Register the subcommand under the existing Cobra/root command pattern; document usage in help text.
- [x] T018: CLI or integration tests for resolution and flag passthrough.

### 6. Acceptance, concurrency, and repo quality gate
Commit: Same-run-id concurrency story verified end-to-end, any operator-facing run-id format documented, and `make check` (plus e2e script per `AGENTS.md`) green.
- [x] T019: Verify concurrent same-run-id execution is guarded (stress or targeted test).
- [x] T020: Document run-id derivation rule and `--ignore-attempt-limit` vs persisted attempt where operators need it (implementation note or short doc pointer).
- [x] T021: `make check` passes; run `bash kira_e2e_tests.sh` and fix failures tied to this feature.

### 7. Step data unmarshal (`UnmarshalStepData` / `UnmarshalStepDataAs`)
Commit: First-class `kirarun` helpers for turning `Do[any]` results into typed structs via JSON (same shape on first run vs `--resume`); Yaegi registers `UnmarshalStepData`; canonical workflow drops local `jsonRoundTrip`.
- [x] T022: Implement `kirarun.UnmarshalStepData(v any, ptr any) error` — `ptr` must be a non-nil pointer; `json.Marshal(v)` then `json.Unmarshal` into `ptr`; clear errors if `ptr` is invalid or JSON shape does not match.
- [x] T023: Implement `kirarun.UnmarshalStepDataAs[T any](v any) (T, error)` as thin wrapper over the same logic (compiled ergonomics); unit tests cover struct-in-`any`, `map[string]any`, and mismatch cases for both APIs.
- [x] T024: Register `UnmarshalStepData` in `internal/kirarun/yaegi/exports.go` (`UnmarshalStepDataAs` compiled-only unless Yaegi gains reliable generic calls); extend validation/tests if the symbol surface is asserted anywhere.
- [x] T025: Update `.workflows/hello_world.go` to use `UnmarshalStepData`; remove the local `jsonRoundTrip` helper; ensure `kira run` / resume still passes existing integration coverage.

## Implementation Notes

- Step typing and Yaegi vs compiled behavior are specified once under **Requirements → Script → Step outputs**; avoid duplicating that narrative here.
- Narrow Yaegi surface via `interp.Use`; recover panics in the runner and record like a returned error where compatible with pre-invoke session writes.
- Document `--ignore-attempt-limit` vs persisted attempt counter (flag is advisory for script guards, not a counter override).
- Session YAML: follow **Session file format (YAML)** in Requirements; pin exact timestamp formats and error object shapes in code/tests.
- Run id for new runs: `<script-name>-<UTC-timestamp>` (second precision, e.g. `hello-20060102150405`). `--ignore-attempt-limit` does not change the persisted `attempt` counter in the session file; scripts gate on `ctx.Run.IgnoreAttemptLimit()` locally.

## Release Notes
