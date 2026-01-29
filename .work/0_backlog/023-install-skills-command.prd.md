---
id: 023
title: install skills command
status: backlog
kind: prd
assigned:
estimate: 0
created: 2026-01-27
due: 2026-01-27
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

**Installation Location:** Skills and commands are installed globally in the user's home directory (`~/.cursor/skills/` and `~/.cursor/commands/`) to make them available across all projects. However, **Kira's project-specific configuration controls variations and behavior per project**, allowing different projects to use the same skills with different parameters, templates, or workflows.

The installation process:
1. Creates the appropriate directory structure in user's home directory (`~/.cursor/skills/` and `~/.cursor/commands/`)
2. Installs skills from a curated repository or local source
3. Installs commands as markdown files in `~/.cursor/commands/`
4. Validates the installation and provides feedback
5. Skills and commands are designed to read Kira project configuration to adapt behavior per project

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
- `/break-into-phases` - Break work items into testable phases
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
   - Create `.cursor/skills/` directory if it doesn't exist
   - Install skills from a source (GitHub repository, local path, or bundled with Kira)
   - Each skill should be installed as a folder with `SKILL.md` file
   - Support optional skill directories: `scripts/`, `references/`, `assets/`
   - Validate skill structure (must have `SKILL.md` with valid frontmatter)
   - Handle skill updates (reinstall/update existing skills)

2. **Install Commands**
   - Create `.cursor/commands/` directory if it doesn't exist
   - Install commands as markdown files (`.md`) in the commands directory
   - Validate command files are valid markdown
   - Handle command updates

3. **Source Management**
   - Support installing from GitHub repository (public or private with auth)
   - Support installing from local directory
   - Support installing bundled skills/commands from Kira installation
   - Track installed skills/commands and their sources
   - Support uninstalling skills/commands

4. **User Experience**
   - Provide clear feedback during installation
   - List installed skills/commands
   - Show installation status and any errors
   - Support dry-run mode to preview what would be installed

### Technical Requirements

1. **Directory Structure**
   - Skills: `~/.cursor/skills/kira-<skill-name>/SKILL.md` (global installation)
   - Commands: `~/.cursor/commands/kira-<command-name>.md` (global installation)
   - Respect existing installations (don't overwrite without confirmation)
   - Skills are installed globally but read project-specific Kira configuration

2. **Project-Specific Configuration**
   - Skills and commands must read Kira project configuration (e.g., `.kira/config.yaml` or similar)
   - Configuration controls project-specific variations:
     - Which skills/commands are enabled/disabled for this project
     - Project-specific templates and artifacts structure
     - Project-specific workflows and conventions
     - Project-specific intervention points and decision criteria
     - Project-specific tooling and command preferences
   - Skills should check for project configuration at runtime
   - If no project configuration exists, skills use sensible defaults

3. **Validation**
   - Validate `SKILL.md` frontmatter (name, description required)
   - Ensure skill folder name matches skill name in frontmatter
   - Validate markdown syntax for commands
   - Validate that skills can access Kira project configuration

4. **Error Handling**
   - Handle network errors when fetching from GitHub
   - Handle permission errors when creating directories
   - Handle conflicts with existing skills/commands
   - Handle missing or invalid project configuration gracefully
   - Provide helpful error messages

## Acceptance Criteria

1. ✅ Running `kira install cursor-skills` successfully installs all required skills to `~/.cursor/skills/` (global)
2. ✅ Running `kira install cursor-commands` successfully installs all required commands to `~/.cursor/commands/` (global)
3. ✅ Installed skills appear in Cursor Settings → Rules → Agent Decides section
4. ✅ Installed commands appear when typing `/` in Cursor chat
5. ✅ Skills can be invoked explicitly via `/skill-name` in chat
6. ✅ Commands can be invoked via `/command-name` in chat
7. ✅ Skills automatically apply when agent determines they're relevant (unless `disable-model-invocation: true`)
8. ✅ Installation handles existing skills/commands gracefully (update vs skip)
9. ✅ Installation provides clear feedback on success/failure
10. ✅ Installation supports GitHub repository as source
11. ✅ Installation validates skill structure and reports errors
12. ✅ User can list installed skills/commands
13. ✅ User can uninstall skills/commands
14. ✅ Skills read project-specific Kira configuration (`.kira/config.yaml`) to adapt behavior per project kira will manage sensible defaults for any config so skills and commands don't need to worry about it.

## Implementation Notes

### Installation Source Strategy

**Option 1: GitHub Repository**
- Skills and commands stored in a dedicated GitHub repository
- Repository structure:
  ```
  kira-cursor-skills/
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
- Installation clones or downloads from GitHub
- Supports versioning via tags/branches

**Option 2: Bundled with Kira**
- Skills and commands bundled in Kira installation
- Stored in `kira/assets/cursor-skills/` or similar
- Copied to `.cursor/` during installation

**Option 3: Hybrid**
- Default skills/commands bundled with Kira
- Option to install additional from GitHub
- Option to install from local path

### Command Implementation

The `kira install cursor-skills` and `kira install cursor-commands` commands should:

1. Check if `~/.cursor/` directory exists in user's home directory, create if needed
2. Determine source (default to bundled, allow override with flag)
3. Fetch skills/commands from source
4. Validate structure
5. Install to global directories (`~/.cursor/skills/` and `~/.cursor/commands/`)
6. Report results
7. Optionally create or update project `.kira/config.yaml` with default cursor configuration if it doesn't exist

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
   - Check for `.kira/config.yaml` in the project root
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
   - Use project-specific template and location from `.kira/config.yaml`
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
- Skills can call `kira` commands (e.g., `kira create work-item`)
- Skills understand Kira's PRD format and structure
- Skills can read and update `.work/` directory structure
- Commands can trigger Kira workflows
- Skills prompt users at intervention points using Kira's decision framework

### Project-Specific Configuration

**Critical Design Consideration:** Since skills and commands are installed globally (`~/.cursor/skills/` and `~/.cursor/commands/`), Kira's project configuration must control variations per project.

**Configuration Strategy:**

1. **Kira Project Configuration File**
   - Location: `.kira/config.yaml` (or similar, in project root)
   - Contains project-specific settings that skills read at runtime
   - Examples of configurable aspects:
     ```yaml
     # .kira/config.yaml
     cursor:
       skills:
         enabled:
           - product-discovery
           - domain-discovery
           # Disable certain skills for this project
         disabled:
           - roadmap-planning

       commands:
         enabled:
           - product-discovery
           - create-adr

       project:
         # Project-specific templates
         prd_template: ".work/templates/prd-template.md"
         adr_template: ".work/templates/adr-template.md"

         # Project-specific conventions
         work_structure: ".work/{status}/{id}-{title}.prd.md"
         artifact_locations:
           prds: ".work/"
           adrs: "docs/adr/"

         # Project-specific workflows
         intervention_points:
           - stakeholder_identification
           - architecture_decisions
           - risk_assessment

         # Project-specific tooling
         test_command: "make test"
         lint_command: "make lint"
     ```

2. **Skill Behavior Adaptation**
   - Skills check for `.kira/config.yaml` at runtime
   - Skills read configuration to adapt:
     - Which templates to use
     - Where to create artifacts
     - What workflows to follow
     - Which intervention points to use
     - Project-specific conventions and patterns
   - If config doesn't exist, skills use sensible defaults
   - Skills can prompt user to create/update config if needed

3. **Command Behavior Adaptation**
   - Commands also read project configuration
   - Commands adapt workflows based on project settings
   - Commands can be enabled/disabled per project via config

4. **Configuration Management**
   - `kira init` or `kira configure` can create initial `.kira/config.yaml`
   - Skills can suggest configuration updates when patterns are detected
   - Configuration can be version-controlled with the project
   - Multiple projects can share the same global skills but have different behaviors

**Example Skill Reading Config:**

```markdown
## Instructions

1. **Check Project Configuration**
   - Read `.kira/config.yaml` if it exists
   - Use project-specific PRD template if configured
   - Use project-specific artifact locations
   - Adapt workflow based on project conventions

2. **If No Configuration**
   - Use default Kira conventions
   - Suggest creating project configuration
   - Use standard templates and locations
```

This design allows:
- ✅ One-time global installation of skills/commands
- ✅ Project-specific behavior via configuration
- ✅ Version-controlled project settings
- ✅ Easy onboarding (defaults work, config is optional)
- ✅ Flexibility to customize per project

## Release Notes

### v1.0.0 (Initial Release)

- Add `kira install cursor-skills` command
- Add `kira install cursor-commands` command
- Install skills for: Product Discovery, Domain Discovery, Technical Discovery, Roadmap Planning, Work Item Elaboration, RALF on Work Items
- Install commands for all workflow phases
- Support GitHub repository as installation source
- Support bundled skills/commands installation
- Validate skill and command structure
- Provide installation feedback and error handling

