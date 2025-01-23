package retry

import (
	"errors"
	"net"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgconn"
)

// backoffIntervals описывает интервалы ожидания между повторными попытками.
var backoffIntervals = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// IsRetriableNetError проверяет, является ли ошибка сетевой и «временной».
func IsRetriableNetError(err error) bool {
	var netErr net.Error
	if !errors.As(err, &netErr) {
		return false
	}
	// 1. Проверяем, не вышло ли время (Timeout)
	if netErr.Timeout() {
		return true
	}
	// 2. Дополнительно можно проверить текст ошибки на "connection refused", "connection reset" и т.п.
	lowerMsg := strings.ToLower(err.Error())
	if strings.Contains(lowerMsg, "connection refused") ||
		strings.Contains(lowerMsg, "connection reset") ||
		strings.Contains(lowerMsg, "network is unreachable") {
		return true
	}
	return false
}

// IsRetriablePGError проверяет, является ли ошибка PostgreSQL из категории "08" (Connection Exception).
func IsRetriablePGError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Код ошибки можно посмотреть через pgErr.Code.
		// Class 08 — это Connection Exception (08xxx).
		if len(pgErr.Code) >= 2 && pgErr.Code[:2] == "08" {
			return true
		}
		// Можно расширить логику под ваши нужды
		// if pgErr.Code == pgerrcode.UniqueViolation { ... }
	}
	return false
}

// IsRetriableFileError пример проверки, можно ли считать ошибку при работе с файлом временной.
func IsRetriableFileError(err error) bool {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		// Например, если в тексте ошибки есть "busy" или "temporarily" — считаем её временной.
		lowerMsg := strings.ToLower(pathErr.Err.Error())
		if strings.Contains(lowerMsg, "busy") || strings.Contains(lowerMsg, "temporarily") {
			return true
		}
	}
	return false
}

// DoWithRetry делает до 4 попыток вызвать fn().
func DoWithRetry(fn func() error) error {
	var lastErr error
	for i := 0; i <= len(backoffIntervals); i++ {
		err := fn()
		if err == nil {
			// Успешно
			return nil
		}
		lastErr = err

		// Проверяем, нужно ли повторять
		if !(IsRetriableNetError(err) ||
			IsRetriablePGError(err) ||
			IsRetriableFileError(err)) {
			// Если ошибка не считается "временной" — сразу выходим
			return err
		}

		// Иначе делаем паузу и повторяем (если ещё есть попытки)
		if i < len(backoffIntervals) {
			time.Sleep(backoffIntervals[i])
		}
	}

	return lastErr
}
