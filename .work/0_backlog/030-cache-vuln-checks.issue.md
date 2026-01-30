---
id: 030
title: cache vuln checks
status: backlog
kind: issue
assigned:
created: 2026-01-30
tags: []
---

# cache vuln checks

When running make check without internet access, the cache vuln checks fail because they need to download the vulnerability database from the internet.


$ make check
0 issues.
Running govulncheck vulnerability scanner...
govulncheck: fetching vulnerabilities: Get "https://vuln.go.dev/index/modules.json.gz": net/http: TLS handshake timeout
make: *** [security] Error 1

## Solution

Cache the Go vulnerability database locally and point `govulncheck` at it during `make check`.

Proposed approach:
- Add a dedicated cache location (per platform) and document it:
  - Linux: `~/.cache/kira/govulndb`
  - macOS: `~/Library/Caches/kira/govulndb`
  - Windows: `%LOCALAPPDATA%\Kira\cache\govulndb`
- Create a `make vuln-db-update` (or similar) target that:
  - Requires network access.
  - Refreshes the local DB cache.
  - Exits with a clear message if the network is unavailable.
- Update the `security`/`make check` flow to:
  - Set `GOVULNDB` to the local cache path before invoking `govulncheck`.
  - If the cache is missing or stale, fail with a friendly message that instructs
    the user to run `make vuln-db-update` while online.
  - Avoid any implicit network calls during the offline run.

Notes:
- The cache should not be committed to git.
- The cache path should be configurable via an env var (default to a sensible path).
- If desired, add a CI job to refresh/upload the cache artifact so offline developers
  can pull it without needing direct access to the internet.

