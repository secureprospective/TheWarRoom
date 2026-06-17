// Command ifaceguard is the vettool entry point for the ifaceguard analyzer.
//
// Build it, then run it through `go vet`:
//
//	go build -o bin/ifaceguard ./cmd/ifaceguard
//	go vet -vettool=$(pwd)/bin/ifaceguard ./...
//
// go vet reports any ifaceguard diagnostic and exits non-zero, so it gates a
// `make lint` step and a pre-commit hook exactly like the other linters.
package main

import (
	"ifaceguard"

	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() {
	unitchecker.Main(ifaceguard.Analyzer)
}
