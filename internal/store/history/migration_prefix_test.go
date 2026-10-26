package history

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

func TestHistoryEveryMigrationPrefixConverges(t *testing.T) {
	t.Parallel()
	var expected string
	for version := 0; version <= SchemaVersion(); version++ {
		t.Run(fmt.Sprintf("v%d", version), func(t *testing.T) {
			pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "history.db"), SchemaVersion())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = pools.Close() }()
			if _, err := pools.Migrate(t.Context(), historyMigrations()[:version]...); err != nil {
				t.Fatal(err)
			}
			store := New(pools, registry(t, ""))
			if err := store.Initialize(t.Context()); err != nil {
				t.Fatal(err)
			}
			got := historyJSON(t, pools.Read(), `SELECT json_group_array(json_array(type,name,tbl_name,sql)) FROM (SELECT type,name,tbl_name,sql FROM sqlite_schema WHERE name NOT LIKE 'sqlite_%' ORDER BY type,name)`)
			if version == 0 {
				expected = got
			} else if got != expected {
				t.Fatalf("schema differs after upgrading v%d", version)
			}
			var stamped int
			if err := pools.Read().QueryRowContext(t.Context(), "PRAGMA user_version").Scan(&stamped); err != nil || stamped != SchemaVersion() {
				t.Fatalf("version=%d: %v", stamped, err)
			}
			if err := store.Initialize(t.Context()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
