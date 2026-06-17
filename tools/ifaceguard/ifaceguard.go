// Package ifaceguard provides a go/analysis Analyzer that reports use of the
// empty interface (interface{} / any) in the signatures of EXPORTED functions
// and methods.
//
// Why this exists. TheWarRoom has hard layer-boundary rules — e.g. "Layer 2 /
// Layer 4 zero scoring leaks": no sub-signal in Film, RAS, or Breakout may
// reference fantasy points or scoring config. The Go type system is what
// normally turns a boundary violation into a *compile* error. A bare
// interface{}/any parameter or result defeats that: a value of any shape
// crosses the boundary untyped — invisible to the compiler AND to every
// golangci-lint linter currently enabled (empirically confirmed: Fable
// Friction #10, 2026-06-13 — interfacebloat only checks interface *type
// declarations* with too many methods, not bare any in a signature). This
// analyzer closes that specific gap and nothing wider.
//
// Scope. Exported funcs/methods only. An unexported helper that uses `any`
// internally is not a cross-package boundary escape, so it is not the risk
// this guard targets; flagging it would be noise.
//
// Escape hatch. A deliberate, legitimate empty-interface boundary (a generic
// JSON/marshalling helper, a printf-like API) is suppressed with an
// `//ifaceguard:allow` directive in the function's doc comment. The directive
// is intentionally specific to this tool so it cannot be confused with
// golangci-lint's //nolint or staticcheck's //lint:ignore.
package ifaceguard

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Doc is the analyzer's help text (shown by `go vet -vettool=… help`).
const Doc = `report empty-interface (interface{}/any) escapes in exported signatures

ifaceguard flags any exported function or method whose parameters or results
use the empty interface (interface{} or any), directly or nested inside a
pointer, slice, array, map, channel, or variadic. The empty interface turns
off the type checker at a package boundary; for a project with hard
layer-boundary rules that is a silent correctness hole. Suppress a deliberate,
legitimate use with an //ifaceguard:allow directive in the func's doc comment.`

// allowDirective suppresses the diagnostic for a single function when present
// in that function's doc comment.
const allowDirective = "//ifaceguard:allow"

// Analyzer is the ifaceguard analyzer, wired for use via singlechecker,
// unitchecker (go vet -vettool), or golangci-lint's module plugin system.
var Analyzer = &analysis.Analyzer{
	Name:     "ifaceguard",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	insp.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if !fn.Name.IsExported() {
			return
		}
		if hasAllowDirective(fn.Doc) {
			return
		}
		checkFieldList(pass, fn.Type.Params, "parameter", fn.Name.Name)
		checkFieldList(pass, fn.Type.Results, "result", fn.Name.Name)
	})

	return nil, nil
}

func checkFieldList(pass *analysis.Pass, list *ast.FieldList, role, fnName string) {
	if list == nil {
		return
	}
	for _, field := range list.List {
		if pos := findEmptyIface(field.Type); pos != token.NoPos {
			pass.Reportf(pos,
				"exported func %s: %s uses the empty interface (interface{}/any); "+
					"give it a concrete type so the compiler enforces the boundary "+
					"(add //ifaceguard:allow to the doc comment if this generic boundary is deliberate)",
				fnName, role)
		}
	}
}

// findEmptyIface returns the position of an empty-interface use anywhere inside
// expr (bare, or nested in *, [], [N], ..., map, chan), or token.NoPos if none.
// A named interface (io.Reader) or a non-empty interface literal is NOT an
// empty interface and is left alone; a constraint interface (with a type set)
// has a non-empty method list and is likewise left alone.
func findEmptyIface(expr ast.Expr) token.Pos {
	switch t := expr.(type) {
	case *ast.InterfaceType:
		if t.Methods == nil || len(t.Methods.List) == 0 {
			return t.Pos()
		}
	case *ast.Ident:
		if t.Name == "any" {
			return t.Pos()
		}
	case *ast.StarExpr:
		return findEmptyIface(t.X)
	case *ast.ArrayType:
		return findEmptyIface(t.Elt)
	case *ast.Ellipsis:
		return findEmptyIface(t.Elt)
	case *ast.ChanType:
		return findEmptyIface(t.Value)
	case *ast.MapType:
		if p := findEmptyIface(t.Key); p != token.NoPos {
			return p
		}
		return findEmptyIface(t.Value)
	}
	return token.NoPos
}

func hasAllowDirective(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, c := range doc.List {
		if strings.HasPrefix(strings.TrimSpace(c.Text), allowDirective) {
			return true
		}
	}
	return false
}
