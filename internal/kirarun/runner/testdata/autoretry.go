package main

import (
	"fmt"

	"kira/kirarun"
)

func Run(ctx *kirarun.Context, _ *kirarun.Step, _ kirarun.Agents) error {
	if ctx.Run.Attempt() < 2 {
		return fmt.Errorf("transient failure")
	}
	return nil
}
