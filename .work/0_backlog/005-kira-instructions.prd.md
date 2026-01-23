---
id: 005
title: Kira instructions
status: backlog
kind: task
assigned:
estimate: 0
created: 2025-11-18
tags: []
---

# Kira Instructions

## Context

Humans andAgents (LLMs/clankers) need clear instructions on how to use kira to manage work items effectively. Documentation should be organized across multiple files with distinct purposes:
- **README.md**: General project documentation for both humans and agents
- **AGENTS.md**: Agent-specific quick reference and decision-making guidance
- **docs/agents/using-kira.md**: Comprehensive agent-focused detailed documentation

Each file serves a different purpose and audience, ensuring information is accessible and appropriately scoped.

## Details

### File Structure and Purpose

#### README.md (Root directory - already exists)
- **Purpose**: General project documentation for both humans (meatbags) and agents
- **Audience**: Anyone working with the project
- **Content**: Installation, quick start, command reference, configuration overview
- **Scope**: High-level overview, installation, basic usage
- **Tone**: Professional, user-friendly, comprehensive but not exhaustive

#### AGENTS.md (Root directory - already exists)
- **Purpose**: Agent-specific quick reference and decision-making guidance
- **Audience**: LLMs/clankers working in this repository
- **Content**: When to use kira, quick workflow reminders, links to detailed docs
- **Scope**: Minimal, actionable guidance (~10-15 lines)
- **Tone**: Direct, action-oriented, decision-focused

#### docs/agents/using-kira.md (To be created)
- **Purpose**: Comprehensive agent-focused detailed documentation
- **Audience**: Agents needing deep understanding of kira workflows
- **Content**: Detailed command documentation, examples, troubleshooting, best practices
- **Scope**: Complete reference guide for agent workflows
- **Tone**: Technical, detailed, example-rich

### Documentation Delineation

**README.md** should contain:
- Installation instructions
- Quick start guide
- Command reference (all commands with basic usage)
- Configuration overview (kira.yml structure)
- Work item types overview
- Status folder overview
- General best practices
- Links to AGENTS.md and docs/agents/using-kira.md for agent-specific guidance

**AGENTS.md** should contain:
- When agents should interact with kira work items
- Quick workflow reminders (create → move → save)
- Decision guidance (when to create vs update, when to move items)
- Link to detailed documentation
- When to review detailed docs

**docs/agents/using-kira.md** should contain:
- Deep dive into agent workflows
- Detailed command examples with agent-focused use cases
- Troubleshooting guide
- Advanced patterns and best practices
- Configuration interpretation guide
- Work item lifecycle details

### README.md Content Requirements

The README.md file should include or update a section about using kira in this repository. This section should:

#### Overview Section
- Brief mention that this repository uses kira for work item management
- Link to AGENTS.md for agent-specific guidance
- Link to `docs/agents/using-kira.md` for detailed agent documentation
- Note that kira configuration is in `kira.yml`

#### Quick Start Section (if not already present)
- How to initialize kira in this repository: `kira init`
- Basic workflow: ideas → create → move → save
- Reference to check `kira.yml` for configured work item types and statuses

#### Commands Reference Section
- List all kira commands with brief descriptions
- Reference `kira.yml` for available template types and status values
- Link to full command documentation in README.md or dedicated docs
- Note that agents should see AGENTS.md for workflow guidance

#### Configuration Section
- Overview of `kira.yml` structure
- Key configuration sections: templates, status_folders, validation
- Note that configuration drives available work item types and statuses
- Link to full configuration documentation

### AGENTS.md Content Requirements

The AGENTS.md file should include a new section titled "Managing Work Items with Kira" that provides:

#### When to Use Kira
- Clear guidance on when agents should interact with kira work items
- When to create new work items vs updating existing ones
- When to move work items through statuses
- When to review the detailed documentation

#### Quick Reference
- Brief overview of kira (1-2 sentences)
- Link to detailed documentation: `docs/agents/using-kira.md`
- Quick workflow reminder: create → move → save
- When to run `kira lint` before committing

#### When to Review Detailed Documentation
- Before creating your first work item
- When encountering errors or validation issues
- When unsure which template type to use
- When needing detailed command syntax or examples

### Dedicated Documentation File Requirements

The `docs/agents/using-kira.md` file should contain comprehensive documentation covering:

#### 1. Overview
- Brief explanation of what kira is (git-based, plaintext productivity tool)
- Why agents should use kira (transparency, git integration, simple workflow)
- Key concepts: work items, status folders, templates

#### 2. Quick Start for Agents
- How to check if kira is initialized (`ls .work/`)
- Basic workflow: create → move → save
- Example of a typical agent workflow

#### 3. Common Commands for Agents
Document the following commands with agent-focused examples:

- **`kira new`**: Creating new work items
  - Work item types are defined in `kira.yml` under `templates:` section
  - Reference available template types from the config (e.g., prd, issue, spike, task)
  - When to use each template type based on the configured templates
  - How to use `--input` flag for non-interactive creation
  - Examples for creating items in different statuses
  - Note: Available statuses are defined in `kira.yml` under `validation.status_values`

- **`kira move`**: Moving work items through workflow
  - Status progression should reference `kira.yml` `status_folders:` mapping
  - Show typical flow using configured status folders (e.g., backlog → todo → doing → review → done)
  - When to move items (e.g., move to doing when starting work)
  - Using `--commit` flag for automatic commits
  - Note: Valid status values are defined in `kira.yml` under `validation.status_values`

- **`kira save`**: Committing changes
  - When to use (after creating, moving, or updating work items)
  - How to provide meaningful commit messages
  - Validation behavior (fails on errors)

- **`kira idea`**: Quick idea capture
  - When to use vs creating a formal work item
  - Format of ideas in IDEAS.md

- **`kira lint`**: Validating work items
  - When to run (before committing)
  - What it checks

- **`kira doctor`**: Fixing common issues
  - When to use (duplicate IDs, corrupted files)

#### 4. Work Item Lifecycle
- Status folder meanings should be sourced from `kira.yml`:
  - Read `status_folders:` section to understand status-to-folder mapping
  - Document the meaning of each status based on the configured folders
  - Example mappings (but note these are configurable):
    - `backlog` → Ideas being shaped
    - `todo` → Ready to work on
    - `doing` → Currently in progress
    - `review` → Ready for review/PR
    - `done` → Completed work
    - `archived` → Archived items
  - Emphasize that agents should check `kira.yml` for the actual configuration
- Typical flow for agents working on tasks
- When to update work items vs creating new ones

#### 5. Best Practices for Agents
- Always run `kira lint` before `kira save`
- Use meaningful commit messages with `kira save`
- Update work items as you progress (move through statuses)
- Use `kira idea` for quick thoughts, formal work items for actionable tasks
- Keep work item descriptions clear and detailed
- Update the `updated` timestamp is handled automatically by `kira save`

#### 6. Work Item Format
- YAML front matter structure
- Required fields are defined in `kira.yml` under `validation.required_fields`
- Optional fields (assigned, estimate, updated, due, tags, etc.)
- File naming convention: `{id}-{kebab-case-name}.{type}.md`
  - `{type}` should match a key from `kira.yml` `templates:` section
- ID format is defined in `kira.yml` under `validation.id_format` (typically `^\d{3}$`)
- How to read existing work items
- How to update work items (edit markdown directly)

#### 7. Integration with Git
- How kira integrates with git
- `kira save` only commits `.work/` changes
- External changes detection
- Using git history to track work item changes

#### 8. Examples and Use Cases
Provide concrete examples:
- Creating a task for a bug fix
- Moving a PRD from todo to doing when starting work
- Updating a work item with implementation notes
- Moving to review when PR is ready
- Moving to done when complete

#### 9. Troubleshooting
- What to do if `kira lint` fails
- How to use `kira doctor` to fix issues
- Common errors and solutions

### Technical Requirements
- AGENTS.md section should be concise (aim for ~10-15 lines)
- Use clear, actionable language
- Include a prominent link to the detailed documentation
- The dedicated doc file should be well-formatted markdown
- Include code examples with proper syntax highlighting
- Use clear headings and subheadings
- Make it scannable for agents (use lists, code blocks, examples)
- Reference existing kira documentation where appropriate
- Ensure all examples are accurate and tested

### Configuration-Driven Documentation
- **Work item types**: Should reference `kira.yml` `templates:` section rather than hardcoding types
- **Status folders**: Should reference `kira.yml` `status_folders:` mapping rather than hardcoding folder names
- **Status values**: Should reference `kira.yml` `validation.status_values:` rather than hardcoding valid statuses
- **Required fields**: Should reference `kira.yml` `validation.required_fields:` rather than hardcoding
- **ID format**: Should reference `kira.yml` `validation.id_format:` rather than hardcoding
- Documentation should instruct agents to check `kira.yml` for the actual configuration
- Provide default examples but emphasize configurability
- Include guidance on how to read and interpret `kira.yml` configuration

### File Organization
- **README.md**: General project documentation, installation, command reference
- **AGENTS.md**: Minimal, decision-making guidance for agents
- **docs/agents/using-kira.md**: Comprehensive reference documentation for agents
- Clear separation of concerns:
  - README.md: General usage for all users
  - AGENTS.md: Quick agent decision-making
  - docs/agents/using-kira.md: Deep agent reference

### Validation
- Verify README.md exists in the root directory
- Verify README.md includes section about kira usage in this repository
- Verify README.md links to AGENTS.md and docs/agents/using-kira.md
- Verify AGENTS.md exists in the root directory
- Verify new section in AGENTS.md is concise and actionable
- Verify `docs/agents/using-kira.md` file is created
- Verify links between all three files work correctly
- Verify all command examples are correct
- Verify links and references are valid
- Verify documentation references `kira.yml` configuration rather than hardcoding values
- Verify work item types match those in `kira.yml` `templates:` section
- Verify status folders match those in `kira.yml` `status_folders:` mapping
- Verify status values match those in `kira.yml` `validation.status_values:`
- Verify clear delineation between README.md (general), AGENTS.md (quick reference), and docs/agents/using-kira.md (detailed)

## Acceptance Criteria

### README.md
- [ ] README.md includes section about using kira in this repository
- [ ] Section includes links to AGENTS.md and docs/agents/using-kira.md
- [ ] Quick start guidance references `kira.yml` configuration
- [ ] Commands reference section lists all kira commands
- [ ] Configuration overview explains `kira.yml` structure
- [ ] Content is appropriate for both human and agent audiences

### AGENTS.md
- [ ] AGENTS.md file exists in the root directory
- [ ] New section "Managing Work Items with Kira" is added to AGENTS.md
- [ ] Section in AGENTS.md is concise (~10-15 lines) and provides clear guidance on when to use kira
- [ ] Section includes link to `docs/agents/using-kira.md`
- [ ] Content is agent-focused and decision-oriented

### docs/agents/using-kira.md
- [ ] `docs/agents/using-kira.md` file is created with comprehensive documentation
- [ ] Dedicated doc file covers all required topics (overview, commands, lifecycle, best practices, examples)
- [ ] Content is detailed and agent-focused

### Configuration-Driven Documentation
- [ ] Documentation references `kira.yml` configuration for work item types, status folders, and status values
- [ ] Work item types are sourced from `kira.yml` `templates:` section
- [ ] Status folders are sourced from `kira.yml` `status_folders:` mapping
- [ ] Status values are sourced from `kira.yml` `validation.status_values:`
- [ ] Required fields reference `kira.yml` `validation.required_fields:`
- [ ] ID format references `kira.yml` `validation.id_format:`
- [ ] Documentation includes guidance on reading `kira.yml` configuration

### Quality and Accuracy
- [ ] All code examples are accurate and use correct kira command syntax
- [ ] Files are properly formatted markdown
- [ ] Content is clear and actionable
- [ ] Examples demonstrate real-world workflows
- [ ] Clear delineation between README.md (general), AGENTS.md (quick reference), and docs/agents/using-kira.md (detailed)
- [ ] Links between all three files work correctly

## Release Notes
- Added kira usage section to README.md with general project documentation
- Added minimal kira guidance section to AGENTS.md with clear when-to-use instructions
- Created comprehensive kira usage guide at `docs/agents/using-kira.md`
- Established clear documentation hierarchy: README.md (general) → AGENTS.md (quick reference) → docs/agents/using-kira.md (detailed)
- README.md now links to AGENTS.md and docs/agents/using-kira.md for agent-specific guidance
- AGENTS.md links to dedicated documentation for detailed reference
- Improved agent workflow by separating general, quick reference, and detailed documentation
- Documentation now sources work item types, status folders, and validation rules from `kira.yml` configuration
- Makes documentation accurate for any kira workspace configuration
