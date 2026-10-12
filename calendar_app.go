package main

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func (a *App) mergeMFLCalendar(ctx context.Context, now time.Time, clock *snapshot.Sourced[leagueclock.Reading]) {
	held := archivedOnly(ctx, a, "calendar", false, ingestion.ParseCalendar)
	events := make([]leagueclock.Event, 0, len(held.value))
	for _, r := range held.value {
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
	for i := range added.Deadlines {
		d := &added.Deadlines[i]
		if d.At != nil && d.At.Before(now) {
			d.Urgency, d.Pinned, d.Promoted = leagueclock.U0, false, false
			d.Label += " · past calendar event"
		}
	}
	clock.Value.Deadlines = append(clock.Value.Deadlines, added.Deadlines...)
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
	if held.fetchedAt.IsZero() {
		clock.Provenance.Freshness.Note += "; MFL calendar unavailable; showing app clocks only"
	} else {
		clock.Provenance.Freshness.Note += "; MFL calendar held as of " + reportTime(held.fetchedAt, time.Local)
	}
}
