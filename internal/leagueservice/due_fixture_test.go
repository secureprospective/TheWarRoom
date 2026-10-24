package leagueservice

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

const dueLeague = "native.worker"

type dueTestClock struct {
	instant   atomic.Int64
	untrusted atomic.Bool
}

func (c *dueTestClock) Now() time.Time { return time.Unix(0, c.instant.Load()).UTC() }
func (c *dueTestClock) Trusted() error {
	if c.untrusted.Load() {
		return fmt.Errorf("synchronization lost")
	}
	return nil
}

type dueEnvironment struct{}

func (dueEnvironment) BidEnvironment(context.Context, string, time.Time) (commissioner.BidEnvironment, error) {
	return commissioner.BidEnvironment{Season: nativecheck.BidSeason{Week: 5, WeekKnown: true}}, nil
}

type dueFixture struct {
	league string
	store  *history.Store
	pools  *db.Pools
	clock  *dueTestClock
	worker *DueWorker
	facts  nativecheck.Facts
	path   string
}

func newDueFixture(t *testing.T) *dueFixture {
	t.Helper()
	f := &dueFixture{league: dueLeague, clock: &dueTestClock{}, path: filepath.Join(t.TempDir(), "history.db")}
	f.clock.instant.Store(time.Date(2026, 10, 24, 12, 0, 0, 321, time.UTC).UnixNano())
	f.facts = nativecheck.Facts{"0099": {Position: leaguerules.QB, DesignationHeld: true}, "5001": {Position: leaguerules.QB, DesignationHeld: true}}
	f.open(t)
	f.seed(t)
	return f
}

func (f *dueFixture) seed(t *testing.T) {
	t.Helper()
	f.execute(t, "creator", commissioner.Create, commissioner.RawArgs{Name: "Worker league", Preset: commissioner.PresetLegacy, Season: 2026, FranchiseCount: 32})
	at := f.clock.Now()
	seed := &commissioner.RawNativeSnapshot{Season: 2026, RosterAt: at, DraftAt: at, FutureAt: at,

		Players: []commissioner.NativePlayer{{ID: "0099", Franchise: "franchise.02", Status: domain.RosterActive, Salary: 100000000, ExpirationYear: 2028, ContractStatus: domain.CStatusUFA}},
	}
	order := []string{}
	for n := 1; n <= 32; n++ {
		id := fmt.Sprintf("franchise.%02d", n)
		seed.Franchises = append(seed.Franchises, commissioner.NativeFranchise{ID: id, HeldID: fmt.Sprintf("%04d", n), Name: id})
		order = append(order, id)
	}
	f.execute(t, "creator", commissioner.NativeSeed, commissioner.RawArgs{Native: seed})
	for _, rule := range []struct{ field, value string }{
		{"roster.limits", `{"min":0,"max":100}`}, {"cap.amount", `100000000000`},
		{"calendar.timezone", `"UTC"`}, {"calendar.boundary", `"start_of_day"`},
		{"calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["cut","waiver_claim","ufa_bid"]}}`},
	} {
		f.execute(t, "creator", commissioner.RulesSet, commissioner.RawArgs{Field: rule.field, Value: json.RawMessage(rule.value)})
	}
	for _, seat := range []struct{ team, owner string }{{"franchise.01", "owner.one"}, {"franchise.02", "owner.two"}} {
		f.execute(t, "creator", "commish.seat.replace", commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: seat.team, Principal: seat.owner}})
	}
	f.execute(t, "creator", commissioner.WaiverOrderSet, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Order: order}})
}

func (f *dueFixture) open(t *testing.T) {
	t.Helper()
	pools, err := db.Open(t.Context(), f.path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	f.pools = pools
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	f.store = history.New(pools, &measures.Registry{})
	if err := f.store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	f.store.SetClock(f.clock.Now)
	f.store.SetBidEnvironment(dueEnvironment{})
	f.worker = NewDueWorker(f.store, func() nativecheck.Facts { return f.facts }, f.clock)
}

func (f *dueFixture) raw(t *testing.T, command string, args commissioner.RawArgs) commissioner.RawCommand {
	t.Helper()
	events, err := f.store.Events(t.Context(), f.league, commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	visibility := "league"
	if command == commissioner.BidPlace || command == commissioner.BidRaise || command == commissioner.BidWithdraw || command == commissioner.WaiverClaim {
		visibility = "private"
	}
	return commissioner.RawCommand{League: f.league, Command: command, Args: args, Expected: int64(len(events)), Request: fmt.Sprintf("worker.%d.%s", len(events), command), Reason: "Exercise due scheduling", Effective: "now", Visibility: visibility}
}

func (f *dueFixture) execute(t *testing.T, principal, command string, args commissioner.RawArgs) commissioner.Receipt {
	t.Helper()
	receipt, err := f.store.ExecuteCommissionerWithFacts(t.Context(), f.raw(t, command, args), principal, nil, f.facts, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func (f *dueFixture) bid(t *testing.T) commissioner.Receipt {
	t.Helper()
	return f.execute(t, "owner.one", commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5001", Franchise: "franchise.01", Salary: 500000000, Years: 1}})
}

func (f *dueFixture) events(t *testing.T) []history.LeagueEvent {
	t.Helper()
	events, err := f.store.Events(t.Context(), f.league, commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	return events
}
