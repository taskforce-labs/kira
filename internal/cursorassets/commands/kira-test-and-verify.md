# Test and Verify

## Overview
Run tests and verify implementation: run test suite, lint, and checks; fix failures and report results.

## Steps

1. **Run Tests**
   - Run project test suite (e.g. `make test` or `go test ./...`)
   - Capture failures and coverage if configured

2. **Run Lint and Checks**
   - Run linter and security checks (e.g. `make check`)
   - Fix reported issues where appropriate

3. **Verify Acceptance Criteria**
   - Map test results to work item acceptance criteria
   - Identify unmet criteria or regressions

4. **Report and Fix**
   - Report pass/fail and any failures
   - Apply fixes and re-run until checks pass (or escalate to user)

## Output

Test and check results; fixes applied until verification passes or user intervention is needed.
