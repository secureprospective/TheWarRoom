package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestMainSchemaAdoptsAllStoresOnceWithDataIntact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "thewarroom.db")
	fixture, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	// Existing main files have independent stores; one file migration must own all their DDL.
	if _, err := fixture.ExecContext(t.Context(), mainMirrorDDL+`INSERT INTO league_mirror VALUES (1,2026,'held','{"season":2026}','2026-10-01');`); err != nil {
		t.Fatal(err)
	}
	if err := fixture.Close(); err != nil {
		t.Fatal(err)
	}
	pools, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	if err := pools.InitializeMain(t.Context()); err != nil {
		t.Fatal(err)
	}
	assertVersion(t, pools, MainVersion())
	var raw string
	if err := pools.Read().QueryRowContext(t.Context(), "SELECT snapshot FROM league_mirror").Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if raw != "{\"season\":2026}" {
		t.Fatalf("adoption changed mirror: %s", raw)
	}
	for _, name := range []string{"rulebook_versions", "param_defaults", "move_envelopes", "bid_facts", "release_facts", "contract_facts", "personal_entries", "personal_offers", "league_mirror"} {
		var count int
		if err := pools.Read().QueryRowContext(t.Context(), "SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("missing main table %s", name)
		}
	}
	first := pools.MigrationFindings()
	if first.Backup != path+".pre-v0.bak" {
		t.Fatalf("backup=%q", first.Backup)
	}
	if err := pools.InitializeMain(t.Context()); err != nil {
		t.Fatal(err)
	}
	if pools.MigrationFindings().Backup != "" {
		t.Fatal("second initializer reran file migration")
	}
}
