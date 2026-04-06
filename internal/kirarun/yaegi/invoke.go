package yaegi

import (
	"fmt"
	"reflect"

	"github.com/traefik/yaegi/interp"

	"kira/kirarun"
)

// InvokeRun calls the interpreted main.Run with host-provided values.
func InvokeRun(i *interp.Interpreter, ctx *kirarun.Context, step *kirarun.Step, agents kirarun.Agents) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("workflow panic: %v", r)
		}
	}()
	runV, err := mainRunFunc(i)
	if err != nil {
		return err
	}
	out := runV.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(step),
		reflect.ValueOf(agents),
	})
	if len(out) != 1 {
		return fmt.Errorf("run returned %d values", len(out))
	}
	if out[0].IsNil() {
		return nil
	}
	e, ok := out[0].Interface().(error)
	if !ok {
		return fmt.Errorf("run last return value is not error")
	}
	return e
}

func mainRunFunc(i *interp.Interpreter) (reflect.Value, error) {
	exports := i.Symbols("main")
	for _, syms := range exports {
		if v, ok := syms["Run"]; ok {
			return v, nil
		}
	}
	return reflect.Value{}, fmt.Errorf("main.Run not found")
}
