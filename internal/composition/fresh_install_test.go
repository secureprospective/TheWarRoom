package composition

import (
	"context"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type freshInstallTransport struct {
	moveFeedTransport
	draftPaths []string
}

func (f *freshInstallTransport) RoundTrip(r *http.Request) (*http.Response, error) {
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
	for _, export := range []string{"draftResults", "futureDraftPicks"} {
		if !slices.Contains(transport.requests, export) {
			t.Fatalf("did not exercise %s failure: %v", export, transport.requests)
		}
	}
}
