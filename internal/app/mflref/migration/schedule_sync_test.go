package migration

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"

	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

type unavailableFeed struct{}

func (unavailableFeed) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("offline fixture")
}

func TestScheduleSyncWithoutHistory(t *testing.T) {
	(ScheduleSync{}).Sync(t.Context())
}

func TestFollowedLineupsWorkerLifetime(t *testing.T) {
	client, err := mfl.New("api", 10000, mfl.WithTransport(unavailableFeed{}))
	if err != nil {
		t.Fatal(err)
	}
	work, cancel := context.WithCancel(t.Context())
	defer cancel()
	var running, started atomic.Bool
	var queued []func()
	s := ScheduleSync{Client: client, WorkContext: func() context.Context { return work }, Running: &running, Started: &started,
		StartWorker: func(f func()) { queued = append(queued, f) }}
	ledger := leagueservice.MigrationLedger{}
	s.syncFollowedLineups(t.Context(), ledger, []followedLeague{{"league", "owner", 2026, "reference"}, {"other", "owner", 2026, "reference"}})
	s.syncFollowedLineups(t.Context(), ledger, []followedLeague{{"league", "owner", 2026, "reference"}, {"other", "owner", 2026, "reference"}})
	if len(queued) != 1 || !running.Load() || started.Load() {
		t.Fatalf("before worker: queued=%d running=%t started=%t", len(queued), running.Load(), started.Load())
	}
	cancel()
	queued[0]()
	if running.Load() || !started.Load() {
		t.Fatal("cancelled first-launch worker did not release its guard")
	}
	if playedLineupsTimeout != 10*time.Minute || playedLineupsLaunchDelay != 5*time.Minute {
		t.Fatal("reference import timing changed")
	}
}

// Every eligible league must reach the same single history worker, even when an earlier feed fails.
func TestHistoryWorkerProcessesEveryLeague(t *testing.T) {
	store := syncTestStore(t)
	var leagues []followedLeague
	for _, id := range []string{"native.one", "native.two"} {
		plan, err := mflsource.Prepare(migrationtest.Synthetic(id, migrationtest.Specs()["8 teams"]))
		if err != nil {
			t.Fatal(err)
		}
		raw := commissioner.RawCommand{League: id, Command: commissioner.Create, Args: plan.Create, Reason: "start", Request: "create", Effective: "now", Visibility: "league"}
		if _, err := store.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
			t.Fatal(err)
		}
		leagues = append(leagues, followedLeague{id, "creator", 2026, id})
	}
	feed := &countedFeed{reads: map[string]int{}}
	client, err := mfl.New("api", 10000, mfl.WithTransport(feed))
	if err != nil {
		t.Fatal(err)
	}
	var running, started atomic.Bool
	started.Store(true)
	var queued []func()
	s := ScheduleSync{History: store, Client: client, Now: time.Now, WorkContext: func() context.Context { return t.Context() }, Running: &running, Started: &started, StartWorker: func(f func()) { queued = append(queued, f) }}
	s.syncFollowedLineups(t.Context(), leagueservice.MigrationLedger{Store: store}, leagues)
	if len(queued) != 1 {
		t.Fatalf("workers %d", len(queued))
	}
	queued[0]()
	for _, league := range leagues {
		if feed.reads[league.source] == 0 {
			t.Fatalf("dropped %s: %v", league.league, feed.reads)
		}
	}
	if running.Load() {
		t.Fatal("worker guard retained")
	}
}

type countedFeed struct{ reads map[string]int }

func (f *countedFeed) RoundTrip(r *http.Request) (*http.Response, error) {
	f.reads[r.URL.Query().Get("L")]++
	f.reads[r.URL.Query().Get("TYPE")]++
	if r.URL.Query().Get("TYPE") == "league" {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"league":{"baseURL":"https://www1.myfantasyleague.com/2026/home/00123"}}`)), Header: make(http.Header)}, nil
	}
	return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(strings.NewReader("offline")), Header: make(http.Header)}, nil
}

func syncTestStore(t *testing.T) *history.Store {
	t.Helper()
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "history.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(pools, reg)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	return store
}

func TestFollowSettingsIndependent(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		schedule, transactions bool
	}{
		{"both", true, true}, {"schedule only", true, false}, {"transactions only", false, true}, {"neither", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := syncTestStore(t)
			plan, err := mflsource.Prepare(migrationtest.Synthetic("Follow", migrationtest.Specs()["8 teams"]))
			if err != nil {
				t.Fatal(err)
			}
			raw := commissioner.RawCommand{League: "native.follow", Command: commissioner.Create, Args: plan.Create, Reason: "create", Request: "create", Effective: "now", Visibility: "league"}
			receipt, err := store.ExecuteCommissioner(t.Context(), raw, "creator", nil)
			if err != nil {
				t.Fatal(err)
			}
			changes := []struct{ field, value string }{}
			if !tc.transactions {
				changes = append(changes, struct{ field, value string }{"transactions.follow", strconv.FormatBool(tc.transactions)})
			}
			if !tc.schedule {
				changes = append(changes, struct{ field, value string }{"schedule.follow", `{"Source":"","League":"","Season":0}`})
			}
			for _, change := range changes {
				raw.Command, raw.Request, raw.Expected = commissioner.RulesSet, change.field, receipt.Revision
				raw.Args = commissioner.RawArgs{Field: change.field, Value: json.RawMessage(change.value)}
				receipt, err = store.ExecuteCommissioner(t.Context(), raw, "creator", nil)
				if err != nil {
					t.Fatal(err)
				}
			}
			feed := &countedFeed{reads: map[string]int{}}
			client, err := mfl.New("api", 10000, mfl.WithTransport(feed))
			if err != nil {
				t.Fatal(err)
			}
			s := ScheduleSync{History: store, Client: client, Now: time.Now, Authorizer: func() commissioner.Authorizer { return nil }, FillScheduleDates: func(context.Context, []commissioner.Record) error { return nil }, SyncScoring: func(context.Context, string, []commissioner.Record) error { return nil }, WorkContext: func() context.Context { return nil }}
			s.Sync(t.Context())
			for _, want := range []struct {
				kind    string
				enabled bool
			}{{"transactions", tc.transactions}, {"schedule", tc.schedule}, {"liveScoring", tc.schedule}, {"weeklyResults", tc.schedule}} {
				if (feed.reads[want.kind] > 0) != want.enabled {
					t.Errorf("%s requests=%d enabled=%t", want.kind, feed.reads[want.kind], want.enabled)
				}
			}
		})
	}
}
