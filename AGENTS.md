# Agents

## Checking your work
Check your work by running `make check` after making changes.
This will run the following checks:
- Linting
- Security
- Testing (unit tests and e2e tests)
- Code coverage

If make check passes, run the following to verify the e2e tests pass:
`bash kira_e2e_tests.sh` will run the e2e tests and are worth running

## Golang

See [Go Secure Coding Practices](docs/security/golang-secure-coding.md) for comprehensive security guidelines covering:
- File path validation patterns
- File permissions
- Command execution security
- When to use `#nosec` comments

DON'T RELAX THESE RULES FOR TEST FILES DO NOT CHANGE .golangci.yml TO RELAX RULES UNDER ANY CIRCUMSTANCES UNLESS I TELL YOU TO DO SO EXPLICITLY.