package middleware

import (
	"bytes"
	"encoding/json"
	"testing"
)

type benchMetric struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

func BenchmarkJSONUnmarshal_Update(b *testing.B) {
	v := float64(123.456)
	m := benchMetric{ID: "Alloc", MType: "gauge", Value: &v}
	payload, _ := json.Marshal(m)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var dst MetricsJSON
		dec := json.NewDecoder(bytes.NewReader(payload))
		if err := dec.Decode(&dst); err != nil {
			b.Fatal(err)
		}
	}
}
