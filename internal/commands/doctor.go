package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"kira/internal/config"
	"kira/internal/validation"
)

const (
	// Error categories used for grouping validation results
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

// runDoctor validates work items, applies automatic fixes, then reports what
// was fixed and what still needs manual attention.
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

	// Re-validate after applying automatic fixes so that any issues which were
	// successfully fixed (for example by applying default field values) are no
	// longer reported as requiring manual attention.
	postFixResult, err := validateWorkItems(cfg)
	if err != nil {
		return err
	}

	unfixableErrors := getUnfixableErrors(postFixResult.Errors)
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
		successCount := 0
		var failed []validation.ValidationError

		for _, err := range dateResult.Errors {
			switch {
			case strings.Contains(err.Message, "fixed created date format"):
				successCount++
			case strings.Contains(err.Message, "failed to fix created date"):
				failed = append(failed, err)
			}
		}

		if successCount > 0 {
			fmt.Println("\n✅ Fixed date formats:")
			for _, err := range dateResult.Errors {
				if strings.Contains(err.Message, "fixed created date format") {
					fmt.Printf("  %s: %s\n", err.File, err.Message)
				}
			}
		}

		if len(failed) > 0 {
			fmt.Println("\n⚠️  Failed to fix some date formats:")
			for _, err := range failed {
				fmt.Printf("  %s: %s\n", err.File, err.Message)
			}
		}

		return successCount
	}
	return 0
}

func fixFieldIssues(cfg *config.Config) int {
	fieldResult, err := validation.FixFieldIssues(cfg)
	if err != nil {
		return 0
	}
	if fieldResult.HasErrors() {
		successCount := 0
		var failed []validation.ValidationError

		for _, err := range fieldResult.Errors {
			switch {
			case strings.HasPrefix(err.Message, "fixed field"):
				successCount++
			case strings.HasPrefix(err.Message, "failed to fix fields"):
				failed = append(failed, err)
			}
		}

		if successCount > 0 {
			fmt.Println("\n✅ Fixed field issues:")
			for _, err := range fieldResult.Errors {
				if strings.HasPrefix(err.Message, "fixed field") {
					fmt.Printf("  %s: %s\n", err.File, err.Message)
				}
			}
		}

		if len(failed) > 0 {
			fmt.Println("\n⚠️  Failed to fix some field issues:")
			for _, err := range failed {
				fmt.Printf("  %s: %s\n", err.File, err.Message)
			}
		}

		return successCount
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

	// Categorize errors using shared helpers from lint.go so both commands
	// present errors consistently.
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

	printErrorCategory("Field Validation Errors", fieldErrors, true)
	printErrorCategory("Unknown Fields", unknownFieldErrors, true)
	printErrorCategory("Workflow Errors", workflowErrors, false)
	printErrorCategory("Duplicate ID Errors", duplicateErrors, true)
	printErrorCategory("Parse Errors", parseErrors, true)
	printErrorCategory("Other Errors", otherErrors, true)
}

// getUnfixableErrors returns the set of validation errors that remain after all
// automatic fixes have been applied.
//
// The doctor command runs validation, then attempts to automatically fix any
// issues it knows how to handle (duplicate IDs, hardcoded date formats, and
// certain field-level problems such as applying defaults or normalising enum
// case / email format). After those fixers run, we validate again.
//
// At that point, any remaining validation errors are, by definition, issues
// that the doctor command cannot fix automatically and therefore require
// manual attention. Returning all remaining errors here ensures we never claim
// that "All issues have been resolved!" while validation failures still exist.
func getUnfixableErrors(errors []validation.ValidationError) []validation.ValidationError {
	return errors
}
