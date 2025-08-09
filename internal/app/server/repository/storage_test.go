package repository

import "testing"

func TestMemStorageGaugeAndCounter(t *testing.T) {
    m := NewMemStorage()

    // Gauge ok
    if err := m.UpdateGaugeRaw("G", "10.5"); err != nil {
        t.Fatalf("unexpected: %v", err)
    }
    if v, ok := m.GetGaugeRaw("G"); !ok || v != "10.5" {
        t.Fatalf("expected gauge G=10.5, got %q ok=%v", v, ok)
    }

    // Gauge invalid
    if err := m.UpdateGaugeRaw("B", "x"); err == nil {
        t.Fatalf("expected error for invalid gauge value")
    }

    // Counter accumulate
    m.UpdateCounter("C", 5)
    m.UpdateCounter("C", 7)
    if v, ok := m.GetCounter("C"); !ok || v != 12 {
        t.Fatalf("expected counter C=12, got %d ok=%v", v, ok)
    }

    gs := m.GetAllGauges()
    if gs["G"] != "10.5" {
        t.Fatalf("expected G in GetAllGauges")
    }
    cs := m.GetAllCounters()
    if cs["C"] != 12 {
        t.Fatalf("expected C in GetAllCounters")
    }
}


