// Package kirarun is the workflow API for kira run scripts (loaded under Yaegi).
package kirarun

// Agents is the host-provided surface for non-deterministic agent work.
// Workflows that do not use agents may ignore this parameter.
type Agents struct {
	// Reserved for future host wiring (e.g. Cursor); zero value is valid.
}
