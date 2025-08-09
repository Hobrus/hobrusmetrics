package retry

import (
    "errors"
    "os"
    "testing"
    "time"
)

// fakeNetErr implements net.Error
type fakeNetErr struct{ timeout bool }

func (e fakeNetErr) Error() string   { return "fake net err" }
func (e fakeNetErr) Timeout() bool   { return e.timeout }
func (e fakeNetErr) Temporary() bool { return true }

func TestIsRetriableNetError(t *testing.T) {
    if IsRetriableNetError(errors.New("plain")) {
        t.Fatalf("expected false for non-net error")
    }
    if !IsRetriableNetError(fakeNetErr{timeout: true}) {
        t.Fatalf("expected true for timeout net error")
    }
}

func TestIsRetriableFileError(t *testing.T) {
    pathErr := &os.PathError{Op: "open", Path: "x", Err: errors.New("device busy")}
    if !IsRetriableFileError(pathErr) {
        t.Fatalf("expected true for busy PathError")
    }
    if IsRetriableFileError(errors.New("other")) {
        t.Fatalf("expected false for non-PathError")
    }
}

func TestDoWithRetry(t *testing.T) {
    // speed up: shrink backoff for test
    old := backoffIntervals
    backoffIntervals = []time.Duration{1 * time.Millisecond, 1 * time.Millisecond}
    defer func() { backoffIntervals = old }()

    attempts := 0
    err := DoWithRetry(func() error {
        attempts++
        if attempts < 3 {
            // return a fake net timeout error to trigger retries
            return fakeNetErr{timeout: true}
        }
        return nil
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if attempts != 3 {
        t.Fatalf("expected 3 attempts, got %d", attempts)
    }
}


