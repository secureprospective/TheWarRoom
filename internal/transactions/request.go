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

// Kind names a transaction type.
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
	KindCorrect          Kind = "CORRECT"
)

// Request is a transaction the Coordinator can execute. The types live in this package so callers
// never import a handler, and the unexported sealed method closes the set.
//
// validate makes cheap shape checks before the transaction opens. Everything that depends on state
// (roster membership, money, limits) is resolved from authoritative state in apply; a request
// carries intent only.
type Request interface {
	Kind() Kind
	validate() error
	// apply runs the steps on the shared transaction and returns players moved plus cap-impact lines.
	// It never commits; WriteTx does, and Preview rolls back.
	apply(ctx context.Context, w state.TxWriter) (applyResult, error)
	sealed()
}

// RosterStatusChange moves one player between active, taxi and IR.
type RosterStatusChange struct {
	MFLID  string
	Status domain.RosterStatus
}

func (RosterStatusChange) Kind() Kind { return KindRosterStatus }
func (RosterStatusChange) sealed()    {}

// The status whitelist lives in the state layer only, so the two can't drift.
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
	// A status move changes no cap figure.
	return applyResult{PlayersAffected: 1}, nil
}

// enforceRosterLimits checks the taxi or IR slot cap when moving into one; moving out frees a slot,
// and total roster size doesn't change.
func (r RosterStatusChange) enforceRosterLimits(_ context.Context, rd state.Reader, p RosterPolicy) error {
	var limit int
	switch r.Status {
	case domain.RosterTaxi:
		limit = p.TaxiSquad()
	case domain.RosterIR:
		limit = p.InjuredReserve()
	case domain.RosterActive:
		return nil // moving out frees a slot
	default:
		return nil
	}
	if limit <= 0 {
		return nil // unlimited
	}
	cur, ok := rd.Player(r.MFLID)
	if !ok {
		// Reject here rather than trust apply to: the gate must not depend on apply's behaviour.
		return fmt.Errorf("transactions: roster status change: player %q is not rostered", r.MFLID)
	}
	if cur.RosterStatus == r.Status {
		return nil // a no-op; apply rejects it
	}
	roster, ok := rd.Roster(cur.FranchiseID)
	if !ok {
		return nil // defensive: cur implies a roster
	}
	current := countByStatus(roster, r.Status)
	if current+1 > limit {
		return &errRosterLimit{detail: fmt.Sprintf(
			"roster limit: franchise %q would hold %d %s players (cap %d) — exceeds the %s slot limit",
			cur.FranchiseID, current+1, r.Status, limit, r.Status)}
	}
	return nil
}

// Waiver cuts one player (§8). The franchise owes 35% of annual salary per remaining year (50% if
// restructured) against this season's cap. v1 models the unclaimed cut only; claims come with free
// agency.
type Waiver struct {
	MFLID string
}

func (Waiver) Kind() Kind { return KindWaiver }
func (Waiver) sealed()    {}

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

// Restructure lowers a player's cap salary this year by Move (§11), up to the tier max
// ($1M/$2M/$3M by salary), and flags the contract (a later cut then charges 50%).
type Restructure struct {
	MFLID string
	Move  domain.Money
}

func (Restructure) Kind() Kind { return KindRestructure }
func (Restructure) sealed()    {}

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
	// Money moves between the player's own years; the cap drop shows after commit.
	return applyResult{PlayersAffected: 1}, nil
}

// Tag applies a §9 franchise tag. The price is resolved by Coordinator.ExecuteTag into an
// unexported field, so callers send only the id; an unresolved (zero) price is rejected.
type Tag struct {
	MFLID string
	price domain.Money
}

func (Tag) Kind() Kind { return KindTag }
func (Tag) sealed()    {}

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
	// The cap increase shows after commit.
	return applyResult{PlayersAffected: 1}, nil
}

// Extension adds AddedYears (1-3) PAID years at 150% of the top remaining year, raised to the
// position floor (§10). The floor is resolved by Coordinator.ExecuteExtension into an unexported
// field. The §10 limits (a year remaining, at most 6 total, no prior extension, one per franchise
// per season) are enforced in the handler.
type Extension struct {
	MFLID      string
	AddedYears int
	floor      domain.Money
}

func (Extension) Kind() Kind { return KindExtension }
func (Extension) sealed()    {}

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
	// Extension years are in the future; this season's cap is unchanged.
	return applyResult{PlayersAffected: 1}, nil
}

// Buyout releases a player under §12: the franchise owes 60/75/90% (2/3/4 years remaining) of his
// average remaining salary this season. Offseason only, two per franchise per season.
type Buyout struct {
	MFLID string
}

func (Buyout) Kind() Kind { return KindBuyout }
func (Buyout) sealed()    {}

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

// Retirement releases a player under §13: 30% of his remaining contract (every year after this
// one) becomes dead cap.
type Retirement struct {
	MFLID string
}

func (Retirement) Kind() Kind { return KindRetirement }
func (Retirement) sealed()    {}

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

// Death removes a player under §13's Gaines Adams Rule, with no cap penalty (a $0 dead-cap row is
// still recorded).
type Death struct {
	MFLID string
}

func (Death) Kind() Kind { return KindDeath }
func (Death) sealed()    {}

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
	// A $0 charge yields no line in the quote.
	return applyResult{PlayersAffected: 1, Deltas: deadCapDeltas(entry)}, nil
}

// CapRelief is a §13 cap-relief appeal: the commissioner reduces a franchise's cap hit by Amount.
// Amount is a caller field because it is discretionary; no formula resolves it.
type CapRelief struct {
	FranchiseID string
	Amount      domain.Money
	Reason      string
}

func (CapRelief) Kind() Kind { return KindCapRelief }
func (CapRelief) sealed()    {}

// A relief needs a franchise, a positive amount and a reason.
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
	// A negative delta: CapUsed subtracts relief. Use the store's snapped amount and reason so the
	// quote matches the ledger row.
	return applyResult{PlayersAffected: 0, Deltas: []CapDelta{{FranchiseID: c.FranchiseID, Cents: -entry.Amount, Reason: entry.Reason}}}, nil
}
