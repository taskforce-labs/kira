// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"kira/internal/config"
	"kira/internal/validation"
)

const (
	errorCategoryWorkflow     = "workflow"
	errorCategoryDuplicate    = "duplicate"
	errorCategoryParse        = "parse"
	errorCategoryUnknownField = "unknown_field"
	errorCategoryField        = "field"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check for and fix duplicate work item IDs and field issues",
	Long:  `Checks for and fixes duplicate work item IDs and field validation issues.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := checkWorkDir(); err != nil {
			return err
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Override strict mode if flag is set
		strictFlag, _ := cmd.Flags().GetBool("strict")
		if strictFlag {
			cfg.Validation.Strict = true
		}

		return runDoctor(cfg)
	},
}

func init() {
	doctorCmd.Flags().Bool("strict", false, "Enable strict mode: flag fields not defined in configuration")
}

func runDoctor(cfg *config.Config) error {
	validationResult, err := validateWorkItems(cfg)
	if err != nil {
		return err
	}

	if !validationResult.HasErrors() {
		fmt.Println("No issues found. All work items are valid.")
		return nil
	}

	fmt.Println()
	printCategorizedErrors(validationResult.Errors)

	fixedCount := runAutoFixes(cfg)
	unfixableErrors := getUnfixableErrors(validationResult.Errors)
	reportResults(fixedCount, unfixableErrors)

	return nil
}

// validateWorkItems validates all work items and returns the result.
func validateWorkItems(cfg *config.Config) (*validation.ValidationResult, error) {
	fmt.Println("Validating work items...")
	validationResult, err := validation.ValidateWorkItems(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to validate work items: %w", err)
	}
	return validationResult, nil
}

// runAutoFixes runs all automatic fixes and returns the count of fixed issues.
func runAutoFixes(cfg *config.Config) int {
	fmt.Println("\nAttempting to fix issues...")
	fixedCount := 0

	if count := fixDuplicateIDs(); count > 0 {
		fixedCount += count
	}

	if count := fixHardcodedDateFormats(); count > 0 {
		fixedCount += count
	}

	if count := fixFieldIssues(cfg); count > 0 {
		fixedCount += count
	}

	return fixedCount
}

func fixDuplicateIDs() int {
	idResult, err := validation.FixDuplicateIDs()
	if err != nil {
		return 0
	}
	if idResult.HasErrors() {
		count := len(idResult.Errors)
		fmt.Println("\n✅ Fixed duplicate IDs:")
		for _, err := range idResult.Errors {
			fmt.Printf("  %s\n", err.Error())
		}
		return count
	}
	return 0
}

func fixHardcodedDateFormats() int {
	dateResult, err := validation.FixHardcodedDateFormats()
	if err != nil {
		return 0
	}
	if dateResult.HasErrors() {
		count := len(dateResult.Errors)
		fmt.Println("\n✅ Fixed date formats:")
		for _, err := range dateResult.Errors {
			if strings.Contains(err.Message, "fixed created date format") {
				fmt.Printf("  %s: %s\n", err.File, err.Message)
			}
		}
		return count
	}
	return 0
}

func fixFieldIssues(cfg *config.Config) int {
	fieldResult, err := validation.FixFieldIssues(cfg)
	if err != nil {
		return 0
	}
	if fieldResult.HasErrors() {
		count := len(fieldResult.Errors)
		fmt.Println("\n✅ Fixed field issues:")
		for _, err := range fieldResult.Errors {
			fmt.Printf("  %s: %s\n", err.File, err.Message)
		}
		return count
	}
	return 0
}

// reportResults reports the final results of the doctor command.
func reportResults(fixedCount int, unfixableErrors []validation.ValidationError) {
	if len(unfixableErrors) > 0 {
		fmt.Println("\n⚠️  Issues requiring manual attention:")
		printUnfixableErrors(unfixableErrors)
		fmt.Println("\nThese issues cannot be automatically fixed and need manual intervention.")
	} else if fixedCount > 0 {
		fmt.Println("\n✅ All fixable issues have been resolved!")
	}

	if fixedCount == 0 && len(unfixableErrors) == 0 {
		fmt.Println("\n✅ All issues have been resolved!")
	}
}

// printUnfixableErrors prints unfixable errors without the "Validation errors found" header.
func printUnfixableErrors(errors []validation.ValidationError) {
	if len(errors) == 0 {
		return
	}

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

	// Print categories without the total count header
	printErrorCategory("Field Validation Errors", fieldErrors, true)
	printErrorCategory("Unknown Fields", unknownFieldErrors, true)
	printErrorCategory("Workflow Errors", workflowErrors, false)
	printErrorCategory("Duplicate ID Errors", duplicateErrors, true)
	printErrorCategory("Parse Errors", parseErrors, true)
	printErrorCategory("Other Errors", otherErrors, true)
}

// getUnfixableErrors filters out errors that can be automatically fixed.
func getUnfixableErrors(errors []validation.ValidationError) []validation.ValidationError {
	var unfixable []validation.ValidationError

	for _, err := range errors {
		// These can be fixed automatically:
		// - Duplicate IDs (handled by FixDuplicateIDs)
		// - Invalid created date format (handled by FixHardcodedDateFormats)
		// - Field validation issues (handled by FixFieldIssues)

		// These cannot be fixed automatically:
		// - Workflow violations (user needs to decide which item to move)
		// - Invalid status values (user needs to choose correct status)
		// - Invalid ID format (user needs to fix the ID)
		// - Missing required fields without defaults (user needs to provide values)
		// - Unknown fields in strict mode (user needs to add to config or remove field)
		// - Parse errors (file corruption, user needs to fix)

		if err.File == workflowErrorFile {
			// Workflow errors can't be auto-fixed
			unfixable = append(unfixable, err)
		} else if strings.Contains(err.Message, "invalid status") {
			// Invalid status needs user decision
			unfixable = append(unfixable, err)
		} else if strings.Contains(err.Message, "invalid ID format") {
			// Invalid ID format needs user to fix
			unfixable = append(unfixable, err)
		} else if strings.Contains(err.Message, "missing required field") && !strings.Contains(err.Message, "default") {
			// Missing required fields without defaults
			unfixable = append(unfixable, err)
		} else if strings.Contains(err.Message, "unknown fields found") {
			// Unknown fields in strict mode
			unfixable = append(unfixable, err)
		} else if strings.HasPrefix(err.Message, "failed to parse") {
			// Parse errors
			unfixable = append(unfixable, err)
		}
		// Note: duplicate IDs, date formats, and field validation issues are fixable
		// so we don't include them in unfixable
	}

	return unfixable
}
