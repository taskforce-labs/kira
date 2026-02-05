---
name: technical-discovery
description: Discover and articulate technical constraints, dependencies, and requirements. Use target architecture design patterns and ADR (Architectural Decision Records) creation and structure. Use when making technical or architectural decisions.
disable-model-invocation: false
---

# Technical Discovery

Guide the user through technical discovery and architecture decisions.

## When to Use

- Making technical or architectural decisions
- Documenting constraints and dependencies
- Creating or updating ADRs
- Designing target architecture

## Instructions

1. **Read Project Configuration**
   - Check `kira.yaml` for architecture doc path and conventions
   - Use project artifact locations

2. **Identify Technical Constraints**
   - Platform and technology constraints
   - Integration and dependency constraints
   - Non-functional requirements

3. **Target Architecture**
   - Apply design patterns appropriate to the domain
   - Document components and boundaries
   - Identify integration points

4. **Architecture Decision Records**
   - Use ADR structure (context, decision, consequences)
   - Store ADRs in project-configured location
   - Reference ADRs in work items where relevant

5. **Create Artifacts**
   - ADRs for significant decisions
   - Architecture overview or context diagram
   - Store in configured artifact location

## Intervention Points

- When technical constraints are unclear
- When trade-offs need stakeholder input
- When ADR consequences need validation

At each intervention point, present options and guide the user to make informed decisions. When presenting clarifying questions with selectable options, follow the clarifying-questions-format skill.
