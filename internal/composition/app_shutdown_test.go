package composition

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func TestShutdownCancelsJoinsThenClosesDatabases(t *testing.T) {
	a := NewApp()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	a.core.WorkCtx, a.core.WorkCancel = ctx, cancel
	for i, dest := range []**db.Pools{&a.core.Pools, &a.core.HistPools, &a.core.WhatifPools} {
		p, err := db.Open(t.Context(), filepath.Join(t.TempDir(), []string{"main.db", "history.db", "whatif.db"}[i]))
		if err != nil {
			t.Fatal(err)
		}
		*dest = p
	}
	result := make(chan error, 1)
	a.startWorker(func() {
		<-ctx.Done()
		_, err := a.core.HistPools.Write().ExecContext(context.WithoutCancel(ctx), `CREATE TABLE worker_finished (id INTEGER)`)
		result <- err
	})
	a.shutdown(context.Background())
	if err := <-result; err != nil {
		t.Fatalf("database closed before worker finished: %v", err)
	}
	for _, p := range []*db.Pools{a.core.Pools, a.core.HistPools, a.core.WhatifPools} {
		if err := p.Read().PingContext(context.Background()); err == nil {
			t.Fatal("database left open")
		}
	}
	a.startWorker(func() { t.Error("worker started after shutdown") })
	a.shutdown(context.Background())
}
