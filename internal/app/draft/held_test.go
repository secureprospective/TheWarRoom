package draft

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/draft/drafttest"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type fakeHistory struct {
	run      history.Run
	ok       bool
	err      error
	scores   []history.ModelScore
	asked    []int64
	runErr   error
	seasonOf int
}

func (h *fakeHistory) LatestRun(_ context.Context, season int, kind history.RunKind) (history.Run, bool, error) {
	if kind != history.RunModel {
		return history.Run{}, false, errors.New("wrong run kind")
	}
	h.seasonOf = season
	return h.run, h.ok, h.runErr
}

func (h *fakeHistory) ModelScores(_ context.Context, runID int64) ([]history.ModelScore, error) {
	h.asked = append(h.asked, runID)
	return h.scores, h.err
}

func TestValuesReadTheSeasonsLatestRun(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	meanings := func(names []string) []string { return []string{"Date of birth.", ""}[:len(names)] }
	at := time.Date(2026, 10, 5, 13, 44, 0, 0, time.UTC)
	h := &fakeHistory{run: history.Run{ID: 7, AsOf: at, MissingMeasures: []string{"context.birth_date", "x.y"}}, ok: true,
		scores: []history.ModelScore{{MFLID: "0531", Value: model.Value{DynastyPPG: 12.5}}, {MFLID: "0532", Value: model.Value{DynastyPPG: 3}}}}
	values, note := Values(ctx, h, 2026, meanings, time.UTC)
	if h.seasonOf != 2026 || !reflect.DeepEqual(h.asked, []int64{7}) || !reflect.DeepEqual(values, map[string]float64{"0531": 12.5, "0532": 3}) ||
		note != "Engine run: October 5, 2026, 1:44 PM. Missing player information: Date of birth; an unknown player measure." {
		t.Fatal(h, values, note)
	}
	h.ok = false
	if values, note := Values(ctx, h, 2026, meanings, time.UTC); len(values) != 0 || note != "Engine points unavailable until the engine first runs." {
		t.Fatal(values, note)
	}
	h.ok, h.err = true, errors.New("locked")
	if values, note := Values(ctx, h, 2026, meanings, time.UTC); len(values) != 0 || note != "Engine points unavailable: locked" {
		t.Fatal(values, note)
	}
	h.runErr = errors.New("gone")
	if _, note := Values(ctx, h, 2026, meanings, time.UTC); note != "Engine points unavailable: gone" {
		t.Fatal(note)
	}
}

func TestNamesOnlyKnownDraftedPlayers(t *testing.T) {
	t.Parallel()
	picks := []draftreports.Pick{drafttest.Pick(t, 1, 1, "0001", "0531", 0), drafttest.Pick(t, 1, 2, "0001", "0532", 0)}
	picks = append(picks, draftreports.Pick{Round: 1, Slot: 3})
	got := Names(picks, func(id string) (string, bool) { return "Known " + id, id == "0531" })
	if !reflect.DeepEqual(got, map[string]string{"0531": "Known 0531"}) {
		t.Fatal(got)
	}
}

func TestNextDraftReadOrderAndFailure(t *testing.T) {
	t.Parallel()
	at := time.Unix(1800000000, 0)
	next := at.Add(time.Hour)
	for _, tc := range []struct {
		name      string
		available bool
		err       error
		windows   []leagueclock.Window
		want      string
		calls     []string
	}{
		{name: "no store", want: "Next draft: no date set by the commissioner."},
		{name: "clock failed", available: true, err: errors.New("clock failed"), want: "Next draft date unavailable: clock failed", calls: []string{"clock"}},
		{name: "no date", available: true, want: "Next draft: no date set by the commissioner.", calls: []string{"clock", "now"}},
		{name: "held date", available: true, windows: []leagueclock.Window{{Kind: leagueclock.Draft, At: &next}}, want: "Next draft: " + next.In(time.UTC).Format("January 2, 2006, 3:04 PM"), calls: []string{"clock", "now"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var calls []string
			got := NextDraft(tc.available, func() (snapshot.Sourced[leagueclock.Reading], error) {
				calls = append(calls, "clock")
				return snapshot.Sourced[leagueclock.Reading]{Value: leagueclock.Reading{Windows: tc.windows}}, tc.err
			}, func() time.Time { calls = append(calls, "now"); return at }, time.UTC)
			if got != tc.want || !reflect.DeepEqual(calls, tc.calls) {
				t.Fatalf("%q calls %v", got, calls)
			}
		})
	}
}
