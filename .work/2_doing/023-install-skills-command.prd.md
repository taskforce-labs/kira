---
id: 023
title: install skills command
status: doing
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

Installs the skills and commands required for kira to leverage cursor agents to guide discovery, plan work, elaborate work items into phases, and ralf on work items in parallel. These skills and commands also know how to prompt users at the right intervention points when key decisions need to be made.

## Overview

This command installs Cursor Agent Skills and Commands that extend Cursor's AI agent capabilities to work seamlessly with Kira's workflow. Skills provide domain-specific knowledge and workflows, while commands provide reusable slash-command workflows that can be triggered with `/` prefix in chat.

**Kira comes loaded with defaults:** Default skills and commands are bundled with Kira. Running `kira install cursor-skills` (and `kira install cursor-commands`) copies these into the Cursor directories so they work across all projects. Only the bundled skills and commands are installed; there is no install from GitHub or local path.

**Installation Location:** The install path is configurable; the only config for this command is to override where skills and commands get installed. Default is always the user's home directory (`~/.cursor/skills/` and `~/.cursor/commands/`). Skills and commands are installed there to make them available across all projects.

The installation process:
1. Creates the appropriate directory structure at the configured path (default: user's home directory)
2. Installs the skills/commands bundled with Kira
3. If skills/commands are already present at the target path, detects them and offers to overwrite or cancel
4. Validates the installation and provides feedback

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

**Proposed Commands:**
- `/product-discovery` - Guide through product discovery process
- `/domain-modelling` - Create domain models and context maps
- `/create-adr` - Generate architecture decision records
- `/target-architecture` - Design target architecture
- `/elaborate-work-item` - Elaborate work item criteria and requirements
- `/break-into-slices` - Break work items into testable slices that can be committed to git
- `/ralf-work-item` - RALF on work items in parallel
- `/test-and-verify` - Run tests and verify implementation

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

**Proposed Skills:**
1. **Product Discovery** (`product-discovery`)
   - How to discover and articulate user needs, constraints, assumptions, risks and commercials
   - When to prompt users for clarification
   - How to structure discovery artifacts
   - How to suggest data sources and discovery techniques

2. **Domain Discovery** (`domain-discovery`)
   - How to discover and articulate the domain and context of the product
   - Domain modeling techniques
   - Context mapping approaches

3. **Technical Discovery** (`technical-discovery`)
   - How to discover and articulate technical constraints, dependencies, and requirements
   - Target architecture design patterns
   - ADR (Architectural Decision Records) creation and structure

4. **Roadmap Planning** (`roadmap-planning`)
   - How to break down features into work items
   - Sequencing work items into a roadmap optimized for parallel development
   - Minimizing rework and merge conflicts
   - Dependency analysis

5. **Work Item Elaboration** (`work-item-elaboration`)
   - How to elaborate work items into small testable phases
   - Acceptance criteria definition
   - Phase sequencing and dependencies

6. **RALF on Work Items** (`ralf-work-items`)
   - How to RALF (Read, Analyze, Learn, Fix) on work items in parallel
   - Parallel execution strategies
   - Coordination and conflict resolution

Skills are automatically discovered by Cursor from `.cursor/skills/` and made available to the agent. The agent decides when skills are relevant based on context, or they can be explicitly invoked via `/skill-name`.




## Requirements

### Functional Requirements

1. **Install Skills**
   - Create the skills directory at the configured path if it doesn't exist (default: `~/.cursor/skills/`)
   - Install only the skills bundled with Kira
   - Each skill is installed as a folder with `SKILL.md` file
   - Support optional skill directories: `scripts/`, `references/`, `assets/`
   - Validate skill structure (must have `SKILL.md` with valid frontmatter)
   - If skills already exist at the target path, detect and offer to overwrite or cancel

2. **Install Commands**
   - Create the commands directory at the configured path if it doesn't exist (default: `~/.cursor/commands/`)
   - Install only the commands bundled with Kira as markdown files (`.md`)
   - Validate command files are valid markdown
   - If commands already exist at the target path, detect and offer to overwrite or cancel

3. **User Experience**
   - Provide clear feedback during installation
   - Show installation status and any errors

4. **Missing skills/commands**
   - When a required skill or command is not present (e.g. a Kira workflow or command depends on it), detect the missing item(s) and automatically run the appropriate install (`kira install cursor-skills` and/or `kira install cursor-commands`); no user confirmation required.

### Technical Requirements

1. **Directory Structure**
   - Install path is configurable (e.g. in `kira.yaml`); default is always the user's home directory
   - Skills: `<configured-base>/.cursor/skills/kira-<skill-name>/SKILL.md` (default: `~/.cursor/skills/kira-<skill-name>/SKILL.md`)
   - Commands: `<configured-base>/.cursor/commands/kira-<command-name>.md` (default: `~/.cursor/commands/kira-<command-name>.md`)
   - When target already has skills/commands, detect and offer to overwrite or cancel (no overwrite without user choice)

2. **Install Path Config**
   - The only config for install is to override where skills and commands get installed
   - Config source: `kira.yaml` (in project root) or equivalent; default is always the user's home directory
   - Install reads the configured path from config; if absent, use `~/.cursor/skills/` and `~/.cursor/commands/`

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

1. ✅ Running `kira install cursor-skills` successfully installs all bundled skills to the configured path (default: `~/.cursor/skills/`)
2. ✅ Running `kira install cursor-commands` successfully installs all bundled commands to the configured path (default: `~/.cursor/commands/`)
3. ✅ Skills are installed to the correct path and format that Cursor expects (so they can appear in Cursor Settings → Rules → Agent Decides when Cursor loads them)
4. ✅ Commands are installed to the correct path and format that Cursor expects (so they can appear when typing `/` in Cursor chat when Cursor loads them)
5. ✅ Skills can be invoked explicitly via `/skill-name` in chat
6. ✅ Commands can be invoked via `/command-name` in chat
7. ✅ Skills automatically apply when agent determines they're relevant (unless `disable-model-invocation: true`)
8. ✅ When skills/commands already exist at the target path, install detects them and offers to overwrite or cancel
9. ✅ Installation provides clear feedback on success/failure
10. ✅ Installation validates skill structure and reports errors
11. ✅ Install path can be overridden via config (e.g. `kira.yaml`); default is always the user's home directory
12. ✅ When a required skill or command is not present, Kira automatically installs it (runs the appropriate install for the missing items); no user confirmation required.

## Implementation Notes

### Installation Source Strategy

Kira ships with bundled skills and commands only. The install command:

- **Source:** Only the skills and commands bundled with Kira (e.g. in `kira/assets/cursor-skills/` or similar). Running `kira install cursor-skills` / `kira install cursor-commands` copies these into the configured path (default: `~/.cursor/skills/` and `~/.cursor/commands/`). The only config is to override where skills and commands get installed; default is always the user's home directory.

Bundled structure (for reference):
```
kira/assets/cursor-skills/   (or similar)
├── skills/
│   ├── product-discovery/
│   │   ├── SKILL.md
│   │   └── scripts/
│   └── domain-discovery/
│       └── SKILL.md
└── commands/
    ├── product-discovery.md
    └── domain-modelling.md
```

### Command Implementation

The `kira install cursor-skills` and `kira install cursor-commands` commands should:

1. Resolve install path from config (the only config: where to install skills/commands); default is user's home directory. Create the target base directory (e.g. `~/.cursor/`) if it doesn't exist.
2. Copy bundled skills/commands from Kira's bundle to the configured path
3. If skills/commands already exist at the target path, detect them and offer to overwrite or cancel (no overwrite without user choice)
4. Validate structure
5. Install to the configured directories (default: `~/.cursor/skills/` and `~/.cursor/commands/`)
6. Report results

**When a skill or command is not present:** Any Kira command or workflow that depends on a skill or command should check the configured path for required items before proceeding. If something is missing, automatically run the appropriate install (`kira install cursor-skills` and/or `kira install cursor-commands`) for the missing items; no user confirmation required.

### Skill Structure Example

```markdown
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
```

### Command Structure Example

```markdown
# Product Discovery

## Overview
Guide through a structured product discovery process to identify user needs, constraints, assumptions, risks, and commercial considerations.

## Steps

1. **Stakeholder Identification**
   - Identify key stakeholders
   - Understand their roles and interests
   - Document stakeholder map

2. **User Needs Discovery**
   - Identify target users
   - Understand their problems and needs
   - Create user stories

3. **Constraints Analysis**
   - Technical constraints
   - Business constraints
   - Timeline and resource constraints

4. **Assumptions Documentation**
   - List all assumptions
   - Validate critical assumptions with user

5. **Risk Assessment**
   - Identify and categorize risks
   - Develop mitigation strategies

6. **Commercial Analysis**
   - Revenue model
   - Cost structure
   - Market positioning

## Output

Create a product discovery document with:
- Stakeholder map
- User needs and stories
- Constraints list
- Assumptions log
- Risk register
- Commercial model

Use the product-discovery skill for detailed guidance on each step.
```

### Integration with Kira Workflow

Skills and commands should integrate with Kira's workflow:
- Skills can call `kira` commands (e.g., `kira new`)
- Skills understand Kira's PRD format and structure
- Skills can read and update `.work/` directory structure
- Commands can trigger Kira workflows
- Skills prompt users at intervention points using Kira's decision framework

### Install Path Config

The only config for this command is to override where skills and commands get installed. Default is always the user's home directory (`~/.cursor/skills/` and `~/.cursor/commands/`). Config is read from `kira.yaml` (project root) or equivalent; we only use it to get the install path override when present.

## Release Notes

### v1.0.0 (Initial Release)

- Add `kira install cursor-skills` command
- Add `kira install cursor-commands` command
- Kira ships with bundled skills/commands only; install command copies them to the configured path (default: `~/.cursor/skills/` and `~/.cursor/commands/`)
- Only config: override where skills and commands get installed (e.g. via `kira.yaml`); default is always the user's home directory
- When skills/commands already exist at the target path, offer to overwrite or cancel
- When a required skill or command is not present, Kira automatically runs the appropriate install for the missing items (no user confirmation required)
- Install skills for: Product Discovery, Domain Discovery, Technical Discovery, Roadmap Planning, Work Item Elaboration, RALF on Work Items
- Install commands for all workflow phases
- Validate skill and command structure
- Provide installation feedback and error handling
- Uninstalling skills/commands is out of scope

## Slices

### Install path config
- [x] T001: Add install path config keys in kira.yaml (e.g. cursor-skills-path, cursor-commands-path or base path) and document in schema/docs
- [x] T002: Implement path resolver: read config, default to ~/.cursor/skills/ and ~/.cursor/commands/ when absent

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
- [ ] T013: Add check for required skills/commands at configured path where Kira workflows depend on them
- [ ] T014: When required skill or command is missing, run kira install cursor-skills and/or cursor-commands automatically (no user confirmation)

### Bundled skills and commands content
- [ ] T015: Add bundled skill folders and SKILL.md for Product Discovery, Domain Discovery, Technical Discovery, Roadmap Planning, Work Item Elaboration, RALF on Work Items
- [ ] T016: Add bundled command .md files for product-discovery, domain-modelling, create-adr, target-architecture, elaborate-work-item, break-into-slices, ralf-work-item, test-and-verify

