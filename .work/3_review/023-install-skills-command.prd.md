---
id: 023
title: install skills command
status: review
kind: prd
assigned:
created: 2026-01-27
tags: []
---

# install skills command

```bash
kira install cursor-skills
```

```bash
kira install cursor-commands
```

Installs the skills and commands required for kira to leverage cursor agents to guide discovery, plan work, elaborate work items into phases, and build work items systematically. These skills and commands also know how to prompt users at the right intervention points when key decisions need to be made.

## Overview

This command installs Cursor Agent Skills and Commands that extend Cursor's AI agent capabilities to work seamlessly with Kira's workflow. Skills provide domain-specific knowledge and workflows, while commands provide reusable slash-command workflows that can be triggered with `/` prefix in chat.

**Kira comes loaded with defaults:** Default skills and commands are bundled with Kira. Running `kira install cursor-skills` (and `kira install cursor-commands`) copies these into the Cursor directories so they work across all projects. Only the bundled skills and commands are installed; there is no install from GitHub or local path.

**Installation Location:** The install path is configurable; the only config for this command is to override where skills and commands get installed. Default is the project root (`.agent/skills/` for skills and `.cursor/commands/` for commands). Skills and commands are installed there to make them available to the project.

The installation process:
1. Checks if skills/commands already exist at the target path
2. If existing items are found, prompts the user to choose: overwrite or cancel
3. Creates the appropriate directory structure at the configured path if it doesn't exist (default: project root)
4. Installs the skills/commands bundled with Kira (only if user chooses to overwrite or if items don't exist)
5. Validates the installation and provides feedback

**Overwrite Protection:** Before installing, the command checks each skill/command at the target path. If any already exist, it lists them and asks the user whether to overwrite them or cancel the installation. The installation will not proceed without explicit user confirmation to overwrite existing items.

When a required skill or command is not present (e.g. another Kira command or workflow needs it), Kira detects the gap and automatically installs the missing skill(s)/command(s) (runs the appropriate install); no user confirmation is required.

## Context

Kira's design philosophy is to be a tool that helps humans and agents work together to get things done. It's not a tool that will do the work for you, but it will help you get the work done by providing a way to manage the work and coordinate the work between humans and agents. Intervention points are key to this process and where key decisions or errors need to be dealt with by the most informed human.

Skills inform agents how to work through each aspect of the development workflow providing criteria for when human intervention is needed and how to prompt the user for the right information, list assumptions, provide options and guide the user to the next step.

Skills tell the agent how to call the right commands and when

### Cursor Commands

Commands are markdown files stored in `.cursor/commands/` that define reusable workflows triggered with `/` prefix in chat. The following commands combine skills, kira commands and cursor agents to produce artifacts or code for different phases of the product development lifecycle - automating more of the mundane work and freeing up humans to focus on the creative and strategic parts of the process.

Each command is a plain markdown file that describes:
- What the command does
- Step-by-step instructions for the agent
- When to use it
- How to prompt users for decisions at intervention points

**Bundled Commands:**
- `/elaborate-work-item` - Elaborate work item criteria and requirements
- `/break-into-slices` - Break work items into testable slices that can be committed to git
- `/plan-and-build` - Plan and build implementation

Commands can accept parameters - anything typed after the command name is included in the model prompt alongside the command content.

### Skills

Skills are portable, version-controlled packages that teach agents how to perform domain-specific tasks. Each skill is a folder containing a `SKILL.md` file with YAML frontmatter defining:
- `name`: Skill identifier (lowercase, hyphens)
- `description`: When and how to use the skill (used by agent for relevance detection)
- `disable-model-invocation`: Optional flag to make skill only available via explicit `/skill-name` invocation

Skills can include optional directories:
- `scripts/`: Executable code that agents can run
- `references/`: Additional documentation loaded on demand
- `assets/`: Static resources like templates, images, or data files

**Bundled Skills:**
1. **Work Item Elaboration** (`kira-work-item-elaboration`)
   - How to elaborate work items into small testable phases
   - Acceptance criteria definition
   - Phase sequencing and dependencies

2. **Clarifying Questions Format** (`kira-clarifying-questions-format`)
   - How to format and structure clarifying questions for users
   - Best practices for gathering requirements and context

Skills are automatically discovered by Cursor from `.agent/skills/` and made available to the agent. The agent decides when skills are relevant based on context, or they can be explicitly invoked via `/skill-name`.

## Removed from Scope

The following skills and commands have been removed from this work item and will be implemented in separate work items:

### Removed Commands

1. **`kira-create-adr`** - Generate architecture decision records
   - Scope: Command to guide creation of ADRs following project conventions
   - Future work item: Create separate work item for ADR command

2. **`kira-discover-domain-mode`** - Create domain models and context maps
   - Scope: Command to guide domain modeling and context mapping
   - Future work item: Create separate work item for domain discovery command

3. **`kira-product-discovery`** - Guide through product discovery process
   - Scope: Command to guide structured product discovery workflow
   - Future work item: Create separate work item for product discovery command

4. **`kira-target-architecture`** - Design target architecture
   - Scope: Command to guide target architecture design
   - Future work item: Create separate work item for target architecture command

### Removed Skills

1. **`kira-domain-discovery`** - Domain discovery skill
   - Scope: Skill teaching agents how to discover and articulate the domain and context of the product, domain modeling techniques, and context mapping approaches
   - Future work item: Create separate work item for domain discovery skill

2. **`kira-product-discovery`** - Product discovery skill
   - Scope: Skill teaching agents how to discover and articulate user needs, constraints, assumptions, risks and commercials, when to prompt users for clarification, how to structure discovery artifacts, and how to suggest data sources and discovery techniques
   - Future work item: Create separate work item for product discovery skill

3. **`kira-roadmap-planning`** - Roadmap planning skill
   - Scope: Skill teaching agents how to break down features into work items, sequence work items into a roadmap optimized for parallel development, minimize rework and merge conflicts, and perform dependency analysis
   - Future work item: Create separate work item for roadmap planning skill

4. **`kira-technical-discovery`** - Technical discovery skill
   - Scope: Skill teaching agents how to discover and articulate technical constraints, dependencies, and requirements, target architecture design patterns, and ADR creation and structure
   - Future work item: Create separate work item for technical discovery skill




## Requirements

### Functional Requirements

1. **Install Skills**
   - **Check for existing skills:** Before installing, check if any bundled skills already exist at the target path
   - **Prompt for overwrite:** If existing skills are found, list them and prompt the user to choose: overwrite existing skills or cancel installation
   - **Create directory structure:** Create the skills directory at the configured path if it doesn't exist (default: `.agent/skills/`)
   - **Install bundled skills:** Install only the skills bundled with Kira (only proceeds if user confirms overwrite or if items don't exist)
   - Each skill is installed as a folder with `SKILL.md` file
   - Support optional skill directories: `scripts/`, `references/`, `assets/`
   - Validate skill structure (must have `SKILL.md` with valid frontmatter)

2. **Install Commands**
   - **Check for existing commands:** Before installing, check if any bundled commands already exist at the target path
   - **Prompt for overwrite:** If existing commands are found, list them and prompt the user to choose: overwrite existing commands or cancel installation
   - **Create directory structure:** Create the commands directory at the configured path if it doesn't exist (default: `.cursor/commands/`)
   - **Install bundled commands:** Install only the commands bundled with Kira as markdown files (`.md`) (only proceeds if user confirms overwrite or if items don't exist)
   - Validate command files are valid markdown

3. **User Experience**
   - Provide clear feedback during installation
   - Show installation status and any errors

4. **Missing skills/commands**
   - When a required skill or command is not present (e.g. a Kira workflow or command depends on it), detect the missing item(s) and automatically run the appropriate install (`kira install cursor-skills` and/or `kira install cursor-commands`); no user confirmation required.

### Technical Requirements

1. **Directory Structure**
   - Install path is configurable (e.g. in `kira.yaml`); default is the project root
   - Skills: `<configured-base>/.agent/skills/kira-<skill-name>/SKILL.md` (default: `.agent/skills/kira-<skill-name>/SKILL.md`)
   - Commands: `<configured-base>/.cursor/commands/kira-<command-name>.md` (default: `.cursor/commands/kira-<command-name>.md`)
   - When target already has skills/commands, detect and offer to overwrite or cancel (no overwrite without user choice)

2. **Install Path Config**
   - The only config for install is to override where skills and commands get installed
   - Config source: `kira.yaml` (in project root) or equivalent; default is the project root
   - Install reads the configured path from config; if absent, use `.agent/skills/` and `.cursor/commands/` relative to project root

3. **Validation**
   - Validate `SKILL.md` frontmatter (name, description required)
   - Ensure skill folder name matches skill name in frontmatter
   - Validate markdown syntax for commands

4. **Error Handling**
   - Handle permission errors when creating directories
   - Handle existing skills/commands by offering overwrite or cancel
   - Handle missing or invalid path config gracefully
   - When a required skill or command is missing, run the install for the missing items
   - Provide helpful error messages

## Acceptance Criteria

1. ✅ Running `kira install cursor-skills` successfully installs all bundled skills to the configured path (default: `.agent/skills/`)
2. ✅ Running `kira install cursor-commands` successfully installs all bundled commands to the configured path (default: `.cursor/commands/`)
3. ✅ Skills are installed to the correct path and format that Cursor expects (so they can appear in Cursor Settings → Rules → Agent Decides when Cursor loads them)
4. ✅ Commands are installed to the correct path and format that Cursor expects (so they can appear when typing `/` in Cursor chat when Cursor loads them)
5. ✅ Skills can be invoked explicitly via `/skill-name` in chat
6. ✅ Commands can be invoked via `/command-name` in chat
7. ✅ Skills automatically apply when agent determines they're relevant (unless `disable-model-invocation: true`)
8. ✅ When skills/commands already exist at the target path, install detects them, lists the existing items, and prompts the user to choose: overwrite existing items or cancel installation (installation does not proceed without explicit user confirmation)
9. ✅ Installation provides clear feedback on success/failure
10. ✅ Installation validates skill structure and reports errors
11. ✅ Install path can be overridden via config (e.g. `kira.yaml`); default is the project root
12. ✅ When a required skill or command is not present, Kira automatically installs it (runs the appropriate install for the missing items); no user confirmation required.

## Implementation Notes

### Installation Source Strategy

Kira ships with bundled skills and commands only. The install command:

- **Source:** Only the skills and commands bundled with Kira (e.g. in `kira/assets/cursor-skills/` or similar). Running `kira install cursor-skills` / `kira install cursor-commands` copies these into the configured path (default: `.agent/skills/` and `.cursor/commands/`). The only config is to override where skills and commands get installed; default is the project root.

Bundled structure (for reference):
```
internal/cursorassets/
├── skills/
│   ├── kira-work-item-elaboration/
│   │   └── SKILL.md
│   └── kira-clarifying-questions-format/
│       └── SKILL.md
└── commands/
    ├── kira-elaborate-work-item.md
    ├── kira-break-work-item-into-slices.md
    └── kira-plan-and-build.md
```

### Command Implementation

The `kira install cursor-skills` and `kira install cursor-commands` commands should:

1. Resolve install path from config (the only config: where to install skills/commands); default is project root
2. **Check for existing items:** Check if any skills/commands already exist at the target path
3. **Handle existing items:** If existing items are found:
   - List all existing items that would be overwritten
   - Prompt the user: "The following skills/commands already exist. Overwrite them? (y/n)"
   - If user chooses "no" or cancels, abort installation and report cancellation
   - If user chooses "yes", proceed with installation
4. Create the target base directories (e.g. `.agent/` for skills, `.cursor/` for commands) if they don't exist
5. Copy bundled skills/commands from Kira's bundle to the configured path
6. Validate structure
7. Install to the configured directories (default: `.agent/skills/` and `.cursor/commands/`)
8. Report results (success, skipped items, errors)

**When a skill or command is not present:** Any Kira command or workflow that depends on a skill or command should check the configured path for required items before proceeding. If something is missing, automatically run the appropriate install (`kira install cursor-skills` and/or `kira install cursor-commands`) for the missing items; no user confirmation required.

### Skill Structure Example

```markdown
---
name: work-item-elaboration
description: Articulate a work item's intent and behaviour: value, outcomes, business rules, and flows. Operates in the behavioural space (what value is delivered and what correct behaviour looks like), not solution or implementation design. Use existing technical context only to inform accuracy, not to decide solutions. Intended as a prerequisite for slicing and delivery planning.
disable-model-invocation: false
---

# Work Item Elaboration

Guide the user through behavioural elaboration of a work item: clarifying value and outcomes, defining business rules and constraints, and describing flows and scenarios.

## When to Use

- When a work item is likely to be built but its value, rules, or behaviour are unclear
- After discovery, before technical design or slicing
- As a prerequisite for story slicing or delivery planning
- When acceptance criteria or examples are missing or ambiguous

## Instructions

1. **Read Project Configuration**
   - Check `kira.yaml` for work folder locations and templates
   - Use the `.work/` structure and `kira slice` if the project uses slices

2. **Read Work Item and Context**
   - Read the work item (title, description) and any linked docs
   - Review existing technical context only to ensure accuracy of value, rules, and flows
   - Do not introduce new technical or solution decisions

3. **Establish Value and Outcomes**
   - Clarify the user and business value this work item delivers
   - Identify who benefits and how
   - Define success in plain language (what changes if this ships)
   - Capture explicit non-goals or exclusions

4. **Articulate Business Rules and Constraints**
   - Capture rules, policies, and constraints that govern behaviour
   - Identify edge cases and exceptional conditions
   - Express rules in business language, not technical terms
   - Avoid solution or implementation decisions

5. **Describe Flows and Scenarios**
   - Describe user or system flows step by step
   - Highlight decision points and alternate paths
   - Use examples or scenarios where helpful
   - Surface ambiguities or unresolved decisions

6. **Record Open Questions and Assumptions**
   - List unresolved decisions or assumptions
   - Articulate assumptions in the work item explicitly
   - Note where technical choices may influence behaviour
   - Flag items that require follow-up before slicing or delivery

## Intervention Points

- When value or outcomes are unclear from the title or description
- When business rules conflict or are ambiguous
- When flows reveal missing decisions or assumptions
- When flows cannot be described without choosing a technical solution

At each intervention point, present options and guide the user to make informed decisions, or explicitly record the open question for follow-up.
```

### Command Structure Example

```markdown
# Elaborate Work Item

## Overview
Articulate the details of a work item from its title and description: value, business rules, and flows. Use existing technical context only to understand and inform that articulation; do not include in-depth technical design.

## Steps

1. **Read Work Item and Context**
   - Read the work item (title, description) and any linked docs
   - Review existing technical details only to understand value, rules, and flows

2. **Articulate Value and Business Rules**
   - Clarify the value and outcomes the work item delivers
   - Capture business rules and constraints in plain language

3. **Articulate Flows**
   - Describe user or system flows (steps, decisions, outcomes)
   - Note dependencies and touchpoints with other parts of the system

4. **Update Work Item**
   - Add or refine value, business rules, and flows in the work item
   - Store in project work folder

## Output

Work item with articulated value, business rules, and flows. Use the work-item-elaboration skill for detailed guidance.
```

### Integration with Kira Workflow

Skills and commands should integrate with Kira's workflow:
- Skills can call `kira` commands (e.g., `kira new`)
- Skills understand Kira's PRD format and structure
- Skills can read and update `.work/` directory structure
- Commands can trigger Kira workflows
- Skills prompt users at intervention points using Kira's decision framework

### Install Path Config

The only config for this command is to override where skills and commands get installed. Default is the project root (`.agent/skills/` and `.cursor/commands/`). Config is read from `kira.yaml` (project root) or equivalent; we only use it to get the install path override when present.

## Release Notes

### v1.0.0 (Initial Release)

- Add `kira install cursor-skills` command
- Add `kira install cursor-commands` command
- Kira ships with bundled skills/commands only; install command copies them to the configured path (default: `.agent/skills/` and `.cursor/commands/`)
- Only config: override where skills and commands get installed (e.g. via `kira.yaml`); default is the project root
- When skills/commands already exist at the target path, offer to overwrite or cancel
- When a required skill or command is not present, Kira automatically runs the appropriate install for the missing items (no user confirmation required)
- Install skills for: Work Item Elaboration, Clarifying Questions Format
- Install commands for: elaborate-work-item, break-into-slices, plan-and-build
- Validate skill and command structure
- Provide installation feedback and error handling
- Uninstalling skills/commands is out of scope

## Slices

### Install path config
- [x] T001: Add install path config keys in kira.yaml (e.g. cursor-skills-path, cursor-commands-path or base path) and document in schema/docs
- [x] T002: Implement path resolver: read config, default to .agent/skills/ and .cursor/commands/ when absent

### Bundled assets layout
- [x] T003: Define bundled source layout under kira assets (e.g. kira/assets/cursor-skills/skills/ and commands/)
- [x] T004: Implement listing/loading of bundled skills and commands from the bundle

### Install cursor-skills
- [x] T005: Register `kira install cursor-skills` subcommand and wire to installer
- [x] T006: Create target skills directory at configured path if it does not exist
- [x] T007: Copy bundled skills to configured path; detect existing and offer overwrite or cancel
- [x] T008: Validate skill structure (SKILL.md with required frontmatter) and report errors; provide install feedback

### Install cursor-commands
- [x] T009: Register `kira install cursor-commands` subcommand and wire to installer
- [x] T010: Create target commands directory at configured path if it does not exist
- [x] T011: Copy bundled commands to configured path; detect existing and offer overwrite or cancel
- [x] T012: Validate command markdown and report errors; provide install feedback

### Auto-install missing
- [x] T013: Add check for required skills/commands at configured path where Kira workflows depend on them
- [x] T014: When required skill or command is missing, run kira install cursor-skills and/or cursor-commands automatically (no user confirmation)

### Bundled skills and commands content
- [x] T015: Add bundled skill folders and SKILL.md for Work Item Elaboration, Clarifying Questions Format
- [x] T016: Add bundled command .md files for elaborate-work-item, break-into-slices, plan-and-build (ralf-work-item merged into plan-and-build)

### Documentation alignment
- [x] T017: Update PRD to remove references to ralf-work-item command (merged into plan-and-build): update line 21 description, line 220 bundled structure, line 363 release notes, and line 396 task description
- [x] T018: Verify plan-and-build command content matches spec expectations and uses correct kira commands (kira check, kira slice commit generate, git commit -F -)
- [x] T019: Ensure all bundled commands are correctly documented in PRD (elaborate-work-item, break-into-slices, plan-and-build only)

### Test fixes
- [x] T020: Update install tests to check for actual bundled skills and commands (kira-work-item-elaboration, kira-clarifying-questions-format, kira-elaborate-work-item, kira-break-work-item-into-slices, kira-plan-and-build) instead of non-existent kira-product-discovery
- [x] T021: Fix YAML frontmatter parsing issue in SKILL.md files by quoting description fields that contain colons

