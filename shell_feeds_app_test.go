package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

func TestDailyShellArchiveReadStatus(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, time.Local)
	for _, export := range []string{"calendar", "tradeBait"} {
		for _, test := range []struct {
			name    string
			at      time.Time
			loss    string
			status  string
			pending bool
		}{
			{"today", now.Add(-time.Hour), "", "ok", false},
			{"yesterday", now.AddDate(0, 0, -1), "", "stale", true},
			{"failed", now.Add(-time.Hour), "offline", "stale", false},
		} {
			t.Run(export+"/"+test.name, func(t *testing.T) {
				t.Parallel()
				a, transport, _, _ := seasonTestApp(t)
				a.core.ClockNow = func() time.Time { return now }
				body := `{"calendar":{"event":{"id":"one","type":"TRADE","title":"Event"}}}`
				if export == "tradeBait" {
					body = `{"tradeBaits":{"tradeBait":{"franchise_id":"0001","inExchangeFor":"Bait"}}}`
				}
				recordSeasonArchive(t, a, export, "", body, test.at)
				var at time.Time
				var note, status, source, provenance string
				if export == "calendar" {
					feed := readShellFeed(a.core.Ctx, a, &a.core.SeasonCalendar, export, ingestion.ParseCalendar)
					if test.loss != "" {
						holdSeasonFeed(&feed, nil, Freshness{}, errors.New(test.loss))
					}
					r := reportShell("C-04", "Calendar", "", feed, time.Local)
					at, note, status = feed.FetchedAt, r.Note, r.Status
					source, provenance = shellFeedTime(feed, time.Local), feed.Provenance("MFL", export).Freshness.Note
				} else {
					feed := readShellFeed(a.core.Ctx, a, &a.core.SeasonTradeBait, export, ingestion.ParseTradeBait)
					if test.loss != "" {
						holdSeasonFeed(&feed, nil, Freshness{}, errors.New(test.loss))
					}
					r := reportShell("M-116", "Trade bait", "", feed, time.Local)
					at, note, status = feed.FetchedAt, r.Note, r.Status
					source, provenance = shellFeedTime(feed, time.Local), feed.Provenance("MFL", export).Freshness.Note
				}
				if !at.Equal(test.at) || status != test.status {
					t.Fatal(at, status, note)
				}
				if strings.Contains(note, "Refresh failed") != (test.loss != "") {
					t.Fatal(note)
				}
				for _, text := range []string{note, source, provenance} {
					if strings.Contains(text, "today's read pending") != test.pending {
						t.Fatal(text)
					}
				}
				if test.pending && (!strings.Contains(note, reportTime(test.at, time.Local)) ||
					!strings.HasPrefix(source, "archived copy")) {
					t.Fatal(note)
				}
				if test.name == "today" && (note != "" || strings.Contains(provenance, "pending")) {
					t.Fatal(note, provenance)
				}
				if len(transport.requests) != 0 {
					t.Fatal("screen read fetched", transport.requests)
				}
			})
		}
	}
}
