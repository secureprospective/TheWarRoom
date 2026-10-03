package transactions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions/acquisitions"
)

// KindSetTradeDeadline is the commissioner's trade-deadline toggle.
const KindSetTradeDeadline Kind = "SET_TRADE_DEADLINE"

// PlayerMove is one leg of a trade.
type PlayerMove struct {
	MFLID         string
	ToFranchiseID string
}

// Trade moves a set of players between franchises: every leg lands or none does. Rationale is
// required; PicksNote is unvalidated free text, since there is no pick-ownership ledger yet. The
// trade itself is an audit record (LogTradeNote), not just its player moves.
type Trade struct {
	Moves     []PlayerMove
	PicksNote string
	Rationale string
}

func (Trade) Kind() Kind { return KindTrade }
func (Trade) sealed()    {}

// maxTradeLegs bounds a trade so a malformed request can't run thousands of updates.
const maxTradeLegs = 256

// validate rejects no legs, a blank player or target, the same player moved twice (the last write
// would silently win), or a missing rationale.
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

// enforceRosterLimits checks roster size and per-position caps for every franchise that gains
// players, on the net result (committed − outgoing + incoming): a QB-for-QB swap at the cap is
// legal.
func (t Trade) enforceRosterLimits(ctx context.Context, r state.Reader, p RosterPolicy) error {
	// Resolve each leg's position once. An unresolved position counts toward roster size only.
	legs := make([]tradeLeg, 0, len(t.Moves))
	affected := map[string]struct{}{}
	for _, m := range t.Moves {
		lg := tradeLeg{mflID: m.MFLID, toF: m.ToFranchiseID}
		if pos, ok := p.Position(ctx, m.MFLID); ok {
			lg.pos, lg.posOK = pos, true
		}
		if cur, ok := r.Player(m.MFLID); ok {
			lg.fromF = cur.FranchiseID
		}
		legs = append(legs, lg)
		affected[m.ToFranchiseID] = struct{}{}
		if lg.fromF != "" {
			affected[lg.fromF] = struct{}{} // senders can receive too
		}
	}

	for fid := range affected {
		if err := enforceTradeFranchiseLimits(ctx, r, p, fid, legs); err != nil {
			return err
		}
	}
	return nil
}

// enforceTradeFranchiseLimits tallies one franchise's incoming and outgoing legs and applies the
// net roster-size and per-position checks.
func enforceTradeFranchiseLimits(ctx context.Context, r state.Reader, p RosterPolicy, fid string, legs []tradeLeg) error {
	roster, _ := r.Roster(fid) // empty if the franchise has no roster yet

	var incomingTotal, outgoingTotal int
	incomingByPos := map[domain.Position]int{}
	outgoingByPos := map[domain.Position]int{}
	for _, lg := range legs {
		if lg.toF == fid {
			incomingTotal++
			if lg.posOK {
				incomingByPos[lg.pos]++
			}
		}
		if lg.fromF == fid {
			outgoingTotal++
			if lg.posOK {
				outgoingByPos[lg.pos]++
			}
		}
	}

	// Only a net-gaining franchise can overflow.
	if incomingTotal > outgoingTotal {
		if err := checkRosterSizeFromNet(p, fid, len(roster), outgoingTotal, incomingTotal); err != nil {
			return err
		}
	}
	// A position overflows only where incoming exceeds outgoing.
	for pos, inc := range incomingByPos {
		if out := outgoingByPos[pos]; inc <= out {
			continue
		}
		committedAtPos, err := positionCount(ctx, p, roster, pos)
		if err != nil {
			return err
		}
		if err := checkPositionLimit(p, fid, pos, committedAtPos, inc-outgoingByPos[pos]); err != nil {
			return err
		}
	}
	return nil
}

// checkRosterSizeFromNet checks committed − outgoing + incoming against the cap.
func checkRosterSizeFromNet(p RosterPolicy, franchiseID string, committed, outgoing, incoming int) error {
	limit := p.RosterSize()
	if limit <= 0 {
		return nil
	}
	net := committed - outgoing + incoming
	if net > limit {
		return &errRosterLimit{detail: fmt.Sprintf(
			"roster limit: franchise %q would hold %d players after the trade (cap %d) — exceeds the roster-size limit",
			franchiseID, net, limit)}
	}
	return nil
}

// tradeLeg is one move with its resolved position and current franchise.
type tradeLeg struct {
	mflID string
	toF   string
	fromF string // "" if unrostered
	pos   domain.Position
	posOK bool
}

func (t Trade) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	moves := make([]acquisitions.Move, len(t.Moves))
	franchises := make(map[string]struct{}, len(t.Moves)*2)
	for i, m := range t.Moves {
		moves[i] = acquisitions.Move{MFLID: m.MFLID, ToFranchiseID: m.ToFranchiseID}
		franchises[m.ToFranchiseID] = struct{}{}
		// Record the sending franchise before the move: afterwards Player reports the new one, and a
		// one-sided trade must still log its sender.
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
	// Per-leg cap movement shows after commit.
	return applyResult{PlayersAffected: len(moves)}, nil
}

// SetTradeDeadline is the commissioner's §14 trade deadline, stored as a directive on the
// season-phase log like the signing window. Once it passes, TRADE is rejected until cleared; a zero
// Deadline clears it.
type SetTradeDeadline struct {
	Deadline time.Time
	Note     string
}

func (SetTradeDeadline) Kind() Kind { return KindSetTradeDeadline }
func (SetTradeDeadline) sealed()    {}

// A zero Deadline is valid: it clears the deadline.
func (SetTradeDeadline) validate() error { return nil }

func (s SetTradeDeadline) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	if err := w.AppendTradeDeadline(ctx, s.Deadline, s.Note); err != nil {
		return applyResult{}, fmt.Errorf("set trade deadline: %w", err)
	}
	return applyResult{}, nil
}
