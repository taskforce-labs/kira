package runevents

import (
	"encoding/json"
	"io"
)

type flusher interface {
	Flush() error
}

// JSONLSink writes one JSON object per line (UTF-8) for each event.
type JSONLSink struct {
	w io.Writer
}

// NewJSONLSink returns a sink that marshals events with encoding/json and appends a newline.
func NewJSONLSink(w io.Writer) *JSONLSink {
	return &JSONLSink{w: w}
}

// Emit implements Sink.
func (j *JSONLSink) Emit(e Event) error {
	if j == nil || j.w == nil {
		return nil
	}
	e = e.withDefaults()
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := j.w.Write(append(b, '\n')); err != nil {
		return err
	}
	if f, ok := j.w.(flusher); ok {
		return f.Flush()
	}
	return nil
}
