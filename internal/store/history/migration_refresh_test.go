package history

import (
	"errors"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
)

func TestFranchiseEditIsBatchable(t *testing.T) {
	t.Parallel()
	if !batchable("commish.franchise.edit", false) {
		t.Fatal("commish.franchise.edit must be batchable for league refresh")
	}
}

func TestRefreshBatchFaultRollsBackAtomically(t *testing.T) {
	t.Parallel()
	export := migrationtest.Synthetic("Rollback league", migrationtest.Specs()["8 teams"])
	svc, store, src := migrationService(t, export)
	ctx := t.Context()
	if _, err := svc.Import(ctx, export.LeagueID, "Initial load", "creator"); err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	before, err := store.nativeRecords(ctx, league)
	if err != nil {
		t.Fatal(err)
	}

	// Mutate multiple asset types (roster, contract, franchise name) so the batch has multiple steps.
	mutated := src.Export
	mutated.Config.Franchises = append(mutated.Config.Franchises[:0:0], mutated.Config.Franchises...)
	mutated.Config.Franchises[0].Name = "Renamed On MFL"
	mutated.Rosters = append(mutated.Rosters[:0:0], mutated.Rosters...)
	mutated.Rosters[0].Salary = "9.99"
	mutated.Rosters[1].Franchise = "0002"
	src.Export = mutated

	// Inject a fault mid-batch inside the real SQLite write transaction.
	store.batchFault = func(step int) error {
		if step == 1 {
			return errors.New("injected disk failure mid-batch")
		}
		return nil
	}
	_, err = svc.Refresh(ctx, leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID,
		League:    league,
		Reason:    "Refresh from MFL",
		Principal: "creator",
	})
	if err == nil {
		t.Fatal("expected batch fault error")
	}
	store.batchFault = nil

	after, err := store.nativeRecords(ctx, league)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("failed refresh batch left partial records: before=%d after=%d", len(before), len(after))
	}

	// After clearing the fault, the same refresh succeeds cleanly and is idempotent on retry.
	report, err := svc.Refresh(ctx, leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID,
		League:    league,
		Reason:    "Refresh from MFL",
		Principal: "creator",
	})
	if err != nil {
		t.Fatalf("retry after rollback failed: %v", err)
	}
	if len(report.Receipts) == 0 || report.Summary.Franchises != 1 {
		t.Fatalf("unexpected retry report: %+v", report)
	}
	again, err := svc.Refresh(ctx, leaguemigration.RefreshRequest{
		MFLLeague: export.LeagueID,
		League:    league,
		Reason:    "Refresh from MFL",
		Principal: "creator",
	})
	if err != nil || len(again.Commands) != 0 || len(again.Receipts) != 0 {
		t.Fatalf("second refresh should write nothing: report=%+v err=%v", again, err)
	}
	// Verify the franchise rename actually landed in the directory projection.
	records, err := store.nativeRecords(ctx, league)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := commissioner.NativeDirectoryReading(league, records, svc.Now())
	if err != nil || dir.Seats().Of("franchise.01").Name != "Renamed On MFL" {
		t.Fatalf("franchise rename missing: %+v %v", dir.Seats().Of("franchise.01"), err)
	}
}
