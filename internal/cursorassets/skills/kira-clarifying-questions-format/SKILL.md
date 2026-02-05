---
name: clarifying-questions-format
description: Present clarifying questions with selectable options. Use when any skill or command needs to ask stakeholders to choose among options—format questions so answers are easy to give (check a box) and assumptions are visible (suggest an option and why).
disable-model-invocation: false
---

# Clarifying questions format

When you need to present clarifying questions with selectable options, use this format. Any skill that asks stakeholders to choose among options should follow it.

## When to use

- When you need to clarify scope, rules, flows, or decisions before proceeding
- When multiple valid options exist and the stakeholder must choose
- When you want to surface your assumptions by recommending an option and explaining why

## Structure

Put all clarifying questions under a **## Questions** section. For each thing to clarify:

1. **Heading**: `### N. [Short title of what we need to clarify]` (N = 1, 2, 3, …)
2. **Context**: One or more sentences describing the clarification and why it matters.
3. **Options subheading**: `#### Options`
4. **Options**: Each option on its own line with a checkbox: `- [ ] Option N.X: short description` (X = A, B, C, …). Optionally add details, examples, or diagrams under an option before the next option.

After listing options for a question, **suggest an option**: state which option you recommend and why in one or two sentences. This exposes assumptions; the stakeholder can confirm or check a different option.

## Example

```markdown
## Questions

### 1. Things we need to clarify
Details of the work item that we need to clarify.

#### Options
- [ ] Option 1.A: short description

Details about option 1.A that we need to clarify. Maybe some diagrams or examples.

- [ ] Option 1.B: short description

Details about option 1.B that we need to clarify. Maybe some diagrams or examples.

**Suggested:** Option 1.A because [one sentence]. [Optional: assumption this exposes.]
```

## After answers

- When the stakeholder selects an option, mark it in the doc (e.g. `- [x] Option 1.A`) and record any new assumption.
- If your recommended option was not chosen, note that and update assumptions in the work item.
