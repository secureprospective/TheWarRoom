package draft

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/draft/drafttest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
)

func TestDraftLandingCompletionAndClock(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	picks := []draftreports.Pick{drafttest.Pick(t, 1, 1, "0001", "0531", at.Unix()),
		drafttest.Pick(t, 1, 2, "0001", "0532", at.Add(time.Hour).Unix())}
	feed := Held[draftreports.Pick]{Value: picks, Feed: report.Feed{FetchedAt: at.Add(2 * time.Hour)}}
	r := draftStatus(2026, picks, nil, feed, "", time.UTC)
	if len(r.Rows) != 1 || !strings.Contains(r.Rows[0].Cells[1],
		"2026 rookie draft complete · 2 picks, May 1, 2026–May 1, 2026") {
		t.Fatal(r)
	}
	for _, row := range r.Rows {
		if row.Cells[0] == "On the clock" || row.Cells[0] == "Remaining" {
			t.Fatal(r)
		}
	}
	// Read the following January, the draft is still the 2026 draft.
	late := Held[draftreports.Pick]{Value: picks, Feed: report.Feed{FetchedAt: time.Date(2027, 1, 5, 0, 0, 0, 0, time.UTC)}}
	if r := draftStatus(2026, picks, nil, late, "", time.UTC); !strings.Contains(r.Rows[0].Cells[1], "2026 rookie draft") {
		t.Fatal("draft year from the feed's read time:", r.Rows)
	}
	picks[1].Player = nil
	if completedDraft(2026, picks, at.Add(2*time.Hour), time.UTC) != "" {
		t.Fatal("unmade completed")
	}
	picks[1].Skipped = true
	if completedDraft(2026, picks, at, time.UTC) != "" {
		t.Fatal("future pick completed")
	}
	picks[1].At = time.Time{}
	if completedDraft(2026, picks, at.Add(2*time.Hour), time.UTC) != "" {
		t.Fatal("missing time completed")
	}
	date := at.AddDate(1, 0, 0)
	windows := []leagueclock.Window{{Kind: leagueclock.Draft, At: &date}}
	if got := NextDate(windows, at, time.UTC); got != "Next draft: May 1, 2027, 10:00 AM" {
		t.Fatal(got)
	}
	if got := NextDate(nil, at, time.UTC); got != "Next draft: no date set by the commissioner." {
		t.Fatal(got)
	}
}

func TestDraftLandingEngineLabel(t *testing.T) {
	t.Parallel()
	r := reportByID(t, testBoard(t), "M-065")
	if r.Columns[2] != "Engine points per game" || !strings.Contains(r.Note, "Engine run: October 11, 2026") {
		t.Fatal(r)
	}
}
