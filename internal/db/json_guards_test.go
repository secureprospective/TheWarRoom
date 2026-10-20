package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestJSONGuardsCoverInsertAndUpdateAfterMigration(t *testing.T) {
	p, err := Open(t.Context(), filepath.Join(t.TempDir(), "guards.db"), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	first := func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `CREATE TABLE payloads (id INTEGER PRIMARY KEY, body TEXT); INSERT INTO payloads VALUES(1, '{}')`)
		return err
	}
	if _, err := p.Migrate(t.Context(), first); err != nil {
		t.Fatal(err)
	}
	next := func(ctx context.Context, tx *sql.Tx) error {
		return GuardJSON(ctx, tx, JSONColumn{Table: "payloads", Column: "body", Versioned: true})
	}
	if _, err := p.Migrate(t.Context(), first, next); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []*string{nil, strPointer(`{}`), strPointer(`{"version":null}`), strPointer(`{"version":"1"}`), strPointer(`not json`)} {
		if _, err := p.Write().ExecContext(t.Context(), `INSERT INTO payloads(body) VALUES (?)`, raw); err == nil {
			t.Fatal("invalid JSON inserted")
		}
		if _, err := p.Write().ExecContext(t.Context(), `UPDATE payloads SET body=? WHERE id=1`, raw); err == nil {
			t.Fatal("invalid JSON updated")
		}
	}
	if _, err := p.Write().ExecContext(t.Context(), `UPDATE payloads SET body='{"version":1}' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
}

func strPointer(s string) *string { return &s }
