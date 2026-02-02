---
name: roadmap-planning
description: Break down features into work items and sequence them into a roadmap optimized for parallel development. Minimize rework and merge conflicts; use dependency analysis. Use when planning a release or ordering work.
disable-model-invocation: false
---

# Roadmap Planning

Guide the user through roadmap and work-item sequencing.

## When to Use

- Planning a release or iteration
- Ordering work items for parallel development
- Reducing rework and merge conflicts
- Analyzing dependencies between work items

## Instructions

1. **Read Project Configuration**
   - Check `kira.yaml` for work folder and status folders
   - Use `.work/` structure (backlog, todo, doing, review, done)

2. **Break Down Features**
   - Decompose features into work items (PRDs, tasks, issues)
   - Ensure each work item is independently shippable where possible
   - Use `kira new` and move to appropriate status folders

3. **Dependency Analysis**
   - Identify dependencies between work items
   - Map blocking and optional dependencies
   - Document in work item content or a lightweight map

4. **Sequence for Parallel Development**
   - Order work to maximize parallel streams
   - Minimize merge conflicts (e.g. avoid parallel edits to same areas)
   - Surface critical path and milestones

5. **Create Artifacts**
   - Roadmap or ordered list of work items
   - Dependency notes in work items or a central doc
   - Use `kira move` and status folders to reflect order

## Intervention Points

- When dependencies are unclear
- When sequencing trade-offs need stakeholder input
- When scope or priority changes

At each intervention point, present options and guide the user to make informed decisions.
