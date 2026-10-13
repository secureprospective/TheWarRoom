package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

func realMarketHeld(t *testing.T) map[string]heldSeasonFeed[[]marketfeeds.Feed] {
	t.Helper()
	held := map[string]heldSeasonFeed[[]marketfeeds.Feed]{}
	for _, kind := range marketKinds() {
		body := seasonFixture(t, "marketfeeds", kind+".json")
		value, err := marketfeeds.Parse(kind, []byte(body))
		if err != nil {
			t.Fatal(err)
		}
		held[kind] = heldSeasonFeed[[]marketfeeds.Feed]{value: []marketfeeds.Feed{value}, fetchedAt: time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)}
	}
	return held
}

func TestMarketReportsFactsAndMissing(t *testing.T) {
	held := realMarketHeld(t)
	names := map[string]string{"16162": "Bijan Robinson", "16161": "Jahmyr Gibbs", "13589": "Josh Allen"}
	board := buildMarketReports(held, names, 6, time.UTC)
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
	if !slices.ContainsFunc(ranks.Rows, func(row ReportRow) bool { return row.Cells[2] == "New this week" && row.Cells[3] == "—" }) {
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
	for report, axis := range map[*Report]float64{&adp: 848, &aav: 164.14} {
		for _, row := range report.Rows {
			if row.Range.Max != axis {
				t.Fatalf("%s bar axis %v, want %v", report.ID, row.Range.Max, axis)
			}
		}
	}
	facts := marketFacts(held)
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
	for _, f := range marketFacts(nil)[""] {
		if f.Text != "—" || f.Range != nil || !strings.Contains(f.Reason, "unavailable") {
			t.Fatal(f)
		}
	}
}

func TestMarketEmptyADPAndIndependentFailures(t *testing.T) {
	held := realMarketHeld(t)
	feed, err := marketfeeds.Parse(marketfeeds.ADP, []byte(`{"adp":{"totalDrafts":"0","totalPicks":"0"}}`))
	if err != nil {
		t.Fatal(err)
	}
	h := held[marketfeeds.ADP]
	h.value = []marketfeeds.Feed{feed}
	held[marketfeeds.ADP] = h
	held[marketfeeds.Ranks] = heldSeasonFeed[[]marketfeeds.Feed]{refreshError: "offline"}
	b := buildMarketReports(held, nil, 6, time.UTC)
	adp := reportByID(t, b, "M-067")
	if adp.Status != "empty" || len(adp.Rows) != 0 || !strings.Contains(adp.Note, "no draft picks") {
		t.Fatal(adp)
	}
	if reportByID(t, b, "M-047").Status != "failed" || reportByID(t, b, "M-046").Status != "ok" {
		t.Fatal("failure blocked good feed")
	}
}

func TestMarketRefreshDailyWeekAndHeldReads(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	a.week = leagueweek.Week{Number: 6}
	transport.bodies[marketfeeds.Projections] = seasonFixture(t, "marketfeeds", "projectedScores.json")
	if err := a.TargetMarketReportsRefresh(false); err != nil {
		t.Fatal(err)
	}
	if len(transport.requests) != 4 {
		t.Fatal(transport.requests)
	}
	for range 2 {
		if err := a.TargetMarketReportsRefresh(false); err != nil {
			t.Fatal(err)
		}
		if _, err := a.TargetMarketReports(); err != nil {
			t.Fatal(err)
		}
	}
	if len(transport.requests) != 4 {
		t.Fatal("held read or fresh refresh fetched")
	}
	a.week.Number = 7
	if a.heldMarket()[marketfeeds.Projections].fetchedAt != (time.Time{}) {
		t.Fatal("old projection masquerades as this week")
	}
	transport.bodies[marketfeeds.Projections] = strings.Replace(transport.bodies[marketfeeds.Projections], `"week":"6"`, `"week":"7"`, 1)
	if err := a.TargetMarketReportsRefresh(false); err != nil {
		t.Fatal(err)
	}
	if len(transport.requests) != 5 || transport.requests[4] != marketfeeds.Projections || transport.weeks[4] != "7" {
		t.Fatal(transport.requests)
	}
	a.seasonMu.Lock()
	h := a.seasonMarket[marketfeeds.ADP]
	h.fetchedAt = time.Now().Add(-25 * time.Hour)
	a.seasonMarket[marketfeeds.ADP] = h
	a.seasonMu.Unlock()
	if err := a.TargetMarketReportsRefresh(false); err != nil {
		t.Fatal(err)
	}
	if len(transport.requests) != 6 || transport.requests[5] != marketfeeds.ADP {
		t.Fatal("daily ADP not refreshed")
	}
	if err := a.TargetMarketReportsRefresh(true); err != nil {
		t.Fatal(err)
	}
	if len(transport.requests) != 10 {
		t.Fatal("force did not refresh all feeds")
	}
}

func TestMarketArchiveScopeAndFallback(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	a.week.Number = 6
	at := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	for _, kind := range marketKinds() {
		query := ""
		if kind == marketfeeds.ADP {
			query = "&PERIOD=ALL"
		}
		if kind == marketfeeds.Projections {
			query = "&L=14432&W=6"
		}
		recordSeasonBareArchive(t, a, kind+query, seasonFixture(t, "marketfeeds", kind+".json"), at)
		transport.failure = kind
		a.refreshMarketFeed(a.ctx, kind, 6)
		h := a.heldMarket()[kind]
		if len(h.value) != 1 || !h.fetchedAt.Equal(at) || h.refreshError == "" {
			t.Fatalf("%s fallback: %+v", kind, h)
		}
	}
	before := a.heldMarket()[marketfeeds.Projections]
	transport.failure = marketfeeds.Projections
	a.week.Number = 7
	a.refreshMarketFeed(a.ctx, marketfeeds.Projections, 7)
	if !a.heldMarket()[marketfeeds.Projections].fetchedAt.IsZero() {
		t.Fatal("wrong-week fallback accepted")
	}
	if before.value[0].Week != 6 {
		t.Fatal("held copy mutated")
	}
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
		if marketArchiveScope(scope.url, scope.kind, scope.week) != scope.want {
			t.Fatal(scope)
		}
	}
}

func TestMarketReportWireMatchesGo(t *testing.T) {
	held := realMarketHeld(t)
	for kind, feed := range held {
		feed.value[0].Players = feed.value[0].Players[:3]
		held[kind] = feed
	}
	wire := struct {
		Board   ReportBoard  `json:"board"`
		Facts   []MarketFact `json:"facts"`
		Missing []MarketFact `json:"missing"`
	}{buildMarketReports(held, map[string]string{"16162": "Bijan Robinson", "16161": "Jahmyr Gibbs", "13589": "Josh Allen"}, 6, time.UTC), marketFactsAt(held, time.UTC)["16162"], marketFactsAt(nil, time.UTC)[""]}
	body, err := json.MarshalIndent(wire, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join("frontend", "src", "app", "reports", "wire.testdata.json")
	body = append(body, '\n')
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(file, body, 0o644); err != nil { //nolint:gosec // a committed test fixture
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(body) {
		t.Fatal("market wire stale; TWR_WRITE_WIRE=1 go test -run TestMarketReportWireMatchesGo .")
	}
}

func TestMarketFeedCancelledDoesNotFetch(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	ctx, cancel := context.WithCancel(a.ctx)
	cancel()
	a.refreshMarketFeeds(ctx, true)
	if len(transport.requests) != 0 {
		t.Fatal(transport.requests)
	}
}

func TestPoolMarketColumnsMissingFeedDoesNotBlock(t *testing.T) {
	a, _ := poolApp(t)
	if err := a.TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	a.seasonMu.Lock()
	a.seasonMarket = nil
	a.seasonMu.Unlock()
	reading := a.TargetPool(RawPoolScope{Sort: "name"})
	if reading.Status != ReadingOK || len(reading.Value.Rows) != 3 {
		t.Fatal("missing feed blocked pool")
	}
	for _, row := range reading.Value.Rows {
		if len(row.Market) != 4 {
			t.Fatal("missing columns")
		}
		for _, column := range row.Market {
			if column.Text != "—" || column.Range != nil || !strings.Contains(column.Reason, "unavailable") {
				t.Fatal(column)
			}
		}
	}
	report, err := a.TargetMarketReports()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range report.Reports {
		if r.Status != "failed" || len(r.Rows) != 0 {
			t.Fatal(r)
		}
	}
	if marketWeekText(0) != "NFL week not known" {
		t.Fatal("unknown week printed as zero")
	}
}

func TestMarketNamesLabelTeamUnits(t *testing.T) {
	lookup, err := normalize.NewLookup([]players.RawPlayer{
		{ID: "16162", Name: "Robinson, Bijan", Position: "RB", Team: "ATL"},
		{ID: "0513", Name: "Broncos, Denver", Position: "Def", Team: "DEN"},
	})
	if err != nil {
		t.Fatal(err)
	}
	names := marketNames(realMarketHeld(t), lookup)
	if names["16162"] != "Robinson, Bijan · RB · ATL" || names["0513"] != "Broncos, Denver · team unit" {
		t.Fatal(names["16162"], names["0513"])
	}
}
