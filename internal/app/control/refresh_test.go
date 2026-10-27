package control

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livescoring"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/pendingtrades"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

func TestSeasonRefreshOrderAndCancellation(t *testing.T) {
	for _, stop := range []int{0, 1, 3, 6} {
		calls := []string{}
		step := func(name string) func(context.Context) { return func(context.Context) { calls = append(calls, name) } }
		pauses := 0
		s := SeasonRefresh{Changed: step("changed"), Transactions: step("transactions"), Comms: step("comms"), Shell: step("shell"),
			Lineups: step("lineups"), Trades: step("trades"), Injuries: step("injuries"), Draft: step("draft"), Market: step("market"),
			Race: step("race"), Firsts: step("firsts"), Revalidate: step("revalidate"), Week: func() int { return 3 },
			Schedule: func(_ context.Context, week int) {
				if week != 3 {
					t.Fatal(week)
				}
				calls = append(calls, "schedule")
			},
			Pause: func() func(context.Context, time.Duration) error {
				return func(context.Context, time.Duration) error {
					pauses++
					if pauses == stop {
						return context.Canceled
					}
					return nil
				}
			},
		}
		s.Run(context.Background())
		want := []string{"transactions", "comms", "shell", "lineups", "changed", "trades", "injuries", "draft", "market", "race", "firsts", "schedule", "revalidate", "changed"}
		switch stop {
		case 1:
			want = []string{"transactions", "comms", "shell", "changed"}
		case 3:
			want = []string{"transactions", "comms", "shell", "lineups", "changed", "trades", "changed"}
		case 6:
			want = append(want[:11], "changed")
		}
		if !reflect.DeepEqual(calls, want) {
			t.Fatal(stop, calls, want)
		}
		calls = nil
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		s.Run(ctx)
		if len(calls) != 0 {
			t.Fatal("canceled refresh effects", calls)
		}
	}
}

func TestPendingRefreshKeyAndGenerationFences(t *testing.T) {
	at := time.Now()
	original := report.HeldFeed[[]leaguefeed.PendingTrade]{Value: []leaguefeed.PendingTrade{{ID: "held"}}, FetchedAt: at}
	read := func(context.Context) (pendingtrades.RawPendingTrades, error) {
		t.Fatal("fetch without key")
		return pendingtrades.RawPendingTrades{}, nil
	}
	RefreshPending(context.Background(), mflkey.ErrNoKey, read, func(value []leaguefeed.PendingTrade, err error) {
		if len(value) != 0 || !errors.Is(err, mflkey.ErrNoKey) {
			t.Fatal(value, err)
		}
	})
	if err := PendingKey(context.Background(), nil); !errors.Is(err, mflkey.ErrUnavailable) {
		t.Fatal(err)
	}
	for _, keyErr := range []error{mflkey.ErrNoKey, mflkey.ErrUnavailable, mflkey.ErrTimeout, errors.New("offline"), nil} {
		held := original
		recorded, logged := 0, 0
		record := func(_ context.Context, _ []leaguefeed.PendingTrade, _ time.Time, err error) error {
			recorded++
			if err != keyErr {
				t.Fatal(err)
			}
			return errors.New("save failed")
		}
		CurrentPending(1, 2, func() { t.Fatal("entered season publication for old key generation") })
		if !reflect.DeepEqual(held, original) {
			t.Fatal("stale generation changed held", held)
		}
		PublishPending(context.Background(), &held, nil, keyErr, func() time.Time { return at }, record, func(string) { logged++ })
		switch keyErr {
		case mflkey.ErrNoKey:
			if len(held.Value) != 0 || !held.FetchedAt.IsZero() || held.RefreshError != "MFL not connected" || recorded != 0 {
				t.Fatal(held, recorded)
			}
		case mflkey.ErrUnavailable, mflkey.ErrTimeout:
			if len(held.Value) != 1 || held.RefreshError != "MFL keyring is locked or unavailable" || recorded != 0 {
				t.Fatal(held, recorded)
			}
		case nil:
			if len(held.Value) != 0 || held.RefreshError != "" || recorded != 1 || logged != 1 {
				t.Fatal(held, recorded, logged)
			}
		default:
			if len(held.Value) != 1 || held.RefreshError != "offline" || recorded != 1 || logged != 0 {
				t.Fatal(held, recorded, logged)
			}
		}
	}
}

func TestLineupWeekIsolationAndFailure(t *testing.T) {
	if LineupWeekParam(0) != "" || LineupWeekParam(3) != "3" {
		t.Fatal("week encoding")
	}
	if err := CheckLineupWeek(livescoring.RawLineups{Week: "2"}, 3); err == nil {
		t.Fatal("wrong week accepted")
	}
	if err := CheckLineupWeek(livescoring.RawLineups{Week: "2"}, 0); err != nil {
		t.Fatal(err)
	}
	at := time.Now()
	held := report.HeldFeed[leaguefeed.Lineups]{FetchedAt: at}
	week := 2
	PublishLineups(&held, &week, 3, leaguefeed.Lineups{}, domain.Freshness{}, errors.New("offline"))
	if week != 3 || !held.FetchedAt.IsZero() || held.RefreshError != "offline" {
		t.Fatal(held, week)
	}
	held.FetchedAt = at
	PublishLineups(&held, &week, 3, leaguefeed.Lineups{}, domain.Freshness{}, errors.New("offline"))
	if !held.FetchedAt.Equal(at) {
		t.Fatal("lost same-week last good")
	}
}

func TestDailyShellCacheAndClaim(t *testing.T) {
	now := time.Now()
	old := report.HeldFeed[[]int]{Value: []int{1}, FetchedAt: now.Add(-48 * time.Hour)}
	if !DailyShellReading(old, now).ReadPending {
		t.Fatal("old day not pending")
	}
	bad := old
	bad.RefreshError = "offline"
	if DailyShellReading(bad, now).ReadPending {
		t.Fatal("failure replaced with pending")
	}
	result := ReadDailyShell(context.Background(), old, func() time.Time { return now }, func(context.Context) report.HeldFeed[[]int] { t.Fatal("archive read for held feed"); return old }, nil)
	if !result.ReadPending {
		t.Fatal(result)
	}
	latest := report.HeldFeed[[]int]{Value: []int{2}, FetchedAt: now}
	merged := MergeDailyShell(&latest, old, now)
	if merged.Value[0] != 2 {
		t.Fatal("archive replaced current live", merged)
	}
	var days map[string]string
	if !ClaimShellDay(&days, "calendar", now, old) || ClaimShellDay(&days, "calendar", now, old) {
		t.Fatal("daily attempts not fenced")
	}
	if ClaimShellDay(&days, "bait", now, latest) {
		t.Fatal("today's success refetched")
	}
	calls := []string{}
	RefreshShell(context.Background(), func(context.Context, time.Duration) error { return nil }, func(context.Context) { calls = append(calls, "calendar") }, func(context.Context) { calls = append(calls, "bait") })
	if !reflect.DeepEqual(calls, []string{"calendar", "bait"}) {
		t.Fatal(calls)
	}
	calls = nil
	RefreshShell(context.Background(), func(context.Context, time.Duration) error { return context.Canceled }, func(context.Context) { calls = append(calls, "calendar") }, func(context.Context) { calls = append(calls, "bait") })
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}

func TestWeekRefreshFailureAndWorkerOrder(t *testing.T) {
	at := time.Now()
	held := leagueweek.Week{Number: 2}
	fetched := at
	failure := ""
	HoldWeek(&held, &fetched, &failure, leagueweek.Week{}, time.Time{}, errors.New("offline"))
	if held.Number != 2 || !fetched.Equal(at) || failure != "offline" {
		t.Fatal(held, fetched, failure)
	}
	calls := []string{}
	step := func(name string) func(context.Context) { return func(context.Context) { calls = append(calls, name) } }
	s := WeekRefresh{Now: func() time.Time { return at }, Read: func(context.Context, time.Time) (leagueweek.Week, error) {
		return leagueweek.Week{}, errors.New("offline")
	},
		Publish: func(leagueweek.Week, time.Time, error) { calls = append(calls, "publish") }, Derive: func(context.Context, leagueweek.Week, time.Time) { t.Fatal("derived failed read") }, Revalidate: step("revalidate"), Changed: step("changed"), Log: func(string) { calls = append(calls, "log") }}
	s.Run(context.Background())
	if !reflect.DeepEqual(calls, []string{"publish", "log", "revalidate", "changed"}) {
		t.Fatal(calls)
	}
	calls = nil
	ctx, cancel := context.WithCancel(context.Background())
	worker := WeekWorker{Timeout: time.Minute, Week: step("week"), Season: step("season"), Finished: step("finished"), Engine: step("engine"), Changed: func(context.Context) { calls = append(calls, "changed"); cancel() }, Launch: step("launch"), LastRegular: func() int { return 0 }, Schedule: func(_ context.Context, week int) {
		if week != 9 {
			t.Fatal(week)
		}
		calls = append(calls, "schedule")
	}}
	worker.Run(ctx, make(chan time.Time))
	if !reflect.DeepEqual(calls, []string{"week", "season", "schedule", "finished", "engine", "changed"}) {
		t.Fatal(calls)
	}
}

func TestSeedWeekRefusesAbsentAndAmbiguousArchive(t *testing.T) {
	for _, rows := range [][]nflschedule.RawWeek{nil, {{}, {}}} {
		logs := 0
		SeedWeek(time.Now(), func(int) ([]nflschedule.RawWeek, time.Time, bool, error) { return rows, time.Now(), true, nil }, func(leagueweek.Week, time.Time) { t.Fatal("published invalid archive") }, func(string) { logs++ })
		if logs != 1 {
			t.Fatal(logs)
		}
	}
}

func TestDeriveWeekPhaseAdmission(t *testing.T) {
	cfg := league.RawConfig{StartWeek: "1", LastRegularSeasonWeek: "14", EndWeek: "18"}
	calls := 0
	read := func(context.Context) (domain.Phase, error) {
		calls++
		return domain.Phase(""), errors.New("state unavailable")
	}
	execute := func(context.Context, transactions.Request) (transactions.Receipt, error) {
		t.Fatal("executed with invalid inputs")
		return transactions.Receipt{}, nil
	}
	DeriveWeekPhase(context.Background(), leagueweek.Week{Number: 19}, time.Now(), cfg, read, execute, func(string) {})
	if calls != 0 {
		t.Fatal("read state after season over")
	}
	DeriveWeekPhase(context.Background(), leagueweek.Week{Number: 3}, time.Now(), cfg, read, execute, func(string) {})
	if calls != 1 {
		t.Fatal("did not read valid phase")
	}
}
