package transactions

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions/acquisitions"
	"github.com/secureprospective/TheWarRoom/internal/transactions/contracts"
	"github.com/secureprospective/TheWarRoom/internal/transactions/deadcap"
)

// Kind names a transaction type — the discriminator carried on a Receipt and logged.
type Kind string

const (
	KindTrade        Kind = "TRADE"
	KindRosterStatus Kind = "ROSTER_STATUS"
	KindWaiver       Kind = "WAIVER"
	KindRestructure  Kind = "RESTRUCTURE"
	KindTag          Kind = "TAG"
	KindExtension    Kind = "EXTENSION"
	KindAdvancePhase Kind = "ADVANCE_PHASE"
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

// Waiver cuts one player (§8): the releasing franchise loses him from its roster and
// owes the §8 dead-cap penalty (35% × annual salary × remaining years, 50% if
// restructured) against the current season's cap. v1 models the UNCLAIMED cut; a claim
// (which ends the dead-cap obligation and moves the player) arrives with free agency.
type Waiver struct {
	MFLID string
}

func (Waiver) Kind() Kind { return KindWaiver }
func (Waiver) sealed()    {}

// validate rejects only an empty player id here; whether the player is actually rostered
// (and every money figure) is resolved from authoritative state inside apply, never
// trusted from the request.
func (wv Waiver) validate() error {
	if strings.TrimSpace(wv.MFLID) == "" {
		return fmt.Errorf("transactions: waiver has an empty player id")
	}
	return nil
}

func (wv Waiver) apply(ctx context.Context, w state.TxWriter) (int, error) {
	if _, err := deadcap.Waive(ctx, w, wv.MFLID); err != nil {
		return 0, fmt.Errorf("waiver: %w", err)
	}
	return 1, nil
}

// Restructure lowers a player's cap-counting salary by the owner-chosen Move (§11),
// bounded by the tier max ($1M/$2M/$3M by contract-year salary), and flags the contract
// restructured (a later §8 cut then charges 50%). The tier, limits, and every money figure
// are resolved from authoritative state inside apply — the request carries only the intent.
type Restructure struct {
	MFLID string
	Move  domain.Money
}

func (Restructure) Kind() Kind { return KindRestructure }
func (Restructure) sealed()    {}

// validate rejects an empty player id or a non-positive move here; the tier max, the
// eligibility floor, and the per-season/per-contract limits are enforced against real state
// inside apply, never trusted from the request.
func (r Restructure) validate() error {
	if strings.TrimSpace(r.MFLID) == "" {
		return fmt.Errorf("transactions: restructure has an empty player id")
	}
	if r.Move <= 0 {
		return fmt.Errorf("transactions: restructure move must be positive")
	}
	return nil
}

func (r Restructure) apply(ctx context.Context, w state.TxWriter) (int, error) {
	if err := contracts.Restructure(ctx, w, r.MFLID, r.Move); err != nil {
		return 0, fmt.Errorf("restructure: %w", err)
	}
	return 1, nil
}

// Tag applies a §9 franchise tag: the player's salary becomes the top-5-by-position
// league-wide average (floored at 120% of his prior-year salary). The price is NOT a field
// a caller sets — it is resolved authoritatively by Coordinator.ExecuteTag from committed
// state and stored in the unexported price field, so the IPC boundary carries only the
// player id. A zero-price Tag (constructed directly, never resolved) is rejected in apply.
type Tag struct {
	MFLID string
	price domain.Money // resolved by Coordinator.ExecuteTag; unexported so no caller supplies it
}

func (Tag) Kind() Kind { return KindTag }
func (Tag) sealed()    {}

// validate rejects only an empty player id here; the position, the top-5 average, the 120%
// floor, and the per-season limit are all resolved/enforced against authoritative state
// (ExecuteTag + the handler), never trusted from the request.
func (t Tag) validate() error {
	if strings.TrimSpace(t.MFLID) == "" {
		return fmt.Errorf("transactions: tag has an empty player id")
	}
	return nil
}

func (t Tag) apply(ctx context.Context, w state.TxWriter) (int, error) {
	if err := contracts.Tag(ctx, w, t.MFLID, t.price); err != nil {
		return 0, fmt.Errorf("tag: %w", err)
	}
	return 1, nil
}

// Extension applies a §10 contract extension: it appends AddedYears (1..3) new PAID years
// priced at 150% of the player's highest-paid remaining year, raised to the position floor.
// The floor is NOT a caller field — it is resolved authoritatively by
// Coordinator.ExecuteExtension from the player's position and stored in the unexported floor
// field, so the IPC boundary carries only the id and the year count. Every §10 limit (≥1 year
// remaining, ≤6 total years, no prior extension, one per franchise per season) is enforced
// against real state in the handler, never trusted from the request.
type Extension struct {
	MFLID      string
	AddedYears int
	floor      domain.Money // resolved by Coordinator.ExecuteExtension; unexported so no caller supplies it
}

func (Extension) Kind() Kind { return KindExtension }
func (Extension) sealed()    {}

// validate rejects only an empty player id or an out-of-range year count here (the cheap
// pre-transaction gate); the floor, the 150% price, and every eligibility/limit rule are
// resolved and enforced against authoritative state (ExecuteExtension + the handler).
func (e Extension) validate() error {
	if strings.TrimSpace(e.MFLID) == "" {
		return fmt.Errorf("transactions: extension has an empty player id")
	}
	if e.AddedYears < 1 || e.AddedYears > 3 {
		return fmt.Errorf("transactions: extension adds %d years, must be 1..3 (§10)", e.AddedYears)
	}
	return nil
}

func (e Extension) apply(ctx context.Context, w state.TxWriter) (int, error) {
	if err := contracts.Extend(ctx, w, e.MFLID, e.AddedYears, e.floor); err != nil {
		return 0, fmt.Errorf("extension: %w", err)
	}
	return 1, nil
}

// AdvancePhase moves the league-year's season phase to To (Vision-2026 D3) — a first-class,
// commissioner-confirmed transition appended to the append-only season_phases log. It touches
// no players (PlayersAffected 0). Any real target phase is allowed in v1 (correction/rollback);
// a no-op (To == current) is rejected in the store primitive. Note is the commissioner's
// freeform reason (optional). ADVANCE_PHASE is itself legal in every phase (it is the op that
// changes the phase), so the op-eligibility gate never blocks it.
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
