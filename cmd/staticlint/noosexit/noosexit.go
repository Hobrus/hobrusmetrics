package noosexit

// Package noosexit implements an analyzer that forbids direct calls to
// os.Exit from the function main of package main.
//
// Rationale
//
// Directly exiting the process from main makes graceful shutdown and tests
// harder. Prefer returning from main or propagating errors and using log.Fatal
// or structured shutdown logic.
//
// Reported diagnostics
//
//   - any call to os.Exit inside the top-level main() function when the package
//     name is main.

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer is the exported analyzer instance.
var Analyzer = &analysis.Analyzer{
	Name: "noosexit",
	Doc:  "forbid direct calls to os.Exit in main function of package main",
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
	Run: run,
}

const modulePath = "github.com/Hobrus/hobrusmetrics.git"

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg == nil || pass.Pkg.Name() != "main" {
		return nil, nil
	}
	// Apply only to packages within this module to avoid flagging synthetic
	// driver packages created by the analysis framework or other modules.
	if pkgPath := pass.Pkg.Path(); pkgPath == "" || !strings.HasPrefix(pkgPath, modulePath) {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Find the top-level main() function declarations and inspect their bodies.
	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name == nil || fn.Name.Name != "main" || fn.Body == nil {
			return
		}

		// Restrict to source files living under the module's cmd/ directory to
		// avoid flagging generated test mains and other synthetic packages.
		pos := pass.Fset.Position(fn.Pos())
		filename := filepath.ToSlash(pos.Filename)
		if filename == "" || !(strings.Contains(filename, "/cmd/") || strings.Contains(filename, "\\cmd\\")) {
			return
		}

		// Inspect calls inside main()
		ast.Inspect(fn.Body, func(nn ast.Node) bool {
			call, ok := nn.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if sel.Sel == nil || sel.Sel.Name != "Exit" {
				return true
			}

			// Ensure selector is from package os.
			// Two strategies:
			// 1) check X is a PkgName referring to os
			// 2) check the function object of Sel belongs to package os
			if id, ok := sel.X.(*ast.Ident); ok {
				if obj, ok := pass.TypesInfo.Uses[id].(*types.PkgName); ok {
					if obj.Imported().Path() == "os" {
						pass.Reportf(call.Lparen, "запрещён прямой вызов os.Exit в функции main пакета main")
						return true
					}
				}
			}
			if obj := pass.TypesInfo.ObjectOf(sel.Sel); obj != nil {
				if fn, ok := obj.(*types.Func); ok {
					if pkg := fn.Pkg(); pkg != nil && pkg.Path() == "os" && fn.Name() == "Exit" {
						pass.Reportf(call.Lparen, "запрещён прямой вызов os.Exit в функции main пакета main")
						return true
					}
				}
			}
			return true
		})
	})

	return nil, nil
}
