package transactions

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// phasePolicy lists the phases an op is legal in. It is default-deny: a new op must be added here
// deliberately. v1 (Christopher's ruling): every op is legal in every phase except BUYOUT
// (offseason only), ROLLOVER_SEASON (playoffs only) and SIGN (the signing window). The real in-season
// windows wait for finer phases, which are one enum constant and one case each.
func phasePolicy(kind Kind) ([]domain.Phase, bool) {
	switch kind {
	case KindTrade, KindRosterStatus, KindWaiver, KindRestructure, KindTag, KindExtension, KindAdvancePhase,
		KindRetirement, KindDeath, KindCapRelief, KindSetSigningWindow, KindSetTradeDeadline,
		KindScheduleEvent, KindRescheduleEvent, KindCancelEvent, KindCorrect:
		// §13 events happen whenever, and the commissioner sets calendar directives whenever. The
		// directives themselves restrict SIGN and TRADE, inside gatePhase.
		return allPhases(), true
	case KindBuyout:
		// §12: offseason only.
		return []domain.Phase{domain.PhaseOffseason}, true
	case KindRolloverSeason:
		// §14: only from PLAYOFFS; any other from-phase would strand the season's ledgers.
		return []domain.Phase{domain.PhasePlayoffs}, true
	case KindSign:
		// §6: the signing window.
		return signingWindow(), true
	default:
		return nil, false
	}
}

// signingWindow is the phases a SIGN is legal in: offseason and regular season. Playoffs are
// blocked ("closes when the Super Bowl goes live"). The commissioner's window toggle is checked on
// top of this in gatePhase.
func signingWindow() []domain.Phase {
	return []domain.Phase{domain.PhaseOffseason, domain.PhaseRegularSeason}
}

// allPhases is "legal in every phase"; a new phase applies to these ops automatically.
func allPhases() []domain.Phase {
	return []domain.Phase{domain.PhaseOffseason, domain.PhaseRegularSeason, domain.PhasePlayoffs}
}

// LegalOps returns the per-player op kinds legal in phase p, from phasePolicy, so the UI shows only
// legal buttons without re-encoding the policy. It is a coarse filter: commissioner-only ops are
// excluded, and a SIGN shown here can still be rejected if the window is closed.
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

// gatePhase reads the phase once, before any mutation, and denies an unmapped op or one the phase
// disallows.
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
	// SIGN also checks the commissioner's signing window (open by default). It reads committed state,
	// which is correct only because each WriteTx runs one request: a batch path must keep
	// SET_SIGNING_WINDOW in its own transaction, or a SIGN could slip past a window closed earlier in
	// the same one.
	if kind == KindSign {
		closed, werr := w.SigningWindowClosed(ctx)
		if werr != nil {
			return fmt.Errorf("transactions: signing-window gate: %w", werr)
		}
		if closed {
			return fmt.Errorf("transactions: op %q rejected — the free-agency signing window is closed by the commissioner (§6 UFA calendar)", kind)
		}
	}
	// TRADE checks the commissioner's §14 trade deadline the same way.
	if kind == KindTrade {
		passed, werr := w.TradeDeadlinePassed(ctx)
		if werr != nil {
			return fmt.Errorf("transactions: trade-deadline gate: %w", werr)
		}
		if passed {
			return fmt.Errorf("transactions: op %q rejected — the trade deadline has passed (§14)", kind)
		}
	}
	return nil
}

func phaseAllowed(allowed []domain.Phase, cur domain.Phase) bool {
	for _, p := range allowed {
		if p == cur {
			return true
		}
	}
	return false
}
