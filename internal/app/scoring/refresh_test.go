package scoring

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/modelrun"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestEngineRefreshAdmissionAndOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                    string
		available, locked, fail bool
		want                    []string
	}{
		{name: "unavailable"}, {name: "busy", available: true, want: []string{"lock"}},
		{name: "points failure", available: true, locked: true, fail: true, want: []string{"lock", "signals", "points", "log", "changed", "unlock"}},
		{name: "ready", available: true, locked: true, want: []string{"lock", "signals", "points", "log", "free", "check", "changed", "unlock"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			add := func(name string) func(context.Context) { return func(context.Context) { calls = append(calls, name) } }
			RefreshEngine(t.Context(), EngineRefreshSource{Available: func() bool { return tc.available }, TryLock: func() bool { calls = append(calls, "lock"); return tc.locked }, Unlock: func() { calls = append(calls, "unlock") }, Changed: add("changed"), Timeout: time.Second,
				Signals: add("signals"), Points: func(ctx context.Context) (string, error) {
					calls = append(calls, "points")
					if _, ok := ctx.Deadline(); !ok {
						t.Fatal("no bound")
					}
					if tc.fail {
						return "", errors.New("failed")
					}
					return "warning", nil
				}, FreeAgents: add("free"), Check: add("check"), Log: func(string) { calls = append(calls, "log") }})
			if !reflect.DeepEqual(calls, tc.want) {
				t.Fatalf("calls %v want %v", calls, tc.want)
			}
		})
	}
}

func TestEngineCheckDirtyFailureAndConcurrentChange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                      string
		initial, fail, concurrent bool
		want                      bool
	}{
		{name: "dirty failure restored", initial: true, fail: true, want: true},
		{name: "clean failure stays clean", fail: true},
		{name: "success consumes dirty", initial: true},
		{name: "concurrent dirty survives success", initial: true, concurrent: true, want: true},
		{name: "concurrent dirty survives failure", fail: true, concurrent: true, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var dirty atomic.Bool
			dirty.Store(tc.initial)
			finished := 0
			CheckEngine(t.Context(), EngineCheckSource{Latest: func(context.Context) (history.Run, bool, error) { return history.Run{}, false, nil }, InputsChanged: func(context.Context, time.Time) (bool, error) { return false, nil }, Now: time.Now, Started: time.Now, DirtyLoad: dirty.Load, DirtySwap: dirty.Swap, DirtyStore: dirty.Store,
				Directory: func(context.Context) (normalize.Lookup, error) { return normalize.Lookup{}, nil }, Run: func(context.Context, normalize.Lookup) (modelrun.Report, error) {
					if dirty.Load() {
						t.Fatal("dirty not consumed before run")
					}
					if tc.concurrent {
						dirty.Store(true)
					}
					if tc.fail {
						return modelrun.Report{}, errors.New("run failed")
					}
					return modelrun.Report{}, nil
				}, Finished: func(context.Context, time.Time, modelrun.Report) { finished++ }, Log: func(string) {}})
			if dirty.Load() != tc.want || (finished == 1) == tc.fail {
				t.Fatalf("dirty %v finished %d", dirty.Load(), finished)
			}
		})
	}
}

func TestEngineCheckEarlyFailureAndHealthNotification(t *testing.T) {
	t.Parallel()
	logged := ""
	CheckEngine(t.Context(), EngineCheckSource{Latest: func(context.Context) (history.Run, bool, error) { return history.Run{}, false, errors.New("disk") }, Log: func(s string) { logged = s }})
	if logged != "engine latest run: disk" {
		t.Fatal(logged)
	}
	SourceHealthChanged(t.Context(), nil)
	notified := 0
	at := time.Unix(1800000000, 0)
	EngineFinished(t.Context(), at, modelrun.Report{Scored: 3, RosteredScored: 2, FreeAgentScored: 1}, func(time.Time) time.Duration { return time.Second }, func(context.Context) { notified++ }, func(s string) { logged = s })
	if notified != 1 || logged == "engine latest run: disk" {
		t.Fatal("completion not published")
	}
}
