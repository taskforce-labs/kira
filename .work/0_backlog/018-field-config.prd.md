---
id: 018
title: field-config
status: backlog
kind: prd
assigned:
estimate: 0
created: 2026-01-22
due: 2026-01-22
tags: [configuration, validation, fields]
---

# field-config

A configuration system that allows users to define custom fields for work items, including field types, validation rules, default values, and field metadata. This enables teams to customize work items to match their specific workflow needs while maintaining validation and type safety.

## Context

Currently, Kira's validation system is hardcoded to support only a fixed set of fields (`id`, `title`, `status`, `kind`, `created`) with limited validation rules. The validation logic uses hardcoded switch statements and pattern matching (e.g., date fields are detected by checking if the field name contains "date" or "due").

**Important Constraint**: The `id`, `title`, `status`, `kind`, and `created` fields remain hardcoded and cannot be configured. These fields are fundamental to the work item system and are already managed by other configuration:
- `id` and `title`: Core work item identifiers with hardcoded validation
- `status`: Managed by `status_folders` and `status_values` in validation config
- `kind`: Managed by templates configuration
- `created`: Core metadata field with hardcoded date validation

Only custom fields can be configured through the field configuration system.

This limitation creates several problems:

1. **Inflexible Validation**: Teams cannot define custom fields with specific validation rules (e.g., email format, URL format, enum values, date ranges)
2. **No Type Safety**: There's no way to specify that a field should be a date, email, array, or other type
3. **Hardcoded Logic**: Date validation relies on naming conventions rather than explicit configuration
4. **Limited Extensibility**: Commands like `kira assign` support custom fields via `--field`, but there's no validation or configuration for what fields are valid
5. **No Field Metadata**: There's no way to document fields, provide descriptions, or set defaults

The `kira assign` command (PRD 017) already supports custom fields, but without field configuration, there's no way to:
- Validate that a field name is valid
- Ensure field values match expected types or formats
- Provide defaults for fields
- Document available fields

This feature will enable teams to:
- Define custom fields that match their workflow (e.g., `assignee`, `reviewer`, `priority`, `sprint`, `epic`)
- Configure validation rules for each field (format, allowed values, required conditions)
- Set default values for fields
- Document fields with descriptions
- Ensure type safety across all Kira commands

**Dependencies**: This feature builds on:
- Existing validation system (`internal/validation/validator.go`)
- Configuration system (`internal/config/config.go`)
- Commands that manipulate work item fields (e.g., `kira assign`, `kira new`)

## Requirements

### Core Functionality

#### Field Definition Schema

Fields are defined in `kira.yml` under a new `fields` configuration section. Each field definition includes:

- **Name**: Field identifier (e.g., `assigned`, `reviewer`, `priority`)
- **Type**: Field data type (`string`, `date`, `email`, `url`, `number`, `array`, `enum`)
- **Required**: Whether the field is required (boolean or conditional)
- **Default**: Default value for the field
- **Format**: Validation format (regex pattern, date format, etc.)
- **Allowed Values**: For enum types, list of allowed values
- **Description**: Human-readable description of the field
- **Validation Rules**: Custom validation rules (min/max length, date ranges, etc.)

#### Field Types

Support the following field types with appropriate validation:

1. **string**: Plain text field
   - Optional format validation (regex pattern)
   - Optional min/max length constraints

2. **date**: Date field (YYYY-MM-DD format)
   - Optional date range validation (min/max dates)
   - Optional relative date validation (e.g., must be in future)

3. **email**: Email address field
   - Email format validation
   - Optional domain whitelist/blacklist

4. **url**: URL field
   - URL format validation
   - Optional scheme restrictions (http/https only)

5. **number**: Numeric field
   - Optional min/max value constraints
   - Integer or float support

6. **array**: Array/list field
   - Array of strings, numbers, or enums
   - Optional min/max length constraints
   - Optional unique value constraint

7. **enum**: Enumeration field
   - List of allowed string values
   - Case-sensitive or case-insensitive matching

#### Configuration Schema

**Note**: `id`, `title`, `status`, `kind`, and `created` fields are hardcoded and cannot be configured. They are managed by other configuration systems. Only custom fields can be configured.

```yaml
fields:
  # Only custom fields can be configured
  # id, title, status, kind, and created are hardcoded and managed by other config

  # Custom fields
  assigned:
    type: email
    required: false
    description: "Assigned user email address"
    default: ""

  reviewer:
    type: email
    required: false
    description: "Reviewer email address"

  priority:
    type: enum
    required: false
    allowed_values:
      - low
      - medium
      - high
      - critical
    default: medium
    description: "Priority level"

  due:
    type: date
    required: false
    format: "2006-01-02"
    min_date: "today"  # Relative date: must be today or future
    description: "Due date"

  tags:
    type: array
    required: false
    item_type: string
    unique: true
    description: "Tags for categorization"

  estimate:
    type: number
    required: false
    min: 0
    max: 100
    description: "Estimate in days"

  epic:
    type: string
    required: false
    format: ^[A-Z]+-\d+$  # e.g., EPIC-001
    description: "Epic identifier"

  sprint:
    type: string
    required: false
    description: "Sprint identifier"

  url:
    type: url
    required: false
    schemes: [http, https]
    description: "Related URL"
```

#### Field Validation

Validation should occur:

1. **During Work Item Creation**: When creating new work items via `kira new`
2. **During Work Item Updates**: When updating work items via `kira assign`, `kira move`, or direct file edits
3. **During Validation**: When running `kira lint` or `kira doctor`
4. **During Import**: When importing or promoting work items (e.g., from roadmap)

Validation rules:

- **Required Fields**: Check that required fields are present and non-empty
- **Type Validation**: Validate that field values match the declared type
- **Format Validation**: Apply regex patterns for string fields, date formats for date fields
- **Enum Validation**: Ensure enum fields only contain allowed values
- **Range Validation**: Check min/max constraints for numbers, dates, and array lengths
- **Email/URL Validation**: Validate email and URL formats
- **Array Validation**: Validate array item types and constraints

#### Default Values

Fields can have default values that are applied:

- When creating new work items via `kira new`
- When fields are missing from existing work items (optional, configurable)
- Defaults can be static values or dynamic (e.g., current date for date fields)

#### Field Metadata

Fields can include metadata for documentation and tooling:

- **Description**: Human-readable description shown in help text
- **Display Name**: Alternative display name for UI/tooling
- **Category**: Optional grouping of related fields
- **Deprecated**: Mark fields as deprecated with migration guidance

### Integration with Existing Commands

#### `kira new` Command

- Use field configuration to:
  - Generate prompts for required fields
  - Apply default values
  - Validate input during creation
  - Show field descriptions in help text

#### `kira assign` Command

- Use field configuration to:
  - Validate that `--field` specifies a valid field name
  - Validate that assigned values match field type (e.g., email for `assigned` field)
  - Show available fields when field name is invalid
  - Apply type-specific logic (e.g., array handling for array fields)

#### `kira lint` Command

- Use field configuration to:
  - Validate all fields against their definitions
  - Report type mismatches
  - Report format violations
  - Report missing required fields
  - Report invalid enum values

#### `kira doctor` Command

- Use field configuration to:
  - Suggest fixes for invalid field values
  - Auto-fix common issues (e.g., date format corrections)
  - Add missing required fields with defaults
  - Remove invalid fields (with confirmation)

### Backward Compatibility

- **Existing Work Items**: Must continue to work without field configuration
- **Default Configuration**: If no field configuration is provided, use existing hardcoded validation
- **Migration Path**: Provide migration guidance for teams adopting field configuration
- **Gradual Adoption**: Teams can define fields incrementally without breaking existing work items

### Configuration Validation

- Validate field configuration itself:
  - Reject attempts to configure `id`, `title`, `status`, `kind`, or `created` fields (show clear error)
  - Check for duplicate field names
  - Validate type names
  - Validate format patterns (regex syntax)
  - Validate enum value lists
  - Validate date format strings
  - Check for circular dependencies in conditional requirements

## Acceptance Criteria

### Configuration

- [ ] Field configuration can be defined in `kira.yml` under `fields` section
- [ ] `id`, `title`, `status`, `kind`, and `created` fields remain hardcoded and cannot be configured
- [ ] Attempts to configure hardcoded fields (`id`, `title`, `status`, `kind`, `created`) are rejected with clear error
- [ ] Custom fields can be defined with any of the supported types
- [ ] Field configuration is validated for syntax errors and conflicts
- [ ] Invalid field configuration shows clear error messages

### Field Types

- [ ] `string` type fields support format validation (regex)
- [ ] `string` type fields support min/max length constraints
- [ ] `date` type fields validate YYYY-MM-DD format
- [ ] `date` type fields support min/max date constraints
- [ ] `date` type fields support relative dates (e.g., "today", "future")
- [ ] `email` type fields validate email format
- [ ] `url` type fields validate URL format
- [ ] `url` type fields support scheme restrictions
- [ ] `number` type fields validate numeric values
- [ ] `number` type fields support min/max value constraints
- [ ] `array` type fields validate array structure
- [ ] `array` type fields support item type validation
- [ ] `array` type fields support unique value constraint
- [ ] `enum` type fields validate against allowed values list
- [ ] `enum` type fields support case-sensitive and case-insensitive matching

### Validation

- [ ] Required fields are validated during work item creation
- [ ] Required fields are validated during work item updates
- [ ] Required fields are validated during `kira lint`
- [ ] Type mismatches are detected and reported
- [ ] Format violations are detected and reported
- [ ] Enum value violations are detected and reported
- [ ] Range violations (min/max) are detected and reported
- [ ] Validation errors show clear messages with field name and expected format

### Default Values

- [ ] Default values are applied when creating new work items
- [ ] Default values respect field types
- [ ] Default values can be static strings, numbers, dates, or arrays
- [ ] Default values are documented in field descriptions

### Command Integration

- [ ] `kira new` uses field configuration for prompts and validation
- [ ] `kira new` applies default values for fields
- [ ] `kira assign` validates field names against configuration
- [ ] `kira assign` validates field values against field types
- [ ] `kira assign` shows helpful error when field name is invalid
- [ ] `kira lint` validates all fields against configuration
- [ ] `kira lint` reports all validation errors clearly
- [ ] `kira doctor` can fix field validation issues
- [ ] `kira doctor` can add missing required fields with defaults

### Backward Compatibility

- [ ] Existing work items without field configuration continue to work
- [ ] Default validation behavior is preserved when no field configuration exists
- [ ] `id`, `title`, `status`, `kind`, and `created` fields always use hardcoded validation regardless of configuration
- [ ] Migration from hardcoded validation to field configuration is seamless
- [ ] Teams can adopt field configuration incrementally

### Error Handling

- [ ] Invalid field configuration shows clear error messages
- [ ] Attempts to configure `id`, `title`, `status`, `kind`, or `created` show clear error: "Fields 'id', 'title', 'status', 'kind', and 'created' cannot be configured and must use hardcoded validation"
- [ ] Field validation errors show field name and expected format
- [ ] Missing field configuration falls back to default behavior
- [ ] Type conversion errors are handled gracefully
- [ ] Format validation errors include examples of valid formats

### Documentation

- [ ] Field descriptions are accessible via help text
- [ ] Field configuration is documented in README or docs
- [ ] Examples of field configuration are provided
- [ ] Migration guide is provided for teams adopting field configuration

## Implementation Notes

### Architecture

#### Configuration Structure

**Important**: The validation system must reject any attempt to configure `id`, `title`, `status`, `kind`, or `created` fields. These fields are always validated using hardcoded logic and are managed by other configuration systems.

```go
// internal/config/config.go

type FieldConfig struct {
    Type         string            `yaml:"type"`          // string, date, email, url, number, array, enum
    Required     interface{}       `yaml:"required"`     // bool or conditional expression
    Default      interface{}       `yaml:"default"`       // default value
    Format       string            `yaml:"format"`       // regex pattern or date format
    AllowedValues []string         `yaml:"allowed_values"` // for enum type
    Description  string            `yaml:"description"`
    DisplayName  string            `yaml:"display_name"`
    Category     string            `yaml:"category"`
    Deprecated   bool              `yaml:"deprecated"`
    MinLength    *int              `yaml:"min_length"`   // for strings/arrays
    MaxLength    *int              `yaml:"max_length"`   // for strings/arrays
    MinValue     *float64          `yaml:"min"`          // for numbers
    MaxValue     *float64          `yaml:"max"`          // for numbers
    MinDate      string            `yaml:"min_date"`      // for dates (absolute or relative)
    MaxDate      string            `yaml:"max_date"`      // for dates (absolute or relative)
    ItemType     string            `yaml:"item_type"`     // for arrays
    Unique       bool              `yaml:"unique"`        // for arrays
    Schemes      []string          `yaml:"schemes"`       // for URLs
    CaseSensitive bool             `yaml:"case_sensitive"` // for enums
}

type Config struct {
    // ... existing fields ...
    Fields       map[string]FieldConfig `yaml:"fields"`
}
```

#### Validation System Enhancement

```go
// internal/validation/validator.go

func validateFieldValue(fieldName string, value interface{}, fieldConfig *config.FieldConfig) error {
    // Type validation
    if err := validateFieldType(value, fieldConfig.Type); err != nil {
        return fmt.Errorf("field '%s': %w", fieldName, err)
    }

    // Format validation
    if fieldConfig.Format != "" {
        if err := validateFieldFormat(value, fieldConfig.Format, fieldConfig.Type); err != nil {
            return fmt.Errorf("field '%s': %w", fieldName, err)
        }
    }

    // Enum validation
    if fieldConfig.Type == "enum" {
        if err := validateEnumValue(value, fieldConfig.AllowedValues, fieldConfig.CaseSensitive); err != nil {
            return fmt.Errorf("field '%s': %w", fieldName, err)
        }
    }

    // Range validation
    if err := validateFieldRange(value, fieldConfig); err != nil {
        return fmt.Errorf("field '%s': %w", fieldName, err)
    }

    return nil
}

func validateFieldType(value interface{}, fieldType string) error {
    switch fieldType {
    case "string":
        if _, ok := value.(string); !ok {
            return fmt.Errorf("expected string, got %T", value)
        }
    case "date":
        if str, ok := value.(string); ok {
            if _, err := time.Parse("2006-01-02", str); err != nil {
                return fmt.Errorf("invalid date format: %s", str)
            }
        } else {
            return fmt.Errorf("expected date string, got %T", value)
        }
    case "email":
        if str, ok := value.(string); ok {
            if !isValidEmail(str) {
                return fmt.Errorf("invalid email format: %s", str)
            }
        } else {
            return fmt.Errorf("expected email string, got %T", value)
        }
    case "url":
        if str, ok := value.(string); ok {
            if !isValidURL(str) {
                return fmt.Errorf("invalid URL format: %s", str)
            }
        } else {
            return fmt.Errorf("expected URL string, got %T", value)
        }
    case "number":
        if !isNumeric(value) {
            return fmt.Errorf("expected number, got %T", value)
        }
    case "array":
        if _, ok := value.([]interface{}); !ok {
            return fmt.Errorf("expected array, got %T", value)
        }
    case "enum":
        if _, ok := value.(string); !ok {
            return fmt.Errorf("expected enum string, got %T", value)
        }
    default:
        return fmt.Errorf("unknown field type: %s", fieldType)
    }
    return nil
}
```

#### Default Value Application

```go
func applyFieldDefaults(workItem *WorkItem, cfg *config.Config) {
    for fieldName, fieldConfig := range cfg.Fields {
        // Check if field already exists
        if _, exists := workItem.Fields[fieldName]; exists {
            continue
        }

        // Apply default if configured
        if fieldConfig.Default != nil {
            workItem.Fields[fieldName] = fieldConfig.Default
        }
    }
}
```

### Migration Strategy

1. **Phase 1: Add Configuration Support**
   - Add `Fields` map to `Config` struct
   - Load field configuration from `kira.yml`
   - Validate field configuration syntax
   - Reject attempts to configure `id`, `title`, `status`, `kind`, or `created` fields

2. **Phase 2: Enhance Validation**
   - Update `validateRequiredFields` to use field configuration
   - Add field type validation
   - Add format validation
   - Maintain backward compatibility with hardcoded validation

3. **Phase 3: Integrate with Commands**
   - Update `kira new` to use field configuration
   - Update `kira assign` to validate field names and values
   - Update `kira lint` to use field configuration
   - Update `kira doctor` to fix field issues

4. **Phase 4: Default Values**
   - Implement default value application
   - Add default value support to `kira new`
   - Document default value behavior

### Testing Strategy

#### Unit Tests
- Test field configuration parsing
- Test field type validation
- Test format validation (regex, date, email, URL)
- Test enum validation
- Test range validation (min/max)
- Test default value application
- Test backward compatibility

#### Integration Tests
- Test `kira new` with field configuration
- Test `kira assign` with field configuration
- Test `kira lint` with field configuration
- Test `kira doctor` with field configuration
- Test validation errors are reported correctly

#### E2E Tests
- Test complete workflow with custom fields
- Test migration from hardcoded to configured validation
- Test backward compatibility with existing work items

### Security Considerations

- **Input Validation**: All field values must be validated before use
- **Path Safety**: Field names must be validated to prevent path traversal
- **Regex Safety**: Regex patterns must be validated to prevent ReDoS attacks
- **Type Safety**: Type validation prevents injection attacks
- **Default Values**: Default values must be validated for type safety
- **Hardcoded Fields**: `id`, `title`, `status`, `kind`, and `created` fields must always use hardcoded validation to prevent configuration-based attacks

## Release Notes

### New Features

- **Field Configuration System**: Define custom fields for work items in `kira.yml`
- **Field Types**: Support for string, date, email, URL, number, array, and enum field types
- **Validation Rules**: Configure format validation, range constraints, and allowed values
- **Default Values**: Set default values for fields that are applied during work item creation
- **Field Metadata**: Add descriptions and display names for fields
- **Enhanced Validation**: Type-safe validation for all configured fields

### Improvements

- **Flexible Validation**: Replace hardcoded validation with configurable field definitions (except `id`, `title`, `status`, `kind`, and `created` which remain hardcoded)
- **Type Safety**: Ensure field values match their declared types
- **Better Error Messages**: Clear validation errors with field names and expected formats
- **Command Integration**: All commands (`kira new`, `kira assign`, `kira lint`) respect field configuration
- **Backward Compatibility**: Existing work items continue to work without field configuration
- **Core Field Protection**: `id`, `title`, `status`, `kind`, and `created` fields remain hardcoded to ensure system stability and consistency with existing configuration

### Configuration Example

```yaml
fields:
  assigned:
    type: email
    required: false
    description: "Assigned user email address"

  priority:
    type: enum
    required: false
    allowed_values: [low, medium, high, critical]
    default: medium
    description: "Priority level"

  due:
    type: date
    required: false
    format: "2006-01-02"
    min_date: "today"
    description: "Due date"

  tags:
    type: array
    required: false
    item_type: string
    unique: true
    description: "Tags for categorization"
```

### Migration

- Existing work items continue to work without changes
- Teams can adopt field configuration incrementally
- Default validation behavior is preserved when no field configuration exists
- Field configuration is optional and can be added gradually

