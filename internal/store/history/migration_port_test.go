package history

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"

	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
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

func TestMigrationAnswersRollbackWithCreation(t *testing.T) {
	t.Parallel()
	export := migrationtest.Synthetic("Answered league", migrationtest.Specs()["8 teams"])
	svc, store, _ := migrationService(t, export)
	// Read without a bundle so contract years is one of the catalogue's waiting settings.
	options := leaguemigration.ImportOptions{Answers: []commissioner.SeedValue{
		{Field: "contracts.maximumYears", Value: json.RawMessage(`7`)},
	}}
	store.batchFault = func(i int) error {
		if i == 2 {
			return errors.New("answer write failed")
		}
		return nil
	}
	if _, err := svc.ImportWithOptions(t.Context(), export.LeagueID, "Move with answers", "creator", options); err == nil {
		t.Fatal("failed answer was reported successful")
	}
	if n := count(t, store, "league_event"); n != 0 {
		t.Fatalf("partial league after answer failure: %d", n)
	}
	store.batchFault = nil
	if _, err := svc.ImportWithOptions(t.Context(), export.LeagueID, "Move with answers", "creator", options); err != nil {
		t.Fatal(err)
	}
	if n := count(t, store, "league_event"); n != 3 {
		t.Fatalf("want creation, snapshot and answer; got %d", n)
	}
}

func TestPrepareAndJobsRulingDifference(t *testing.T) {
	fixture, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	fixture.Base = leaguerules.LegacyNFL()
	fixture.DraftType = "SAME"
	svc, store, _ := migrationService(t, fixture)
	prepared, err := mflsource.Prepare(fixture)
	if err != nil {
		t.Fatal(err)
	}
	// An existing ledger can lack the draft-kind evidence a fresh read now holds.
	fixture.DraftType = ""
	plan, err := mflsource.Prepare(fixture)
	if err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID(fixture.LeagueID, fixture.Season)
	_, err = svc.Ledger.Batch(t.Context(), []commissioner.RawCommand{
		{League: league, Command: commissioner.Create, Args: plan.Create, Request: "create", Reason: "Check rulings", Effective: "now", Visibility: "league"},
		{League: league, Command: commissioner.NativeSeed, Args: commissioner.RawArgs{Native: &plan.Seed}, Request: "seed", Reason: "Check rulings", Effective: "now", Visibility: "league"},
	}, "creator")
	if err != nil {
		t.Fatal(err)
	}
	records, err := store.nativeRecords(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	rules, err := commissioner.RulesNow(records, svc.Now())
	if err != nil {
		t.Fatal(err)
	}
	waiting := map[string]bool{}
	for _, setting := range prepared.Waiting {
		waiting[setting.Field] = true
	}
	var extra []string
	fields, values := leaguerules.Catalogue(), leaguerules.ReadSettings(rules)
	for i, field := range fields {
		if values[i].State == "Needs a ruling" && !leaguerules.Unused(rules, field.Path) && !waiting[string(field.Path)] {
			extra = append(extra, string(field.Path))
		}
	}
	slices.Sort(extra)
	jobs, err := store.CommissionerJobs(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	index := slices.IndexFunc(jobs.Jobs, func(job commissioner.Job) bool { return job.ID == "setup" })
	if index < 0 || jobs.Jobs[index].Words != "6 league rules still need a ruling." {
		t.Fatalf("setup job: %+v", jobs.Jobs)
	}
	t.Logf("Prepare waits for %d; extra setup rulings: %v", len(waiting), extra)
	want := []string{"draft.laterPattern"}
	if len(waiting) != 5 || !slices.Equal(extra, want) {
		t.Fatalf("unexpected count/difference: %d / %v", len(waiting), extra)
	}
	// Once the old ledger has the same draft evidence, its count must match Prepare.
	_, err = svc.Ledger.Execute(t.Context(), commissioner.RawCommand{League: league,
		Command: commissioner.RulesSet, Args: commissioner.RawArgs{Field: "draft.laterPattern",
			Value: json.RawMessage(`"linear"`)}, Expected: int64(len(records)), Request: "draft-evidence",
		Reason: "Record the held draft pattern", Effective: "now", Visibility: "league"}, "creator")
	if err != nil {
		t.Fatal(err)
	}
	jobs, err = store.CommissionerJobs(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	index = slices.IndexFunc(jobs.Jobs, func(job commissioner.Job) bool { return job.ID == "setup" })
	if index < 0 || jobs.Jobs[index].Words != "5 league rules still need a ruling." {
		t.Fatalf("Jobs must match Prepare with the same settings: %+v", jobs.Jobs)
	}
}
