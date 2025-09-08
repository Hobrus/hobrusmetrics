package collector

import (
	"testing"
)

func TestMetricsCollectAndGetAll(t *testing.T) {
	m := NewMetrics()
	var pc int64
	m.Collect(&pc)

	if pc == 0 {
		t.Fatalf("expected poll count to increase")
	}
	all := m.GetAll()
	// A few key runtime fields should be present
	if _, ok := all["Alloc"]; !ok {
		t.Fatalf("expected Alloc present")
	}
	if _, ok := all["PollCount"]; !ok {
		t.Fatalf("expected PollCount present")
	}
	if _, ok := all["RandomValue"]; !ok {
		t.Fatalf("expected RandomValue present")
	}
}

func TestMetricsCollectSystemMetrics(t *testing.T) {
	m := NewMetrics()
	m.CollectSystemMetrics()
	all := m.GetAll()
	// Keys may or may not be present depending on platform, but maps should be safe to read
	_ = all
}
