package agent

import "testing"

// Smoke-test: construct agent to ensure wiring does not panic.
func TestNewAgent(t *testing.T) {
	a := NewAgent()
	if a.Config == nil || a.Metrics == nil || a.Sender == nil {
		t.Fatalf("agent fields must be initialized")
	}
}
