package history

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func TestHostedFilesMigrateOnlyOwnedTables(t *testing.T) {
	leagueTables := []string{"league_event", "authority_policy", "league_season_summary", "audit_chain", "audit_chain_head", "league_import", "transfer_identity", "transfer_transition", "server_release", "league_roster_restore", "league_recovery_log", "league_player", "player_ownership"}
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
