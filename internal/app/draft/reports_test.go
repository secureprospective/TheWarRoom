package draft

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/draft/drafttest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
)

func input(d drafttest.Draft, zone *time.Location) Input {
	return Input{Season: 2026, Results: Held[draftreports.Pick]{Value: d.Picks, Feed: d.Read},
		Future: Held[draftreports.FuturePick]{Value: d.Future, Feed: d.FutureRead}, Names: d.Names, Franchises: d.Franchises,
		Values: d.Values, ValueNote: d.ValueNote, Next: d.Next, Zone: zone}
}

func testBoard(t *testing.T) report.Board {
	t.Helper()
	return Build(input(drafttest.Held(t), time.UTC))
}

// built is the board of picks read at read, with names and no future picks or values.
func built(picks []draftreports.Pick, read time.Time, names map[string]string, zone *time.Location) report.Board {
	return Build(Input{Season: 2026, Results: Held[draftreports.Pick]{Value: picks, Feed: report.Feed{FetchedAt: read}},
		Names: names, Zone: zone})
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

func TestDraftReportsOrderGridAndOwnership(t *testing.T) {
	t.Parallel()
	b := testBoard(t)
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
	t.Parallel()
	b := testBoard(t)
	r := reportByID(t, b, "M-179")
	if !reflect.DeepEqual(r.Rows[0].Cells, []string{"One", "1", "under 1 min", "under 1 min", "Mostly pre-queued picks"}) || !reflect.DeepEqual(r.Rows[1].Cells, []string{"Two", "1", "under 1 min", "under 1 min", "Mostly pre-queued picks"}) {
		t.Fatal(r.Rows)
	}
	picks := []draftreports.Pick{drafttest.Pick(t, 1, 1, "0001", "0531", 100), drafttest.Pick(t, 1, 2, "0001", "0532", 100), drafttest.Pick(t, 1, 3, "0001", "0533", 160)}
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

func TestDraftEmptyUnavailableAndStale(t *testing.T) {
	t.Parallel()
	b := Build(Input{Season: 2026, ValueNote: "No model", Next: "Next unknown", Zone: time.UTC})
	for _, r := range b.Reports {
		if r.Status != "failed" || !strings.Contains(r.Note, "Report data unavailable") || r.How == "" || r.Source == "" {
			t.Fatal(r)
		}
	}
	at := time.Now()
	b = Build(Input{Season: 2026, Results: Held[draftreports.Pick]{Value: []draftreports.Pick{}, Feed: report.Feed{FetchedAt: at}},
		Future: Held[draftreports.FuturePick]{Value: []draftreports.FuturePick{}, Feed: report.Feed{FetchedAt: at}}, ValueNote: "No model", Next: "Next unknown", Zone: time.UTC})
	for _, r := range b.Reports {
		if r.Status != "empty" || !strings.Contains(r.Note, "no records") {
			t.Fatal(r)
		}
	}
}

func TestDraftCellLinesAndNameFallback(t *testing.T) {
	t.Parallel()
	picks := []draftreports.Pick{drafttest.Pick(t, 1, 1, "0001", "0531", 100), drafttest.Pick(t, 1, 2, "0001", "0532", 130)}
	b := built(picks, time.Unix(100, 0), map[string]string{"0531": "First"}, time.UTC)
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

func TestDraftDurationAndMedian(t *testing.T) {
	t.Parallel()
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
	r := reportByID(t, testBoard(t), "M-179")
	if r.Columns[3] != "Median" || r.Rows[0].Cells[4] != "Mostly pre-queued picks" {
		t.Fatal(r)
	}
}

func TestDraftStaleRefreshPendingReadAndSkippedPick(t *testing.T) {
	t.Parallel()
	b := Build(input(drafttest.Stale(t), time.UTC))
	results := reportByID(t, b, "M-063")
	if results.Status != "stale" || !strings.Contains(results.Note, "showing the last known data") ||
		!reflect.DeepEqual(results.Rows[0].Cells[:3], []string{"1.01", "Franchise 0002", "Pick skipped by MFL"}) || results.Rows[1].Cells[2] != "Not made" {
		t.Fatal(results)
	}
	if r := reportByID(t, b, "M-102"); r.Rows[0].Cells[4] != "From One" {
		t.Fatal(r.Rows)
	}
	if r := reportByID(t, b, "M-068"); r.Status != "stale" || r.Note != "Last read Oct 11, 2026, 12:00 PM; today's read pending" {
		t.Fatal(r)
	}
}
