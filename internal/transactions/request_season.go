package transactions

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// This file groups the SEASON-LIFECYCLE ops (phase advance + season rollover). They share the
// season_phases log as their write surface and are the only ops that move the league through its
// calendar. Split from request.go to keep both files within the 400-line cap (AD-14/AD-17).

// AdvancePhase moves the league-year's season phase to To (Vision-2026 D3) — a first-class,
// commissioner-confirmed transition appended to the append-only season_phases log. It touches
// no players (PlayersAffected 0). Any real target phase is allowed in v1 (correction/rollback);
// a no-op (To == current) is rejected in the store primitive. Note is the commissioner's
// freeform reason (optional). ADVANCE_PHASE is itself legal in every phase (it is the op that
// changes the phase), so the op-eligibility gate never blocks it. It NEVER moves the season int —
// that is ROLLOVER_SEASON's sole job — so a phase advance to OFFSEASON stays within the same year.
type AdvancePhase struct {
	To   domain.Phase
	Note string
}

func (AdvancePhase) Kind() Kind { return KindAdvancePhase }
func (AdvancePhase) sealed()    {}

// validate rejects an unknown target phase before a transaction is opened; the no-op guard
// (To == current) needs the committed phase and so is enforced in apply, atomically.
func (a AdvancePhase) validate() error {
	if !a.To.Valid() {
		return fmt.Errorf("transactions: advance-phase target %q is not a known phase", a.To)
	}
	return nil
}

func (a AdvancePhase) apply(ctx context.Context, w state.TxWriter) (int, error) {
	if err := w.AppendPhaseTransition(ctx, a.To, a.Note); err != nil {
		return 0, fmt.Errorf("advance phase: %w", err)
	}
	return 0, nil
}

// RolloverSeason advances the league from PLAYOFFS(N) to OFFSEASON(N+1) — the season boundary
// (Season_Rollover_Design D5). It is commissioner-confirmed, PLAYOFFS-gated (the one legal
// from-phase, enforced by the op→phase policy), and touches no players (PlayersAffected 0). The
// season int moves ONLY through this op and only forward by one (monotonic); the roster/contract
// snapshot advances with it while the per-year ledgers stay put as audit (D7). Rolling the season
// is high-consequence and one-way in v1 — there is no reverse op; a backward crossing is disallowed
// (a phase rollback via ADVANCE_PHASE never rewinds the year). Note is the commissioner's reason.
type RolloverSeason struct {
	Note string
}

func (RolloverSeason) Kind() Kind { return KindRolloverSeason }
func (RolloverSeason) sealed()    {}

// validate has no request shape to check — a rollover carries only an optional note. Its sole
// precondition (current phase == PLAYOFFS) needs committed state, so it is enforced by the phase
// gate and the store primitive inside the transaction, never trusted from the request.
func (RolloverSeason) validate() error { return nil }

func (r RolloverSeason) apply(ctx context.Context, w state.TxWriter) (int, error) {
	if err := w.RolloverSeason(ctx, r.Note); err != nil {
		return 0, fmt.Errorf("season rollover: %w", err)
	}
	return 0, nil
}

// SetSigningWindow is the commissioner's UFA-CALENDAR toggle (§6, Free_Agency_Design Q4): it OPENS
// or CLOSES the free-agency signing window WITHOUT changing the season phase. A CLOSED window blocks
// every SIGN even in an otherwise-legal phase (the SIGN phase gate consults it); reopening restores
// it. The directive persists until the next toggle — carrying across phase transitions and rollovers
// ("closes when the Super Bowl goes live, stays closed until the commissioner reopens it"). It
// touches no players (0) and is legal in every phase (the commissioner sets the calendar whenever).
// It rides the append-only season_phases log via the meta slot, not a phase transition. Note is a
// freeform reason.
type SetSigningWindow struct {
	Open bool
	Note string
}

func (SetSigningWindow) Kind() Kind { return KindSetSigningWindow }
func (SetSigningWindow) sealed()    {}

// validate has no request shape to check — the toggle carries only a bool and an optional note. The
// redundancy guard (already in the requested state) needs committed state and so is enforced in the
// store primitive, atomically.
func (SetSigningWindow) validate() error { return nil }

func (s SetSigningWindow) apply(ctx context.Context, w state.TxWriter) (int, error) {
	if err := w.AppendSigningWindow(ctx, s.Open, s.Note); err != nil {
		return 0, fmt.Errorf("set signing window: %w", err)
	}
	return 0, nil
}
