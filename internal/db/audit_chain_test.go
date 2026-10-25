package db

import (
	"path/filepath"
	"testing"
)

func TestAuditHashV1(t *testing.T) {
	digest := auditDigest("payload")
	got := auditHash("league_event", 1, "event.1", digest, "")
	if got != "d6c092670a1371083941d983dbb3951a870e6879893fbbbc33e263023c881517" {
		t.Fatalf("v1 checkpoint format changed: %s", got)
	}
	for _, changed := range []string{
		auditHash("account_security_audit", 1, "event.1", digest, ""),
		auditHash("league_event", 2, "event.1", digest, ""),
		auditHash("league_event", 1, "event.2", digest, ""),
		auditHash("league_event", 1, "event.1", auditDigest("changed"), ""),
		auditHash("league_event", 1, "event.1", digest, got),
	} {
		if changed == got {
			t.Fatal("hash did not bind every field")
		}
	}
	if auditHash("a\x00b", 1, "c", digest, "") == auditHash("a", 1, "b\x00c", digest, "") {
		t.Fatal("ambiguous hash field framing")
	}
}

func TestSavedCheckpointDetectsCoherentRewriteAndRollback(t *testing.T) {
	for _, attack := range []string{"rewrite", "rollback"} {
		t.Run(attack, func(t *testing.T) {
			ctx := t.Context()
			pools, err := Open(ctx, filepath.Join(t.TempDir(), "audit.db"), 1)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = pools.Close() })
			tx, err := pools.Write().BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback() }()
			if _, err := tx.ExecContext(ctx, `CREATE TABLE events (id TEXT PRIMARY KEY, payload TEXT NOT NULL)`); err != nil {
				t.Fatal(err)
			}
			query := `SELECT id,payload FROM events ORDER BY rowid`
			if err := InitializeAudit(ctx, tx, "league_event", query); err != nil {
				t.Fatal(err)
			}
			for _, id := range []string{"first", "last"} {
				if _, err := tx.ExecContext(ctx, `INSERT INTO events VALUES (?,?)`, id, id); err != nil {
					t.Fatal(err)
				}
				if err := AppendAudit(ctx, tx, "league_event", id, id); err != nil {
					t.Fatal(err)
				}
			}
			anchor, err := VerifyAudit(ctx, tx, "league_event", query)
			if err != nil {
				t.Fatal(err)
			}
			if err := CheckAuditAnchor(ctx, tx, anchor, anchor); err != nil {
				t.Fatal(err)
			}
			if _, err := tx.ExecContext(ctx, `DROP TRIGGER audit_chain_no_delete; DROP TRIGGER audit_chain_no_update`); err != nil {
				t.Fatal(err)
			}
			if attack == "rollback" {
				if _, err := tx.ExecContext(ctx, `DELETE FROM events WHERE id='last'; DELETE FROM audit_chain WHERE sequence=2; UPDATE audit_chain_head SET size=1,hash=(SELECT hash FROM audit_chain WHERE sequence=1)`); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := tx.ExecContext(ctx, `UPDATE events SET payload='rewritten' WHERE id='first'`); err != nil {
					t.Fatal(err)
				}
				previous := ""
				for i, payload := range []string{"rewritten", "last"} {
					digest := auditDigest(payload)
					hash := auditHash("league_event", int64(i+1), []string{"first", "last"}[i], digest, previous)
					if _, err := tx.ExecContext(ctx, `UPDATE audit_chain SET digest=?,previous=?,hash=? WHERE sequence=?`, digest, previous, hash, i+1); err != nil {
						t.Fatal(err)
					}
					previous = hash
				}
				if _, err := tx.ExecContext(ctx, `UPDATE audit_chain_head SET hash=?`, previous); err != nil {
					t.Fatal(err)
				}
			}
			current, err := VerifyAudit(ctx, tx, "league_event", query)
			if err != nil {
				t.Fatalf("attack should evade purely local verification: %v", err)
			}
			if err := CheckAuditAnchor(ctx, tx, current, anchor); err == nil {
				t.Fatal("saved checkpoint failed to detect attack")
			}
		})
	}
}
