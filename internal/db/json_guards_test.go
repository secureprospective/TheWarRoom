package db

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
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

// A pre-v2 main file can already hold guard triggers under the v2 names, in an older shape
// that lets a NULL verdict through; v2 must replace them inside its transaction.
func TestMainV2ReplacesOlderJSONGuards(t *testing.T) {
	path := filepath.Join(t.TempDir(), "thewarroom.db")
	fixture, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	ddl := mainRulebookDDL + mainMovesDDL + mainMirrorDDL
	for _, c := range [][2]string{{"rulebook_versions", "payload"}, {"move_envelopes", "spec"}, {"league_mirror", "snapshot"}} {
		for _, op := range []string{"insert", "update"} {
			ddl += fmt.Sprintf("\nCREATE TRIGGER %[1]s_%[2]s_json_%[3]s BEFORE %[3]s ON %[1]s WHEN NOT json_valid(NEW.%[2]s) BEGIN SELECT RAISE(ABORT, 'old guard'); END;", c[0], c[1], op)
		}
	}
	if _, err := fixture.ExecContext(t.Context(), ddl+"\nPRAGMA user_version = 1;"); err != nil {
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
	var old int
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name LIKE '%_json_%' AND sql NOT LIKE '%COALESCE%'`).Scan(&old); err != nil || old != 0 {
		t.Fatalf("older guards left in force: %d %v", old, err)
	}
	if _, err := pools.Write().ExecContext(t.Context(), `INSERT INTO move_envelopes VALUES ('m', '2026-10-01', 'not json')`); err == nil || !strings.Contains(err.Error(), "invalid JSON payload") {
		t.Fatalf("v2 guard not in force: %v", err)
	}
}
