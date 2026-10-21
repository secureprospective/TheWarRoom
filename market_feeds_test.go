package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/market/markettest"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

func realMarketHeld(t *testing.T) map[string]heldSeasonFeed[[]marketfeeds.Feed] {
	t.Helper()
	held := map[string]heldSeasonFeed[[]marketfeeds.Feed]{}
	for kind, value := range markettest.Feeds(t) {
		held[kind] = heldSeasonFeed[[]marketfeeds.Feed]{Value: []marketfeeds.Feed{value}, FetchedAt: markettest.Read()}
	}
	return held
}

func TestMarketRefreshDailyWeekAndHeldReads(t *testing.T) {
	t.Parallel()
	a, transport, _, _ := seasonTestApp(t)
	a.core.Week = leagueweek.Week{Number: 6}
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
	a.core.Week.Number = 7
	if a.heldMarket()[marketfeeds.Projections].FetchedAt != (time.Time{}) {
		t.Fatal("old projection masquerades as this week")
	}
	transport.bodies[marketfeeds.Projections] = strings.Replace(transport.bodies[marketfeeds.Projections], `"week":"6"`, `"week":"7"`, 1)
	if err := a.TargetMarketReportsRefresh(false); err != nil {
		t.Fatal(err)
	}
	if len(transport.requests) != 5 || transport.requests[4] != marketfeeds.Projections || transport.weeks[4] != "7" {
		t.Fatal(transport.requests)
	}
	a.core.SeasonMu.Lock()
	h := a.core.SeasonMarket[marketfeeds.ADP]
	h.FetchedAt = time.Now().Add(-25 * time.Hour)
	a.core.SeasonMarket[marketfeeds.ADP] = h
	a.core.SeasonMu.Unlock()
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
	t.Parallel()
	a, transport, _, _ := seasonTestApp(t)
	a.core.Week.Number = 6
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
		a.refreshMarketFeed(a.core.Ctx, kind, 6)
		h := a.heldMarket()[kind]
		if len(h.Value) != 1 || !h.FetchedAt.Equal(at) || h.RefreshError == "" {
			t.Fatalf("%s fallback: %+v", kind, h)
		}
	}
	before := a.heldMarket()[marketfeeds.Projections]
	transport.failure = marketfeeds.Projections
	a.core.Week.Number = 7
	a.refreshMarketFeed(a.core.Ctx, marketfeeds.Projections, 7)
	if !a.heldMarket()[marketfeeds.Projections].FetchedAt.IsZero() {
		t.Fatal("wrong-week fallback accepted")
	}
	if before.Value[0].Week != 6 {
		t.Fatal("held copy mutated")
	}
}

func TestMarketReportWireMatchesGo(t *testing.T) {
	t.Parallel()
	held := realMarketHeld(t)
	for kind, feed := range held {
		feed.Value[0].Players = feed.Value[0].Players[:3]
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
	t.Parallel()
	a, transport, _, _ := seasonTestApp(t)
	ctx, cancel := context.WithCancel(a.core.Ctx)
	cancel()
	a.refreshMarketFeeds(ctx, true)
	if len(transport.requests) != 0 {
		t.Fatal(transport.requests)
	}
}

func TestPoolMarketColumnsMissingFeedDoesNotBlock(t *testing.T) {
	t.Parallel()
	a, _ := poolApp(t)
	if err := a.TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	a.core.SeasonMu.Lock()
	a.core.SeasonMarket = nil
	a.core.SeasonMu.Unlock()
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
