---
name: kira-analyse-work-item-risk
description: "Scan @risk / @final_review annotations, match to work item scope (planned or diff), set work item risk and final_review in frontmatter. Pre- and post-implementation."
disable-model-invocation: false
---
# Kira: Analyse Work Item Risk

**When:** planning (planned touch set) or after implementation (`git diff`).
**Goal:** set work item `risk` and `final_review` from annotations in source and, when used, code-owner files.

## Risk vs `final_review` (do not conflate)

- **Risk** — impact if wrong (`low` | `medium` | `high` | `critical`).
- **`final_review`** — who may accept: `human` (human owns accept; agents may assist) or `agent` (automated accept path). Normalize `agents` → `agent`.
- High risk does **not** imply `final_review: human`.

## Annotations (grep: `@risk`, `@final_review`)

- `@risk <level>[: optional reason]`
- `@final_review <human|agent> [<target>]` — one line. Optional target: `@handle`, literal **`CODEOWNERS`** (resolve from repo file), or bot/service id for `agent`. Keep targets **in code only**; frontmatter is just `final_review: human` or `agent`.

### Where to look for markers

- **In source** — on or next to the code that carries the risk or acceptance boundary (line- or file-level signals).
- **Code owner files** — for **directory- or tree-wide** expectations (everything under a path), prefer **`CODEOWNERS`** (or the repo’s equivalent owner/review rules) so policy is explicit by path instead of inferring from a single file comment. Use inline `@risk` / `@final_review` when the signal is about specific logic; use owner files when the signal is “this area as a whole.”

## Workflow

1. **Index** markers: file, line, risk level, `final_review` who + optional target (grep source; include **CODEOWNERS** and other owner/review rule files if the repo annotates them for path-wide policy).
2. **Scope** — pre: planned files; post: changed files/lines from diff.
3. **Match** — same line (best) → same block/function → same file (weaker).
4. **Aggregate** — work item `risk` = highest matched level; default `low` if none. `final_review` from matched lines; if both human and agent apply, **human wins** or escalate; if none match, omit or leave unchanged (org defaults).
5. **Frontmatter** — set `risk` and `final_review` only (no separate owners field).

## Output

State risky areas and why (if annotated). Cite file, line, and inline target when reporting. Update frontmatter; call out risk increase post-implementation or ambiguous `final_review`.

## Principles

Lightweight, grep-able annotations; line-level over directory guesses unless directory policy lives in **CODEOWNERS** (or equivalent); signal over completeness.
