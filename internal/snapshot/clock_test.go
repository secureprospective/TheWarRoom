package snapshot

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type heldClock struct {
	phase  domain.Phase
	events []state.CalendarEvent
	err    error
}

func (h heldClock) CurrentPhase(context.Context) (domain.Phase, error) { return h.phase, h.err }
func (h heldClock) CalendarEvents(context.Context) ([]state.CalendarEvent, error) {
	return h.events, nil
}

func TestClockBuilderPreservesHeldFacts(t *testing.T) {
	at := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	src := heldClock{phase: domain.PhaseRegularSeason, events: []state.CalendarEvent{{
		EventID: "x", Kind: "cut", ScheduledAt: "2026-10-06T08:00:00-04:00", Status: state.CalStatusPlanned,
	}}}
	p := Provenance{Source: "test", Kind: "fixture"}
	got, err := BuildClock(context.Background(), at, 2026, src, p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Value.Season != 2026 || got.Value.Phase != domain.PhaseRegularSeason {
		t.Fatal(got)
	}
	d := got.Value.Deadlines
	if len(d) != 1 || d[0].At.Location() != time.UTC || !d[0].At.Equal(at) {
		t.Fatal(got)
	}
	if !reflect.DeepEqual(got.Provenance, p) {
		t.Fatal("provenance changed")
	}
}

func TestClockBuilderRejectsBadHeldData(t *testing.T) {
	ok := "2026-10-06T12:00:00Z"
	for name, src := range map[string]heldClock{
		"phase error":   {err: errors.New("closed")},
		"invalid phase": {phase: "NOPE"},
		"bad time": {phase: domain.PhaseOffseason, events: []state.CalendarEvent{
			{EventID: "x", Kind: "cut", ScheduledAt: "bad", Status: state.CalStatusPlanned}}},
		"bad status": {phase: domain.PhaseOffseason, events: []state.CalendarEvent{
			{EventID: "x", Kind: "cut", ScheduledAt: ok, Status: "bad"}}},
		"no identity": {phase: domain.PhaseOffseason, events: []state.CalendarEvent{
			{Kind: "cut", ScheduledAt: ok, Status: state.CalStatusPlanned}}},
	} {
		if _, err := BuildClock(context.Background(), time.Now(), 2026, src, Provenance{}); err == nil {
			t.Fatal("accepted", name)
		}
	}
}
