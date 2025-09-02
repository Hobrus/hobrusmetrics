package repository

import (
	"context"
	"testing"
)

func TestCreateMetricsTable_NoDB(t *testing.T) {
	var db *DBConnection // nil
	if err := db.CreateMetricsTable(context.Background()); err == nil {
		t.Fatalf("expected error for nil db")
	}
}

func TestPostgresStorage_ShutdownNoop(t *testing.T) {
	ps := &PostgresStorage{}
	if err := ps.Shutdown(); err != nil {
		t.Fatalf("shutdown should be noop: %v", err)
	}
}
