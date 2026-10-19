package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestTargetSetWindowRoundTrip(t *testing.T) {
	t.Parallel()
	a, transport, _ := weekTestApp(t, "17")
	emitted := 0
	a.clockChanged = func(context.Context) { emitted++ }
	date := "2026-11-01T18:00:00Z"
	for _, kind := range []string{"trade_deadline", "cut_day"} {
		if err := a.TargetSetWindow(kind, ""); err == nil || !strings.Contains(err.Error(), "nothing to reset") {
			t.Fatal("reset wording", err)
		}
		for _, at := range []string{date, "2026-11-02T18:00:00Z", ""} {
			if err := a.TargetSetWindow(kind, at); err != nil {
				t.Fatal(err)
			}
			clock, err := a.TargetClock()
			if err != nil {
				t.Fatal(err)
			}
			for _, w := range clock.Value.Windows {
				if string(w.Kind) != kind {
					continue
				}
				if w.Moved != (at != "") {
					t.Fatal(w)
				}
				if at != "" && (w.At == nil || w.At.Format(time.RFC3339) != at) {
					t.Fatal(w)
				}
				if at == "" && w.At != nil {
					t.Fatal("reset did not restore base", w)
				}
			}
		}
		if err := a.TargetSetWindow(kind, ""); err == nil || !strings.Contains(err.Error(), "nothing to reset") {
			t.Fatal("repeated reset wording", err)
		}
	}
	heads, err := a.whatif.CalendarEvents(a.ctx)
	if err != nil || len(heads) != 1 {
		t.Fatal(heads, err)
	}
	if heads[0].EventID != "window:2026:cut_day" || heads[0].Kind != "WINDOW_DATE" ||
		heads[0].Status != state.CalStatusCancelled || heads[0].Payload != `{"window":"cut_day"}` ||
		heads[0].Note != "commissioner set cut_day" {
		t.Fatal(heads)
	}
	if emitted != 6 || len(transport.requests) != 0 {
		t.Fatal("events or network", emitted, transport.requests)
	}
	for _, raw := range []RawWindowDate{{Kind: "nope", At: date}, {Kind: "draft", At: "tomorrow"}} {
		if err := a.TargetSetWindow(raw.Kind, raw.At); err == nil {
			t.Fatal("accepted", raw)
		}
	}
}

func TestTradeDeadlineReasonsUsesEffectiveWindow(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	for _, tc := range []struct {
		name   string
		at     *time.Time
		status leagueclock.WindowStatus
		want   string
	}{
		{"passed", &past, leagueclock.WindowDated, "Trade deadline passed "},
		{"equal", &now, leagueclock.WindowDated, "Trade deadline passed "},
		{"future", &future, leagueclock.WindowDated, ""},
		{"undated", nil, leagueclock.WindowUnknown, "Trade deadline cannot be checked: "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := leagueclock.Window{Kind: leagueclock.TradeDeadline, Status: tc.status, At: tc.at, Reason: "rule basis"}
			got := tradeDeadlineReasons(now, []leagueclock.Window{w})
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatal(got)
				}
				return
			}
			if len(got) != 1 || !strings.HasPrefix(got[0], tc.want) || !strings.Contains(got[0], "rule basis") {
				t.Fatal(got)
			}
			if tc.at != nil && !strings.Contains(got[0], reportTime(*tc.at, time.Local)) {
				t.Fatal(got)
			}
		})
	}
}

func TestWindowInputsUsesHeldSchedules(t *testing.T) {
	t.Parallel()
	a, transport, _ := weekTestApp(t, "17")
	raw, err := nflschedule.Parse([]byte(weekFixture(t, "nflSchedule-w5.json")))
	if err != nil {
		t.Fatal(err)
	}
	raw.Week = "9"
	a.now.schedules = map[int]heldSeasonFeed[[]nflschedule.RawWeek]{9: {value: []nflschedule.RawWeek{raw}}}
	now := time.Now()
	week, err := nflschedule.ToWeek(raw, now)
	if err != nil {
		t.Fatal(err)
	}
	in := a.windowInputs(now)
	if in.LastRegular != 13 || !in.FirstKickoff[9].Equal(leagueweek.FirstLock(week)) {
		t.Fatal(in)
	}
	clock, err := a.TargetClock()
	if err != nil {
		t.Fatal(err)
	}
	if clock.Value.Windows[0].Status != leagueclock.WindowDated || len(transport.requests) != 0 {
		t.Fatal(clock)
	}
}
