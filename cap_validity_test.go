package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestMissingPaidCellBlocksWhatIfAndReachesRosterReading(t *testing.T) {
	a := targetTestApp(t)
	a.hasLookup = true
	pools, err := db.Open(a.ctx, filepath.Join(t.TempDir(), "whatif.db"), state.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	a.whatifPools = pools
	snap, _, _ := a.league.ContractsView()
	session := state.WhatIfSession{Generation: "invalid-cap", Mine: "0001", Snapshot: snap, TaxiPercent: 100, IRPercent: 100}
	store, err := a.loadWhatIf(a.ctx, session)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pools.Write().ExecContext(a.ctx, `UPDATE contract_years SET year_status='VOID' WHERE league_id=? AND league_year=2026`, session.Generation); err != nil {
		t.Fatal(err)
	}
	if err := store.Initialize(a.ctx, state.SnapshotSource{Snapshot: snap}); err != nil {
		t.Fatal(err)
	}
	a.whatif = store
	reading := a.GetRoster("0001")
	if reading.OK || !strings.Contains(reading.Detail, "cap total invalid") {
		t.Fatalf("cap invalidity absent from roster reading: %+v", reading)
	}
	if _, err := a.loadWhatIf(a.ctx, session); err == nil || !strings.Contains(err.Error(), "cap total invalid") {
		t.Fatalf("what-if accepted invalid cap: %v", err)
	}
}
