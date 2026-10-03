package transactions

import (
	"context"
	"errors"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// RosterPolicy is the roster-limit rule surface: roster size, taxi and IR slots, and
// per-position maximums. app.go adapts the rulebook and players directory to it, because this
// package may not import either store. 0 means unlimited on that axis; otherwise a limit is
// inclusive (exactly at the cap is legal). A nil policy disables enforcement.
type RosterPolicy interface {
	// RosterSize is the total roster cap (active, taxi and IR).
	RosterSize() int
	// TaxiSquad is the taxi-squad slot cap.
	TaxiSquad() int
	// InjuredReserve is the IR slot cap.
	InjuredReserve() int
	// PositionLimit is the roster max for one position.
	PositionLimit(pos domain.Position) int
	// Position resolves a player's engine position. ok=false for an unknown player: his position
	// check is skipped, roster size still applies.
	Position(ctx context.Context, mflID string) (domain.Position, bool)
}

// rosterAware is implemented by requests that change roster composition. The Coordinator checks
// them after the phase gate and before apply, against committed state; other requests are never
// limit-checked. The error names the franchise and the limit.
type rosterAware interface {
	enforceRosterLimits(ctx context.Context, r state.Reader, p RosterPolicy) error
}

// errRosterLimit marks a roster-limit rejection, distinct from any other rejection.
type errRosterLimit struct{ detail string }

func (e *errRosterLimit) Error() string { return e.detail }

// isRosterLimitReject reports whether err is a roster-limit rejection.
func isRosterLimitReject(err error) bool {
	var r *errRosterLimit
	return errors.As(err, &r)
}

// checkRosterSize fails if adding `adding` players would exceed the roster cap.
func checkRosterSize(p RosterPolicy, franchiseID string, current, adding int) error {
	limit := p.RosterSize()
	if limit <= 0 {
		return nil // unlimited
	}
	if current+adding > limit {
		return &errRosterLimit{detail: fmt.Sprintf(
			"roster limit: franchise %q would hold %d players (cap %d) — adding %d exceeds the roster-size limit",
			franchiseID, current+adding, limit, adding)}
	}
	return nil
}

// positionCount counts the committed roster's players at `target`. Players with an unresolved
// position don't count toward any position limit.
func positionCount(ctx context.Context, p RosterPolicy, roster []state.PlayerState, target domain.Position) (int, error) {
	n := 0
	for _, ps := range roster {
		pos, ok := p.Position(ctx, ps.MFLID)
		if !ok {
			continue
		}
		if pos == target {
			n++
		}
	}
	return n, nil
}

// checkPositionLimit fails if adding `adding` players at pos would exceed its limit. An
// unresolved position (FLAG) is skipped.
func checkPositionLimit(p RosterPolicy, franchiseID string, pos domain.Position, current, adding int) error {
	if pos == domain.PosFlag {
		return nil
	}
	limit := p.PositionLimit(pos)
	if limit <= 0 {
		return nil // unlimited
	}
	if current+adding > limit {
		return &errRosterLimit{detail: fmt.Sprintf(
			"roster limit: franchise %q would hold %d %s players (cap %d) — exceeds the per-position roster limit",
			franchiseID, current+adding, pos, limit)}
	}
	return nil
}

// countByStatus counts a roster by status, for the taxi and IR checks.
func countByStatus(roster []state.PlayerState, status domain.RosterStatus) int {
	n := 0
	for _, ps := range roster {
		if ps.RosterStatus == status {
			n++
		}
	}
	return n
}
