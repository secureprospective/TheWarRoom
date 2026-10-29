package history

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
)

func TestMigrationSourcePortPrepareImportReview(t *testing.T) {
	t.Parallel()
	export := migrationtest.Synthetic("Port league", migrationtest.Specs()["8 teams"])
	svc, store, src := migrationService(t, export)
	ctx := t.Context()
	ready, err := svc.Prepare(ctx, export.LeagueID)
	if err != nil || !ready.Ready || ready.Players != len(export.Rosters) || src.Reads != 5 {
		t.Fatalf("prepare: %+v, reads=%d, err=%v", ready, src.Reads, err)
	}
	if n := count(t, store, "league_event"); n != 0 {
		t.Fatalf("prepare wrote %d ledger rows", n)
	}
	// Import must use a fresh read, not the candidate shown by Prepare.
	src.Export.Rosters[0].Salary = "9.99"
	receipts, err := svc.Import(ctx, export.LeagueID, "Import from the fixture", "creator")
	if err != nil || len(receipts) != 2 || src.Reads != 10 {
		t.Fatalf("import: receipts=%d, reads=%d, err=%v", len(receipts), src.Reads, err)
	}
	req := leaguemigration.Request{MFLLeague: export.LeagueID, League: ready.League}
	review, err := svc.Review(ctx, req)
	if err != nil || review.Differences != 0 || src.Reads != 15 {
		t.Fatalf("review: %+v, reads=%d, err=%v", review, src.Reads, err)
	}
	records, err := store.nativeRecords(ctx, req.League)
	if err != nil {
		t.Fatal(err)
	}
	app, err := leaguemigration.ReadApp(req.League, records, svc.Now())
	if err != nil {
		t.Fatal(err)
	}
	// Independent raw-source comparisons cover rosters, contracts, picks and rules.
	assertAppHoldsSource(t, src.Export, app)
	// The only source is an in-memory fixture reader: it owns no HTTP client or
	// write capability. Every source operation above is one of its five read methods.
}

func TestMalformedSourcePortLeavesNoDestination(t *testing.T) {
	t.Parallel()
	export := migrationtest.Synthetic("Malformed source", migrationtest.Specs()["8 teams"])
	export.Rosters[0].Salary = "lots"
	svc, store, src := migrationService(t, export)
	ctx := t.Context()
	ready, err := svc.Prepare(ctx, export.LeagueID)
	if err != nil || ready.Ready || !strings.Contains(strings.Join(ready.Problems, " "), "not an amount") {
		t.Fatalf("prepare must explain how to fix the source: %+v %v", ready, err)
	}
	if _, err = svc.Import(ctx, export.LeagueID, "Import malformed source", "creator"); err == nil ||
		!strings.Contains(err.Error(), "not an amount") {
		t.Fatalf("import must refuse with an actionable error: %v", err)
	}
	records, err := store.nativeRecords(ctx, ready.League)
	if err != nil || len(records) != 0 || count(t, store, "league_event") != 0 {
		t.Fatalf("malformed source left a partial destination: %d records, %v", len(records), err)
	}
	src.Export.Rosters[0].Salary = "1.30"
	if _, err := svc.Import(ctx, export.LeagueID, "Retry corrected source", "creator"); err != nil {
		t.Fatalf("corrected source could not be imported: %v", err)
	}
}
