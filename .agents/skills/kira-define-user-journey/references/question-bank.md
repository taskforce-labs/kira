## Question bank (pick what fits; skip irrelevant)

Use these as menus, not a mandatory checklist. When you surface them to stakeholders, wrap them in `## Questions` / `### N.` (and include `#### Options` only where real choices exist).

### Framing

- Draft the **Job to be done**: **As a** … **I want** … **So that** … (iterate until each clause is specific).
- Who is the primary actor, in one sentence? Link to a persona doc under `.docs/product/` when one exists.
- What starts this journey (trigger, intent, inbound event)?
- What “done” looks like: observable end state or artifact (success criteria)?
- How often does this run; how time-sensitive is it?
- What adjacent journeys or docs should this link to (plan, roadmap, product, ADRs)?
- Confirm this file stays one scenario—split if the actor–goal–situation combo branches into a different story.

### Job to be done — value and evidence

**Purpose:** Stress-test whether the journey is a good bet early. Use the same headings in the finished journey doc (under `## Job to be done`) so readers see why next to what.

#### Challenge the bet

- What problem/gap exists today if we do not improve this journey—what do actors actually do instead (workarounds, tools, avoidance)?
- Why now? What changes if we wait a quarter?
- If we shipped only a thin slice, what is the smallest improvement that would still make “So that” believable?
- What would falsify the “So that” claim—what would we observe if we were wrong about value?

#### Supporting evidence

- What facts already support the need (analytics, volumes, costs, incidents, support themes, prior research, legal/compliance drivers)?
- What is still assumption—and what would we do to confirm or reject it before over-investing?
- Any links (dashboards, interview notes, tickets, briefs) worth citing in the doc?

### Flow and stages

- Preconditions (access, data, branch, tool, feature flag)?
- Within each stage: user action → system response → what is recorded?
- Touchpoints (product-facing): where does the actor meet the product or organization (surfaces, channels)?
- Artifacts (internal/dev): repo paths, work items, commands, dashboards—often the meaningful “surfaces” for this audience.
- Where are human decisions vs automated actions?
- Split the story into stages (phases over time). Optional per stage: Actions · Touchpoints/artifacts · System/org response · Produces/records.

### Pain points and opportunities

- Where does the actor stall, worry, or waste time (pain)?
- What improvement would make the next pass easier (opportunity)?

### Edges and boundaries

- Top failure modes and how the actor recovers or escalates?
- Non-goals / out of scope (what people mistakenly think is included)?
- Permissions, safety, or compliance constraints?
- Metrics or success signals tied to success criteria (optional but LLM-useful).

### Measures of success (complete last)

**Why last:** you need stable stages/touchpoints and a first pass at success criteria before you can name credible signals.

- What signals will we watch (leading + lagging) — completion, time-to-done, drop-offs, quality, satisfaction, operational load?
- How do those signals map to `## Success criteria` (align, don’t contradict)?
- Cadence: when will we review reality vs this journey, and who owns that review?

### LLM- and agent-oriented (optional)

- Vocabulary table: domain term → meaning in this journey.
- Acceptance-style checks: bullets a reviewer/agent could verify.
- Optional appendix: a small Gherkin `Feature` / `Scenario` set for the main path + 1–2 variants.
