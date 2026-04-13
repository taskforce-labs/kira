package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"kira/internal/config"
	"kira/internal/kirarun/runner"
	"kira/internal/kirarun/workflow"
)

var runCmd = &cobra.Command{
	Use:   "run <workflow|path.go> [args...]",
	Short: "Run a Yaegi workflow under .workflows/",
	Long: `Runs a Go workflow script with the kira/kirarun API.

The first argument is a workflow name (see workflows.scripts in kira.yml), a path
relative to the project root, or a path to a .go file under the configured workflows root
(default .workflows/). Remaining arguments are exposed to the script via os.Args.

Flags:
  --resume             Continue a previous run (session file must exist)
  --auto-retry         Retry until success or SIGINT (same run id)
  --ignore-attempt-limit  Sets ctx.Run.IgnoreAttemptLimit() for this process only
  --run-events         Write runner progress as JSON Lines to a file (truncated each invocation)`,
	Args: cobra.MinimumNArgs(1),
	RunE: runRun,
}

func init() {
	runCmd.Flags().String("resume", "", "Run id to resume (matches .workflows/sessions/<id>.yml)")
	runCmd.Flags().Bool("auto-retry", false, "Retry until Run succeeds or the process is interrupted")
	runCmd.Flags().Bool("ignore-attempt-limit", false, "Workflow may treat attempt limits as non-fatal for this invocation")
	runCmd.Flags().String("run-events", "", "Write runner progress events as JSON Lines to this path (file is truncated at the start of each invocation)")
}

func runRun(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := checkWorkDir(cfg); err != nil {
		return err
	}

	resume, err := cmd.Flags().GetString("resume")
	if err != nil {
		return err
	}
	autoRetry, err := cmd.Flags().GetBool("auto-retry")
	if err != nil {
		return err
	}
	ignoreLimit, err := cmd.Flags().GetBool("ignore-attempt-limit")
	if err != nil {
		return err
	}
	runEventsPath, err := cmd.Flags().GetString("run-events")
	if err != nil {
		return err
	}

	wfPath, displayName, err := workflow.Resolve(cfg, args[0])
	if err != nil {
		return err
	}

	interpArgs := append([]string{"kira-run"}, args...)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rc := &runner.Config{
		ProjectRoot:        cfg.ConfigDir,
		WorkflowPath:       wfPath,
		KiraVersion:        Version,
		ResumeID:           resume,
		AutoRetry:          autoRetry,
		IgnoreAttemptLimit: ignoreLimit,
		ScriptDisplayName:  displayName,
		Stdout:             os.Stdout,
		RunEventsPath:      runEventsPath,
		InterpArgs:         interpArgs,
	}

	if _, err := rc.Execute(ctx); err != nil {
		if err == context.Canceled {
			return fmt.Errorf("interrupted")
		}
		return err
	}
	return nil
}
