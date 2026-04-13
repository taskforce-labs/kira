package main

import "kira/kirarun"

type out struct {
	N int `json:"n"`
}

func Run(_ *kirarun.Context, step *kirarun.Step, _ kirarun.Agents) error {
	_, err := kirarun.Do(step, "a", func(_ kirarun.StepContext) (any, error) {
		return out{N: 1}, nil
	})
	if err != nil {
		return err
	}
	_, err = kirarun.Do(step, "b", func(_ kirarun.StepContext) (any, error) {
		return out{N: 2}, nil
	})
	if err != nil {
		return err
	}
	_, err = kirarun.Do(step, "c", func(_ kirarun.StepContext) (any, error) {
		return out{N: 3}, nil
	})
	return err
}
