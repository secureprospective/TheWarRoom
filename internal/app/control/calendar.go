package control

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func MergeMFLCalendar(now time.Time, clock *snapshot.Sourced[leagueclock.Reading], held report.HeldFeed[[]ingestion.RawCalendarEvent]) {
	events := make([]leagueclock.Event, 0, len(held.Value))
	for _, r := range held.Value {
		var at time.Time
		if r.Start != "" {
			seconds, _ := strconv.ParseInt(r.Start, 10, 64)
			at = time.Unix(seconds, 0).UTC()
		}
		label := r.Title
		if label == "" {
			label = strings.ToLower(strings.ReplaceAll(r.Type, "_", " "))
		}
		events = append(events, leagueclock.Event{ID: "mfl-calendar:" + r.ID, Kind: "MFL · " + label, Status: "PLANNED", At: at})
	}
	added := leagueclock.Clock(now, leagueclock.Inputs{Events: events})
	// A calendar event that has passed is history, not a deadline: the clock lists what is coming.
	upcoming := slices.DeleteFunc(added.Deadlines, func(d leagueclock.Deadline) bool { return d.At != nil && d.At.Before(now) })
	clock.Value.Deadlines = append(clock.Value.Deadlines, upcoming...)
	slices.SortFunc(clock.Value.Deadlines, func(a, b leagueclock.Deadline) int {
		if a.Urgency != b.Urgency {
			if a.Urgency > b.Urgency {
				return -1
			}
			return 1
		}
		if a.At != nil && b.At != nil && !a.At.Equal(*b.At) {
			return a.At.Compare(*b.At)
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	clock.Provenance.Source += "+held-mfl-calendar"
	if held.FetchedAt.IsZero() {
		clock.Provenance.Freshness.Note += "; MFL calendar unavailable; showing app clocks only"
	} else {
		clock.Provenance.Freshness.Note += "; MFL calendar " + report.FeedTime(held.FeedState(), time.Local)
	}
}
