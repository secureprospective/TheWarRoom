package state

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// PlayerState is one player's mutable runtime state: franchise, roster status and live contract
// (a rosters row joined to a contracts row). Name and position live in the players directory.
type PlayerState struct {
	MFLID        string
	FranchiseID  string // "0001"–"0032"
	RosterStatus domain.RosterStatus
	Salary       domain.Money // base annual salary
	// CapSalary is derived from the current season's PAID ledger cell, the source of truth. It equals
	// Salary until a restructure moves money out of this year.
	CapSalary      domain.Money
	ContractYears  int // never populated (0); the ledger cells hold the real contract length
	ExpirationYear int
	ContractStatus domain.ContractStatus
	IsRestructured bool
	IsTagged       bool
}

// FranchiseState is one franchise's players plus cap usage, computed from contracts, never
// stored. The cap amount itself is rulebook config.
type FranchiseState struct {
	FranchiseID string
	Players     []PlayerState
	CapUsed     domain.Money
}

// ContractChange is the full set of contract terms a transaction applies, replaced atomically.
type ContractChange struct {
	AnnualSalary   domain.Money
	ContractYears  int
	ExpirationYear int
	ContractStatus domain.ContractStatus
	IsRestructured bool
	IsTagged       bool
}

// Reader is the read-only surface for the engine, modules and IPC. Store.Reader returns a
// wrapper that does not embed *Store, so not even a type assertion can mutate state.
type Reader interface {
	FranchiseState(franchiseID string) (FranchiseState, bool)
	Roster(franchiseID string) ([]PlayerState, bool)
	CapUsed(franchiseID string) (domain.Money, bool)
	Player(mflID string) (PlayerState, bool)
	Franchises() []string
}

// Writer is the mutation surface, given only to the transaction coordinator. Every mutation goes
// through WriteTx: one or many ops on a TxWriter commit together, reload memory, or roll back as a
// unit. There is no single-op mutator; a one-step change is a one-op transaction.
type Writer interface {
	Reader
	WriteTx(ctx context.Context, fn func(TxWriter) error) error
}

// TxWriter runs ops against one shared SQLite transaction without committing; the enclosing
// WriteTx commits once at the end, so any failing step rolls back the whole transaction.
type TxWriter interface {
	RosterWriter
	CapLedgerWriter
	AuditWriter
	LedgerWriter
	SeasonScope
	PhaseDirectives
	StatusWriter
	CalendarWriter
}

// RosterWriter moves players between franchises, roster slots and contracts.
type RosterWriter interface {
	MovePlayer(ctx context.Context, mflID, toFranchiseID string) error
	SetRosterStatus(ctx context.Context, mflID string, status domain.RosterStatus) error
	ApplyContract(ctx context.Context, mflID string, c ContractChange) error
	// ReleasePlayer removes a player from a franchise and records where the player went. The
	// status is mandatory (FREE_AGENT, RETIRED or DECEASED) so a removed player is never
	// unfindable. Dead cap is recorded separately via AddDeadCap. Terminal: a free agent returns
	// only through SIGN.
	ReleasePlayer(ctx context.Context, mflID string, status domain.PlayerStatus, reason string) error
	// Player returns the committed (pre-transaction) state, not this transaction's own writes.
	Player(mflID string) (PlayerState, bool)
}

// CapLedgerWriter appends the two cap-ledger rows: dead cap (a debit) and cap relief (a credit).
// They are separate ledgers so dead cap stays non-negative; CapUsed sums the debits and subtracts
// the credits.
type CapLedgerWriter interface {
	// AddDeadCap appends one dead-cap charge against an absolute league year. Non-positive amounts
	// and duplicates are rejected.
	AddDeadCap(ctx context.Context, e DeadCapEntry) error
	// AddCapRelief appends one commissioner cap-relief credit (§13) against an absolute league year.
	AddCapRelief(ctx context.Context, e CapReliefEntry) error
}

// AuditWriter appends the audit trail: trade notes and corrections. Both are append-only.
type AuditWriter interface {
	// LogTradeNote appends the audit row for an executed trade. picksNote is unvalidated free text
	// (no pick-ownership ledger yet); rationale is already validated non-empty.
	LogTradeNote(ctx context.Context, picksNote, rationale string, involvedFranchises []string) error
	// AppendCorrection appends a correction (CORRECTED note or REVERSED marker) tied to the original
	// by tx_id. The original row is never updated or deleted.
	AppendCorrection(ctx context.Context, e CorrectionEntry) error
}

// CalendarWriter appends to the commissioner calendar. Scheduling, rescheduling, cancelling and
// firing all append a row with the same event_id; the database rejects updates and deletes.
type CalendarWriter interface {
	// AppendCalendarEvent appends one calendar row. The payload is stored, not executed: the calendar
	// records intent only.
	AppendCalendarEvent(ctx context.Context, e CalendarEvent) error
}

// StatusWriter records player availability. A removed player's destination is an append-only
// event; the current status is the latest one. Which status a release maps to is the handler's
// rule.
type StatusWriter interface {
	// RecordStatus appends one availability event. ReleasePlayer calls it, so no removal path can
	// skip it.
	RecordStatus(ctx context.Context, mflID string, status domain.PlayerStatus, reason string) error
	// CurrentStatus returns the latest committed status; found=false means never released.
	CurrentStatus(ctx context.Context, mflID string) (domain.PlayerStatus, bool, error)
	// SignContract rosters a free agent on a new flat contract: it clears his prior cells (logged),
	// adds roster and contract rows, and lays `years` PAID cells from the current season plus one UFA
	// slot. The rule math (eligibility, floor, years, lockout) is the handler's.
	SignContract(ctx context.Context, mflID, franchiseID string, salary domain.Money, years int, source, reason string) error
	// ActiveBuyoutLockout reports whether a buyout lockout still applies: a dead_cap_ledger row with
	// `reason` for this season or later (§12). The caller passes the reason string.
	ActiveBuyoutLockout(ctx context.Context, mflID, reason string, season int) (bool, error)
}

// SeasonScope is the season-scoped state that gates and prices transactions: the league year,
// the per-season op counters and the current phase.
type SeasonScope interface {
	// Season is the absolute league year. During OFFSEASON it is the upcoming season.
	Season() int

	// OpCount is how many times a franchise has run an op this season (for the per-season limits).
	// It reads committed state, so an op that bumps the same counter twice in one transaction must
	// count its own increments.
	OpCount(ctx context.Context, franchiseID, opKind string) (int, error)
	// IncOpCount bumps that counter inside the transaction: read, check, mutate, bump.
	IncOpCount(ctx context.Context, franchiseID, opKind string) error
	// CurrentPhase returns the committed season phase. A missing seed row or an unknown stored phase
	// is an error.
	CurrentPhase(ctx context.Context) (domain.Phase, error)
	// AppendPhaseTransition moves to a new phase. Any real target is allowed (commissioner correction);
	// a no-op is rejected.
	AppendPhaseTransition(ctx context.Context, to domain.Phase, note string) error
	// RolloverSeason moves the league from PLAYOFFS(N) to OFFSEASON(N+1) and advances the roster and
	// contract snapshot. It is the only primitive that moves the season, and only forward by one.
	// Ledgers and counters are untouched; is_restructured persists (§11 lifetime guard).
	RolloverSeason(ctx context.Context, note string) error
}

// PhaseDirectives are commissioner settings that keep the phase: the §6 signing window and the
// §14 trade deadline.
type PhaseDirectives interface {
	// SigningWindowClosed reports whether the commissioner has closed the §6 signing window. Open by
	// default. The directive persists across phase changes until the commissioner reopens it.
	SigningWindowClosed(ctx context.Context) (bool, error)
	// AppendSigningWindow records a signing-window toggle as a season_phases row that keeps the
	// phase. A redundant toggle is rejected.
	AppendSigningWindow(ctx context.Context, open bool, note string) error
	// TradeDeadlinePassed reports whether the commissioner's §14 trade deadline has passed. No
	// directive, or a cleared one, means no block.
	TradeDeadlinePassed(ctx context.Context) (bool, error)
	// AppendTradeDeadline records a trade-deadline directive; a zero deadline clears it.
	AppendTradeDeadline(ctx context.Context, deadline time.Time, note string) error
}

// LedgerWriter is the per-year salary-cell surface. The store owns the mechanics (change log,
// non-negativity); each handler owns the rule math.
type LedgerWriter interface {
	// MoveCellMoney moves amount from one contract year to another, conserving the total (§11
	// restructure). Both deltas are logged. Fails if a cell is missing or would go negative.
	MoveCellMoney(ctx context.Context, mflID string, fromYear, toYear int, amount domain.Money, reason string) error
	// SetCell sets one PAID cell to an absolute value and logs the change (§9 franchise tag). Unlike
	// MoveCellMoney it does not conserve the total.
	SetCell(ctx context.Context, mflID string, year int, value domain.Money, reason string) error
	// VoidCells marks every PAID cell VOID ($0 cap, kept for history) and logs each change (§8
	// waiver cut).
	VoidCells(ctx context.Context, mflID string, reason string) error
	// PaidCells returns a player's PAID cells by year, read through the transaction. The §10
	// extension handler derives its facts from them: the top-paid year, the contract length, and
	// whether a prior extension exists (source "extension"). UFA and VOID cells are omitted.
	PaidCells(ctx context.Context, mflID string) ([]LedgerCell, error)
	// AppendExtensionYears adds `addedYears` PAID cells at pricePerYear (§10): the UFA slot becomes
	// the first new year, further years are inserted, and a new UFA slot follows the last paid year.
	// New cells are tagged "extension" and logged. The rule math is already resolved by the handler.
	AppendExtensionYears(ctx context.Context, mflID string, addedYears int, pricePerYear domain.Money, reason string) error
}

// LedgerCell is one contract-year cell for a handler to read: year, salary and the source that
// wrote it ("seed", "op" or "extension").
type LedgerCell struct {
	Year   int
	Salary domain.Money
	Source string
}

// DeadCapEntry is one append-only dead-cap charge against an absolute league year. The formula is
// the handler's; the store records the result.
type DeadCapEntry struct {
	FranchiseID string
	MFLID       string
	LeagueYear  int
	DeadCap     domain.Money // >= 0
	Reason      string
}

// CapReliefEntry is one commissioner cap-relief credit (§13), franchise-scoped and strictly
// positive.
type CapReliefEntry struct {
	FranchiseID string
	LeagueYear  int
	Amount      domain.Money // > 0
	Reason      string
}

// Source supplies the normalized rosters that seed a fresh store.
type Source interface {
	Rosters(ctx context.Context) ([]domain.Roster, error)
}
