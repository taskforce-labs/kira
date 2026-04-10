package kirarun

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type greetOutUnmarshal struct {
	Message string `json:"message"`
}

func TestUnmarshalStepDataFromStructInAny(t *testing.T) {
	var v any = greetOutUnmarshal{Message: "hello"}
	var out greetOutUnmarshal
	require.NoError(t, UnmarshalStepData(v, &out))
	require.Equal(t, "hello", out.Message)
}

func TestUnmarshalStepDataFromMapStringAny(t *testing.T) {
	v := any(map[string]any{"message": "hello"})
	var out greetOutUnmarshal
	require.NoError(t, UnmarshalStepData(v, &out))
	require.Equal(t, "hello", out.Message)
}

func TestUnmarshalStepDataPtrNil(t *testing.T) {
	err := UnmarshalStepData(greetOutUnmarshal{Message: "x"}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil")
}

func TestUnmarshalStepDataPtrNotPointer(t *testing.T) {
	err := UnmarshalStepData(greetOutUnmarshal{Message: "x"}, greetOutUnmarshal{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "pointer")
}

func TestUnmarshalStepDataUnmarshalMismatch(t *testing.T) {
	v := any(map[string]any{"message": 123})
	var out greetOutUnmarshal
	err := UnmarshalStepData(v, &out)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unmarshal")
}

func TestUnmarshalStepDataAsFromMap(t *testing.T) {
	v := any(map[string]any{"message": "hi"})
	out, err := UnmarshalStepDataAs[greetOutUnmarshal](v)
	require.NoError(t, err)
	require.Equal(t, "hi", out.Message)
}
