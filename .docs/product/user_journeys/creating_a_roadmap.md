# User journey: creating a roadmap the Kira way

## Job to be done

- **As a** lead, product-minded engineer, or anyone who turns **direction into a queue others can pick up**—so teammates (and agents) are not guessing what “next” means  
- **I want** **one** readable plan (`PLAN.md`), **one** machine-readable roadmap (`ROADMAP.yml`) that Kira can validate, and **work items** elaborated enough to slice and execute **without re-negotiating scope every day**  
- **So that** parallel work is **safe**: everyone shares the same ordering and commitments instead of interpreting different subsets of the docs

**Scenario:** Strategy still lives in different places—product notes, journeys, architecture, half-formed tickets—and you do not yet have a single story of intent wired to real ids on the map.  
**Trigger:** You are **ready to sequence**—after direction is “good enough,” or when the team asks “what’s actually next?”

## Who this is for

Anyone who owns **ordering and commitment** from scattered intent to a lint-clean `ROADMAP.yml` and elaborated work.

## Touchpoints and artifacts

- **Prose plan:** `PLAN.md` (default under `.docs/`, or your `kira.yml` `docs_folder`).
- **Structured queue:** `ROADMAP.yml` beside `kira.yml`.
- **Validation:** `kira roadmap lint` (and optional `--check-adhoc`, `--check-deps`) from the repo root.
- **Funded work:** work items under `.work/`; `kira roadmap apply` for ad-hoc → real ids.
- **Replanning:** draft/promote flow (`kira roadmap draft`, `kira roadmap promote`) when the story shifts materially.

## Pain points and opportunities

- **Pain:** **Drift**—plan and YAML disagree, or ids go stale after renames. **Opportunity:** Lint early; fix YAML or work items until checks pass.
- **Pain:** **Placeholders forever**—titles on the map that never become real work. **Opportunity:** `kira roadmap apply` (with dry-run first) or deliberately keep ad-hoc lines you still own.
- **Pain:** **Big-bang replans** with no history. **Opportunity:** Draft → promote so the old tree is archived, not silently overwritten.

---

## Stage 1: Gather the signal

**Actions:** Skim durable sources—product direction, user journeys, anything that fixes sequencing (architecture, ADRs).  
**Produces:** Plain-language answers to *what wins look like this horizon* and *what we are deliberately not doing*.

Before you write anything new, you skim the durable sources. You are not copying them; you are stealing the conclusions—outcomes, workstreams, hard dependencies, explicit non-goals.

You are done with this stage when you can answer in plain language: *what wins look like this horizon* and *what we are deliberately not doing*.

---

## Stage 2: Commit intent to prose

**Actions:** Open `PLAN.md` in your docs folder (by default `.docs/PLAN.md`; your `kira.yml` may point `docs_folder` elsewhere).  
**Touchpoints / artifacts:** `PLAN.md` — narrative for humans and LLMs: goals, ordering, why one stream before another, risks. Keep file names and ticket IDs light; this is the narrative, not the database.

If someone joins mid-quarter, they read `PLAN.md` first and understand *why* the work is shaped the way it is.

---

## Stage 3: Turn the story into a tree

**Actions:** Create or edit `ROADMAP.yml` next to `kira.yml`.  
**Produces:** A `roadmap` list—entries with existing work item **`id`**, placeholders (**`title` only**), or **`group`** blocks for nested structure (phases, workstreams, releases).

You might draft this by hand, paste from a planning session, or ask an assistant to extract structure from `PLAN.md`. The built-in `kira roadmap plan-to-roadmap` command is still a stub, so you treat that conversion as **yours to own** until automation catches up.

When you need ordering assumptions checked later, you can attach `meta` to entries—common keys include `period`, `workstream`, `owner`, and `depends_on` on items that already have an `id`. For example, a placeholder might sit under the same group as the work item it must follow, with `depends_on` pointing at that id once you want lint to enforce the graph.

---

## Stage 4: Let Kira disagree with you early

**Actions:** From the repo root (where `kira.yml` lives), run `kira roadmap lint`. Use `kira roadmap lint --check-adhoc` for placeholders; `kira roadmap lint --check-deps` when you use `depends_on`.  
**System response:** Friction on renamed work items, typoed ids, or dependency cycles—**before** merge day.

You fix the YAML or the work items until lint is clean. **That moment** is when the roadmap becomes shared truth, not personal notes.

---

## Stage 5: Fund the placeholders

**Actions:** `kira roadmap apply --dry-run`, then `kira roadmap apply` when ready; filter with `meta` flags as needed (`--period`, `--workstream`, `--owner`, `--filter`).  
**Produces:** Backlog work items for selected ad-hoc entries; YAML lines rewritten with real ids.

Sometimes work appears outside the roadmap first (`kira new`, a spike from a meeting). You add that work item’s `id` into `ROADMAP.yml` so the next lint run still passes and the queue stays honest.

---

## Stage 6: Pull work off the map and into the loop

**Actions:** Move work items through your process folders when ready (`0_backlog` → `1_todo` → … per your setup). Elaborate until another person (or agent) could implement without inventing requirements; add `## Slices` and run `kira slice lint`.  
**Touchpoints / artifacts:** Work items under `.work/`, slice tasks, commits.

The roadmap tells you **what to open next**. It does not run your tests. Day to day you run the slice loop: see the current task, implement, mark done, commit the slice when it is complete.

You feel the handoff: **roadmap** = ordering and discovery; **work item + slices** = execution and commits.

---

## Stage 7: The plan shifts

**Actions:** For small corrections, edit `PLAN.md` and `ROADMAP.yml` together and lint again. For a bigger pivot, `kira roadmap draft <name>`, edit the draft file, then `kira roadmap promote <name>` (confirm when prompted, or `--yes` when appropriate).  
**Produces:** Updated canonical roadmap; prior tree archived when you promote.

---

## Success criteria

- Someone can read `PLAN.md` and understand intent without opening ten other files.
- `kira roadmap lint` passes for the checks your team relies on.
- Everything you intend to ship soon is either a real `id` on the map or a deliberate `title` you still own—not a forgotten line.
- Active work items are elaborated and sliced so `kira slice` can run without reinterpretation.

You have moved from scattered intent to a roadmap **Kira can enforce** and work **Kira can help execute**—without collapsing the plan into tickets too early, or leaving tickets disconnected from the plan.
