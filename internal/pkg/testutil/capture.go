package testutil

import (
	"bytes"
	"os"
)

// CaptureStdout runs fn while redirecting stdout and returns captured output as string.
func CaptureStdout(fn func()) string {
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return ""
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = orig
	return buf.String()
}
