package transactions

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// phasePolicy is the DECLARATIVE season-phase eligibility policy (Vision-2026 D3): which season
// phases an op_kind is legal in, and whether the kind has a policy at all. Encoded as a pure
// step function (the house idiom, like PositionFloor) rather than a package-level map. Execute
// enforces it as the FIRST step inside the spanning transaction, so the check is atomic with the
// mutation it guards and reads the committed current phase (race-free under the single-writer law).
//
// It is DEFAULT-DENY: an op_kind that falls through to `default` returns ok=false and is rejected
// (a new op must be classified here deliberately, never silently allowed — proven by
// TestExecute_DefaultDenyUnknownKind).
//
// v1 policy (locked with Christopher, expert-panel triaged): every shipped op is legal in ALL
// phases — no behavior change to already-verified ops — with the real windows (the §14 Week-9
// trade deadline, offseason-only tags/extensions) deferred until finer in-season phases exist
// (added as one enum constant + one case, no schema change). The ONE phase-restricted op is
// BUYOUT (§12), offseason-only. ADVANCE_PHASE is legal in every phase — it is the op that
// changes the phase.
func phasePolicy(kind Kind) ([]domain.Phase, bool) {
	switch kind {
	case KindTrade, KindRosterStatus, KindWaiver, KindRestructure, KindTag, KindExtension, KindAdvancePhase,
		KindRetirement, KindDeath, KindCapRelief:
		// §13 special situations (retirement, death, cap relief) can happen in any phase — a
		// player retires or dies whenever, and a commissioner cap-relief appeal is not
		// phase-bound. Only §12 buyout is offseason-restricted in v1.
		return allPhases(), true
	case KindBuyout:
		// §12: buyouts are OFFSEASON-only. This is the one phase-restricted op in v1.
		return []domain.Phase{domain.PhaseOffseason}, true
	case KindRolloverSeason:
		// §14: the season rollover is legal ONLY from PLAYOFFS — it moves PLAYOFFS(N)→OFFSEASON(N+1),
		// so any other from-phase would strand the current season's ledgers (Season_Rollover_Design D5).
		return []domain.Phase{domain.PhasePlayoffs}, true
	case KindSign:
		// §6: a free-agency signing is legal in the SIGNING WINDOW (v1 = offseason + regular season).
		return signingWindow(), true
	default:
		return nil, false
	}
}

// signingWindow returns the phases in which a free-agency SIGN is legal. v1 = OFFSEASON +
// REGULAR_SEASON: playoffs are blocked to protect postseason competitive integrity (no churning
// the pool to patch or block during the bracket).
//
// FORWARD SEAM (Free_Agency_Design Q4, Christopher): a future first-class COMMISSIONER-DISCRETION
// UFA calendar is coming — the UFA window CLOSES when the Super Bowl goes live, STAYS closed until
// the commissioner reopens it, and STAYS open until it closes again (finer than the 3-phase model;
// the season_phases.meta slot carries that granularity). This function is the SINGLE point that
// window plugs into: replace the hardcoded phase set with the commissioner-toggled window state
// and neither the SIGN handler nor gatePhase changes.
func signingWindow() []domain.Phase {
	return []domain.Phase{domain.PhaseOffseason, domain.PhaseRegularSeason}
}

// allPhases is the "legal in every phase" allow-list. A new phase added to the enum applies to
// every v1 no-restriction op automatically.
func allPhases() []domain.Phase {
	return []domain.Phase{domain.PhaseOffseason, domain.PhaseRegularSeason, domain.PhasePlayoffs}
}

// gatePhase enforces the op→phase policy for one request inside the transaction. It denies an
// unmapped op_kind (default-deny) and an op whose current phase is not in its allow-list, reading
// the phase once, up front, before any domain mutation runs.
func gatePhase(ctx context.Context, w state.TxWriter, kind Kind) error {
	allowed, ok := phasePolicy(kind)
	if !ok {
		return fmt.Errorf("transactions: op %q has no season-phase policy (default-deny) — classify it in phasePolicy", kind)
	}
	cur, err := w.CurrentPhase(ctx)
	if err != nil {
		return fmt.Errorf("transactions: phase gate for %q: %w", kind, err)
	}
	for _, p := range allowed {
		if p == cur {
			return nil
		}
	}
	return fmt.Errorf("transactions: op %q is not permitted in phase %q", kind, cur)
}
