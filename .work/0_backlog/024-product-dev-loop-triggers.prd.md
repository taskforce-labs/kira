---
id: 024
title: product dev loop triggers
status: backlog
kind: prd
assigned:
estimate: 0
created: 2026-01-27
due: 2026-01-27
tags: []
---

# product dev loop triggers

Commands that can be configured for an agent paradigm like cursor agents or claude to run the different product development phases - product discovery, domain discovery, target architecture, etc

## Overview

Product dev loop triggers are Kira commands that orchestrate agent-driven workflows for different phases of the product development lifecycle. These triggers leverage the Cursor CLI to start agent sessions with the appropriate skills and commands installed via `kira install cursor-skills` and `kira install cursor-commands` (see PRD 023).

Each trigger is a Kira command (e.g., `kira run product-discovery`) that:
1. Configures and invokes the Cursor CLI agent with appropriate prompts
2. Activates relevant skills and commands for the specific development phase
3. Provides context from the Kira project (work items, PRDs, configuration)
4. Handles intervention points where human input is required
5. Integrates results back into Kira's workflow (creating ADRs, PRDs, work items, etc.)

**Design Philosophy:** While initially targeting Cursor CLI, the implementation should use an abstraction layer that allows future support for other agent CLIs (Claude, GitHub Copilot CLI, etc.) without major refactoring.

## Context

Kira's product development workflow consists of multiple phases that benefit from AI agent assistance:
- **Product Discovery**: Understanding user needs, constraints, assumptions, risks, and commercial considerations
- **Domain Discovery**: Modeling the problem domain and context
- **Technical Discovery**: Identifying technical constraints, dependencies, and architecture decisions
- **Roadmap Planning**: Breaking features into work items and sequencing them
- **Work Item Elaboration**: Detailing work items into testable phases
- **RALF on Work Items**: Parallel execution of work items with coordination

Each phase requires:
- Specific skills to guide the agent (installed via PRD 023)
- Specific commands to orchestrate workflows (installed via PRD 023)
- Context from the Kira project (PRDs, work items, configuration)
- Integration with Kira's artifact management (creating/updating PRDs, work items, ADRs)

The triggers provide a unified interface to kick off these agent-driven workflows, abstracting away the specifics of the underlying agent CLI while maintaining flexibility for future extensibility.

### Cursor CLI Integration

The Cursor CLI (`agent` command) provides:
- Interactive mode: `agent "prompt"` for conversational sessions
- Non-interactive mode: `agent -p "prompt" --output-format text` for automation
- Session management: `agent ls`, `agent resume` for continuing conversations
- Cloud Agent handoff: `& prompt` to push to cloud agents

Run commands will primarily use interactive mode for discovery phases (where human intervention is frequent) and non-interactive mode for structured tasks (where automation is preferred).

### Artifact-Based Workflow Orchestration

Run commands can operate in two modes: **manual execution** and **watch mode**. Both modes leverage artifact dependencies to orchestrate workflows intelligently.

**Artifact Dependencies:**
- Each development phase requires certain input artifacts to be present before execution
- Each phase produces specific output artifacts that downstream phases depend on
- Dependencies are defined in phase configuration (e.g., technical discovery requires product discovery PRD)

**Manual Execution:**
- `kira run <phase>` checks for required input artifacts
- If missing, provides clear guidance on what's needed
- Executes until required output artifacts are produced or intervention is needed
- Stops when artifacts are complete or cannot proceed without additional input

**Watch Mode:**
- `kira run <phase> --watch` monitors artifact directories for changes
- When input artifacts change, automatically evaluates downstream impact
- Triggers re-evaluation of dependent phases
- Presents options for handling architectural changes when code already exists
- Supports incremental/transitory states rather than requiring complete rewrites

**Transitory States:**
When architecture is already implemented in code and new facts invalidate assumptions:
- System analyzes current codebase state
- Generates options for incremental migration paths
- Presents refactoring strategies that minimize disruption
- Allows course correction without full architectural restart
- Tracks assumption invalidation and suggests updates to related artifacts

**Example Workflow:**
1. Product discovery PRD is created → triggers domain discovery evaluation
2. Domain model is updated → triggers technical discovery re-evaluation
3. Technical discovery reveals new constraint → presents options:
   - Option A: Incremental refactor (add adapter layer)
   - Option B: Transitory state (deprecate old, introduce new)
   - Option C: Full architectural change (with migration plan)
4. User selects option → system updates artifacts and code incrementally

## Requirements

### Functional Requirements

1. **Run Commands**
   - `kira run product-discovery [options]` - Start product discovery workflow
   - `kira run domain-discovery [options]` - Start domain modeling workflow
   - `kira run technical-discovery [options]` - Start technical discovery workflow
   - `kira run roadmap-planning [options]` - Start roadmap planning workflow
   - `kira run elaborate-work-item <work-item-id> [options]` - Elaborate a specific work item
   - `kira run ralf-work-item <work-item-id> [options]` - RALF on a specific work item
   - Each run command should accept options for:
     - `--interactive` / `--non-interactive` (default: interactive for discovery, non-interactive for execution)
     - `--resume <session-id>` - Resume a previous agent session
     - `--context <file>` - Provide additional context file
     - `--output-format <format>` - Control output format (text, json, etc.)

2. **Agent CLI Abstraction Layer**
   - Abstract agent CLI operations behind an interface
   - Initial implementation for Cursor CLI (`cursor` provider)
   - Design interface to support future providers (Claude, GitHub Copilot, etc.)
   - Provider selection via configuration or flag: `--agent-provider cursor` (default)
   - Provider-specific configuration in `.kira/config.yaml`

3. **Context Preparation**
   - Gather relevant Kira project context before invoking agent:
     - Project configuration (`.kira/config.yaml`)
     - Relevant PRDs from `.work/` directory
     - Work items and their status
     - Existing ADRs and architecture docs
     - Codebase context (via codebase search or summaries)
   - Package context into a format the agent can consume
   - Include instructions for which skills/commands to use

4. **Skill and Command Activation**
   - Ensure required skills are installed (check `~/.cursor/skills/`)
   - Ensure required commands are installed (check `~/.cursor/commands/`)
   - Provide prompts that guide agent to use appropriate skills/commands
   - For Cursor CLI, reference skills/commands by name in prompts
   - Handle missing skills/commands gracefully (prompt user to install)

5. **Intervention Point Handling**
   - Detect when agent reaches intervention points (via agent output or session state)
   - Pause agent session and present options/decisions to user
   - Resume session with user input
   - For non-interactive mode, log intervention points for later review

6. **Result Integration**
   - Parse agent outputs to extract artifacts (PRDs, ADRs, work items, code)
   - Create/update Kira artifacts in appropriate locations (`.work/`, `docs/adr/`, etc.)
   - Update work item status and metadata
   - Commit artifacts to version control (optional, configurable)

7. **Session Management**
   - Track active agent sessions (session ID, phase, status)
   - Support resuming interrupted sessions
   - List and manage sessions: `kira run list-sessions`
   - Clean up completed sessions

8. **Artifact Dependency Management**
   - Define artifact dependencies per phase (input artifacts required, output artifacts produced)
   - Check for required input artifacts before phase execution
   - Validate artifact completeness before marking phase as complete
   - Track artifact versions/timestamps to detect changes
   - Support partial completion states (phase can stop when blocked, resume when unblocked)

9. **Watch Mode**
   - `kira run <phase> --watch` monitors specified artifact directories
   - Detect artifact changes (creation, modification, deletion)
   - Evaluate downstream impact when artifacts change
   - Automatically trigger dependent phase re-evaluation
   - Support watching multiple artifacts or artifact patterns
   - Configurable watch patterns per phase in `.kira/config.yaml`

10. **Transitory State Management**
    - When code already exists and assumptions are invalidated:
      - Analyze current codebase structure
      - Generate incremental migration options (not just full rewrites)
      - Present refactoring strategies with trade-offs
      - Support "deprecate old, introduce new" patterns
      - Track assumption changes and propagate to related artifacts
    - Options should include:
      - Incremental changes (add adapter, wrapper, facade)
      - Transitory states (parallel implementations, feature flags)
      - Migration paths (phased rollout, gradual replacement)
      - Full architectural change (with detailed migration plan)
    - User selects option → system updates artifacts and suggests code changes incrementally

11. **Assumption Tracking and Invalidation**
    - Track assumptions in artifacts (PRDs, ADRs, work items)
    - Detect when new facts invalidate existing assumptions
    - Identify all artifacts dependent on invalidated assumptions
    - Present impact analysis (what needs to change, what's affected)
    - Suggest artifact updates and code changes
    - Support assumption validation workflows

### Technical Requirements

1. **Agent Provider Interface**
   ```go
   type AgentProvider interface {
       // Start a new agent session with a prompt
       StartSession(ctx context.Context, prompt string, options SessionOptions) (*Session, error)

       // Resume an existing session
       ResumeSession(ctx context.Context, sessionID string) (*Session, error)

       // List active sessions
       ListSessions(ctx context.Context) ([]Session, error)

       // Get session status
       GetSession(ctx context.Context, sessionID string) (*Session, error)

       // Check if provider is available (CLI installed, authenticated, etc.)
       IsAvailable(ctx context.Context) (bool, error)

       // Get provider name
       Name() string
   }
   ```

2. **Cursor CLI Provider Implementation**
   - Use Cursor CLI (`agent` command) for agent interactions
   - Parse Cursor CLI output to extract session IDs, status, artifacts
   - Handle Cursor CLI specific features (Cloud Agent handoff, session resume)
   - Support both interactive and non-interactive modes
   - Integrate with Cursor skills/commands installed via PRD 023

3. **Context Builder**
   - Read `.kira/config.yaml` for project configuration
   - Scan `.work/` directory for relevant PRDs and work items
   - Build context summary or include relevant files
   - Format context for agent consumption (markdown, structured prompts)

4. **Configuration**
   - Agent provider selection in `.kira/config.yaml`:
     ```yaml
     agent:
       provider: cursor  # cursor, claude, copilot (future)
       cursor:
         cli_path: "agent"  # or full path
         default_mode: interactive
         cloud_agent_enabled: false
       # Future: claude, copilot configs
     ```
   - Per-run configuration with artifact dependencies:
     ```yaml
     runs:
       product-discovery:
         skills: [product-discovery, domain-discovery]
         commands: [product-discovery]
         context_files: [.work/0_backlog/*.prd.md]
         # Artifact dependencies
         requires: []  # No dependencies - can start from scratch
         produces:
           - type: prd
             pattern: ".work/0_backlog/{id}-{title}.prd.md"
             required: true
         completion_criteria:
           - artifact_exists: "prd"
           - artifact_complete: true  # PRD has all required sections

       domain-discovery:
         requires:
           - type: prd
             pattern: ".work/0_backlog/*.prd.md"
             min_count: 1
         produces:
           - type: domain-model
             pattern: "docs/domain/{name}.md"
           - type: context-map
             pattern: "docs/domain/context-map.md"
         watch:
           enabled: true
           watch_patterns:
             - ".work/0_backlog/*.prd.md"
           trigger_on: [create, update]
           downstream_phases: [technical-discovery, roadmap-planning]

       technical-discovery:
         requires:
           - type: prd
             pattern: ".work/0_backlog/*.prd.md"
           - type: domain-model
             pattern: "docs/domain/*.md"
         produces:
           - type: adr
             pattern: "docs/adr/{id}-{title}.md"
         watch:
           enabled: true
           watch_patterns:
             - ".work/0_backlog/*.prd.md"
             - "docs/domain/*.md"
         transitory_states:
           enabled: true
           code_analysis: true
           migration_strategies: [incremental, transitory, full]

       elaborate-work-item:
         skills: [work-item-elaboration]
         commands: [elaborate-work-item]
         auto_commit: false
         requires:
           - type: work-item
             pattern: ".work/{status}/{id}-{title}.prd.md"
             status: [todo, in-progress]
     ```

5. **Error Handling**
   - Handle agent CLI not installed or not available
   - Handle missing skills/commands (prompt to install)
   - Handle agent session failures
   - Handle context preparation errors
   - Provide helpful error messages with remediation steps

6. **Extensibility Design**
   - Agent provider interface allows adding new providers without changing run command logic
   - Provider-specific code isolated in provider implementations
   - Shared utilities for context building, artifact parsing, session management
   - Configuration schema extensible for new provider options

7. **Artifact Dependency System**
   - Artifact dependency checker validates required artifacts exist before phase execution
   - Artifact pattern matcher (glob patterns) to find artifacts
   - Artifact completeness validator (checks for required sections, fields)
   - Artifact version/timestamp tracker to detect changes
   - Dependency graph builder to understand phase relationships
   - Partial completion state support (phase can pause when blocked)

8. **Watch Mode Implementation**
   - File system watcher for artifact directories (use platform-appropriate library)
   - Debounce/throttle file change events to avoid excessive triggers
   - Pattern matching for watched files (glob patterns)
   - Change detection (create, update, delete events)
   - Downstream impact analyzer (which phases depend on changed artifacts)
   - Automatic phase re-evaluation trigger
   - Watch mode session management (long-running sessions)

9. **Transitory State Engine**
   - Codebase analyzer (AST parsing, dependency analysis)
   - Current state detector (what's already implemented)
   - Assumption extractor (parse assumptions from artifacts)
   - Impact analyzer (what code/artifacts depend on invalidated assumptions)
   - Migration strategy generator:
     - Incremental: adapter patterns, wrappers, facades
     - Transitory: parallel implementations, feature flags, deprecation paths
     - Full: complete architectural change with migration plan
   - Option presenter (multiple strategies with trade-offs)
   - Incremental change executor (suggest code changes step-by-step)

10. **Assumption Tracking**
    - Assumption parser (extract assumptions from PRDs, ADRs, work items)
    - Assumption dependency graph (which artifacts/code depend on which assumptions)
    - Invalidation detector (compare assumptions with new facts)
    - Impact analyzer (identify affected artifacts and code)
    - Assumption update suggester (how to update artifacts when assumptions change)

## Acceptance Criteria

1. ✅ Running `kira run product-discovery` successfully starts a Cursor CLI agent session with product discovery skills/commands activated
2. ✅ Agent session receives relevant Kira project context (PRDs, configuration, work items)
3. ✅ Agent uses installed skills and commands from PRD 023 appropriately
4. ✅ Intervention points pause the session and prompt user for input
5. ✅ Agent outputs (PRDs, ADRs) are integrated into Kira's `.work/` structure
6. ✅ `kira run list-sessions` shows active and completed sessions
7. ✅ `kira run resume <session-id>` successfully resumes an interrupted session
8. ✅ Run commands work with both interactive and non-interactive Cursor CLI modes
9. ✅ Missing skills/commands are detected and user is prompted to install via `kira install cursor-skills`
10. ✅ Agent provider can be configured via `.kira/config.yaml`
11. ✅ Cursor CLI provider implementation works end-to-end
12. ✅ Agent provider interface is designed to support future providers (Claude, etc.) without major refactoring
13. ✅ Context preparation includes relevant project artifacts and configuration
14. ✅ Error handling provides clear messages for common failure scenarios
15. ✅ Non-interactive mode logs intervention points for later review
16. ✅ Run commands check for required input artifacts before execution and provide clear guidance if missing
17. ✅ Run commands stop when required output artifacts are produced or cannot proceed without additional input
18. ✅ `kira run <phase> --watch` successfully monitors artifact directories and detects changes
19. ✅ Watch mode automatically triggers downstream phase re-evaluation when input artifacts change
20. ✅ Transitory state engine analyzes existing codebase and generates incremental migration options
21. ✅ When assumptions are invalidated, system presents multiple migration strategies (incremental, transitory, full) with trade-offs
22. ✅ Assumption tracking identifies all artifacts and code dependent on invalidated assumptions
23. ✅ Artifact dependency graph correctly identifies phase execution order and dependencies
24. ✅ Partial completion states allow phases to pause when blocked and resume when unblocked

## Implementation Notes

### Architecture

```
kira run <phase> [--watch]
  ├── Artifact Dependency Checker
  │   ├── Validate required input artifacts exist
  │   ├── Check artifact completeness
  │   ├── Build dependency graph
  │   └── Determine if phase can execute
  ├── Context Builder
  │   ├── Read .kira/config.yaml
  │   ├── Gather relevant PRDs/work items
  │   ├── Extract assumptions from artifacts
  │   └── Build context prompt
  ├── Agent Provider Interface
  │   ├── Cursor CLI Provider (initial)
  │   │   ├── Check skills/commands installed
  │   │   ├── Build agent prompt with context
  │   │   ├── Invoke Cursor CLI (agent command)
  │   │   └── Parse session output
  │   └── [Future: Claude Provider, Copilot Provider]
  ├── Session Manager
  │   ├── Track active sessions (markdown files in .kira/sessions/)
  │   ├── Store session metadata (YAML frontmatter + markdown content)
  │   └── Handle resume/cleanup
  ├── Result Integrator
  │   ├── Parse agent outputs
  │   ├── Create/update Kira artifacts
  │   ├── Update work item status
  │   └── Validate output artifacts produced
  ├── Watch Mode (if --watch)
  │   ├── File System Watcher
  │   ├── Change Detector
  │   ├── Downstream Impact Analyzer
  │   └── Auto-trigger dependent phases
  └── Transitory State Engine (when code exists)
      ├── Codebase Analyzer
      ├── Assumption Invalidation Detector
      ├── Migration Strategy Generator
      └── Option Presenter
```

### Agent Provider Interface Design

The abstraction layer should be minimal but sufficient:

```go
// Agent provider abstraction
type AgentProvider interface {
    // Core operations
    StartSession(ctx context.Context, req SessionRequest) (*Session, error)
    ResumeSession(ctx context.Context, sessionID string) (*Session, error)
    ListSessions(ctx context.Context) ([]Session, error)

    // Provider capabilities
    SupportsMode(mode SessionMode) bool
    IsAvailable(ctx context.Context) (bool, error)
    Name() string
}

// Session request includes context, skills, commands
type SessionRequest struct {
    Prompt       string
    Context      string  // Prepared context from Kira
    Skills       []string
    Commands     []string
    Mode         SessionMode
    Options      map[string]interface{}  // Provider-specific options
}

// Session represents an active agent session
type Session struct {
    ID          string
    Provider    string
    Phase       string
    Status      SessionStatus
    CreatedAt   time.Time
    UpdatedAt   time.Time
    Metadata    map[string]interface{}  // Provider-specific metadata
}
```

### Cursor CLI Provider Implementation

```go
type CursorProvider struct {
    cliPath string
    config  CursorConfig
}

func (p *CursorProvider) StartSession(ctx context.Context, req SessionRequest) (*Session, error) {
    // 1. Check Cursor CLI available
    if err := p.checkCLIAvailable(); err != nil {
        return nil, fmt.Errorf("cursor CLI not available: %w", err)
    }

    // 2. Verify skills/commands installed
    if err := p.verifySkillsCommands(req.Skills, req.Commands); err != nil {
        return nil, fmt.Errorf("missing skills/commands: %w", err)
    }

    // 3. Build Cursor CLI prompt
    prompt := p.buildPrompt(req)

    // 4. Invoke Cursor CLI
    var cmd *exec.Cmd
    if req.Mode == Interactive {
        cmd = exec.Command(p.cliPath, prompt)
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
    } else {
        cmd = exec.Command(p.cliPath, "-p", prompt, "--output-format", "text")
        // Capture output for parsing
    }

    // 5. Track session
    session := &Session{
        ID:        generateSessionID(),
        Provider:  "cursor",
        Phase:     extractPhase(req.Prompt),
        Status:    SessionActive,
        CreatedAt: time.Now(),
    }

    return session, nil
}
```

### Context Building

```go
type ContextBuilder struct {
    projectRoot string
    config      *KiraConfig
}

func (b *ContextBuilder) BuildContext(phase string) (string, error) {
    var context strings.Builder

    // 1. Project configuration summary
    context.WriteString("## Project Configuration\n")
    context.WriteString(b.summarizeConfig())

    // 2. Relevant PRDs
    context.WriteString("\n## Relevant PRDs\n")
    prds := b.findRelevantPRDs(phase)
    for _, prd := range prds {
        context.WriteString(fmt.Sprintf("- %s: %s\n", prd.ID, prd.Title))
        context.WriteString(prd.Summary + "\n")
    }

    // 3. Work items
    context.WriteString("\n## Work Items\n")
    workItems := b.findRelevantWorkItems(phase)
    for _, item := range workItems {
        context.WriteString(fmt.Sprintf("- %s: %s (%s)\n", item.ID, item.Title, item.Status))
    }

    // 4. Skills and commands to use
    context.WriteString("\n## Available Skills and Commands\n")
    context.WriteString(fmt.Sprintf("Skills: %s\n", strings.Join(b.getSkillsForPhase(phase), ", ")))
    context.WriteString(fmt.Sprintf("Commands: %s\n", strings.Join(b.getCommandsForPhase(phase), ", ")))

    return context.String(), nil
}
```

### Prompt Construction

Prompts should guide the agent to use the right skills and commands:

```
You are helping with [PHASE] for the Kira project.

## Context
[Prepared context from ContextBuilder]

## Available Skills
- product-discovery: Guide through product discovery process...
- domain-discovery: Create domain models and context maps...

## Available Commands
- /product-discovery: Start product discovery workflow...
- /domain-modelling: Create domain models...

## Instructions
1. Use the [SKILL-NAME] skill for detailed guidance
2. Use the /[COMMAND-NAME] command to start the workflow
3. Follow the intervention points defined in the skills
4. Create artifacts in the appropriate locations (see project config)
5. Update work items as you progress

## Your Task
[Phase-specific task description]

Begin by using the /[COMMAND-NAME] command.
```

### Integration with PRD 023

Run commands leverage skills and commands installed via PRD 023:

1. **Skills Activation**: Run commands reference skills by name in prompts, and Cursor automatically makes them available when relevant
2. **Commands Activation**: Run commands instruct the agent to use specific commands (e.g., `/product-discovery`)
3. **Configuration**: Skills/commands read `.kira/config.yaml` for project-specific behavior (as designed in PRD 023)
4. **Installation Check**: Run commands verify required skills/commands are installed before starting sessions

### Future Extensibility

To add support for other agent CLIs (Claude, GitHub Copilot, etc.):

1. **Implement AgentProvider interface** for the new CLI
2. **Add provider configuration** to `.kira/config.yaml` schema
3. **Update provider factory** to instantiate new provider
4. **No changes needed** to run commands or context building

Example for future Claude provider:
```go
type ClaudeProvider struct {
    apiKey string
    config ClaudeConfig
}

func (p *ClaudeProvider) StartSession(ctx context.Context, req SessionRequest) (*Session, error) {
    // Use Claude API or CLI
    // Adapt skills/commands to Claude's format
    // Return Session with Claude-specific metadata
}
```

### Artifact Dependency System

**Dependency Checking:**

```go
type ArtifactDependencyChecker struct {
    config *RunConfig
    fs     filesystem.FS
}

func (c *ArtifactDependencyChecker) CheckDependencies(phase string) (*DependencyStatus, error) {
    config := c.config.Runs[phase]

    // Check required artifacts
    missing := []ArtifactRequirement{}
    for _, req := range config.Requires {
        artifacts := c.findArtifacts(req.Pattern)
        if len(artifacts) < req.MinCount {
            missing = append(missing, req)
        }

        // Validate completeness
        for _, artifact := range artifacts {
            if !c.isComplete(artifact, req) {
                return &DependencyStatus{
                    CanExecute: false,
                    Missing: missing,
                    Incomplete: []string{artifact},
                }, nil
            }
        }
    }

    return &DependencyStatus{
        CanExecute: len(missing) == 0,
        Missing: missing,
    }, nil
}
```

**Artifact Completeness Validation:**

Artifacts are considered complete when they have all required sections. For PRDs, this might include:
- Title, description, context
- Requirements section
- Acceptance criteria
- Implementation notes (optional)

The validator parses markdown frontmatter and content to check for required sections.

### Watch Mode Implementation

**File System Watcher:**

```go
type ArtifactWatcher struct {
    patterns    []string
    watcher     fsnotify.Watcher
    debounce    time.Duration
    callbacks   []ChangeCallback
}

func (w *ArtifactWatcher) Watch(ctx context.Context) error {
    for {
        select {
        case event := <-w.watcher.Events:
            // Debounce rapid changes
            w.debounceEvents(event)
        case err := <-w.watcher.Errors:
            return err
        case <-ctx.Done():
            return nil
        }
    }
}

func (w *ArtifactWatcher) OnChange(event fsnotify.Event) {
    // Determine which phases depend on changed artifact
    affectedPhases := w.findDependentPhases(event.Name)

    // Trigger re-evaluation for each affected phase
    for _, phase := range affectedPhases {
        w.triggerPhaseEvaluation(phase, event)
    }
}
```

**Downstream Impact Analysis:**

When an artifact changes, the system:
1. Identifies which phases require that artifact type
2. Checks if those phases have active sessions
3. Triggers re-evaluation with updated context
4. Presents options if code already exists (transitory states)

### Transitory State Engine

**Codebase Analysis:**

```go
type TransitoryStateEngine struct {
    codebaseAnalyzer CodebaseAnalyzer
    assumptionTracker AssumptionTracker
    migrationGenerator MigrationGenerator
}

func (e *TransitoryStateEngine) AnalyzeInvalidation(
    invalidatedAssumptions []Assumption,
    currentCodebase *Codebase,
) (*MigrationOptions, error) {
    // 1. Extract assumptions from artifacts
    assumptions := e.assumptionTracker.ExtractFromArtifacts()

    // 2. Find code dependent on invalidated assumptions
    affectedCode := e.codebaseAnalyzer.FindDependentCode(
        invalidatedAssumptions,
        currentCodebase,
    )

    // 3. Generate migration strategies
    strategies := e.migrationGenerator.GenerateStrategies(
        invalidatedAssumptions,
        affectedCode,
        currentCodebase,
    )

    // 4. Present options with trade-offs
    return &MigrationOptions{
        Strategies: strategies,
        ImpactAnalysis: e.analyzeImpact(affectedCode),
    }, nil
}
```

**Migration Strategy Generation:**

The engine generates three types of strategies:

1. **Incremental (Adapter Pattern):**
   - Add adapter/wrapper layer
   - Minimal code changes
   - Gradual migration path
   - Example: "Add `NewPaymentAdapter` that wraps old payment system"

2. **Transitory (Parallel Implementation):**
   - Run old and new implementations in parallel
   - Feature flags for gradual rollout
   - Deprecation timeline
   - Example: "Introduce `PaymentServiceV2` alongside existing, migrate endpoints one by one"

3. **Full (Architectural Change):**
   - Complete redesign with migration plan
   - Phased rollout strategy
   - Detailed step-by-step guide
   - Example: "Refactor to event-driven architecture: Phase 1 (infrastructure), Phase 2 (core services), Phase 3 (migration)"

**Option Presentation:**

```go
type MigrationOption struct {
    Strategy      string  // "incremental", "transitory", "full"
    Description   string
    Pros          []string
    Cons          []string
    Effort        string  // "low", "medium", "high"
    Risk          string  // "low", "medium", "high"
    Timeline      string
    Steps         []MigrationStep
    CodeChanges   []CodeChangeSuggestion
}
```

The system presents options in a structured format, allowing users to:
- Compare strategies side-by-side
- Understand trade-offs
- See estimated effort and risk
- Review step-by-step migration plans
- Choose the approach that fits their context

### Assumption Tracking

**Assumption Extraction:**

Assumptions are extracted from artifacts using pattern matching:
- PRDs: "We assume that...", "Assuming...", "Assumption:"
- ADRs: Assumptions section
- Work items: Assumption tracking in notes

**Assumption Dependency Graph:**

```go
type AssumptionTracker struct {
    assumptions map[string]*Assumption
    dependencies map[string][]string  // assumption -> [artifacts, code files]
}

func (t *AssumptionTracker) TrackAssumption(assumption *Assumption) {
    // Find all artifacts that reference this assumption
    artifacts := t.findReferencingArtifacts(assumption)

    // Find all code that depends on this assumption
    codeFiles := t.findDependentCode(assumption)

    t.dependencies[assumption.ID] = append(artifacts, codeFiles...)
}

func (t *AssumptionTracker) Invalidate(assumptionID string) *ImpactAnalysis {
    affected := t.dependencies[assumptionID]
    return &ImpactAnalysis{
        Assumption: assumptionID,
        AffectedArtifacts: filterArtifacts(affected),
        AffectedCode: filterCodeFiles(affected),
        SuggestedUpdates: t.generateUpdateSuggestions(affected),
    }
}
```

When an assumption is invalidated:
1. System identifies all dependent artifacts and code
2. Presents impact analysis
3. Suggests updates to artifacts
4. Triggers transitory state engine if code is affected
5. Updates assumption tracking in artifacts

### Session Management

Sessions are tracked as markdown files in `.kira/sessions/` directory:
- `sessions/<session-id>.md` - Session details in markdown format
- Files are human-readable and can be reviewed/edited if needed
- Active sessions remain in `sessions/` directory
- Completed sessions can be archived to `sessions/completed/` (optional)

Each session markdown file contains:
- YAML frontmatter with structured metadata (session ID, provider, phase, status, timestamps)
- Session details in markdown format (context used, skills/commands activated, intervention points, artifacts created/updated)
- Notes and observations from the session
- Links to related artifacts (PRDs, work items, ADRs)

**Example session file structure:**

```markdown
---
id: sess-abc123
provider: cursor
phase: product-discovery
status: active
created: 2026-01-27T10:30:00Z
updated: 2026-01-27T10:45:00Z
---

# Product Discovery Session

## Context
- Project: Kira
- Phase: Product Discovery
- Related PRDs: 023-install-skills-command, 024-product-dev-loop-triggers

## Skills Activated
- product-discovery
- domain-discovery

## Commands Used
- /product-discovery

## Intervention Points
1. **Stakeholder Identification** (2026-01-27T10:32:00Z)
   - Decision: Include product managers and engineering leads
   - User input: "Focus on technical stakeholders for MVP"

2. **Risk Assessment** (2026-01-27T10:40:00Z)
   - Decision: Mitigate technical complexity risk
   - User input: "Add proof-of-concept phase"

## Artifacts Created
- `.work/0_backlog/025-new-feature.prd.md`
- `docs/adr/001-architecture-decision.md`

## Notes
- User requested additional focus on security considerations
- Next steps: Review PRD and schedule follow-up session
```

This markdown-based approach:
- ✅ Aligns with Kira's markdown-first philosophy
- ✅ Human-readable and reviewable
- ✅ Easy to version control
- ✅ Can include rich formatting and links
- ✅ Still machine-parseable via YAML frontmatter

**Session Manager Implementation:**

The Session Manager reads/writes markdown files:
- **Create session**: Generate session ID, create `<session-id>.md` with YAML frontmatter and initial content
- **Update session**: Append intervention points, artifacts, and notes to the markdown file, update frontmatter timestamps
- **Read session**: Parse YAML frontmatter for structured data, read markdown content for details
- **List sessions**: Scan `.kira/sessions/*.md` files, parse frontmatter to show session summaries
- **Resume session**: Read session file to restore context and continue from last intervention point

### Error Handling Strategy

1. **CLI Not Available**: Check if `agent` command exists, provide installation instructions
2. **Skills/Commands Missing**: Check `~/.cursor/skills/` and `~/.cursor/commands/`, prompt to run `kira install cursor-skills`
3. **Session Failures**: Capture error output, save session state, allow resume
4. **Context Errors**: Log missing files, use available context, warn user
5. **Integration Failures**: Validate artifact structure, provide rollback option

### Testing Strategy

1. **Unit Tests**: Agent provider interface, context builder, session manager
2. **Integration Tests**: Cursor CLI provider with mock CLI calls
3. **E2E Tests**: Full run command workflow with test project
4. **Provider Abstraction Tests**: Verify interface allows easy provider swapping

## Release Notes

### v1.0.0 (Initial Release)

- Add `kira run` commands for all product development phases
- Implement Cursor CLI provider (initial agent provider)
- Agent provider abstraction layer for future extensibility
- Context building from Kira project artifacts
- Session management (start, resume, list, cleanup) with markdown-based session files
- Integration with skills/commands from PRD 023
- Result integration (create/update PRDs, work items, ADRs)
- Configuration via `.kira/config.yaml`
- Support for interactive and non-interactive modes
- Error handling and user guidance
- **Artifact dependency system**: Run commands check for required input artifacts and validate completeness
- **Artifact-based completion**: Phases stop when output artifacts are produced or cannot proceed
- **Watch mode**: `kira run <phase> --watch` monitors artifacts and auto-triggers downstream phases
- **Transitory state engine**: Generates incremental migration options when code exists and assumptions are invalidated
- **Assumption tracking**: Tracks assumptions in artifacts, detects invalidation, and presents impact analysis
- **Migration strategy generation**: Presents incremental, transitory, and full architectural change options with trade-offs

