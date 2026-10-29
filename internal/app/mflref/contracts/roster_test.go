package contracts

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/mflref/lineup"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func pid(t *testing.T, raw string) playerid.PlayerID {
	t.Helper()
	id, err := playerid.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// heldRoster is franchise 0025: two active QBs, an active WR and a taxi WR; 0026 holds an active QB.
func heldRoster(t *testing.T) snapshot.Snapshot {
	t.Helper()
	snap := snapshot.Snapshot{}
	snap.Players.Value = []snapshot.Player{
		{ID: pid(t, "1001"), Name: "Zed", Position: domain.PosQB, Team: "BUF"}, {ID: pid(t, "1002"), Name: "Abe", Position: domain.PosQB, Team: "BUF"},
		{ID: pid(t, "1003"), Name: "Cal", Position: domain.PosWR, Team: "KC"}, {ID: pid(t, "1004"), Name: "Dee", Position: domain.PosWR, Team: "KC"},
		{ID: pid(t, "1005"), Name: "Eve", Position: domain.PosQB, Team: "KC"},
	}
	snap.Rosters.Value = []snapshot.Roster{{FranchiseID: "0025", Players: []snapshot.RosterPlayer{
		{ID: pid(t, "1003"), RosterStatus: domain.RosterActive}, {ID: pid(t, "1001"), RosterStatus: domain.RosterActive},
		{ID: pid(t, "1002"), RosterStatus: domain.RosterActive}, {ID: pid(t, "1004"), RosterStatus: domain.RosterTaxi},
		{ID: pid(t, "1006"), RosterStatus: domain.RosterActive},
	}}, {FranchiseID: "0026", Players: []snapshot.RosterPlayer{{ID: pid(t, "1005"), RosterStatus: domain.RosterActive}}}}
	return snap
}

func heldRules() lineup.Rules {
	return lineup.Rules{Positions: []lineup.PositionRule{{Position: domain.PosQB, Min: 1, Max: 1}, {Position: domain.PosWR, Min: 1, Max: 3}}}
}

func TestAcquisitionValidateRefusesEachMalformedRequest(t *testing.T) {
	t.Parallel()
	for want, r := range map[string]Acquisition{
		"acquisition scope":                         {Mine: "bad"},
		"choose franchise and one source":           {Mine: "0001", Player: "0042", Bid: "b"},
		"acquisition player: playerid: \"x\" is":    {Mine: "0001", Player: "x"},
		"acquisition player: playerid: \"cut\" is ": {Mine: "0001", Cut: "cut"},
	} {
		if err := r.Validate(); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%+v: %v, want %q", r, err, want)
		}
	}
	if err := (Acquisition{Mine: "0001", Bid: "b", Cut: "0042"}).Validate(); err != nil {
		t.Fatal(err)
	}
	snap := heldRoster(t)
	if Name(snap, "1002") != "Abe" || Name(snap, "9999") != "Name unavailable" {
		t.Fatal("names")
	}
}

func TestSelectionRefusesAMalformedOrRepeatedStarter(t *testing.T) {
	t.Parallel()
	if ids, err := (Selection{Starters: []string{"1001", "1003"}}).Validate(); err != nil || len(ids) != 2 || ids[1].String() != "1003" {
		t.Fatal(ids, err)
	}
	if _, err := (Selection{Starters: []string{"1001", "x"}}).Validate(); err == nil || !strings.HasPrefix(err.Error(), "lineup selection: player") {
		t.Fatal(err)
	}
	if _, err := (Selection{Starters: []string{"1001", "1001"}}).Validate(); err == nil || err.Error() != "lineup selection: duplicate player 1001" {
		t.Fatal(err)
	}
	r := UnreadableRules(errors.New("count missing"))
	if len(r.Problems) != 1 || r.Problems[0].Kind != lineup.KindUnknown || r.Problems[0].Message != "The league's lineup rules could not be read: count missing" {
		t.Fatal(r)
	}
}

func TestCompetingPlansChangeTheSameWeekOrTrade(t *testing.T) {
	t.Parallel()
	week := func(intent string, w int) envelope.Spec {
		return envelope.Spec{Intent: intent, Expected: envelope.ExpectedChange{Lineup: &envelope.ExpectedLineup{Week: w}}}
	}
	trade := func(intent, id string) envelope.Spec {
		return envelope.Spec{Intent: intent, Expected: envelope.ExpectedChange{Trade: &envelope.ExpectedTrade{TradeID: id}}}
	}
	for _, tc := range []struct {
		left, right envelope.Spec
		want        bool
	}{
		{week("mfl.lineup.set", 5), week("mfl.lineup.set", 5), true}, {week("mfl.lineup.set", 5), week("mfl.lineup.set", 6), false},
		{trade("mfl.trade.accept", "t1"), trade("mfl.trade.accept", "t1"), true}, {trade("trade.revoke", "t1"), trade("trade.revoke", "t2"), false},
		{trade("mfl.trade.accept", "t1"), trade("trade.reject", "t1"), false}, {week("roster.ir", 5), week("roster.ir", 5), false},
	} {
		if got := Competing(tc.left, tc.right); got != tc.want {
			t.Errorf("%+v %+v: %v", tc.left, tc.right, got)
		}
	}
}

func TestLayoutOrdersStartersAndBenchByRulesThenName(t *testing.T) {
	t.Parallel()
	snap := heldRoster(t)
	saved := leaguefeed.Lineups{Franchises: []leaguefeed.Lineup{{Franchise: "0026"}, {Franchise: "0025", Starters: []playerid.PlayerID{pid(t, "1003"), pid(t, "1001")}}}}
	starters, bench, found := Layout(snap, saved, "0025", heldRules())
	if !found || len(starters) != 2 || starters[0].ID != "1001" || starters[1].Position != domain.PosWR {
		t.Fatalf("starters %+v %v", starters, found)
	}
	// The unknown player sorts last: no position, so after every rule.
	if len(bench) != 2 || bench[0].ID != "1002" || bench[1].ID != "1006" || bench[1].Position != "" {
		t.Fatalf("bench %+v", bench)
	}
	tied := heldRoster(t)
	tied.Players.Value[0].Name, tied.Players.Value[1].Name = "Same", "Same"
	if _, bench, _ := Layout(tied, leaguefeed.Lineups{}, "0025", heldRules()); bench[0].ID != "1001" || bench[1].ID != "1002" {
		t.Fatalf("a name tie is not broken by id: %+v", bench)
	}
	starters, bench, found = Layout(snap, leaguefeed.Lineups{}, "0025", heldRules())
	if found || len(starters) != 0 || starters == nil || len(bench) != 4 {
		t.Fatalf("no saved lineup %+v %+v %v", starters, bench, found)
	}
}

func TestRosterReportsReadHeldScoresAndByes(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	season := Scores{Feed: report.Feed{FetchedAt: at}, Rows: []ingestion.RawPlayerScore{{ID: "1001", Score: "134.80"}}}
	week := Scores{Feed: report.Feed{FetchedAt: at}, Rows: []ingestion.RawPlayerScore{{ID: "1001", Score: "0.00"}}}
	byes := Byes{Feed: report.Feed{FetchedAt: at}, Rows: []nflschedule.RawBye{{Team: "BUF", Week: "7"}, {Team: "KC", Week: "9"}}}
	stats, bye := RosterReports(heldRoster(t), "0025", season, week, byes, heldRules(), time.UTC)
	if stats.ID != "M-101" || stats.Status != "ok" || len(stats.Rows) != 5 || !strings.Contains(stats.Note, "Last-week scores as of ") {
		t.Fatalf("stats %+v", stats)
	}
	if got := strings.Join(stats.Rows[1].Cells, "|"); got != "Zed|QB|BUF|134.80|0.00" {
		t.Fatal(got)
	}
	if got := strings.Join(stats.Rows[4].Cells, "|"); got != "Player name unavailable|||No score|No score" {
		t.Fatal(got)
	}
	if bye.ID != "M-154" || bye.Note != "" || strings.Join(bye.Rows[0].Cells, "|") != "Cal|WR|KC|9" || bye.Rows[4].Cells[3] != "Unavailable" {
		t.Fatalf("bye rows %+v", bye.Rows)
	}
	// Week 7 takes both QBs and week 9 the only active WR (the taxi WR does not count).
	if len(bye.Summary) != 2 || strings.Join(bye.Summary[0].Cells, "|") != "Week 7 · QB|0 available; needs 1" ||
		strings.Join(bye.Summary[1].Cells, "|") != "Week 9 · WR|0 available; needs 1" {
		t.Fatalf("bye summary %+v", bye.Summary)
	}
	stats, bye = RosterReports(heldRoster(t), "0025", season, Scores{}, byes, lineup.Rules{}, time.UTC)
	if stats.Status != "stale" || !strings.HasSuffix(stats.Note, " Last-week points unavailable.") || bye.Note != "No position minimum runs short in the held bye-week schedule." {
		t.Fatalf("missing week %+v / %+v", stats, bye)
	}
}
