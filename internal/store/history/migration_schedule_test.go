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

func TestScheduleBackfillRollbackAndChangedPublication(t *testing.T) {
	t.Parallel()
	export, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	svc, store, _ := migrationService(t, export)
	if _, err = svc.Import(t.Context(), export.LeagueID, "Import league", "creator"); err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, 2026)
	ledger := migrationLedger{store}
	raw := migrationSchedule(t)
	sync := func(principal string, season int) ([]commissioner.Receipt, error) {
		return mflsource.BackfillSchedule(t.Context(), ledger, league, "Carry schedule", principal, season, raw)
	}
	for _, tc := range []struct {
		name, principal string
		season          int
		fault           bool
	}{
		{"authorization", "outsider", 2026, false},
		{"wrong season", "creator", 2027, false},
		{"publication fails after draft", "creator", 2026, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.fault {
				store.batchFault = func(int) error { return migrationtest.ErrUnreachable }
			}
			if _, err := sync(tc.principal, tc.season); err == nil {
				t.Fatal("failed update accepted")
			}
			store.batchFault = nil
			held, err := store.nativeRecords(t.Context(), league)
			if err != nil || len(held) != 2 {
				t.Fatalf("failed update changed ledger: %d %v", len(held), err)
			}
		})
	}
	if _, err := sync("creator", 2026); err != nil {
		t.Fatal(err)
	}
	raw[0].Matchups[0].Franchises[0].IsHome = "1"
	raw[0].Matchups[0].Franchises[1].IsHome = "0"
	receipts, err := sync("creator", 2026)
	if err != nil || len(receipts) != 2 {
		t.Fatalf("changed schedule: %v %v", receipts, err)
	}
	records, err := store.nativeRecords(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	schedule, err := commissioner.ScheduleReading(league, records)
	if err != nil || schedule.Weeks[0].Games[0].Home != "franchise.01" {
		t.Fatal("new publication not visible", schedule, err)
	}
	retry, err := sync("creator", 2026)
	if err != nil || len(retry) != 0 {
		t.Fatal("changed publication retry duplicated", retry, err)
	}
	if len(records) != 6 {
		t.Fatal("unexpected ledger length", len(records))
	}
}
