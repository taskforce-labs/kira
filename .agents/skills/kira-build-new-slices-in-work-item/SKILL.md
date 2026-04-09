---
name: kira-build-new-slices-in-work-item
description: "Adds and delivers follow-on slices after an initial PRD/plan-and-build pass. Use when extending a work item with new slices, incremental scope, or post-PRD changes; includes a divergence circuit breaker before implementation."
disable-model-invocation: false
---

# Follow-on slices after the PRD

This skill applies **after** a work item already has a PRD and was planned/built (e.g. via `kira-plan-and-build-work-item`). You are **not** greenfield planning the whole item—you are **adding slices** (and tasks) for incremental or corrective work that still belongs to the same initiative.

## Before you add slices: divergence circuit breaker

Compare the **requested follow-on work** to the **original PRD** (Requirements, scope, non-goals, and any explicit constraints in the work item).

**Trip the breaker** (stop implementing on this branch; surface this to the user) when any of these are true:

- The new work **changes core goals**, **replaces** major requirements, or **contradicts** settled decisions in the PRD (not a small adjustment).
- The diff in intent is large enough that **commits would tell a misleading story** (many unrelated concerns, or “fixing” the product direction mid-branch).
- You would need to **rewrite most slices** or **delete/replace** large parts of what shipped to stay coherent.

When the breaker trips, tell the user clearly:

1. **Abandon or park this branch** for this direction of work (or split a separate branch/topic).
2. **Refine the work item**: update the PRD (or successor spec) so intent, scope, and acceptance criteria match what they actually want.
3. **Rework slices** from that updated baseline (`kira slice` / direct edit + `kira slice lint`), then plan delivery again.

Do **not** keep stacking follow-on slices on a branch that has **diverged** from the written PRD—align the document first.

**Safe to proceed** when the follow-on is clearly **incremental** (new bugfix slice, small feature add-on, perf/tests/docs, sub-scope that was always compatible) and still **traceable** to the PRD without rewriting history-of-intent.

If unsure, **ask** whether to treat the change as incremental follow-on or as a PRD refresh + slice reset.

## Adding new slices

1. Extend the work item’s `## Slices` section with new slice headings and tasks (or use `kira slice` commands—see project `AGENTS.md`).
2. Run `kira slice lint [current | <work-item-id>]` and fix errors.
3. Optionally re-check risk/readiness if the work item uses those annotations (`kira-analyse-work-item-risk`, `kira-review-work-item-ready-to-start`).

## Build loop (same as plan-and-build, per slice)

For each **slice** (not each task):

1. Get current slice/tasks: `kira slice show current` (or `kira slice show <work-item-id>`). Use `kira slice task show current` / `--output json` when helpful.
2. **Implement** all tasks in that slice; add/update tests; follow secure coding practices.
3. **Verify:** `kira check -t commit` before committing. Fix failures; only commit when checks pass.
4. **Commit** (one commit per slice): work item has that slice’s tasks marked done; `kira slice lint`; then `kira slice commit` (defaults: work item `current`, slice `completed`). Use `--dry-run` to preview.
5. Repeat for the next slice.

When all slices (including the new follow-on slices) are done, run `kira check -t done`.

Do not mark the work item done or in review unless the user asks; leave status per project convention.

## Relationship to `kira-plan-and-build-work-item`

| | Initial PRD delivery | This skill (follow-on) |
|---|----------------------|-------------------------|
| Context | First implementation from PRD | PRD exists; add slices for later changes |
| Planning | Full slice breakdown from requirements | Only new slices + circuit breaker |
| Commits | Same per-slice commit rules | Same per-slice commit rules |
