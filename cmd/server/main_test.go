package main

import (
	"strings"
	"testing"

	"github.com/Hobrus/hobrusmetrics.git/internal/pkg/buildinfo"
	"github.com/Hobrus/hobrusmetrics.git/internal/pkg/testutil"
)

func TestPrintBuildInfo_Defaults(t *testing.T) {
	buildinfo.Version = ""
	buildinfo.Date = ""
	buildinfo.Commit = ""

	out := testutil.CaptureStdout(func() { buildinfo.PrintSelf() })
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
	buildinfo.Version = "v1.2.3"
	buildinfo.Date = "2025-01-02"
	buildinfo.Commit = "abcdef1"

	out := testutil.CaptureStdout(func() { buildinfo.PrintSelf() })
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
