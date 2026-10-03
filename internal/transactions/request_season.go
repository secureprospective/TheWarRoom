package transactions

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// AdvancePhase moves the season phase to To. Any real target is allowed (correction); a no-op is
// rejected in the store. It never moves the season number: that is RolloverSeason's job.
type AdvancePhase struct {
	To   domain.Phase
	Note string
}

func (AdvancePhase) Kind() Kind { return KindAdvancePhase }
func (AdvancePhase) sealed()    {}

// Unknown phases fail here; the no-op check needs committed state and runs in apply.
func (a AdvancePhase) validate() error {
	if !a.To.Valid() {
		return fmt.Errorf("transactions: advance-phase target %q is not a known phase", a.To)
	}
	return nil
}

func (a AdvancePhase) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	if err := w.AppendPhaseTransition(ctx, a.To, a.Note); err != nil {
		return applyResult{}, fmt.Errorf("advance phase: %w", err)
	}
	return applyResult{}, nil
}

// RolloverSeason moves PLAYOFFS(N) to OFFSEASON(N+1). It is the only op that moves the season, only
// forward, and one-way in v1: no reverse op exists, and moving the phase back never rewinds the
// year.
type RolloverSeason struct {
	Note string
}

func (RolloverSeason) Kind() Kind { return KindRolloverSeason }
func (RolloverSeason) sealed()    {}

func (RolloverSeason) validate() error { return nil }

func (r RolloverSeason) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	if err := w.RolloverSeason(ctx, r.Note); err != nil {
		return applyResult{}, fmt.Errorf("season rollover: %w", err)
	}
	return applyResult{}, nil
}

// SetSigningWindow opens or closes the §6 signing window without changing the phase. A closed
// window blocks every SIGN; it stays as set until the next toggle, across phases and rollovers.
type SetSigningWindow struct {
	Open bool
	Note string
}

func (SetSigningWindow) Kind() Kind { return KindSetSigningWindow }
func (SetSigningWindow) sealed()    {}

func (SetSigningWindow) validate() error { return nil }

func (s SetSigningWindow) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	if err := w.AppendSigningWindow(ctx, s.Open, s.Note); err != nil {
		return applyResult{}, fmt.Errorf("set signing window: %w", err)
	}
	return applyResult{}, nil
}
