package accounts

import (
	"path/filepath"
	"testing"
)

func TestSecurityAuditDetectsTampering(t *testing.T) {
	credential := fixtureCredential(t)
	for _, tc := range []struct{ name, sql string }{
		{"edit", `DROP TRIGGER account_security_audit_no_update; UPDATE account_security_audit SET reason='changed' WHERE sequence=1`},
		{"delete", `DROP TRIGGER account_security_audit_no_delete; DELETE FROM account_security_audit WHERE sequence=1`},
		{"truncate", `DROP TRIGGER account_security_audit_no_delete; DELETE FROM account_security_audit WHERE sequence=2`},
		{"chain_edit", `DROP TRIGGER audit_chain_no_update; UPDATE audit_chain SET digest='changed' WHERE sequence=1`},
		{"both_tails", `DROP TRIGGER account_security_audit_no_delete; DROP TRIGGER audit_chain_no_delete; DELETE FROM account_security_audit WHERE sequence=2; DELETE FROM audit_chain WHERE sequence=2`},
		{"head", `UPDATE audit_chain_head SET size=0,hash=''`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, p := openStore(t, filepath.Join(t.TempDir(), "server.db"))
			if err := s.Insert(t.Context(), credential, true); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				if err := s.ResetAccount(t.Context(), credential.Username, credential.PasswordHash, "", "offline recovery"); err != nil {
					t.Fatal(err)
				}
			}
			head, err := s.VerifyAudit(t.Context())
			if err != nil || head.Size != 2 {
				t.Fatalf("head %+v: %v", head, err)
			}
			if _, err := p.Write().ExecContext(t.Context(), tc.sql); err != nil {
				t.Fatal(err)
			}
			if _, err := s.VerifyAudit(t.Context()); err == nil {
				t.Fatal("tampering not detected")
			}
			if err := New(p).Initialize(t.Context()); err == nil {
				t.Fatal("damaged audit admitted at startup")
			}
		})
	}
}

func TestSecurityChainFailureRollsBackRecovery(t *testing.T) {
	credential := fixtureCredential(t)
	replacement := fixtureCredential(t)
	for _, failure := range []string{
		`CREATE TRIGGER fail_audit BEFORE INSERT ON audit_chain BEGIN SELECT RAISE(ABORT,'audit unavailable'); END`,
		`CREATE TRIGGER fail_audit BEFORE UPDATE ON audit_chain_head BEGIN SELECT RAISE(ABORT,'head unavailable'); END`,
		`CREATE TRIGGER fail_audit BEFORE INSERT ON audit_chain BEGIN SELECT RAISE(IGNORE); END`,
		`CREATE TRIGGER fail_audit BEFORE UPDATE ON audit_chain_head BEGIN SELECT RAISE(IGNORE); END`,
	} {
		s, p := openStore(t, filepath.Join(t.TempDir(), "server.db"))
		if err := s.Insert(t.Context(), credential, true); err != nil {
			t.Fatal(err)
		}
		before, err := s.VerifyAudit(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.Write().ExecContext(t.Context(), failure); err != nil {
			t.Fatal(err)
		}
		if err := s.ResetAccount(t.Context(), credential.Username, replacement.PasswordHash, "", "offline recovery"); err == nil {
			t.Fatal("unaudited reset accepted")
		}
		stored, err := s.Credential(t.Context(), credential.Username)
		if err != nil || stored != credential {
			t.Fatalf("failed audit changed password: %v", err)
		}
		generation, err := s.AuthGeneration(t.Context(), credential.ID)
		if err != nil || generation != 0 {
			t.Fatalf("failed audit changed generation: %d %v", generation, err)
		}
		after, err := s.VerifyAudit(t.Context())
		if err != nil || after != before {
			t.Fatalf("failed audit changed head: %+v %v", after, err)
		}
		if _, err := p.Write().ExecContext(t.Context(), `DROP TRIGGER fail_audit`); err != nil {
			t.Fatal(err)
		}
		if err := s.ResetAccount(t.Context(), credential.Username, replacement.PasswordHash, "", "offline recovery"); err != nil {
			t.Fatal(err)
		}
		if _, err := s.VerifyAudit(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSecurityAuditTrustedBaseline(t *testing.T) {
	s, p := openStore(t, filepath.Join(t.TempDir(), "server.db"))
	credential := fixtureCredential(t)
	if err := s.Insert(t.Context(), credential, true); err != nil {
		t.Fatal(err)
	}
	if err := s.ResetAccount(t.Context(), credential.Username, credential.PasswordHash, "", "old recovery"); err != nil {
		t.Fatal(err)
	}
	var original string
	if err := p.Read().QueryRowContext(t.Context(), securityAuditColumns).Scan(new(string), &original); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Write().ExecContext(t.Context(), `DROP TABLE audit_chain; DROP TABLE audit_chain_head`); err != nil {
		t.Fatal(err)
	}
	if err := s.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	head, err := s.VerifyAudit(t.Context())
	if err != nil || head.Baseline != 1 || head.Size != 1 {
		t.Fatalf("baseline %+v %v", head, err)
	}
	var adopted string
	if err := p.Read().QueryRowContext(t.Context(), securityAuditColumns).Scan(new(string), &adopted); err != nil || adopted != original {
		t.Fatalf("baseline rewrote audit: %v", err)
	}
	for _, query := range []string{`UPDATE account_security_audit SET reason='changed'`, `DELETE FROM account_security_audit`, `UPDATE audit_chain SET hash='changed'`, `DELETE FROM audit_chain`} {
		if _, err := p.Write().ExecContext(t.Context(), query); err == nil {
			t.Fatalf("mutable audit: %s", query)
		}
	}
	if err := s.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	again, err := s.VerifyAudit(t.Context())
	if err != nil || head != again {
		t.Fatalf("baseline changed on restart: %+v %v", again, err)
	}
}
