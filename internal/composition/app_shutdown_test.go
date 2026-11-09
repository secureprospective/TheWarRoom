package composition

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func TestShutdownWaitsForStartupWorker(t *testing.T) {
	a := NewApp()
	a.core.StartupErr = errors.New("fixture startup stopped")
	a.startWorker(func() { _ = a.ready() })
	done := make(chan struct{})
	go func() { a.shutdown(context.Background()); close(done) }()
	select {
	case <-done:
		close(a.core.Started)
		t.Fatal("shutdown returned with a worker still waiting for startup")
	case <-time.After(30 * time.Millisecond):
	}
	close(a.core.Started)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not join the worker")
	}
}

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
