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
	MFLID          string                // canonical player id (string, leading zeros)
	FranchiseID    string                // owning franchise, "0001"–"0032"
	RosterStatus   domain.RosterStatus   // ROSTER or TAXI_SQUAD
	Salary         float64               // annual_salary (millions)
	AdjustedSalary float64               // after adjustment items; 0 until B7 sets it
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
	CapUsed     float64
}

// ContractChange is the full set of contract terms B7 applies to a player (a tag, a
// restructure, an extension, a year rollover all land here). It replaces the
// player's live contract fields atomically.
type ContractChange struct {
	AnnualSalary   float64
	AdjustedSalary float64
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
	CapUsed(franchiseID string) (float64, bool)
	Player(mflID string) (PlayerState, bool)
	Franchises() []string
}

// Writer is the MUTATION surface. It is injected via dependency injection to
// B7a ONLY (the transaction coordinator), which is the SOLE runtime mutator of
// league state (AD-02). It embeds Reader because B7 reads current state before
// it writes. B3c DEFINES this interface and the atomic write path; it does NOT wire
// B7 (that is B7a's session).
type Writer interface {
	Reader
	MovePlayer(ctx context.Context, mflID, toFranchiseID string) error
	SetRosterStatus(ctx context.Context, mflID string, status domain.RosterStatus) error
	ApplyContract(ctx context.Context, mflID string, c ContractChange) error
}

// Source is the SEED source B3c pulls from at Initialize — the normalized rosters
// from Layer 1 (internal/normalize). B3 NEVER pushes; B3c pulls (the injected-source
// seam, cloned from B3b's league.Source). A fake implements it in tests.
type Source interface {
	Rosters(ctx context.Context) ([]domain.Roster, error)
}
