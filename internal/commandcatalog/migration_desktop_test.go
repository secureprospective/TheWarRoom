package catalog

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type baseFixture struct {
	reader *migrationtest.Source
	base   leaguerules.RuleSet
}

func (f baseFixture) Read(ctx context.Context, now time.Time) (leaguemigration.Candidate, error) {
	return mflsource.ReadCandidate(ctx, f.reader, now, f.base)
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
