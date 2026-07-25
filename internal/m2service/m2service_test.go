package m2service

import (
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/output"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// fakeReader is a minimal state.Reader: only Player feeds the franchise-aggregation
// join, so the rest are stubbed to satisfy the interface (mirrors
// internal/transactions/pricing_test.go's fakeReader).
type fakeReader struct {
	players map[string]state.PlayerState // mflID -> state
}

func (f fakeReader) Franchises() []string                      { return nil }
func (f fakeReader) Roster(string) ([]state.PlayerState, bool) { return nil, false }
func (f fakeReader) FranchiseState(string) (state.FranchiseState, bool) {
	return state.FranchiseState{}, false
}
func (f fakeReader) CapUsed(string) (domain.Money, bool) { return 0, false }
func (f fakeReader) Player(mflID string) (state.PlayerState, bool) {
	p, ok := f.players[mflID]
	return p, ok
}

// fakeRulebook is a minimal FranchiseSource.
type fakeRulebook struct {
	names        map[string]string
	starterCount string
}

func (f fakeRulebook) FranchiseNames() map[string]string { return f.names }
func (f fakeRulebook) ActiveConfig() league.RawConfig {
	return league.RawConfig{Starters: league.Starters{Count: f.starterCount}}
}

func TestNew_NilDependency(t *testing.T) {
	if _, err := New(nil, fakeRulebook{}); err == nil {
		t.Error("New(nil state, ...) = nil error, want error")
	}
	if _, err := New(fakeReader{}, nil); err == nil {
		t.Error("New(..., nil rulebook) = nil error, want error")
	}
}

func TestBuildBoard_AggregatesBlendsAndJoins(t *testing.T) {
	rd := fakeReader{players: map[string]state.PlayerState{
		"1001": {MFLID: "1001", FranchiseID: "0001"},
		"1002": {MFLID: "1002", FranchiseID: "0001"},
		"1003": {MFLID: "1003", FranchiseID: "0002"},
	}}
	rb := fakeRulebook{
		names:        map[string]string{"0001": "Alpha", "0002": "Bravo"},
		starterCount: "1",
	}
	svc, err := New(rd, rb)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	standings := []leaguestandings.RawStanding{
		{FranchiseID: "0001", H2HW: "8", H2HL: "5", AllPlayW: "80", AllPlayL: "40", PF: "1500.5"},
		{FranchiseID: "0002", H2HW: "6", H2HL: "7", AllPlayW: "60", AllPlayL: "60", PF: "1400.25"},
	}
	scores := []output.SeasonScore{
		{MFLID: "1001", AdjustedScore: 100},
		{MFLID: "1002", AdjustedScore: 50},
		{MFLID: "1003", AdjustedScore: 200},
	}

	board, err := svc.BuildBoard(standings, scores, 0.5, "sum")
	if err != nil {
		t.Fatalf("BuildBoard: %v", err)
	}
	if board.Mode != AggSum {
		t.Errorf("Mode = %q, want %q", board.Mode, AggSum)
	}
	if len(board.Rows) != 2 {
		t.Fatalf("len(Rows) = %d, want 2", len(board.Rows))
	}
	byFID := map[string]Row{}
	for _, r := range board.Rows {
		byFID[r.FranchiseID] = r
	}
	if got := byFID["0001"].ScoutingScore; got != 150 {
		t.Errorf("0001 ScoutingScore (sum of 100+50) = %v, want 150", got)
	}
	if got := byFID["0002"].ScoutingScore; got != 200 {
		t.Errorf("0002 ScoutingScore = %v, want 200", got)
	}
	if byFID["0001"].Name != "Alpha" || byFID["0002"].Name != "Bravo" {
		t.Errorf("names not joined: 0001=%q 0002=%q", byFID["0001"].Name, byFID["0002"].Name)
	}
	if byFID["0001"].H2HW != 8 || byFID["0001"].H2HL != 5 {
		t.Errorf("0001 h2h passthrough = %d-%d, want 8-5", byFID["0001"].H2HW, byFID["0001"].H2HL)
	}
	// Rank comes back deterministic (Blend sorts descending), both ranks present exactly once.
	seen := map[int]bool{}
	for _, r := range board.Rows {
		seen[r.Rank] = true
	}
	if !seen[1] || !seen[2] {
		t.Errorf("ranks not 1,2: %+v", board.Rows)
	}
}

func TestBuildBoard_TopNDegradesToSumWithoutStarterCount(t *testing.T) {
	rd := fakeReader{players: map[string]state.PlayerState{
		"1001": {MFLID: "1001", FranchiseID: "0001"},
	}}
	rb := fakeRulebook{names: map[string]string{}, starterCount: ""} // unparseable -> starterCount 0
	svc, err := New(rd, rb)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	standings := []leaguestandings.RawStanding{{FranchiseID: "0001"}}
	scores := []output.SeasonScore{{MFLID: "1001", AdjustedScore: 42}}

	board, err := svc.BuildBoard(standings, scores, 0.5, AggTopN)
	if err != nil {
		t.Fatalf("BuildBoard: %v", err)
	}
	if board.Mode != AggSum {
		t.Errorf("Mode = %q, want degrade to %q when starterCount is unreadable", board.Mode, AggSum)
	}
	if board.StarterN != 0 {
		t.Errorf("StarterN = %d, want 0 (echoed only for topn)", board.StarterN)
	}
}

func TestAggregateScouting(t *testing.T) {
	scores := []float64{10, 30, 20}
	if got := aggregateScouting(scores, AggSum, 0); got != 60 {
		t.Errorf("sum = %v, want 60", got)
	}
	if got := aggregateScouting(scores, AggTopN, 2); got != 50 { // top 2: 30+20
		t.Errorf("top-2 = %v, want 50", got)
	}
	if got := aggregateScouting(scores, AggTopN, 10); got != 60 { // N >= len -> whole roster
		t.Errorf("top-N(N>=len) = %v, want 60", got)
	}
	// Caller's slice must be untouched by the top-N sort-on-copy.
	if scores[0] != 10 || scores[1] != 30 || scores[2] != 20 {
		t.Errorf("aggregateScouting mutated caller slice: %v", scores)
	}
}

func TestParseStanding_EmptyFieldsAreZeroAndWinPctOverActualGames(t *testing.T) {
	ps, err := parseStanding(leaguestandings.RawStanding{
		FranchiseID: "0001",
		AllPlayW:    "3", AllPlayL: "1", AllPlayT: "0",
		// H2H/PF/PA/etc left blank on purpose.
	})
	if err != nil {
		t.Fatalf("parseStanding: %v", err)
	}
	if ps.h2hW != 0 || ps.pf != 0 {
		t.Errorf("blank fields did not zero out: h2hW=%d pf=%v", ps.h2hW, ps.pf)
	}
	want := 3.0 / 4.0
	if ps.allPlayWinPct != want {
		t.Errorf("allPlayWinPct = %v, want %v", ps.allPlayWinPct, want)
	}
}

func TestParseStanding_ZeroGamesNeverNaN(t *testing.T) {
	ps, err := parseStanding(leaguestandings.RawStanding{FranchiseID: "0001"})
	if err != nil {
		t.Fatalf("parseStanding: %v", err)
	}
	if ps.allPlayWinPct != 0 {
		t.Errorf("zero-games win pct = %v, want 0 (not NaN)", ps.allPlayWinPct)
	}
}

func TestClampWeight(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{0.5, 0.5},
		{-1, 0},
		{2, 1},
		{math.NaN(), 0.60}, // DefaultScoutingWeight
		{math.Inf(1), 0.60},
	}
	for _, c := range cases {
		if got := clampWeight(c.in); got != c.want {
			t.Errorf("clampWeight(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestFranchiseDisplayName_FallsBackToLabeledID(t *testing.T) {
	names := map[string]string{"0001": "Alpha", "0002": ""}
	if got := franchiseDisplayName(names, "0001"); got != "Alpha" {
		t.Errorf("known name = %q, want Alpha", got)
	}
	if got := franchiseDisplayName(names, "0002"); got != "(franchise 0002)" {
		t.Errorf("empty name = %q, want fallback", got)
	}
	if got := franchiseDisplayName(names, "0099"); got != "(franchise 0099)" {
		t.Errorf("unmapped name = %q, want fallback", got)
	}
}
