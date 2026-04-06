---
name: kira-design-solution-for-work-item
description: "After elaboration, articulate a technical solution grounded in the repo: code, journeys, ADRs, and architecture. Surface refactors, split oversized scope into follow-on work items, and escalate ambiguities via clarifying questions or ADRs."
disable-model-invocation: false
---

# Kira: Design solution for work item

## Overview

Usual run after kira-elaborate-work-item: turn behaviour/rules/flows into a concrete technical approach grounded in the repo (code paths, journeys, ADRs, architecture, tests)—not generic patterns. Produces the solution narrative kira-review-work-item expects.

## Scope

One work item should stay one coherent technical problem. If the design surfaces several unrelated deliverables, an enormous change surface, or themes that do not belong together, split into follow-on work items (titles, deps, “split from”) and trim this item—do not treat “one giant item” as the default.

## Principles

Anchor in real symbols/paths; extend existing abstractions; flag when the obvious approach fights local style. Note refactors: required for correctness vs optional/nice-to-have (deferrals + optional follow-up item). Don’t invent architecture silently—`## Questions` via kira-clarifying-questions-format. Long-lived or cross-cutting forks → capture with **kira-create-adr** and link the ADR from the question or solution text.

## Steps

1. Orient — `kira.yaml`, templates, work-item path; note existing `## Questions`. Stay in design scope—do not break work into commit-sized chunks here (that is a later skill).

2. Context — Code (entry points, similar features, libs, errors, config); journeys (CLI/API/UI); existing ADRs/arch constraints (add new ones via **kira-create-adr**); how comparable behaviour is tested.

3. Write solution (match project template, else `## Solution` or `## Technical approach`): short summary; approach (components, flow, integration, links to code/docs); refactors; risks (rollout, compat, perf, security as relevant); testing (what to add/update); out of scope / deferred.

4. Scope — If multiple disjoint deliverables or the plan is too large to reason about as one item, add `## Follow-on work items` (title, one-liner, blocking vs parallel) and narrow this item.

5. Gaps — Product/behaviour unknowns → `## Questions` (clarifying format). Architectural decision needed → **kira-create-adr** plus a question that links the ADR path (e.g. under `{docs_folder}/architecture/`).

## If blocked

Missing context: say what you looked for; ask or propose a small spike item. ADR pending: summarise the fork, link the ADR path (**kira-create-adr** when the file does not exist yet), state what can be designed now vs after the decision.

## Output

Updated work item: grounded solution (+ refactors, risks, testing, follow-ons). Questions and ADR links where decisions block implementation—ready for review and separate breakdown into delivery chunks when that skill is used later.
