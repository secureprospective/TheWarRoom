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
	KindTrade            Kind = "TRADE"
	KindRosterStatus     Kind = "ROSTER_STATUS"
	KindWaiver           Kind = "WAIVER"
	KindRestructure      Kind = "RESTRUCTURE"
	KindTag              Kind = "TAG"
	KindExtension        Kind = "EXTENSION"
	KindBuyout           Kind = "BUYOUT"
	KindAdvancePhase     Kind = "ADVANCE_PHASE"
	KindRolloverSeason   Kind = "ROLLOVER_SEASON"
	KindRetirement       Kind = "RETIREMENT"
	KindDeath            Kind = "DEATH"
	KindCapRelief        Kind = "CAP_RELIEF"
	KindSign             Kind = "SIGN"
	KindSetSigningWindow Kind = "SET_SIGNING_WINDOW"
	KindScheduleEvent    Kind = "SCHEDULE_EVENT"
	KindRescheduleEvent  Kind = "RESCHEDULE_EVENT"
	KindCancelEvent      Kind = "CANCEL_EVENT"
)

// Request is a transaction the Coordinator can execute. The concrete types live in THIS
// root package, so a caller (the IPC layer) builds one directly and never imports a
// handler subpackage (the depguard boundary). The set is closed by the unexported
// sealed marker — a new transaction type is added here, deliberately, never externally.
type Request interface {
	Kind() Kind
	validate() error
	// apply runs the transaction's steps against the shared tx writer and returns how many
	// players it touched plus its pre-commit cap-impact line items (applyResult). It performs
	// no commit — WriteTx owns the transaction; a Preview reads the applyResult and rolls back.
	apply(ctx context.Context, w state.TxWriter) (applyResult, error)
	sealed()
}

// Trade, PlayerMove, and maxTradeLegs live in request_trade.go (split out to stay within the
// 400-line file cap — AD-14/AD-17 pre-splits).

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

func (r RosterStatusChange) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	if err := acquisitions.SetStatus(ctx, w, r.MFLID, r.Status); err != nil {
		return applyResult{}, fmt.Errorf("roster status: %w", err)
	}
	// A roster-status move (active ↔ taxi/IR) changes no cap figure, so it carries no cap delta.
	return applyResult{PlayersAffected: 1}, nil
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

func (wv Waiver) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	entry, err := deadcap.Waive(ctx, w, wv.MFLID)
	if err != nil {
		return applyResult{}, fmt.Errorf("waiver: %w", err)
	}
	return applyResult{PlayersAffected: 1, Deltas: deadCapDeltas(entry)}, nil
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

func (r Restructure) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	if err := contracts.Restructure(ctx, w, r.MFLID, r.Move); err != nil {
		return applyResult{}, fmt.Errorf("restructure: %w", err)
	}
	// A §11 restructure MOVES money between the player's own cells (conserved) — the current-season
	// cap drop is not yet surfaced pre-commit; it lands on the post-commit refresh.
	return applyResult{PlayersAffected: 1}, nil
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

func (t Tag) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	if err := contracts.Tag(ctx, w, t.MFLID, t.price); err != nil {
		return applyResult{}, fmt.Errorf("tag: %w", err)
	}
	// A §9 tag raises the player's cap salary to the resolved price — the cap increase is not yet
	// surfaced pre-commit; it lands on the post-commit refresh.
	return applyResult{PlayersAffected: 1}, nil
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

func (e Extension) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	if err := contracts.Extend(ctx, w, e.MFLID, e.AddedYears, e.floor); err != nil {
		return applyResult{}, fmt.Errorf("extension: %w", err)
	}
	// A §10 extension appends future PAID years priced off the highest remaining year; the current
	// season's cap is unchanged, so any breakdown is a future-year concern (post-commit refresh).
	return applyResult{PlayersAffected: 1}, nil
}

// Buyout executes a §12 contract buyout: the franchise releases the player and owes a §12
// dead-cap charge (rate by years remaining — 60/75/90% for 2/3/4 — times his average remaining
// salary) against the current season's cap. It is OFFSEASON-only (the Coordinator phase gate)
// and capped at two per franchise per season (transaction_counts, op_kind "BUYOUT"). Every money
// figure and the remaining-year count are resolved from authoritative state inside apply; the
// request carries only the player id.
type Buyout struct {
	MFLID string
}

func (Buyout) Kind() Kind { return KindBuyout }
func (Buyout) sealed()    {}

// validate rejects only an empty player id here; roster membership, the §12 rate/charge, the
// 2..4-remaining-year range, and the per-season limit are all resolved against real state in apply.
func (b Buyout) validate() error {
	if strings.TrimSpace(b.MFLID) == "" {
		return fmt.Errorf("transactions: buyout has an empty player id")
	}
	return nil
}

func (b Buyout) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	entry, err := deadcap.Buyout(ctx, w, b.MFLID)
	if err != nil {
		return applyResult{}, fmt.Errorf("buyout: %w", err)
	}
	return applyResult{PlayersAffected: 1, Deltas: deadCapDeltas(entry)}, nil
}

// Retirement executes a §13 retirement: the franchise releases the player and owes a §13
// dead-cap charge (30% of his remaining contract — the salary of every year strictly after the
// current season). It reuses the §8 release/void path; every money figure and the remaining-year
// sum are resolved from authoritative state inside apply — the request carries only the id.
type Retirement struct {
	MFLID string
}

func (Retirement) Kind() Kind { return KindRetirement }
func (Retirement) sealed()    {}

// validate rejects only an empty player id here; roster membership and the §13 charge are
// resolved against real state in apply.
func (r Retirement) validate() error {
	if strings.TrimSpace(r.MFLID) == "" {
		return fmt.Errorf("transactions: retirement has an empty player id")
	}
	return nil
}

func (r Retirement) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	entry, err := deadcap.Retire(ctx, w, r.MFLID)
	if err != nil {
		return applyResult{}, fmt.Errorf("retirement: %w", err)
	}
	return applyResult{PlayersAffected: 1, Deltas: deadCapDeltas(entry)}, nil
}

// Death executes a §13 Gaines Adams Rule removal: a player's death removes him from his roster
// with NO cap penalty (a cut with zero dead cap). It reuses the §8 release/void path and records
// a $0 dead-cap audit row. The request carries only the id.
type Death struct {
	MFLID string
}

func (Death) Kind() Kind { return KindDeath }
func (Death) sealed()    {}

// validate rejects only an empty player id here; roster membership is resolved in apply.
func (d Death) validate() error {
	if strings.TrimSpace(d.MFLID) == "" {
		return fmt.Errorf("transactions: death has an empty player id")
	}
	return nil
}

func (d Death) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	entry, err := deadcap.Death(ctx, w, d.MFLID)
	if err != nil {
		return applyResult{}, fmt.Errorf("death: %w", err)
	}
	// Gaines Adams Rule: removal at $0 dead cap. deadCapDeltas returns no line for a zero charge,
	// so the quote correctly shows a removal with no cap penalty.
	return applyResult{PlayersAffected: 1, Deltas: deadCapDeltas(entry)}, nil
}

// CapRelief executes a §13 Cap Relief Appeal: the commissioner reduces a franchise's cap hit by
// Amount (career-ending injury, recurring injury, behavioral suspension). It appends a positive
// credit to the cap-relief ledger, which CapUsed subtracts — a franchise-scoped adjustment with
// no player release. Unlike the priced ops, Amount IS a caller field: it is the commissioner's
// discretionary figure (there is no formula to resolve), carried as exact cents and validated for
// shape here and at the store.
type CapRelief struct {
	FranchiseID string
	Amount      domain.Money
	Reason      string
}

func (CapRelief) Kind() Kind { return KindCapRelief }
func (CapRelief) sealed()    {}

// validate enforces the shape a commissioner relief must have — a franchise, a positive amount,
// and a reason for the audit trail — before a transaction is opened.
func (c CapRelief) validate() error {
	if strings.TrimSpace(c.FranchiseID) == "" {
		return fmt.Errorf("transactions: cap relief has an empty franchise id")
	}
	if c.Amount <= 0 {
		return fmt.Errorf("transactions: cap relief amount must be positive")
	}
	if strings.TrimSpace(c.Reason) == "" {
		return fmt.Errorf("transactions: cap relief requires a reason (audit trail)")
	}
	return nil
}

func (c CapRelief) apply(ctx context.Context, w state.TxWriter) (applyResult, error) {
	entry, err := deadcap.Relieve(ctx, w, c.FranchiseID, c.Amount, c.Reason)
	if err != nil {
		return applyResult{}, fmt.Errorf("cap relief: %w", err)
	}
	// A §13 relief is a NEGATIVE cap delta — a credit CapUsed subtracts (Σcells + Σdead_cap −
	// Σcap_relief). Use the store's SNAPPED amount AND its own reason (entry.Amount/entry.Reason,
	// the commissioner's audit basis) so the quote matches the committed ledger row verbatim — the
	// same reason-from-the-entry rule the dead-cap deltas follow (GLM slice-3 review L2). No player
	// is released (PlayersAffected 0).
	return applyResult{PlayersAffected: 0, Deltas: []CapDelta{{FranchiseID: c.FranchiseID, Cents: -entry.Amount, Reason: entry.Reason}}}, nil
}
