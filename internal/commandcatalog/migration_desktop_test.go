package catalog

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type baseFixture struct {
	reader *migrationtest.Source
	base   leaguerules.RuleSet
}

func (f baseFixture) Read(ctx context.Context, now time.Time) (leaguemigration.Candidate, error) {
	return mflsource.ReadCandidate(ctx, f.reader, now, f.base)
}

type importedLineupEnvironment struct{}

func (importedLineupEnvironment) PlayerFacts() nativecheck.Facts { return nativecheck.Facts{} }

func (importedLineupEnvironment) LineupFacts(context.Context, int, int) (commissioner.LineupFacts, error) {
	return commissioner.LineupFacts{}, nil
}

type importedLineupScoreSource struct{}

func (importedLineupScoreSource) Lineup(_ context.Context, records []commissioner.Record, franchise string,
	week int) (commissioner.WeekLineup, error) {
	return commissioner.LedgerWeekLineup(records, franchise, 2026, week)
}

func (importedLineupScoreSource) PlayerScore(_ context.Context, _ []commissioner.Record, player string,
	week int) (commissioner.NativePlayerScore, error) {
	points := leaguerules.Decimal(10000)
	return commissioner.NativePlayerScore{Player: player, Season: 2026, Week: week, Status: "Complete",
		Words: "1 point", Points: &points, Position: leaguerules.RB}, nil
}

func TestMigrationImportsCurrentLineupsAsNativeEventsAndScoresThem(t *testing.T) {
	t.Parallel()
	export, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	var starter string
	for _, roster := range export.Rosters {
		if roster.Franchise == "0001" && roster.Status == "ROSTER" {
			starter = roster.Player
			break
		}
	}
	if starter == "" {
		t.Fatal("fixture has no active player on franchise 0001")
	}
	id, err := playerid.New(starter)
	if err != nil {
		t.Fatal(err)
	}
	export.LiveLineups = &leaguefeed.Lineups{Week: 1, Matchups: []leaguefeed.Matchup{{Home: "0001", Away: "0002"}},
		Franchises: []leaguefeed.Lineup{
			{Franchise: "0001", Starters: []playerid.PlayerID{id}},
			{Franchise: "0002"},
		}}
	export.Schedule = []leagueschedule.RawScheduleWeek{{Week: "1", Matchups: []leagueschedule.RawMatchup{{
		Franchises: [2]leagueschedule.RawMatchupSide{{FranchiseID: "0001", IsHome: "1"}, {FranchiseID: "0002", IsHome: "0"}},
	}}}}
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "history.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	store := history.New(pools, &measures.Registry{})
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	store.SetLineupEnvironment(importedLineupEnvironment{})
	now := migrationtest.ReadAt.Add(time.Hour)
	store.SetClock(func() time.Time { return now })
	service := &leaguemigration.Service{Now: func() time.Time { return now },
		Ledger: leagueservice.MigrationLedger{Store: store},
		SourceWithBase: func(_ string, base leaguerules.RuleSet) (leaguemigration.Source, error) {
			return baseFixture{reader: &migrationtest.Source{Export: export}, base: base}, nil
		}}
	preview, err := service.Prepare(t.Context(), export.LeagueID, "default32")
	if err != nil || !preview.Ready || len(preview.Lineups) != 1 ||
		len(preview.Lineups[0].Starters) != 1 || preview.Lineups[0].Week != 1 ||
		preview.Lineups[0].Franchise != "franchise.01" || preview.Lineups[0].Starters[0].Player != starter {
		t.Fatalf("import preview: %+v, %v", preview, err)
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	if _, err := service.ImportWithOptions(t.Context(), export.LeagueID, "Import fixture", "commish",
		leaguemigration.ImportOptions{Preset: "default32"}); err != nil {
		t.Fatal(err)
	}
	records, err := service.Ledger.Records(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	lineup, found, err := commissioner.RecordedLineupFor(records, "franchise.01", 2026, 1)
	if err != nil || !found || len(lineup.Starters) != 1 || lineup.Starters[0] != starter {
		t.Fatalf("imported native lineup: %+v, found=%v, err=%v", lineup, found, err)
	}
	absent, found, err := commissioner.RecordedLineupFor(records, "franchise.02", 2026, 1)
	if err != nil || found || len(absent.Starters) != 0 {
		t.Fatalf("missing MFL lineup was imputed: %+v, found=%v, err=%v", absent, found, err)
	}
	result, err := commissioner.NativeResultsReading(t.Context(), league, records, importedLineupScoreSource{}, 2026, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	status := map[string]string{}
	for _, score := range result.Scores {
		status[score.Franchise] = score.Status
	}
	if status["franchise.01"] != "Complete" || status["franchise.02"] != "Incomplete" {
		t.Fatalf("week score statuses: %+v", status)
	}
}

func TestDesktopMigrationBaseKeyAndAnswers(t *testing.T) {
	t.Parallel()
	export, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "history.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	store := history.New(pools, &measures.Registry{})
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	now := migrationtest.ReadAt.Add(time.Hour)
	store.SetClock(func() time.Time { return now })
	keys := mflkey.NewMemory("other", "2026")
	var readBase leaguerules.RuleSet
	service := &leaguemigration.Service{Now: func() time.Time { return now },
		Ledger: leagueservice.MigrationLedger{Store: store},
		SourceWithBase: func(id string, base leaguerules.RuleSet) (leaguemigration.Source, error) {
			if id != export.LeagueID {
				t.Fatalf("wrong league %s", id)
			}
			readBase = base
			return baseFixture{&migrationtest.Source{Export: export}, base}, nil
		}}
	in := MigrationInputs{Service: service, Principal: func() (string, error) { return "commish", nil },
		SaveKey: func(ctx context.Context, id string, key string) error {
			return keys.ForLeague(id).Set(ctx, mflkey.Key(key))
		}}
	raw := json.RawMessage(`{"mflLeague":"14432","preset":"default32","apiKey":"fixture-key-never-real"}`)
	body, err := RunMigration(t.Context(), in, leaguemigration.CommandPrepare, raw)
	if err != nil {
		t.Fatal(err)
	}
	var prep leaguemigration.Preparation
	if err := json.Unmarshal(body, &prep); err != nil {
		t.Fatal(err)
	}
	if !prep.Ready || prep.Preset != "default32" || readBase.Roster.Limits.Value.Max.Value != 44 {
		t.Fatalf("base not threaded: %+v", prep)
	}
	key, err := keys.ForLeague(export.LeagueID).Get(t.Context())
	if err != nil || key.Reveal() != "fixture-key-never-real" {
		t.Fatal("league key not saved")
	}
	if _, err := keys.Get(t.Context()); err == nil {
		t.Fatal("key crossed leagues")
	}
	if readBase.IR.Slots.Value != 12 {
		t.Fatal("wrong base")
	}
	// A setting MFL does hold cannot be smuggled in through answers.
	bad := json.RawMessage(`{"mflLeague":"14432","preset":"default32","reason":"Move","answers":[{"field":"cap.amount","value":1}]}`)
	if _, err := RunMigration(t.Context(), in, leaguemigration.CommandImport, bad); err == nil {
		t.Fatal("overrode MFL")
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	if rows, err := service.Ledger.Records(t.Context(), league); err != nil || len(rows) != 0 {
		t.Fatalf("rejected import wrote records: %v %d", err, len(rows))
	}
	// Leave one rule genuinely unanswered in the base; supply it with the import.
	service.SourceWithBase = func(_ string, base leaguerules.RuleSet) (leaguemigration.Source, error) {
		base.Contracts.MaximumYears = leaguerules.Needs[int]("Choose contract years")
		return baseFixture{&migrationtest.Source{Export: export}, base}, nil
	}
	good := json.RawMessage(`{"mflLeague":"14432","preset":"default32","reason":"Move the league","answers":[{"field":"contracts.maximumYears","value":7}]}`)
	if _, err := RunMigration(t.Context(), in, leaguemigration.CommandImport, good); err != nil {
		t.Fatal(err)
	}
	rows, err := service.Ledger.Records(t.Context(), league)
	if err != nil || len(rows) != 3 {
		t.Fatalf("import batch: %v %d", err, len(rows))
	}
	rules, err := commissioner.RulesNow(rows, now)
	if err != nil || rules.Contracts.MaximumYears.Value != 7 || rules.Cap.Amount.Value != 12500000000 {
		t.Fatalf("answers or MFL settings lost: %v", err)
	}
}
