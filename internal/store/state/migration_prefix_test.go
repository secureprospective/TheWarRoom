package state

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func TestWhatIfEveryMigrationPrefixConverges(t *testing.T) {
	t.Parallel()
	var expected string
	for version := 0; version <= SchemaVersion(); version++ {
		t.Run(fmt.Sprintf("v%d", version), func(t *testing.T) {
			pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "whatif.db"), SchemaVersion())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = pools.Close() }()
			if _, err := pools.Migrate(t.Context(), whatIfMigrations()[:version]...); err != nil {
				t.Fatal(err)
			}
			if _, err := pools.Migrate(t.Context(), whatIfMigrations()...); err != nil {
				t.Fatal(err)
			}
			var got string
			err = pools.Read().QueryRowContext(t.Context(), `SELECT json_group_array(json_array(type,name,tbl_name,sql)) FROM (SELECT type,name,tbl_name,sql FROM sqlite_schema WHERE name NOT LIKE 'sqlite_%' ORDER BY type,name)`).Scan(&got)
			if err != nil {
				t.Fatal(err)
			}
			if version == 0 {
				expected = got
			} else if got != expected {
				t.Fatalf("schema differs after upgrading v%d", version)
			}
			var stamped int
			if err := pools.Read().QueryRowContext(t.Context(), "PRAGMA user_version").Scan(&stamped); err != nil || stamped != SchemaVersion() {
				t.Fatalf("version=%d: %v", stamped, err)
			}
			if _, err := pools.Migrate(t.Context(), whatIfMigrations()...); err != nil {
				t.Fatal(err)
			}
		})
	}
}
