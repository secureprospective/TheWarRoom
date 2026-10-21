package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"

	"github.com/secureprospective/TheWarRoom/internal/app/core"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func TestDatabaseChecksReportAllFilesAndFailures(t *testing.T) {
	a := &App{raceHeld: &raceapp.Held{}, core: &core.State{}}
	for _, file := range []struct {
		dest **db.Pools
		name string
	}{{&a.core.Pools, "main"}, {&a.core.HistPools, "history"}, {&a.core.WhatifPools, "whatif"}} {
		p, err := db.Open(t.Context(), filepath.Join(t.TempDir(), file.name+".db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = p.Close() })
		*file.dest = p
	}
	if r := a.databaseChecksReport(); r.Status != "empty" || len(r.Rows) != 3 {
		t.Fatalf("unchecked: %+v", r)
	}
	at := time.Now().UTC()
	a.checkDatabases(t.Context(), at)
	r := a.databaseChecksReport()
	if r.Status != "ok" {
		t.Fatalf("healthy: %+v", r)
	}
	for _, row := range r.Rows {
		if row.Cells[2] != "OK" {
			t.Fatalf("unchecked file: %+v", row)
		}
	}
	if err := a.core.HistPools.Close(); err != nil {
		t.Fatal(err)
	}
	a.checkDatabases(t.Context(), at.Add(7*24*time.Hour))
	r = a.databaseChecksReport()
	if r.Status != "failed" || !strings.Contains(r.Rows[1].Cells[2], "closed") || r.Rows[2].Cells[2] != "OK" {
		t.Fatalf("failure hidden or skipped another file: %+v", r)
	}
	body, err := json.Marshal(sourcesapp.DataSources{DatabaseChecks: a.databaseChecksReading()})
	if err != nil || !strings.Contains(string(body), `"databaseChecks":`) {
		t.Fatalf("reading field missing: %s %v", body, err)
	}
}

func TestDatabaseChecksWorkerJoinsOnShutdown(t *testing.T) {
	a := NewApp()
	ctx, cancel := context.WithCancel(t.Context())
	a.core.WorkCtx, a.core.WorkCancel = ctx, cancel
	a.core.Ctx, a.core.Started = ctx, make(chan struct{})
	close(a.core.Started)
	a.startDatabaseChecks()
	a.startDatabaseChecks()
	done := make(chan struct{})
	go func() { a.shutdown(ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("database checks worker did not join")
	}
}

func TestDataSourcesIncludesLocalDatabaseChecks(t *testing.T) {
	a := targetTestApp(t)
	data, err := sourcesapp.New(a.core, a.sourcesHeld).TargetDataSources()
	if err != nil {
		t.Fatal(err)
	}
	if data.DatabaseChecks == nil || len(data.DatabaseChecks.Rows) != 3 {
		t.Fatalf("database checks missing from reading: %+v", data.DatabaseChecks)
	}
}
