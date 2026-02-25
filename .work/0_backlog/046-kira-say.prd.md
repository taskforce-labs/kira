---
id: 046
title: kira say
status: backlog
kind: prd
assigned: 
estimate: 0
created: 2026-02-25
due: 2026-02-25
tags: []
---

# kira say

Speak kira content aloud using pandoc (markdown → plain text) and a TTS tool (e.g. macOS `say`). This PRD captures the idea and re-evaluates options from scratch — no decision yet.

## The idea

From IDEAS.md: use pandoc, `say`, and kira to talk through questions/sections for quicker clarity — e.g. hear the current work item or the next question read out so you can listen while doing something else or in a discussion.

## Say as a flag on any console output

The flag could apply to **any command that has console output**. The behaviour would be:

- **`--say`** — Run the command as usual, **print** the output to the terminal **and** read it aloud (pandoc → TTS). You see and hear.
- **`--say-only`** — Run the command but **don’t print** the output; only **read it aloud**. You hear only.

So the “say” behaviour is just: take whatever would go to stdout, optionally suppress printing it, and pipe it through pandoc → say. That could sit on:
- **`kira current --body`** — speak (and optionally show) the current work item body.
- **`kira current --title`** — speak (and optionally show) the title.
- **A future `kira doc <doc-name>`** — output a doc from the docs folder (e.g. `.docs/agents/foo.md`); `--say` would print + read it, `--say-only` would just read it.
- **Any other command** that writes to the console — same pattern: same output, optionally spoken, optionally suppressed from the terminal.

So we’re not tying “say” to one command; we’re saying “any command with console output can get `--say` and `--say-only`, and the say pipeline just reads that output.”

## Particularly useful for

- **Slice** — e.g. `kira slice task current --say` or `kira slice task next --say-only`. Hear the current or next task read out so you can focus without staring at the terminal; useful when moving through a slice list or standing up.
- **Questions** — when we have a question command (e.g. `kira question next`), `--say` or `--say-only` would read the question aloud. Great for talking through questions to get clarity quickly (as in IDEAS.md) or in a discussion without having to read the screen.

## Constraints

- **Pandoc**: Converts markdown to plain text for TTS. Not bundled with macOS; user must install (e.g. `brew install pandoc`).
- **Say**: macOS built-in TTS. No install on Mac; not available on Linux/Windows.

## Scope for now (thin slice)

- **Mac-only**. We treat this as a macOS feature for the initial slice. If it’s useful, we can add support for other operating systems later (e.g. configurable TTS, or document alternatives like `espeak`).
- **Prerequisites we check**: When say is requested (e.g. `--say` or `--say-only`), we check:
  1. **Running on macOS** — if not, exit with a clear message that say is currently supported only on Mac.
  2. **Pandoc is available** — if not in PATH, exit with a clear message and how to install (e.g. `brew install pandoc`).
- **User installs prerequisites**. We do not bundle or install Pandoc; we only detect and error with a helpful message. `say` is already on macOS.
- Goal: a thin slice to test the feature; expand OS support and polish once we know it’s useful.

## Ways to look at it

### 1. Where does “say” attach?

- **Flag on any command with console output**  
  Add `--say` (print + speak) and `--say-only` (speak only) to any command that writes to stdout — e.g. `kira current --body`, `kira current --title`, a future `kira doc <doc-name>`, `kira latest`, `kira review`, etc. The say pipeline just reads whatever that command would output.
- **Dedicated command**  
  e.g. `kira say` that takes a subcommand or path and speaks that content (would need to define what it reads).
- **No kira change**  
  Rely on piping: `kira current --body | pandoc -f markdown -t plain | say`. Document the pattern; no new flags or commands.

### 2. How do we deliver it?

- **Script only**  
  A script (e.g. in repo or `.work/`) that runs kira, pandoc, and `say`. Documents deps; no CLI surface in kira.
- **First-class in kira**  
  `--say` (or `kira say`) implemented in Go: run pandoc + say when requested; detect missing pandoc and error with install hint.
- **Hybrid**  
  Script first to validate; later add `--say` on selected commands if it proves useful.

### 3. How we handle dependencies (for this slice)

- **Check and error**: When say is requested, check (1) we’re on macOS, (2) `pandoc` is in PATH. If either fails, exit with a clear message (and install hint for pandoc). User installs Pandoc themselves; we don’t bundle or install it.
- **Later**: If we expand to other OSes, we can add configurable TTS or document alternatives.

## What we’re not deciding yet

- Whether say is a **flag on commands** (e.g. `current`, `slice`) vs a **separate command** vs **piping only**.
- Whether it’s **script** vs **first-class** vs **hybrid**.

## Context (for when we do decide)

- **`kira current --body`** already outputs markdown to stdout; piping to `pandoc -f markdown -t plain | say` works today with no code changes.
- Any command that can output a “section” to stdout could support the same pipeline or a `--say` flag.
- For this slice: Mac-only; we check macOS and pandoc availability and expect the user to install Pandoc; clear error messages if not met.

## Requirements

(To be filled once we pick a direction.)

## Acceptance Criteria

(To be filled once we pick a direction.)

## Slices

(To be filled once we pick a direction.)

## Implementation Notes

- **Pipeline**: markdown → `pandoc -f markdown -t plain` → plain text → `say`. Content is kira-generated; exec patterns should follow existing kira rules (trusted config; no unsanitized user input in exec).
- **Prerequisite checks**: Before running the pipeline, detect macOS (e.g. `runtime.GOOS == "darwin"`) and that `pandoc` is executable (e.g. `exec.LookPath("pandoc")`). Exit with clear, actionable messages if not.
- **Optional later**: config for TTS command and/or pandoc path; support for other OSes.

## Release Notes

(To be filled once we pick a direction.)
