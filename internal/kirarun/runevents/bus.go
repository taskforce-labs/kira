package runevents

import "sync"

// Sink receives normalized runner events.
type Sink interface {
	Emit(e Event) error
}

// Bus fans out events to all sinks.
type Bus struct {
	mu    sync.Mutex
	sinks []Sink
}

// NewBus returns an empty bus; add sinks before Emit.
func NewBus() *Bus {
	return &Bus{}
}

// AddSink registers a sink (call before Emit from multiple goroutines; AddSink is not concurrency-safe with Emit).
func (b *Bus) AddSink(s Sink) {
	if b == nil || s == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sinks = append(b.sinks, s)
}

// Emit delivers a copy of the event to every sink; timestamps and schema are filled if missing.
func (b *Bus) Emit(e Event) {
	if b == nil {
		return
	}
	e = e.withDefaults()
	b.mu.Lock()
	sinks := append([]Sink(nil), b.sinks...)
	b.mu.Unlock()
	for _, s := range sinks {
		_ = s.Emit(e)
	}
}
