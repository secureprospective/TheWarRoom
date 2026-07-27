package transactions

import (
	"context"
	"errors"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// RosterPolicy is the roster-composition rule surface the roster-affecting transactions enforce
// against — Session 2's roster/position/taxi/IR limit gate. The rulebook store owns the limit
// data (RosterSize, TaxiSquad, InjuredReserve from MFL's league export, override-aware via
// GetSetting) and the players-DB Lookup owns each player's position; an adapter in the composition
// root (app.go) composes them into this interface, because depguard forbids the transactions
// package from importing either store directly (the three-layer law). A nil policy (tests, an
// unwired caller) disables enforcement — every roster-affecting op then commits as it did before
// Session 2, so existing tests stay green without a policy fixture.
//
// Each method returns 0 to mean "unlimited / unconfigured" — an MFL rosterLimits limit of "0-0"
// (the league has no per-position cap), an unset rosterSize, or a taxi/IR slot count of "0" (the
// Session 0 "taxi/IR off" override) all read as 0, and the enforcement treats 0 as "do not gate
// on this axis". A non-zero value is the inclusive cap (roster of exactly the cap is legal; the
// cap+1th addition is rejected).
type RosterPolicy interface {
	// RosterSize is the per-franchise total roster cap (active + taxi + IR). 0 = unlimited.
	RosterSize() int
	// TaxiSquad is the per-franchise taxi-squad slot cap. 0 = unlimited / taxi disabled.
	TaxiSquad() int
	// InjuredReserve is the per-franchise IR slot cap. 0 = unlimited / IR disabled.
	InjuredReserve() int
	// PositionLimit is the per-franchise inclusive roster max for one position (the league's
	// rosterLimits max for that position). 0 = unlimited (no per-position cap configured).
	PositionLimit(pos domain.Position) int
	// Position resolves a player id to its engine position via the players-DB join (the same
	// Lookup the Directory port uses). ok=false for an unknown player (commissioner-created,
	// stale, not yet loaded): the enforcement skips the per-position check for that id (it
	// cannot reject on a limit it cannot resolve) but still enforces roster size.
	Position(ctx context.Context, mflID string) (domain.Position, bool)
}

// rosterAware is an OPTIONAL interface a Request implements when its apply moves players across
// roster/position/taxi/IR boundaries and so must be limit-checked before it runs. The Coordinator
// type-asserts to it inside WriteTx (after gatePhase, before apply); a Request that does not
// implement it (Tag, Extension, Restructure, CapRelief, calendar ops, corrections — none change
// roster composition) is never limit-checked. This keeps the Request interface itself unchanged
// (no signature ripple to the ops that need no gate) while letting the roster-affecting ops opt in.
//
// The check reads COMMITTED pre-op state via r (the Coordinator's writer, which is a Reader by
// embedding) and projects the request's effect against p. It returns nil if the projection
// respects every configured limit, or an error naming the violated limit + the offending franchise
// (a clear rejection for the IPC to surface). r reflects committed state — the single-writer law
// serializes transactions, so nothing can shift underneath this read.
type rosterAware interface {
	enforceRosterLimits(ctx context.Context, r state.Reader, p RosterPolicy) error
}

// errRosterLimit is the sentinel error shape every limit violation returns — a single named kind
// so the IPC layer (and a test) can distinguish "rejected by a roster limit" from any other
// in-tx rejection (a phase gate, a §6 floor). The wrapped detail names the franchise + the limit.
type errRosterLimit struct{ detail string }

func (e *errRosterLimit) Error() string { return e.detail }

// isRosterLimitReject reports whether err is a roster-limit rejection (a limit violation, not a
// phase gate / eligibility / money rejection). Used by tests; available for an IPC that wants to
// color the rejection differently.
func isRosterLimitReject(err error) bool {
	var r *errRosterLimit
	return errors.As(err, &r)
}

// checkRosterSize verifies that adding `adding` players to franchiseID's roster would not exceed
// the policy's RosterSize cap. A 0 cap means unlimited (no gate). `current` is the franchise's
// committed roster size (len of its PlayerState slice). Returns nil if within the cap or unlimited.
func checkRosterSize(p RosterPolicy, franchiseID string, current, adding int) error {
	limit := p.RosterSize()
	if limit <= 0 {
		return nil // unlimited / unconfigured — do not gate
	}
	if current+adding > limit {
		return &errRosterLimit{detail: fmt.Sprintf(
			"roster limit: franchise %q would hold %d players (cap %d) — adding %d exceeds the roster-size limit",
			franchiseID, current+adding, limit, adding)}
	}
	return nil
}

// positionCount scans one franchise's committed roster and counts the players at `target` (the
// position being added). Players whose position the policy cannot resolve are NOT counted toward
// any specific position limit (they are counted toward roster size by checkRosterSize, which does
// not need positions) — the enforcement cannot reject on a limit it cannot resolve for an existing
// rostered player. The scan is O(roster) policy.Position calls, once per affected franchise per op.
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

// checkPositionLimit verifies that adding `adding` players at `pos` to franchiseID would not
// exceed the policy's PositionLimit for that position. A 0 limit means unlimited. `current` is the
// count of players already at `pos` on the franchise's committed roster (from positionCount). If
// `pos` is PosFlag (unresolved), the check is skipped — position limits apply to known positions.
func checkPositionLimit(p RosterPolicy, franchiseID string, pos domain.Position, current, adding int) error {
	if pos == domain.PosFlag {
		return nil
	}
	limit := p.PositionLimit(pos)
	if limit <= 0 {
		return nil // unlimited / unconfigured for this position — do not gate
	}
	if current+adding > limit {
		return &errRosterLimit{detail: fmt.Sprintf(
			"roster limit: franchise %q would hold %d %s players (cap %d) — exceeds the per-position roster limit",
			franchiseID, current+adding, pos, limit)}
	}
	return nil
}

// countByStatus tallies one franchise's committed roster by RosterStatus — the input to the
// taxi/IR slot check. RosterActive players do not count toward taxi or IR slots.
func countByStatus(roster []state.PlayerState, status domain.RosterStatus) int {
	n := 0
	for _, ps := range roster {
		if ps.RosterStatus == status {
			n++
		}
	}
	return n
}
