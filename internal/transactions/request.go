package transactions

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions/acquisitions"
)

// Kind names a transaction type — the discriminator carried on a Receipt and logged.
type Kind string

const (
	KindTrade        Kind = "TRADE"
	KindRosterStatus Kind = "ROSTER_STATUS"
)

// Request is a transaction the Coordinator can execute. The concrete types live in THIS
// root package, so a caller (the IPC layer) builds one directly and never imports a
// handler subpackage (the depguard boundary). The set is closed by the unexported
// sealed marker — a new transaction type is added here, deliberately, never externally.
type Request interface {
	Kind() Kind
	validate() error
	// apply runs the transaction's steps against the shared tx writer and returns how
	// many players it touched. It performs no commit — WriteTx owns the transaction.
	apply(ctx context.Context, w state.TxWriter) (int, error)
	sealed()
}

// PlayerMove is one leg of a trade: which player goes to which franchise.
type PlayerMove struct {
	MFLID         string
	ToFranchiseID string
}

// Trade reassigns a set of players between franchises atomically (an N-leg swap). Every
// leg lands or none does.
type Trade struct {
	Moves []PlayerMove
}

func (Trade) Kind() Kind { return KindTrade }
func (Trade) sealed()    {}

// validate rejects a structurally broken trade BEFORE a transaction is opened: no legs,
// a blank player/target, or the same player moved twice in one trade (ambiguous — the
// last write would silently win).
// maxTradeLegs caps a single trade's legs. Real trades are a handful of players; the
// ceiling is a boundary guard so a malformed/hostile request can't allocate a giant
// slice or run thousands of UPDATEs in one tx (GLM-B7a — unbounded req.Moves).
const maxTradeLegs = 256

func (t Trade) validate() error {
	if len(t.Moves) == 0 {
		return fmt.Errorf("transactions: trade has no moves")
	}
	if len(t.Moves) > maxTradeLegs {
		return fmt.Errorf("transactions: trade has %d moves, exceeds max %d", len(t.Moves), maxTradeLegs)
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

func (t Trade) apply(ctx context.Context, w state.TxWriter) (int, error) {
	moves := make([]acquisitions.Move, len(t.Moves))
	for i, m := range t.Moves {
		moves[i] = acquisitions.Move{MFLID: m.MFLID, ToFranchiseID: m.ToFranchiseID}
	}
	if err := acquisitions.Trade(ctx, w, moves); err != nil {
		return 0, fmt.Errorf("trade: %w", err)
	}
	return len(moves), nil
}

// RosterStatusChange moves one player between roster statuses (active ↔ taxi/IR).
type RosterStatusChange struct {
	MFLID  string
	Status domain.RosterStatus
}

func (RosterStatusChange) Kind() Kind { return KindRosterStatus }
func (RosterStatusChange) sealed()    {}

// validate checks only the player id here; the status whitelist is enforced once, in the
// state layer, so the two never drift.
func (r RosterStatusChange) validate() error {
	if strings.TrimSpace(r.MFLID) == "" {
		return fmt.Errorf("transactions: roster-status change has an empty player id")
	}
	return nil
}

func (r RosterStatusChange) apply(ctx context.Context, w state.TxWriter) (int, error) {
	if err := acquisitions.SetStatus(ctx, w, r.MFLID, r.Status); err != nil {
		return 0, fmt.Errorf("roster status: %w", err)
	}
	return 1, nil
}
