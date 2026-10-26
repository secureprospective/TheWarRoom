package history

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func seededRecoveryStore(t *testing.T) (*Store, commissioner.RawCommand) {
	t.Helper()
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetLegacy, 0), "creator", nil); err != nil {
		t.Fatal(err)
	}
	seed := heldSeedCommand(t)
	if _, err := s.ExecuteCommissioner(t.Context(), seed, "creator", nil); err != nil {
		t.Fatal(err)
	}
	return s, seed
}

func TestRosterRestorePointCapturesExactContractsWithoutHistory(t *testing.T) {
	t.Parallel()
	s, seed := seededRecoveryStore(t)
	point, err := s.RosterRestorePoint(t.Context(), seed.League)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(point.Snapshot.Players, seed.Args.Native.Players) ||
		!reflect.DeepEqual(point.Snapshot.Franchises, seed.Args.Native.Franchises) ||
		!reflect.DeepEqual(point.Snapshot.Adjustments, seed.Args.Native.Adjustments) || point.Revision != 2 {
		t.Fatalf("rosters or exact-cent contracts changed: %+v", point)
	}
	if len(point.Snapshot.Picks) != 0 || point.Validate() != nil {
		t.Fatal("restore point requires draft history")
	}
	if _, err := s.ExecuteCommissioner(t.Context(), seed, "creator", nil); err != nil {
		t.Fatal(err)
	}
	var logs int
	if err := s.pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM league_recovery_log WHERE league=?`, seed.League).Scan(&logs); err != nil || logs != 1 {
		t.Fatalf("retry wrote duplicate recovery log: count=%d err=%v", logs, err)
	}
	for _, query := range []string{`UPDATE league_recovery_log SET message='changed'`, `DELETE FROM league_recovery_log`} {
		if _, err := s.pools.Write().ExecContext(t.Context(), query); err == nil {
			t.Fatal("recovery log was rewritten")
		}
	}
}

func TestRosterRestorePointFailureRollsBackLedgerAndAudit(t *testing.T) {
	t.Parallel()
	for _, table := range []string{"league_roster_restore", "league_recovery_log"} {
		t.Run(table, func(t *testing.T) {
			s, _, _ := newStore(t)
			if _, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetLegacy, 0), "creator", nil); err != nil {
				t.Fatal(err)
			}
			before, err := s.VerifyAudit(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.pools.Write().ExecContext(t.Context(), `CREATE TRIGGER reject_recovery BEFORE INSERT ON `+table+` BEGIN SELECT RAISE(ABORT,'restore storage unavailable'); END`); err != nil {
				t.Fatal(err)
			}
			seed := heldSeedCommand(t)
			if _, err := s.ExecuteCommissioner(t.Context(), seed, "creator", nil); err == nil {
				t.Fatal("league change acknowledged without its restore point")
			}
			after, err := s.VerifyAudit(t.Context())
			if err != nil || before != after {
				t.Fatalf("failed restore capture changed ledger audit: %+v %+v %v", before, after, err)
			}
			if _, err := s.RosterRestorePoint(t.Context(), seed.League); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("partial restore point survived rollback: %v", err)
			}
			assets, err := s.NativeAssets(t.Context(), seed.League)
			if err != nil || assets.Seeded {
				t.Fatalf("failed capture left seeded league: %+v %v", assets, err)
			}
		})
	}
}

func TestRosterRestoreMigrationBackfillsWithoutChangingEventBytes(t *testing.T) {
	t.Parallel()
	s, seed := seededRecoveryStore(t)
	before := historyJSON(t, s.pools.Read(), `SELECT json_group_array(payload) FROM (SELECT payload FROM league_event ORDER BY revision)`)
	if _, err := s.pools.Write().ExecContext(t.Context(), `DROP TABLE league_roster_restore; DROP TABLE league_recovery_log; PRAGMA user_version=8`); err != nil {
		t.Fatal(err)
	}
	if err := s.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	point, err := s.RosterRestorePoint(t.Context(), seed.League)
	if err != nil || !reflect.DeepEqual(point.Snapshot.Players, seed.Args.Native.Players) {
		t.Fatalf("legacy contracts not backfilled: %+v %v", point, err)
	}
	if after := historyJSON(t, s.pools.Read(), `SELECT json_group_array(payload) FROM (SELECT payload FROM league_event ORDER BY revision)`); before != after {
		t.Fatal("migration changed original ledger bytes")
	}
}

func TestRosterRestorePointRejectsStoredIdentityMismatch(t *testing.T) {
	t.Parallel()
	for _, mutation := range []string{
		`json_set(payload,'$.revision',999)`,
		`json_set(payload,'$.league','native.other')`,
		`json_set(payload,'$.snapshot.players[0].salaryCents',-1)`,
	} {
		t.Run(mutation, func(t *testing.T) {
			t.Parallel()
			s, seed := seededRecoveryStore(t)
			if _, err := s.pools.Write().ExecContext(t.Context(), `UPDATE league_roster_restore SET payload=`+mutation); err != nil {
				t.Fatal(err)
			}
			if _, err := s.RosterRestorePoint(t.Context(), seed.League); err == nil {
				t.Fatal("corrupt restore point accepted")
			}
		})
	}
}

func TestRosterRestoreTracksContractChangeAndReversal(t *testing.T) {
	t.Parallel()
	s, seed := seededRecoveryStore(t)
	years, salary := 4, domain.Money(123456789)
	edit := commissioner.RawCommand{League: seed.League, Command: commissioner.ContractEdit,
		Args:     commissioner.RawArgs{Override: &commissioner.RawNativeOverride{Player: "0099", Years: &years, Salary: &salary}},
		Expected: 2, Request: "edit-contract", Reason: "Correct the recorded contract", Effective: "now", Visibility: "league"}
	receipt, err := s.ExecuteCommissioner(t.Context(), edit, "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	point, err := s.RosterRestorePoint(t.Context(), seed.League)
	if err != nil || point.Revision != receipt.Revision || point.Snapshot.Players[0].Salary != salary {
		t.Fatalf("changed contract not captured: %+v %v", point, err)
	}
	assets, err := s.NativeAssets(t.Context(), seed.League)
	if err != nil || !reflect.DeepEqual(point.Snapshot.Players, assets.Snapshot.Players) {
		t.Fatalf("point differs from league: %v", err)
	}
	if _, err := s.ExecuteCommissioner(t.Context(), reverseCommand(receipt.Entry, 3, "undo-contract"), "creator", nil); err != nil {
		t.Fatal(err)
	}
	point, err = s.RosterRestorePoint(t.Context(), seed.League)
	if err != nil || point.Revision != 4 || !reflect.DeepEqual(point.Snapshot.Players, seed.Args.Native.Players) {
		t.Fatalf("reversal not captured: %+v %v", point, err)
	}
}
