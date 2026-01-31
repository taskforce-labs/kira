// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"kira/internal/config"
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Check work item readiness for review",
	Long:  `Checks if the current work item (in doing folder) has all slices/tasks complete. Warns if any tasks are still open.`,
	Args:  cobra.NoArgs,
	RunE:  runReview,
}

func init() {
	reviewCmd.SilenceUsage = true
}

func runReview(_ *cobra.Command, _ []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}
	path, err := resolveSliceWorkItem("", cfg)
	if err != nil {
		return err
	}
	_, slices, err := loadSlicesFromFile(path, cfg)
	if err != nil {
		return err
	}
	if len(slices) == 0 {
		fmt.Println("No slices in current work item. Ready for review.")
		return nil
	}
	var total, open int
	for _, s := range slices {
		for _, t := range s.Tasks {
			total++
			if !t.Done {
				open++
			}
		}
	}
	if open > 0 {
		fmt.Printf("Warning: %d of %d tasks are still open. Run 'kira slice show' or 'kira slice progress' to see details.\n", open, total)
		return fmt.Errorf("work item has %d open task(s); complete them before review", open)
	}
	fmt.Println("All tasks complete. Ready for review.")
	return nil
}
