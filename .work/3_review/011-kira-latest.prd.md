---
id: 011
title: kira latest
status: review
kind: prd
assigned:
estimate: 0
created: 2026-01-19
due: 2026-01-19
tags: []
---

# kira latest

A command that keeps feature branches updated with trunk changes through iterative conflict resolution. It first checks for existing merge conflicts, displays them for external LLM resolution, and only performs fetch/rebase when conflicts are resolved.

## Context

Kira manages work items through git-based workflows, but many users (especially non-technical stakeholders and product managers) struggle with git operations like rebasing and resolving merge conflicts. When feature branches fall behind the trunk branch, users need an easy way to:

1. Update their feature branch with the latest changes from trunk
2. Handle merge conflicts in a user-friendly way
3. Get AI assistance for understanding and resolving conflicts
4. Continue their work without being blocked by version control complexity

The `kira latest` command addresses this by providing a simple, guided workflow that handles the technical git operations while providing LLM-powered assistance for conflict resolution. Since kira supports polyrepo workflows (managing work across multiple repositories), the command must handle rebasing across multiple repos simultaneously, ensuring consistency and coordination between related repositories.

## Requirements

### Core Functionality
- **Iterative Workflow**: Allow repeated calls to work through conflicts progressively
- **Pre-Conflict Detection**: Check for existing merge conflicts before attempting fetch/rebase operations
- **Polyrepo Discovery**: Identify all repositories involved in the current work item based on kira configuration and work item metadata
- **Coordinated Updates**: Update all related repositories simultaneously to maintain consistency
- **Conditional Fetch/Rebase**: Only perform fetch and rebase when no conflicts exist in any repository
- **Cross-Repo Conflict Analysis**: Analyze conflicts across multiple repositories for interdependencies
- **Conflict Presentation**: When conflicts are detected, display them itemized in the terminal:
  - Show conflicts grouped by repository and file
  - Include full conflict content with clear markers (<<<<<<<, =======, >>>>>>>)
  - Provide context lines around each conflict for better understanding
  - Allow easy copy-paste into external LLMs (Cursor, ChatGPT, etc.) for resolution
- **Interactive Guidance**: Provide clear, step-by-step guidance for non-technical users managing multiple repositories
- **Safety Checks**: Ensure all working directories are clean before starting operations
- **Abort Capability**: Allow users to abort the rebase operation across all repositories if needed

### User Experience
- **Iterative Command**: Single command `kira latest` that can be called repeatedly to work through conflicts
- **Smart Status Detection**: Shows different behavior based on current state (conflicts exist vs ready to update)
- **Multi-Repo Progress Feedback**: Clear status messages showing progress across all repositories (e.g., "Updating repo1: fetching... rebasing...", "Updating repo2: fetching... complete")
- **Conflict Display Interface**: When conflicts exist in any repository, display them clearly in terminal:
  - Itemized list of all conflicting files across repositories
  - Full conflict content with git markers for easy copy-paste
  - Repository context for each conflict
  - Clear instructions for manual resolution using external tools
  - Option to abort the rebase and return to previous state
- **Educational Content**: Include explanations of what rebasing does, why it's needed, and how it affects polyrepo consistency
- **Error Handling**: Clear error messages for common failure scenarios across multiple repositories

### Integration Requirements
- **Git Integration**: Work with existing git workflows and remote configurations across multiple repositories
- **Polyrepo Coordination**: Respect repository relationships and dependencies defined in kira configuration
- **Kira Workflow**: Integrate with kira's work item management (potentially update work item status)
- **Configuration**: Respect kira.yml git settings (trunk_branch, remote) and polyrepo configuration
- **Safety**: Never overwrite uncommitted changes across any repository
- **Consistency**: Ensure all repositories are updated to the same logical point to maintain polyrepo consistency

## Acceptance Criteria

### Functional Requirements
- [ ] `kira latest` command exists and is executable
- [ ] Command identifies all repositories involved in the current work item
- [ ] Command first checks for existing merge conflicts before attempting any git operations
- [ ] Command only performs fetch/rebase when no conflicts exist in any repository
- [ ] Command can be called repeatedly to work through conflicts iteratively
- [ ] Command detects and displays existing merge conflicts in any repository
- [ ] Command analyzes conflicts across multiple repositories for interdependencies
- [ ] When conflicts detected, command displays all conflicts itemized in terminal with full content
- [ ] Conflicts are grouped by repository and file for clear organization
- [ ] Conflict display includes git markers (<<<<<<<, =======, >>>>>>>) for easy copy-paste to external LLMs
- [ ] Users can copy-paste conflicts into Cursor or other LLMs for resolution assistance
- [ ] Command validates all working directories are clean before starting
- [ ] Command respects kira.yml git configuration (trunk_branch, remote) and polyrepo settings
- [ ] Command provides clear progress feedback during operations across all repositories

### User Experience Requirements
- [ ] Command provides educational explanation of rebasing process across multiple repositories
- [ ] Conflict display is clear and organized for non-technical users managing polyrepos
- [ ] Conflicts are presented in a format that's easy to copy-paste into external LLMs
- [ ] Clear error messages for common failure scenarios (dirty working directories, no remotes, repo connectivity issues, etc.)
- [ ] Abort option available to return to previous state when conflicts are encountered
- [ ] Progress indicators show current operation per repository (fetching, rebasing, detecting conflicts)
- [ ] Success confirmation when rebase completes without conflicts across all repositories
- [ ] Unified view showing status of all repositories during the update process

### Safety Requirements
- [ ] Never proceeds with rebase if any working directory has uncommitted changes
- [ ] Provides backup/restore options for aborted operations across all repositories
- [ ] Validates git repository state for all involved repositories before starting
- [ ] Graceful handling of network/remote repository issues for individual repos
- [ ] No data loss if operation is interrupted - can rollback changes per repository
- [ ] Ensures atomic updates where possible, or clear rollback procedures for partial failures

### Integration Requirements
- [ ] Works with existing kira work item workflows across multiple repositories
- [ ] Respects all git configuration in kira.yml and polyrepo relationship definitions
- [ ] Compatible with different git hosting providers (GitHub, GitLab, etc.) across repos
- [ ] Integrates with kira's error handling and logging systems for multi-repo operations
- [ ] Coordinates with kira's project planning features to determine which repos need updating

## Implementation Notes

### Technical Architecture
- **State-Aware Execution**: Detect current repository state (conflicts exist vs ready for update) and behave accordingly
- **Polyrepo Discovery**: Implement repository relationship detection based on kira configuration and work item metadata
- **Git Operations**: Use go-git library for git operations across multiple repositories to maintain consistency
- **Parallel Processing**: Handle repository updates concurrently where possible, with proper synchronization
- **Conflict Display**: Format and display merge conflicts in terminal for external LLM consumption
- **Conflict Parsing**: Parse git conflict markers and organize conflicts across multiple repositories for clear display
- **Safety Mechanisms**: Implement comprehensive pre-flight checks and rollback capabilities for multiple repos

### Key Components
1. **State Detector**: Determines if repositories have existing conflicts or are ready for update operations
2. **Polyrepo Discovery**: Identifies all repositories involved in the current work item based on configuration
3. **Pre-flight Validator**: Checks working directory state, git configuration, remote connectivity for all repositories
4. **Git Operations Handler**: Manages fetch and rebase operations across multiple repositories with proper error handling
5. **Conflict Detector**: Identifies and parses merge conflicts from git status/output across all repositories
6. **Conflict Formatter**: Organizes and formats conflicts for clear terminal display with repository context
7. **Conflict Display**: Presents conflicts in itemized, copy-paste friendly format for external LLM use
8. **User Interface**: Provides clear instructions and options when conflicts are encountered
9. **Rollback Handler**: Manages abort scenarios and cleanup operations across all repositories

### Conflict Display Format
Conflicts are displayed in the terminal with:
- Repository name and file path for each conflict
- Full conflict content with standard git markers (<<<<<<<, =======, >>>>>>>)
- Context lines (3 lines before and after) for better understanding
- Clear section headers grouping conflicts by repository
- Instructions for copying conflicts to external LLMs like Cursor for resolution

### Error Scenarios to Handle
- Dirty working directories in any repository (uncommitted changes)
- Missing or inaccessible repositories in the polyrepo configuration
- No remote configured or accessible for individual repositories
- Network connectivity issues during fetch for specific repositories
- Partial success scenarios (some repos update successfully, others fail)
- Existing merge conflicts detected before fetch/rebase operations
- User abort during conflict resolution affecting all repositories
- Git repository corruption or invalid state in individual repositories
- Permission issues with remote repositories for specific repos
- Repository relationship inconsistencies or circular dependencies
- Mixed states across repositories (some with conflicts, some without)

### Configuration Integration
- Use `git.trunk_branch` from kira.yml (defaults to "master") for all repositories
- Use `git.remote` from kira.yml (defaults to "origin") for all repositories
- Respect polyrepo configuration defining which repositories are related to work items
- Allow per-repository git configuration overrides where needed
- Potentially add new configuration options for LLM provider preferences
- Support for repository dependency ordering during updates

## Implementation Strategy

The feature should be implemented incrementally with the following commit progression:

### Implementation Phases

#### Phase 1. CLI Command Structure
```
feat: add kira latest command skeleton

- Add basic command registration and CLI interface
- Implement command help and basic argument parsing
- Add placeholder for main command logic
- Update command routing
```

#### Phase 2. Polyrepo Discovery Logic
```
feat: implement polyrepo discovery for kira latest

- Add logic to identify repositories involved in current work item
- Parse kira configuration for repository relationships
- Implement work item metadata parsing
- Add repository validation and accessibility checks
```

#### Phase 3. State Detection System
```
feat: add state detection for conflict checking

- Implement pre-conflict detection logic
- Add git status checking across multiple repositories
- Create state enum (conflicts_exist, ready_for_update, etc.)
- Add repository state aggregation logic
```

#### Phase 4. Conflict Display Formatting
```
feat: implement conflict display formatting

- Add conflict parsing from git status output
- Implement itemized terminal display format
- Add repository grouping and file organization
- Include context lines and git markers (<<<<<<<, =======, >>>>>>>)
- Add copy-paste friendly formatting
```

#### Phase 5. Git Operations Handler
```
feat: add git operations for fetch and rebase

- Implement fetch logic for multiple repositories
- Add rebase operation with trunk branch
- Create parallel processing for multi-repo operations
- Add progress feedback and status reporting
```

#### Phase 6. Error Handling & Safety
```
feat: implement error handling and safety mechanisms

- Add working directory cleanliness checks
- Implement abort and rollback capabilities
- Add network and permission error handling
- Create comprehensive error messages
- Add partial failure recovery
```

#### Phase 7. Configuration Integration
```
feat: integrate kira configuration settings

- Respect git.trunk_branch and git.remote from kira.yml
- Add polyrepo relationship configuration support
- Implement per-repository override capabilities
- Add dependency ordering for repository updates
```

#### Phase 8. Integration Tests and E2E Tests
```
test: add integration tests for kira latest workflows

- Test full multi-repo coordination
- Add iterative workflow testing
- Implement configuration integration tests
- Test error recovery scenarios

test: add e2e tests for kira latest command

- Test complete command workflow in test environment
- Add polyrepo scenario testing
- Implement conflict resolution workflow testing
- Add performance and reliability tests
```

### Benefits of This Approach

- **Incremental Development**: Each commit adds working functionality
- **Easy Review**: Smaller, focused changes are easier to review
- **Quick Feedback**: Issues can be caught early in smaller chunks
- **Logical Dependencies**: Each commit builds on the previous ones
- **Test Coverage**: Testing is added alongside implementation
- **Revert Safety**: Issues can be isolated to specific commits

## Testing Strategy

### Unit Tests
- Mock git operations (fetch, rebase, status checks)
- Test conflict detection and parsing across multiple repositories
- Validate state detection logic (conflicts exist vs ready for update)
- Test polyrepo discovery based on kira configuration
- Error scenario coverage for network issues, permission problems, and invalid states

### Integration Tests
- Full workflow testing with local git repositories
- Multi-repository coordination testing with dependency relationships
- Conflict resolution workflow testing (detect → display → resolve → continue)
- Configuration integration testing with kira.yml settings
- Iterative command execution testing across multiple calls

### E2E Tests
- `kira latest` command in test polyrepo environment
- Conflict detection and display verification
- Iterative workflow testing (multiple command calls to resolve conflicts)
- Polyrepo coordination verification across related repositories
- Error recovery and rollback testing

## Release Notes

- Added `kira latest` command for iterative feature branch updates across multiple repositories
- Smart state detection - checks for existing conflicts before attempting fetch/rebase operations
- Polyrepo support with coordinated updates to maintain consistency between related repositories
- Clear terminal display of merge conflicts for easy copy-paste into external LLMs (Cursor, ChatGPT, etc.)
- Iterative workflow - call command repeatedly to work through conflicts progressively
- Itemized conflict presentation with repository context and full conflict content
- Safe, guided workflow that prevents data loss and provides clear feedback across all repositories
- Educational content to help users understand git rebasing concepts in polyrepo environments
- Comprehensive error handling for common git operation failures and polyrepo coordination issues
- Integration with existing kira git configuration and polyrepo relationship definitions

