package main

import (
	"errors"

	"kira/kirarun"
)

func Run(_ *kirarun.Context, _ *kirarun.Step, _ kirarun.Agents) error {
	return errors.New("planned failure")
}
