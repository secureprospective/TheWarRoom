package state

import (
	"context"

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
	AddDeadCap(ctx context.Context, e DeadCapEntry) error
	// ReleasePlayer removes a player from his franchise entirely (deletes the rosters +
	// contracts rows) — the roster side of a §8 waiver cut. His salary leaves CapUsed;
	// the dead-cap charge is recorded separately via AddDeadCap. Fails loud on an unknown
	// player. There is no undo — a release is terminal (the player re-enters via free
	// agency, a later build).
	ReleasePlayer(ctx context.Context, mflID string) error

	// Player reads a player's CURRENT state (the pre-transaction snapshot) so a handler
	// can compute new terms — e.g. the §8 dead-cap charge reads salary/expiration/
	// restructured before releasing him. It reflects committed state, NOT this tx's own
	// uncommitted writes; a single read-then-write op (waiver, tag) is consistent.
	Player(mflID string) (PlayerState, bool)

	// LedgerWriter carries the per-year cell mutations (the §9/§11 primitives). It is
	// embedded rather than inlined so TxWriter stays within the interfacebloat cap as more
	// cell ops land — an embedded interface counts as one member.
	LedgerWriter
	// Season is the absolute league year this store operates on — the handler derives
	// "remaining years" (expiration_year − season) for the §8 charge from it.
	Season() int

	// OpCount reads how many times a franchise has run a given op_kind THIS season — the
	// durable per-season limit counter (§11 "one restructure per team per year", and the
	// §9/§10/§12 op limits to come). It reflects COMMITTED state, not this tx's own
	// uncommitted IncOpCount; a single read-check-then-bump op is consistent because the
	// single-writer law serializes transactions. Zero for an unseen (franchise, op) key.
	// FOOTGUN: because it reads committed state, a handler that bumps the SAME counter twice
	// in one tx would not see its own first bump on a second OpCount — safe for the current
	// one-check-one-bump ops (restructure), but a multi-bump op must track its increments
	// in-handler, not re-read via OpCount.
	OpCount(ctx context.Context, franchiseID, opKind string) (int, error)
	// IncOpCount increments that per-season counter by one inside the shared tx (an upsert
	// on (league, franchise, season, op_kind)). A handler that enforces a per-season limit
	// pairs it with OpCount: read, check the limit, mutate, bump — all atomic in WriteTx.
	IncOpCount(ctx context.Context, franchiseID, opKind string) error
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

// Source is the SEED source B3c pulls from at Initialize — the normalized rosters
// from Layer 1 (internal/normalize). B3 NEVER pushes; B3c pulls (the injected-source
// seam, cloned from B3b's league.Source). A fake implements it in tests.
type Source interface {
	Rosters(ctx context.Context) ([]domain.Roster, error)
}
