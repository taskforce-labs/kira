---
id: 050
title: questions section groups
status: backlog
kind: prd
assigned:
created: 2026-04-14
tags: [cli, documentation, agents]
---

# questions section groups

Extend **`kira questions`** so a single markdown file can contain **multiple** Questions sections, each optionally labeled with a **theme** after `## Questions`. **`## Questions: <theme>`** (colon) is the **preferred** author and agent form; the parser should also accept a **themed** section introduced by an **em dash** or an **ASCII hyphen** so common markdown habits do not break discovery.

## Context

Today `ParseQuestionsFromMarkdown` (`internal/commands/questions_parse.go`) only recognizes a level-2 heading **exactly** equal to `## Questions`, and treats everything until the next `## ` as one section. Variants such as `## Questions: Framing` are ignored. That makes thematic bands and multiple rounds in one file awkward.

**Direction for this PRD:** support **several** Questions blocks in one file. Themed headings are **`## Questions: <theme>`** (preferred), with **`## Questions — <theme>`** and **`## Questions - <theme>`** accepted for parsing only. Plain **`## Questions`** (no delimiter, no trailing text) remains valid for untitled bands and must behave as today.

The **kira-define-user-journey** skill references this work item; agent-facing docs should align once behavior ships.

## Acceptance Criteria

- [ ] Parser and `kira questions` treat **`## Questions`** and themed variants as starting a Questions section: **`## Questions: <theme>`** (preferred), **`## Questions — <theme>`**, and **`## Questions - <theme>`** (normalize to a single internal “theme” string; trim whitespace; empty `<theme>` equivalent to untitled). Documentation and **kira-clarifying-questions-format** describe **only** plain **`## Questions`** and **`## Questions: <theme>`**—do not encourage em dash or hyphen in agent-authored text.
- [ ] A file may contain **multiple** such sections; each section runs until the next `## ` heading at level 2. Questions under each section are still **`### N. Title`** with numeric `N`; numbering may **restart per section** or be **file-unique**—the chosen rule is documented in command help and tests (pick one behavior and stick to it).
- [ ] **Backward compatibility:** existing documents with a single plain `## Questions` section and `### N.` items behave as they do today (same unanswered/answered rules for `#### Options`).
- [ ] Default human-readable output makes it obvious **which file, which section theme (if any), and which question** each line refers to (exact format documented in help).
- [ ] `--output json` (if present) includes enough structure to distinguish sections (e.g. theme or section index) without breaking existing consumers; any JSON shape change is documented and covered by tests.
- [ ] Unit tests in `questions_parse_test.go` (and command tests as needed) cover: multiple sections; `## Questions: Theme`; `## Questions — Theme` and `## Questions - Theme`; plain `## Questions` mixed in one file; at least one edge case (e.g. `## Questions:` with nothing after colon, or two adjacent question sections).
- [ ] Command long help and PRD **039**-aligned docs updated so authors do not rely on obsolete rules.
- [ ] **kira-define-user-journey** and **kira-clarifying-questions-format** updated: multiple sections; **colon** themed heading documented as the only variant in the clarifying skill; journey skill aligned (may note parser tolerance for `—` / `-` without recommending them).
- [ ] `make check` passes.
