package control

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestLeagueRefreshStagesTogetherAndPublishesOnlyAfterCommit(t *testing.T) {
	failure := errors.New("failed write")
	for _, step := range []string{"mirror", "commit", "success", "unchanged"} {
		t.Run(step, func(t *testing.T) {
			calls := []string{}
			service := LeagueRefresh{
				Discover: func(context.Context) (league.RawConfig, error) {
					calls = append(calls, "discover")
					return league.RawConfig{CurrentSeason: 2026}, nil
				},
				Fetch: func(_ context.Context, season int) (state.MirrorSnapshot, error) {
					if season != 2026 {
						t.Fatal(season)
					}
					calls = append(calls, "fetch")
					return state.MirrorSnapshot{Season: season}, nil
				},
				StageRules: func(league.RawConfig) (db.PreparedWrite, rulebook.CapPercents, bool, error) {
					calls = append(calls, "rules")
					return db.PreparedWrite{Release: func() { calls = append(calls, "release") }}, rulebook.CapPercents{}, step != "unchanged", nil
				},
				StageMirror: func(state.MirrorSnapshot, state.CapDiscounts) (db.PreparedWrite, bool, error) {
					calls = append(calls, "mirror")
					if step == "mirror" {
						return db.PreparedWrite{}, false, failure
					}
					return db.PreparedWrite{}, false, nil
				},
				Commit: func(_ context.Context, writes ...db.PreparedWrite) error {
					if len(writes) != 2 {
						t.Fatal(len(writes))
					}
					calls = append(calls, "commit")
					if step == "commit" {
						return failure
					}
					return nil
				},
				RulesChecked: func(int64) { calls = append(calls, "rules checked") }, Dirty: func(bool) { calls = append(calls, "dirty") }, RostersChecked: func(time.Time) { calls = append(calls, "rosters checked") }, Now: time.Now,
				DraftReports: func(context.Context) { calls = append(calls, "draft") }, Revalidate: func(context.Context) { calls = append(calls, "revalidate") },
			}
			out, err := service.Refresh(context.Background())
			want := []string{"discover", "fetch", "rules", "mirror"}
			switch step {
			case "mirror":
				want = append(want, "release")
			case "commit":
				want = append(want, "commit")
			default:
				want = append(want, "commit", "rules checked")
				if step != "unchanged" {
					want = append(want, "dirty")
				}
				want = append(want, "rosters checked", "draft", "revalidate")
				if err != nil || out.Season != 2026 || out.RulesChanged != (step != "unchanged") {
					t.Fatal(out, err)
				}
			}
			if (step == "mirror" || step == "commit") && !errors.Is(err, failure) {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, want) {
				t.Fatal(calls, want)
			}
		})
	}
}

func TestBackgroundRefreshReadinessAndLaunchOrder(t *testing.T) {
	workers := []func(){}
	calls := []string{}
	readyErr := error(nil)
	service := BackgroundRefresh{Ready: func() error { return readyErr }, Start: func(work func()) { workers = append(workers, work) }, Due: func() bool { return true },
		Clock: func(context.Context) { calls = append(calls, "clock") }, Watch: func(context.Context) { calls = append(calls, "watch") }, Launch: func(context.Context) { calls = append(calls, "launch") }, Weeks: func(context.Context) { calls = append(calls, "weeks") },
	}
	service.StartWorkers(context.Background())
	if len(workers) != 3 || len(calls) != 0 {
		t.Fatal(len(workers), calls)
	}
	readyErr = context.Canceled
	for _, work := range workers {
		work()
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
	readyErr = nil
	for _, work := range workers {
		work()
	}
	if !reflect.DeepEqual(calls, []string{"clock", "watch", "launch", "weeks"}) {
		t.Fatal(calls)
	}
	dirty, logs := false, 0
	LaunchLeagueRefresh(context.Background(), time.Minute, func(context.Context) (LeagueRefreshResult, error) { return LeagueRefreshResult{}, context.Canceled }, func(bool) { dirty = true }, func(string) { logs++ })
	if dirty || logs != 1 {
		t.Fatal(dirty, logs)
	}
	LaunchLeagueRefresh(context.Background(), time.Minute, func(ctx context.Context) (LeagueRefreshResult, error) {
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("unbounded")
		}
		return LeagueRefreshResult{Changed: true}, nil
	}, func(bool) { dirty = true }, func(string) { logs++ })
	if !dirty || logs != 2 {
		t.Fatal(dirty, logs)
	}
}
