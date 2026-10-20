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

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
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
	a := &App{started: make(chan struct{}), startupErr: errors.New("fixture startup failed")}
	close(a.started)
	values := map[string][]byte{
		"GetParams":               sourcesWire(t, a.GetParams()),
		"SetParam":                sourcesWire(t, a.SetParam("model.k_now", "", 4)),
		"ResetParam":              sourcesWire(t, a.ResetParam("model.k_now", "")),
		"TargetLeagueSettings":    sourcesWire(t, controlResult(a.TargetLeagueSettings())),
		"TargetSiteNews":          sourcesWire(t, controlResult(a.TargetSiteNews())),
		"TargetAccounting":        sourcesWire(t, a.TargetAccounting()),
		"TargetBylaws":            sourcesWire(t, a.TargetBylaws()),
		"TargetOpenMFLAccounting": sourcesWire(t, controlResult(struct{}{}, a.TargetOpenMFLAccounting())),
		"TargetAlerts":            sourcesWire(t, controlResult(a.TargetAlerts("0001"))),
		"TargetSeason":            sourcesWire(t, controlResult(a.TargetSeason())),
		"TargetShellStatus":       sourcesWire(t, controlResult(a.TargetShellStatus())),
		"TargetSetWindow":         sourcesWire(t, controlResult(struct{}{}, a.TargetSetWindow("unknown", ""))),
		"MFLKeyStatus":            sourcesWire(t, controlResult(a.MFLKeyStatus())),
		"SetMFLKey":               sourcesWire(t, controlResult(a.SetMFLKey(""))),
		"DeleteMFLKey":            sourcesWire(t, controlResult(a.DeleteMFLKey())),
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
	a.refreshWeek(a.ctx)
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	week := a.weekSnapshot()
	original := week.Week.Games[0].Kickoff
	a.weekMu.Lock()
	a.week.Games[0].Kickoff = original.Add(time.Hour)
	a.weekMu.Unlock()
	if !week.Week.Games[0].Kickoff.Equal(original) {
		t.Fatal("week snapshot aliases held games")
	}
	raw, err := nflschedule.Parse([]byte(strings.Replace(weekFixture(t, "nflSchedule-w5.json"), `"week":"5"`, `"week":"9"`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	a.seasonMu.Lock()
	a.now.schedules = map[int]heldSeasonFeed[[]nflschedule.RawWeek]{9: {value: []nflschedule.RawWeek{raw}}}
	a.seasonMu.Unlock()
	windows := a.windowSnapshot()
	want := controlapp.WindowInputs(windows, now).FirstKickoff[9]
	if want.IsZero() {
		t.Fatal("missing trade window")
	}
	a.seasonMu.Lock()
	a.now.schedules[9].value[0].Matchups[0].Kickoff = "0"
	a.seasonMu.Unlock()
	if got := controlapp.WindowInputs(windows, now).FirstKickoff[9]; !got.Equal(want) {
		t.Fatal("window snapshot aliases held matchups")
	}
	// A value snapshot can be evaluated while both root locks are held.
	a.weekMu.Lock()
	a.seasonMu.Lock()
	defer a.weekMu.Unlock()
	defer a.seasonMu.Unlock()
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
	a.clockNow = func() time.Time { return time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC) }
	a.seasonPause = func(context.Context, time.Duration) error { return context.Canceled }
	var wg sync.WaitGroup
	start := make(chan struct{})
	wg.Go(func() {
		<-start
		for range 8 {
			a.refreshWeek(a.ctx)
			a.refreshNowWeek(a.ctx, 9, true)
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
	if a.week.Number != 5 || len(a.now.schedules[9].value) != 1 || a.now.schedules[9].refreshError != "" {
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
	for _, in := range [][]controlapp.BylawSection{nil, {}, {{Title: "Section", Lines: []string{"Line"}}}} {
		if string(sourcesWire(t, in)) != string(sourcesWire(t, wireBylaws(in))) {
			t.Fatal("bylaw wire twin drifted")
		}
	}
}
