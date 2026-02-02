# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- **Draft pull requests for `kira start`:** When the remote is GitHub and `KIRA_GITHUB_TOKEN` is set, `kira start` pushes the new branch and creates a draft pull request. PR title is `{id}: {title}`; body is taken from the work item (content after YAML front matter).
- **`--no-draft-pr` flag:** Skip pushing the branch and creating a draft PR when running `kira start`.
- **Config:** `workspace.draft_pr` (default: true), `workspace.git_platform`, `workspace.git_base_url`, and optional `projects[].draft_pr` / `projects[].git_base_url` in `kira.yml` for polyrepo and GitHub Enterprise.
- **Work item `repos` front matter:** Work items can list `repos` in YAML front matter; when set, draft PRs are created only for those projects (by name) in polyrepo setups.
- **Error handling:** Clear error when draft PR would be created but `KIRA_GITHUB_TOKEN` is unset (suggests setting the token or using `--no-draft-pr`). Push failures stop execution with a clear message; PR creation failures are logged and do not fail the start command.
- **Dry run:** Dry-run output now shows whether a draft PR would be created or skipped (e.g. "Would push branch and create draft PR" or "Would skip draft PR (--no-draft-pr)").
- **Check commands:** New `checks` config and `kira check` command to define and run project check commands (e.g. lint, test, security) from a single entry point. Use `kira check` to run all configured checks in order (exits on first failure); use `kira check --list` to list them. Supports agents and scripts that need "run the project's checks."
