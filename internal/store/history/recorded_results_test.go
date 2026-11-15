package history

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/weeklyresults"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func recordedImportFixture(t *testing.T) mflsource.Export {
	t.Helper()
	export := migrationtest.Synthetic("Recorded", migrationtest.Specs()["8 teams"])
	var err error
	export.Schedule, err = leagueschedule.Parse([]byte(`{"schedule":{"weeklySchedule":[{"week":"1","matchup":{"franchise":[{"id":"0001","isHome":"1","score":"105.25"},{"id":"0002","isHome":"0","score":"99.00"}]}},{"week":"2","matchup":{"franchise":[{"id":"0001","isHome":"1","score":"999"},{"id":"0002","isHome":"0","score":"0"}]}}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	export.Results, err = weeklyresults.Parse([]byte(`{"allWeeklyResults":{"weeklyResults":[{"week":"1","matchup":{"regularSeason":"1","franchise":[{"id":"0001","isHome":"1","score":"105.25","result":"W"},{"id":"0002","isHome":"0","score":"99.00","result":"L"}]}},{"week":"2","matchup":{"regularSeason":"1","franchise":[{"id":"0001","isHome":"1","score":""},{"id":"0002","isHome":"0","score":""}]}}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	return export
}

func TestPreImportRecordedScoresRemainUnchanged(t *testing.T) {
	t.Parallel()
	export := recordedImportFixture(t)
	svc, store, _ := migrationService(t, export)
	if _, err := svc.Import(t.Context(), export.LeagueID, "Import recorded results", "creator"); err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	records, err := store.nativeRecords(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	held := commissioner.RecordedResults(records, export.Season)
	if held == nil || held.CutoverWeek != 2 || len(held.Teams) != 2 || held.Teams[0].Result != "W" || held.Teams[1].Result != "L" {
		t.Fatalf("evidence: %+v", held)
	}
	for pass := range 2 {
		store.SetSeasonScoreCache(pass == 0)
		got, err := store.NativeResults(t.Context(), league, export.Season, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Games) != 1 || got.Games[0].HomePoints == nil || *got.Games[0].HomePoints != 1052500 || *got.Games[0].AwayPoints != 990000 || got.Games[0].Winner != "franchise.01" {
			t.Fatalf("recorded scores: %+v", got)
		}
		current, err := store.NativeResults(t.Context(), league, export.Season, 2)
		if err != nil {
			t.Fatal(err)
		}
		if current.Games[0].HomePoints != nil {
			t.Fatal("in-play schedule score used at cutover", current)
		}
		standings, err := store.NativeStandings(t.Context(), league, export.Season)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range standings.Computed {
			switch row.Franchise {
			case "franchise.01":
				if row.Wins != 1 || row.Losses != 0 || row.PointsFor != 1052500 {
					t.Fatal(row)
				}
			case "franchise.02":
				if row.Wins != 0 || row.Losses != 1 || row.PointsFor != 990000 {
					t.Fatal(row)
				}
			default:
				if row.Wins+row.Losses+row.Ties != 0 || row.PointsFor != 0 {
					t.Fatal(row)
				}
			}
		}
		if standings.Decided != 1 || standings.Pending != 1 {
			t.Fatal(standings)
		}
		odds, err := store.NativeOdds(t.Context(), league, export.Season, commissioner.OddsValues{})
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range odds.Rows {
			if row.Franchise == "franchise.01" && row.Record != "1-0" {
				t.Fatal(row)
			}
		}
	}
	// Fresh readers re-decode the ordinary durable ledger, not an in-memory side store.
	again, err := store.nativeRecords(t.Context(), league)
	if err != nil || !reflect.DeepEqual(held, commissioner.RecordedResults(again, export.Season)) {
		t.Fatal("evidence did not round-trip", err)
	}
}

func TestRecordedBoundaryStraddlesAppScoringAndNativeReadings(t *testing.T) {
	t.Parallel()
	f := resultsTest(t)
	baseline, err := f.s.NativeResults(t.Context(), "native.test", 2026, 2)
	if err != nil {
		t.Fatal(err)
	}
	f.s.SetSeasonScoreCache(false)
	uncached, err := f.s.NativeResults(t.Context(), "native.test", 2026, 2)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(baseline)
	after, _ := json.Marshal(uncached)
	if string(before) != string(after) {
		t.Fatal("native readings differ without cache")
	}
	schedule, err := commissioner.ScheduleReading("native.test", scoreRecords(t, f.s))
	if err != nil {
		t.Fatal(err)
	}
	held := &commissioner.RawRecordedResults{Season: 2026, CutoverWeek: 2, Teams: []commissioner.RecordedTeamResult{
		{Franchise: f.a, Week: 1, Points: 1052500, Result: "W"}, {Franchise: f.b, Week: 1, Points: 990000, Result: "L"},
	}}
	if _, err := leaguemigration.SyncSchedule(t.Context(), migrationLedger{f.s}, "native.test", "Capture results", "creator", 2026, schedule.Weeks, held); err != nil {
		t.Fatal(err)
	}
	f.s.SetSeasonScoreCache(true)
	for range 2 {
		got, err := f.s.NativeResults(t.Context(), "native.test", 2026, 2)
		if err != nil {
			t.Fatal(err)
		}
		got.Revision = baseline.Revision
		if !reflect.DeepEqual(got, baseline) {
			t.Fatal("app-scored cutover week changed", got, baseline)
		}
		recorded, err := f.s.NativeResults(t.Context(), "native.test", 2026, 1)
		if err != nil || *recorded.Games[0].HomePoints != 1052500 || *recorded.Games[0].AwayPoints != 990000 {
			t.Fatal(recorded, err)
		}
		f.reopen(t)
	}
}

func TestRecordedResultsBackfillRetriesAndPreservesBoundary(t *testing.T) {
	t.Parallel()
	export := recordedImportFixture(t)
	weeks := export.Results
	export.Results = nil // league already imported by an older build
	svc, store, _ := migrationService(t, export)
	if _, err := svc.Import(t.Context(), export.LeagueID, "Old import", "creator"); err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	ledger := migrationLedger{store}
	importWeek := func(context.Context, int, time.Time) (int, bool) { return 2, true }
	sync := func() ([]commissioner.Receipt, error) {
		return mflsource.BackfillRecordedResults(t.Context(), ledger, league, "Refresh results", "creator", export.Season, weeks, importWeek)
	}
	before, _ := ledger.Records(t.Context(), league)
	store.batchFault = func(int) error { return migrationtest.ErrUnreachable }
	if _, err := sync(); err == nil {
		t.Fatal("failed batch accepted")
	}
	store.batchFault = nil
	unchanged, _ := ledger.Records(t.Context(), league)
	if !reflect.DeepEqual(before, unchanged) {
		t.Fatal("failed capture changed ledger")
	}
	for pass := range 2 {
		receipts, err := sync()
		want := 2
		if pass == 1 {
			want = 0
		}
		if err != nil || len(receipts) != want {
			t.Fatalf("retry %d: %d receipts: %v", pass, len(receipts), err)
		}
	}
	weeks[1].Games[0].Home = leaguefeed.Side{Franchise: "0001", Scored: true, Points: 99900, Result: "W"}
	weeks[1].Games[0].Away = leaguefeed.Side{Franchise: "0002", Scored: true, Points: 0, Result: "L"}
	if receipts, err := sync(); err != nil || len(receipts) != 0 {
		t.Fatal("refresh moved cutover", receipts, err)
	}
	weeks[0].Games[0].Home.Points = 10600
	if receipts, err := sync(); err != nil || len(receipts) != 2 {
		t.Fatal("source correction not captured", receipts, err)
	}
	records, _ := ledger.Records(t.Context(), league)
	held := commissioner.RecordedResults(records, export.Season)
	if held.CutoverWeek != 2 || len(held.Teams) != 2 || held.Teams[0].Points != leaguerules.Decimal(1060000) {
		t.Fatal(held)
	}
	weeks[0].Games[0].Home.Franchise = "unknown"
	if _, err := sync(); err == nil {
		t.Fatal("unknown identity accepted")
	}
	unchanged, _ = ledger.Records(t.Context(), league)
	if !reflect.DeepEqual(records, unchanged) {
		t.Fatal("bad source changed ledger")
	}
}

func TestRecordedVerdictStandsEvenWhenScoresAreLevel(t *testing.T) {
	t.Parallel()
	export := recordedImportFixture(t)
	export.Results[0].Games[0].Away.Points = 10525
	svc, store, _ := migrationService(t, export)
	if _, err := svc.Import(t.Context(), export.LeagueID, "Keep the source's decision", "creator"); err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	got, err := store.NativeResults(t.Context(), league, export.Season, 1)
	if err != nil || got.Games[0].Winner != "franchise.01" || *got.Games[0].HomePoints != *got.Games[0].AwayPoints {
		t.Fatal(got, err)
	}
	standings, err := store.NativeStandings(t.Context(), league, export.Season)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range standings.Computed {
		if row.Franchise == "franchise.01" && (row.Wins != 1 || row.PointsFor != 1052500) {
			t.Fatal(row)
		}
		if row.Franchise == "franchise.02" && (row.Losses != 1 || row.PointsFor != 1052500) {
			t.Fatal(row)
		}
	}
}

// A league imported by an older build in week 1 keeps app scoring for week 1 even
// though the source has since finished it: the boundary is the import week, never later.
func TestRecordedBackfillBoundaryIsImportWeekNotRefreshWeek(t *testing.T) {
	t.Parallel()
	export := recordedImportFixture(t)
	weeks := export.Results
	export.Results = nil
	svc, store, _ := migrationService(t, export)
	if _, err := svc.Import(t.Context(), export.LeagueID, "Old import", "creator"); err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	ledger := migrationLedger{store}
	before, _ := ledger.Records(t.Context(), league)
	unknown := func(context.Context, int, time.Time) (int, bool) { return 0, false }
	if _, err := mflsource.BackfillRecordedResults(t.Context(), ledger, league, "Refresh", "creator", export.Season, weeks, unknown); err == nil {
		t.Fatal("unknown import week guessed a boundary")
	}
	if after, _ := ledger.Records(t.Context(), league); !reflect.DeepEqual(before, after) {
		t.Fatal("unknown import week changed ledger")
	}
	weekOne := func(context.Context, int, time.Time) (int, bool) { return 1, true }
	if _, err := mflsource.BackfillRecordedResults(t.Context(), ledger, league, "Refresh", "creator", export.Season, weeks, weekOne); err != nil {
		t.Fatal(err)
	}
	records, _ := ledger.Records(t.Context(), league)
	held := commissioner.RecordedResults(records, export.Season)
	if held == nil || held.CutoverWeek != 1 || len(held.Teams) != 0 {
		t.Fatal(held)
	}
	got, err := store.NativeResults(t.Context(), league, export.Season, 1)
	if err != nil || got.Games[0].HomePoints != nil {
		t.Fatal("app-scored week replaced by recorded result", got, err)
	}
}
