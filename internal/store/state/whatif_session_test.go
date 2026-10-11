package state

import (
	"context"
	"path/filepath"
	"testing"
)

func TestWhatIfMirrorStampsOptionalLedgersAndNanoseconds(t *testing.T) {
	ctx := context.Background()
	m := openMirror(t, filepath.Join(t.TempDir(), "mirror.db"))
	if _, err := m.Replace(ctx, mirrorSnap()); err != nil {
		t.Fatal(err)
	}
	initial, err := m.Stamps(ctx)
	if err != nil || initial.Rosters == "" || initial.Contract != "" || initial.DeadCap != "" {
		t.Fatalf("initial stamps %+v %v", initial, err)
	}
	for _, query := range []string{
		`CREATE TABLE contract_years(last_updated TEXT)`,
		`INSERT INTO contract_years VALUES ('2026-10-10T10:00:00Z')`,
		`CREATE TABLE dead_cap_ledger(created_at TEXT)`,
		`INSERT INTO dead_cap_ledger VALUES ('2026-10-10T11:00:00Z')`,
	} {
		if _, err := m.pools.Write().ExecContext(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	newest, err := m.Stamps(ctx)
	if err != nil || newest.Contract != "2026-10-10T10:00:00Z" || newest.DeadCap != "2026-10-10T11:00:00Z" || !newest.NewerThan(initial) {
		t.Fatalf("new stamps %+v %v", newest, err)
	}
	if newest.NewerThan(newest) {
		t.Fatal("identical stamps stale")
	}
	for _, current := range []MirrorStamps{
		{Rosters: "2026-10-10T10:00:00.000000001Z"},
		{Contract: "2026-10-10T10:00:00.000000001Z"},
		{DeadCap: "2026-10-10T10:00:00.000000001Z"},
	} {
		old := MirrorStamps{Rosters: "2026-10-10T10:00:00Z", Contract: "2026-10-10T10:00:00Z", DeadCap: "2026-10-10T10:00:00Z"}
		if !current.NewerThan(old) {
			t.Fatalf("same-second change missed %+v", current)
		}
	}
}

func TestReleaseStampsAreScopedToPlayerAndFranchise(t *testing.T) {
	ctx := context.Background()
	m := openMirror(t, filepath.Join(t.TempDir(), "scoped.db"))
	if _, err := m.Replace(ctx, mirrorSnap()); err != nil {
		t.Fatal(err)
	}
	if _, err := m.pools.Write().ExecContext(ctx, `
 CREATE TABLE contract_years (mfl_id TEXT,last_updated TEXT);
 INSERT INTO contract_years VALUES ('0042','2026-01-01T00:00:00Z'),('0099','2026-12-01T00:00:00Z');
 CREATE TABLE dead_cap_ledger (franchise_id TEXT,created_at TEXT);
 INSERT INTO dead_cap_ledger VALUES ('0001','2026-02-01T00:00:00Z'),('0002','2026-12-01T00:00:00Z');`); err != nil {
		t.Fatal(err)
	}
	stamps, err := m.ReleaseStamps(ctx, "0042", "0001")
	if err != nil || stamps.Contract != "2026-01-01T00:00:00Z" || stamps.DeadCap != "2026-02-01T00:00:00Z" {
		t.Fatal(stamps, err)
	}
	missing, err := m.ReleaseStamps(ctx, "0055", "0003")
	if err != nil || missing.Contract != "" || missing.DeadCap != "" {
		t.Fatal(missing, err)
	}
}
