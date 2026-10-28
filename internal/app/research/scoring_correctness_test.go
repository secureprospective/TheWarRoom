package research

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestScoreArchiveRejectsEnginePreviousSeason(t *testing.T) {
	at := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	for _, window := range []string{"YTD", "1"} {
		for _, explicit := range []string{"", "&YEAR=2026"} {
			t.Run(window+explicit, func(t *testing.T) {
				body := func(value float64) []byte {
					return []byte(fmt.Sprintf(`{"playerScores":{"playerScore":[{"id":"13589","week":%q,"score":%q}]}}`, window, fmt.Sprint(value)))
				}
				base := "/2026/export?TYPE=playerScores&L=14432&W=" + window
				bodies := func(_ context.Context, _ string, accept func(string, []byte, time.Time) bool) (bool, error) {
					if accept(base+"&YEAR=2025", body(456.10), at.Add(time.Minute)) {
						return true, nil
					}
					return accept(base+explicit, body(197.50), at), nil
				}
				cold := ReadArchivedFeed(t.Context(), bodies, 2026, "playerScores", window)
				if rows := Rows(cold); len(rows) != 1 || rows[0].Value != 197.50 || !cold.FetchedAt.Equal(at) {
					t.Errorf("cold hydration used another score season: %+v", cold)
				}
				rows, fresh, err := report.LiveOrArchive(t.Context(), t.Context(), 2026, report.ArchivedFeed[researchfeeds.Feed]{
					Export: "playerScores", Bodies: bodies,
					Fetch: func(context.Context) ([]researchfeeds.Feed, error) { return nil, errors.New("fixture offline") },
					Parse: func(body []byte) ([]researchfeeds.Feed, error) {
						feed, err := researchfeeds.Parse(body, "playerScores", window)
						return []researchfeeds.Feed{feed}, err
					},
				}, func(string) {})
				if err != nil || len(rows) != 1 || rows[0].Players[0].Value != 197.50 || fresh.FetchedAt != at.Format(time.RFC3339) {
					t.Fatalf("refresh fallback used another score season: %+v %+v %v", rows, fresh, err)
				}
			})
		}
	}
}

func TestPerformanceTotalsUseExactlyDisplayedWeeks(t *testing.T) {
	for _, tc := range []struct {
		name            string
		ytd             float64
		missing, failed bool
	}{
		{"initial", 134.80, false, false},
		{"engine previous season", 456.10, false, false},
		{"no YTD", 0, true, false},
		{"failed refresh", 456.10, false, true},
		{"partial weeks", 456.10, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := realResearch(t)
			id, err := playerid.New("13589")
			if err != nil {
				t.Fatal(err)
			}
			in.Held.Feeds[Key("playerScores", "YTD")] = Feed[[]researchfeeds.Feed]{Value: []researchfeeds.Feed{{Players: []researchfeeds.Entry{{ID: id, Value: tc.ytd}}}}, FetchedAt: in.Now.Add(time.Minute)}
			if tc.missing {
				delete(in.Held.Feeds, Key("playerScores", "YTD"))
			}
			if tc.failed {
				feed := in.Held.Feeds[Key("playerScores", "2")]
				feed.RefreshError = "fixture offline"
				in.Held.Feeds[Key("playerScores", "2")] = feed
			}
			if tc.missing && tc.failed {
				delete(in.Held.Feeds, Key("playerScores", "1"))
			}
			board := Build(in, Scope{Mine: "0002", Group: "QB"})
			for _, reportID := range []string{"M-037", "M-038", "M-040"} {
				r := reportByID(t, board.Board, reportID)
				if len(r.Rows) == 0 {
					t.Fatalf("%s lost weekly-only players", reportID)
				}
				for _, row := range r.Rows {
					var sum float64
					for _, cell := range row.Cells[4:] {
						if cell == "—" {
							continue
						}
						n, err := strconv.ParseFloat(cell, 64)
						if err != nil {
							t.Fatal(err)
						}
						sum += n
					}
					total, err := strconv.ParseFloat(row.Cells[3], 64)
					if err != nil {
						t.Fatal(err)
					}
					if math.Abs(sum-total) > 1e-9 {
						t.Errorf("%s %s: season %.2f != displayed weeks %.2f", reportID, row.Cells[0], total, sum)
					}
				}
				if tc.failed && r.Status != "stale" {
					t.Errorf("%s hid missing/stale weekly evidence: %s", reportID, r.Status)
				}
			}
		})
	}
}

func TestPerformanceTotalsIncludeCurrentWeekAtDisplayPrecision(t *testing.T) {
	in := realResearch(t)
	id, err := playerid.New("13589")
	if err != nil {
		t.Fatal(err)
	}
	for w, value := range []float64{1.004, -1.004, 0, 0, 2.004} {
		in.Held.Feeds[Key("playerScores", strconv.Itoa(w+1))] = Feed[[]researchfeeds.Feed]{Value: []researchfeeds.Feed{{Players: []researchfeeds.Entry{{ID: id, Value: value}}}}, FetchedAt: in.Now}
	}
	delete(in.Held.Feeds, Key("playerScores", "4"))
	r := reportByID(t, Build(in, Scope{Group: "QB"}).Board, "M-040")
	if len(r.Rows) != 1 {
		t.Fatal(r.Rows)
	}
	want := []string{"2.00", "1.00", "-1.00", "0.00", "—", "2.00"}
	for i, text := range want {
		if r.Rows[0].Cells[i+3] != text {
			t.Fatalf("season/weeks = %v, want %v", r.Rows[0].Cells[3:], want)
		}
	}
}
