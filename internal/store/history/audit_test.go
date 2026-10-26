package history

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestLedgerAuditDetectsTampering(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, sql string }{
		{"edit", `DROP TRIGGER league_event_no_update; UPDATE league_event SET actor='changed' WHERE id='first'`},
		{"delete", `DROP TRIGGER league_event_no_delete; DELETE FROM league_event WHERE id='first'`},
		{"truncate", `DROP TRIGGER league_event_no_delete; DELETE FROM league_event WHERE id='last'`},
		{"chain_tail", `DROP TRIGGER audit_chain_no_delete; DELETE FROM audit_chain WHERE sequence=2`},
		{"both_tails", `DROP TRIGGER league_event_no_delete; DROP TRIGGER audit_chain_no_delete; DELETE FROM league_event WHERE id='last'; DELETE FROM audit_chain WHERE sequence=2`},
		{"head", `UPDATE audit_chain_head SET hash='changed'`},
		{"missing_head", `DELETE FROM audit_chain_head`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s, _, p := newStore(t)
			for i, id := range []string{"first", "last"} {
				if _, err := s.AppendEvent(t.Context(), eventFixture(id), int64(i)); err != nil {
					t.Fatal(err)
				}
			}
			head, err := s.VerifyAudit(t.Context())
			if err != nil || head.Size != 2 || head.Version != 1 || len(head.Hash) != 64 {
				t.Fatalf("head %+v: %v", head, err)
			}
			if _, err := p.Write().ExecContext(t.Context(), tc.sql); err != nil {
				t.Fatal(err)
			}
			if _, err := s.VerifyAudit(t.Context()); err == nil {
				t.Fatal("tampering not detected")
			}
			if err := New(p, s.Registry()).Initialize(t.Context()); err == nil {
				t.Fatal("damaged history admitted at startup")
			}
		})
	}
}

func TestLedgerAuditFailureRollsBackPrivilegedMutation(t *testing.T) {
	t.Parallel()
	for _, failure := range []string{
		`CREATE TRIGGER fail_audit BEFORE INSERT ON audit_chain BEGIN SELECT RAISE(ABORT,'audit unavailable'); END`,
		`CREATE TRIGGER fail_audit BEFORE UPDATE ON audit_chain_head BEGIN SELECT RAISE(ABORT,'head unavailable'); END`,
		`CREATE TRIGGER fail_audit BEFORE INSERT ON audit_chain BEGIN SELECT RAISE(IGNORE); END`,
		`CREATE TRIGGER fail_audit BEFORE UPDATE ON audit_chain_head BEGIN SELECT RAISE(IGNORE); END`,
	} {
		s, _, p := newStore(t)
		before, err := s.VerifyAudit(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.Write().ExecContext(t.Context(), failure); err != nil {
			t.Fatal(err)
		}
		if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err == nil {
			t.Fatal("unaudited privileged change accepted")
		}
		events, err := s.Events(t.Context(), "native.test", commissioner.Stream, 0)
		if err != nil || len(events) != 0 {
			t.Fatalf("failed audit appended history: %+v %v", events, err)
		}
		after, err := s.VerifyAudit(t.Context())
		if err != nil || after != before {
			t.Fatalf("failed audit changed head: %+v %v", after, err)
		}
		if _, err := p.Write().ExecContext(t.Context(), `DROP TRIGGER fail_audit`); err != nil {
			t.Fatal(err)
		}
		if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLedgerAuditAdoptsExistingHistoryWithoutChangingReceipts(t *testing.T) {
	t.Parallel()
	s, _, p := newStore(t)
	receipt, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	var original string
	if err := p.Read().QueryRowContext(t.Context(), `SELECT payload FROM league_event`).Scan(&original); err != nil {
		t.Fatal(err)
	}
	// Reproduce the pre-chain schema, leaving the original event and receipt untouched.
	if _, err := p.Write().ExecContext(t.Context(), `DROP TABLE audit_chain; DROP TABLE audit_chain_head;
DROP TRIGGER league_import_event_guard;
DROP TRIGGER league_import_authority_insert;
DROP TRIGGER league_import_authority_update;
DROP TABLE league_import;
DROP VIEW transfer_current; DROP TABLE transfer_transition; DROP TABLE transfer_identity;
DROP TABLE server_release;
DROP TABLE league_roster_restore; DROP TABLE league_recovery_log;
DROP TRIGGER observations_player_identity; DROP TRIGGER run_scores_player_identity; DROP TRIGGER model_scores_player_identity;
CREATE TABLE legacy_player_ids AS SELECT * FROM player_ids;
DROP VIEW player_ids;
ALTER TABLE legacy_player_ids RENAME TO player_ids;
DROP TABLE player_ownership; DROP TABLE player_external_id; DROP TABLE canonical_player;
PRAGMA user_version=4`); err != nil {
		t.Fatal(err)
	}
	reopened := New(p, s.Registry())
	if err := reopened.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	head, err := reopened.VerifyAudit(t.Context())
	if err != nil || head.Baseline != 1 || head.Size != 1 {
		t.Fatalf("baseline %+v: %v", head, err)
	}
	retry, err := reopened.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil)
	if err != nil || !reflect.DeepEqual(receipt, retry) {
		t.Fatalf("old receipt changed: %+v %v", retry, err)
	}
	var adopted string
	if err := p.Read().QueryRowContext(t.Context(), `SELECT payload FROM league_event`).Scan(&adopted); err != nil || adopted != original {
		t.Fatalf("adoption rewrote history: %v", err)
	}
	if err := reopened.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	again, err := reopened.VerifyAudit(t.Context())
	if err != nil || again != head {
		t.Fatalf("baseline changed on restart: %+v %v", again, err)
	}
	next := reverseCommand(receipt.Entry, receipt.Revision, "reverse")
	if _, err := reopened.ExecuteCommissioner(t.Context(), next, "creator", nil); err != nil {
		t.Fatal(err)
	}
	tail, err := reopened.VerifyAudit(t.Context())
	if err != nil || tail.Size != 2 || tail.Baseline != 1 {
		t.Fatalf("adopted tail %+v: %v", tail, err)
	}
}
