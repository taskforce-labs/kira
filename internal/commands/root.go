// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kira",
	Short: "A git-based, plaintext productivity tool",
	Long: `Kira is a git-based, plaintext productivity tool designed with both
clankers (LLMs) and meatbags (people) in mind. It uses markdown files, git,
and a lightweight CLI to manage and coordinate work.`,
}

// Execute runs the root command and returns any error encountered.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(moveCmd)
	rootCmd.AddCommand(ideaCmd)
	rootCmd.AddCommand(lintCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(abandonCmd)
	rootCmd.AddCommand(saveCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(usersCmd)
}

func checkWorkDir() error {
	if _, err := os.Stat(".work"); os.IsNotExist(err) {
		return fmt.Errorf("not a kira workspace (no .work directory found). Run 'kira init' first")
	}
	return nil
}
