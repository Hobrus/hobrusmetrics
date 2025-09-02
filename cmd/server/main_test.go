package main

import (
	"strings"
	"testing"

	"github.com/Hobrus/hobrusmetrics.git/internal/pkg/testutil"
)

func TestPrintBuildInfo_Defaults(t *testing.T) {
	buildVersion = ""
	buildDate = ""
	buildCommit = ""

	out := testutil.CaptureStdout(func() { printBuildInfo() })
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

	out := testutil.CaptureStdout(func() { printBuildInfo() })
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
