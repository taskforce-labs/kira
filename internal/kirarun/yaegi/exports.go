// Package yaegi loads workflow scripts with Yaegi and validates kira/kirarun usage.
package yaegi

import (
	"reflect"
	"time"

	"github.com/traefik/yaegi/interp"

	"kira/kirarun"
)

// KirarunExports returns binary symbols for the kira/kirarun package (Yaegi interp.Use).
func KirarunExports() interp.Exports {
	return interp.Exports{
		"kira/kirarun/kirarun": {
			// Generic function symbol uses any instantiation for reflect registration.
			"Do":           reflect.ValueOf(kirarun.Do[any]),
			"NewLogger":    reflect.ValueOf(kirarun.NewLogger),
			"NewRunHandle": reflect.ValueOf(kirarun.NewRunHandle),
			"NewStep":      reflect.ValueOf(kirarun.NewStep),
			"NewWorkspace": reflect.ValueOf(kirarun.NewWorkspace),

			"Agents":        reflect.ValueOf((*kirarun.Agents)(nil)),
			"CommandsView":  reflect.ValueOf((*kirarun.CommandsView)(nil)),
			"Context":       reflect.ValueOf((*kirarun.Context)(nil)),
			"Logger":        reflect.ValueOf((*kirarun.Logger)(nil)),
			"RunHandle":     reflect.ValueOf((*kirarun.RunHandle)(nil)),
			"SkillsView":    reflect.ValueOf((*kirarun.SkillsView)(nil)),
			"Step":          reflect.ValueOf((*kirarun.Step)(nil)),
			"StepContext":   reflect.ValueOf((*kirarun.StepContext)(nil)),
			"StepPersister": reflect.ValueOf((*kirarun.StepPersister)(nil)),
			"Workspace":     reflect.ValueOf((*kirarun.Workspace)(nil)),

			"_StepPersister": reflect.ValueOf((*stepPersisterWrap)(nil)),
		},
	}
}

// stepPersisterWrap is an interface wrapper for StepPersister (Yaegi).
type stepPersisterWrap struct {
	IValue       interface{}
	WGetStepData func(name string) (data map[string]any, ok bool)
	WPutStep     func(
		name string,
		attempt int,
		startedAt, finishedAt time.Time,
		data map[string]any,
	) error
}

func (w stepPersisterWrap) GetStepData(name string) (data map[string]any, ok bool) {
	return w.WGetStepData(name)
}

func (w stepPersisterWrap) PutStep(name string, attempt int, startedAt, finishedAt time.Time, data map[string]any) error {
	return w.WPutStep(name, attempt, startedAt, finishedAt, data)
}
