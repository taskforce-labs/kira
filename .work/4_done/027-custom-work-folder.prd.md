---
id: 027
title: custom work folder
status: done
kind: prd
assigned: 
estimate: 0
created: 2026-01-27
due: 2026-01-27
tags: []
---

# custom work folder

A way to configure the folder kira looks in to find work items, allowing users to customize the work directory location instead of being limited to the default `.work` folder.

## Overview

Currently, kira hardcodes `.work` as the directory where all work items, status folders, templates, and IDEAS.md are stored. This PRD adds the ability to configure a custom work folder path through `kira.yml`, enabling users to:
- Use alternative folder names (e.g., `work`, `tasks`, `issues`)
- Organize work items in a different location relative to the project root
- Maintain compatibility with existing workflows that may use different naming conventions
- Support monorepo or polyrepo setups where work items might be stored in a shared location

**Design Philosophy:** The default should remain `.work` for backward compatibility, but users should be able to override this via configuration. All path references throughout the codebase should use a centralized function to resolve the work folder path.

## Context

### Current State

Kira currently hardcodes `.work` in numerous places:
- `internal/commands/utils.go`: `validateWorkPath()` checks paths are within `.work/`
- `internal/commands/root.go`: Checks for `.work` directory existence
- `internal/commands/init.go`: Creates `.work` directory structure
- `internal/commands/new.go`: Creates work items in `.work/{status_folder}/`
- `internal/validation/validator.go`: Walks `.work` directory for validation
- `internal/commands/start.go`: References `.work` for worktree operations
- Many test files: Create and reference `.work` directories

### Use Cases

1. **Alternative Naming**: Teams may prefer `work`, `tasks`, `issues`, or other folder names
2. **Monorepo Organization**: In monorepos, work items might be stored at a shared root level (e.g., `../work/` relative to individual projects)
3. **Integration with Existing Tools**: Some teams may already have a `work/` or `tasks/` folder structure
4. **Polyrepo Workspaces**: When using kira's workspace features, different projects might need different work folder locations
5. **Hidden vs Visible**: Some teams prefer visible folders (e.g., `work/`) over hidden ones (`.work/`)

### Related Features

- **Workspace Configuration** (PRD 024): Workspace config already has `root` and `worktree_root` settings, but doesn't configure the work folder name
- **Polyrepo Support**: Custom work folders could be useful when managing multiple repositories with different conventions

## Requirements

### Functional Requirements

#### FR1: Configuration Support
- Add `workspace.work_folder` field to `kira.yml` configuration
- Default value: `.work` (maintains backward compatibility)
- Accept relative paths (e.g., `work`, `tasks`, `../shared-work`)
- Accept absolute paths (e.g., `/path/to/work`)
- Validate path doesn't contain path traversal attacks (`..` in unsafe ways)
- Path should be resolved relative to the repository root (where `kira.yml` is located)

#### FR2: Centralized Path Resolution
- Create a centralized function `GetWorkFolderPath()` or similar in `internal/config` package
- All code that references `.work` should use this function instead
- Function should:
  - Load config if not already loaded
  - Return configured work folder path or default `.work`
  - Resolve relative paths to absolute paths for validation
  - Cache the resolved path to avoid repeated config loading

#### FR3: Path Validation Updates
- Update `validateWorkPath()` in `internal/commands/utils.go` to use configured work folder
- Update `safeReadFile()` to validate against configured work folder
- Update `findWorkItemFile()` to search in configured work folder
- Update `archiveWorkItems()` to use configured work folder for archive paths
- Ensure all path validation functions use the configured work folder

#### FR4: Command Updates
- **`kira init`**: Create work folder structure in configured location
  - If `--force` is used, should respect configured work folder
  - Should create all status folders in configured location
- **`kira new`**: Create work items in configured work folder
- **`kira move`**: Move items within configured work folder structure
- **`kira save`**: Commit changes from configured work folder
- **`kira lint`**: Validate work items in configured work folder
- **`kira doctor`**: Operate on configured work folder
- **`kira start`**: Use configured work folder for worktree operations
- **`kira latest`**: Search in configured work folder
- **`kira idea`**: Read/write IDEAS.md in configured work folder

#### FR5: Workspace Detection
- Update `root.go` workspace detection to check for configured work folder
- If configured work folder doesn't exist, provide clear error message
- Error message should indicate the configured path, not hardcoded `.work`

#### FR6: Template Resolution
- Update template path resolution to use configured work folder
- Templates should be located in `{work_folder}/templates/`
- Template paths in config should be relative to work folder or absolute

#### FR7: Archive Path Resolution
- Archive paths should use configured work folder: `{work_folder}/z_archive/{date}/...`
- Archive operations should respect configured work folder

### Non-Functional Requirements

#### NFR1: Backward Compatibility
- Existing workspaces with no `workspace.work_folder` config should continue using `.work`
- No breaking changes for existing users
- Migration should be optional (users can continue using `.work`)

#### NFR2: Performance
- Path resolution should be cached to avoid repeated config file reads
- No significant performance impact from using configured paths

#### NFR3: Security
- Path validation must prevent directory traversal attacks
- Absolute paths should be validated to ensure they're within reasonable bounds
- Relative paths should be resolved and validated before use

#### NFR4: Error Messages
- Error messages should reference the configured work folder path, not hardcoded `.work`
- Clear error messages when configured work folder doesn't exist
- Helpful suggestions when path resolution fails

### Configuration Schema

```yaml
workspace:
  work_folder: ".work"  # default, can be overridden
  # ... other workspace settings
```

**Examples:**
```yaml
# Use visible folder instead of hidden
workspace:
  work_folder: "work"

# Use different name
workspace:
  work_folder: "tasks"

# Use shared location (monorepo)
workspace:
  work_folder: "../shared-work"

# Absolute path (less common, but supported)
workspace:
  work_folder: "/path/to/work"
```

## Acceptance Criteria

### AC1: Configuration Loading
- [ ] `kira.yml` can specify `workspace.work_folder` field
- [ ] Default value is `.work` when not specified
- [ ] Config validation accepts valid relative and absolute paths
- [ ] Config validation rejects paths with unsafe traversal patterns

### AC2: Path Resolution
- [ ] `GetWorkFolderPath()` function exists in `internal/config` package
- [ ] Function returns configured path or default `.work`
- [ ] Function resolves relative paths correctly
- [ ] Function caches resolved path for performance

### AC3: Code Updates
- [ ] All hardcoded `.work` references replaced with config-based resolution
- [ ] `validateWorkPath()` uses configured work folder
- [ ] `findWorkItemFile()` searches in configured work folder
- [ ] `archiveWorkItems()` uses configured work folder
- [ ] All commands respect configured work folder

### AC4: Command Functionality
- [ ] `kira init` creates structure in configured work folder
- [ ] `kira new` creates items in configured work folder
- [ ] `kira move` operates within configured work folder
- [ ] `kira save` commits from configured work folder
- [ ] `kira lint` validates items in configured work folder
- [ ] `kira doctor` operates on configured work folder
- [ ] `kira start` uses configured work folder for worktrees
- [ ] `kira latest` searches in configured work folder
- [ ] `kira idea` reads/writes IDEAS.md in configured work folder

### AC5: Workspace Detection
- [ ] `kira` commands detect workspace using configured work folder
- [ ] Error messages reference configured path, not `.work`
- [ ] Clear error when configured work folder doesn't exist

### AC6: Backward Compatibility
- [ ] Existing workspaces without config continue using `.work`
- [ ] No breaking changes for existing users
- [ ] All existing tests pass with default `.work` behavior

### AC7: Testing
- [ ] Unit tests for path resolution function
- [ ] Unit tests for config validation
- [ ] Integration tests with custom work folder
- [ ] Tests verify backward compatibility
- [ ] Tests cover relative and absolute paths
- [ ] Tests verify path traversal protection

### AC8: Documentation
- [ ] README.md documents `workspace.work_folder` configuration
- [ ] Configuration examples show custom work folder usage
- [ ] Migration guide (if needed) explains how to change work folder
- [ ] Error messages are clear and helpful

## Implementation Notes

### Phase 1: Configuration Support

1. **Add Config Field**
   - Add `WorkFolder string` to `WorkspaceConfig` struct in `internal/config/config.go`
   - Default to `.work` in `mergeWithDefaults()`
   - Add validation in `validateConfig()` to ensure path is safe

2. **Path Resolution Function**
   ```go
   // GetWorkFolderPath returns the configured work folder path, defaulting to ".work"
   func GetWorkFolderPath(config *Config) string {
     if config.Workspace != nil && config.Workspace.WorkFolder != "" {
       return config.Workspace.WorkFolder
     }
     return ".work"
   }
   
   // GetWorkFolderAbsPath returns the absolute path to the work folder
   func GetWorkFolderAbsPath(config *Config) (string, error) {
     workFolder := GetWorkFolderPath(config)
     absPath, err := filepath.Abs(workFolder)
     if err != nil {
       return "", fmt.Errorf("failed to resolve work folder path: %w", err)
     }
     return absPath, nil
   }
   ```

3. **Path Validation**
   - Validate that `work_folder` doesn't contain unsafe `..` patterns
   - For relative paths, ensure they resolve within reasonable bounds
   - For absolute paths, consider if any restrictions are needed

### Phase 2: Update Core Utilities

1. **Update `validateWorkPath()`**
   ```go
   func validateWorkPath(path string, config *Config) error {
     workFolder, err := GetWorkFolderAbsPath(config)
     if err != nil {
       return err
     }
     // ... rest of validation using workFolder instead of ".work"
   }
   ```

2. **Update `findWorkItemFile()`**
   - Use `GetWorkFolderPath(config)` instead of hardcoded `.work`
   - Pass config through function calls

3. **Update `archiveWorkItems()`**
   - Use configured work folder for archive directory paths

### Phase 3: Update Commands

1. **Update `root.go`**
   - Check for configured work folder instead of hardcoded `.work`
   - Update error messages to reference configured path

2. **Update `init.go`**
   - Create work folder structure in configured location
   - Use `GetWorkFolderPath()` for all path construction

3. **Update `new.go`**
   - Create work items in configured work folder
   - Resolve template paths relative to configured work folder

4. **Update All Other Commands**
   - `move.go`, `save.go`, `lint.go`, `doctor.go`, `start.go`, `latest.go`, `idea.go`
   - Replace all `.work` references with config-based resolution

### Phase 4: Testing

1. **Unit Tests**
   - Test `GetWorkFolderPath()` with various configs
   - Test path validation with safe and unsafe paths
   - Test backward compatibility (no config = `.work`)

2. **Integration Tests**
   - Test `kira init` with custom work folder
   - Test `kira new` with custom work folder
   - Test `kira move` with custom work folder
   - Test all commands with custom work folder
   - Test workspace detection with custom work folder

3. **Update Existing Tests**
   - Many tests create `.work` directories - these should continue to work
   - Consider adding test helpers that use configured work folder
   - Some tests may need to be updated to use config-based paths

### Phase 5: Documentation

1. **Update README.md**
   - Document `workspace.work_folder` configuration option
   - Provide examples of different work folder configurations
   - Explain use cases (monorepo, alternative naming, etc.)

2. **Update Configuration Documentation**
   - Add `workspace.work_folder` to config schema documentation
   - Explain default behavior and backward compatibility

3. **Error Message Improvements**
   - Ensure all error messages reference configured path
   - Add helpful suggestions when work folder doesn't exist

### Technical Considerations

1. **Config Loading**
   - Config is loaded in `root.go` before command execution
   - Need to ensure config is available to all utility functions
   - Consider passing config explicitly vs. loading it in each function

2. **Path Resolution**
   - Relative paths should be resolved relative to repository root (where `kira.yml` is)
   - Absolute paths should be validated for security
   - Consider symlink handling

3. **Performance**
   - Cache resolved work folder path to avoid repeated resolution
   - Config is typically loaded once per command execution

4. **Migration**
   - No migration needed - existing workspaces continue using `.work`
   - Users can optionally add `workspace.work_folder` to their config
   - Consider a `kira migrate` command in the future if needed

5. **Edge Cases**
   - What if work folder is outside repository? (probably should allow for monorepo cases)
   - What if work folder is a symlink? (should follow symlink)
   - What if work folder doesn't exist? (error with helpful message)

### Code Locations to Update

**High Priority (Core Functionality):**
- `internal/config/config.go`: Add config field and resolution functions
- `internal/commands/utils.go`: Update path validation and utility functions
- `internal/commands/root.go`: Update workspace detection
- `internal/validation/validator.go`: Update validation to use configured path

**Medium Priority (Commands):**
- `internal/commands/init.go`: Create structure in configured location
- `internal/commands/new.go`: Create items in configured location
- `internal/commands/move.go`: Move items within configured structure
- `internal/commands/save.go`: Commit from configured location
- `internal/commands/lint.go`: Validate items in configured location
- `internal/commands/doctor.go`: Operate on configured location
- `internal/commands/start.go`: Use configured location for worktrees
- `internal/commands/latest.go`: Search in configured location
- `internal/commands/idea.go`: Read/write IDEAS.md in configured location

**Lower Priority (Tests):**
- Update test files to use config-based paths where appropriate
- Add tests for custom work folder functionality
- Ensure backward compatibility tests pass

## Release Notes

- Added `workspace.work_folder` configuration option to `kira.yml` for customizing the work directory location
- Default behavior remains `.work` for backward compatibility
- All kira commands now respect the configured work folder path
- Supports relative paths (e.g., `work`, `tasks`) and absolute paths
- Path validation ensures security and prevents directory traversal attacks
- Error messages now reference the configured work folder path instead of hardcoded `.work`
- Enables better integration with monorepo setups and alternative folder naming conventions

