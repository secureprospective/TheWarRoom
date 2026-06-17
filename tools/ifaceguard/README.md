# ifaceguard

A `go/analysis` vettool that flags empty-interface (`interface{}` / `any`)
escapes in the signatures of **exported** functions and methods.

## Why

The Go type system is what normally turns a layer-boundary violation into a
compile error. A bare `interface{}`/`any` parameter or result defeats that — a
value of any shape crosses the boundary untyped, invisible to the compiler and
to every golangci-lint linter currently enabled (empirically confirmed:
TheWarRoom Fable Friction #10, 2026-06-13 — `interfacebloat` only checks
interface *type declarations* with too many methods, not bare `any` in a
signature). `ifaceguard` closes that one gap and nothing wider.

Scope is deliberately narrow: **exported** funcs/methods only (an unexported
helper using `any` internally is not a cross-package escape), and it descends
into `*T`, `[]T`, `[N]T`, `...T`, `map[K]V`, and `chan T` so a nested escape
(`[]any`, `map[string]interface{}`) is caught too. A named interface
(`io.Reader`) or a constraint interface (with a type set) is **not** the empty
interface and is left alone.

## Build & run

```bash
go build -o bin/ifaceguard ./cmd/ifaceguard
go vet -vettool=$(pwd)/bin/ifaceguard ./...     # run against a target module
```

`go vet` reports any diagnostic and exits non-zero, so it gates `make lint` and
a pre-commit hook like any other linter. In the Go overlay, `make ifaceguard`
(a prerequisite of `make lint`) builds and runs it for you.

## Escape hatch

Suppress a deliberate, legitimate empty-interface boundary with an
`//ifaceguard:allow` directive in the function's doc comment:

```go
//ifaceguard:allow generic marshalling boundary, validated downstream
func Marshal(v any) ([]byte, error) { /* ... */ }
```

The directive is specific to this tool so it cannot be confused with
golangci-lint's `//nolint` or staticcheck's `//lint:ignore`.

## Supply chain

Depends only on `golang.org/x/tools`, pinned by explicit version in `go.mod`
with its hash locked in `go.sum` (committed). To bump: change the version and
re-commit `go.sum`. Never a mutable ref. The `analysistest` suite
(`ifaceguard_test.go` + `testdata/src/a/a.go`) is the regression guard — a
custom analyzer that silently stops firing is the failure mode (Friction #8's
depguard-glob lesson), so this test must stay green in CI.

```bash
go test ./...
```
