// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"kira/internal/config"
	"kira/internal/validation"
)

var lintCmd = &cobra.Command{
	Use:           "lint",
	Short:         "Check for issues in work items",
	Long:          `Scans folders and files to check for issues and reports any found.`,
	Args:          cobra.ExactArgs(0),
	SilenceUsage:  true, // Don't show usage on error
	SilenceErrors: true, // Don't show error message (main.go handles it)
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if err := checkWorkDir(cfg); err != nil {
			return err
		}

		// Override strict mode if flag is set
		strictFlag, _ := cmd.Flags().GetBool("strict")
		if strictFlag {
			cfg.Validation.Strict = true
		}

		return lintWorkItems(cfg)
	},
}

func init() {
	lintCmd.Flags().Bool("strict", false, "Enable strict mode: flag fields not defined in configuration")
}

func lintWorkItems(cfg *config.Config) error {
	result, err := validation.ValidateWorkItems(cfg)
	if err != nil {
		return fmt.Errorf("failed to validate work items: %w", err)
	}

	if result.HasErrors() {
		printCategorizedErrors(result.Errors)
		return fmt.Errorf("validation failed")
	}

	fmt.Println("No issues found. All work items are valid.")
	return nil
}

// printCategorizedErrors groups errors by category and displays them with counts.
func printCategorizedErrors(errors []validation.ValidationError) {
	// Categorize errors
	fieldErrors := []validation.ValidationError{}
	unknownFieldErrors := []validation.ValidationError{}
	workflowErrors := []validation.ValidationError{}
	duplicateErrors := []validation.ValidationError{}
	parseErrors := []validation.ValidationError{}
	otherErrors := []validation.ValidationError{}

	for _, err := range errors {
		category := categorizeError(err)
		switch category {
		case errorCategoryWorkflow:
			workflowErrors = append(workflowErrors, err)
		case errorCategoryDuplicate:
			duplicateErrors = append(duplicateErrors, err)
		case errorCategoryParse:
			parseErrors = append(parseErrors, err)
		case errorCategoryUnknownField:
			unknownFieldErrors = append(unknownFieldErrors, err)
		case errorCategoryField:
			fieldErrors = append(fieldErrors, err)
		default:
			otherErrors = append(otherErrors, err)
		}
	}

	// Print categorized errors with counts
	totalErrors := len(errors)
	fmt.Printf("Validation errors found (%d total):\n\n", totalErrors)

	printErrorCategory("Field Validation Errors", fieldErrors, true)
	printErrorCategory("Unknown Fields", unknownFieldErrors, true)
	printErrorCategory("Workflow Errors", workflowErrors, false)
	printErrorCategory("Duplicate ID Errors", duplicateErrors, true)
	printErrorCategory("Parse Errors", parseErrors, true)
	printErrorCategory("Other Errors", otherErrors, true)
}

// printErrorCategory prints a category of errors if any exist.
func printErrorCategory(categoryName string, errors []validation.ValidationError, includeFile bool) {
	if len(errors) == 0 {
		return
	}
	fmt.Printf("%s (%d):\n", categoryName, len(errors))
	for _, err := range errors {
		if includeFile {
			fmt.Printf("  %s: %s\n", err.File, err.Message)
		} else {
			fmt.Printf("  %s\n", err.Message)
		}
	}
	fmt.Println()
}

const workflowErrorFile = "workflow"

// categorizeError determines the category of a validation error.
func categorizeError(err validation.ValidationError) string {
	if err.File == workflowErrorFile {
		return errorCategoryWorkflow
	}
	if strings.Contains(err.Message, "duplicate ID") {
		return errorCategoryDuplicate
	}
	if strings.HasPrefix(err.Message, "failed to parse") {
		return errorCategoryParse
	}
	if strings.Contains(err.Message, "unknown fields found") {
		return errorCategoryUnknownField
	}
	if isFieldError(err.Message) {
		return errorCategoryField
	}
	return "other"
}

// isFieldError checks if an error message indicates a field validation error.
func isFieldError(message string) bool {
	if strings.HasPrefix(message, "field '") {
		return true
	}
	if !strings.Contains(message, "invalid ") {
		return false
	}
	fieldErrorKeywords := []string{
		"date format", "email", "format", "enum", "number",
		"not in allowed values", "does not match format",
	}
	for _, keyword := range fieldErrorKeywords {
		if strings.Contains(message, keyword) {
			return true
		}
	}
	return false
}
