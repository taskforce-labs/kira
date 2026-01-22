// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var latestCmd = &cobra.Command{
	Use:   "latest",
	Short: "Update feature branches with latest trunk changes through iterative conflict resolution",
	Long: `A command that keeps feature branches updated with trunk changes through iterative conflict resolution.

The command first checks for existing merge conflicts, displays them for external LLM resolution,
and only performs fetch/rebase when conflicts are resolved. Since kira supports polyrepo workflows
(managing work across multiple repositories), the command handles rebasing across multiple repos
simultaneously, ensuring consistency and coordination between related repositories.

The command can be called repeatedly to work through conflicts progressively.`,
	Args: cobra.NoArgs,
	RunE: runLatest,
}

func runLatest(_ *cobra.Command, _ []string) error {
	if err := checkWorkDir(); err != nil {
		return err
	}

	// Placeholder: command not yet implemented
	return fmt.Errorf("kira latest is not yet implemented")
}
