package snapshot

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// ClockSource is the league state the clock reads: the phase log and the commissioner calendar.
// The live binding and cmd/fixtures both pass a *state.Store, so they share this one builder.
type ClockSource interface {
	CurrentPhase(context.Context) (domain.Phase, error)
	CalendarEvents(context.Context) ([]state.CalendarEvent, error)
}

// BuildClock reads the source and grades its deadlines at the supplied instant.
func BuildClock(ctx context.Context, at time.Time, season int, src ClockSource,
	p Provenance) (Sourced[leagueclock.Reading], error) {
	phase, err := src.CurrentPhase(ctx)
	if err != nil {
		return Sourced[leagueclock.Reading]{}, fmt.Errorf("snapshot: clock phase: %w", err)
	}
	if !phase.Valid() {
		return Sourced[leagueclock.Reading]{}, fmt.Errorf("snapshot: clock phase %q", phase)
	}
	events, err := src.CalendarEvents(ctx)
	if err != nil {
		return Sourced[leagueclock.Reading]{}, fmt.Errorf("snapshot: clock calendar: %w", err)
	}
	in := leagueclock.Inputs{Season: season, Phase: phase, Events: make([]leagueclock.Event, 0, len(events))}
	for _, e := range events {
		ev, err := clockEvent(e)
		if err != nil {
			return Sourced[leagueclock.Reading]{}, err
		}
		in.Events = append(in.Events, ev)
	}
	return Sourced[leagueclock.Reading]{Value: leagueclock.Clock(at, in), Provenance: p}, nil
}

func clockEvent(e state.CalendarEvent) (leagueclock.Event, error) {
	if e.EventID == "" || e.Kind == "" {
		return leagueclock.Event{}, fmt.Errorf("snapshot: clock calendar identity required")
	}
	switch e.Status {
	case state.CalStatusPlanned, state.CalStatusFired, state.CalStatusCancelled:
	default:
		return leagueclock.Event{}, fmt.Errorf("snapshot: clock calendar status %q", e.Status)
	}
	at, err := time.Parse(time.RFC3339, e.ScheduledAt)
	if err != nil {
		return leagueclock.Event{}, fmt.Errorf("snapshot: clock calendar %s: %w", e.EventID, err)
	}
	return leagueclock.Event{ID: e.EventID, Kind: e.Kind, Status: e.Status, At: at.UTC()}, nil
}
