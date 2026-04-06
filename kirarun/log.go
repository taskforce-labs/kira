package kirarun

import (
	"io"
	"log"
	"os"
)

// Logger is a thin read-only logging surface for workflows.
type Logger struct {
	std *log.Logger
}

// NewLogger builds a logger that writes to w (defaults to os.Stderr).
func NewLogger(w io.Writer) *Logger {
	if w == nil {
		w = os.Stderr
	}
	return &Logger{std: log.New(w, "", log.LstdFlags)}
}

// Info logs an informational message.
func (l *Logger) Info(msg string) {
	if l == nil || l.std == nil {
		return
	}
	l.std.Print(msg)
}
