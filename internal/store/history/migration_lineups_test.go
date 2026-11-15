package history

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
)

func TestMigrationLineupSyncIdempotenceAndSeasonIsolation(t *testing.T) {
	t.Parallel()
	export, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	svc, store, _ := migrationService(t, export)
	if _, err := svc.Import(t.Context(), export.LeagueID, "Import", "creator"); err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	records, err := store.nativeRecords(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	assets, err := commissioner.NativeAssetsReading(league, records)
	if err != nil {
		t.Fatal(err)
	}
	p := assets.Snapshot.Players[0]
	lineup := commissioner.RawLineupChange{Season: export.Season, Week: 1, Franchise: p.Franchise, Starters: []string{p.ID}}
	store.SetLineupEnvironment(&lineupEnvFake{})
	ledger := migrationLedger{store}
	for pass := range 2 {
		receipts, err := leaguemigration.SyncLineups(t.Context(), ledger, league, "Import starters", "creator", export.Season, []commissioner.RawLineupChange{lineup})
		want := 1
		if pass == 1 {
			want = 0
		}
		if err != nil || len(receipts) != want {
			t.Fatalf("pass %d: %v %v", pass, receipts, err)
		}
	}
	records, err = store.nativeRecords(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	held, found, err := commissioner.RecordedLineupFor(records, p.Franchise, export.Season, 1)
	if err != nil || !found || held.By != "creator" || !held.Corrected || held.Starters[0] != p.ID {
		t.Fatalf("%+v %t %v", held, found, err)
	}
	if _, found, err := commissioner.RecordedLineupFor(records, p.Franchise, export.Season+1, 1); err != nil || found {
		t.Fatal("season leaked", err)
	}
	for _, tc := range []struct {
		name, principal string
		season          int
		player          string
	}{
		{"next season", "creator", export.Season + 1, p.ID},
		{"unauthorized", "outsider", export.Season, p.ID},
		{"not rostered", "creator", export.Season, "999999"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := lineup
			changed.Season, changed.Starters = tc.season, []string{tc.player}
			changed.Week = 2
			if _, err := leaguemigration.SyncLineups(t.Context(), ledger, league, "Import starters", tc.principal, tc.season, []commissioner.RawLineupChange{changed}); err == nil {
				t.Fatal("accepted")
			}
			after, err := store.nativeRecords(t.Context(), league)
			if err != nil || len(after) != len(records) {
				t.Fatal("failed sync changed ledger", err)
			}
		})
	}
	t.Run("one refused lineup does not block another", func(t *testing.T) {
		bad, good := lineup, lineup
		bad.Week, bad.Starters = 3, []string{"999999"}
		good.Week = 2
		receipts, err := leaguemigration.SyncLineups(t.Context(), ledger, league, "Import starters", "creator", export.Season, []commissioner.RawLineupChange{bad, good})
		if err == nil || len(receipts) != 1 {
			t.Fatalf("receipts %v err %v", receipts, err)
		}
		after, err := store.nativeRecords(t.Context(), league)
		if err != nil || len(after) != len(records)+1 {
			t.Fatal("successful lineup missing", err)
		}
		if _, found, err := commissioner.RecordedLineupFor(after, p.Franchise, export.Season, 3); err != nil || found {
			t.Fatal("refused lineup recorded", err)
		}
		receipts, err = leaguemigration.SyncLineups(t.Context(), ledger, league, "Import starters", "creator", export.Season, []commissioner.RawLineupChange{bad, good})
		if err == nil || len(receipts) != 0 {
			t.Fatal("partial retry duplicated lineup", receipts, err)
		}
	})

}
