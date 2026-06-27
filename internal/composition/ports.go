// Package composition is the boundary B5a deliberately left out: the layer that reads
// the stores (B3b rulebook, B4 params) plus harness-supplied per-player data and
// assembles engine.PlayerInput + engine.Calibration. The pure engine forbids itself
// from importing any store (depguard engine-is-pure); composition is the consumer that
// IS allowed to, so the engine stays a pure function and this package owns the wiring.
//
// It is consumed by BOTH the testing harness and (later) the production app, so it
// carries NO Wails/desktop/runtime coupling — plain Go, both consumers wrap it. It
// depends on the stores through narrow PORT interfaces (below), not concrete types, so
// it is unit-testable with fakes and never needs a live database.
//
// FORWARD-CONTRACT NOTE (this session's "version the boundary" decision): per-position
// calibration values (peak limit, scarcity rank) and the L1 hygiene constants are NOT
// in B4 yet. composition supplies documented defaults for them HERE. When B5b ships its
// per-position tables, only this package changes — the engine contract and the UI do
// not. That is the whole point of putting the seam here.
package composition

import "github.com/secureprospective/TheWarRoom/internal/store/params"

// ParamReader is the slice of the B4 params store composition needs: the cap-tier
// percentages and the global scalars. *params.Store satisfies it structurally, so the
// real store drops in and a fake drops in for tests.
type ParamReader interface {
	GetCapTiers() (params.CapTiers, error)
	GetGlobal(key string) (float64, error)
}

// CapReader is the slice of the B3b rulebook store composition needs: the league salary
// cap AMOUNT. MFL encodes it as a string, so the rulebook hands it back as a string and
// composition parses it (the boundary's job, not the engine's). *rulebook.Store
// satisfies it structurally.
type CapReader interface {
	GetSalaryCap() string
}
