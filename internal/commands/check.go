// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"kira/internal/config"
)

const (
	noChecksMessage = "no checks configured; add a `checks` section to kira.yml"
	checkTimeout    = 10 * time.Minute
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run or list project check commands",
	Long: `Run configured check commands (e.g. lint, test, security) from kira.yml in order.
Use --list to print configured checks without running them.
When no checks are configured, exits 0 with an informational message.`,
	RunE: runCheck,
}

func init() {
	checkCmd.Flags().BoolP("list", "l", false, "List configured checks (name and description)")
}

func runCheck(cmd *cobra.Command, _ []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	listFlag, _ := cmd.Flags().GetBool("list")
	if listFlag {
		return runCheckList(cfg)
	}
	return runCheckRun(cfg)
}

func runCheckRun(cfg *config.Config) error {
	if len(cfg.Checks) == 0 {
		fmt.Println(noChecksMessage)
		return nil
	}

	configDir := cfg.ConfigDir
	if configDir == "" {
		configDir = "."
	}

	for _, entry := range cfg.Checks {
		ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
		// #nosec G204 -- command comes from trusted config (kira.yml), not user input
		c := exec.CommandContext(ctx, "sh", "-c", entry.Command)
		c.Dir = configDir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		err := c.Run()
		cancel()
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				_, _ = fmt.Fprintln(os.Stderr, "check command timed out")
				return fmt.Errorf("check %q timed out", entry.Name)
			}
			_, _ = fmt.Fprintf(os.Stderr, "check %q failed: %v\n", entry.Name, err)
			return fmt.Errorf("check %q failed: %w", entry.Name, err)
		}
	}
	return nil
}

func runCheckList(cfg *config.Config) error {
	if len(cfg.Checks) == 0 {
		fmt.Println(noChecksMessage)
		return nil
	}
	// Slice 3: print each check name and description
	for _, entry := range cfg.Checks {
		desc := entry.Description
		if desc == "" {
			desc = "-"
		}
		fmt.Printf("%s\t%s\n", entry.Name, desc)
	}
	return nil
}
