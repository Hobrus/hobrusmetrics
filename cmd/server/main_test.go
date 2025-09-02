package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	os.Stdout = w
	defer func() {
		_ = w.Close()
		os.Stdout = orig
	}()

	fn()
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestPrintBuildInfo_Defaults(t *testing.T) {
	buildVersion = ""
	buildDate = ""
	buildCommit = ""

	out := captureStdout(t, func() { printBuildInfo() })
	if !strings.Contains(out, "Build version: N/A") {
		t.Fatalf("expected version N/A, got: %q", out)
	}
	if !strings.Contains(out, "Build date: N/A") {
		t.Fatalf("expected date N/A, got: %q", out)
	}
	if !strings.Contains(out, "Build commit: N/A") {
		t.Fatalf("expected commit N/A, got: %q", out)
	}
}

func TestPrintBuildInfo_WithValues(t *testing.T) {
	buildVersion = "v1.2.3"
	buildDate = "2025-01-02"
	buildCommit = "abcdef1"

	out := captureStdout(t, func() { printBuildInfo() })
	if !strings.Contains(out, "Build version: v1.2.3") {
		t.Fatalf("expected version printed, got: %q", out)
	}
	if !strings.Contains(out, "Build date: 2025-01-02") {
		t.Fatalf("expected date printed, got: %q", out)
	}
	if !strings.Contains(out, "Build commit: abcdef1") {
		t.Fatalf("expected commit printed, got: %q", out)
	}
}
