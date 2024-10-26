package models

type Metrics struct {
	ID    string   `json:"id"`              // metric name
	MType string   `json:"type"`            // gauge or counter
	Delta *int64   `json:"delta,omitempty"` // counter value
	Value *float64 `json:"value,omitempty"` // gauge value
}
