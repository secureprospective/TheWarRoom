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
		KindRetirement, KindDeath, KindCapRelief, KindSetSigningWindow,
		KindScheduleEvent, KindRescheduleEvent, KindCancelEvent:
		// §13 special situations (retirement, death, cap relief) can happen in any phase — a
		// player retires or dies whenever, and a commissioner cap-relief appeal is not
		// phase-bound. SET_SIGNING_WINDOW (§6 UFA calendar) is likewise every-phase — the
		// commissioner sets the window whenever, and it is the sub-phase directive that changes
		// SIGN's window, not a signing itself. Only §12 buyout is offseason-restricted in v1.
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

// signingWindow returns the PHASES in which a free-agency SIGN is legal: OFFSEASON + REGULAR_SEASON,
// with PLAYOFFS blocked to protect postseason competitive integrity (no churning the pool during
// the bracket — this is the automatic "closes when the Super Bowl goes live"). It is the coarse,
// always-on floor of SIGN legality.
//
// The COMMISSIONER-DISCRETION UFA calendar (Free_Agency_Design Q4) layers ON TOP of this in
// gatePhase: within the legal phase set, a commissioner-CLOSED window (SigningWindowClosed) still
// blocks a SIGN, and reopening restores it (the window "stays closed until the commissioner reopens
// it"). That toggle rides season_phases.meta and is set by SET_SIGNING_WINDOW — so the calendar
// plugs in WITHOUT touching the SIGN handler: the phase floor here + the window override in
// gatePhase together are SIGN's single legality predicate.
func signingWindow() []domain.Phase {
	return []domain.Phase{domain.PhaseOffseason, domain.PhaseRegularSeason}
}

// allPhases is the "legal in every phase" allow-list. A new phase added to the enum applies to
// every v1 no-restriction op automatically.
func allPhases() []domain.Phase {
	return []domain.Phase{domain.PhaseOffseason, domain.PhaseRegularSeason, domain.PhasePlayoffs}
}

// LegalOps returns the OPERATOR-facing op kinds that are phase-legal in phase p, straight from the
// single phasePolicy source of truth. It lets the M4 workspace show only phase-legal moves
// (Vision-2026 D1) without re-encoding — and drifting from — the engine's policy: the UI shows a
// button iff the kind is in this set. It reports only the per-player ops the workspace can stage;
// pure commissioner/calendar ops (ADVANCE_PHASE / ROLLOVER_SEASON / SET_SIGNING_WINDOW / CAP_RELIEF /
// RETIREMENT / DEATH) live on their own surfaces and are excluded here. It does NOT apply the finer
// SIGN signing-window override (committed state, read in-tx): a SIGN shown here can still be rejected
// at preview/commit if the commissioner has closed the window — gatePhase remains the authoritative
// gate, this is only the coarse phase filter that hides never-legal ops.
func LegalOps(p domain.Phase) []Kind {
	candidates := []Kind{
		KindRosterStatus, KindWaiver, KindSign, KindTag, KindExtension, KindBuyout, KindRestructure, KindTrade,
	}
	out := make([]Kind, 0, len(candidates))
	for _, k := range candidates {
		if allowed, ok := phasePolicy(k); ok && phaseAllowed(allowed, p) {
			out = append(out, k)
		}
	}
	return out
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
	if !phaseAllowed(allowed, cur) {
		return fmt.Errorf("transactions: op %q is not permitted in phase %q", kind, cur)
	}
	// SIGN alone carries a second, finer gate: the COMMISSIONER UFA-CALENDAR window (§6,
	// Free_Agency_Design Q4). Even inside its legal phase set, a commissioner-closed window blocks
	// a signing. The window rides season_phases.meta and defaults OPEN, so this is a no-op for every
	// league that never touches the calendar — v1 behavior is preserved exactly.
	//
	// LOAD-BEARING (same class as the phase read above): SigningWindowClosed reads COMMITTED state
	// off the read pool, which is correct ONLY because Coordinator.Execute runs exactly one Request
	// per WriteTx — so a SET_SIGNING_WINDOW toggle can never co-reside with a SIGN in the same tx
	// and be missed here. Any future batch/composite path MUST keep SET_SIGNING_WINDOW in its own tx
	// (the same sole-occupancy discipline RolloverSeason documents), or a SIGN would slip through a
	// window closed earlier in the same transaction.
	if kind == KindSign {
		closed, werr := w.SigningWindowClosed(ctx)
		if werr != nil {
			return fmt.Errorf("transactions: signing-window gate: %w", werr)
		}
		if closed {
			return fmt.Errorf("transactions: op %q rejected — the free-agency signing window is closed by the commissioner (§6 UFA calendar)", kind)
		}
	}
	return nil
}

// phaseAllowed reports whether the current phase is in an op's allow-list.
func phaseAllowed(allowed []domain.Phase, cur domain.Phase) bool {
	for _, p := range allowed {
		if p == cur {
			return true
		}
	}
	return false
}
