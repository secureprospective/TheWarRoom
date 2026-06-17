// Package a is the ifaceguard analysistest fixture. Lines carrying an
// expectation comment must produce exactly one matching diagnostic; lines
// without one must produce none.
package a

// --- SHOULD FIRE: exported signatures using the empty interface ---

func BadParam(v interface{}) {} // want `exported func BadParam: parameter uses the empty interface`

func BadReturn() any { return nil } // want `exported func BadReturn: result uses the empty interface`

func BadSlice(xs []any) {} // want `exported func BadSlice: parameter uses the empty interface`

func BadMap(m map[string]interface{}) {} // want `exported func BadMap: parameter uses the empty interface`

func BadPtr(p *interface{}) {} // want `exported func BadPtr: parameter uses the empty interface`

func BadVariadic(args ...any) {} // want `exported func BadVariadic: parameter uses the empty interface`

// BadMethod is an exported method — the receiver does not exempt it.
func (T) BadMethod(v any) {} // want `exported func BadMethod: parameter uses the empty interface`

// --- SHOULD NOT FIRE ---

// helper is unexported: an internal use of any is not a boundary escape.
func helper(v any) any { return v }

// AllowedGeneric is exported and uses any, but the directive suppresses it.
//
//ifaceguard:allow generic marshalling boundary, validated downstream
func AllowedGeneric(v any) any { return v }

// Good uses only concrete types.
func Good(n int, s string) error { return nil }

// GoodReader takes a NON-empty (named) interface — not the empty interface.
func GoodReader(r SomeReader) {}

// GoodConstraint takes a constraint interface (non-empty method/type set).
func GoodConstraint[V interface{ ~int | ~string }](v V) {}

type T struct{}

type SomeReader interface{ Read() error }
