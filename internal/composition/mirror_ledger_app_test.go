package composition

import (
	"testing"
	"time"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/mflref/seed"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func heldMirrorSource(t *testing.T, at time.Time) nativeapp.SeedSource {
	t.Helper()
	id, err := playerid.New("0099")
	if err != nil {
		t.Fatal(err)
	}
	return nativeapp.SeedSource{
		Available: func() bool { return true },
		Roster: func() (state.MirrorSnapshot, string) {
			return state.MirrorSnapshot{Season: 2026, Players: []state.MirrorPlayer{{MFLID: "0099", FranchiseID: "0002",
				RosterStatus: domain.RosterActive, Salary: 5, ContractYear: 2028}}}, at.Format(time.RFC3339Nano)
		},
		Season: func() int { return 2026 },
		Draft: func() (season.Feed[[]draftreports.Pick], season.Feed[[]draftreports.FuturePick]) {
			return season.Feed[[]draftreports.Pick]{FetchedAt: at}, season.Feed[[]draftreports.FuturePick]{FetchedAt: at}
		},
		Config: func() league.RawConfig {
			return league.RawConfig{Name: "Held League", Franchises: []league.Franchise{{ID: "0001", Name: "One"},
				{ID: "0002", Name: "Two"}}}
		},
		Directory: func() []snapshot.Player { return []snapshot.Player{{ID: id, Name: "Player, Named"}} },
	}
}

// The mirror is a league setting: the held MFL league replays through the same native readings.
func TestMirrorLedgerAnswersNativeReadingsFromHeldData(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	records, err := mirrorRecords(heldMirrorSource(t, at))
	if err != nil {
		t.Fatal(err)
	}
	now := at.Add(time.Hour)
	rules, err := commissioner.RulesReading(MirrorLeague, records, now)
	if err != nil || rules.Name != "Held League" || len(rules.Sections) == 0 {
		t.Fatalf("rules: %v %+v", err, rules.Name)
	}
	assets, err := commissioner.NativeAssetsReading(MirrorLeague, records, now)
	if err != nil || assets.Snapshot == nil || len(assets.Snapshot.Players) != 1 || len(assets.Snapshot.Franchises) != 2 {
		t.Fatalf("assets: %v %+v", err, assets)
	}
	if f := assets.Snapshot.Franchises[1]; f.HeldID != "0002" || assets.Snapshot.Players[0].Franchise != f.ID {
		t.Fatalf("held franchise IDs map to the seeded franchise: %+v %+v", f, assets.Snapshot.Players[0])
	}
	if _, err := commissioner.NativeCapCheckReading(commissioner.RawNativeCapScope{League: MirrorLeague,
		ViewerZone: "America/Chicago"}, records, now); err != nil {
		t.Fatalf("cap: %v", err)
	}
	if _, err := commissioner.NativeDirectoryReading(MirrorLeague, records, now); err != nil {
		t.Fatalf("directory: %v", err)
	}
}
