package market

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/market/markettest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

func heldFixture(t *testing.T) map[string]Held {
	t.Helper()
	held := map[string]Held{}
	for kind, feed := range markettest.Feeds(t) {
		held[kind] = Held{Value: []marketfeeds.Feed{feed}, Feed: report.Feed{FetchedAt: markettest.Read()}}
	}
	return held
}

func reportByID(t *testing.T, board report.Board, id string) report.Report {
	t.Helper()
	for _, r := range board.Reports {
		if r.ID == id {
			return r
		}
	}
	t.Fatal(id)
	return report.Report{}
}

func TestMarketReportsFactsAndMissing(t *testing.T) {
	t.Parallel()
	held := heldFixture(t)
	names := map[string]string{"16162": "Bijan Robinson", "16161": "Jahmyr Gibbs", "13589": "Josh Allen"}
	board := Reports(held, names, 6, time.UTC)
	for _, r := range board.Reports {
		if r.Status != "ok" || r.AsOf != "As of: Oct 11, 2026, 12:00 PM" || r.How == "" || r.Source == "" {
			t.Fatal(r)
		}
		for _, row := range r.Rows {
			if len(row.Sort) != len(r.Columns) || len(row.Cells) != len(r.Columns) {
				t.Fatal(row)
			}
		}
	}
	projections := reportByID(t, board, "M-046")
	if projections.Rows[0].Cells[1] != "32.21" || !strings.Contains(projections.Title, "week 6") || len(projections.Rows) != 957 {
		t.Fatal(projections.Title)
	}
	ranks := reportByID(t, board, "M-047")
	if ranks.Rows[1].Cells[2] != "Up 1" || ranks.Rows[4].Cells[2] != "Down 3" {
		t.Fatal(ranks.Rows[:5])
	}
	if !slices.ContainsFunc(ranks.Rows, func(row report.Row) bool { return row.Cells[2] == "New this week" && row.Cells[3] == "—" }) {
		t.Fatal("NEW not labelled")
	}
	adp := reportByID(t, board, "M-067")
	if adp.Rows[0].Range.Value != 1.85 || adp.Rows[0].Range.Lo != 1 || adp.Rows[0].Range.Hi != 34 || !strings.Contains(adp.Note, "6734 drafts") {
		t.Fatal(adp.Rows[0])
	}
	if got := adp.Rows[0].Range.Text; got != "Pick 1.9 · picks 1–34" {
		t.Fatalf("ADP text %q", got)
	}
	aav := reportByID(t, board, "M-069")
	// The costliest value leads.
	if got := aav.Rows[0].Range.Text; got != "$30.1 of a $1,000 league pool · $3.4–$113.6" {
		t.Fatalf("AAV text %q", got)
	}
	if aav.Rows[0].Range.Value != 30.1 || !strings.Contains(aav.How, "not this league's dollars") || !strings.Contains(aav.Note, "111 auctions") {
		t.Fatal(aav.How)
	}
	// Every bar in a report shares one axis, so bars compare down the column; the pool's facts
	// read against the same axis as the report.
	for report, axis := range map[*report.Report]float64{&adp: 848, &aav: 164.14} {
		for _, row := range report.Rows {
			if row.Range.Max != axis {
				t.Fatalf("%s bar axis %v, want %v", report.ID, row.Range.Max, axis)
			}
		}
	}
	facts := Facts(held, time.UTC)
	if got := facts["16162"][2].Range.Max; got != 848 {
		t.Fatalf("pool ADP axis %v", got)
	}
	if len(facts["16162"]) != 4 {
		t.Fatal(facts["16162"])
	}
	for _, f := range facts[""] {
		if f.Text != "—" || f.Range != nil || !strings.Contains(f.Reason, "does not list this player") {
			t.Fatal(f)
		}
	}
	for _, f := range Facts(nil, time.UTC)[""] {
		if f.Text != "—" || f.Range != nil || !strings.Contains(f.Reason, "unavailable") {
			t.Fatal(f)
		}
	}
}

func TestMarketEmptyADPAndIndependentFailures(t *testing.T) {
	t.Parallel()
	held := heldFixture(t)
	feed, err := marketfeeds.Parse(marketfeeds.ADP, []byte(`{"adp":{"totalDrafts":"0","totalPicks":"0"}}`))
	if err != nil {
		t.Fatal(err)
	}
	h := held[marketfeeds.ADP]
	h.Value = []marketfeeds.Feed{feed}
	held[marketfeeds.ADP] = h
	held[marketfeeds.Ranks] = Held{Feed: report.Feed{RefreshError: "offline"}}
	b := Reports(held, nil, 6, time.UTC)
	adp := reportByID(t, b, "M-067")
	if adp.Status != "empty" || len(adp.Rows) != 0 || !strings.Contains(adp.Note, "no draft picks") {
		t.Fatal(adp)
	}
	if reportByID(t, b, "M-047").Status != "failed" || reportByID(t, b, "M-046").Status != "ok" {
		t.Fatal("failure blocked good feed")
	}
}

func TestMarketNamesLabelTeamUnits(t *testing.T) {
	t.Parallel()
	lookup, err := normalize.NewLookup([]players.RawPlayer{
		{ID: "16162", Name: "Robinson, Bijan", Position: "RB", Team: "ATL"},
		{ID: "0513", Name: "Broncos, Denver", Position: "Def", Team: "DEN"},
	})
	if err != nil {
		t.Fatal(err)
	}
	names := Names(heldFixture(t), lookup)
	if names["16162"] != "Robinson, Bijan · RB · ATL" || names["0513"] != "Broncos, Denver · team unit" {
		t.Fatal(names["16162"], names["0513"])
	}
}

func TestArchiveScopeMatchesTheKindAndWeek(t *testing.T) {
	t.Parallel()
	for _, scope := range []struct {
		url, kind string
		week      int
		want      bool
	}{
		{"https://fixture.invalid/2026/export?W=6", marketfeeds.Projections, 6, true},
		{"https://fixture.invalid/2026/export?W=5", marketfeeds.Projections, 6, false},
		{"https://fixture.invalid/2026/export?PERIOD=ALL", marketfeeds.ADP, 0, true},
		{"https://fixture.invalid/2026/export", marketfeeds.ADP, 0, false},
	} {
		if ArchiveScope(scope.url, scope.kind, scope.week) != scope.want {
			t.Fatal(scope)
		}
	}
}

func TestWeekTextNeverPrintsWeekZero(t *testing.T) {
	t.Parallel()
	if WeekText(0) != "NFL week not known" || WeekText(19) != "NFL week not known" || WeekText(6) != "NFL week 6" {
		t.Fatal(WeekText(0), WeekText(19), WeekText(6))
	}
}
