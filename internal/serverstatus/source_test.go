package serverstatus

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
)

type fakeHistory struct {
	size              int64
	restore, incident accountstore.Reset
}

func (f fakeHistory) VerifyAudit(context.Context) (db.AuditHead, error) {
	return db.AuditHead{Size: f.size}, nil
}

func (f fakeHistory) LastResets(context.Context) (accountstore.Reset, accountstore.Reset, error) {
	return f.restore, f.incident, nil
}

func TestBackupStatesAndResetNotes(t *testing.T) {
	now := time.Date(2026, 10, 25, 15, 0, 0, 0, time.UTC)
	signIns := fakeHistory{size: 56,
		restore:  accountstore.Reset{At: now.Add(-3 * time.Hour), Reason: "restored onto a spare server from the backup saved 2026-10-25T11:55:00Z", Accounts: 32},
		incident: accountstore.Reset{At: now.Add(-50 * time.Hour), Reason: "lost laptop", Accounts: 32}}
	source := Source{League: fakeHistory{size: 1234}, SignIns: signIns, Now: func() time.Time { return now }}
	for _, tc := range []struct {
		status     offsite.Status
		configured bool
		state      State
		want       string
	}{
		{offsite.Status{Captured: now.Add(-4 * time.Minute), Writable: true}, true, OK, "Last backup saved 2026-10-25 14:56 UTC (4 minutes ago)"},
		{offsite.Status{Captured: now.Add(-4 * time.Minute), Writable: true, Failure: "prune refused"}, true, Problem, "League changes still work, but the newest backup attempt failed: prune refused"},
		{offsite.Status{Captured: now.Add(-20 * time.Minute), Failure: "upload timed out"}, true, Problem, "Last good backup: 2026-10-25 14:40 UTC (20 minutes ago). Newest error: upload timed out."},
		{offsite.Status{}, false, Problem, "Off-site backups are not set up"},
	} {
		source.Backup = func() (offsite.Status, bool) { return tc.status, tc.configured }
		checks := source.Checks(t.Context())
		if checks[0].State != tc.state || !strings.Contains(checks[0].Detail, tc.want) {
			t.Errorf("backup line %+v, want %s %q", checks[0], tc.state, tc.want)
		}
		if tc.state == Problem && checks[0].Action == "" {
			t.Errorf("problem without an action: %+v", checks[0])
		}
	}
	source.Backup = func() (offsite.Status, bool) {
		return offsite.Status{Captured: now.Add(-time.Minute), Writable: true}, true
	}
	var out bytes.Buffer
	problems, err := Render(&out, now, source.Checks(t.Context()))
	if err != nil || problems != 0 {
		t.Fatalf("healthy server shows %d problems: %v\n%s", problems, err, out.String())
	}
	for _, want := range []string{
		"OK       History: League history (1234 entries) and sign-in history (56 entries) check out.",
		"NOTE     Last restore: 2026-10-25 12:00 UTC (3 hours ago): restored onto a spare server from the backup saved 2026-10-25T11:55:00Z. 32 accounts were signed out.",
		`NOTE     Last sign-everyone-out: 2026-10-23 13:00 UTC (2 days ago), reason "lost laptop", 32 accounts.`,
		"Everything is working. Nothing to do.",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("dashboard lacks %q:\n%s", want, out.String())
		}
	}
}
