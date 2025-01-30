package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/middleware"
	"github.com/Hobrus/hobrusmetrics.git/internal/pkg/retry"
)

// PostgresStorage — хранит counter в ivalue, а gauge как строку в grawvalue
type PostgresStorage struct {
	db *DBConnection
	mu sync.RWMutex
}

// Новая структура таблицы (пример):
//
// CREATE TABLE IF NOT EXISTS metrics (
//
//	id TEXT PRIMARY KEY,
//	mtype TEXT NOT NULL,          -- 'counter' | 'gauge'
//	ivalue BIGINT DEFAULT 0,      -- для counter
//	grawvalue TEXT DEFAULT ''     -- для gauge (сырая строка, наподобие "123.45")
//
// );
//
// (fvalue, если был, можно убрать или оставить неиспользуемым)
func (db *DBConnection) CreateMetricsTable(ctx context.Context) error {
	if db == nil || db.Pool == nil {
		return fmt.Errorf("database not configured")
	}
	schema := `
	CREATE TABLE IF NOT EXISTS metrics (
		id TEXT PRIMARY KEY,
		mtype TEXT NOT NULL,
		ivalue BIGINT DEFAULT 0,
		grawvalue TEXT DEFAULT ''
	);
	`
	_, err := db.Pool.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to create metrics table: %w", err)
	}
	return nil
}

func NewPostgresStorage(dbConn *DBConnection) (*PostgresStorage, error) {
	ps := &PostgresStorage{db: dbConn}
	if err := dbConn.CreateMetricsTable(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to create metrics table: %w", err)
	}
	return ps, nil
}

// ========== gauge ==========

func (ps *PostgresStorage) UpdateGaugeRaw(name, rawValue string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// проверяем парсинг
	if _, err := strconv.ParseFloat(rawValue, 64); err != nil {
		return fmt.Errorf("invalid gauge value: %w", err)
	}

	query := `
	INSERT INTO metrics (id, mtype, grawvalue)
	VALUES ($1, 'gauge', $2)
	ON CONFLICT (id) DO UPDATE
	  SET mtype='gauge',
	      grawvalue = EXCLUDED.grawvalue;
	`
	return ps.execWithRetry(context.Background(), query, name, rawValue)
}

func (ps *PostgresStorage) GetGaugeRaw(name string) (string, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var rawValue string
	var mtype string
	query := `SELECT mtype, grawvalue FROM metrics WHERE id = $1;`
	err := ps.db.Pool.QueryRow(context.Background(), query, name).Scan(&mtype, &rawValue)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", false
		}
		fmt.Printf("GetGaugeRaw error: %v\n", err)
		return "", false
	}
	if mtype != "gauge" {
		return "", false
	}
	return rawValue, true
}

// ========== counter ==========

func (ps *PostgresStorage) UpdateCounter(name string, value Counter) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	query := `
	INSERT INTO metrics (id, mtype, ivalue)
	VALUES ($1, 'counter', $2)
	ON CONFLICT (id) DO UPDATE
	  SET mtype='counter',
	      ivalue = metrics.ivalue + EXCLUDED.ivalue;
	`
	err := ps.execWithRetry(context.Background(), query, name, int64(value))
	if err != nil {
		fmt.Printf("UpdateCounter error after retries: %v\n", err)
	}
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

// ========== batch update ==========

func (ps *PostgresStorage) UpdateMetricsBatch(batch []middleware.MetricsJSON) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ctx := context.Background()
	type dedupKey struct {
		ID    string
		MType string
	}
	type accumVal struct {
		iValue   int64
		gRawVal  string
		override bool // gauge перезаписываем
	}

	deduped := make(map[dedupKey]*accumVal)

	for _, m := range batch {
		key := dedupKey{
			ID:    m.ID,
			MType: strings.ToLower(string(m.MType)),
		}
		if _, ok := deduped[key]; !ok {
			deduped[key] = &accumVal{}
		}
		switch key.MType {
		case "counter":
			if m.Delta != nil {
				deduped[key].iValue += *m.Delta
			}
		case "gauge":
			if m.Value != nil {
				// Преобразуем float в строку (или можно хранить исходную строку m, если есть)
				gstr := strconv.FormatFloat(*m.Value, 'g', 17, 64)
				// Или, если хотите вообще НЕ терять исходную строку,
				// тогда надо расширять middleware, чтобы она не теряла raw.
				deduped[key].gRawVal = gstr
				deduped[key].override = true
			}
		default:
			return fmt.Errorf("unsupported metric type in batch: %q", m.MType)
		}
	}

	if len(deduped) == 0 {
		return nil
	}

	// Собираем VALUES для INSERT
	var values []string
	for key, val := range deduped {
		if key.MType == "counter" {
			values = append(values, fmt.Sprintf(
				"('%s','counter',%d,'')",
				key.ID, val.iValue,
			))
		} else {
			// gauge
			values = append(values, fmt.Sprintf(
				"('%s','gauge',0,'%s')",
				key.ID, val.gRawVal,
			))
		}
	}

	insertQuery := `
	INSERT INTO metrics (id, mtype, ivalue, grawvalue)
	VALUES %s
	ON CONFLICT (id) DO UPDATE
	  SET mtype = EXCLUDED.mtype,
	      ivalue = CASE WHEN EXCLUDED.mtype='counter'
	                    THEN metrics.ivalue + EXCLUDED.ivalue
	                    ELSE metrics.ivalue
	               END,
	      grawvalue = CASE WHEN EXCLUDED.mtype='gauge'
	                    THEN EXCLUDED.grawvalue
	                    ELSE metrics.grawvalue
	               END;
	`
	insertQuery = fmt.Sprintf(insertQuery, strings.Join(values, ","))

	err := retry.DoWithRetry(func() error {
		tx, beginErr := ps.db.Pool.Begin(ctx)
		if beginErr != nil {
			return beginErr
		}
		defer func() {
			_ = tx.Rollback(ctx) // игнорируем ошибку, если уже закрыт
		}()

		_, execErr := tx.Exec(ctx, insertQuery)
		if execErr != nil {
			return execErr
		}
		return tx.Commit(ctx)
	})
	if err != nil {
		fmt.Printf("UpdateMetricsBatch error after retries: %v\n", err)
	}
	return err
}

// ========== getAll* ==========

func (ps *PostgresStorage) GetAllGauges() map[string]string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	gauges := make(map[string]string)
	rows, err := ps.db.Pool.Query(context.Background(), `SELECT id, grawvalue FROM metrics WHERE mtype='gauge'`)
	if err != nil {
		fmt.Printf("GetAllGauges error: %v\n", err)
		return gauges
	}
	defer rows.Close()

	for rows.Next() {
		var id, rawVal string
		if err := rows.Scan(&id, &rawVal); err != nil {
			fmt.Printf("GetAllGauges scan error: %v\n", err)
			continue
		}
		gauges[id] = rawVal
	}
	return gauges
}

func (ps *PostgresStorage) GetAllCounters() map[string]Counter {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	counters := make(map[string]Counter)
	rows, err := ps.db.Pool.Query(context.Background(), `SELECT id, ivalue FROM metrics WHERE mtype='counter'`)
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

// Вспомогательная функция для Exec с retry
func (ps *PostgresStorage) execWithRetry(ctx context.Context, query string, args ...any) error {
	return retry.DoWithRetry(func() error {
		_, err := ps.db.Pool.Exec(ctx, query, args...)
		return err
	})
}
