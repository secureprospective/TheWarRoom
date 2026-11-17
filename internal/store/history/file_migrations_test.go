package history

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func TestHostedFilesMigrateOnlyOwnedTables(t *testing.T) {
	leagueTables := []string{"league_event", "authority_policy", "league_season_summary", "audit_chain", "audit_chain_head", "league_import", "transfer_identity", "transfer_transition", "server_release", "league_roster_restore", "league_recovery_log", "league_player", "player_ownership", "league_file", "private_entries"}
	for _, tc := range []struct {
		name          string
		version, base int
		steps         []db.Migration
		tables        []string
	}{
		{"league", LeagueSchemaVersion(), leagueMigrationBase, leagueMigrations(), leagueTables},
		{"reference", ReferenceSchemaVersion(), 0, referenceMigrations(), append(ReferenceTables(), "server_release")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tc.name+".db")
			pools, err := db.Open(t.Context(), path, tc.version)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = pools.Close() }()
			for range 2 {
				if _, err := pools.MigrateFrom(t.Context(), tc.base, tc.steps...); err != nil {
					t.Fatal(err)
				}
			}
			rows, err := pools.Read().QueryContext(t.Context(), `SELECT name FROM sqlite_schema WHERE type='table' ORDER BY name`)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = rows.Close() }()
			var got []string
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					t.Fatal(err)
				}
				got = append(got, name)
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			slices.Sort(tc.tables)
			if !slices.Equal(got, tc.tables) {
				t.Fatalf("tables %v, want %v", got, tc.tables)
			}
			var version int
			if err := pools.Read().QueryRowContext(t.Context(), `PRAGMA user_version`).Scan(&version); err != nil || version != tc.version {
				t.Fatalf("version %d, want %d: %v", version, tc.version, err)
			}
			if _, err := pools.Write().ExecContext(t.Context(), `PRAGMA user_version=999`); err != nil {
				t.Fatal(err)
			}
			if _, err := pools.MigrateFrom(t.Context(), tc.base, tc.steps...); err == nil {
				t.Fatal("migrated a newer file")
			}
			if newer, err := db.Open(t.Context(), path, tc.version); err == nil {
				_ = newer.Close()
				t.Fatal("opened a newer file")
			}
		})
	}
}

func TestPrivateEntriesMigrationUpgradesOldAndFreshLeagues(t *testing.T) {
	for _, tc := range []struct {
		name   string
		legacy bool
	}{
		{name: "schema 15", legacy: true},
		{name: "fresh", legacy: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pools := privateEntriesLeagueFixture(t, tc.legacy)
			if tc.legacy {
				if found, err := db.HasTable(t.Context(), pools, "private_entries"); err != nil || found {
					t.Fatalf("private table before migration = %v, %v", found, err)
				}
			}
			if err := MigrateLeague(t.Context(), pools); err != nil {
				t.Fatal(err)
			}
			assertPrivateEntriesSchema(t, pools)
			if tc.legacy {
				var league string
				if err := pools.Read().QueryRowContext(t.Context(), `SELECT league FROM league_file WHERE singleton=1`).Scan(&league); err != nil || league != "native.test" {
					t.Fatalf("existing league identity = %q, %v", league, err)
				}
			}
			var before int
			if err := pools.Read().QueryRowContext(t.Context(), `PRAGMA schema_version`).Scan(&before); err != nil {
				t.Fatal(err)
			}
			if err := MigrateLeague(t.Context(), pools); err != nil {
				t.Fatalf("rerun migration: %v", err)
			}
			var after int
			if err := pools.Read().QueryRowContext(t.Context(), `PRAGMA schema_version`).Scan(&after); err != nil || after != before {
				t.Fatalf("rerun changed schema version %d -> %d: %v", before, after, err)
			}
		})
	}
}

func privateEntriesLeagueFixture(t *testing.T, legacy bool) *db.Pools {
	t.Helper()
	ctx := t.Context()
	dir := t.TempDir()
	reference, err := db.Open(ctx, filepath.Join(dir, "reference.db"), ReferenceSchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := reference.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := PrepareReferenceFile(ctx, reference); err != nil {
		t.Fatal(err)
	}
	pools, err := db.OpenWithReference(ctx, filepath.Join(dir, "league.db"), LeagueSchemaVersion(), reference)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	if !legacy {
		if err := CreateLeagueFile(ctx, pools, LeagueFile{League: "native.test"}); err != nil {
			t.Fatal(err)
		}
		return pools
	}
	if _, err := pools.MigrateFrom(ctx, leagueMigrationBase, createLeagueFile, addLeagueIdentity); err != nil {
		t.Fatal(err)
	}
	tx, err := pools.Write().BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := claimLeagueFile(ctx, tx, LeagueFile{League: "native.test"}); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return pools
}

func assertPrivateEntriesSchema(t *testing.T, pools *db.Pools) {
	t.Helper()
	var version int
	if err := pools.Read().QueryRowContext(t.Context(), `PRAGMA user_version`).Scan(&version); err != nil || version != LeagueSchemaVersion() {
		t.Fatalf("league schema version = %d, want %d: %v", version, LeagueSchemaVersion(), err)
	}
	rows, err := pools.Read().QueryContext(t.Context(), `SELECT name, "notnull" FROM pragma_table_info('private_entries') ORDER BY cid`)
	if err != nil {
		t.Fatal(err)
	}
	var columns []string
	franchiseRequired := false
	for rows.Next() {
		var name string
		var notNull int
		if err := rows.Scan(&name, &notNull); err != nil {
			_ = rows.Close()
			t.Fatal(err)
		}
		columns = append(columns, name)
		franchiseRequired = franchiseRequired || name == "franchise_id" && notNull == 1
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		t.Fatal(err)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(columns, []string{"franchise_id", "mine", "kind", "entity", "entity_id", "text", "at", "ordinal"}) || !franchiseRequired {
		t.Fatalf("private_entries columns = %v, franchise_id required = %v", columns, franchiseRequired)
	}
	rows, err = pools.Read().QueryContext(t.Context(), `PRAGMA index_info(private_entries_franchise_kind)`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	columns = nil
	for rows.Next() {
		var seq, cid int
		var column string
		if err := rows.Scan(&seq, &cid, &column); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(columns, []string{"franchise_id", "kind"}) {
		t.Fatalf("private_entries index columns = %v", columns)
	}
	if found, err := db.HasTable(t.Context(), pools, "personal_entries"); err != nil || found {
		t.Fatalf("league file contains desktop personal_entries = %v, %v", found, err)
	}
}
