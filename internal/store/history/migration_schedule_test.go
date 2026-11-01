package history

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
)

func migrationSchedule(t *testing.T) []leagueschedule.RawScheduleWeek {
	t.Helper()
	raw, err := leagueschedule.Parse([]byte(`{"schedule":{"weeklySchedule":[{"week":"1","matchup":{"franchise":[{"id":"0001","isHome":"0","score":"120"},{"id":"0002","isHome":"1","score":"90"}]}},{"week":"2"}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestMigrationImportsAndBackfillsSchedule(t *testing.T) {
	t.Parallel()
	for _, existing := range []bool{false, true} {
		name := "new"
		if existing {
			name = "existing"
		}
		t.Run(name, func(t *testing.T) {
			export, err := migrationtest.Legacy()
			if err != nil {
				t.Fatal(err)
			}
			raw := migrationSchedule(t)
			if !existing {
				export.Schedule = raw
			}
			svc, store, _ := migrationService(t, export)
			receipts, err := svc.Import(t.Context(), export.LeagueID, "Import the league", "creator")
			want := 4
			if existing {
				want = 2
			}
			if err != nil || len(receipts) != want {
				t.Fatalf("import %d: %v", len(receipts), err)
			}
			league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
			before, err := store.NativeAssets(t.Context(), league)
			if err != nil {
				t.Fatal(err)
			}
			ledger := migrationLedger{store}
			for pass := range 2 {
				receipts, err = mflsource.BackfillSchedule(t.Context(), ledger, league, "Carry the schedule", "creator", 2026, raw)
				want = 0
				if existing && pass == 0 {
					want = 2
				}
				if err != nil || len(receipts) != want {
					t.Fatalf("backfill pass %d, %d receipts: %v", pass, len(receipts), err)
				}
			}
			if !existing {
				retry, retryErr := svc.Import(t.Context(), export.LeagueID, "Import the league", "creator")
				if retryErr != nil || len(retry) != 0 {
					t.Fatalf("re-import appended changes: %v %v", retry, retryErr)
				}
			}
			records, err := store.nativeRecords(t.Context(), league)
			if err != nil {
				t.Fatal(err)
			}
			held, err := commissioner.ScheduleReading(league, records)
			if err != nil || len(held.Weeks) != 2 || len(held.Weeks[0].Games) != 1 || len(held.Weeks[1].Games) != 0 {
				t.Fatalf("published: %+v %v", held, err)
			}
			after, err := store.NativeAssets(t.Context(), league)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before.Snapshot, after.Snapshot) {
				t.Fatal("schedule update changed starting assets")
			}
			raw[0].Matchups[0].Franchises[0].FranchiseID = "9999"
			if _, err = mflsource.BackfillSchedule(t.Context(), ledger, league, "Bad schedule", "creator", 2026, raw); err == nil {
				t.Fatal("unknown team accepted")
			}
			unchanged, _ := store.nativeRecords(t.Context(), league)
			if len(unchanged) != len(records) {
				t.Fatal("failed mapping changed ledger")
			}
		})
	}
}
