package leagueservice

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/mflmirror"
)

// mflMovesMFL is a fake MFL source with recorded-shape data that counts calls; no live MFL is ever called.
type mflMovesMFL struct{ calls atomic.Int32 }

func (f *mflMovesMFL) FetchFreeAgents(context.Context, string, int) ([]string, error) {
	f.calls.Add(1)
	return []string{"0531", "13593"}, nil
}

func (f *mflMovesMFL) FetchPlayers(context.Context, string, int) ([]byte, error) {
	f.calls.Add(1)
	return []byte(`[{"ID":"0531","Name":"Doe, John","Position":"WR","Team":"BUF","DraftYear":"2020","Birthdate":"820000000"},` +
		`{"ID":"13593","Name":"Roe, Rich","Position":"RB","Team":"FA","DraftYear":"2025","Birthdate":"1000000000"}]`), nil
}

type mflMovesFixture struct {
	service  *Service
	mfl      *mflMovesMFL
	clock    *time.Time
	admitted *commissioner.Access
	league   string
}

type mflMovesDirectory map[string]domain.PlayerFacts

func (d mflMovesDirectory) Facts(id string) (domain.PlayerFacts, bool) { f, ok := d[id]; return f, ok }

func (f *mflMovesMFL) Directory([]byte) (mflmirror.Directory, error) {
	return mflMovesDirectory{
		"0531":  {Name: "Doe, John", Position: domain.PosWR, Team: "BUF", DraftYear: 2020, HasDraftYear: true, Birthdate: 820000000, HasBirthdate: true},
		"13593": {Name: "Roe, Rich", Position: domain.PosRB, Team: "FA", DraftYear: 2025, HasDraftYear: true, Birthdate: 1000000000, HasBirthdate: true},
	}, nil
}

func newMflMovesFixture(t *testing.T, principal string) mflMovesFixture {
	t.Helper()
	fixture := newDueFixture(t)
	pools, err := db.OpenScratch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	mirror := mflmirror.NewStore(pools, func() time.Time { return now })
	if err := mirror.Register(t.Context(), mflmirror.Registration{MFLLeague: "14432", Franchise: "0001"}); err != nil {
		t.Fatal(err)
	}
	fake := &mflMovesMFL{}
	var admitted commissioner.Access
	host := StoredHost{Store: fixture.store, Context: func() context.Context { return t.Context() },
		Principal: principal, Mirror: mirror, Fetcher: fake}
	service := NewHosted(host, admissionFunc(func(_ context.Context, access commissioner.Access) error {
		admitted = access
		return nil
	}))
	return mflMovesFixture{service: service, mfl: fake, clock: &now, admitted: &admitted, league: fixture.league}
}

func TestMflMovesSeatedOwnerRefreshesAndReadsPool(t *testing.T) {
	f := newMflMovesFixture(t, "owner.one")
	res, err := f.service.MflMovesRefreshContext(t.Context(), RawMflMovesScope{League: f.league})
	if err != nil || !res.AskedMFL || res.FreeAgents != 2 || res.MFLLeague != "14432" {
		t.Fatalf("refresh = %+v, %v", res, err)
	}
	pool, err := f.service.MflMovesPoolReadContext(t.Context(), RawMflPoolScope{League: f.league})
	if err != nil || len(pool.Rows) != 2 || pool.Rows[0].Name == "" {
		t.Fatalf("pool = %+v, %v", pool, err)
	}
	for _, row := range pool.Rows {
		if row.Value == "" {
			t.Fatalf("row has no value words: %+v", row)
		}
	}
	only, err := f.service.MflMovesPoolReadContext(t.Context(), RawMflPoolScope{League: f.league, Position: "RB", Sort: "name"})
	if err != nil || len(only.Rows) != 1 || only.Rows[0].Position != "RB" {
		t.Fatalf("position filter = %+v, %v", only, err)
	}
	if f.admitted.Franchise != "franchise.01" {
		t.Fatalf("access came from %q, not the seat", f.admitted.Franchise)
	}
}

func TestMflMovesRefreshThrottleReturnsHeldCopyWithAge(t *testing.T) {
	f := newMflMovesFixture(t, "owner.one")
	if _, err := f.service.MflMovesRefreshContext(t.Context(), RawMflMovesScope{League: f.league}); err != nil {
		t.Fatal(err)
	}
	calls := f.mfl.calls.Load()
	*f.clock = f.clock.Add(3 * time.Minute)
	res, err := f.service.MflMovesRefreshContext(t.Context(), RawMflMovesScope{League: f.league})
	if err != nil {
		t.Fatal(err)
	}
	if res.AskedMFL || f.mfl.calls.Load() != calls || res.AgeSeconds != 180 || res.FreeAgents != 2 {
		t.Fatalf("second refresh inside 5 minutes: %+v, calls %d -> %d", res, calls, f.mfl.calls.Load())
	}
}

func TestMflMovesRefusesARequestedFranchise(t *testing.T) {
	f := newMflMovesFixture(t, "owner.one")
	_, err := f.service.MflMovesPoolReadContext(t.Context(), RawMflPoolScope{League: f.league, Franchise: "franchise.02"})
	if err == nil || !strings.Contains(err.Error(), "no request franchise") {
		t.Fatalf("pool read named another franchise: %v", err)
	}
	_, err = f.service.MflMovesRefreshContext(t.Context(), RawMflMovesScope{League: f.league, Franchise: "franchise.02"})
	if err == nil || !strings.Contains(err.Error(), "no request franchise") || f.mfl.calls.Load() != 0 {
		t.Fatalf("refresh named another franchise: %v (calls %d)", err, f.mfl.calls.Load())
	}
}

func TestMflMovesRequiresASeatedOwner(t *testing.T) {
	f := newMflMovesFixture(t, "stranger")
	if _, err := f.service.MflMovesRefreshContext(t.Context(), RawMflMovesScope{League: f.league}); err == nil ||
		!strings.Contains(err.Error(), "seated franchise owner") || f.mfl.calls.Load() != 0 {
		t.Fatalf("unseated principal refreshed: %v (calls %d)", err, f.mfl.calls.Load())
	}
	if _, err := f.service.MflMovesPoolReadContext(t.Context(), RawMflPoolScope{League: f.league}); err == nil {
		t.Fatal("unseated principal read the pool")
	}
}

func TestMflMovesNeedsAnMflLeagueSetting(t *testing.T) {
	f := newMflMovesFixture(t, "owner.one")
	host := f.service.host.(StoredHost)
	pools, err := db.OpenScratch(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	host.Mirror = mflmirror.NewStore(pools, nil)
	service := NewHosted(host, admissionFunc(func(context.Context, commissioner.Access) error { return nil }))
	if _, err := service.MflMovesRefreshContext(t.Context(), RawMflMovesScope{League: f.league}); err == nil ||
		!strings.Contains(err.Error(), "no MFL league number") {
		t.Fatalf("refresh without a registered MFL league: %v", err)
	}
}
