---
id: 015
title: kira-release
status: backlog
kind: prd
assigned:
estimate: 0
created: 2026-01-21
due: 2026-01-21
tags: []
---

# kira-release


A two step command that releases a single, group or folder of work items.

kira release plan [version] [status|path] [subfolder]:
- accepts optional version parameter (e.g., "v1.2.3") - if not provided, uses date-based versioning
- shows which work items will be released
- creates a temporary release-notes-plan.md file with the release notes from the work items and version
- shows operations that will be performed including version/tag information

kira release confirm:
- reads version from the plan file (provided version or date-based)
- updates the status of the work items to "released"
- archives them to a date-organized archive structure with the release notes
- updates the releases file with the release notes from the temporary file
- removes the temporary release-notes-plan.md file
- runs pre commit scripts e.g. toggles a feature flag
- commits, tags (using version from plan) and pushes the changes to git
- runs post commit scripts e.g. toggles a release flag in feature

## Context

In the Kira workflow, work items progress through different statuses: backlog → todo → doing → review → done → released. When work items are completed and ready for release, teams need a safe, controlled way to:

1. Preview what will be released before committing to the release
2. Review consolidated release notes from multiple work items
3. Execute releases with proper git integration (commit, tag, push)
4. Run automation scripts before and after release (e.g., feature flag toggles, deployment triggers)
5. Maintain a clear audit trail of what was released and when

The current `kira release` command performs all operations in a single step, which can be risky for production releases. The two-step approach (`kira release plan` and `kira release confirm`) addresses this by:

- **Planning Phase**: Preview operations, generate release notes, and create a temporary plan file for review
- **Confirmation Phase**: Execute the release with git integration, script hooks, and cleanup

This approach is particularly useful for:
- Production releases requiring careful review before execution
- Teams that need to coordinate releases across multiple systems
- Workflows requiring feature flag toggles or deployment automation
- Maintaining clear separation between planning and execution

**Dependencies**: This command requires:
- Git repository with proper remote configuration
- Script execution capabilities for pre/post commit hooks
- GitHub API access (if scripts interact with external services)

## Requirements

### Core Functionality

#### Command Interface
- **Command Structure**: Two subcommands
  - `kira release plan [version] [status|path] [subfolder]` - Planning and preview phase
  - `kira release confirm` - Execution and confirmation phase
- **Arguments** (for `plan` subcommand):
  - `version` (optional) - Version string for the release (e.g., "v1.2.3", "2.0.0"). If not provided, uses date-based versioning
  - `status|path` (optional) - Status name (backlog, todo, doing, review, done) or direct path to folder, defaults to "done"
  - `subfolder` (optional) - Subfolder within the status folder to release
- **Flags**:
  - `--dry-run` - Preview operations without creating plan file (for `plan` subcommand)
  - `--force` - Skip validation checks (use with caution, for `confirm` subcommand)
  - `--no-scripts` - Skip pre/post commit script execution (for `confirm` subcommand)
  - `--no-git` - Skip git operations (commit, tag, push) - useful for testing (for `confirm` subcommand)

#### `kira release plan` - Planning Phase

##### Work Item Discovery
- Accept status name (e.g., "done") and resolve to corresponding folder via `status_folders` config
- Accept direct path (e.g., "4_done/v2") for flexible targeting
- Support subfolder specification for granular releases (e.g., "done v2")
- Validate source path exists before proceeding
- Recursively find all `.md` work item files in source path (excluding templates)
- Handle empty source paths gracefully with informative message
- Display list of work items that will be released with their IDs, titles, and current status

##### Release Notes Generation
- Scan all work items in source path for "# Release Notes" section
- Extract release notes content from each work item that has the section
- Include all content between "# Release Notes" header and next top-level header
- Combine release notes from multiple work items with blank line separators
- Handle work items without release notes sections (skip silently, but note in output)
- Preserve formatting and structure of release notes content
- Generate consolidated release notes with date header

##### Plan File Creation
- Create temporary plan file: `release-notes-plan.md` in project root
- File format:
  ```markdown
  # Release Plan - {date} {time}
  Version: {version}

  ## Work Items to Release
  - {id}: {title} (from {source_path})
  ...

  ## Release Notes
  {consolidated_release_notes}

  ## Operations to Perform
  1. Update work item statuses to "released"
  2. Archive work items to `.work/z_archive/{date}/{original-path}/`
  3. Update releases file: {releases_file_path}
  4. Run pre-commit scripts: {list_of_scripts}
  5. Commit changes with message: "{commit_message}"
  6. Create git tag: {tag_name}
  7. Push changes and tag to remote
  8. Run post-commit scripts: {list_of_scripts}
  9. Remove temporary plan file
  ```
- Include metadata: version (or date-based if not provided), date, time, source path, work item count, release notes preview
- Version determination:
  - If version provided: use as-is for tag name and commit message
  - If version not provided: use date-based format from config (e.g., "v{date}" or "release-{date}")
- Set appropriate file permissions (0o600)
- Display plan file location and instructions for review

##### Preview Display
- Show summary of operations that will be performed:
  - Version (or date-based version if not provided)
  - Number of work items to release
  - Source path
  - Archive destination
  - Releases file path
  - Pre-commit scripts (if configured)
  - Post-commit scripts (if configured)
  - Git operations (commit message, tag name, remote)
- Display release notes preview (first 500 characters with truncation indicator)
- Provide clear instructions: "Review release-notes-plan.md and run 'kira release confirm' to execute"

#### `kira release confirm` - Execution Phase

##### Plan File Validation
- Check for existence of `release-notes-plan.md` file
- Validate plan file format and content
- Parse plan file to extract work items, release notes, and operations
- Verify work items still exist and are in expected status
- Validate source path still exists
- Handle missing or invalid plan file with clear error message

##### Status Updates
- Update work item status field to "released" before archival
- Preserve all other front matter fields and content
- Update status in-place before archiving to maintain consistency
- Handle status update failures gracefully with clear error messages
- Update all work items listed in plan file

##### Archival Process
- Create archive directory structure: `.work/z_archive/{date}/{original-path}/`
- Use date format from config (default: "2006-01-02" format)
- Preserve folder structure by using `filepath.Base(sourcePath)` for archive subdirectory
- Copy work items to archive location with same filenames
- Include release notes in archive (create `RELEASE_NOTES.md` in archive directory)
- Set appropriate file permissions (0o600) for archived files
- Create archive directory with appropriate permissions (0o700)
- Handle archive creation failures with clear error messages

##### Releases File Management
- Read existing releases file if it exists (path from `release.releases_file` config, default: `RELEASES.md`)
- Prepend new release notes with date header: "# Release {date}"
- Format: `# Release {date}\n\n{release_notes}\n\n{existing_content}`
- Create releases file if it doesn't exist
- Skip releases file update if no release notes were generated (warn user)
- Write releases file with appropriate permissions (0o600)
- Handle file read/write errors gracefully

##### Pre-Commit Scripts
- Execute pre-commit scripts (if configured in `release.pre_commit_scripts`)
- Scripts run in project root directory
- Support both command strings and script file paths
- Execute scripts sequentially (stop on first failure unless configured otherwise)
- Pass environment variables: `KIRA_RELEASE_DATE`, `KIRA_RELEASE_TAG`, `KIRA_WORK_ITEMS`
- Capture and display script output
- Handle script failures with clear error messages
- Allow skipping scripts with `--no-scripts` flag

##### Git Operations
- Stage all changes: work item status updates, archive additions, releases file updates
- Create commit with formatted message (from config or default)
- Commit message format:
  - If version provided: `Release {version}: {work_item_count} work items` (configurable)
  - If version not provided: `Release {date}: {work_item_count} work items` (configurable)
- Create git tag with version/tag name:
  - If version provided: use version as tag name (e.g., "v1.2.3")
  - If version not provided: use date-based format from config (e.g., "v{date}" or "release-{date}")
- Tag message: Include release notes summary and version/date
- Push commit to remote (branch from `git.trunk_branch` config)
- Push tag to remote
- Handle git operation failures with clear error messages
- Allow skipping git operations with `--no-git` flag (for testing)

##### Post-Commit Scripts
- Execute post-commit scripts (if configured in `release.post_commit_scripts`)
- Scripts run in project root directory
- Support both command strings and script file paths
- Execute scripts sequentially (stop on first failure unless configured otherwise)
- Pass environment variables: `KIRA_RELEASE_DATE`, `KIRA_RELEASE_TAG`, `KIRA_WORK_ITEMS`, `KIRA_COMMIT_SHA`
- Capture and display script output
- Handle script failures with clear error messages (non-fatal, log warning)
- Allow skipping scripts with `--no-scripts` flag

##### File Cleanup
- Remove original work item files after successful archival and releases file update
- Remove temporary `release-notes-plan.md` file after successful execution
- Log warnings for removal failures but don't fail the entire operation
- Ensure all operations complete successfully before removing originals

### Configuration

#### kira.yml Extensions
```yaml
release:
  releases_file: RELEASES.md
  archive_date_format: "2006-01-02"
  plan_file: release-notes-plan.md
  commit_message: "Release {version}: {count} work items"  # {version} replaced with version or date
  commit_message_no_version: "Release {date}: {count} work items"  # Used when version not provided
  tag_format: "v{date}"  # Used when version not provided. Options: "v{date}", "release-{date}", or custom template
  tag_message: "Release {version}\n\n{release_notes_summary}"  # {version} replaced with version or date
  pre_commit_scripts:
    - "./scripts/pre-release.sh"
    - "echo 'Pre-release check complete'"
  post_commit_scripts:
    - "./scripts/post-release.sh"
    - "curl -X POST https://api.example.com/releases"
  script_fail_fast: true  # Stop on first script failure
  skip_git_on_error: false  # Skip git operations if scripts fail
```

#### Archive Structure
- Archive location: `.work/z_archive/{date}/{original-path}/`
- Date format: Configurable via `release.archive_date_format` (default: "2006-01-02")
- Original path: Base name of source path to preserve folder structure context
- Release notes: Included as `RELEASE_NOTES.md` in archive directory
- Example: `kira release plan done v2` → `.work/z_archive/2026-01-21/v2/`

### Error Handling

#### Validation Errors
- Invalid status: "invalid status: {status}" (when status name not found in config)
- Source path doesn't exist: "source path does not exist: {path}"
- No work items found: Informative message, exit successfully (not an error)
- Invalid version format: "invalid version format: {version}" (optional validation, warn but allow)
- Plan file missing: "release-notes-plan.md not found. Run 'kira release plan' first"
- Plan file invalid: "release-notes-plan.md is invalid or corrupted. Run 'kira release plan' again"

#### Processing Errors
- Failed to read work item: "failed to read work item: {error}"
- Failed to update status: "failed to update work item status: {error}"
- Failed to create archive: "failed to create archive directory: {error}"
- Failed to write archive: "failed to write to archive: {error}"
- Failed to read releases file: "failed to read releases file: {error}"
- Failed to write releases file: "failed to write releases file: {error}"
- Script execution failure: "pre-commit script failed: {script}: {error}"
- Git operation failure: "git {operation} failed: {error}"

#### Partial Failures
- Continue processing other work items if one fails (where possible)
- Provide clear error messages indicating which work item/operation failed
- Rollback capability: if critical operation fails, attempt to restore state
- Script failures: stop execution if `script_fail_fast` is true, otherwise continue with warnings

## Acceptance Criteria

### `kira release plan` Command
- [ ] `kira release plan` command exists and is executable
- [ ] Command accepts optional version parameter (e.g., "v1.2.3")
- [ ] Command defaults to date-based versioning when version not provided
- [ ] Command defaults to "done" status when no path arguments provided
- [ ] Command accepts status name (e.g., "done", "review") and resolves to folder
- [ ] Command accepts direct path (e.g., "4_done/v2") for flexible targeting
- [ ] Command accepts subfolder argument for granular releases
- [ ] Command validates source path exists before proceeding
- [ ] Command finds all work item files recursively in source path
- [ ] Command displays list of work items that will be released
- [ ] Command generates release notes from work items with "# Release Notes" sections
- [ ] Command creates `release-notes-plan.md` file with plan details including version
- [ ] Command uses provided version for tag name and commit message when specified
- [ ] Command falls back to date-based versioning when version not provided
- [ ] Command displays preview of operations that will be performed including version
- [ ] Command shows release notes preview in output
- [ ] Command provides clear instructions for next step

### `kira release confirm` Command
- [ ] `kira release confirm` command exists and is executable
- [ ] Command validates `release-notes-plan.md` file exists
- [ ] Command parses plan file and validates content
- [ ] Command verifies work items still exist and are in expected status
- [ ] Command updates work item statuses to "released"
- [ ] Command archives work items to correct location with date-based structure
- [ ] Command includes release notes in archive directory
- [ ] Command updates releases file with new release notes
- [ ] Command executes pre-commit scripts (if configured)
- [ ] Command commits changes with proper message
- [ ] Command creates git tag with proper name and message
- [ ] Command pushes commit and tag to remote
- [ ] Command executes post-commit scripts (if configured)
- [ ] Command removes original work item files after successful operations
- [ ] Command removes temporary plan file after successful execution

### Release Notes Generation
- [ ] Release notes are extracted from work items with "# Release Notes" sections
- [ ] Release notes include all content between header and next top-level header
- [ ] Multiple work items' release notes are combined with blank line separators
- [ ] Work items without release notes sections are noted in plan file
- [ ] Release notes formatting and structure are preserved
- [ ] Release notes are included in archive directory

### Plan File Management
- [ ] Plan file is created with correct format and content
- [ ] Plan file includes all work items, release notes, and operations
- [ ] Plan file has appropriate permissions (0o600)
- [ ] Plan file is validated before execution
- [ ] Plan file is removed after successful execution

### Script Execution
- [ ] Pre-commit scripts execute in correct order
- [ ] Post-commit scripts execute in correct order
- [ ] Scripts receive proper environment variables
- [ ] Script output is captured and displayed
- [ ] Script failures are handled according to configuration
- [ ] `--no-scripts` flag skips script execution
- [ ] Script paths and commands are validated

### Git Operations
- [ ] Changes are staged correctly
- [ ] Commit message follows configured format (uses version if provided, otherwise date)
- [ ] Git tag is created with proper name (uses version if provided, otherwise date-based format)
- [ ] Tag message includes version or date appropriately
- [ ] Commit and tag are pushed to remote
- [ ] Git operations handle errors gracefully
- [ ] `--no-git` flag skips git operations
- [ ] Git operations respect trunk branch and remote configuration
- [ ] Version from plan file is used for git operations during confirm

### Configuration Integration
- [ ] Respects `release.releases_file` from kira.yml
- [ ] Respects `release.archive_date_format` from kira.yml
- [ ] Respects `release.plan_file` from kira.yml
- [ ] Respects `release.commit_message` from kira.yml (with version substitution)
- [ ] Respects `release.commit_message_no_version` from kira.yml (when version not provided)
- [ ] Respects `release.tag_format` from kira.yml (when version not provided)
- [ ] Respects `release.tag_message` from kira.yml (with version substitution)
- [ ] Respects `release.pre_commit_scripts` from kira.yml
- [ ] Respects `release.post_commit_scripts` from kira.yml
- [ ] Respects `status_folders` mapping from kira.yml
- [ ] Uses default values when configuration is missing
- [ ] Version provided to plan command overrides date-based versioning

### Error Scenarios
- [ ] Invalid status name shows clear error message
- [ ] Non-existent source path shows clear error message
- [ ] Missing plan file shows clear error message with guidance
- [ ] Invalid plan file shows clear error message
- [ ] Work item read failures are handled gracefully
- [ ] Archive creation failures show clear error messages
- [ ] Releases file write failures show clear error messages
- [ ] Script execution failures are handled according to configuration
- [ ] Git operation failures show clear error messages
- [ ] Partial failures are handled appropriately

### User Experience
- [ ] Progress messages are clear and informative for both commands
- [ ] Success messages show summary of operations performed
- [ ] Error messages are actionable and helpful
- [ ] Command output is consistent and predictable
- [ ] Plan file is clearly documented and easy to review

## Implementation Notes

### Architecture

#### Command Structure
```
internal/commands/release.go
├── releaseCmd - Main cobra command with subcommands
├── releasePlanCmd - Plan subcommand
│   ├── planRelease() - Core planning logic
│   ├── discoverWorkItems() - Find work items to release
│   ├── generateReleaseNotes() - Extract release notes
│   ├── determineVersion() - Determine version (provided or date-based)
│   ├── createPlanFile() - Write plan file with version
│   └── displayPreview() - Show operations preview with version
├── releaseConfirmCmd - Confirm subcommand
│   ├── confirmRelease() - Core execution logic
│   ├── validatePlanFile() - Parse and validate plan (including version)
│   ├── executeRelease() - Execute all release operations
│   ├── runPreCommitScripts() - Execute pre-commit hooks
│   ├── performGitOperations() - Commit, tag, push (using version from plan)
│   └── runPostCommitScripts() - Execute post-commit hooks
└── (uses utils.go functions)
    ├── archiveWorkItems() - Archive work items
    ├── updateWorkItemStatus() - Update status to "released"
    └── getWorkItemFiles() - Find work item files
```

#### Dependencies
- Uses existing utility functions from `utils.go`
- Integrates with configuration system via `config.LoadConfig()`
- Leverages existing work item file discovery and status update utilities
- Uses standard library for file operations, path handling, and script execution
- Uses `executeCommand` utilities for git operations (similar to `save.go` and `start.go`)

### Plan File Format

#### Structure
```go
type ReleasePlan struct {
    Version     string  // Version string (provided or date-based)
    Date        string
    Time        string
    SourcePath  string
    WorkItems   []WorkItemInfo
    ReleaseNotes string
    Operations  []Operation
    Metadata    map[string]string
}

type WorkItemInfo struct {
    ID      string
    Title   string
    Path    string
    Status  string
}

type Operation struct {
    Type        string
    Description string
    Details     map[string]string
}
```

#### Serialization
- Use YAML format for plan file (easier to parse and human-readable)
- Include validation schema/version for future compatibility
- Store in project root as `release-notes-plan.md` (markdown with YAML front matter)

### Release Notes Extraction

#### Section Detection
```go
func generateReleaseNotes(workItems []string) (string, error) {
    var releaseNotes []string

    for _, workItem := range workItems {
        content, err := safeReadFile(workItem)
        if err != nil {
            return "", err
        }

        // Check if file has release notes section
        if strings.Contains(string(content), "# Release Notes") {
            // Extract release notes section
            lines := strings.Split(string(content), "\n")
            var inReleaseNotes bool
            var releaseNoteLines []string

            for _, line := range lines {
                if strings.Contains(line, "# Release Notes") {
                    inReleaseNotes = true
                    continue
                }
                if inReleaseNotes {
                    // Stop at next top-level header
                    if strings.HasPrefix(line, "#") && !strings.Contains(line, "Release Notes") {
                        break
                    }
                    releaseNoteLines = append(releaseNoteLines, line)
                }
            }

            if len(releaseNoteLines) > 0 {
                releaseNotes = append(releaseNotes, strings.Join(releaseNoteLines, "\n"))
            }
        }
    }

    return strings.Join(releaseNotes, "\n\n"), nil
}
```

### Script Execution

#### Pre/Post Commit Script Runner
```go
func runScripts(scripts []string, envVars map[string]string, failFast bool) error {
    for _, script := range scripts {
        // Determine if script is a command or file path
        var cmd *exec.Cmd
        if isScriptPath(script) {
            // Execute as script file
            cmd = exec.Command("bash", script)
        } else {
            // Execute as command string
            cmd = exec.Command("sh", "-c", script)
        }

        // Set environment variables
        cmd.Env = os.Environ()
        for k, v := range envVars {
            cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
        }

        // Capture output
        var stdout, stderr bytes.Buffer
        cmd.Stdout = &stdout
        cmd.Stderr = &stderr

        // Execute
        if err := cmd.Run(); err != nil {
            if failFast {
                return fmt.Errorf("script failed: %s: %w\nOutput: %s\nError: %s",
                    script, err, stdout.String(), stderr.String())
            } else {
                fmt.Printf("Warning: script failed: %s: %v\n", script, err)
            }
        } else {
            fmt.Printf("Script output: %s\n", stdout.String())
        }
    }
    return nil
}
```

### Version Determination

#### Version Resolution
```go
func determineVersion(providedVersion string, cfg *config.Config) (string, error) {
    if providedVersion != "" {
        // Validate version format (optional: check for valid semver or tag format)
        // Return provided version as-is
        return providedVersion, nil
    }

    // Use date-based versioning
    date := time.Now().Format(cfg.Release.ArchiveDateFormat)
    tagFormat := cfg.Release.TagFormat
    if tagFormat == "" {
        tagFormat = "v{date}" // default
    }

    // Replace {date} placeholder
    version := strings.ReplaceAll(tagFormat, "{date}", date)
    return version, nil
}
```

### Git Operations

#### Commit, Tag, and Push
```go
func performGitOperations(cfg *config.Config, version string, commitMessage, tagName, tagMessage string) error {
    // Stage changes
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if _, err := executeCommand(ctx, "git", []string{"add", ".work/", cfg.Release.ReleasesFile}, "", false); err != nil {
        return fmt.Errorf("failed to stage changes: %w", err)
    }

    // Format commit message with version
    formattedCommitMessage := formatCommitMessage(cfg, version, commitMessage)

    // Commit
    if _, err := executeCommand(ctx, "git", []string{"commit", "-m", formattedCommitMessage}, "", false); err != nil {
        return fmt.Errorf("failed to commit: %w", err)
    }

    // Create tag (tagName already determined from version)
    if _, err := executeCommand(ctx, "git", []string{"tag", "-a", tagName, "-m", tagMessage}, "", false); err != nil {
        return fmt.Errorf("failed to create tag: %w", err)
    }

    // Push commit
    trunkBranch := cfg.Git.TrunkBranch
    remote := cfg.Git.Remote
    if _, err := executeCommand(ctx, "git", []string{"push", remote, trunkBranch}, "", false); err != nil {
        return fmt.Errorf("failed to push commit: %w", err)
    }

    // Push tag
    if _, err := executeCommand(ctx, "git", []string{"push", remote, tagName}, "", false); err != nil {
        return fmt.Errorf("failed to push tag: %w", err)
    }

    return nil
}

func formatCommitMessage(cfg *config.Config, version string, template string) string {
    // Replace {version} placeholder with actual version
    message := strings.ReplaceAll(template, "{version}", version)
    // Replace {date} placeholder
    date := time.Now().Format(cfg.Release.ArchiveDateFormat)
    message = strings.ReplaceAll(message, "{date}", date)
    return message
}
```

### Archive Structure Enhancement

#### Include Release Notes in Archive
```go
func archiveWorkItems(workItems []string, sourcePath string, releaseNotes string) (string, error) {
    // Create archive directory
    date := time.Now().Format("2006-01-02")
    archiveDir := filepath.Join(".work", "z_archive", date, filepath.Base(sourcePath))

    if err := os.MkdirAll(archiveDir, 0o700); err != nil {
        return "", fmt.Errorf("failed to create archive directory: %w", err)
    }

    // Copy work items to archive
    for _, workItem := range workItems {
        filename := filepath.Base(workItem)
        archivePath := filepath.Join(archiveDir, filename)

        content, err := safeReadFile(workItem)
        if err != nil {
            return "", fmt.Errorf("failed to read work item: %w", err)
        }

        if err := os.WriteFile(archivePath, content, 0o600); err != nil {
            return "", fmt.Errorf("failed to write to archive: %w", err)
        }
    }

    // Write release notes to archive
    if releaseNotes != "" {
        releaseNotesPath := filepath.Join(archiveDir, "RELEASE_NOTES.md")
        releaseNotesContent := fmt.Sprintf("# Release Notes\n\n%s", releaseNotes)
        if err := os.WriteFile(releaseNotesPath, []byte(releaseNotesContent), 0o600); err != nil {
            return "", fmt.Errorf("failed to write release notes to archive: %w", err)
        }
    }

    return archiveDir, nil
}
```

### Error Handling Strategy

#### Graceful Degradation
- Continue processing other work items if one fails (where possible)
- Log warnings for non-critical failures (e.g., file removal, post-commit scripts)
- Provide clear error messages indicating which operation failed
- Ensure atomic operations where possible (all-or-nothing for critical steps)
- Rollback capability: if git operations fail after status updates, attempt to restore

#### Validation Order (for `confirm`)
1. Check plan file exists
2. Parse and validate plan file
3. Verify work items still exist
4. Check source path still exists
5. Validate git repository state
6. Execute operations in order
7. Clean up on success or partial failure

### Testing Strategy

#### Unit Tests
- Test plan file creation and parsing
- Test release notes extraction from various markdown formats
- Test path resolution (status names vs direct paths)
- Test script execution (mocked)
- Test git operations (mocked)
- Test error handling for various failure scenarios
- Mock file operations for isolated testing

#### Integration Tests
- Full workflow testing with test work items
- Test plan file creation and validation
- Test archive structure creation and file copying
- Test releases file creation and updates
- Test status updates and file removal
- Test script execution with test scripts
- Test git operations with test repository
- Test subfolder and path variations

#### E2E Tests
- `kira release plan` command in test environment
- `kira release confirm` command in test environment
- Verify complete two-step workflow end-to-end
- Test with various source path configurations
- Verify archive structure and releases file updates
- Test script execution and git operations
- Test error scenarios and edge cases

### Security Considerations

#### Path Validation
- All paths are validated to be within `.work/` directory using `validateWorkPath()`
- Releases file path is validated to be in project root using `safeReadProjectFile()`
- Plan file path is validated to be in project root
- No path traversal vulnerabilities (validated by existing utility functions)

#### Script Execution
- Validate script paths are within project directory
- Use safe command execution with timeouts
- Sanitize environment variables passed to scripts
- Limit script execution time to prevent hanging
- Validate script file permissions before execution

#### File Permissions
- Archive files: 0o600 (read/write for owner only)
- Archive directories: 0o700 (read/write/execute for owner only)
- Releases file: 0o600 (read/write for owner only)
- Plan file: 0o600 (read/write for owner only)

#### Input Validation
- Status names are validated against configuration
- Source paths are validated for existence
- File operations use safe read/write utilities
- Git commands use sanitized inputs
- Tag names are validated for git compatibility

## Implementation Strategy

The feature should be implemented incrementally with the following commit progression:

### Implementation Phases

#### Phase 1. Command Structure and Plan Subcommand
```
feat: add two-step release command structure

- Add releaseCmd with plan and confirm subcommands
- Implement basic plan subcommand skeleton
- Add command help and argument parsing
- Add placeholder for plan logic
- Update command routing
```

#### Phase 2. Version Handling and Work Item Discovery
```
feat: implement version handling and work item discovery

- Add version parameter parsing for plan command
- Implement version determination logic (provided vs date-based)
- Add work item discovery logic for plan command
- Implement release notes extraction from work items
- Add release notes combination and formatting
- Create work item list display
- Add unit tests for version determination
- Add unit tests for release notes extraction
```

#### Phase 3. Plan File Creation
```
feat: implement plan file creation with version

- Add plan file structure and serialization (including version field)
- Implement plan file writing with YAML format
- Add plan file metadata and operations list with version
- Create plan file validation structure
- Include version in plan file operations preview
- Add unit tests for plan file creation and parsing with version
```

#### Phase 4. Preview Display
```
feat: add preview display for release plan

- Implement operations preview display
- Add release notes preview with truncation
- Create summary output with work item count
- Add clear instructions for next step
- Format output for readability
```

#### Phase 5. Confirm Subcommand and Plan Validation
```
feat: implement confirm subcommand with plan validation

- Add confirm subcommand skeleton
- Implement plan file existence checking
- Add plan file parsing and validation
- Verify work items still exist
- Add unit tests for plan validation
```

#### Phase 6. Status Updates and Archival
```
feat: add status updates and archival for confirm

- Implement work item status updates to "released"
- Add archival with release notes inclusion
- Update releases file management
- Add file cleanup logic
- Add unit tests for archival operations
```

#### Phase 7. Script Execution System
```
feat: implement pre/post commit script execution

- Add script execution infrastructure
- Implement pre-commit script runner
- Implement post-commit script runner
- Add environment variable passing
- Add script failure handling
- Add unit tests for script execution
```

#### Phase 8. Git Operations with Version
```
feat: add git operations for release confirm with version support

- Implement git staging and commit
- Add version-aware commit message formatting
- Add git tag creation with version (from plan file)
- Implement git push for commit and tag
- Add commit message and tag name formatting (version vs date-based)
- Handle version substitution in commit and tag messages
- Add unit tests for git operations with version (mocked)
```

#### Phase 9. Configuration Integration
```
feat: integrate release configuration settings

- Add release configuration schema
- Respect commit message, tag format from config
- Add script configuration support
- Integrate with existing config system
- Add configuration validation
- Add unit tests for configuration parsing
```

#### Phase 10. Error Handling and User Experience
```
feat: enhance error handling and UX

- Add comprehensive error messages
- Implement rollback for partial failures
- Add progress messages for all operations
- Improve success/error output formatting
- Add dry-run support for plan command
- Add flags for skipping operations
```

#### Phase 11. Integration Tests
```
test: add integration tests for release workflow

- Test full two-step workflow
- Test plan file creation and validation
- Test script execution with test scripts
- Test git operations with test repository
- Test error recovery scenarios
```

#### Phase 12. E2E Tests
```
test: add e2e tests for release commands

- Test kira release plan in test environment
- Test kira release confirm in test environment
- Verify complete workflow end-to-end
- Test with various configurations
- Test error scenarios
```

### Benefits of This Approach

- **Incremental Development**: Each commit adds working functionality
- **Easy Review**: Smaller, focused changes are easier to review
- **Quick Feedback**: Issues can be caught early in smaller chunks
- **Logical Dependencies**: Each commit builds on the previous ones
- **Test Coverage**: Testing is added alongside implementation
- **Revert Safety**: Issues can be isolated to specific commits
- **Two-Step Safety**: Planning phase allows review before execution

## Release Notes

### New Features
- **Two-Step Release Process**: `kira release plan` and `kira release confirm` commands for safe, controlled releases
- **Planning Phase**: Preview operations, generate release notes, and create reviewable plan file
- **Confirmation Phase**: Execute releases with git integration, script hooks, and proper cleanup
- **Script Hooks**: Pre-commit and post-commit script execution for automation (feature flags, deployments, etc.)
- **Git Integration**: Automatic commit, tag creation, and push to remote repository
- **Enhanced Archival**: Release notes included in archive directory for historical reference
- **Plan File Review**: Human-readable plan file for review before execution

### Workflow
1. Work items are completed and moved to "done" status
2. Work items include "# Release Notes" sections with release content
3. `kira release plan` is run to preview and generate release plan
4. Plan file (`release-notes-plan.md`) is reviewed
5. `kira release confirm` is run to execute the release
6. Pre-commit scripts execute (e.g., toggle feature flags)
7. Work items are marked as "released" and archived
8. Release notes are prepended to centralized releases file
9. Changes are committed and tagged in git
10. Commit and tag are pushed to remote
11. Post-commit scripts execute (e.g., trigger deployments)
12. Temporary plan file is removed

### Usage Examples
```bash
# Plan a release with version from done folder
kira release plan v1.2.3

# Plan a release with version from specific subfolder
kira release plan v2.0.0 done v2

# Plan a release without version (uses date-based)
kira release plan

# Plan a release without version from specific path
kira release plan 4_done/v2

# Review the generated release-notes-plan.md file
cat release-notes-plan.md

# Confirm and execute the release
kira release confirm

# Skip scripts during confirmation (for testing)
kira release confirm --no-scripts

# Skip git operations (for testing)
kira release confirm --no-git
```

### Configuration Example
```yaml
release:
  releases_file: RELEASES.md
  archive_date_format: "2006-01-02"
  plan_file: release-notes-plan.md
  commit_message: "Release {version}: {count} work items"  # {version} replaced with version or date
  commit_message_no_version: "Release {date}: {count} work items"  # Used when version not provided
  tag_format: "v{date}"  # Used when version not provided
  tag_message: "Release {version}\n\n{release_notes_summary}"  # {version} replaced with version or date
  pre_commit_scripts:
    - "./scripts/toggle-feature-flag.sh production enable"
  post_commit_scripts:
    - "./scripts/notify-deployment.sh"
    - "curl -X POST https://api.example.com/releases"
```

### Version Handling
- **With Version**: When version is provided (e.g., `kira release plan v1.2.3`), it is used directly for:
  - Git tag name: `v1.2.3`
  - Commit message: `Release v1.2.3: 5 work items`
  - Tag message: `Release v1.2.3\n\n{release_notes_summary}`

- **Without Version**: When version is not provided (e.g., `kira release plan`), date-based versioning is used:
  - Git tag name: `v2026-01-21` (from `tag_format` config)
  - Commit message: `Release 2026-01-21: 5 work items` (from `commit_message_no_version` config)
  - Tag message: `Release 2026-01-21\n\n{release_notes_summary}`
