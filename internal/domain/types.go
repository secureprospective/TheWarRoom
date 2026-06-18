package domain

import "github.com/secureprospective/TheWarRoom/internal/playerid"

// PlayerRecord is one fully-typed roster entry: the cross-reference join of a
// rosters row (salary/contract/roster-status, per franchise) with the players
// database (name/position/team/rookie). Every field is typed and validated — this
// is what the engine and stores consume. Raw strings stop at the normalize
// boundary; nothing downstream re-parses MFL text.
type PlayerRecord struct {
	MFLID          playerid.PlayerID // canonical, validated id (RISK-003 site #2)
	Name           string            // "Last, First", from the players DB
	Position       Position          // normalized engine position (PosFlag = needs review)
	NFLTeam        string            // 3-letter NFL code, or "FA"
	Salary         float64           // millions, parsed from the raw string
	ContractYear   int               // final contract year (0 if absent)
	ContractStatus ContractStatus    // normalized (CStatusFlag = needs review)
	ContractInfo   string            // free-text MFL note — DISPLAY ONLY
	RosterStatus   RosterStatus      // ROSTER or TAXI_SQUAD
	IsRookie       bool              // players DB status == "R"
	FranchiseID    string            // owning franchise, "0001"–"0032"
}

// NeedsAdminReview reports whether normalize could not fully classify this record
// (an unknown position or contract status). Such records are surfaced for manual
// resolution rather than silently dropped or guessed.
func (r PlayerRecord) NeedsAdminReview() bool {
	return r.Position == PosFlag || r.ContractStatus == CStatusFlag
}

// Roster is one franchise's full set of normalized player records.
type Roster struct {
	FranchiseID string
	Players     []PlayerRecord
}
