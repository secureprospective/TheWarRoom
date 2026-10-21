package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
)

type controlWireResult[T any] struct {
	Value T      `json:"value"`
	Error string `json:"error"`
}

func controlResult[T any](value T, err error) controlWireResult[T] {
	text := ""
	if err != nil {
		text = err.Error()
	}
	return controlWireResult[T]{Value: value, Error: text}
}

func TestControlBindingGolden(t *testing.T) {
	a := &App{marketHeld: &marketapp.Held{}, raceHeld: &raceapp.Held{}, core: &core.State{Started: make(chan struct{}), StartupErr: errors.New("fixture startup failed")}}
	close(a.core.Started)
	values := map[string][]byte{
		"GetParams":               sourcesWire(t, a.controlAPI().GetParams()),
		"SetParam":                sourcesWire(t, a.controlAPI().SetParam("model.k_now", "", 4)),
		"ResetParam":              sourcesWire(t, a.controlAPI().ResetParam("model.k_now", "")),
		"TargetLeagueSettings":    sourcesWire(t, controlResult(a.controlAPI().TargetLeagueSettings())),
		"TargetSiteNews":          sourcesWire(t, controlResult(a.controlAPI().TargetSiteNews())),
		"TargetAccounting":        sourcesWire(t, a.controlAPI().TargetAccounting()),
		"TargetBylaws":            sourcesWire(t, a.controlAPI().TargetBylaws()),
		"TargetOpenMFLAccounting": sourcesWire(t, controlResult(struct{}{}, a.controlAPI().TargetOpenMFLAccounting())),
		"TargetAlerts":            sourcesWire(t, controlResult(a.TargetAlerts("0001"))),
		"TargetSeason":            sourcesWire(t, controlResult(a.TargetSeason())),
		"TargetShellStatus":       sourcesWire(t, controlResult(a.TargetShellStatus())),
		"TargetSetWindow":         sourcesWire(t, controlResult(struct{}{}, a.TargetSetWindow("unknown", ""))),
		"MFLKeyStatus":            sourcesWire(t, controlResult(a.controlAPI().MFLKeyStatus())),
		"SetMFLKey":               sourcesWire(t, controlResult(a.controlAPI().SetMFLKey(""))),
		"DeleteMFLKey":            sourcesWire(t, controlResult(a.controlAPI().DeleteMFLKey())),
	}
	for name, body := range values {
		path := filepath.Join("testdata", "wire", "control", name+".json")
		if os.Getenv("TWR_WRITE_WIRE") == "1" {
			if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, body, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		held, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(held) != string(body) {
			t.Errorf("%s binding wire changed", name)
		}
	}
}

func TestControlSnapshotsDetachHeldInputs(t *testing.T) {
	a, _, _ := weekTestApp(t, "17", weekFixture(t, "nflSchedule-w5.json"))
	a.refreshWeek(a.core.Ctx)
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	week := a.weekSnapshot()
	original := week.Week.Games[0].Kickoff
	a.core.WeekMu.Lock()
	a.core.Week.Games[0].Kickoff = original.Add(time.Hour)
	a.core.WeekMu.Unlock()
	if !week.Week.Games[0].Kickoff.Equal(original) {
		t.Fatal("week snapshot aliases held games")
	}
	raw, err := nflschedule.Parse([]byte(strings.Replace(weekFixture(t, "nflSchedule-w5.json"), `"week":"5"`, `"week":"9"`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	a.core.SeasonMu.Lock()
	a.now.schedules = map[int]heldSeasonFeed[[]nflschedule.RawWeek]{9: {Value: []nflschedule.RawWeek{raw}}}
	a.core.SeasonMu.Unlock()
	windows := a.windowSnapshot()
	want := controlapp.WindowInputs(windows, now).FirstKickoff[9]
	if want.IsZero() {
		t.Fatal("missing trade window")
	}
	a.core.SeasonMu.Lock()
	a.now.schedules[9].Value[0].Matchups[0].Kickoff = "0"
	a.core.SeasonMu.Unlock()
	if got := controlapp.WindowInputs(windows, now).FirstKickoff[9]; !got.Equal(want) {
		t.Fatal("window snapshot aliases held matchups")
	}
	// A value snapshot can be evaluated while both root locks are held.
	a.core.WeekMu.Lock()
	a.core.SeasonMu.Lock()
	defer a.core.WeekMu.Unlock()
	defer a.core.SeasonMu.Unlock()
	done := make(chan struct{})
	go func() { controlapp.WeekClock(week, now); controlapp.WindowInputs(windows, now); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("service took a root lock")
	}
}

func TestControlConcurrentRefreshAndRead(t *testing.T) {
	body := weekFixture(t, "nflSchedule-w5.json")
	bodies := make([]string, 0, 16)
	for range 8 {
		bodies = append(bodies, body, strings.Replace(body, `"week":"5"`, `"week":"9"`, 1))
	}
	a, _, _ := weekTestApp(t, "17", bodies...)
	a.core.ClockNow = func() time.Time { return time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC) }
	a.core.SeasonPause = func(context.Context, time.Duration) error { return context.Canceled }
	var wg sync.WaitGroup
	start := make(chan struct{})
	wg.Go(func() {
		<-start
		for range 8 {
			a.refreshWeek(a.core.Ctx)
			a.refreshNowWeek(a.core.Ctx, 9, true)
		}
	})
	for range 4 {
		wg.Go(func() {
			<-start
			for range 100 {
				lock, fresh := a.weekClock(a.timeNow())
				if lock != nil && lock.Week != 5 {
					t.Errorf("mixed week snapshot: %+v", lock)
				}
				if fresh.State != FreshLive {
					t.Errorf("unexpected freshness: %+v", fresh)
				}
				a.windowInputs(a.timeNow())
				controlapp.Gameday(a.weekSnapshot(), a.timeNow())
			}
		})
	}
	close(start)
	wg.Wait()
	if a.core.Week.Number != 5 || len(a.now.schedules[9].Value) != 1 || a.now.schedules[9].RefreshError != "" {
		t.Fatal("refresh did not publish both snapshots")
	}
}

func TestControlWireTwins(t *testing.T) {
	t.Parallel()
	in := controlapp.Alert{Kind: "bye", Urgency: "U1", Title: "Title", Detail: "Detail", Node: "hq", Workspace: "lineup-and-roster", At: time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC), PlayerID: "0042", MFLURL: "https://example.invalid"}
	n := 0
	fillWire(reflect.ValueOf(&in.Provenance).Elem(), &n)
	if string(sourcesWire(t, in)) != string(sourcesWire(t, Alert(in))) {
		t.Fatal("alert wire twin drifted")
	}
}
