---
name: product-discovery
description: Guide through product discovery process to identify user needs, constraints, assumptions, risks, and commercial considerations. Use when starting a new feature or product, or when requirements are unclear.
disable-model-invocation: false
---

# Product Discovery

Guide the user through a structured product discovery process.

## When to Use

- Starting a new feature or product
- Requirements are unclear or incomplete
- Need to identify stakeholders and their needs
- Need to understand constraints and assumptions

## Instructions

1. **Read Project Configuration**
   - Check for `kira.yaml` in the project root
   - Use project-specific PRD template if configured, otherwise use default
   - Use project-specific artifact locations (e.g., `.work/` structure)
   - Adapt workflow based on project conventions

2. **Identify Stakeholders**
   - Ask the user: "Who are the key stakeholders for this product/feature?"
   - Document stakeholders and their roles

3. **Discover User Needs**
   - Prompt: "What problem are we solving? Who has this problem?"
   - Use the ask questions tool to gather detailed requirements
   - Document user stories and use cases

4. **Identify Constraints**
   - Technical constraints
   - Business constraints
   - Timeline constraints
   - Resource constraints

5. **Document Assumptions**
   - List all assumptions explicitly
   - Ask user to validate critical assumptions

6. **Assess Risks**
   - Technical risks
   - Business risks
   - Timeline risks

7. **Commercial Considerations**
   - Revenue model
   - Cost implications
   - Market considerations

8. **Create Artifacts**
   - Use project-specific template and location from `kira.yaml`
   - Create PRD following project conventions
   - Store in configured artifact location (default: `.work/0_backlog/`)

## Intervention Points

- When stakeholder identification is incomplete
- When user needs are ambiguous
- When critical assumptions need validation
- When risks require mitigation planning
- When commercial model needs definition

At each intervention point, present options, list assumptions, and guide the user to make informed decisions.
