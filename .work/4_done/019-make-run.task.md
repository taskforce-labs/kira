---
id: 019
title: make run
status: done
kind: task
assigned:
estimate: 0
created: 2026-01-24
tags: []
---

# make run

run the kira cli command via go for local development

## Details

Initially, the goal for this task was to add a `make run` target that would let us execute kira commands like this:

```bash
make run <command> <args>
```

However, overloading `make` targets to act as CLI arguments (e.g. `make run lint`, `make run check`) created confusing interactions with existing Make targets such as `lint` and `check`, and required conditional logic in the Makefile to avoid target conflicts and misleading errors.

During implementation it became clear that a simpler and more maintainable approach is to run kira directly from source via a small helper script at the repo root:

```bash
./kdev <command> <args>
```

This script wraps:

```bash
go run cmd/kira/main.go <command> <args>
```

so contributors can test kira commands locally without building or installing the binary, without adding extra complexity or hidden behaviour to the Makefile.

We also document `./kdev` in `CONTRIBUTING.md` so people know how to use it during development.

