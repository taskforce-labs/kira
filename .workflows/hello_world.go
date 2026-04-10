package main

import (
	"fmt"

	"kira/kirarun"
)

type getGreetingOut struct {
	Phrase string `json:"phrase"`
	Style  string `json:"style"`
}

type constructGreetingOut struct {
	Message string `json:"message"`
}

type sayGreetingOut struct {
	Printed bool `json:"printed"`
}

func runStepOneGetGreeting(step *kirarun.Step, ctx *kirarun.Context, agents kirarun.Agents) (getGreetingOut, error) {
	raw, err := kirarun.Do(step, "get_greeting", func(_ kirarun.StepContext) (any, error) {
		_ = ctx
		_ = agents
		return getGreetingOut{Phrase: "G'day", Style: "australia"}, nil
	})
	if err != nil {
		return getGreetingOut{}, err
	}
	var out getGreetingOut
	if err := kirarun.UnmarshalStepData(raw, &out); err != nil {
		return getGreetingOut{}, fmt.Errorf("get_greeting: %w", err)
	}
	return out, nil
}

func runStepTwoConstructGreeting(step *kirarun.Step, in getGreetingOut) (constructGreetingOut, error) {
	raw, err := kirarun.Do(step, "construct_greeting", func(_ kirarun.StepContext) (any, error) {
		return constructGreetingOut{Message: in.Phrase + " world"}, nil
	})
	if err != nil {
		return constructGreetingOut{}, err
	}
	var out constructGreetingOut
	if err := kirarun.UnmarshalStepData(raw, &out); err != nil {
		return constructGreetingOut{}, fmt.Errorf("construct_greeting: %w", err)
	}
	return out, nil
}

func runStepThreeSayGreeting(step *kirarun.Step, in constructGreetingOut) (sayGreetingOut, error) {
	raw, err := kirarun.Do(step, "say_greeting", func(_ kirarun.StepContext) (any, error) {
		fmt.Println(in.Message)
		return sayGreetingOut{Printed: true}, nil
	})
	if err != nil {
		return sayGreetingOut{}, err
	}
	var out sayGreetingOut
	if err := kirarun.UnmarshalStepData(raw, &out); err != nil {
		return sayGreetingOut{}, fmt.Errorf("say_greeting: %w", err)
	}
	return out, nil
}

func Run(ctx *kirarun.Context, step *kirarun.Step, agents kirarun.Agents) error {
	fetched, err := runStepOneGetGreeting(step, ctx, agents)
	if err != nil {
		return err
	}
	built, err := runStepTwoConstructGreeting(step, fetched)
	if err != nil {
		return err
	}
	_, err = runStepThreeSayGreeting(step, built)
	return err
}
