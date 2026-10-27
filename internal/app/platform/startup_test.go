package platform

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestStoreFloorOrderSeedAndNoPartialPublication(t *testing.T) {
	for _, seed := range []bool{false, true} {
		for _, fail := range []bool{false, true} {
			calls := []string{}
			step := func(name string) func(context.Context) error {
				return func(context.Context) error { calls = append(calls, name); return nil }
			}
			service := StoreFloor{History: step("history"), Seat: step("seat"), Params: step("params"), Moves: step("moves"), Personal: step("personal"), Rules: step("rules"), Mirror: step("mirror"),
				MirrorSeason: func() int {
					if seed {
						return 0
					}
					return 2026
				}, Refresh: step("refresh"), WhatIf: step("whatif"),
				Coordinator: func(context.Context) error {
					calls = append(calls, "coordinator")
					if fail {
						return context.Canceled
					}
					return nil
				},
				MigrationFindings: func() { calls = append(calls, "findings") }, Publish: func() { calls = append(calls, "publish") }, DraftReports: func(context.Context) { calls = append(calls, "draft") }, SeedClock: func(context.Context) { calls = append(calls, "clock") }, Log: func(string) {},
			}
			refreshed, err := service.Run(context.Background())
			want := []string{"history", "seat", "params", "moves", "personal", "rules", "mirror"}
			if seed {
				want = append(want, "refresh")
			}
			want = append(want, "whatif", "coordinator")
			if fail {
				if !errors.Is(err, context.Canceled) || refreshed || !strings.Contains(err.Error(), "transaction coordinator") {
					t.Fatal(err, refreshed)
				}
			} else {
				if err != nil || refreshed != seed {
					t.Fatal(err, refreshed)
				}
				want = append(want, "findings", "publish")
				if seed {
					want = append(want, "draft")
				}
				want = append(want, "clock")
			}
			if !reflect.DeepEqual(calls, want) {
				t.Fatal(calls, want)
			}
		}
	}
}

func TestStartupNonfatalLogFailureAndLaunchRefreshSelection(t *testing.T) {
	for _, refreshed := range []bool{false, true} {
		calls := []string{}
		due := false
		service := StartupWorkflow{
			Directory: func() (string, error) { return "fixture", nil }, SetDirectory: func(string) { calls = append(calls, "directory") }, Logging: func(string) error { return errors.New("logging blocked") },
			Databases: func(context.Context, string) (string, *history.Store, error) {
				calls = append(calls, "databases")
				return "fixture.db", nil, nil
			}, Client: func() error { calls = append(calls, "client"); return nil }, Photos: func() { calls = append(calls, "photos") },
			Floor: func(context.Context, *history.Store) (bool, error) {
				calls = append(calls, "floor")
				return refreshed, nil
			}, Keys: func(context.Context) { calls = append(calls, "keys") }, LaunchDue: func(v bool) { due = v }, Season: func() int { return 2026 }, Log: func(string) {},
		}
		got, err := service.Run(context.Background())
		if err != nil || got != refreshed || due == refreshed || !reflect.DeepEqual(calls, []string{"directory", "databases", "client", "photos", "floor", "keys"}) {
			t.Fatal(got, err, due, calls)
		}
	}
}

func TestFinishStartupPublishesBeforeConditionalRevalidation(t *testing.T) {
	for _, fail := range []bool{false, true} {
		for _, refreshed := range []bool{false, true} {
			published, revalidated := false, false
			failure := error(nil)
			if fail {
				failure = context.Canceled
			}
			FinishStartup(failure, refreshed, func() { published = true }, func() {
				if !published {
					t.Fatal("revalidate before ready")
				}
				revalidated = true
			})
			if !published || revalidated != (!fail && refreshed) {
				t.Fatal(published, revalidated)
			}
		}
	}
}

func TestDatabaseOpeningLockRecoveryAndStoreOrder(t *testing.T) {
	for _, fail := range []string{"lock", "recover", "history", "none"} {
		calls := []string{}
		call := func(name string) error {
			calls = append(calls, name)
			if fail == name {
				return context.Canceled
			}
			return nil
		}
		service := DatabaseOpening{
			Lock: func(string) error { return call("lock") }, Backups: func(string) error { return call("backups") }, Recover: func(context.Context) error { return call("recover") },
			BeforeMigrations: func(context.Context, time.Time) error { return call("before migrations") },
			Main: func(_ context.Context, path string) error {
				if !strings.HasSuffix(path, "thewarroom-dev.db") {
					t.Fatal(path)
				}
				return call("main")
			},
			Registry: func() (*measures.Registry, error) { return nil, call("registry") }, History: func(context.Context, string) error { return call("history") }, WhatIf: func(context.Context, string) error { return call("whatif") },
			Attach: func(*measures.Registry) *history.Store { calls = append(calls, "attach"); return nil }, Archive: func(*history.Store) { calls = append(calls, "archive") }, DevBuild: true,
		}
		_, _, err := service.Open(context.Background(), "fixture")
		want := []string{"lock", "backups", "recover", "before migrations", "main", "registry", "history", "whatif", "attach", "archive"}
		if fail != "none" {
			if !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			for i, name := range want {
				if name == fail {
					want = want[:i+1]
					break
				}
			}
		} else if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(calls, want) {
			t.Fatal(calls, want)
		}
	}
}

func TestWorkerGroupStopsAdmissionAndJoins(t *testing.T) {
	var mutex sync.Mutex
	var workers sync.WaitGroup
	stopping := false
	service := WorkerGroup{Mutex: &mutex, Stopping: &stopping, Workers: &workers}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started, finished := make(chan struct{}), make(chan struct{})
	service.Start(func() { close(started); <-ctx.Done(); close(finished) })
	<-started
	service.Stop(&cancel)
	select {
	case <-finished:
	default:
		t.Fatal("returned without join")
	}
	service.Start(func() { t.Error("admitted after stopping") })
	workers.Wait()
	if !stopping {
		t.Fatal("admission remains open")
	}
}
