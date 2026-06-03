---
name: kira-define-user-journey
description: >-
  Define or refine a user journey (JTBD + stages + success criteria) via structured discovery questions, then produce a markdown journey doc. Use when the user asks for a user journey/journey map, JTBD, stages/phases, or success criteria.
disable-model-invocation: false
---

# Kira: Define user journey

## When to use

Use this skill when the user wants to:

- Capture or document a journey end-to-end (“how does X work?”, “document the flow”)
- Create or refine a journey map (stages/phases)
- Write or improve a JTBD (“As a / I want / So that”)
- Make success criteria/outcomes explicit for a journey

## Goal

Produce a journey document that:

1. **Reads well for people** with a clear **Job to be done**, story, motivation, and stages.
2. **Reads well for LLMs** with stable vocabulary, explicit actors/triggers, **success criteria** (checkable outcomes), boundaries, and links to artifacts/touchpoints.

Do that by **eliciting** missing detail in structured question blocks—not one giant unstructured list.

## Inputs (confirm briefly)

Confirm briefly:

- **Working title** of the journey (becomes the H1).
- **Primary actor** - exactly one main “who”; note secondary actors if needed.
- **Target path** if they want a file (Kira default: `.docs/product/user_journeys/<slug>.md`; slug = short kebab-case).

Scope:
One journey file should represent a single core scenario end-to-end (one primary actor + goal + situation). Split if it branches into a different story.

If they already have notes, skim them first; prefer **extending** over restarting.

## Instructions

### 1) Discovery: ask structured questions

Read **`kira-clarifying-questions-format`** and follow it for every question block:

- Use `## Questions` (or `## Questions: <theme>` when appropriate) and `### N. Title`.
- Only include `#### Options` + checkboxes when there is a real choice.
- Prefer one theme per message/turn; don’t send one giant unrelated list.

Repo/tooling constraints for question headings and compatibility are documented in:

- `references/repo-constraints.md`

### 2) Draft: write the journey doc quickly (iterate)

Use the bundled template:

- `references/journey-template.md`

Recommended authoring order:

- Draft **JTBD** + (optional) **Supporting evidence**
- Draft **Stages** (what happens over time; what gets produced/recorded)
- Draft **Success criteria** (checkable outcomes)
- Fill **Measures of success** last (can remain `TBD` until flow is stable)

### 3) Quality bar (don’t invent)

- Prefer concrete observables over vibes.
- Use one canonical term per concept; add a glossary if needed.
- Mark unknowns as `TBD` and capture them under `## Open questions`.
- Split multiple unrelated scenarios into separate journey files.

Use the question prompt menu when discovery stalls:

- `references/question-bank.md`

## Output

- **Default path**: `.docs/product/user_journeys/<slug>.md`
- **Recommended structure**: JTBD first, then `## Stage N: ...`, then `## Success criteria`, with optional sections (touchpoints/artifacts, pain points, edges).

See also:

- `../../../.docs/product/user_journeys/README.md`

## Examples

- If the user says “capture the buyer onboarding journey,” confirm Inputs, ask one JTBD-focused Questions block, then draft using the template and iterate.
- If the user already has a journey doc, extend it (often `## Success criteria` + `## Edge cases and non-goals`) instead of rewriting.

## Conventional journey-map terms (how Kira lines up)

These docs are **markdown journeys**, not wall-sized posters, but the vocabulary should match what design and product folks already use:

| Common practice | In Kira journey docs |
| --- | --- |
| Actor / persona | **`## Job to be done`** — **As a** [actor]; link a persona doc when one exists; expand in **`## Who this is for`** if secondary actors matter |
| Goal / motivation (incl. user-story spine) | **`## Job to be done`** — **I want** [outcome]; **So that** [benefit]. Same intent as classic **Jobs to be Done** and “user story” framing |
| Situation, kickoff | Optional lines under the same section: **Scenario** (context/constraints), **Trigger** (what starts this path) |
| Stages / phases (timeline) | **`## Stage N: …`** (or **`### Stage N: …`** under **User Flow**) — use **Stage**, not “Step”, for journey beats |
| Actions, system response | Inside each stage: prose or bullets (**Actions**, **System / org response**, **Recorded / produced**). Finer-grained work is **actions** or domain **items** (e.g. checklist rows), not numbered “steps” of the journey. **Heading format:** `Stage 1: Title` (colon and space after the number), not an em dash |
| Touchpoints | **Product-facing:** channels and surfaces (app, email, store, support). **Internal / dev:** often **artifacts** (repo paths, CLI, tickets)—call that out explicitly |
| Pain points, opportunities | **`## Pain points and opportunities`** |
| Thoughts / emotions | **Optional** **`## Thoughts and emotions`** for user-facing journeys only—always pair with observables, never instead of them |
| Success / outcomes | **`## Success criteria`** (checkable outcomes; same idea as “observable outcomes”) |
| Value hypothesis, evidence, learning | Under **`## Job to be done`**: **`### Supporting evidence`**, **`### Measures of success`** — compare to real **usage** after ship |

A **mermaid diagram** or summary **table** is optional but encouraged when it replaces long prose (see the [user journeys README](../../../.docs/product/user_journeys/README.md)).

## Job to be done (JTBD) — “As a …, I want …, so that …”

Lead the journey with a **Job to be done** so the **why** is obvious before the stages.

**Required shape** (use exactly these labels so readers and tools can scan consistently):

```markdown
## Job to be done

- **As a** <single primary actor or persona>
- **I want** <concise outcome or capability this journey delivers>
- **So that** <concrete benefit — why it matters; avoid empty platitudes>

**Scenario:** <optional — situation, constraints, “when this applies” if not clear above>
**Trigger:** <optional — event or request that starts this path>
```

**Using the three clauses well**

- **As a** — One **primary** actor per journey. Name the real role or persona (“buyer”, “decision owner on the loop”), not only “user”. Secondary actors belong in **`## Who this is for`** or in the stages, not crammed into **As a**.
- **I want** — What this **journey** achieves end-to-end, not the entire product vision unless this file truly spans it. Verbs and outcomes beat vague nouns.
- **So that** — Must add information **beyond** *I want*; spell out the stakeholder-visible benefit (confidence, time saved, risk reduced, fewer repeats). If it repeats *I want*, tighten the wording.
- **Internal / developer-tooling journeys** — Still use the same three lines; the actor is the role (e.g. “As a maintainer…”, “As a tech lead…”). Do not switch to “we want” unless the actor is genuinely the whole team as one unit.
- **Single paragraph** — Acceptable **only** if all three intents (actor, want, benefit) are **unmistakably** present in one or two sentences; prefer the bulleted form for new and updated files.

**Issues to avoid**

- **So that** fluff (“so that things are better”, “so that I can succeed”) with no checkable or emotional specificity.
- Ten unrelated **I want** clauses — split into **separate journey files** (one scenario per doc).
- **As a** lists multiple incompatible actors — pick one primary journey or split.

**Challenge the value, not only the wording** — The JTBD is the right moment to ask whether this journey is **worth building**. If stakeholders cannot point to evidence, treat “So that” as a hypothesis to test, not a promise. See `references/question-bank.md`.

## References (bundled)

- `references/question-bank.md` — prompt menu for discovery (pick what fits; skip irrelevant).
- `references/journey-template.md` — the recommended journey markdown template.
- `references/repo-constraints.md` — repo-specific `## Questions` heading constraints / tooling compatibility.

## After the draft

1. Offer a **short pass** for clarity (shorter sentences, stronger verbs, fewer duplicate ideas).
2. **Finalize** **`### Measures of success`** if it is still **TBD**—only after **`## Success criteria`** and stages reflect the agreed journey.
3. If the file is new, **write** it to the agreed path under `.docs/product/user_journeys/` (or the user’s path).
4. If `.docs/README.md` should mention the journey folder for newcomers, suggest an update—**do not** edit README unless the user asked to maintain docs indexes.

## Anti-patterns

- One message with an oversized **`## Questions`** section (many unrelated `###` items)—split by theme across turns, or use `## Questions: <theme>` sections where appropriate.
- **Multiple unrelated scenarios** in one file—split into separate journeys (one actor–goal–situation path per doc).
- Stages that only describe feelings without **observable** behavior (unless **`## Thoughts and emotions`** is explicit and **success criteria** still stand on their own).
- Mixing **two primary actors** in one journey without signposting (split or declare a chapter per actor).
- Implementation-only jargon in the title or stages without glossary when outsiders read it.
- Labeling repo paths as **touchpoints** without explaining what the actor does there—or the reverse, ignoring real customer **touchpoints** in product journeys.
- Numbered **Step** headings for the **journey timeline**—use **`## Stage N: …`** (top-level) or **`### Stage N: …`** under **`## User Flow`** when the file also uses other **`##`** sections for reference material (URLs, data model, etc.); reserve **items** / **actions** for finer-grained work inside a stage.
- **JTBD** with a vague **So that**, or **I want** that bundles several unrelated journeys—tighten or split files.
- **High-stakes bets** (customer-facing, revenue, compliance) with **no** **`### Supporting evidence`** or **`### Measures of success`**—add them or record **TBD** and owner.

## Related skills

- **kira-clarifying-questions-format** — required structure for stakeholder question blocks; add **`#### Options`** + checkboxes + **Suggested** when (and only when) they must choose among alternatives.
- **kira-elaborate-work-item** — when the journey should fold into a single work item’s behaviour section instead of a standalone doc.

