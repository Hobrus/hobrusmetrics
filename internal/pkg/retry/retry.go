package retry

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgconn"
)

// backoffIntervals описывает интервалы ожидания между повторными попытками.
var backoffIntervals = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// IsRetriableNetError проверяет, является ли ошибка сетевой временной (можно ли её повторять).
func IsRetriableNetError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		// net.Error может иметь методы Temporary() или Timeout().
		// Считаем такую ошибку "временной" и даём шанс на повтор.
		if netErr.Temporary() || netErr.Timeout() {
			return true
		}
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
		// Можно расширить: например, проверять pgerrcode.UniqueViolation и т.п.
		// if pgErr.Code == pgerrcode.UniqueViolation {...}
	}
	return false
}

// IsRetriableFileError пример проверки, можно ли считать ошибку при работе с файлом временной.
// Здесь мы лишь демонстрируем идею: проверяем *os.PathError, а внутри неё код ошибки — EAGAIN, EBUSY и пр.
// В реальном коде можно тщательно уточнить список "временных" ошибок.
func IsRetriableFileError(err error) bool {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		// Для примера: если текст ошибки содержит "resource busy" или "temporarily unavailable",
		// считаем её временной.
		// В реальном проекте можно делать проверки через syscall.Errno.
		lowerMsg := strings.ToLower(pathErr.Err.Error())
		if strings.Contains(lowerMsg, "busy") || strings.Contains(lowerMsg, "temporarily") {
			return true
		}
	}
	return false
}

// DoWithRetry делает до 4 попыток вызвать fn().
// После каждой неудачной попытки, если ошибка "retriable", ждёт время из backoffIntervals.
// Если ошибка не retriable, или попытки закончились — возвращает последнюю ошибку.
func DoWithRetry(fn func() error) error {
	var lastErr error
	for i := 0; i <= len(backoffIntervals); i++ {
		err := fn()
		if err == nil {
			// Успешно
			return nil
		}
		lastErr = err

		// Проверяем, нужно ли вообще повторять.
		if !(IsRetriableNetError(err) ||
			IsRetriablePGError(err) ||
			IsRetriableFileError(err)) {
			// Если ошибка не считается "временной" — выходим сразу.
			return err
		}
		// Иначе делаем паузу и повторяем. Если это была последняя итерация, больше не ждём.
		if i < len(backoffIntervals) {
			time.Sleep(backoffIntervals[i])
		}
	}

	return fmt.Errorf("all retries failed: %w", lastErr)
}
