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
- **New run id:** when not using `--resume`, the runner derives `run-id` from the resolved workflow identity (e.g. mapped name or script basename) and a timestamp, prints it (so the user can `--resume` later), and uses it for `.workflows/sessions/<run-id>.yml`. If a session file for that id already exists, fail fast with a clear error (collisions should be vanishingly rare with a fine-grained timestamp).

**Script**

- Entrypoint: `Run(ctx *kirarun.Context, step kirarun.Step, agents kirarun.Agents) error` — the third parameter is the agent execution surface; workflows that do not invoke agents may ignore it with `_`. The concrete `kirarun.Agents` type is defined by the host for this kira version.
- **Validate before run:** the workflow source must load under Yaegi (parse/compile), expose `Run` with that exact signature (package `main`, correct parameter types and arity), and only use the `kirarun` API the host registers for this kira version. If the script is invalid, the signature is wrong, or it calls types/methods not provided by the host, fail with a clear error — do not create or advance a session.
- `ctx` is **read-only** from the workflow: workspace paths, logging, read-only view of skills/commands, plus `ctx.Run` (e.g. `Attempt()` 1-based, `IsResume()` when `--resume` was passed, `IgnoreAttemptLimit()` when `--ignore-attempt-limit` was passed). Do not mutate host or agent state through `ctx`. Non-deterministic agent work goes through the third parameter, not side effects on `ctx`. Retry/backoff and caps live in the script via `Attempt()` and normal Go code (optional `const maxAttempts`).
- `step.Do[T](name, fn)` — idempotent steps: persist successful `T` as JSON per step name; skip `fn` if that step already succeeded for this run; `fn` error marks step failed/resumable. Prefer typed structs; `map[string]any` only if shape is dynamic. If persisted JSON cannot decode into `T` on resume, fail with a clear step-scoped error.
- Interpreter: no broad `unsafe` / syscall surface by default.

**Session file**

- One YAML per run: `.workflows/sessions/<run-id>.yml` (not a directory). Create/update **before** calling `Run` with at least current `attempt` so panics on the first line still leave a record. Successful steps under `steps`; failures under `attempts`. Remove the file when the run completes successfully. Session create failure → do not run the workflow.
- **Validate on load:** if the file exists but is not valid YAML, or does not match the expected session schema (required fields, types, version if versioned), fail immediately with an error that identifies the file and what is wrong — do not run the workflow or partially apply state.
- Same session holds run-level attempt counter and per-step outputs; later attempts or `--resume` still skip completed `step.Do` work.

**Other**

- Concurrent runs for the same run id: lock session (or sibling lock) and fail fast if busy.

## Invoking

```bash
kira run hello_world
kira run .workflows/hello_world.go

# new runs print a derived run-id (workflow identity + timestamp) for later resume
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

	"kirarun"
)

func Run(ctx *kirarun.Context, _ kirarun.Step, _ kirarun.Agents) error {
	root := ctx.Workspace.Root()
	doing := filepath.Join(root, ".work", "2_doing")
	ctx.Log.Info(fmt.Sprintf("Scanning %s", doing))
	return nil
}
```

### B — `step.Do` with types

```go
package main

import (
	"fmt"

	"kirarun"
)

func Run(ctx *kirarun.Context, step kirarun.Step, _ kirarun.Agents) error {
	type step1Out struct {
		Foo   string `json:"foo"`
		Model string `json:"model"`
	}
	type step2Out struct {
		Summary string `json:"summary"`
	}

	s1, err := step.Do[step1Out]("step_1", func(_ kirarun.StepContext) (step1Out, error) {
		return step1Out{Foo: "bar", Model: "gpt-5"}, nil
	})
	if err != nil {
		return err
	}

	s2, err := step.Do[step2Out]("step_2", func(_ kirarun.StepContext) (step2Out, error) {
		return step2Out{Summary: fmt.Sprintf("foo=%s", s1.Foo)}, nil
	})
	if err != nil {
		return err
	}
	_ = s2
	return nil
}
```

### C — attempt / backoff (illustrative)

```go
package main

import (
	"errors"
	"time"

	"kirarun"
)

const maxAttempts = 3

func Run(ctx *kirarun.Context, step kirarun.Step, _ kirarun.Agents) error {
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

- [ ] `kira run` runs Yaegi workflows with `Run(ctx, step, agents)` and the flags above; invalid scripts or wrong `Run` signature / mismatched `kirarun` usage fail before execution with a clear error.
- [ ] Session file lifecycle (pre-invoke write, steps/attempts, delete on success, strict decode) matches this spec; malformed or schema-invalid session files error clearly and do not run the workflow.
- [ ] `step.Do` persists and skips completed steps across resume and retries; `ctx.Run.Attempt()` is 1-based; `--ignore-attempt-limit` sets `ctx.Run.IgnoreAttemptLimit()` without faking the counter.
- [ ] `ctx` is read-only to workflows; agent invocation uses the third `Run` parameter (`kirarun.Agents`). Provider wiring (e.g. Cursor first) targets that parameter. [PRD 024](.work/2_doing/024-product-dev-loop-triggers.prd.md) is background and may lag this spec.
- [ ] Concurrent same-run-id execution is guarded.
- [ ] `make check` passes.

## Slices

## Implementation Notes

- Narrow Yaegi surface via `interp.Use`; recover panics in the runner and record like a returned error where compatible with pre-invoke session writes.
- Document `--ignore-attempt-limit` vs persisted attempt counter (flag is advisory for script guards, not a counter override).
- Session YAML schema: path, kira version, run id, attempt, `attempts[]`, `steps[]` with per-step `data` — details belong in implementation docs/tests, not repeated here.
- Run id for new runs: encode workflow identity + timestamp in the derivation rule used in code and tests; document the exact format where operators need it.

## Release Notes
