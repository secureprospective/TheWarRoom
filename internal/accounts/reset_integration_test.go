package accounts_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
)

// An incident reset ends sign-ins but keeps printed codes; a restore reset also drops every
// code and unredeemed invitation, since one used after the snapshot reads as unused.
func TestSignInResetAndRestoreReset(t *testing.T) {
	f, recovery, codes, proof := recoveryFixture(t)
	service := accounts.NewSessions(accounts.New(f.store), f.store)
	principal, err := f.account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.IssueEnrollment(t.Context(), digestToken("invitation"), "new.owner", "native.test", principal); err != nil {
		t.Fatal(err)
	}
	before, err := f.store.VerifyAudit(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if restore, incident, err := f.store.LastResets(t.Context()); err != nil || !restore.At.IsZero() || !incident.At.IsZero() {
		t.Fatalf("resets reported before any: %+v %+v %v", restore, incident, err)
	}
	session := loginDevice(t, f, service, "before incident")
	if _, err := f.store.ResetSignIns(t.Context(), "\x00"); err == nil {
		t.Fatal("unprintable reason accepted")
	}
	if n, err := f.store.ResetSignIns(t.Context(), "suspected stolen laptop"); err != nil || n != 1 {
		t.Fatalf("incident reset: %d %v", n, err)
	}
	if _, err := service.Authenticate(t.Context(), session.RawSession); !errors.Is(err, accounts.ErrSessionRefused) {
		t.Fatalf("session survived reset: %v", err)
	}
	if err := f.service.CheckPrivileged(t.Context(), f.policy, principal, proof); err == nil {
		t.Fatal("recent second-factor proof survived reset")
	}
	var codesLeft, invitations int
	count := func() {
		t.Helper()
		if err := f.pools.Read().QueryRowContext(t.Context(), `SELECT (SELECT count(*) FROM account_recovery_codes),
 (SELECT count(*) FROM account_enrollments WHERE redeemed_account IS NULL)`).Scan(&codesLeft, &invitations); err != nil {
			t.Fatal(err)
		}
	}
	if count(); codesLeft != len(codes.Codes) || invitations != 1 {
		t.Fatalf("incident reset removed codes or invitations: %d %d", codesLeft, invitations)
	}
	if n, err := f.store.ResetAfterRestore(t.Context(), "restored from backup"); err != nil || n != 1 {
		t.Fatalf("restore reset: %d %v", n, err)
	}
	if count(); codesLeft != 0 || invitations != 0 {
		t.Fatalf("restore kept revivable secrets: %d %d", codesLeft, invitations)
	}
	err = recovery.Recover(t.Context(), accounts.RawRecovery{Username: f.account.Username, Code: codes.Codes[1], Password: "replacement-password"})
	if !errors.Is(err, accounts.ErrRecoveryRefused) {
		t.Fatalf("printed code survived restore: %v", err)
	}
	after, err := f.store.VerifyAudit(t.Context())
	if err != nil || after.Size != before.Size+2 {
		t.Fatalf("resets not in sign-in history: %+v %v", after, err)
	}
	var reasons string
	if err := f.pools.Read().QueryRowContext(t.Context(), `SELECT group_concat(operation || ':' || reason) FROM account_security_audit WHERE sequence > ?`, before.Size).Scan(&reasons); err != nil ||
		!strings.Contains(reasons, "sign_ins_reset:suspected stolen laptop") || !strings.Contains(reasons, "restored_sign_ins_reset:restored from backup") {
		t.Fatalf("audit reasons %q: %v", reasons, err)
	}
	// The dashboard reads both back from the same history.
	restore, incident, err := f.store.LastResets(t.Context())
	if err != nil || restore.Reason != "restored from backup" || incident.Reason != "suspected stolen laptop" ||
		restore.Accounts != 1 || incident.Accounts != 1 || restore.At.Before(incident.At) {
		t.Fatalf("last resets %+v %+v: %v", restore, incident, err)
	}
	if _, err := service.Authenticate(t.Context(), loginDevice(t, f, service, "after restore").RawSession); err != nil {
		t.Fatal("password sign-in refused after reset:", err)
	}
}
