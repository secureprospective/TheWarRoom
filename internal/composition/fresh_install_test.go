package composition

import (
	"context"
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/mflref/moves"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type freshInstallTransport struct {
	moveFeedTransport
	draftPaths []string
}

func (f *freshInstallTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Query().Get("TYPE") == "rosters" {
		return seasonResponse(`{"rosters":{"franchise":{"id":"0001","player":
			{"id":"0042","status":"TAXI_SQUAD","salary":"1","contractYear":"2026"}}}}`), nil
	}
	if slices.Contains([]string{"draftResults", "futureDraftPicks"}, r.URL.Query().Get("TYPE")) {
		f.draftPaths = append(f.draftPaths, r.URL.Path)
	}
	return f.moveFeedTransport.RoundTrip(r)
}

func TestFreshInstallDraftFailureKeepsLeagueUsable(t *testing.T) {
	t.Parallel()
	a := NewApp()
	a.core.Ctx = t.Context()
	a.core.SeasonChanged = func(context.Context) {}
	a.core.SeasonPause = func(ctx context.Context, _ time.Duration) error { return ctx.Err() }
	transport := &freshInstallTransport{}
	var err error
	a.core.MFLClient, err = mfl.New("api", 10000, mfl.WithTransport(transport))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, file := range []struct {
		name    string
		version int
		pool    **db.Pools
	}{
		{"league.db", 0, &a.core.Pools},
		{"history.db", history.SchemaVersion(), &a.core.HistPools},
		{"whatif.db", state.SchemaVersion(), &a.core.WhatifPools},
	} {
		*file.pool, err = db.Open(t.Context(), filepath.Join(dir, file.name), file.version)
		if err != nil {
			t.Fatal(err)
		}
		pool := *file.pool
		t.Cleanup(func() {
			if err := pool.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	hist := history.New(a.core.HistPools, reg)
	a.attachSeat(hist)
	// A persisted Ready IR plan predates the first mirror, which has 0042 on taxi.
	a.core.Moves = moves.New(a.core.Pools)
	if err := a.core.Moves.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	ready := readyIRPlan(t, a, time.Now().UTC())
	refreshed, err := a.initStoreFloor(t.Context(), hist)
	if err != nil {
		t.Fatal(err)
	}
	if !refreshed || a.core.History != hist || a.core.Season != 2026 || a.core.League.Season() != 2026 || a.core.Whatif == nil || a.core.Coordinator == nil {
		t.Fatal("fresh store did not finish initialization")
	}
	players, _, _ := a.core.League.ContractsView()
	if len(players.Players) != 1 || players.Players[0].MFLID != "0042" {
		t.Fatalf("league roster lost after optional feed failure: %+v", players)
	}
	if len(transport.draftPaths) != 2 || transport.draftPaths[0] != "/2026/export" || transport.draftPaths[1] != "/2026/export" {
		t.Fatalf("draft feeds requested before season discovery: %v", transport.draftPaths)
	}
	results, future := a.core.DraftFeeds()
	if len(results.Value) != 0 || len(future.Value) != 0 || !results.FetchedAt.IsZero() || !future.FetchedAt.IsZero() {
		t.Fatal("unavailable draft feeds were presented as held data")
	}
	for _, failure := range []string{results.RefreshError, future.RefreshError} {
		if !strings.Contains(failure, "no archived copy") {
			t.Fatalf("missing honest unavailable-feed message: %q", failure)
		}
	}
	verifyFreshStartupCompletion(t, a, refreshed, ready)
	for _, export := range []string{"draftResults", "futureDraftPicks"} {
		if !slices.Contains(transport.requests, export) {
			t.Fatalf("did not exercise %s failure: %v", export, transport.requests)
		}
	}
}

func verifyFreshStartupCompletion(t *testing.T, a *App, refreshed bool, ready envelope.Envelope) {
	t.Helper()
	select {
	case <-a.core.Started:
		t.Fatal("readiness published before startup completion")
	default:
	}
	stored, err := a.core.Moves.Get(t.Context(), ready.ID())
	if err != nil || stored.State() != envelope.Ready {
		t.Fatalf("plan changed before readiness: %v %v", stored.State(), err)
	}
	health, revalidated := 0, 0
	a.core.SeasonChanged = func(context.Context) {
		if err := a.ready(); err != nil {
			t.Error(err)
		}
		health++
	}
	a.core.MovesChanged = func(context.Context) {
		if err := a.ready(); err != nil {
			t.Error(err)
		}
		revalidated++
	}
	done := make(chan struct{})
	go func() {
		a.finishStartup(t.Context(), refreshed)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("startup completion blocked on its own readiness")
	}
	stored, err = a.core.Moves.Get(t.Context(), ready.ID())
	if err != nil || stored.State() != envelope.Draft || health != 1 || revalidated != 1 {
		t.Fatalf("startup effects: state=%s health=%d revalidated=%d err=%v", stored.State(), health, revalidated, err)
	}
	if audit := stored.Receipt().Audit; len(audit) != 3 || !strings.Contains(audit[1].Note, "on the taxi squad") {
		t.Fatalf("plan not revalidated against startup mirror: %+v", audit)
	}
}

func TestStartupCompletionWithoutFreshMirror(t *testing.T) {
	t.Parallel()
	for _, failed := range []bool{false, true} {
		a := NewApp()
		if failed {
			a.core.StartupErr = errors.New("store floor failed")
		}
		health := 0
		a.core.SeasonChanged = func(context.Context) {
			select {
			case <-a.core.Started:
			default:
				t.Error("health update before startup publication")
			}
			health++
		}
		a.finishStartup(t.Context(), failed)
		if health != 1 {
			t.Fatalf("got %d health updates, want 1", health)
		}
		select {
		case <-a.core.Started:
		default:
			t.Fatal("startup did not signal readiness")
		}
		if err := a.ready(); (err != nil) != failed {
			t.Fatalf("startup failure not preserved: %v", err)
		}
	}
}

// refusedTransport fails every fetch the way a computer with no connection does, with no network.
type refusedTransport struct{ addr string }

func (r refusedTransport) RoundTrip(*http.Request) (*http.Response, error) {
	conn, err := net.Dial("tcp", r.addr)
	if err == nil {
		_ = conn.Close()
		return nil, errors.New("closed port answered")
	}
	return nil, err
}

// Issue 15: a first start that cannot reach MFL fails with the plain offline sentence, not the
// network chain, and every screen reads that sentence through ready.
func TestOfflineFirstStartFailsInPlainWords(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	a := restoreDrillApp(t, t.TempDir())
	a.core.MFLClient, err = mfl.New("api", 10000, mfl.WithTransport(refusedTransport{addr: addr}))
	if err != nil {
		t.Fatal(err)
	}
	_, a.core.StartupErr = a.initStoreFloor(t.Context(), a.core.History)
	if a.core.StartupErr == nil || !strings.Contains(a.core.StartupErr.Error(), addr) {
		t.Fatalf("store floor did not fail on the network: %v", a.core.StartupErr)
	}
	a.settleStartupFailure()
	if _, err := newCommissionerAPI(a).TargetCommissionerSeat(); err == nil || err.Error() != report.Offline {
		t.Fatalf("screens read: %v", err)
	}
}
