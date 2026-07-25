package transactions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions/acquisitions"
)

// This file groups the TRADE ops (the trade itself + the commissioner's Week-9 deadline toggle).
// Split from request.go to keep both within the 400-line cap (AD-14/AD-17 pre-splits).

// KindSetTradeDeadline is the commissioner's trade-deadline toggle — see SetTradeDeadline below.
const KindSetTradeDeadline Kind = "SET_TRADE_DEADLINE"

// PlayerMove is one leg of a trade: which player goes to which franchise.
type PlayerMove struct {
	MFLID         string
	ToFranchiseID string
}

// Trade reassigns a set of players between franchises atomically (an N-leg swap). Every leg
// lands or none does. PicksNote and Rationale are ALPHA scope (panel-locked): PicksNote is
// free-text and deliberately UNVALIDATED — there is no pick-ownership ledger yet, and building
// one now would be premature (post-Alpha work). Rationale is required — every trade must carry
// the commissioner's reason, since (unlike a player move alone) the Trade event itself is now a
// first-class audit record (see LogTradeNote), not just its player-move side effects.
type Trade struct {
	Moves     []PlayerMove
	PicksNote string
	Rationale string
}

func (Trade) Kind() Kind { return KindTrade }
func (Trade) sealed()    {}

// maxTradeLegs caps a single trade's legs. Real trades are a handful of players; the ceiling is
// a boundary guard so a malformed/hostile request can't allocate a giant slice or run thousands
// of UPDATEs in one tx (GLM-B7a — unbounded req.Moves).
const maxTradeLegs = 256

// validate rejects a structurally broken trade BEFORE a transaction is opened: no legs, a blank
// player/target, the same player moved twice in one trade (ambiguous — the last write would
// silently win), or a missing Rationale. PicksNote is intentionally NOT validated — Alpha scope
// is free-text only.
func (t Trade) validate() error {
	if len(t.Moves) == 0 {
		return fmt.Errorf("transactions: trade has no moves")
	}
	if len(t.Moves) > maxTradeLegs {
		return fmt.Errorf("transactions: trade has %d moves, exceeds max %d", len(t.Moves), maxTradeLegs)
	}
	if strings.TrimSpace(t.Rationale) == "" {
		return fmt.Errorf("transactions: trade requires a rationale")
	}
	seen := make(map[string]struct{}, len(t.Moves))
	for i, m := range t.Moves {
		if strings.TrimSpace(m.MFLID) == "" {
			return fmt.Errorf("transactions: trade move %d has an empty player id", i)
		}
		if strings.TrimSpace(m.ToFranchiseID) == "" {
			return fmt.Errorf("transactions: trade move %d (player %q) has an empty target franchise", i, m.MFLID)
		}
		if _, dup := seen[m.MFLID]; dup {
			return fmt.Errorf("transactions: trade moves player %q more than once", m.MFLID)
		}
		seen[m.MFLID] = struct{}{}
	}
	return nil
}

func (t Trade) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	moves := make([]acquisitions.Move, len(t.Moves))
	franchises := make(map[string]struct{}, len(t.Moves)*2)
	for i, m := range t.Moves {
		moves[i] = acquisitions.Move{MFLID: m.MFLID, ToFranchiseID: m.ToFranchiseID}
		franchises[m.ToFranchiseID] = struct{}{}
		// Capture the SOURCE franchise too (pre-move committed state, via Player — never this tx's
		// own uncommitted writes) so a one-sided trade (e.g. player-for-picks, no player coming
		// back) still records the sending franchise. Read before acquisitions.Trade runs: after the
		// move, Player would report the player's NEW (destination) franchise for both legs.
		if p, ok := w.Player(m.MFLID); ok {
			franchises[p.FranchiseID] = struct{}{}
		}
	}
	if err := acquisitions.Trade(ctx, w, moves); err != nil {
		return applyResult{}, fmt.Errorf("trade: %w", err)
	}
	involved := make([]string, 0, len(franchises))
	for f := range franchises {
		involved = append(involved, f)
	}
	if err := w.LogTradeNote(ctx, t.PicksNote, t.Rationale, involved); err != nil {
		return applyResult{}, fmt.Errorf("trade: log note: %w", err)
	}
	// The per-leg cap movement (each traded contract leaves one cap, joins another) is not yet
	// surfaced pre-commit; it lands on the post-commit refresh. Deadcap-first breakdown slice.
	return applyResult{PlayersAffected: len(moves)}, nil
}

// SetTradeDeadline is the commissioner's §14 Week-9 trade-deadline toggle — mirrors
// SetSigningWindow (request_season.go) exactly, riding the same season_phases.meta sub-phase
// directive mechanism rather than a new phase-policy entry (neither per-week phases nor a
// separate deadline table exist in the state layer, and building either now would be premature).
// Deadline is the cutoff instant; once passed, KindTrade is rejected by the phase gate until the
// commissioner clears it (an empty/zero Deadline clears the block — see AppendTradeDeadline).
// Note is a freeform reason. It touches no players (0) and is legal in every phase.
type SetTradeDeadline struct {
	Deadline time.Time
	Note     string
}

func (SetTradeDeadline) Kind() Kind { return KindSetTradeDeadline }
func (SetTradeDeadline) sealed()    {}

// validate has no request shape to check — a zero Deadline is the valid "clear the deadline"
// input, not an error.
func (SetTradeDeadline) validate() error { return nil }

func (s SetTradeDeadline) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	if err := w.AppendTradeDeadline(ctx, s.Deadline, s.Note); err != nil {
		return applyResult{}, fmt.Errorf("set trade deadline: %w", err)
	}
	return applyResult{}, nil
}
