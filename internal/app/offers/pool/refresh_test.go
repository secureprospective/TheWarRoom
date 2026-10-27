package pool

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/freeagents"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestFreeAgentPublicationRetainsGoodDataAndDirtyEvidence(t *testing.T) {
	t.Parallel()
	ids, err := FreeAgentIDs([]freeagents.RawFreeAgent{{ID: "0100"}, {ID: "0099"}, {ID: "0100"}}, nil)
	if err != nil || len(ids) != 2 || ids[0].String() != "0099" {
		t.Fatalf("normalized ids %v %v", ids, err)
	}
	changed := 0
	at := time.Now().Truncate(time.Second)
	held := report.HeldFeed[[]playerid.PlayerID]{}
	HoldFreeAgents(&held, ids, report.LiveFreshness(at), nil, func() { changed++ })
	HoldFreeAgents(&held, ids, report.LiveFreshness(at), nil, func() { changed++ })
	HoldFreeAgents(&held, nil, report.LiveFreshness(at), errors.New("offline"), func() { changed++ })
	if changed != 1 || len(held.Value) != 2 || held.RefreshError == "" || !held.FetchedAt.Equal(at) {
		t.Fatalf("held or dirty lost: %+v %d", held, changed)
	}
}

func TestFreeAgentRestoreScopeAndFirstGoodCopy(t *testing.T) {
	t.Parallel()
	calls := 0
	body := []byte(`{"freeAgents":{"leagueUnit":{"player":[{"id":"0100"},{"id":"0099"}]}}}`)
	bodies := func(_ context.Context, _ string, accept func(string, []byte, time.Time) bool) (bool, error) {
		calls++
		base := "https://fixture.invalid/2026/export?TYPE=freeAgents&L=" + ingestion.LeagueID
		if accept(base+"x", body, time.Now()) || accept(base, []byte(`{`), time.Now()) {
			t.Fatal("accepted foreign or invalid archive")
		}
		return accept(base, body, time.Now()), nil
	}
	published := 0
	publish := func(ids []playerid.PlayerID, _ time.Time) {
		published++
		if len(ids) != 2 || ids[0].String() != "0099" {
			t.Fatal(ids)
		}
	}
	RestoreFreeAgents(t.Context(), true, 2026, bodies, publish, nil)
	if calls != 0 {
		t.Fatal("held restore read archive")
	}
	RestoreFreeAgents(t.Context(), false, 2026, bodies, publish, func(string) { t.Fatal("good archive logged failure") })
	if calls != 1 || published != 1 {
		t.Fatal("restore did not publish first good copy")
	}
}

func TestPoolRefreshAdmissionAndFreshMarketPath(t *testing.T) {
	t.Parallel()
	failure := errors.New("startup")
	if err := RefreshPool(false, func() error { return failure }, nil, nil, nil, nil); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	at := time.Now()
	refreshes, markets := 0, 0
	for _, force := range []bool{false, true} {
		err := RefreshPool(force, func() error { return nil }, func() (time.Time, string) { return at, "" }, func(time.Time) time.Duration { return time.Minute }, func() error { markets++; return failure }, func() { refreshes++ })
		if (!force && err != failure) || (force && err != nil) {
			t.Fatal(err)
		}
	}
	if refreshes != 1 || markets != 1 {
		t.Fatalf("refreshes %d market %d", refreshes, markets)
	}
}
