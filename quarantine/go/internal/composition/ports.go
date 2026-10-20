// Package composition assembles engine inputs from the stores (rulebook, params) and per-player
// data. The engine may not import a store, so this package owns that wiring and the engine
// stays a pure function.
//
// It has no Wails coupling and reads the stores through narrow interfaces, so it tests with
// fakes. Values not yet in the params store (peak limits, scarcity, L1 constants) live in
// defaults.go; moving them into the store changes only this package.
package composition

import "github.com/secureprospective/TheWarRoom/internal/store/params"

// ParamReader is what composition needs from the params store.
type ParamReader interface {
	GetCapTiers() (params.CapTiers, error)
	GetGlobal(key string) (float64, error)
	GetPosition(key, position string) (float64, error)
}

// CapReader is what composition needs from the rulebook: the cap amount, which MFL encodes as a
// string.
type CapReader interface {
	GetSalaryCap() string
}
