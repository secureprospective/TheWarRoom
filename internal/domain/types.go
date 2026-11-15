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

// Freshness is shared by stored snapshots and network-backed IPC results.
type Freshness struct {
	State     string `json:"state"`
	FetchedAt string `json:"fetchedAt"`
	Note      string `json:"note"`
}

// These strings cross IPC and must match frontend/src/app/data/freshness.ts.
const (
	FreshLive  = "live"
	FreshStale = "stale"
	FreshFail  = "fail"
)

// PlayerFacts is the players-DB fact set the M1 runner reads per rostered id. It carries no
// contract fields: contract state lives in the state store and must not be copied from a
// static feed.
type PlayerFacts struct {
	Name         string
	Team         string
	Position     Position
	IsRookie     bool
	Birthdate    int64  // epoch seconds
	HasBirthdate bool   // commissioner-created players lack one
	DraftYear    int    // the §6 experience source
	HasDraftYear bool   // false for undrafted ("0")
	DraftRound   int    // NFL draft round; 0 when MFL lists no selection
	DraftPick    int    // the pick within that round; 0 when MFL lists no selection
	DraftTeam    string // the NFL team that drafted him, when MFL lists a selection
	College      string // raw; the SchoolTier source
}
