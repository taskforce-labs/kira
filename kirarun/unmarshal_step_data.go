package kirarun

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// UnmarshalStepData turns the value returned from kirarun.Do[any] into a concrete struct (or other
// JSON-unmarshalable type) matching steps[].data. It JSON-marshals v and unmarshals into ptr, like a
// stable json.Unmarshal after normalizing through JSON.
//
// ptr must be a non-nil pointer (e.g. &greet), for the same reason as encoding/json.Unmarshal:
// the unmarshaler writes fields into your variable. Passing greet by value would only fill a
// temporary copy, so your greet would stay zero.
//
// Use after Do[any] in Yaegi workflows: on first run v is often a struct inside any; after
// --resume it is often map[string]any. Both round-trip through JSON into the type ptr points to.
func UnmarshalStepData(v, ptr any) error {
	if ptr == nil {
		return fmt.Errorf("UnmarshalStepData: ptr is nil")
	}
	rv := reflect.ValueOf(ptr)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("UnmarshalStepData: ptr must be a non-nil pointer")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("UnmarshalStepData: marshal: %w", err)
	}
	if err := json.Unmarshal(b, ptr); err != nil {
		return fmt.Errorf("UnmarshalStepData: unmarshal: %w", err)
	}
	return nil
}

// UnmarshalStepDataAs is the same as [UnmarshalStepData] but returns a T instead of writing into ptr.
// Prefer it in compiled code; Yaegi registers [UnmarshalStepData] only.
func UnmarshalStepDataAs[T any](v any) (T, error) {
	var zero T
	var out T
	if err := UnmarshalStepData(v, &out); err != nil {
		return zero, err
	}
	return out, nil
}
