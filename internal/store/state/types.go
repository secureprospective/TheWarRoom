package state

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// PlayerState is one player's MUTABLE runtime state: which franchise holds him, his
// roster status, and his live contract terms. It is the join of a rosters row with
// a contracts row (Backend_Architecture §8). Unlike domain.PlayerRecord (the
// normalize seed shape, which also carries DISPLAY fields like Name/Position), this
// is only the state B7 mutates at runtime — name/position live in the players DB and
// are looked up separately, never duplicated into mutable state.
type PlayerState struct {
	MFLID        string              // canonical player id (string, leading zeros)
	FranchiseID  string              // owning franchise, "0001"–"0032"
	RosterStatus domain.RosterStatus // ROSTER or TAXI_SQUAD
	Salary       domain.Money        // annual (BASE) salary, exact cents — the §9/§11 rule base
	// CapSalary is the cap-counting salary: DERIVED from the current-season PAID ledger cell
	// (the KING), NOT stored competing truth. Equals Salary until a §11 restructure moves
	// money out of this year's cell (Ship 3 read-flip).
	CapSalary      domain.Money
	ContractYears  int                   // years remaining; 0 until B7 computes it
	ExpirationYear int                   // final contract year (seeded from normalize)
	ContractStatus domain.ContractStatus // UFA/RFA/FT1/FT2
	IsRestructured bool
	IsTagged       bool
}

// FranchiseState is one franchise's full runtime state: its players plus the derived
// cap usage. CapUsed is COMPUTED from the contracts (never stored as truth) — AD-21:
// the cap AMOUNT is config (B3b); this is the USAGE against it.
type FranchiseState struct {
	FranchiseID string
	Players     []PlayerState
	CapUsed     domain.Money
}

// ContractChange is the full set of contract terms B7 applies to a player (a tag, a
// restructure, an extension, a year rollover all land here). It replaces the
// player's live contract fields atomically.
type ContractChange struct {
	AnnualSalary   domain.Money
	ContractYears  int
	ExpirationYear int
	ContractStatus domain.ContractStatus
	IsRestructured bool
	IsTagged       bool
}

// Reader is the READ-ONLY surface handed to the engine, modules, and IPC
// handlers. It exposes no mutation — a consumer holding a Reader cannot change
// league state, even by type assertion (Store.Reader returns a wrapper that does not
// embed *Store). This is the read half of the AD-02/AD-05 single-writer law.
type Reader interface {
	FranchiseState(franchiseID string) (FranchiseState, bool)
	Roster(franchiseID string) ([]PlayerState, bool)
	CapUsed(franchiseID string) (domain.Money, bool)
	Player(mflID string) (PlayerState, bool)
	Franchises() []string
}

// Writer is the MUTATION surface. It is injected via dependency injection to
// B7a ONLY (the transaction coordinator), which is the SOLE runtime mutator of
// league state (AD-02). It embeds Reader because B7 reads current state before it
// writes. Every mutation goes through WriteTx: the Coordinator hands a callback that
// performs one OR MANY per-player ops on the supplied TxWriter, and the whole set
// commits (and reloads memory) atomically or rolls back as a unit. There is no
// standalone single-op mutator on this interface by design — a mutation IS a
// transaction (a single-step change is just a one-op transaction). B3c DEFINES this
// interface and the atomic write path; it does NOT wire B7 (that is B7a's session).
type Writer interface {
	Reader
	WriteTx(ctx context.Context, fn func(TxWriter) error) error
}

// TxWriter is the mutation surface INSIDE one spanning transaction. It exposes the
// same per-player ops as Writer, but each runs against a single shared SQLite tx and
// does NOT commit or reload memory — the enclosing WriteTx does that once, at the end.
// The transaction Coordinator (B7a) sequences a multi-step transaction (e.g. a trade =
// N MovePlayer + contract updates) through one TxWriter so the whole set is atomic:
// any step's error rolls the ENTIRE transaction back (AD-02 single-writer law, now
// single-transaction too). It carries no context field — each op takes ctx explicitly.
type TxWriter interface {
	MovePlayer(ctx context.Context, mflID, toFranchiseID string) error
	SetRosterStatus(ctx context.Context, mflID string, status domain.RosterStatus) error
	ApplyContract(ctx context.Context, mflID string, c ContractChange) error
	// ReleasePlayer removes a player from his franchise entirely (deletes the rosters +
	// contracts rows) AND records where he went via a mandatory player-status event — the
	// roster side of a §8 waiver cut, and the shared removal path for buyout (§12), retirement
	// and death (§13), and the §14 rollover UFA-expiry. His salary leaves CapUsed; any dead-cap
	// charge is recorded separately via AddDeadCap. The status param is the ENFORCED chokepoint
	// (expert-panel top risk): every removal MUST declare the player's destination status
	// (FREE_AGENT / RETIRED / DECEASED) so a removed player is never left unfindable. Fails loud
	// on an unknown player, an invalid status, or an empty reason. There is no undo — a release
	// is terminal; a FREE_AGENT re-enters a roster only via a SIGN.
	ReleasePlayer(ctx context.Context, mflID string, status domain.PlayerStatus, reason string) error

	// Player reads a player's CURRENT state (the pre-transaction snapshot) so a handler
	// can compute new terms — e.g. the §8 dead-cap charge reads salary/expiration/
	// restructured before releasing him. It reflects committed state, NOT this tx's own
	// uncommitted writes; a single read-then-write op (waiver, tag) is consistent.
	Player(mflID string) (PlayerState, bool)

	// CapLedgerWriter carries the two cap-ledger append primitives (dead cap + cap relief). They
	// are siblings — a debit and a credit against an absolute league year, each in its own
	// append-only ledger — so they group under one embedded member to keep TxWriter within the
	// interfacebloat cap, the same grouping device as LedgerWriter / SeasonScope.
	CapLedgerWriter
	// LedgerWriter carries the per-year cell mutations (the §9/§11 primitives). It is
	// embedded rather than inlined so TxWriter stays within the interfacebloat cap as more
	// cell ops land — an embedded interface counts as one member.
	LedgerWriter
	// SeasonScope carries the season-scoped bookkeeping surface (the league year, the
	// per-season op counters, and the current phase). Embedded so TxWriter stays within the
	// interfacebloat cap — the same grouping device as LedgerWriter.
	SeasonScope
	// StatusWriter carries the free-agency pool's availability surface (record a status event,
	// read the current status). Embedded so TxWriter stays within the interfacebloat cap — the
	// same grouping device as LedgerWriter / SeasonScope.
	StatusWriter
	// CalendarWriter carries the commissioner-calendar append surface (schedule/reschedule/cancel/
	// fire, all as append-only rows). Embedded so TxWriter stays within the interfacebloat cap —
	// the same grouping device as LedgerWriter / SeasonScope / StatusWriter.
	CalendarWriter
}

// CapLedgerWriter is the cap-ledger append surface — the two sibling primitives that move a
// franchise's season cap hit: AddDeadCap (a debit into the append-only dead_cap_ledger) and
// AddCapRelief (a credit into the sibling cap_relief_ledger). They are kept in SEPARATE ledgers so
// the dead-cap non-negativity invariant (CHECK ≥ 0) stays pure while a relief remains a positive
// credit; CapUsed sums the debits and subtracts the credits. Both fail loud on a non-positive
// amount or a missing field. Grouped under one embedded member to keep TxWriter within the
// interfacebloat cap — the same grouping device as LedgerWriter / SeasonScope. LogTradeNote rides
// along here too (not cap money, but the same "append-only audit row in the shared tx" shape, and
// TxWriter itself is already at the interfacebloat cap with no room for a sixth embedded member).
type CapLedgerWriter interface {
	// AddDeadCap appends one dead-cap charge (§8/§12/§13) against an absolute league year in the
	// shared tx. The ledger is append-only and non-negative; a duplicate is rejected. Fails loud
	// on a non-positive amount or a missing field.
	AddDeadCap(ctx context.Context, e DeadCapEntry) error
	// AddCapRelief appends one commissioner cap-relief CREDIT (§13 Cap Relief Appeal) to the
	// cap_relief_ledger in the shared tx — a franchise-scoped reduction of the season's cap hit.
	// It is the sibling of AddDeadCap: a positive credit against an absolute league year, kept in
	// its own append-only ledger so the dead-cap non-negativity invariant stays pure. CapUsed
	// subtracts it. Fails loud on a non-positive amount or a missing field.
	AddCapRelief(ctx context.Context, e CapReliefEntry) error
	// LogTradeNote appends the trade_notes audit row for one executed Trade — the event itself
	// (picksNote + rationale), which MovePlayer alone never records. Alpha scope: picksNote is
	// free-text and unvalidated (no pick-ownership ledger yet); rationale arrives already
	// non-empty (Trade.validate() enforces it before the tx opens). involvedFranchises is every
	// franchise a leg of the trade touched.
	LogTradeNote(ctx context.Context, picksNote, rationale string, involvedFranchises []string) error
	// AppendCorrection appends one Session-2 transaction-correction row (a clerical CORRECTED note
	// or a REVERSED marker) in the shared tx — the same "no room for a sixth embedded member" reason
	// LogTradeNote rides here instead of its own CorrectionWriter grouping. A correction is NEVER an
	// update/delete of the original ledger row (append-only honored); it is a new row tying back to
	// the original by tx_id. Fails loud on a missing field or an unknown status.
	AppendCorrection(ctx context.Context, e CorrectionEntry) error
}

// CalendarWriter is the commissioner-calendar's append surface — the single write primitive behind
// the append-only calendar_events log. Scheduling, rescheduling (a drag to a new time), cancelling,
// and firing a blob all APPEND a row sharing the logical event_id; the store never updates or
// deletes (the DB triggers reject it too). The store owns the mechanics (append-only, immutable,
// latest-row-per-event_id read); which status a given op writes is the handler's rule. It is
// embedded in TxWriter so the calendar op groups under one member of that interface.
type CalendarWriter interface {
	// AppendCalendarEvent appends one calendar row in the shared tx. The payload (the eventual op's
	// fields) is stored VERBATIM and NOT executed here — the calendar records intent only. Fails
	// loud on a missing field, an unparseable RFC3339 scheduled_at, or an unknown status.
	AppendCalendarEvent(ctx context.Context, e CalendarEvent) error
}

// StatusWriter is the free-agency pool's availability surface — the player-status primitives.
// A removed player's destination (FREE_AGENT / RETIRED / DECEASED) is recorded as an append-only
// event; the current status is the latest event. It is embedded in TxWriter so the pool ops group
// under one member of that interface (the interfacebloat cap). The store owns the mechanics
// (append-only, immutable, latest-row read); the rule of which status a given release maps to is
// the handler's (a cut/expiry → FREE_AGENT, retirement → RETIRED, death → DECEASED).
type StatusWriter interface {
	// RecordStatus appends one availability event for a player in the shared tx. Called by
	// ReleasePlayer (the enforced chokepoint) so no removal path can forget it. Fails loud on an
	// unknown status or an empty reason.
	RecordStatus(ctx context.Context, mflID string, status domain.PlayerStatus, reason string) error
	// CurrentStatus returns a player's latest availability status and whether any event exists.
	// It reflects COMMITTED state (a prior release), which the single-writer law makes race-free
	// — the SIGN eligibility gate reads the status as it stood before the op. found=false means
	// the player was never released (rostered or unknown). Fails loud on a stored non-status.
	CurrentStatus(ctx context.Context, mflID string) (domain.PlayerStatus, bool, error)
	// SignContract rosters a free agent on a NEW flat contract (the SIGN write primitive): it
	// clears the player's prior cells (logged), inserts the roster+contract parent rows, and lays
	// `years` contiguous PAID cells at the current season onward plus one trailing UFA slot, each
	// tagged `source`. The rule math (eligibility, min-salary floor, 1..4 years, lockout) is the
	// handler's, already resolved into the args. Fails loud on years < 1, a negative salary, or a
	// player already on a roster.
	SignContract(ctx context.Context, mflID, franchiseID string, salary domain.Money, years int, source, reason string) error
	// ActiveBuyoutLockout reports whether a buyout lockout is still active for the player — a
	// dead_cap_ledger row with `reason` whose league_year >= season (§12 "cannot bid on a
	// bought-out player until the following offseason"). Derived, zero new writes. The reason is
	// passed in so the store stays agnostic to the deadcap package's audit string.
	ActiveBuyoutLockout(ctx context.Context, mflID, reason string, season int) (bool, error)
}

// SeasonScope is the season-scoped bookkeeping surface: the absolute league year, the durable
// per-season op-limit counters, and the current season phase. Grouped into one embedded
// interface so TxWriter stays within the interfacebloat cap as season-scoped reads/writes
// accrue. All three sit at the same conceptual level — state keyed to (league, season) that
// gates and prices the transactions of that season.
type SeasonScope interface {
	// Season is the absolute league year this store operates on — the handler derives
	// "remaining years" (expiration_year − season) for the §8/§12 charge from it. Per the
	// locked invariant (domain.Phase), during OFFSEASON this is the UPCOMING managed season.
	Season() int

	// OpCount reads how many times a franchise has run a given op_kind THIS season — the
	// durable per-season limit counter (§11 "one restructure per team per year", and the
	// §9/§10/§12 op limits). It reflects COMMITTED state, not this tx's own uncommitted
	// IncOpCount; a single read-check-then-bump op is consistent because the single-writer
	// law serializes transactions. Zero for an unseen (franchise, op) key. FOOTGUN: because it
	// reads committed state, a handler that bumps the SAME counter twice in one tx would not
	// see its own first bump on a second OpCount — safe for the current one-check-one-bump ops,
	// but a multi-bump op must track its increments in-handler, not re-read via OpCount.
	OpCount(ctx context.Context, franchiseID, opKind string) (int, error)
	// IncOpCount increments that per-season counter by one inside the shared tx (an upsert
	// on (league, franchise, season, op_kind)). A handler that enforces a per-season limit
	// pairs it with OpCount: read, check the limit, mutate, bump — all atomic in WriteTx.
	IncOpCount(ctx context.Context, franchiseID, opKind string) error
	// CurrentPhase returns the league-year's current season phase (the latest transition's
	// to_phase). It reflects COMMITTED state — the op-eligibility gate checks the phase as it
	// stood before the op, which the single-writer law makes race-free. Fails loud on a missing
	// seed row (no fallback — the seed is the one source of truth) or a stored non-phase (drift).
	CurrentPhase(ctx context.Context) (domain.Phase, error)
	// AppendPhaseTransition advances the phase by appending one transition row (current → to)
	// in the shared tx — the ADVANCE_PHASE write primitive. It rejects a no-op (to == current);
	// any real target is allowed (v1 commissioner correction/rollback). `note` is a freeform
	// reason. Fails loud on an unknown target phase or a missing seed.
	AppendPhaseTransition(ctx context.Context, to domain.Phase, note string) error
	// RolloverSeason advances the league from PLAYOFFS(N) to OFFSEASON(N+1) — the ROLLOVER_SEASON
	// write primitive (Season_Rollover_Design D5/D7). In the shared tx it appends the
	// PLAYOFFS→OFFSEASON transition carrying season N+1 (so the derived current season becomes
	// N+1) and advances the roster/contract snapshot to N+1. The season is MONOTONIC — this is the
	// ONLY primitive that moves it, and only forward by one. Requires the current phase to be
	// PLAYOFFS (the op gate enforces it; asserted here for defense in depth). Dead-cap/relief/count
	// rows are untouched (per-year audit; the new season reads them as 0 by absence) and
	// is_restructured is left intact (§11 lifetime guard, D4). `note` is a freeform reason.
	RolloverSeason(ctx context.Context, note string) error
	// SigningWindowClosed reports whether the §6 free-agency signing window has been CLOSED by an
	// explicit commissioner directive — the COMMISSIONER UFA-CALENDAR read (Free_Agency_Design Q4).
	// Absent any directive it returns false (OPEN by default — the v1 posture where SIGN's only
	// phase restriction is the playoffs block). The directive is the latest season_phases row that
	// carries a ufa_window meta value, so it PERSISTS across phase transitions and rollovers (which
	// write meta='') — "closes when the Super Bowl goes live, stays closed until the commissioner
	// reopens it". Reads committed state; the SIGN phase gate consults it inside the tx.
	SigningWindowClosed(ctx context.Context) (bool, error)
	// AppendSigningWindow records a commissioner UFA-window toggle by APPENDING a season_phases row
	// that keeps the current phase (from == to) and carries the ufa_window directive in meta — the
	// COMMISSIONER UFA-CALENDAR write primitive (Free_Agency_Design Q4). It is NOT a phase transition;
	// it rides the same append-only log because meta is the slot built for sub-phase granularity.
	// Rejects a redundant toggle (the window is already in the requested state — the no-silent-no-op
	// house rule). `note` is a freeform reason.
	AppendSigningWindow(ctx context.Context, open bool, note string) error
	// TradeDeadlinePassed reports whether the §14 Week-9 commissioner trade deadline has passed —
	// the COMMISSIONER TRADE-DEADLINE read (mirrors SigningWindowClosed). Absent any directive, or
	// a directive that CLEARED the deadline, it returns false (no block — the v1 default). Reads
	// committed state; the TRADE phase gate consults it inside the tx.
	TradeDeadlinePassed(ctx context.Context) (bool, error)
	// AppendTradeDeadline records a commissioner trade-deadline directive by APPENDING a
	// season_phases row that keeps the current phase (from == to) and carries the trade_deadline
	// directive in meta — mirrors AppendSigningWindow. A zero Deadline CLEARS any standing
	// deadline (the commissioner's "reopen trades" action). `note` is a freeform reason.
	AppendTradeDeadline(ctx context.Context, deadline time.Time, note string) error
}

// LedgerWriter is the per-year salary-cell mutation surface — the money primitives the
// ledger cutover ops write through. It is embedded in TxWriter so the cell ops group under
// one member of that interface (the interfacebloat cap). The store owns the mechanics
// (read-through-tx, immutable change log, non-negativity); each handler owns the rule math.
type LedgerWriter interface {
	// MoveCellMoney moves `amount` from one contract-year cell to another for a player,
	// conserving the contract total (the §11 restructure primitive — "just moving money
	// from year to year"). It subtracts from fromYear, adds to toYear, and logs BOTH cell
	// deltas to the immutable change log with the given reason, all in the shared tx. The
	// store conserves by construction (one move = equal and opposite deltas); the rule math
	// (tier max, which years) is the handler's. Fails loud if a cell is missing or would go
	// negative.
	MoveCellMoney(ctx context.Context, mflID string, fromYear, toYear int, amount domain.Money, reason string) error
	// SetCell sets one PAID cell to an ABSOLUTE value and logs the old→new change, in the
	// shared tx. Unlike MoveCellMoney it does NOT conserve the contract total — it is the §9
	// franchise-tag primitive (replace the season salary with the resolved tag price). Fails
	// loud if the cell is missing or the value is negative.
	SetCell(ctx context.Context, mflID string, year int, value domain.Money, reason string) error
	// VoidCells marks ALL of a player's PAID cells VOID ($0 cap, kept for history) and logs
	// each old→0 change, in the shared tx — the §8 waiver-cut primitive: a cut relieves every
	// remaining cap-bearing cell while preserving the contract's history (cells are flipped to
	// VOID, never deleted). Fails loud if the player has no PAID cell to void.
	VoidCells(ctx context.Context, mflID string, reason string) error
	// PaidCells returns a player's PAID contract-year cells (year, salary, source), ordered by
	// year, read THROUGH the shared tx. It is a READ grouped onto this cell interface (rather
	// than TxWriter, which sits at the interfacebloat cap) because it serves the cell ops: the
	// §10 extension handler derives its rule facts from it — the highest-paid remaining year
	// (§10 pricing), the total contract length (the ≤6-year cap), and whether the contract
	// already carries an extension cell (source "extension" — the "no second extension off a
	// prior extension" guard, answered from the cells per the ledger_schema design intent, no
	// is_extended flag). UFA/VOID cells are omitted (they carry no cap).
	PaidCells(ctx context.Context, mflID string) ([]LedgerCell, error)
	// AppendExtensionYears appends `addedYears` new PAID contract-year cells at pricePerYear —
	// the §10 extension write primitive. It PROMOTES the UFA placeholder slot (the offseason
	// after the last paid year) to the first new PAID year, inserts any further added years as
	// new PAID cells, and creates a fresh UFA slot the offseason after the new last paid year,
	// so the ledger invariant (contiguous PAID cells followed by exactly one UFA slot) still
	// holds. Every new/promoted cell is tagged source "extension" (the prior-extension marker)
	// and logged to the immutable change log, in the shared tx. The store owns the mechanics;
	// the §10 rule math (price, added-year bounds, ≤6 total, position floors) is the handler's,
	// already resolved into the arguments. Fails loud on a non-positive addedYears/price, a
	// player with no PAID cell, or a missing/misplaced UFA slot (drift).
	AppendExtensionYears(ctx context.Context, mflID string, addedYears int, pricePerYear domain.Money, reason string) error
}

// LedgerCell is one contract-year cell exposed to a handler for reading — the league year,
// its cap-bearing salary (exact cents), and the source tag recording what wrote it ("seed",
// "op", or "extension"). The §10 handler derives its rule facts from a player's cells (the
// highest-paid remaining year, the total contract length, whether a prior extension exists)
// without the store running any rule logic — the store returns data, the handler judges it.
type LedgerCell struct {
	Year   int
	Salary domain.Money
	Source string
}

// DeadCapEntry is one append-only dead-cap charge against a franchise's cap for an
// ABSOLUTE league year — the §8 waiver-cut penalty (or §11/§12/§13 charges in B7c).
// The store records it verbatim; the §8 formula (35%/50% × salary × remaining years,
// 0 if claimed) is COMPUTED by the B7c/deadcap handler, never here — this store runs no
// rule logic (the B3c divergence). DeadCap is exact cents (OQ-014); it is keyed by an
// absolute LeagueYear, never a relative slot, so a charge lands in the right cap year
// regardless of when it is queried. Reason is a required audit string.
type DeadCapEntry struct {
	FranchiseID string
	MFLID       string       // the released player the charge is attributed to
	LeagueYear  int          // ABSOLUTE league year the charge counts against
	DeadCap     domain.Money // exact cents, >= 0
	Reason      string       // audit trail, e.g. "waiver-cut §8"
}

// CapReliefEntry is one append-only commissioner cap-relief credit (§13 Cap Relief Appeal)
// against a franchise's cap for an ABSOLUTE league year. Unlike DeadCapEntry it is
// FRANCHISE-scoped (no player id): a relief appeal reduces the franchise's cap hit, not a single
// player's dead-cap row. Amount is exact cents and strictly positive (a $0 relief is a no-op).
// Reason is a required audit string recording the commissioner's basis (career-ending injury,
// recurring injury, behavioral suspension). CapUsed subtracts the sum of these credits.
type CapReliefEntry struct {
	FranchiseID string
	LeagueYear  int          // ABSOLUTE league year the credit counts against
	Amount      domain.Money // exact cents, > 0
	Reason      string       // audit trail, e.g. "cap-relief §13: career-ending injury"
}

// Source is the SEED source B3c pulls from at Initialize — the normalized rosters
// from Layer 1 (internal/normalize). B3 NEVER pushes; B3c pulls (the injected-source
// seam, cloned from B3b's league.Source). A fake implements it in tests.
type Source interface {
	Rosters(ctx context.Context) ([]domain.Roster, error)
}
