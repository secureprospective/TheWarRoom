package m2service

import (
	"math"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/model"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// fakeReader is a minimal state.Reader: only Player feeds the franchise-aggregation
// join, so the rest are stubbed to satisfy the interface (mirrors
// internal/transactions/pricing_test.go's fakeReader).
type fakeReader struct {
	players map[string]state.PlayerState // mflID -> state
	capUsed map[string]domain.Money      // franchise -> cap charged
}

func (f fakeReader) Franchises() []string                      { return nil }
func (f fakeReader) Roster(string) ([]state.PlayerState, bool) { return nil, false }
func (f fakeReader) FranchiseState(string) (state.FranchiseState, bool) {
	return state.FranchiseState{}, false
}
func (f fakeReader) CapUsed(fid string) (domain.Money, bool) {
	m, ok := f.capUsed[fid]
	return m, ok
}
func (f fakeReader) Player(mflID string) (state.PlayerState, bool) {
	p, ok := f.players[mflID]
	return p, ok
}

// fakeRulebook is a minimal FranchiseSource.
type fakeRulebook struct {
	names        map[string]string
	starterCount string
	starters     *league.Starters // the full rules; nil gives only the count
	cap          string
}

func (f fakeRulebook) FranchiseNames() map[string]string { return f.names }
func (f fakeRulebook) ActiveConfig() league.RawConfig {
	if f.starters != nil {
		return league.RawConfig{Starters: *f.starters}
	}
	return league.RawConfig{Starters: league.Starters{Count: f.starterCount}}
}
func (f fakeRulebook) GetSalaryCap() string { return f.cap }

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
	scores := []PlayerValue{
		score("1001", 100),
		score("1002", 50),
		score("1003", 200),
	}

	board, err := svc.BuildBoard(standings, scores, powerrankings.Weights{Roster: 0.5}, "sum", PickCapital{})
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
	if got := byFID["0001"].RosterValue; got != 150 {
		t.Errorf("0001 RosterValue (sum of 100+50) = %v, want 150", got)
	}
	if got := byFID["0002"].RosterValue; got != 200 {
		t.Errorf("0002 RosterValue = %v, want 200", got)
	}
	if byFID["0001"].Name != "Alpha" || byFID["0002"].Name != "Bravo" {
		t.Errorf("names not joined: 0001=%q 0002=%q", byFID["0001"].Name, byFID["0002"].Name)
	}
	if byFID["0001"].H2HW != 8 || byFID["0001"].H2HL != 5 {
		t.Errorf("0001 h2h passthrough = %d-%d, want 8-5", byFID["0001"].H2HW, byFID["0001"].H2HL)
	}
	// Rank comes back deterministic (Blend sorts descending by PowerScore) — GLM 5.2
	// review lead A2 (Session 43): a set-membership check alone would pass even if sort
	// order inverted or both rows landed on the same rank. With only 2 franchises and
	// weight=0.5, 0001's stronger all-play record (0.667 vs 0.500 win%, a full z-score
	// apart on a 2-point distribution) outweighs 0002's higher raw roster sum — this
	// is powerrankings.Blend's existing z-score math, not something this test asserts
	// independently; the exact expected ranks were confirmed by running the case, not
	// derived from roster-value intuition alone.
	if byFID["0001"].Rank != 1 {
		t.Errorf("0001 Rank = %d, want 1 (all-play component dominates at these inputs)", byFID["0001"].Rank)
	}
	if byFID["0002"].Rank != 2 {
		t.Errorf("0002 Rank = %d, want 2", byFID["0002"].Rank)
	}
}

// The blend reads all-play when MFL reports it, otherwise points for as a share of the league's
// best, and nothing before any result; each row carries the result read.
func TestBuildBoard_ResultsFallBackFromAllPlayToPointsFor(t *testing.T) {
	svc, err := New(fakeReader{}, fakeRulebook{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cases := []struct {
		name      string
		standings []leaguestandings.RawStanding
		perf      string
		results   map[string]float64
	}{
		{"all-play", []leaguestandings.RawStanding{
			{FranchiseID: "0001", AllPlayW: "3", AllPlayL: "1", PF: "400"},
			{FranchiseID: "0002", AllPlayW: "1", AllPlayL: "3", PF: "500"},
		}, PerfAllPlay, map[string]float64{"0001": 0.75, "0002": 0.25}},
		{"points for", []leaguestandings.RawStanding{
			{FranchiseID: "0001", H2HW: "1", PF: "400"},
			{FranchiseID: "0002", H2HL: "1", PF: "500"},
		}, PerfPointsFor, map[string]float64{"0001": 0.8, "0002": 1}},
		{"none", []leaguestandings.RawStanding{{FranchiseID: "0001"}, {FranchiseID: "0002"}},
			PerfNone, map[string]float64{"0001": 0, "0002": 0}},
	}
	for _, c := range cases {
		board, err := svc.BuildBoard(c.standings, nil, powerrankings.Weights{Roster: 0.6}, AggSum, PickCapital{})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if board.Performance != c.perf {
			t.Errorf("%s: Performance = %q, want %q", c.name, board.Performance, c.perf)
		}
		for _, r := range board.Rows {
			if math.Abs(r.Results-c.results[r.FranchiseID]) > 1e-12 {
				t.Errorf("%s: %s Results = %v, want %v", c.name, r.FranchiseID, r.Results, c.results[r.FranchiseID])
			}
		}
	}
}

// TestBuildBoard_FranchiseWithNoScoresContributesZero covers a franchise present in
// standings but with no scored players (GLM 5.2 review lead A4, Session 43): an
// expansion/empty-roster franchise must still get a row, with 0 roster value rather than
// being dropped or erroring.
func TestBuildBoard_FranchiseWithNoScoresContributesZero(t *testing.T) {
	rd := fakeReader{players: map[string]state.PlayerState{
		"1001": {MFLID: "1001", FranchiseID: "0001"},
	}}
	rb := fakeRulebook{names: map[string]string{}, starterCount: "1"}
	svc, err := New(rd, rb)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	standings := []leaguestandings.RawStanding{
		{FranchiseID: "0001"},
		{FranchiseID: "0002"}, // no players scored for this franchise
	}
	scores := []PlayerValue{score("1001", 100)}

	board, err := svc.BuildBoard(standings, scores, powerrankings.Weights{Roster: 0.5}, AggSum, PickCapital{})
	if err != nil {
		t.Fatalf("BuildBoard: %v", err)
	}
	if len(board.Rows) != 2 {
		t.Fatalf("len(Rows) = %d, want 2 (0002 must not be dropped)", len(board.Rows))
	}
	byFID := map[string]Row{}
	for _, r := range board.Rows {
		byFID[r.FranchiseID] = r
	}
	if got := byFID["0002"].RosterValue; got != 0 {
		t.Errorf("0002 (no scores) RosterValue = %v, want 0", got)
	}
}

// TestBuildBoard_EmptyInputsAreNoOps covers empty standings/scores (GLM 5.2 review
// lead A5, Session 43): must return an empty board, not error or panic.
func TestBuildBoard_EmptyInputsAreNoOps(t *testing.T) {
	svc, err := New(fakeReader{}, fakeRulebook{names: map[string]string{}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	board, err := svc.BuildBoard(nil, nil, powerrankings.Weights{Roster: 0.5}, AggSum, PickCapital{})
	if err != nil {
		t.Fatalf("BuildBoard(empty): %v", err)
	}
	if len(board.Rows) != 0 {
		t.Errorf("len(Rows) = %d, want 0", len(board.Rows))
	}
}

// TestBuildBoard_UnmappedFranchiseFallsBackToLabeledID is the A7 end-to-end
// counterpart to TestFranchiseDisplayName_FallsBackToLabeledID (GLM 5.2 review lead
// A7, Session 43): a RawStanding whose FranchiseID has no rulebook name must still
// produce a row, labeled, through the full BuildBoard path — not just the unit.
func TestBuildBoard_UnmappedFranchiseFallsBackToLabeledID(t *testing.T) {
	rd := fakeReader{}
	rb := fakeRulebook{names: map[string]string{}, starterCount: "1"} // no franchise names at all
	svc, err := New(rd, rb)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	board, err := svc.BuildBoard([]leaguestandings.RawStanding{{FranchiseID: "0099"}}, nil, powerrankings.Weights{Roster: 0.5}, AggSum, PickCapital{})
	if err != nil {
		t.Fatalf("BuildBoard: %v", err)
	}
	if len(board.Rows) != 1 || board.Rows[0].Name != "Franchise 0099" {
		t.Errorf("Rows = %+v, want one row named \"Franchise 0099\"", board.Rows)
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
	scores := []PlayerValue{score("1001", 42)}

	board, err := svc.BuildBoard(standings, scores, powerrankings.Weights{Roster: 0.5}, AggTopN, PickCapital{})
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

func TestCountByMode(t *testing.T) {
	vals := []PlayerValue{
		{Position: "QB", Value: 10, Age: 30}, {Position: "QB", Value: 30, Age: 24}, {Position: "WR", Value: 20, Age: math.NaN()},
	}
	rules := powerrankings.LineupRules{Total: 2, Defense: 0, Slots: []powerrankings.Slot{
		{Position: "QB", Min: 1, Max: 1}, {Position: "WR", Min: 1, Max: 2}}}
	for _, c := range []struct {
		mode       string
		value, age float64
	}{
		{AggSum, 60, (10*30 + 30*24) / 40.0},
		{AggTopN, 50, 24},   // top 2 by value: QB 30 and WR 20, whose age is unknown
		{AggLineup, 50, 24}, // one QB may start: QB 30 and WR 20
	} {
		value, age := counter{mode: c.mode, rules: rules}.count(vals)
		if value != c.value || math.Abs(age-c.age) > 1e-9 {
			t.Errorf("%s: %v at age %v, want %v at %v", c.mode, value, age, c.value, c.age)
		}
	}
	rules.Total = 3
	rules.Slots[1].Max = 1
	if value, _ := (counter{mode: AggLineup, rules: rules}).count(vals); value != 50 {
		t.Errorf("a second QB must not start in the open slot: lineup %v, want 50", value)
	}
	if vals[0].Value != 10 || vals[1].Value != 30 {
		t.Errorf("count reordered the caller's slice: %v", vals)
	}
	if _, age := (counter{mode: AggSum}).count(nil); !math.IsNaN(age) {
		t.Errorf("an empty roster's age = %v, want NaN", age)
	}
}

func TestLineupRulesReadMFLsStarters(t *testing.T) {
	st := league.Starters{Count: "21", IDPStarters: "12", Positions: []league.PositionLimit{
		{Name: "QB", Limit: "1"}, {Name: "RB", Limit: "1-3"}, {Name: "DT", Limit: "2-4"}}}
	r, ok := LineupRules(st)
	if !ok || r.Total != 21 || r.Defense != 12 || len(r.Slots) != 3 {
		t.Fatalf("rules %+v ok %v", r, ok)
	}
	if r.Slots[0] != (powerrankings.Slot{Position: "QB", Min: 1, Max: 1}) ||
		r.Slots[1] != (powerrankings.Slot{Position: "RB", Min: 1, Max: 3}) {
		t.Fatalf("bounds misread: %+v", r.Slots)
	}
	for _, bad := range []string{"", "x", "3-1", "1-"} {
		st.Positions[0].Limit = bad
		if _, ok := LineupRules(st); ok {
			t.Errorf("limit %q should not read", bad)
		}
	}
	if ResolveAggMode("nope", AggLineup) != AggLineup || ResolveAggMode(AggTopN, AggSum) != AggTopN {
		t.Error("ResolveAggMode")
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

func score(mflID string, value float64) PlayerValue {
	return PlayerValue{MFLID: mflID, Value: value, Age: math.NaN()}
}

func TestLineupCountsCanonicalRulePositions(t *testing.T) {
	for _, tc := range []struct {
		name, rule, model string
		defense           string
	}{
		{"kicker", "PK", "K", "0"},
		{"edge alias", "EDGE", "DE", "1"},
		{"unchanged offense", "QB", "QB", "0"},
		{"unchanged defense", "DE", "DE", "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := league.Starters{Count: "2", IDPStarters: tc.defense, Positions: []league.PositionLimit{
				{Name: tc.rule, Limit: "1"}, {Name: "WR", Limit: "1-2"},
			}}
			svc, err := New(fakeReader{}, fakeRulebook{starters: &st})
			if err != nil {
				t.Fatal(err)
			}
			c := svc.counter(AggLineup)
			if c.mode != AggLineup {
				t.Fatalf("lineup unexpectedly fell back to %s", c.mode)
			}
			values := []PlayerValue{
				{Position: tc.model, Value: 10, Age: 30},
				{Position: "WR", Value: 40, Age: 20},
				{Position: "WR", Value: 30, Age: 25},
			}
			value, age := c.count(values)
			if value != 50 || age != 22 {
				t.Fatalf("%s rule did not count %s: value %v age %v, want 50 and 22", tc.rule, tc.model, value, age)
			}
			if values[0].Position != tc.model || st.Positions[0].Name != tc.rule {
				t.Fatal("normalization mutated the supplied model values or rules")
			}
		})
	}
}

func TestPicksAddAfterLineupWithoutChangingAge(t *testing.T) {
	rd := fakeReader{players: map[string]state.PlayerState{"1": {FranchiseID: "A"}, "2": {FranchiseID: "A"}, "3": {FranchiseID: "B"}}}
	svc, err := New(rd, fakeRulebook{starters: &league.Starters{Count: "1", IDPStarters: "0", Positions: []league.PositionLimit{{Name: "QB", Limit: "1"}}}})
	if err != nil {
		t.Fatal(err)
	}
	standings := []leaguestandings.RawStanding{{FranchiseID: "A"}, {FranchiseID: "B"}}
	values := []PlayerValue{{MFLID: "1", Position: "QB", Value: 10, Age: 30}, {MFLID: "2", Position: "QB", Value: 8, Age: 20}, {MFLID: "3", Position: "QB", Value: 12, Age: 25}}
	wt := powerrankings.Weights{Roster: 1}
	base, err := svc.BuildBoard(standings, values, wt, AggLineup, PickCapital{})
	if err != nil {
		t.Fatal(err)
	}
	for _, total := range []float64{0, 10, math.NaN(), math.Inf(1), -1} {
		b, err := svc.BuildBoard(standings, values, wt, AggLineup, PickCapital{Totals: map[string]float64{"A": total, "B": 0}, Weight: .6})
		if err != nil {
			t.Fatal(err)
		}
		var a Row
		for _, r := range b.Rows {
			if r.FranchiseID == "A" {
				a = r
			}
		}
		if a.Age != 30 || b.Mode != AggLineup || b.StarterN != 1 {
			t.Fatal("bench/picks changed lineup or age", a, b)
		}
		if total == 10 {
			if a.RosterValue != 16 || a.PickValue != 10 || !a.HasPicks || a.Rank != 1 || base.Rows[0].FranchiseID != "B" {
				t.Fatal("picks did not add after lineup", a, base)
			}
		} else if a.RosterValue != 10 || math.IsNaN(a.PickValue) || math.IsInf(a.PickValue, 0) {
			t.Fatal("unreadable picks affected board", a)
		}
	}
}

type rosterReader struct{ fakeReader }

func (r rosterReader) Franchises() []string { return []string{"0001", "0002"} }
func (r rosterReader) Roster(fid string) ([]state.PlayerState, bool) {
	var roster []state.PlayerState
	for _, p := range r.players {
		if p.FranchiseID == fid {
			roster = append(roster, p)
		}
	}
	return roster, true
}

func TestTeamValueRetainsReducedAndUnavailableReasons(t *testing.T) {
	why := "league points for 2024 are not ready"
	rd := rosterReader{fakeReader{players: map[string]state.PlayerState{
		"1001": {MFLID: "1001", FranchiseID: "0001"}, "1002": {MFLID: "1002", FranchiseID: "0001"}, "1003": {MFLID: "1003", FranchiseID: "0002"},
	}}}
	svc, err := New(rd, fakeRulebook{})
	if err != nil {
		t.Fatal(err)
	}
	vals := []PlayerValue{
		{MFLID: "1001", Value: 10, InputStatus: model.InputStatus{ReducedInputs: true, Reason: why}},
		{MFLID: "1002", InputStatus: model.InputStatus{Unavailable: true, Reason: "games played for 2025 are not ready"}},
		{MFLID: "1003", InputStatus: model.InputStatus{Unavailable: true, Reason: why}},
	}
	board, err := svc.BuildBoard([]leaguestandings.RawStanding{{FranchiseID: "0001"}, {FranchiseID: "0002"}}, vals, powerrankings.Weights{Roster: 1}, AggSum, PickCapital{})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range board.Rows {
		if !row.ReducedInputs || !strings.Contains(row.Reason, why) {
			t.Fatal(row)
		}
		if row.FranchiseID == "0001" && (row.Unavailable || row.RosterValue != 10 || !strings.Contains(row.Reason, "games played for 2025")) {
			t.Fatal(row)
		}
		if row.FranchiseID == "0002" && (!row.Unavailable || row.RosterValue != 0) {
			t.Fatal(row)
		}
	}
}
