package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenKeepsPathsWithURICharactersApart(t *testing.T) {
	for _, name := range []string{"Bob#2", "what?", "100%", "plain"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), name)
			if err := os.Mkdir(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			for _, file := range []string{"a.db", "b.db"} {
				p, err := Open(t.Context(), filepath.Join(dir, file))
				if err != nil {
					t.Fatal(err)
				}
				if _, err := p.Write().ExecContext(t.Context(), "CREATE TABLE marker (value TEXT)"); err != nil {
					t.Fatalf("%s shares a database with another file: %v", file, err)
				}
				if v, err := fileUserVersion(t.Context(), filepath.Join(dir, file)); err != nil || v != 0 {
					t.Fatalf("probe %s: %d %v", file, v, err)
				}
				if err := p.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
