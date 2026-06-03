## Journey template (markdown)

Use this as the base shape for new or heavily refactored journey docs under `.docs/product/user_journeys/`.

```markdown
# User journey: <short title>

## Job to be done

- **As a** <primary actor or persona>
- **I want** <outcome this journey delivers end-to-end>
- **So that** <concrete benefit>

**Scenario:** <optional — context, constraints>
**Trigger:** <optional — what starts this path>

<Optional: persona or product doc link.>

### Supporting evidence

<Facts, research, links — assumptions called out explicitly. Prefer “TBD” over filler.>

### Measures of success

<(Complete last.) Leading and lagging signals, falsifiers, review cadence — how we will know this delivered the “So that”. Align with Success criteria below. Use “TBD” until stages and success criteria are drafted.>

## Who this is for

<Optional: expand actor / secondary actors if not already clear above.>

## How this ties to <product / Kira / the loop>

<Optional. Links + one short paragraph.>

## Touchpoints and artifacts

<Optional. Product: surfaces/channels. Internal: repo paths, tools, tickets, dashboards.>

## Pain points and opportunities

<Optional.>

- **Pain:** … **Opportunity:** …

## Thoughts and emotions

<Optional; user-facing only; never instead of success criteria.>

---

## Stage 1: <short stage title>

<Prose or bullets: what happens, who drives it, what is produced/recorded.>

## Stage 2: <short stage title>

...

## Stage N: <short stage title>

...

---

## Success criteria

Checkable outcomes (same idea as “observable outcomes” in engineering docs).

- <Outcome a reviewer could verify>
- …

## Edge cases and non-goals

- **Edge:** … **Handling:** …
- **Non-goal:** …

## Glossary  <!-- optional -->

| Term | Meaning here |
| --- | --- |
| … | … |

## Open questions  <!-- keep narrow; prefer resolving via kira-clarifying-questions-format -->

- …

## Optional: Behaviors (Gherkin)  <!-- optional -->

```gherkin
Feature: <short name>

  Scenario: <main success path>
    Given …
    When …
    Then …
```
```

Notes:

- Prefer `## Stage N: ...` as the journey timeline. Use finer-grained bullets inside a stage for actions/items.
- It’s valid for `### Measures of success` to remain `TBD` until the rest of the doc stabilizes.
```
