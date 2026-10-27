package snapshot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// ClockSource is the league state the clock reads: the phase log and the commissioner calendar.
// The live binding and tools/fixtures both pass a *state.Store, so they share this one builder.
type ClockSource interface {
	CurrentPhase(context.Context) (domain.Phase, error)
	CalendarEvents(context.Context) ([]state.CalendarEvent, error)
	TradeDeadline(context.Context) (time.Time, error)
}

// BuildClock reads the local source and supplied lineup lock, grading deadlines at the supplied instant.
func BuildClock(ctx context.Context, at time.Time, season int, src ClockSource,
	lineup *leagueclock.LineupLock, rules leagueclock.WindowInputs, p Provenance) (Sourced[leagueclock.Reading], error) {
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
	deadline, err := src.TradeDeadline(ctx)
	if err != nil {
		return Sourced[leagueclock.Reading]{}, fmt.Errorf("snapshot: trade deadline: %w", err)
	}
	moved, ordinary, err := splitWindowDates(season, rules.LastRegular, deadline, events)
	if err != nil {
		return Sourced[leagueclock.Reading]{}, err
	}
	rules.Moved = moved
	in := leagueclock.Inputs{Season: season, Phase: phase, Lineup: lineup, Rules: rules, Events: ordinary}
	return Sourced[leagueclock.Reading]{Value: leagueclock.Clock(at, in), Provenance: p}, nil
}

// splitWindowDates separates the commissioner's window dates from ordinary calendar deadlines. A
// window head moves its window only in its own season and while PLANNED; it is never also a deadline.
// The trade directive is not stored per season, but an NFL trade deadline always falls in its
// season's own year, so a date from another year is last season's and the rulebook applies again.
func splitWindowDates(season, lastRegular int, trade time.Time, events []state.CalendarEvent) (
	map[leagueclock.WindowKind]time.Time, []leagueclock.Event, error) {
	moved := map[leagueclock.WindowKind]time.Time{}
	if !trade.IsZero() && trade.UTC().Year() == season {
		moved[leagueclock.TradeDeadline] = trade
	}
	known := map[leagueclock.WindowKind]bool{}
	for _, rule := range leagueclock.Rules(lastRegular) {
		known[rule.Kind] = rule.Kind != leagueclock.TradeDeadline
	}
	prefix := fmt.Sprintf("%s%d:", leagueclock.WindowEventPrefix, season)
	ordinary := make([]leagueclock.Event, 0, len(events))
	for _, e := range events {
		ev, err := clockEvent(e)
		if err != nil {
			return nil, nil, err
		}
		if !strings.HasPrefix(e.EventID, leagueclock.WindowEventPrefix) {
			ordinary = append(ordinary, ev)
			continue
		}
		kind := leagueclock.WindowKind(strings.TrimPrefix(e.EventID, prefix))
		if strings.HasPrefix(e.EventID, prefix) && e.Status == state.CalStatusPlanned && known[kind] {
			moved[kind] = ev.At
		}
	}
	return moved, ordinary, nil
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
