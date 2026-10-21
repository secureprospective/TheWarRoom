package main

import (
	"path/filepath"
	"strings"
	"testing"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestMissingPaidCellBlocksWhatIf(t *testing.T) {
	a := targetTestApp(t)
	a.core.HasLookup = true
	pools, err := db.Open(a.core.Ctx, filepath.Join(t.TempDir(), "whatif.db"), state.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	a.core.WhatifPools = pools
	snap, _, _ := a.core.League.ContractsView()
	session := state.WhatIfSession{Generation: "invalid-cap", Mine: "0001", Snapshot: snap, TaxiPercent: 100, IRPercent: 100}
	store, err := contractsapp.LoadWhatIf(a.core.Ctx, a.core.WhatifPools, session)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pools.Write().ExecContext(a.core.Ctx, `UPDATE contract_years SET year_status='VOID' WHERE league_id=? AND league_year=2026`, session.Generation); err != nil {
		t.Fatal(err)
	}
	if err := store.Initialize(a.core.Ctx, state.SnapshotSource{Snapshot: snap}); err != nil {
		t.Fatal(err)
	}
	if _, err := contractsapp.LoadWhatIf(a.core.Ctx, a.core.WhatifPools, session); err == nil || !strings.Contains(err.Error(), "cap total invalid") {
		t.Fatalf("what-if accepted invalid cap: %v", err)
	}
}
