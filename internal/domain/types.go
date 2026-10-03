package domain

import "github.com/secureprospective/TheWarRoom/internal/playerid"

// PlayerRecord is one typed roster entry: a rosters row joined with the players database.
// Raw strings stop at normalize; nothing downstream re-parses MFL text.
type PlayerRecord struct {
	MFLID          playerid.PlayerID
	Name           string // "Last, First", from the players DB
	Position       Position
	NFLTeam        string // 3-letter NFL code, or "FA"
	Salary         Money
	ContractYear   int // final contract year (0 if absent)
	ContractStatus ContractStatus
	ContractInfo   string // free-text MFL note, display only
	RosterStatus   RosterStatus
	IsRookie       bool   // players DB status == "R"
	FranchiseID    string // owning franchise, "0001"–"0032"
}

// NeedsAdminReview reports an unknown position or contract status, surfaced for manual
// resolution rather than dropped or guessed.
func (r PlayerRecord) NeedsAdminReview() bool {
	return r.Position == PosFlag || r.ContractStatus == CStatusFlag
}

// Roster is one franchise's full set of normalized player records.
type Roster struct {
	FranchiseID string
	Players     []PlayerRecord
}
