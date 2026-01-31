---
id: 012
title: submit for review
status: backlog
kind: prd
assigned:
created: 2026-01-19
tags: [github, review, cli]
---

# submit for review

A command that moves work items to review status on the current branch, optionally updates the status on the main trunk branch to reflect the review state (treating trunk as source of truth), creates pull requests on GitHub or changes the status of a PR on GitHub from draft to ready for review.

## Context

In the Kira workflow, work items progress through different statuses: backlog → todo → doing → review → done. Currently, moving to review status requires manual `kira move` commands and separate GitHub PR creation. This creates friction in the development workflow, especially when working with teams or when agents need to coordinate code reviews.

The `kira review` command (note: renamed from "submit for review" to avoid confusion with existing "review" status) will streamline this process by:

Workflow:
1. Moving the work item to review status on the current feature branch like `kira move 001 review`
2. Rebasing the current branch onto the updated trunk branch like `kira latest`
3. Push the current branch to the remote repository (if not already pushed)
4. Automatically creating PR on GitHub with proper branch naming and descriptions OR change the status of a PR on GitHub from draft to ready for review
5. Assigning reviewers specified via `--reviewer` flag (supports user numbers from `kira user` command or email addresses)
