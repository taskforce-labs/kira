package runevents

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHumanSinkKindColumnWidth(t *testing.T) {
	var buf bytes.Buffer
	s := NewHumanSink(&buf)
	require.NoError(t, s.Emit(Event{
		Event:    KindRunStart,
		Workflow: "hello",
		RunID:    "hello-20060102150405",
		Attempt:  1,
	}))
	line := buf.String()
	const prefix = "[kira-run]  "
	require.True(t, strings.HasPrefix(line, prefix))
	rest := line[len(prefix):]
	require.GreaterOrEqual(t, len(rest), humanKindW)
	kindCol := rest[:humanKindW]
	wantKind := KindRunStart + strings.Repeat(" ", humanKindW-len(KindRunStart))
	require.Equal(t, wantKind, kindCol)
	require.Contains(t, line, "workflow=hello")
	require.Contains(t, line, "run-id=hello-20060102150405")
	require.Contains(t, line, "attempt=1")
}
