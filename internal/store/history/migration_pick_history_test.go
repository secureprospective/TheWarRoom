package history

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
)

func TestMigrationHistoricalPickImport(t *testing.T) {
	export := migrationtest.Synthetic("Pick history", migrationtest.Specs()["8 teams"])
	raw := transactions.RawTransactions{Transactions: []transactions.RawTransaction{{Kind: "TRADE", Timestamp: fmt.Sprint(export.ReadAt.Add(-time.Hour).Unix()), Franchise: "0001", Counterparty: "0002", Gave: new("FP_0001_2026_3,"), CounterpartyGave: new(""), Expires: fmt.Sprint(export.ReadAt.Unix())}}}
	export.Transactions = &raw
	svc, s, _ := migrationService(t, export)
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	for pass := range 2 {
		receipts, err := svc.Import(t.Context(), export.LeagueID, "Import transaction evidence", "creator")
		if err != nil || (pass == 0 && len(receipts) != 4) || (pass == 1 && len(receipts) != 0) {
			t.Fatalf("pass %d: receipts=%d err=%v", pass, len(receipts), err)
		}
	}
	records, err := s.nativeRecords(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	assets, err := commissioner.NativeAssetsReading(league, records)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets.Snapshot.PickHistory) != 1 || assets.Snapshot.PickHistory[0].Owner != "franchise.02" || !assets.Snapshot.PickHistory[0].Evidence.Unverified {
		t.Fatalf("history=%+v", assets.Snapshot.PickHistory)
	}
	if !reflect.DeepEqual(assets.Snapshot.Picks, exportSeed(t, export).Picks) {
		t.Fatal("historical import changed selectable picks")
	}
	jobs, err := commissioner.JobsReading(league, records, export.ReadAt.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, j := range jobs.Jobs {
		if strings.HasPrefix(j.ID, "pick.") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("pick jobs=%d", count)
	}
	before, err := commissioner.NativeAssetsReading(league, records, export.ReadAt.Add(-time.Hour-time.Nanosecond))
	if err != nil {
		t.Fatal(err)
	}
	if before.Snapshot.PickHistory[0].Owner != "franchise.01" {
		t.Fatal("baseline lost original holder")
	}
}

func exportSeed(t *testing.T, export mflsource.Export) commissioner.RawNativeSnapshot {
	t.Helper()
	plan, err := mflsource.Prepare(export)
	if err != nil {
		t.Fatal(err)
	}
	return plan.Seed
}

func TestMigrationBatchRecoveryIsAtomic(t *testing.T) {
	export := migrationtest.Synthetic("Recovery", migrationtest.Specs()["8 teams"])
	svc, s, _ := migrationService(t, export)
	if _, err := svc.Import(t.Context(), export.LeagueID, "Import", "creator"); err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	initial, err := s.RosterRestorePoint(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	player := initial.Snapshot.Players[0]
	other := "franchise.01"
	if player.Franchise == other {
		other = "franchise.02"
	}
	records, err := s.nativeRecords(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	raws := []commissioner.RawCommand{
		{League: league, Command: commissioner.CorrectRoster, Args: commissioner.RawArgs{Correction: &commissioner.RawCorrection{Player: player.ID, Franchise: player.Franchise, Action: "remove"}}, Expected: int64(len(records)), Request: "recovery.remove", Reason: "Repair membership", Effective: "now", Visibility: "league"},
		{League: league, Command: commissioner.CorrectRoster, Args: commissioner.RawArgs{Correction: &commissioner.RawCorrection{Player: player.ID, Franchise: other, Action: "add"}}, Request: "recovery.add", Reason: "Repair membership", Effective: "now", Visibility: "league"},
	}
	s.batchFault = func(step int) error {
		if step == 1 {
			return fmt.Errorf("stop before commit")
		}
		return nil
	}
	if _, err := s.ExecuteCommissionerBatch(t.Context(), raws, "creator", nil); err == nil {
		t.Fatal("fault accepted")
	}
	after, err := s.RosterRestorePoint(t.Context(), league)
	if err != nil || !reflect.DeepEqual(initial, after) {
		t.Fatal("failed batch changed recovery", err)
	}
	unchanged, err := s.nativeRecords(t.Context(), league)
	if err != nil || !reflect.DeepEqual(records, unchanged) {
		t.Fatal("failed batch changed ledger", err)
	}
	s.batchFault = nil
	if _, err := s.ExecuteCommissionerBatch(t.Context(), raws, "creator", nil); err != nil {
		t.Fatal(err)
	}
	after, err = s.RosterRestorePoint(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	if after.Revision != int64(len(records)+2) || after.Snapshot.PlayerOf(other, player.ID) < 0 || after.Snapshot.PlayerOf(player.Franchise, player.ID) >= 0 {
		t.Fatal("recovery did not capture final batch state")
	}
}
