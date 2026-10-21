// Package coretest builds isolated application state without Wails or feature packages.
package coretest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type Builder struct{ state *core.State }

func New(t *testing.T) *Builder {
	t.Helper()
	ctx := context.Background()
	open := func(name string) *db.Pools {
		t.Helper()
		pools, err := db.Open(ctx, filepath.Join(t.TempDir(), name))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := pools.Close(); err != nil {
				t.Error(err)
			}
		})
		return pools
	}
	p, hp := open("league.db"), open("history.db")
	rb := rulebook.New(p)
	mirror := state.NewMirror(p, rb)
	if err := mirror.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	hist := history.New(hp, reg)
	if err := hist.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	close(started)
	s := &core.State{Ctx: ctx, Started: started, Season: 2026, Pools: p, HistPools: hp,
		Rulebook: rb, League: mirror, History: hist, KeyStore: mflkey.NewMemory(ingestion.LeagueID, "2026"),
		LocalZone: time.UTC, FallbackLog: func(string) {}, SeasonPause: func(ctx context.Context, _ time.Duration) error { return ctx.Err() },
		RefreshTransactions: func(context.Context) {}, OpenURL: func(context.Context, string) {}}
	return &Builder{state: s}
}

func (b *Builder) Build() *core.State { return b.state }
