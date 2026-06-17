// ifaceguard — a go/analysis vettool that flags empty-interface (interface{} /
// any) escapes in EXPORTED function and method signatures.
//
// Self-contained tool module (separate from the project's main module) so the
// project's own `go.mod` stays free of analysis-framework dependencies. It is
// invoked as a vettool: `go vet -vettool=$(…)/bin/ifaceguard ./...`.
//
// SHA-PINNING: golang.org/x/tools is pinned to an explicit version below and
// its cryptographic hash is locked in go.sum (committed). Never bump this to a
// mutable ref — re-pin to a new explicit version and re-commit go.sum.
module ifaceguard

go 1.26

require golang.org/x/tools v0.39.0

require (
	golang.org/x/mod v0.30.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
)
