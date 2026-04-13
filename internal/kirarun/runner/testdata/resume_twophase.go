package main

import (
	"fmt"
	"os"
	"path/filepath"

	"kira/kirarun"
)

type stepOut struct {
	V int `json:"v"`
}

// Phase-two marker (created by tests): workspace root file ".kira_run_test_phase2".
func Run(ctx *kirarun.Context, step *kirarun.Step, _ kirarun.Agents) error {
	_, err := kirarun.Do(step, "first", func(_ kirarun.StepContext) (any, error) {
		return stepOut{V: 1}, nil
	})
	if err != nil {
		return err
	}
	marker := filepath.Join(ctx.Workspace.Root(), ".kira_run_test_phase2")
	if _, err := os.Stat(marker); err != nil {
		return fmt.Errorf("fail phase 1")
	}
	_, err = kirarun.Do(step, "second", func(_ kirarun.StepContext) (any, error) {
		return stepOut{V: 2}, nil
	})
	return err
}
