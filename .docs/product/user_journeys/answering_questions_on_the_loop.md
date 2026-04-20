# User journey: answering questions (every loop, often on the go)

## Who this is for

You want **agents and teammates to keep moving** while you stay authoritative on decisions—but you cannot sit at one machine answering pings one-by-one all day. This journey is how **questions surface at each band of work**, get answered **asynchronously or in small batches**, and land back in **durable artifacts** so nobody re-asks the same thing.

## How this ties to on the loop vs in the loop

**On the loop** is not only “big docs up front.” It is also **how work is staged**: agents progress in the background, collect **bounded** questions with clear options where possible, and you answer **on the go**—between meetings, from a phone, in a few minutes—without being the bottleneck on every line of code.

**In the loop** is when you are deep in implementation yourself; questions tend to be smaller and faster, and you may answer them inline without a formal queue. You still benefit when upstream questions were already resolved in writing.

Kira-shaped habits (work items, `## Questions`, ADRs, plan/roadmap updates) exist so answers **stick** and agents stay productive after a short human burst.

---

## Scene 1 — Same spine at every level

At each loop—direction, architecture, plan, roadmap, elaboration, slicing—the useful rhythm is:

1. **Agent or peer surfaces uncertainty** (missing rule, conflicting assumption, fork in design).
2. **Whoever owns the decision answers** with enough precision to update an artifact.
3. **The artifact is updated** so the next pass does not reopen the same thread.

The only thing that changes by level is **where the answer lives** and **how urgent** blocking is:

| Level | Questions tend to be about… | Answer often lands in… |
| --- | --- | --- |
| Direction / journeys | Who, why, scope, vocabulary | Journeys, product notes, glossary |
| Architecture / ADRs | Boundaries, tradeoffs, non-goals | ADRs, architecture sketches |
| Plan / roadmap | Sequencing, cut lines, dependencies | `PLAN.md`, `ROADMAP.yml` meta |
| Work item | Behaviour, acceptance, edge cases | Work item body, `## Questions` |
| Slices / tasks | Implementation detail, test shape | Task text, code comments sparingly |

You do not need a meeting per row. You need a **habit**: questions point to **checkable options** when possible (see [kira-clarifying-questions-format](../../../.agents/skills/kira-clarifying-questions-format/SKILL.md)), so an on-the-go answer can be “option B” plus one line of nuance.

---

## Scene 2 — Let agents run; collect questions instead of stalling

When you are on the loop, you prefer agents to **do safe parallel work**—research, draft structure, scaffold tests—while parking **decision debt** in one visible place per stream (e.g. a `## Questions` section in a work item, or a short list in a planning doc), rather than stopping cold on the first unknown.

**Productive background work** might include: drafting plan prose, proposing roadmap entries, listing ADR candidates, or spelling out two implementations with pros and cons—**clearly marked** until you answer.

**What pauses** should be explicit: “blocked until human chooses X vs Y,” not silent guessing.

---

## Scene 3 — Answer on the go

You are not at your desk for every question. The journey still works if:

- **Questions are batched** — One notification or one doc section with several items, each skimmable in under a minute.
- **Each item is self-contained** — Enough context that you do not need to reopen the whole repo to decide.
- **Defaults are stated** — “If no reply by Tuesday, we assume A for draft purposes only” (optional; use when your team agrees).

You answer from wherever you are: tick a box, reply in thread, edit the work item, or drop a one-line decision into an ADR stub. The important part is **the artifact updates the same day** when practical, so agents pick up the next morning without a chase.

---

## Scene 4 — Close the loop so agents do not spin

After you answer:

1. **Edit the owning artifact** (work item, ADR, plan, roadmap) so the decision is canonical.
2. **Remove or collapse the question** from the queue so status is honest.
3. **Signal unblock** however your team works (commit message, comment, or next agent prompt: “Questions section resolved—continue from …”).

If the answer **changes scope**, outer artifacts may need a touch-up (`PLAN.md`, roadmap, or a new ADR). That is still cheaper than letting two agents implement opposite assumptions for a week.

---

## Scene 5 — When this is “in the loop” instead

You are typing the change yourself. Many questions never leave your head—you decide and commit. **That is fine.** The overarching journey still applies when:

- You hit something that **should** be shared (security, API contract, data model), or
- You hand off to an agent again—then **writing the answer down once** pays for itself immediately.

---

## How you know this journey is working

- Agents produce **visible question queues** instead of silent wrong guesses.
- You can clear a batch of decisions in **short bursts**, not only in long focused sessions.
- The same question does not return next sprint because **the answer lived in the right doc or work item**.

That is **on the loop** collaboration: high leverage from humans, high throughput from agents, without requiring you to be synchronously “at the computer for each question.”
