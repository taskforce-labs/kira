package kirarun

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type memPersister struct {
	steps map[string]map[string]any
}

func (m *memPersister) GetStepData(name string) (map[string]any, bool) {
	if m.steps == nil {
		return nil, false
	}
	v, ok := m.steps[name]
	return v, ok
}

func (m *memPersister) PutStep(name string, _ int, _, _ time.Time, data map[string]any) error {
	if m.steps == nil {
		m.steps = make(map[string]map[string]any)
	}
	cp := make(map[string]any, len(data))
	for k, v := range data {
		cp[k] = v
	}
	m.steps[name] = cp
	return nil
}

func TestStepDoPersistsAndSkips(t *testing.T) {
	var p memPersister
	run := NewRunHandle(1, false, false)
	step := NewStep(&p, run)

	type out struct {
		N int `json:"n"`
	}

	calls := 0
	v1, err := Do(step, "a", func(_ StepContext) (out, error) {
		calls++
		return out{N: 42}, nil
	})
	require.NoError(t, err)
	require.Equal(t, 42, v1.N)
	require.Equal(t, 1, calls)

	v2, err := Do(step, "a", func(_ StepContext) (out, error) {
		calls++
		return out{N: 99}, nil
	})
	require.NoError(t, err)
	require.Equal(t, 42, v2.N)
	require.Equal(t, 1, calls, "second Do should skip fn")
}

func TestRunHandle(t *testing.T) {
	r := NewRunHandle(3, true, true)
	require.Equal(t, 3, r.Attempt())
	require.True(t, r.IsResume())
	require.True(t, r.IgnoreAttemptLimit())

	r2 := NewRunHandle(0, false, false)
	require.Equal(t, 1, r2.Attempt())
}

func TestStepDoDecodeMismatch(t *testing.T) {
	p := memPersister{steps: map[string]map[string]any{
		"x": {"foo": 123}, // number cannot decode into string field
	}}
	step := NewStep(&p, NewRunHandle(1, false, false))

	type want struct {
		Foo string `json:"foo"`
	}
	_, err := Do(step, "x", func(_ StepContext) (want, error) {
		return want{Foo: "nope"}, nil
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "step \"x\"")
	require.Contains(t, err.Error(), "expected type")
}
