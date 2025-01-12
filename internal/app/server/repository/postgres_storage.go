package repository

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/middleware"
)

type PostgresStorage struct {
	db *DBConnection
	mu sync.RWMutex
}

func NewPostgresStorage(dbConn *DBConnection) (*PostgresStorage, error) {
	ps := &PostgresStorage{
		db: dbConn,
	}
	if err := dbConn.CreateMetricsTable(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to create metrics table: %w", err)
	}
	return ps, nil
}

func (ps *PostgresStorage) UpdateGauge(name string, value Gauge) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	query := `
        INSERT INTO metrics (id, mtype, fvalue)
        VALUES ($1, 'gauge', $2)
        ON CONFLICT (id) DO UPDATE
          SET mtype = 'gauge',
              fvalue = EXCLUDED.fvalue; -- gauge перезаписываем
    `
	_, err := ps.db.Pool.Exec(context.Background(), query, name, float64(value))
	if err != nil {
		fmt.Printf("UpdateGauge error: %v\n", err)
	}
}

func (ps *PostgresStorage) UpdateCounter(name string, value Counter) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	query := `
        INSERT INTO metrics (id, mtype, ivalue)
        VALUES ($1, 'counter', $2)
        ON CONFLICT (id) DO UPDATE
          SET mtype = 'counter',
              ivalue = metrics.ivalue + EXCLUDED.ivalue; -- counter накапливаем
    `
	_, err := ps.db.Pool.Exec(context.Background(), query, name, int64(value))
	if err != nil {
		fmt.Printf("UpdateCounter error: %v\n", err)
	}
}

func (ps *PostgresStorage) UpdateMetricsBatch(batch []middleware.MetricsJSON) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ctx := context.Background()
	tx, err := ps.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var values []string
	for _, m := range batch {
		switch m.MType {
		case "counter":
			values = append(values, fmt.Sprintf(
				"('%s','counter',%d,0)",
				m.ID, *m.Delta,
			))
		case "gauge":
			values = append(values, fmt.Sprintf(
				"('%s','gauge',0,%f)",
				m.ID, *m.Value,
			))
		default:
		}
	}

	if len(values) == 0 {
		return nil
	}

	insertQuery := `
	INSERT INTO metrics (id, mtype, ivalue, fvalue)
	VALUES %s
	ON CONFLICT (id) DO UPDATE
	  SET mtype = EXCLUDED.mtype,
	      ivalue = CASE WHEN EXCLUDED.mtype='counter'
	                    THEN metrics.ivalue + EXCLUDED.ivalue
	                    ELSE metrics.ivalue
	               END,
	      fvalue = CASE WHEN EXCLUDED.mtype='gauge'
	                    THEN EXCLUDED.fvalue
	                    ELSE metrics.fvalue
	               END;
	`

	insertQuery = fmt.Sprintf(insertQuery, strings.Join(values, ","))

	_, err = tx.Exec(ctx, insertQuery)
	if err != nil {
		return fmt.Errorf("batch insert error: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit error: %w", err)
	}

	return nil
}

func (ps *PostgresStorage) GetGauge(name string) (Gauge, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var fVal float64
	var mtype string
	query := `SELECT mtype, fvalue FROM metrics WHERE id = $1;`
	err := ps.db.Pool.QueryRow(context.Background(), query, name).Scan(&mtype, &fVal)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, false
		}
		fmt.Printf("GetGauge error: %v\n", err)
		return 0, false
	}
	if mtype != "gauge" {
		return 0, false
	}
	return Gauge(fVal), true
}

func (ps *PostgresStorage) GetCounter(name string) (Counter, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var iVal int64
	var mtype string
	query := `SELECT mtype, ivalue FROM metrics WHERE id = $1;`
	err := ps.db.Pool.QueryRow(context.Background(), query, name).Scan(&mtype, &iVal)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, false
		}
		fmt.Printf("GetCounter error: %v\n", err)
		return 0, false
	}
	if mtype != "counter" {
		return 0, false
	}
	return Counter(iVal), true
}

func (ps *PostgresStorage) GetAllGauges() map[string]Gauge {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	gauges := make(map[string]Gauge)
	query := `SELECT id, fvalue FROM metrics WHERE mtype = 'gauge';`
	rows, err := ps.db.Pool.Query(context.Background(), query)
	if err != nil {
		fmt.Printf("GetAllGauges error: %v\n", err)
		return gauges
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var fVal float64
		if err := rows.Scan(&id, &fVal); err != nil {
			fmt.Printf("GetAllGauges scan error: %v\n", err)
			continue
		}
		gauges[id] = Gauge(fVal)
	}
	return gauges
}

func (ps *PostgresStorage) GetAllCounters() map[string]Counter {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	counters := make(map[string]Counter)
	query := `SELECT id, ivalue FROM metrics WHERE mtype = 'counter';`
	rows, err := ps.db.Pool.Query(context.Background(), query)
	if err != nil {
		fmt.Printf("GetAllCounters error: %v\n", err)
		return counters
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var iVal int64
		if err := rows.Scan(&id, &iVal); err != nil {
			fmt.Printf("GetAllCounters scan error: %v\n", err)
			continue
		}
		counters[id] = Counter(iVal)
	}
	return counters
}

func (ps *PostgresStorage) Shutdown() error {
	return nil
}
