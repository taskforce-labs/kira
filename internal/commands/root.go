// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"
	"os"

	"kira/internal/config"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kira",
	Short: "A git-based, plaintext productivity tool",
	Long: `Kira is a git-based, plaintext productivity tool designed with both
clankers (LLMs) and meatbags (people) in mind. It uses markdown files, git,
and a lightweight CLI to manage and coordinate work.`,
	PersistentPreRunE: ensureCursorInstall,
}

func ensureCursorInstall(cmd *cobra.Command, _ []string) error {
	// Skip auto-install when user is explicitly running install command or its subcommands
	// Check the command path by walking up the parent chain
	current := cmd
	for current != nil {
		if current.Name() == "install" {
			return nil
		}
		current = current.Parent()
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil // let commands that need config load it and fail
	}
	if err := EnsureCursorSkillsInstalled(cfg); err != nil {
		return err
	}
	if err := EnsureCursorCommandsInstalled(cfg); err != nil {
		return err
	}
	return nil
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
	rootCmd.AddCommand(assignCmd)
	rootCmd.AddCommand(lintCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(abandonCmd)
	rootCmd.AddCommand(saveCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(usersCmd)
	rootCmd.AddCommand(latestCmd)
	rootCmd.AddCommand(sliceCmd)
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(configCmd)
}

func checkWorkDir(cfg *config.Config) error {
	workPath := config.GetWorkFolderPath(cfg)
	if _, err := os.Stat(workPath); os.IsNotExist(err) {
		return fmt.Errorf("not a kira workspace (no %s directory found). Run 'kira init' first", workPath)
	}
	return nil
}
