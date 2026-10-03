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

// FranchiseLabel is a franchise's display name from names, or "Franchise <id>" when the
// league has none for it. Every surface labels franchises through this one function.
func FranchiseLabel(names map[string]string, id string) string {
	if n := names[id]; n != "" {
		return n
	}
	return "Franchise " + id
}
