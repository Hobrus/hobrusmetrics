package repository

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/middleware"
	"github.com/Hobrus/hobrusmetrics.git/internal/pkg/retry"

	"github.com/jackc/pgx/v5"
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

// execWithRetry оборачивает вызов Exec в retry.DoWithRetry.
// Если возникла retriable ошибка (например, ошибка соединения класса 08), будет до 3 дополнительных попыток.
func (ps *PostgresStorage) execWithRetry(ctx context.Context, query string, args ...any) error {
	return retry.DoWithRetry(func() error {
		_, err := ps.db.Pool.Exec(ctx, query, args...)
		return err
	})
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
	err := ps.execWithRetry(context.Background(), query, name, float64(value))
	if err != nil {
		fmt.Printf("UpdateGauge error after retries: %v\n", err)
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
	err := ps.execWithRetry(context.Background(), query, name, int64(value))
	if err != nil {
		fmt.Printf("UpdateCounter error after retries: %v\n", err)
	}
}

func (ps *PostgresStorage) UpdateMetricsBatch(batch []middleware.MetricsJSON) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ctx := context.Background()
	// Накопим метрики в map, чтобы не дублировать INSERT для одних и тех же имён
	type dedupKey struct {
		ID    string
		MType string
	}
	type accumVal struct {
		iValue int64
		fValue float64
	}

	deduped := make(map[dedupKey]accumVal)

	for _, m := range batch {
		key := dedupKey{
			ID:    m.ID,
			MType: strings.ToLower(string(m.MType)),
		}
		val, ok := deduped[key]
		if !ok {
			val = accumVal{}
		}
		switch key.MType {
		case "counter":
			if m.Delta != nil {
				val.iValue += *m.Delta
			}
		case "gauge":
			if m.Value != nil {
				// gauge перезаписываем
				val.fValue = *m.Value
			}
		default:
			return fmt.Errorf("unsupported metric type in batch: %q", m.MType)
		}
		deduped[key] = val
	}

	if len(deduped) == 0 {
		return nil
	}

	var values []string
	for key, val := range deduped {
		if key.MType == "counter" {
			values = append(values, fmt.Sprintf(
				"('%s','counter',%d,0)",
				key.ID, val.iValue,
			))
		} else {
			values = append(values, fmt.Sprintf(
				"('%s','gauge',0,%f)",
				key.ID, val.fValue,
			))
		}
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

	// Обёрнем сам insert в retry:
	err := retry.DoWithRetry(func() error {
		tx, beginErr := ps.db.Pool.Begin(ctx)
		if beginErr != nil {
			return beginErr
		}
		// ВАЖНО: проверяем (или игнорируем) ошибку при Rollback
		defer func() {
			if rerr := tx.Rollback(ctx); rerr != nil && rerr != pgx.ErrTxClosed {
				fmt.Printf("rollback error: %v\n", rerr)
			}
		}()

		_, execErr := tx.Exec(ctx, insertQuery)
		if execErr != nil {
			return execErr
		}

		if commitErr := tx.Commit(ctx); commitErr != nil {
			return commitErr
		}
		return nil
	})

	if err != nil {
		fmt.Printf("UpdateMetricsBatch error after retries: %v\n", err)
	}
	return err
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
	rows, err := ps.db.Pool.Query(context.Background(), `SELECT id, fvalue FROM metrics WHERE mtype = 'gauge'`)
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
	rows, err := ps.db.Pool.Query(context.Background(), `SELECT id, ivalue FROM metrics WHERE mtype = 'counter'`)
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
