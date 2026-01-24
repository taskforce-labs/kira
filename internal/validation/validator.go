// Package validation provides validation functionality for work items.
package validation

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"

	"kira/internal/config"
)

// Field type constants
const (
	fieldTypeString = "string"
	fieldTypeDate   = "date"
	fieldTypeEmail  = "email"
	fieldTypeURL    = "url"
	fieldTypeNumber = "number"
	fieldTypeArray  = "array"
	fieldTypeEnum   = "enum"
)

// Date format constants
const (
	dateFormatDefault = "2006-01-02"
	dateValueToday    = "today"
)

// YAML separator
const yamlSeparator = "---"

// ValidationError represents a validation error for a specific file.
//
//nolint:revive // Stuttering is acceptable for exported types in this package
type ValidationError struct {
	File    string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.File, e.Message)
}

// ValidationResult contains the results of a validation operation.
//
//nolint:revive // Stuttering is acceptable for exported types in this package
type ValidationResult struct {
	Errors []ValidationError
}

// AddError adds a validation error to the result.
func (r *ValidationResult) AddError(file, message string) {
	r.Errors = append(r.Errors, ValidationError{File: file, Message: message})
}

// HasErrors returns true if the validation result contains any errors.
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

func (r *ValidationResult) Error() string {
	if !r.HasErrors() {
		return ""
	}

	var messages []string
	for _, err := range r.Errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "\n")
}

// WorkItem represents a parsed work item with its metadata.
type WorkItem struct {
	ID      string                 `yaml:"id"`
	Title   string                 `yaml:"title"`
	Status  string                 `yaml:"status"`
	Kind    string                 `yaml:"kind"`
	Created string                 `yaml:"created"`
	Fields  map[string]interface{} `yaml:",inline"`
}

// ValidateWorkItems validates all work items in the workspace.
func ValidateWorkItems(cfg *config.Config) (*ValidationResult, error) {
	result := &ValidationResult{}

	// Get all work item files
	files, err := getWorkItemFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get work item files: %w", err)
	}

	// Track IDs for duplicate checking
	idMap := make(map[string][]string)

	for _, file := range files {
		workItem, err := parseWorkItemFile(file)
		if err != nil {
			result.AddError(file, fmt.Sprintf("failed to parse file: %v", err))
			continue
		}

		// Validate required fields
		if err := validateRequiredFields(workItem, cfg); err != nil {
			result.AddError(file, err.Error())
		}

		// Validate ID format
		if err := validateIDFormat(workItem.ID, cfg); err != nil {
			result.AddError(file, err.Error())
		}

		// Validate status values
		if err := validateStatus(workItem.Status, cfg); err != nil {
			result.AddError(file, err.Error())
		}

		// Validate date formats (uses field config if available, falls back to hardcoded logic)
		if err := validateDateFormats(workItem, cfg); err != nil {
			result.AddError(file, err.Error())
		}

		// Validate configured fields
		if err := validateConfiguredFields(workItem, cfg); err != nil {
			result.AddError(file, err.Error())
		}

		// Validate unknown fields in strict mode
		if cfg.Validation.Strict {
			if err := validateUnknownFields(workItem, cfg, file); err != nil {
				result.AddError(file, err.Error())
			}
		}

		// Track ID for duplicate checking
		idMap[workItem.ID] = append(idMap[workItem.ID], file)
	}

	// Check for duplicate IDs
	for id, files := range idMap {
		if len(files) > 1 {
			result.AddError(files[0], fmt.Sprintf("duplicate ID found: %s in files %s", id, strings.Join(files, ", ")))
		}
	}

	// Validate workflow rules
	if err := validateWorkflowRules(cfg); err != nil {
		result.AddError("workflow", err.Error())
	}

	return result, nil
}

func getWorkItemFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(".work", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-markdown files
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Skip template files and IDEAS.md
		if strings.Contains(path, "template") || strings.HasSuffix(path, "IDEAS.md") {
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

// validateWorkItemPath ensures a work item path is safe and within .work/
func validateWorkItemPath(path string) error {
	cleanPath := filepath.Clean(path)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	workDir, err := filepath.Abs(".work")
	if err != nil {
		return fmt.Errorf("failed to resolve .work directory: %w", err)
	}

	workDirWithSep := workDir + string(filepath.Separator)
	if !strings.HasPrefix(absPath+string(filepath.Separator), workDirWithSep) && absPath != workDir {
		return fmt.Errorf("path outside .work directory: %s", path)
	}

	return nil
}

// safeReadWorkItemFile reads a work item file after validating the path
func safeReadWorkItemFile(filePath string) ([]byte, error) {
	if err := validateWorkItemPath(filePath); err != nil {
		return nil, err
	}
	// #nosec G304 - path has been validated by validateWorkItemPath above
	return os.ReadFile(filePath)
}

func parseWorkItemFile(filePath string) (*WorkItem, error) {
	content, err := safeReadWorkItemFile(filePath)
	if err != nil {
		return nil, err
	}

	// Extract YAML front matter between the first pair of --- lines
	lines := strings.Split(string(content), "\n")
	var yamlLines []string
	inYAML := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if i == 0 && trimmed == yamlSeparator {
			inYAML = true
			continue
		}
		if inYAML {
			if trimmed == yamlSeparator {
				break
			}
			yamlLines = append(yamlLines, line)
		}
	}

	wi := &WorkItem{Fields: make(map[string]interface{})}
	if len(yamlLines) > 0 {
		if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), wi); err != nil {
			return nil, fmt.Errorf("failed to parse front matter: %w", err)
		}
	}

	return wi, nil
}

func validateRequiredFields(workItem *WorkItem, cfg *config.Config) error {
	if err := validateHardcodedRequiredFields(workItem, cfg); err != nil {
		return err
	}
	return validateConfiguredRequiredFields(workItem, cfg)
}

func validateHardcodedRequiredFields(workItem *WorkItem, cfg *config.Config) error {
	for _, field := range cfg.Validation.RequiredFields {
		if err := validateHardcodedField(workItem, field); err != nil {
			return err
		}
	}
	return nil
}

func validateHardcodedField(workItem *WorkItem, field string) error {
	switch field {
	case "id":
		if workItem.ID == "" {
			return fmt.Errorf("missing required field: id")
		}
	case "title":
		if workItem.Title == "" {
			return fmt.Errorf("missing required field: title")
		}
	case "status":
		if workItem.Status == "" {
			return fmt.Errorf("missing required field: status")
		}
	case "kind":
		if workItem.Kind == "" {
			return fmt.Errorf("missing required field: kind")
		}
	case "created":
		if workItem.Created == "" {
			return fmt.Errorf("missing required field: created")
		}
	}
	return nil
}

func validateConfiguredRequiredFields(workItem *WorkItem, cfg *config.Config) error {
	if cfg.Fields == nil {
		return nil
	}
	for fieldName, fieldConfig := range cfg.Fields {
		if fieldConfig.Required {
			value, exists := workItem.Fields[fieldName]
			if !exists || isEmptyValue(value) {
				return fmt.Errorf("missing required field: %s", fieldName)
			}
		}
	}
	return nil
}

func validateIDFormat(id string, cfg *config.Config) error {
	matched, err := regexp.MatchString(cfg.Validation.IDFormat, id)
	if err != nil {
		return fmt.Errorf("invalid ID format regex: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid ID format: %s (expected format: %s)", id, cfg.Validation.IDFormat)
	}
	return nil
}

func validateStatus(status string, cfg *config.Config) error {
	for _, validStatus := range cfg.Validation.StatusValues {
		if status == validStatus {
			return nil
		}
	}
	return fmt.Errorf("invalid status '%s'. Valid values: %s", status, strings.Join(cfg.Validation.StatusValues, ", "))
}

func validateDateFormats(workItem *WorkItem, cfg *config.Config) error {
	// Validate created date (always use hardcoded validation)
	if workItem.Created != "" {
		if _, err := time.Parse("2006-01-02", workItem.Created); err != nil {
			return fmt.Errorf("invalid created date format: %s", workItem.Created)
		}
	}

	// Validate other date fields
	// If field config exists for a date field, skip it here (it will be validated in validateConfiguredFields)
	// Otherwise fall back to hardcoded logic
	for key, value := range workItem.Fields {
		// Skip if this field is configured (it will be validated in validateConfiguredFields)
		if cfg.Fields != nil {
			if _, exists := cfg.Fields[key]; exists {
				continue
			}
		}

		// Fall back to hardcoded logic (check if field name contains "date" or "due")
		if strings.Contains(key, "date") || strings.Contains(key, "due") {
			if str, ok := value.(string); ok && str != "" {
				if _, err := time.Parse("2006-01-02", str); err != nil {
					return fmt.Errorf("invalid %s date format: %s", key, str)
				}
			}
		}
	}

	return nil
}

// validateConfiguredFields validates all configured fields in a work item.
func validateConfiguredFields(workItem *WorkItem, cfg *config.Config) error {
	if cfg.Fields == nil {
		return nil // No field configuration, skip validation
	}

	var errors []string

	// Validate each configured field that exists in the work item
	for fieldName, value := range workItem.Fields {
		// Skip hardcoded fields - they use hardcoded validation
		isHardcoded := false
		for _, hardcoded := range config.HardcodedFields {
			if fieldName == hardcoded {
				isHardcoded = true
				break
			}
		}
		if isHardcoded {
			continue
		}

		// Check if field is configured
		fieldConfig, exists := cfg.Fields[fieldName]
		if !exists {
			continue // Field not configured, skip validation
		}

		// Validate the field value
		if err := validateFieldValue(fieldName, value, &fieldConfig); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

// validateFieldValue validates a single field value against its configuration.
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
	if fieldConfig.Type == fieldTypeEnum {
		caseSensitive := isCaseSensitive(fieldConfig.CaseSensitive)
		if err := validateEnumValue(value, fieldConfig.AllowedValues, caseSensitive); err != nil {
			return fmt.Errorf("field '%s': %w", fieldName, err)
		}
	}

	// Range validation
	if err := validateFieldRange(value, fieldConfig); err != nil {
		return fmt.Errorf("field '%s': %w", fieldName, err)
	}

	return nil
}

// validateFieldType checks that a value matches the declared field type.
func validateFieldType(value interface{}, fieldType string) error {
	switch fieldType {
	case fieldTypeString:
		return validateStringType(value)
	case fieldTypeDate:
		return validateDateType(value)
	case fieldTypeEmail:
		return validateEmailType(value)
	case fieldTypeURL:
		return validateURLType(value)
	case fieldTypeNumber:
		return validateNumberType(value)
	case fieldTypeArray:
		return validateArrayType(value)
	case fieldTypeEnum:
		return validateEnumType(value)
	default:
		return fmt.Errorf("unknown field type: %s", fieldType)
	}
}

func validateStringType(value interface{}) error {
	if _, ok := value.(string); !ok {
		return fmt.Errorf("expected string, got %T", value)
	}
	return nil
}

func validateDateType(value interface{}) error {
	// YAML may parse dates as time.Time, so accept both
	if _, ok := value.(string); !ok {
		if _, ok := value.(time.Time); !ok {
			return fmt.Errorf("expected date string or time.Time, got %T", value)
		}
	}
	// Format validation happens in validateFieldFormat
	return nil
}

func validateEmailType(value interface{}) error {
	if str, ok := value.(string); ok {
		if !isValidEmail(str) {
			return fmt.Errorf("invalid email format: %s", str)
		}
		return nil
	}
	return fmt.Errorf("expected email string, got %T", value)
}

func validateURLType(value interface{}) error {
	if str, ok := value.(string); ok {
		if !isValidURL(str) {
			return fmt.Errorf("invalid URL format: %s", str)
		}
		return nil
	}
	return fmt.Errorf("expected URL string, got %T", value)
}

func validateNumberType(value interface{}) error {
	if !IsNumeric(value) {
		return fmt.Errorf("expected number, got %T", value)
	}
	return nil
}

func validateArrayType(value interface{}) error {
	if _, ok := value.([]interface{}); !ok {
		// Also check for []string, []int, etc.
		val := reflect.ValueOf(value)
		if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
			return fmt.Errorf("expected array, got %T", value)
		}
	}
	return nil
}

func validateEnumType(value interface{}) error {
	if _, ok := value.(string); !ok {
		return fmt.Errorf("expected enum string, got %T", value)
	}
	return nil
}

// validateFieldFormat applies format validation (regex for strings, date format for dates).
func validateFieldFormat(value interface{}, format, fieldType string) error {
	switch fieldType {
	case fieldTypeString:
		return validateStringFormat(value, format)
	case fieldTypeDate:
		return validateDateFormat(value, format)
	default:
		return nil
	}
}

func validateStringFormat(value interface{}, format string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("format validation requires string value")
	}
	matched, err := regexp.MatchString(format, str)
	if err != nil {
		return fmt.Errorf("invalid regex format: %w", err)
	}
	if !matched {
		return fmt.Errorf("value '%s' does not match format pattern: %s", str, format)
	}
	return nil
}

func validateDateFormat(value interface{}, format string) error {
	// YAML may parse dates as time.Time, so handle both
	if str, ok := value.(string); ok {
		if format == "" {
			format = dateFormatDefault
		}
		if _, err := time.Parse(format, str); err != nil {
			return fmt.Errorf("date '%s' does not match format: %s", str, format)
		}
		return nil
	}
	if t, ok := value.(time.Time); ok {
		// For time.Time, validate the format by formatting and parsing
		if format != "" {
			formatted := t.Format(format)
			if _, err := time.Parse(format, formatted); err != nil {
				return fmt.Errorf("date format validation failed: %s", format)
			}
		}
		return nil
	}
	return fmt.Errorf("format validation requires string or time.Time value, got %T", value)
}

// isCaseSensitive returns the case-sensitive setting, defaulting to true if not set.
func isCaseSensitive(caseSensitive *bool) bool {
	if caseSensitive == nil {
		return true // Default to case-sensitive
	}
	return *caseSensitive
}

// validateEnumValue checks that a value is in the allowed values list.
func validateEnumValue(value interface{}, allowedValues []string, caseSensitive bool) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("enum validation requires string value")
	}

	for _, allowed := range allowedValues {
		if caseSensitive {
			if str == allowed {
				return nil
			}
		} else {
			if strings.EqualFold(str, allowed) {
				return nil
			}
		}
	}

	return fmt.Errorf("value '%s' is not in allowed values: %s", str, strings.Join(allowedValues, ", "))
}

// validateFieldRange checks min/max constraints for fields.
func validateFieldRange(value interface{}, fieldConfig *config.FieldConfig) error {
	switch fieldConfig.Type {
	case fieldTypeString:
		return validateStringRange(value, fieldConfig)
	case fieldTypeNumber:
		return validateNumberRange(value, fieldConfig)
	case fieldTypeDate:
		return validateDateRangeValue(value, fieldConfig)
	case fieldTypeArray:
		return validateArrayRange(value, fieldConfig)
	case fieldTypeURL:
		return validateURLRange(value, fieldConfig)
	default:
		return nil
	}
}

func validateStringRange(value interface{}, fieldConfig *config.FieldConfig) error {
	str, ok := value.(string)
	if !ok {
		return nil // Type validation will catch this
	}
	length := len(str)
	if fieldConfig.MinLength != nil && length < *fieldConfig.MinLength {
		return fmt.Errorf("string length %d is less than min_length %d", length, *fieldConfig.MinLength)
	}
	if fieldConfig.MaxLength != nil && length > *fieldConfig.MaxLength {
		return fmt.Errorf("string length %d is greater than max_length %d", length, *fieldConfig.MaxLength)
	}
	return nil
}

func validateNumberRange(value interface{}, fieldConfig *config.FieldConfig) error {
	num, err := getNumericValue(value)
	if err != nil {
		return nil // Type validation will catch this
	}
	if fieldConfig.MinValue != nil && num < *fieldConfig.MinValue {
		return fmt.Errorf("value %v is less than min %v", num, *fieldConfig.MinValue)
	}
	if fieldConfig.MaxValue != nil && num > *fieldConfig.MaxValue {
		return fmt.Errorf("value %v is greater than max %v", num, *fieldConfig.MaxValue)
	}
	return nil
}

func validateDateRangeValue(value interface{}, fieldConfig *config.FieldConfig) error {
	var date time.Time
	var err error

	// Handle both string and time.Time (YAML may parse dates as time.Time)
	if str, ok := value.(string); ok {
		dateFormat := fieldConfig.Format
		if dateFormat == "" {
			dateFormat = dateFormatDefault
		}
		// Parse string dates in a canonical timezone (UTC) so that all range
		// checks operate on a consistent representation regardless of the
		// environment's local timezone.
		date, err = time.ParseInLocation(dateFormat, str, time.UTC)
		if err != nil {
			return nil // Format validation will catch this
		}
		date = normalizeToUTCDate(date)
	} else if t, ok := value.(time.Time); ok {
		// Normalize time.Time values to UTC calendar dates as well so that
		// range checks are consistent for both string and time.Time sources.
		date = normalizeToUTCDate(t)
	} else {
		return nil // Type validation will catch this
	}

	return validateDateRange(date, fieldConfig.MinDate, fieldConfig.MaxDate)
}

func validateArrayRange(value interface{}, fieldConfig *config.FieldConfig) error {
	arr, err := convertToArray(value)
	if err != nil {
		return nil // Type validation will catch this
	}
	if err := validateArrayLength(arr, fieldConfig); err != nil {
		return err
	}
	if err := validateArrayItems(arr, fieldConfig); err != nil {
		return err
	}
	return validateArrayUniqueness(arr, fieldConfig)
}

func convertToArray(value interface{}) ([]interface{}, error) {
	switch v := value.(type) {
	case []interface{}:
		return v, nil
	default:
		val := reflect.ValueOf(value)
		if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
			arr := make([]interface{}, val.Len())
			for i := 0; i < val.Len(); i++ {
				arr[i] = val.Index(i).Interface()
			}
			return arr, nil
		}
		return nil, fmt.Errorf("not an array")
	}
}

func validateArrayLength(arr []interface{}, fieldConfig *config.FieldConfig) error {
	length := len(arr)
	if fieldConfig.MinLength != nil && length < *fieldConfig.MinLength {
		return fmt.Errorf("array length %d is less than min_length %d", length, *fieldConfig.MinLength)
	}
	if fieldConfig.MaxLength != nil && length > *fieldConfig.MaxLength {
		return fmt.Errorf("array length %d is greater than max_length %d", length, *fieldConfig.MaxLength)
	}
	return nil
}

func validateArrayItems(arr []interface{}, fieldConfig *config.FieldConfig) error {
	if fieldConfig.ItemType == "" {
		return nil
	}
	for i, item := range arr {
		if err := validateArrayItem(item, fieldConfig); err != nil {
			return fmt.Errorf("array item at index %d: %w", i, err)
		}
	}
	return nil
}

func validateArrayUniqueness(arr []interface{}, fieldConfig *config.FieldConfig) error {
	if !fieldConfig.Unique {
		return nil
	}
	seen := make(map[interface{}]bool)
	for _, item := range arr {
		key := getItemKey(item)
		if seen[key] {
			return fmt.Errorf("array contains duplicate value: %v", item)
		}
		seen[key] = true
	}
	return nil
}

func validateURLRange(value interface{}, fieldConfig *config.FieldConfig) error {
	str, ok := value.(string)
	if !ok {
		return nil // Type validation will catch this
	}
	if len(fieldConfig.Schemes) > 0 {
		parsedURL, err := url.Parse(str)
		if err != nil {
			return nil // URL validation will catch this
		}
		validScheme := false
		for _, scheme := range fieldConfig.Schemes {
			if parsedURL.Scheme == scheme {
				validScheme = true
				break
			}
		}
		if !validScheme {
			return fmt.Errorf("URL scheme '%s' is not allowed. Allowed schemes: %s", parsedURL.Scheme, strings.Join(fieldConfig.Schemes, ", "))
		}
	}
	return nil
}

// validateArrayItem validates a single array item against the item_type.
func validateArrayItem(item interface{}, fieldConfig *config.FieldConfig) error {
	switch fieldConfig.ItemType {
	case fieldTypeString:
		if _, ok := item.(string); !ok {
			return fmt.Errorf("expected string item, got %T", item)
		}
	case fieldTypeNumber:
		if !IsNumeric(item) {
			return fmt.Errorf("expected number item, got %T", item)
		}
	case fieldTypeEnum:
		str, ok := item.(string)
		if !ok {
			return fmt.Errorf("expected enum string item, got %T", item)
		}
		caseSensitive := isCaseSensitive(fieldConfig.CaseSensitive)
		return validateEnumValue(str, fieldConfig.AllowedValues, caseSensitive)
	}
	return nil
}

// normalizeToUTCDate returns a time representing the same calendar date in UTC at midnight.
// This ensures that date comparisons are done purely on calendar dates, independent of timezone.
func normalizeToUTCDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// validateDateRange validates a date against min/max date constraints using calendar-date
// comparisons in a consistent timezone (UTC). This avoids issues where dates parsed in UTC
// are compared against "today" based on the local timezone.
func validateDateRange(date time.Time, minDate, maxDate string) error {
	// Normalize input date to a UTC calendar date.
	date = normalizeToUTCDate(date)

	now := time.Now()
	// Determine today's date in the user's local calendar, then normalize to UTC
	// so all comparisons are done in a single timezone.
	todayLocal := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	today := normalizeToUTCDate(todayLocal)

	if minDate != "" {
		var minTime time.Time
		var err error
		switch minDate {
		case dateValueToday:
			minTime = today
		case "future":
			// "future" means strictly after today, so tomorrow is the minimum.
			minTime = today.AddDate(0, 0, 1)
		default:
			// Try to parse as absolute date in the canonical comparison timezone (UTC).
			minTime, err = time.ParseInLocation(dateFormatDefault, minDate, time.UTC)
			if err != nil {
				return fmt.Errorf("invalid min_date format: %s", minDate)
			}
			minTime = normalizeToUTCDate(minTime)
		}
		if date.Before(minTime) {
			return fmt.Errorf("date %s is before min_date %s", date.Format(dateFormatDefault), minDate)
		}
	}

	if maxDate != "" {
		var maxTime time.Time
		var err error
		switch maxDate {
		case dateValueToday:
			maxTime = today
		case "future":
			// No upper bound for "future".
			return nil
		default:
			// Try to parse as absolute date in the canonical comparison timezone (UTC).
			maxTime, err = time.ParseInLocation(dateFormatDefault, maxDate, time.UTC)
			if err != nil {
				return fmt.Errorf("invalid max_date format: %s", maxDate)
			}
			maxTime = normalizeToUTCDate(maxTime)
		}
		if date.After(maxTime) {
			return fmt.Errorf("date %s is after max_date %s", date.Format(dateFormatDefault), maxDate)
		}
	}

	return nil
}

// Helper functions

// isValidEmail checks if a string is a valid email address.
func isValidEmail(email string) bool {
	// Simple email validation regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// isValidURL checks if a string is a valid URL.
// Note: This function uses url.ParseRequestURI which has a known vulnerability
// (GO-2025-4010) in Go < 1.25.2. For production use, upgrade to Go 1.25.2+.
func isValidURL(urlStr string) bool {
	// Basic validation before parsing to reduce attack surface
	if len(urlStr) == 0 || len(urlStr) > 2048 {
		return false
	}
	// Check for potentially problematic patterns
	if strings.Contains(urlStr, "[") && strings.Contains(urlStr, "]") {
		// IPv6 addresses in brackets - validate format more strictly
		// This is a partial workaround for GO-2025-4010
		if !isValidIPv6Bracketed(urlStr) {
			return false
		}
	}
	_, err := url.ParseRequestURI(urlStr)
	return err == nil
}

// isValidIPv6Bracketed performs basic validation of bracketed IPv6 addresses
// as a workaround for GO-2025-4010 until Go 1.25.2+ is available.
func isValidIPv6Bracketed(urlStr string) bool {
	// Find the bracketed portion
	start := strings.Index(urlStr, "[")
	end := strings.Index(urlStr, "]")
	if start == -1 || end == -1 || end <= start {
		return false
	}
	// Extract the bracketed content
	bracketed := urlStr[start+1 : end]
	// Basic validation: should contain colons and hex characters only
	// This is a simplified check - proper IPv6 validation is complex
	if len(bracketed) == 0 || len(bracketed) > 45 {
		return false
	}
	// Check for valid IPv6 characters (simplified)
	for _, r := range bracketed {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') && r != ':' && r != '.' {
			return false
		}
	}
	return true
}

// IsNumeric checks if a value is numeric (int, int64, float64, etc.).
func IsNumeric(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	}
	return false
}

// getNumericValue converts a value to float64 for comparison.
func getNumericValue(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	default:
		return 0, fmt.Errorf("not a numeric value: %T", value)
	}
}

// getItemKey returns a key for an array item for uniqueness checking.
func getItemKey(item interface{}) interface{} {
	// For strings and numbers, use the value directly
	// For other types, convert to string
	switch v := item.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// isEmptyValue checks if a value is empty (nil, empty string, empty array, etc.).
func isEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return v == ""
	case []interface{}:
		return len(v) == 0
	default:
		val := reflect.ValueOf(value)
		if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
			return val.Len() == 0
		}
		return false
	}
}

// ApplyFieldDefaults applies default values to a work item for configured fields that are missing.
// It returns the names of fields that were added (i.e. had defaults applied) so callers can
// persist changes and report fixes.
func ApplyFieldDefaults(workItem *WorkItem, cfg *config.Config) (added []string, err error) {
	if cfg.Fields == nil {
		return nil, nil // No field configuration
	}

	for fieldName, fieldConfig := range cfg.Fields {
		// Skip if field already exists and is not empty
		if value, exists := workItem.Fields[fieldName]; exists && !isEmptyValue(value) {
			continue
		}

		// Skip hardcoded fields
		isHardcoded := false
		for _, hardcoded := range config.HardcodedFields {
			if fieldName == hardcoded {
				isHardcoded = true
				break
			}
		}
		if isHardcoded {
			continue
		}

		// Apply default if configured
		if fieldConfig.Default != nil {
			defaultValue, resolveErr := resolveDefaultValue(fieldConfig.Default, &fieldConfig)
			if resolveErr != nil {
				return nil, fmt.Errorf("failed to resolve default value for field '%s': %w", fieldName, resolveErr)
			}
			workItem.Fields[fieldName] = defaultValue
			added = append(added, fieldName)
		}
	}

	return added, nil
}

// resolveDefaultValue converts a default value to the appropriate type for the field.
func resolveDefaultValue(defaultValue interface{}, fieldConfig *config.FieldConfig) (interface{}, error) {
	switch fieldConfig.Type {
	case fieldTypeString:
		return resolveStringDefault(defaultValue)
	case fieldTypeDate:
		return resolveDateDefault(defaultValue, fieldConfig)
	case fieldTypeEmail:
		return resolveEmailDefault(defaultValue)
	case fieldTypeURL:
		return resolveURLDefault(defaultValue)
	case fieldTypeNumber:
		return resolveNumberDefault(defaultValue)
	case fieldTypeArray:
		return resolveArrayDefault(defaultValue)
	case fieldTypeEnum:
		return resolveEnumDefault(defaultValue, fieldConfig)
	default:
		return defaultValue, nil
	}
}

func resolveStringDefault(defaultValue interface{}) (interface{}, error) {
	if str, ok := defaultValue.(string); ok {
		return str, nil
	}
	return fmt.Sprintf("%v", defaultValue), nil
}

func resolveDateDefault(defaultValue interface{}, fieldConfig *config.FieldConfig) (interface{}, error) {
	if str, ok := defaultValue.(string); ok {
		// Handle special values like "today"
		if str == dateValueToday {
			return time.Now().Format(dateFormatDefault), nil
		}
		// Validate the date format
		dateFormat := fieldConfig.Format
		if dateFormat == "" {
			dateFormat = dateFormatDefault
		}
		if _, err := time.Parse(dateFormat, str); err != nil {
			return nil, fmt.Errorf("invalid date default value '%s': %w", str, err)
		}
		return str, nil
	}
	return nil, fmt.Errorf("date default must be a string, got %T", defaultValue)
}

func resolveEmailDefault(defaultValue interface{}) (interface{}, error) {
	if str, ok := defaultValue.(string); ok {
		// Allow empty strings as defaults (placeholders), validation happens later
		if str == "" {
			return str, nil
		}
		if !isValidEmail(str) {
			return nil, fmt.Errorf("invalid email default value: %s", str)
		}
		return str, nil
	}
	return nil, fmt.Errorf("email default must be a string, got %T", defaultValue)
}

func resolveURLDefault(defaultValue interface{}) (interface{}, error) {
	if str, ok := defaultValue.(string); ok {
		// Allow empty strings as defaults (placeholders), validation happens later
		if str == "" {
			return str, nil
		}
		if !isValidURL(str) {
			return nil, fmt.Errorf("invalid URL default value: %s", str)
		}
		return str, nil
	}
	return nil, fmt.Errorf("URL default must be a string, got %T", defaultValue)
}

func resolveNumberDefault(defaultValue interface{}) (interface{}, error) {
	if IsNumeric(defaultValue) {
		return defaultValue, nil
	}
	// Try to convert string to number
	if str, ok := defaultValue.(string); ok {
		if num, err := strconv.ParseFloat(str, 64); err == nil {
			return num, nil
		}
	}
	return nil, fmt.Errorf("number default must be numeric, got %T", defaultValue)
}

func resolveArrayDefault(defaultValue interface{}) (interface{}, error) {
	// Default can be an array or a single value that becomes an array
	if arr, ok := defaultValue.([]interface{}); ok {
		return arr, nil
	}
	// Single value becomes array with one element
	return []interface{}{defaultValue}, nil
}

func resolveEnumDefault(defaultValue interface{}, fieldConfig *config.FieldConfig) (interface{}, error) {
	if str, ok := defaultValue.(string); ok {
		// Validate against allowed values
		if len(fieldConfig.AllowedValues) > 0 {
			valid := false
			caseSensitive := isCaseSensitive(fieldConfig.CaseSensitive)
			for _, allowed := range fieldConfig.AllowedValues {
				if caseSensitive {
					if str == allowed {
						valid = true
						break
					}
				} else {
					if strings.EqualFold(str, allowed) {
						valid = true
						break
					}
				}
			}
			if !valid {
				return nil, fmt.Errorf("enum default '%s' is not in allowed values: %s", str, strings.Join(fieldConfig.AllowedValues, ", "))
			}
		}
		return str, nil
	}
	return nil, fmt.Errorf("enum default must be a string, got %T", defaultValue)
}

func validateWorkflowRules(cfg *config.Config) error {
	// Check that only one item is in doing folder
	doingPath := filepath.Join(".work", cfg.StatusFolders["doing"])
	if _, err := os.Stat(doingPath); err == nil {
		files, err := os.ReadDir(doingPath)
		if err != nil {
			return fmt.Errorf("failed to read doing folder: %w", err)
		}

		var workItems []string
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
				workItems = append(workItems, file.Name())
			}
		}

		if len(workItems) > 1 {
			return fmt.Errorf("multiple items in doing folder. Only one item allowed at a time. Found: %s", strings.Join(workItems, ", "))
		}
	}

	return nil
}

// GetNextID generates the next available work item ID.
func GetNextID() (string, error) {
	files, err := getWorkItemFiles()
	if err != nil {
		return "", fmt.Errorf("failed to get work item files: %w", err)
	}

	var maxID int
	for _, file := range files {
		workItem, err := parseWorkItemFile(file)
		if err != nil {
			continue
		}

		if id, err := strconv.Atoi(workItem.ID); err == nil {
			if id > maxID {
				maxID = id
			}
		}
	}

	nextID := maxID + 1
	return fmt.Sprintf("%03d", nextID), nil
}

// FixDuplicateIDs fixes duplicate work item IDs by assigning new IDs.
func FixDuplicateIDs() (*ValidationResult, error) {
	result := &ValidationResult{}

	files, err := getWorkItemFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get work item files: %w", err)
	}

	// Group files by ID
	idGroups := make(map[string][]string)
	for _, file := range files {
		workItem, err := parseWorkItemFile(file)
		if err != nil {
			continue
		}
		idGroups[workItem.ID] = append(idGroups[workItem.ID], file)
	}

	// Fix duplicates by assigning new IDs to newer files
	for _, files := range idGroups {
		if len(files) > 1 {
			// Sort files by modification time (newest first)
			sort.Slice(files, func(i, j int) bool {
				info1, _ := os.Stat(files[i])
				info2, _ := os.Stat(files[j])
				return info1.ModTime().After(info2.ModTime())
			})

			// Keep the oldest file with the original ID, assign new IDs to others
			for i := 1; i < len(files); i++ {
				newID, err := GetNextID()
				if err != nil {
					result.AddError(files[i], fmt.Sprintf("failed to generate new ID: %v", err))
					continue
				}

				// Update the file with new ID
				if err := updateWorkItemID(files[i], newID); err != nil {
					result.AddError(files[i], fmt.Sprintf("failed to update ID: %v", err))
				}
			}
		}
	}

	return result, nil
}

func updateWorkItemID(filePath, newID string) error {
	content, err := safeReadWorkItemFile(filePath)
	if err != nil {
		return err
	}

	// Replace the ID in the YAML front matter
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "id:") {
			lines[i] = fmt.Sprintf("id: %s", newID)
			break
		}
	}

	return os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0o600)
}

// FixFieldIssues fixes field validation issues in work items.
func FixFieldIssues(cfg *config.Config) (*ValidationResult, error) {
	result := &ValidationResult{}

	files, err := getWorkItemFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get work item files: %w", err)
	}

	if cfg.Fields == nil {
		// No field configuration, nothing to fix
		return result, nil
	}

	for _, file := range files {
		if err := fixWorkItemFields(file, cfg, result); err != nil {
			result.AddError(file, fmt.Sprintf("failed to fix fields: %v", err))
		}
	}

	return result, nil
}

// FixHardcodedDateFormats fixes date format issues in hardcoded fields like `created`.
func FixHardcodedDateFormats() (*ValidationResult, error) {
	result := &ValidationResult{}

	files, err := getWorkItemFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get work item files: %w", err)
	}

	for _, file := range files {
		workItem, err := parseWorkItemFile(file)
		if err != nil {
			continue
		}

		// Fix created date format
		if workItem.Created != "" {
			originalDate := workItem.Created
			if _, err := time.Parse(dateFormatDefault, workItem.Created); err != nil {
				// Try to fix the date format
				if fixedDate, fixed := tryFixHardcodedDate(workItem.Created); fixed {
					workItem.Created = fixedDate
					// Write back the fixed work item
					if err := writeWorkItemFile(file, workItem); err != nil {
						result.AddError(file, fmt.Sprintf("failed to fix created date: %v", err))
					} else {
						result.AddError(file, fmt.Sprintf("fixed created date format: %s -> %s", originalDate, fixedDate))
					}
				}
			}
		}
	}

	return result, nil
}

// tryFixHardcodedDate attempts to fix a hardcoded date field value.
func tryFixHardcodedDate(dateStr string) (string, bool) {
	// Try common date formats
	commonFormats := []string{
		dateFormatDefault,           // 2006-01-02
		"2006-01-02T15:04:05Z",      // ISO 8601 with time
		"2006-01-02T15:04:05-07:00", // ISO 8601 with timezone
		"2006-01-02T15:04:05.000Z",  // ISO 8601 with milliseconds
		"2006/01/02",                // Slash format
		"01/02/2006",                // US format
		time.RFC3339,                // RFC3339
		time.RFC3339Nano,            // RFC3339 with nanoseconds
	}

	for _, format := range commonFormats {
		if t, err := time.Parse(format, dateStr); err == nil {
			// Found a valid format, convert to expected format (YYYY-MM-DD)
			return t.Format(dateFormatDefault), true
		}
	}

	return dateStr, false
}

func fixWorkItemFields(file string, cfg *config.Config, result *ValidationResult) error {
	workItem, err := parseWorkItemFile(file)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Apply defaults for missing fields (required or not)
	added, err := ApplyFieldDefaults(workItem, cfg)
	if err != nil {
		return fmt.Errorf("failed to apply defaults: %w", err)
	}
	modified := len(added) > 0
	for _, fieldName := range added {
		result.AddError(file, fmt.Sprintf("fixed field '%s': applied default value", fieldName))
	}

	// Apply value fixes (date format, enum case, etc.) and detect if any were made
	if processFieldFixes(workItem, cfg, result, file) {
		modified = true
	}

	// Write back if modified
	if modified {
		if err := writeWorkItemFile(file, workItem); err != nil {
			return fmt.Errorf("failed to write fixes: %w", err)
		}
	}

	return nil
}

// processFieldFixes applies value-level fixes (date format, enum case, email trim, etc.)
// and returns true if any fix was applied. Default application is handled by ApplyFieldDefaults
// and fixWorkItemFields; this only handles correcting invalid existing values.
func processFieldFixes(workItem *WorkItem, cfg *config.Config, result *ValidationResult, file string) bool {
	modified := false
	for fieldName, fieldConfig := range cfg.Fields {
		if isHardcodedField(fieldName) {
			continue
		}

		// Try to fix invalid field values (date format, enum case, etc.)
		if value, exists := workItem.Fields[fieldName]; exists && !isEmptyValue(value) {
			if fixedValue, shouldUpdate := tryFixFieldValue(fieldName, value, &fieldConfig); shouldUpdate {
				workItem.Fields[fieldName] = fixedValue
				modified = true
				result.AddError(file, fmt.Sprintf("fixed field '%s': corrected value", fieldName))
			}
		}
	}
	return modified
}

func isHardcodedField(fieldName string) bool {
	for _, hardcoded := range config.HardcodedFields {
		if fieldName == hardcoded {
			return true
		}
	}
	return false
}

// validateUnknownFields checks for fields that are not defined in the configuration.
// This is only called when strict mode is enabled.
func validateUnknownFields(workItem *WorkItem, cfg *config.Config, _ string) error {
	var unknownFields []string

	if len(cfg.Fields) == 0 {
		// If no fields are configured, all custom fields are unknown in strict mode
		for fieldName := range workItem.Fields {
			if !isHardcodedField(fieldName) {
				unknownFields = append(unknownFields, fieldName)
			}
		}
	} else {
		// Check each field against configuration
		for fieldName := range workItem.Fields {
			// Skip hardcoded fields
			if isHardcodedField(fieldName) {
				continue
			}
			// Check if field is configured
			if _, exists := cfg.Fields[fieldName]; !exists {
				unknownFields = append(unknownFields, fieldName)
			}
		}
	}

	if len(unknownFields) > 0 {
		return fmt.Errorf("unknown fields found (not in configuration): %s", strings.Join(unknownFields, ", "))
	}

	return nil
}

// tryFixFieldValue attempts to fix an invalid field value.
// Returns the fixed value and true if the value was fixed.
func tryFixFieldValue(_ string, value interface{}, fieldConfig *config.FieldConfig) (interface{}, bool) {
	switch fieldConfig.Type {
	case fieldTypeDate:
		return tryFixDateValue(value, fieldConfig)
	case fieldTypeEnum:
		return tryFixEnumValue(value, fieldConfig)
	case fieldTypeEmail:
		return tryFixEmailValue(value)
	default:
		return value, false
	}
}

func tryFixDateValue(value interface{}, fieldConfig *config.FieldConfig) (interface{}, bool) {
	str, ok := value.(string)
	if !ok {
		return value, false
	}
	dateFormat := fieldConfig.Format
	if dateFormat == "" {
		dateFormat = dateFormatDefault
	}
	// Try parsing with the expected format
	if _, err := time.Parse(dateFormat, str); err != nil {
		// Try common date formats
		commonFormats := []string{dateFormatDefault, "2006/01/02", "01/02/2006", "2006-01-02T15:04:05Z"}
		for _, format := range commonFormats {
			if t, err := time.Parse(format, str); err == nil {
				// Found a valid format, convert to expected format
				return t.Format(dateFormat), true
			}
		}
	}
	return value, false
}

func tryFixEnumValue(value interface{}, fieldConfig *config.FieldConfig) (interface{}, bool) {
	// Try case-insensitive matching
	caseSensitive := isCaseSensitive(fieldConfig.CaseSensitive)
	if str, ok := value.(string); ok && !caseSensitive {
		for _, allowed := range fieldConfig.AllowedValues {
			if strings.EqualFold(str, allowed) {
				// Only treat as a fix when the canonical value differs (e.g. case)
				if str == allowed {
					return value, false
				}
				return allowed, true // Return the canonical value
			}
		}
	}
	return value, false
}

func tryFixEmailValue(value interface{}) (interface{}, bool) {
	// Try to fix common email issues (trim whitespace, lowercase)
	if str, ok := value.(string); ok {
		trimmed := strings.TrimSpace(str)
		lower := strings.ToLower(trimmed)
		if trimmed != str || (lower != trimmed && isValidEmail(lower)) {
			return lower, true
		}
	}
	return value, false
}

// writeWorkItemFile writes a work item back to a file.
func writeWorkItemFile(filePath string, workItem *WorkItem) error {
	content, err := safeReadWorkItemFile(filePath)
	if err != nil {
		return err
	}

	// Extract non-YAML content (everything after the second ---)
	lines := strings.Split(string(content), "\n")
	bodyLines := extractBodyLines(lines)

	// Rebuild YAML front matter
	var newContent strings.Builder
	if err := writeYAMLFrontMatter(&newContent, workItem); err != nil {
		return fmt.Errorf("failed to write YAML front matter: %w", err)
	}
	writeYAMLBody(&newContent, bodyLines)

	return os.WriteFile(filePath, []byte(newContent.String()), 0o600)
}

func extractBodyLines(lines []string) []string {
	var bodyLines []string
	inYAML := false
	yamlEndFound := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if i == 0 && trimmed == yamlSeparator {
			inYAML = true
			continue
		}
		if inYAML {
			if trimmed == yamlSeparator {
				yamlEndFound = true
				inYAML = false
				continue
			}
			// Skip YAML content lines
			continue
		}
		// After YAML ends, collect all remaining lines
		if yamlEndFound {
			bodyLines = append(bodyLines, line)
		}
	}
	return bodyLines
}

func writeYAMLFrontMatter(sb *strings.Builder, workItem *WorkItem) error {
	fmt.Fprintf(sb, "%s\n", yamlSeparator)

	// Write hardcoded fields first (quote when values contain special YAML characters)
	fmt.Fprintf(sb, "id: %s\n", yamlFormatStringValue(workItem.ID))
	fmt.Fprintf(sb, "title: %s\n", yamlFormatStringValue(workItem.Title))
	fmt.Fprintf(sb, "status: %s\n", yamlFormatStringValue(workItem.Status))
	fmt.Fprintf(sb, "kind: %s\n", yamlFormatStringValue(workItem.Kind))
	fmt.Fprintf(sb, "created: %s\n", yamlFormatStringValue(workItem.Created))

	// Write other fields in deterministic (sorted) order so repeated runs
	// of commands like `kira doctor` do not cause spurious diffs.
	var fieldKeys []string
	for key := range workItem.Fields {
		if isHardcodedField(key) {
			continue
		}
		fieldKeys = append(fieldKeys, key)
	}
	sort.Strings(fieldKeys)

	for _, key := range fieldKeys {
		value := workItem.Fields[key]
		if err := writeYAMLField(sb, key, value); err != nil {
			return fmt.Errorf("failed to write field '%s': %w", key, err)
		}
	}

	fmt.Fprintf(sb, "%s\n", yamlSeparator)
	return nil
}

func writeYAMLBody(sb *strings.Builder, bodyLines []string) {
	if len(bodyLines) > 0 {
		sb.WriteString(strings.Join(bodyLines, "\n"))
		if !strings.HasSuffix(sb.String(), "\n") {
			sb.WriteString("\n")
		}
	}
}

// yamlSpecialChars is the set of characters that require a scalar to be double-quoted in YAML.
const yamlSpecialChars = ":#[]{},\"'\\\n\r\t&*!|>%"

// needsYAMLQuoting returns true if the string must be double-quoted for valid YAML output.
func needsYAMLQuoting(s string) bool {
	if s == "" || strings.TrimSpace(s) != s {
		return true
	}
	return strings.ContainsAny(s, yamlSpecialChars)
}

// yamlQuotedString returns a double-quoted YAML scalar with proper escaping.
func yamlQuotedString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// yamlFormatStringValue returns the YAML scalar representation of a string (quoted when necessary).
func yamlFormatStringValue(s string) string {
	if needsYAMLQuoting(s) {
		return yamlQuotedString(s)
	}
	return s
}

// YAMLFormatArrayItem returns the YAML scalar representation of an array element for flow style.
func YAMLFormatArrayItem(item interface{}) string {
	switch v := item.(type) {
	case string:
		return yamlFormatStringValue(v)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return fmt.Sprintf("%v", v)
	default:
		return yamlFormatStringValue(fmt.Sprintf("%v", v))
	}
}

// writeYAMLField writes a YAML field to a string builder.
func writeYAMLField(sb *strings.Builder, key string, value interface{}) error {
	switch v := value.(type) {
	case string:
		fmt.Fprintf(sb, "%s: %s\n", key, yamlFormatStringValue(v))
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		fmt.Fprintf(sb, "%s: %v\n", key, v)
	case bool:
		fmt.Fprintf(sb, "%s: %v\n", key, v)
	case []interface{}:
		fmt.Fprintf(sb, "%s: [", key)
		for i, item := range v {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(YAMLFormatArrayItem(item))
		}
		sb.WriteString("]\n")
	case time.Time:
		fmt.Fprintf(sb, "%s: %s\n", key, yamlFormatStringValue(v.Format(dateFormatDefault)))
	default:
		// For complex types, use YAML marshaling
		// Use a recover to handle panics from yaml.Marshal for unmarshalable types
		var yamlData []byte
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("yaml: cannot marshal type: %v", r)
				}
			}()
			yamlData, err = yaml.Marshal(map[string]interface{}{key: value})
		}()
		if err != nil {
			return err
		}
		// Remove the key from the marshaled output and add it back with proper formatting
		yamlStr := string(yamlData)
		lines := strings.Split(strings.TrimSpace(yamlStr), "\n")
		for _, line := range lines {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	return nil
}
