package main

// Package main provides a multichecker binary that runs a curated set of
// static analyzers over the project sources.
//
// How to run
//
//   - From the repository root:
//       go run ./cmd/staticlint ./...
//     or build a binary:
//       go build -o bin/staticlint ./cmd/staticlint
//       ./bin/staticlint ./...
//
// Included analyzers
//
//   - Standard analyzers from golang.org/x/tools/go/analysis/passes
//     (e.g. asmdecl, assign, atomic, bools, composites, copylock, errorsas,
//     httpresponse, loopclosure, lostcancel, nilfunc, printf, shift, shadow,
//     stringintconv, structtag, testinggoroutine, unreachable, unsafeptr,
//     unusedresult)
//   - All analyzers of the SA class from staticcheck.io (staticcheck)
//   - At least one analyzer from other staticcheck classes (we enable a subset
//     from simple and stylecheck)
//   - Two public analyzers: asciicheck and bidichk
//   - A custom analyzer that forbids direct calls to os.Exit from the main
//     function of package main
//
// The goal is to keep signal high and noise manageable. If the checker reports
// issues, update the code to satisfy the analyzers.

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"strings"

	// standard passes
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"

	// staticcheck
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"

	// extra public analyzers
	"github.com/gostaticanalysis/nilerr"
	"github.com/kyoh86/exportloopref"
	asciicheck "github.com/tdakkota/asciicheck"
	"github.com/timakin/bodyclose/passes/bodyclose"

	// custom analyzer
	"github.com/Hobrus/hobrusmetrics.git/cmd/staticlint/noosexit"
)

func main() {
	var analyzers []*analysisWrapper
	// helper to add analyzers and keep names for potential future filtering
	add := func(a *analysisWrapper) { analyzers = append(analyzers, a) }

	// Collect analyzers from standard passes
	std := []*analysisWrapper{
		wrap(asmdecl.Analyzer),
		wrap(assign.Analyzer),
		wrap(atomic.Analyzer),
		wrap(bools.Analyzer),
		wrap(composite.Analyzer),
		wrap(copylock.Analyzer),
		wrap(deepequalerrors.Analyzer),
		wrap(errorsas.Analyzer),
		wrap(httpresponse.Analyzer),
		wrap(ifaceassert.Analyzer),
		wrap(loopclosure.Analyzer),
		wrap(lostcancel.Analyzer),
		wrap(nilfunc.Analyzer),
		wrap(nilness.Analyzer),
		wrap(printf.Analyzer),
		wrap(shift.Analyzer),
		wrap(shadow.Analyzer),
		wrap(sortslice.Analyzer),
		wrap(stringintconv.Analyzer),
		wrap(structtag.Analyzer),
		wrap(testinggoroutine.Analyzer),
		wrap(tests.Analyzer),
		wrap(unmarshal.Analyzer),
		wrap(unreachable.Analyzer),
		wrap(unsafeptr.Analyzer),
		wrap(unusedresult.Analyzer),
	}
	for _, a := range std {
		add(a)
	}

	// All SA analyzers only
	for _, v := range staticcheck.Analyzers {
		if v.Analyzer != nil && strings.HasPrefix(v.Analyzer.Name, "SA") {
			add(wrap(v.Analyzer))
		}
	}

	// Select a small subset from other classes to satisfy the requirement
	// while keeping noise low.
	// - One from simple
	for _, v := range simple.Analyzers {
		if v.Analyzer != nil && (v.Analyzer.Name == "S1000" || v.Analyzer.Name == "S1009") {
			add(wrap(v.Analyzer))
		}
	}
	// - One from stylecheck (keep conservative)
	for _, v := range stylecheck.Analyzers {
		if v.Analyzer != nil && (v.Analyzer.Name == "ST1005") { // error strings should not be capitalized
			add(wrap(v.Analyzer))
		}
	}

	// Public analyzers (2+)
	add(wrap(bodyclose.Analyzer))
	add(wrap(exportloopref.Analyzer))
	add(wrap(nilerr.Analyzer))
	add(wrap(asciicheck.NewAnalyzer()))

	// Custom analyzer
	add(wrap(noosexit.Analyzer))

	// Unwrap for multichecker
	real := make([]*analysis.Analyzer, 0, len(analyzers))
	for _, a := range analyzers {
		real = append(real, a.Analyzer)
	}
	multichecker.Main(real...)
}

// analysisWrapper is a tiny shim that lets us customize or filter analyzers
// in one place if needed.
type analysisWrapper struct {
	Analyzer *analysis.Analyzer
}

func wrap(a *analysis.Analyzer) *analysisWrapper { return &analysisWrapper{Analyzer: a} }
