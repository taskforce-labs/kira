---
name: work-item-elaboration
description: "Articulate a work item’s intent and behaviour: value, outcomes, business rules, and flows. Operates in the behavioural space (what value is delivered and what correct behaviour looks like), not solution or implementation design. Use existing technical context only to inform accuracy, not to decide solutions. Intended as a prerequisite for slicing and delivery planning."
disable-model-invocation: false
---

# Work Item Elaboration

Guide the user through behavioural elaboration of a work item: clarifying value and outcomes, defining business rules and constraints, and describing flows and scenarios. This skill explicitly avoids solutioning and in-depth technical design. Its purpose is to remove ambiguity so the work item can be sliced and delivered safely.

## When to Use

- When a work item is likely to be built but its value, rules, or behaviour are unclear
- After discovery, before technical design or slicing
- As a prerequisite for story slicing or delivery planning
- When acceptance criteria or examples are missing or ambiguous

## Instructions

### 1. Read Project Configuration
- Check `kira.yaml` for work folder locations and templates
- Use the `.work/` structure and `kira slice` if the project uses slices

### 2. Read Work Item and Context
- Read the work item (title, description) and any linked docs
- Review existing technical context only to ensure accuracy of value, rules, and flows
- Do not introduce new technical or solution decisions

### 3. Establish Value and Outcomes
- Clarify the user and business value this work item delivers
- Identify who benefits and how
- Define success in plain language (what changes if this ships)
- Capture explicit non-goals or exclusions

### 4. Articulate Business Rules and Constraints
- Capture rules, policies, and constraints that govern behaviour
- Identify edge cases and exceptional conditions
- Express rules in business language, not technical terms
- Avoid solution or implementation decisions

### 5. Describe Flows and Scenarios
- Describe user or system flows step by step
- Highlight decision points and alternate paths
- Use examples or scenarios where helpful
- Surface ambiguities or unresolved decisions

### 6. Record Open Questions and Assumptions
- List unresolved decisions or assumptions
- **Articulate assumptions in the work item**: When you assume something to proceed (e.g. scope, rule, flow), record it explicitly in the work item so it can be confirmed or corrected
- Note where technical choices may influence behaviour
- Flag items that require follow-up before slicing or delivery

## Create Artifacts

- Updated work item containing:
  - Value and outcomes
  - Business rules and constraints
  - Flows and scenarios
  - Non-goals and exclusions
  - **Questions** (under `## Questions`, with options listed as checkboxes e.g. `- [ ] Option 2.A: …`)
  - Open questions and assumptions
  - **Assumptions made** (any assumptions you made during elaboration, so they can be validated)
- Use project templates and standard locations

## Intervention Points

- When value or outcomes are unclear from the title or description
- When business rules conflict or are ambiguous
- When flows reveal missing decisions or assumptions
- When flows cannot be described without choosing a technical solution

At each intervention point, present options and guide the user to make informed decisions, or explicitly record the open question for follow-up.

## Questions

Put all clarifying questions in a **## Questions** section in the work item. When presenting clarifying questions with selectable options, follow the **clarifying-questions-format** skill (read that skill for the full structure: ## Questions, ### N. [Title], #### Options, checkboxes, Suggested).

Before handing off to technical design or slicing, ensure these areas have been addressed (or explicitly recorded as open): value and scope, rules and edge cases, flows and scenarios, decisions and assumptions, dependencies and touchpoints. When options are chosen, document the result and any new assumptions in the work item.