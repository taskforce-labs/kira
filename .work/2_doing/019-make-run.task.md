---
id: 019
title: make run
status: doing
kind: task
assigned:
estimate: 0
created: 2026-01-24
tags: []
---

# make run

run the kira cli command via go

## Details

Currently to test commands I run the following:
```bash
go run cmd/kira/main.go <command> <args>
```

This is a pain and I want to be able to run the commands via the Makefile.

I want to be able to run the commands via the Makefile like this:
```bash
make run <command> <args>
```

People should know about this command and how to use it.

