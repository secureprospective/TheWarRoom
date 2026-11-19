package accounts

import (
	"path/filepath"
	"strconv"
	"testing"
	"time"

	identity "github.com/secureprospective/TheWarRoom/internal/accounts"
)

func TestJobSnoozesArePrivateByAccountAndWakeOnlyOneJob(t *testing.T) {
	store, _ := openServerStore(t, filepath.Join(t.TempDir(), "accounts.db"))
	addHomeLayoutTestAccounts(t, store)
	const owner = "12345678-1234-4234-8234-123456789abc"
	until := time.Now().UTC().Add(3 * time.Hour).Format(time.RFC3339Nano)
	first := identity.RawJobSnoozeSet{League: "league.a", JobID: "trade.review", Until: until}
	if _, err := store.SetJobSnooze(t.Context(), owner, first); err != nil {
		t.Fatal(err)
	}
	second := identity.RawJobSnoozeSet{League: "league.b", JobID: "trade.review", Until: until}
	if _, err := store.SetJobSnooze(t.Context(), owner, second); err != nil {
		t.Fatal(err)
	}

	mine, err := store.ReadJobSnoozes(t.Context(), owner)
	if err != nil || len(mine.Snoozes) != 2 {
		t.Fatalf("owner snoozes = %+v err=%v", mine, err)
	}
	other, err := store.ReadJobSnoozes(t.Context(), secondAccountID)
	if err != nil || len(other.Snoozes) != 0 {
		t.Fatalf("another account saw snoozes: %+v err=%v", other, err)
	}

	woken, err := store.WakeJobSnooze(t.Context(), owner,
		identity.RawJobSnoozeWake{League: first.League, JobID: first.JobID})
	if err != nil || len(woken.Snoozes) != 1 || woken.Snoozes[0].League != second.League {
		t.Fatalf("wake removed the wrong mark: %+v err=%v", woken, err)
	}
	other, err = store.ReadJobSnoozes(t.Context(), secondAccountID)
	if err != nil || len(other.Snoozes) != 0 {
		t.Fatalf("wake changed another account: %+v err=%v", other, err)
	}
}

func TestInvalidJobSnoozeDoesNotChangeStoredMark(t *testing.T) {
	store, _ := openServerStore(t, filepath.Join(t.TempDir(), "accounts.db"))
	addHomeLayoutTestAccounts(t, store)
	const owner = "12345678-1234-4234-8234-123456789abc"
	now := time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC)
	store.SetClock(func() time.Time { return now })
	until := now.Add(3 * time.Hour).Format(time.RFC3339Nano)
	valid := identity.RawJobSnoozeSet{League: "league.a", JobID: "trade.review", Until: until}
	if _, err := store.SetJobSnooze(t.Context(), owner, valid); err != nil {
		t.Fatal(err)
	}
	invalid := identity.RawJobSnoozeSet{League: valid.League, JobID: valid.JobID, Until: "tomorrow"}
	if _, err := store.SetJobSnooze(t.Context(), owner, invalid); err == nil {
		t.Fatal("invalid snooze time accepted")
	}
	for _, expired := range []string{now.Format(time.RFC3339Nano), now.Add(-time.Second).Format(time.RFC3339Nano)} {
		past := identity.RawJobSnoozeSet{League: valid.League, JobID: valid.JobID, Until: expired}
		if _, err := store.SetJobSnooze(t.Context(), owner, past); err == nil {
			t.Fatalf("snooze at or before now accepted: %s", expired)
		}
	}
	stored, err := store.ReadJobSnoozes(t.Context(), owner)
	if err != nil || len(stored.Snoozes) != 1 || stored.Snoozes[0].Until != until {
		t.Fatalf("invalid snooze changed stored mark: %+v err=%v", stored, err)
	}
}

func TestJobSnoozeStoreBoundsActiveMarksPerAccount(t *testing.T) {
	store, _ := openServerStore(t, filepath.Join(t.TempDir(), "accounts.db"))
	addHomeLayoutTestAccounts(t, store)
	const owner = "12345678-1234-4234-8234-123456789abc"
	until := time.Now().UTC().Add(3 * time.Hour).Format(time.RFC3339Nano)
	tx, err := store.pools.BeginWrite(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	for index := 0; index < identity.MaxAccountJobSnoozes; index++ {
		if _, err := tx.ExecContext(t.Context(), `INSERT INTO account_job_snoozes(account_id, league, job_id, until)
 VALUES (?, 'league.a', ?, ?)`, owner, "job."+strconv.Itoa(index), until); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	_, err = store.SetJobSnooze(t.Context(), owner, identity.RawJobSnoozeSet{
		League: "league.a", JobID: "job.too-many", Until: until,
	})
	if err == nil {
		t.Fatal("accepted more than the per-account snooze limit")
	}
	state, err := store.ReadJobSnoozes(t.Context(), owner)
	if err != nil || len(state.Snoozes) != identity.MaxAccountJobSnoozes {
		t.Fatalf("limit refusal changed stored marks: count=%d err=%v", len(state.Snoozes), err)
	}
}
