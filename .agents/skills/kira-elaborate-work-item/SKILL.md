---
name: kira-elaborate-work-item
description: "Elaborate a work item’s behaviour and intent: value, outcomes, rules, and flows. Stays in the behavioural space—not implementation design. Use before slicing or delivery planning."
disable-model-invocation: false
---

# Kira: Elaborate Work Item

## Overview

Elaborate the work item so value, rules, and behaviour are clear enough to slice or build. Use project context for accuracy only; do not invent technical solutions here.

## When to use

- Value, rules, or behaviour are unclear but the item is likely to be built
- Before slicing or delivery planning when acceptance criteria or examples are missing or ambiguous

## Principles

- Keep the write-up consistent, concise, and non-repetitive
- **Behavioural space**: outcomes and rules in plain language; avoid solution and implementation decisions
- Technical context from the repo informs accuracy of behaviour, not the choice of design or stack
- Uncertainty → add a **`## Questions`** section using **kira-clarifying-questions-format** (structure, checkboxes, suggested option)
- Any assumption you make to proceed → record it explicitly in the work item so it can be confirmed or corrected

## Steps

1. **Orient** — Check `kira.yaml` for work folders and templates; use `.work/` and `kira slice` if the project uses slices
2. **Read** — Work item (title, description) and linked docs; skim technical context only to ground value, rules, and flows
3. **Value and outcomes** — Who benefits, what success looks like if this ships, **non-goals / exclusions**
4. **Rules** — Policies and constraints in business language; edge and exceptional cases; still no implementation prescriptions
5. **Flows** — User or system flows (steps, decisions, alternates); examples where helpful; dependencies and touchpoints
6. **Update the work item** — Fold in the above; add **`## Questions`** (per kira-clarifying-questions-format) and **assumptions / open items** for handoff

## If blocked

When value is unclear, rules conflict, or describing flow would force a technical decision: offer options to choose from, or record an open question—do not settle the solution inside this elaboration.

## Output

Updated work item (project templates and locations) containing value/outcomes, rules (including edge cases), flows/scenarios, non-goals, **Questions**, and stated assumptions.
