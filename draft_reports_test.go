package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func draftFixturePick(t *testing.T, round, slot int, owner, id string, at int64) draftreports.Pick {
	t.Helper()
	p, err := playerid.New(id)
	if err != nil {
		t.Fatal(err)
	}
	return draftreports.Pick{Round: round, Slot: slot, Franchise: owner, Player: &p, At: time.Unix(at, 0).UTC()}
}
func reportByID(t *testing.T, board ReportBoard, id string) Report {
	t.Helper()
	for _, r := range board.Reports {
		if r.ID == id {
			return r
		}
	}
	t.Fatal(id)
	return Report{}
}
func testDraftBoard(t *testing.T) ReportBoard {
	t.Helper()
	at := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	picks := []draftreports.Pick{draftFixturePick(t, 2, 1, "0002", "0531", 160), draftFixturePick(t, 1, 2, "0001", "0532", 130), draftFixturePick(t, 1, 1, "0002", "0533", 100)}
	return buildDraftReports(heldSeasonFeed[[]draftreports.Pick]{value: picks, fetchedAt: at},
		heldSeasonFeed[[]draftreports.FuturePick]{value: []draftreports.FuturePick{{Year: 2027, Round: 2, Original: "0001", Owner: "0002"}}, fetchedAt: at},
		map[string]string{"0531": "Third", "0532": "Second", "0533": "First"}, map[string]string{"0001": "One", "0002": "Two"},
		map[string]float64{"0531": 50, "0532": 10, "0533": 30}, "Player values: the app's model, run Oct 11, 2026, 12:00 PM.", "Next draft: date unknown.", time.UTC)
}

func TestDraftReportsOrderGridAndOwnership(t *testing.T) {
	b := testDraftBoard(t)
	rows := reportByID(t, b, "M-063").Rows
	if !reflect.DeepEqual(rows[0].Cells[:3], []string{"1.01", "Two", "First"}) || rows[1].Cells[0] != "1.02" || rows[2].Cells[0] != "2.01" {
		t.Fatal(rows)
	}
	grid := reportByID(t, b, "M-064")
	if !reflect.DeepEqual(grid.Columns, []string{"Round", "One", "Two"}) ||
		!reflect.DeepEqual(grid.Rows[0].Cells, []string{"1", "1.02 · Second", "1.01 · First"}) ||
		!reflect.DeepEqual(grid.Rows[1].Cells, []string{"2", "No recorded pick", "2.01 · Third"}) {
		t.Fatal(grid)
	}
	future := reportByID(t, b, "M-068")
	if !reflect.DeepEqual(future.Rows[0].Cells, []string{"2027", "2", "One", "Two"}) {
		t.Fatal(future)
	}
	detail := reportByID(t, b, "M-102")
	if detail.Rows[0].Cells[4] != "MFL lists no trade details" || !strings.Contains(detail.Rows[0].Cells[3], "Jan 1, 1970") {
		t.Fatal(detail)
	}
}

func TestDraftAverageTimeKnownTimestamps(t *testing.T) {
	b := testDraftBoard(t)
	r := reportByID(t, b, "M-179")
	if !reflect.DeepEqual(r.Rows[0].Cells, []string{"One", "1", "under 1 min", "under 1 min", "Mostly pre-queued picks"}) || !reflect.DeepEqual(r.Rows[1].Cells, []string{"Two", "1", "under 1 min", "under 1 min", "Mostly pre-queued picks"}) {
		t.Fatal(r.Rows)
	}
	picks := []draftreports.Pick{draftFixturePick(t, 1, 1, "0001", "0531", 100), draftFixturePick(t, 1, 2, "0001", "0532", 100), draftFixturePick(t, 1, 3, "0001", "0533", 160)}
	elapsed := draftElapsed(picks)
	if !reflect.DeepEqual(elapsed["0001"], []time.Duration{0, time.Minute}) {
		t.Fatal(elapsed)
	}
	picks[1].At = time.Time{}
	if len(draftElapsed(picks)) != 0 {
		t.Fatal("missing time counted")
	}
	picks[1].At = time.Unix(200, 0)
	if got := draftElapsed(picks)["0001"]; len(got) != 1 || got[0] != 100*time.Second {
		t.Fatal(got)
	}
}

func TestDraftValueDistanceAndMissingValues(t *testing.T) {
	r := reportByID(t, testDraftBoard(t), "M-065")
	if r.Rows[0].Range == nil || r.Rows[0].Range.Value != 1.5 || r.Rows[1].Range.Value != 0.5 || r.Rows[2].Range != nil {
		t.Fatal(r.Rows)
	}
	if r.Rows[0].Cells[4] != "+10.00 points/game above the round 1 median" || r.Rows[1].Cells[4] != "10.00 points/game below the round 1 median" || !strings.Contains(r.How, "median") || !strings.Contains(r.How, "dynasty value") {
		t.Fatal(r)
	}
}

func TestDraftEmptyUnavailableAndStale(t *testing.T) {
	b := buildDraftReports(heldSeasonFeed[[]draftreports.Pick]{}, heldSeasonFeed[[]draftreports.FuturePick]{}, nil, nil, nil, "No model", "Next unknown", time.UTC)
	for _, r := range b.Reports {
		if r.Status != "failed" || !strings.Contains(r.Note, "Report data unavailable") || r.How == "" || r.Source == "" {
			t.Fatal(r)
		}
	}
	at := time.Now()
	b = buildDraftReports(heldSeasonFeed[[]draftreports.Pick]{value: []draftreports.Pick{}, fetchedAt: at}, heldSeasonFeed[[]draftreports.FuturePick]{value: []draftreports.FuturePick{}, fetchedAt: at}, nil, nil, nil, "No model", "Next unknown", time.UTC)
	for _, r := range b.Reports {
		if r.Status != "empty" || !strings.Contains(r.Note, "no records") {
			t.Fatal(r)
		}
	}
	a, transport, _, _ := seasonTestApp(t)
	a.refreshDraftReports(a.ctx)
	if len(a.seasonDraftResults.value) != 160 || len(a.seasonFuturePicks.value) != 320 {
		t.Fatal("feeds not filled")
	}
	transport.failure = "draftResults"
	a.refreshDraftReports(a.ctx)
	if len(a.seasonDraftResults.value) != 160 || a.seasonDraftResults.refreshError == "" || a.seasonFuturePicks.refreshError != "" {
		t.Fatal("last good copy not retained independently")
	}
}

func TestDraftWireFixtureMatchesGo(t *testing.T) {
	path := filepath.Join("frontend", "src", "app", "draftreports", "wire.testdata.json")
	body, err := json.MarshalIndent(testDraftBoard(t), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(held) != string(body) {
		t.Fatal("draft wire fixture drift; regenerate with TWR_WRITE_WIRE=1")
	}
}

func TestDraftHeldReadingRefreshCacheAndMFLLink(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	if err := a.TargetOpenMFLDraftRoom(); err == nil {
		t.Fatal("opened undiscovered host")
	}
	if _, err := a.TargetDraftReports(); err != nil || len(transport.requests) != 0 {
		t.Fatal("reading fetched", err)
	}
	if err := a.TargetDraftReportsRefresh(false); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(transport.requests, []string{"draftResults", "futureDraftPicks"}) {
		t.Fatal(transport.requests)
	}
	board, err := a.TargetDraftReports()
	if err != nil || len(reportByID(t, board, "M-063").Rows) != 160 || !strings.Contains(board.MFLURL, "O=17") {
		t.Fatal(board, err)
	}
	if err := a.TargetDraftReportsRefresh(false); err != nil || len(transport.requests) != 2 {
		t.Fatal("fresh copy refetched", err)
	}
	opened := ""
	a.openURL = func(_ context.Context, url string) { opened = url }
	if err := a.TargetOpenMFLDraftRoom(); err != nil || opened != board.MFLURL || len(transport.requests) != 2 {
		t.Fatal("link submitted or fetched", opened, err)
	}
	if err := a.TargetDraftReportsRefresh(true); err != nil || len(transport.requests) != 4 {
		t.Fatal("force refresh skipped", err)
	}
}

func TestDraftArchiveFallbackKeepsOriginalAsOf(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	at := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	recordSeasonArchive(t, a, "draftResults", "", transport.bodies["draftResults"], at)
	transport.failure = "draftResults"
	a.refreshDraftReports(a.ctx)
	board, err := a.TargetDraftReports()
	if err != nil {
		t.Fatal(err)
	}
	results := reportByID(t, board, "M-063")
	if results.Status != "stale" || len(results.Rows) != 160 || !strings.Contains(results.AsOf, pickTime(draftreports.Pick{At: at}, time.Local)) ||
		!strings.Contains(results.Note, "last known data") || reportByID(t, board, "M-068").Status != "ok" {
		t.Fatal(results)
	}
}

func TestDraftExplicitUnmadeStatusAndLocalTime(t *testing.T) {
	p := draftFixturePick(t, 1, 1, "0001", "0531", 100)
	pending := draftreports.Pick{Round: 1, Slot: 2, Franchise: "0002"}
	feed := heldSeasonFeed[[]draftreports.Pick]{value: []draftreports.Pick{p, pending}, fetchedAt: time.Unix(100, 0)}
	r := draftStatus(feed.value, nil, feed, "Method", time.UTC)
	if r.Rows[0].Cells[1] != "1" || !strings.Contains(r.Rows[1].Cells[1], "1.02") || !strings.Contains(r.Rows[2].Cells[1], "1 unmade") {
		t.Fatal(r)
	}
	zone := time.FixedZone("LOCAL", 2*60*60)
	if got := pickTime(p, zone); got != "Jan 1, 1970, 2:01 AM" {
		t.Fatal(got)
	}
}

func TestDraftCellLinesAndNameFallback(t *testing.T) {
	picks := []draftreports.Pick{draftFixturePick(t, 1, 1, "0001", "0531", 100), draftFixturePick(t, 1, 2, "0001", "0532", 130)}
	feed := heldSeasonFeed[[]draftreports.Pick]{value: picks, fetchedAt: time.Unix(100, 0)}
	b := buildDraftReports(feed, heldSeasonFeed[[]draftreports.FuturePick]{}, map[string]string{"0531": "First"}, nil, nil, "", "", time.UTC)
	want := "1.01 · First\n1.02 · Player 0532 (name not held)"
	if got := reportByID(t, b, "M-064").Rows[0].Cells[1]; got != want {
		t.Fatal(got)
	}
	if got := reportByID(t, b, "M-066").Rows[0].Cells[2]; got != want {
		t.Fatal(got)
	}
	if got := reportByID(t, b, "M-063").Rows[0].Cells[2]; got != "First" {
		t.Fatal(got)
	}
}

func TestDraftAllDatesUseAppStyle(t *testing.T) {
	at := time.Date(2026, 10, 11, 12, 0, 51, 0, time.UTC)
	zone := time.FixedZone("LOCAL", -5*3600)
	p := draftFixturePick(t, 1, 1, "0001", "0531", at.Unix())
	b := buildDraftReports(heldSeasonFeed[[]draftreports.Pick]{value: []draftreports.Pick{p}, fetchedAt: at}, heldSeasonFeed[[]draftreports.FuturePick]{fetchedAt: at}, nil, nil, nil, "", "", zone)
	for _, r := range b.Reports {
		if r.AsOf != "As of: Oct 11, 2026, 7:00 AM" {
			t.Fatal(r.AsOf)
		}
	}
	if got := pickTime(p, zone); got != "Oct 11, 2026, 7:00 AM" {
		t.Fatal(got)
	}
}

func TestDraftValueNotePlainWords(t *testing.T) {
	a, _, _, _ := seasonTestApp(t)
	at := time.Date(2026, 10, 5, 13, 44, 0, 0, time.Local)
	run := history.Run{ID: 3, AsOf: at, Engine: "secret-version"}
	want := "Player values: the app's model, run Oct 5, 2026, 1:44 PM."
	if got := a.draftValueNote(run); got != want {
		t.Fatal(got)
	}
	run.MissingMeasures = []string{"context.birth_date"}
	got := a.draftValueNote(run)
	if !strings.Contains(got, "Missing player information:") || strings.Contains(got, "context.birth_date") || strings.Contains(got, "secret-version") || !strings.Contains(got, "Date of birth") || got == want {
		t.Fatal(got)
	}
}

func TestDraftReportLanguage(t *testing.T) {
	b := testDraftBoard(t)
	for _, r := range b.Reports {
		for _, bad := range []string{"held", "export", "sentinel", "UTC instants", "consumed", "draftResults", "futureDraftPick", "originalPickFor", "DynastyPPG"} {
			if strings.Contains(strings.ToLower(r.How+" "+r.Note+" "+r.Title+" "+r.Source), strings.ToLower(bad)) {
				t.Fatalf("%s contains %s", r.ID, bad)
			}
		}
		if r.Source != "Source: MFL" {
			t.Fatal(r.Source)
		}
	}
	r := reportByID(t, b, "M-176")
	if r.Columns[1] != "What MFL shows" || r.Rows[1].Cells[1] != "MFL does not report who is on the clock." || !strings.Contains(r.Note, "All 3 picks MFL lists are made.") {
		t.Fatal(r)
	}
}

func TestDraftDurationAndMedian(t *testing.T) {
	for _, tc := range []struct {
		d    time.Duration
		want string
	}{{3*time.Hour + 49*time.Minute + 53*time.Second, "3 h 50 min"}, {27*time.Minute + 14*time.Second, "27 min"}, {59 * time.Second, "under 1 min"}, {60 * time.Second, "1 min"}} {
		if got := draftDuration(tc.d); got != tc.want {
			t.Fatal(got, tc.want)
		}
	}
	for _, tc := range []struct {
		ds                []time.Duration
		avg, median, note string
	}{
		{[]time.Duration{0, time.Second, 27 * time.Minute}, "9 min", "under 1 min", "Mostly pre-queued picks"},
		{[]time.Duration{0, 2 * time.Minute}, "1 min", "1 min", ""},
		{nil, "No measured interval", "No measured interval", ""},
	} {
		avg, median, note := draftPace(tc.ds)
		if avg != tc.avg || median != tc.median || note != tc.note {
			t.Fatal(avg, median, note)
		}
	}
	r := reportByID(t, testDraftBoard(t), "M-179")
	if r.Columns[3] != "Median" || r.Rows[0].Cells[4] != "Mostly pre-queued picks" {
		t.Fatal(r)
	}
}

func TestDraftLiveRecentPicksAndStatus(t *testing.T) {
	picks := []draftreports.Pick{}
	for i := 1; i <= 20; i++ {
		picks = append(picks, draftFixturePick(t, 1, i, "0001", "0531", int64(i)*100))
	}
	build := func() ReportBoard {
		return buildDraftReports(heldSeasonFeed[[]draftreports.Pick]{value: picks, fetchedAt: time.Unix(100, 0)}, heldSeasonFeed[[]draftreports.FuturePick]{}, nil, nil, nil, "", "", time.UTC)
	}
	b := build()
	r := reportByID(t, b, "M-183")
	if len(r.Rows) != 12 || r.Rows[0].Cells[0] != "1.09" || r.Rows[11].Cells[0] != "1.20" || r.Link == nil || *r.Link != (ReportLink{Label: "See every pick", Endpoint: "M-063"}) || !reflect.DeepEqual(r.Summary, reportByID(t, b, "M-176").Rows) || len(reportByID(t, b, "M-063").Rows) != 20 {
		t.Fatal(r)
	}
	picks = append(picks, draftreports.Pick{Round: 1, Slot: 21, Franchise: "0001"})
	if r = reportByID(t, build(), "M-183"); len(r.Rows) != 21 {
		t.Fatal(r)
	}
}
