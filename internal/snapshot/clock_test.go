package snapshot

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type heldClock struct {
	phase  domain.Phase
	events []state.CalendarEvent
	err    error
	trade  time.Time
}

func (h heldClock) CurrentPhase(context.Context) (domain.Phase, error) { return h.phase, h.err }
func (h heldClock) CalendarEvents(context.Context) ([]state.CalendarEvent, error) {
	return h.events, nil
}

func (h heldClock) TradeDeadline(context.Context) (time.Time, error) { return h.trade, nil }

func TestClockBuilderPreservesHeldFacts(t *testing.T) {
	at := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	src := heldClock{phase: domain.PhaseRegularSeason, events: []state.CalendarEvent{{
		EventID: "x", Kind: "cut", ScheduledAt: "2026-10-06T08:00:00-04:00", Status: state.CalStatusPlanned,
	}}}
	p := Provenance{Source: "test", Kind: "fixture"}
	got, err := BuildClock(context.Background(), at, 2026, src, nil, leagueclock.WindowInputs{}, p)
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
		if _, err := BuildClock(context.Background(), time.Now(), 2026, src, nil, leagueclock.WindowInputs{}, Provenance{}); err == nil {
			t.Fatal("accepted", name)
		}
	}
}

func TestClockBuilderWindowOverrides(t *testing.T) {
	at := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	trade, cut := at.Add(time.Hour), at.Add(2*time.Hour)
	src := heldClock{phase: domain.PhaseRegularSeason, trade: trade, events: []state.CalendarEvent{
		{EventID: "window:2026:cut_day", Kind: "WINDOW_DATE", ScheduledAt: cut.Format(time.RFC3339), Status: state.CalStatusPlanned},
		{EventID: "window:2025:draft", Kind: "WINDOW_DATE", ScheduledAt: cut.Format(time.RFC3339), Status: state.CalStatusPlanned},
		{EventID: "window:2026:rfa_tender", Kind: "WINDOW_DATE", ScheduledAt: cut.Format(time.RFC3339), Status: state.CalStatusCancelled},
	}}
	got, err := BuildClock(context.Background(), at, 2026, src, nil, leagueclock.WindowInputs{}, Provenance{})
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range got.Value.Windows {
		switch w.Kind {
		case leagueclock.TradeDeadline:
			if !w.Moved || w.At == nil || !w.At.Equal(trade) {
				t.Fatal("trade directive lost", w)
			}
		case leagueclock.CutDay:
			if !w.Moved || w.At == nil || !w.At.Equal(cut) {
				t.Fatal("calendar move lost", w)
			}
		case leagueclock.Draft, leagueclock.RFATender, leagueclock.ContractOptions, leagueclock.UFABidding,
			leagueclock.Resign, leagueclock.PlayoffBidsOnly, leagueclock.MultiYearBidsClose:
			if w.Moved || w.At != nil {
				t.Fatal("wrong season or cancelled move", w)
			}
		}
	}
	if len(got.Value.Deadlines) != 2 {
		t.Fatal("window also an ordinary deadline", got.Value.Deadlines)
	}
	for _, d := range got.Value.Deadlines {
		if d.ID != "window:trade_deadline" && d.ID != "window:cut_day" {
			t.Fatal(d)
		}
	}
}

func TestClockBuilderIgnoresLastSeasonsTradeDirective(t *testing.T) {
	at := time.Date(2027, 9, 20, 12, 0, 0, 0, time.UTC)
	src := heldClock{phase: domain.PhaseRegularSeason, trade: time.Date(2026, 11, 5, 0, 15, 0, 0, time.UTC)}
	got, err := BuildClock(context.Background(), at, 2027, src, nil, leagueclock.WindowInputs{}, Provenance{})
	if err != nil {
		t.Fatal(err)
	}
	if w := got.Value.Windows[0]; w.Kind != leagueclock.TradeDeadline || w.Moved || w.At != nil {
		t.Fatal("a 2026 commissioner date must not move the 2027 trade deadline", w)
	}
}
