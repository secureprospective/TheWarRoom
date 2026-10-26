package db

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestMainEveryMigrationPrefixConverges(t *testing.T) {
	t.Parallel()
	var expected string
	for version := 0; version <= MainVersion(); version++ {
		t.Run(fmt.Sprintf("v%d", version), func(t *testing.T) {
			pools, err := Open(t.Context(), filepath.Join(t.TempDir(), "main.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = pools.Close() }()
			if _, err := pools.Migrate(t.Context(), mainMigrations()[:version]...); err != nil {
				t.Fatal(err)
			}
			if err := pools.InitializeMain(t.Context()); err != nil {
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
			assertVersion(t, pools, MainVersion())
			if err := pools.InitializeMain(t.Context()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
