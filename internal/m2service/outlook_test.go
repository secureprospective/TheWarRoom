package m2service

import (
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func game(home, away string) leagueschedule.RawMatchup {
	return leagueschedule.RawMatchup{Franchises: [2]leagueschedule.RawMatchupSide{
		{FranchiseID: home, IsHome: "1"}, {FranchiseID: away, IsHome: "0"}}}
}

func TestOutlooksCapLuckAndProjectedRecord(t *testing.T) {
	rd := fakeReader{
		players: map[string]state.PlayerState{"1": {FranchiseID: "0001"}, "2": {FranchiseID: "0002"}},
		capUsed: map[string]domain.Money{"0001": 120 * 1_000_000 * 100, "0002": 100 * 1_000_000 * 100},
	}
	rb := fakeRulebook{cap: "125", starters: &league.Starters{Count: "1", IDPStarters: "0",
		Positions: []league.PositionLimit{{Name: "QB", Limit: "1"}}}}
	svc, err := New(rd, rb)
	if err != nil {
		t.Fatal(err)
	}
	// Week 1 is played: 0001 won while scoring below the league (all-play 0 of 1).
	standings := []leaguestandings.RawStanding{
		{FranchiseID: "0001", H2HW: "1", PF: "150", AllPlayW: "0", AllPlayL: "1"},
		{FranchiseID: "0002", H2HL: "1", PF: "200", AllPlayW: "1", AllPlayL: "0"},
	}
	now := []PlayerValue{{MFLID: "1", Position: "QB", Value: 150}, {MFLID: "2", Position: "QB", Value: 200}}
	sched := []leagueschedule.RawScheduleWeek{
		{Week: "1", Matchups: []leagueschedule.RawMatchup{game("0001", "0002")}},
		{Week: "2", Matchups: []leagueschedule.RawMatchup{game("0002", "0001")}},
		{Week: "3", Matchups: []leagueschedule.RawMatchup{game("0001", "0002")}},
		{Week: "14", Matchups: []leagueschedule.RawMatchup{game("0001", "0002")}}, // playoffs: not counted
	}
	out, err := svc.Outlooks(OutlookInputs{Standings: standings, Now: now, Schedule: sched, SeasonWeeks: 3,
		DeadCap: map[string]domain.Money{"0001": 2 * 1_000_000 * 100}})
	if err != nil {
		t.Fatal(err)
	}
	a, b := out["0001"], out["0002"]
	if !a.HasCap || a.CapRoom != 5 || a.DeadCap != 2 || b.CapRoom != 25 {
		t.Errorf("cap: %+v / %+v", a, b)
	}
	if !a.HasLuck || a.Luck != 1 || b.Luck != -1 {
		t.Errorf("luck: 0001 %v, 0002 %v; want +1 and −1", a.Luck, b.Luck)
	}
	if !a.HasProj || math.Abs(a.ProjW+a.ProjL-3) > 1e-9 || math.Abs(a.ProjW+b.ProjW-3) > 1e-9 {
		t.Errorf("projected records must cover 3 games and hand out 3 wins: %+v / %+v", a, b)
	}
	if a.ProjW >= 1.5 || a.ProjW <= 1 {
		t.Errorf("0001 has one win and is the weaker team in two games left: projected %v wins", a.ProjW)
	}
	none, err := svc.Outlooks(OutlookInputs{Standings: standings, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if none["0001"].HasProj {
		t.Error("no schedule must mean no projection")
	}
}
