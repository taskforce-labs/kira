---
name: kira-create-adr
description: "Create a new Architecture Decision Record: correct location and naming from project config, standard sections, link from the work item. Use for long-lived or cross-cutting technical forks—not informal notes."
disable-model-invocation: false
---

# Kira: Create ADR

## When

- A design choice is **architectural** (constraints many features, hard to reverse, or needs a durable audit trail)—not a routine implementation detail.
- **kira-design-solution-for-work-item** or review flagged “decide via ADR”.

## Placement and naming

- **Directory:** `{docs_folder}/architecture/` from `kira.yml` (often `.docs/architecture/`). Do not invent another root.
- **File:** `<id>-<slug>.adr.md` — zero-padded numeric `id` (next after existing `*-*.adr.md` in that folder), `slug` kebab-case from the title.
- If **`kira adr new`** exists, use it for path, id, and template; otherwise create the file manually following the same rules.

## Content

- Follow the project template if present (e.g. `architecture/template.adr.md`); else use **Context** (problem, forces, options considered), **Decision** (what we chose and why), **Consequences** (trade-offs, follow-ups, what becomes harder/easier).
- Keep each section tight: enough for a future reader to decide whether the ADR still applies.

## After create

- Add the ADR path to the work item (question, solution, or `## Technical approach`) so implementation and review stay aligned.
- If this ADR **supersedes** an older one, mark the old file’s status accordingly (project convention or `kira adr` if supported).

## Out of scope

- Facilitation / options workshop.