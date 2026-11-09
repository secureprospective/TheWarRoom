package appvalue

import (
	"context"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/modelrun"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type evidence struct {
	weeks map[int][]source.PlayerWeek
	ids   map[string]string
}

func (e evidence) HeldWeeks(_ context.Context, year int) ([]source.PlayerWeek, string, error) {
	return e.weeks[year], "held", nil
}
func (e evidence) SourceIDs(context.Context, string) (map[string]string, error) { return e.ids, nil }

// Real 14432 settings, roster IDs, ownership and paid contracts are imported unchanged.
// Production is a controlled 25-WR panel: CY *.1 in the held 14432 rules, independently
// totaled for the mirror. This is adapter parity, not a live NFL/MFL scoring audit.
func TestImported14432ValueParity(t *testing.T) {
	export, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	export.Base = legacySettings(t)
	plan, err := mflsource.Prepare(export)
	if err != nil {
		t.Fatal(err)
	}
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "history.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	store := history.New(pools, reg)
	store.SetClock(func() time.Time { return migrationtest.ReadAt })
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID("14432", season)
	_, err = store.ExecuteCommissionerBatch(t.Context(), []commissioner.RawCommand{
		{League: league, Command: commissioner.Create, Args: plan.Create, Request: "create", Reason: "Parity fixture", Effective: "now", Visibility: "league"},
		{League: league, Command: commissioner.NativeSeed, Args: commissioner.RawArgs{Native: &plan.Seed}, Request: "seed", Reason: "Parity fixture", Effective: "now", Visibility: "league"},
	}, "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	records := func(ctx context.Context, league string) ([]commissioner.Record, error) {
		events, err := store.Events(ctx, league, commissioner.Stream, 0)
		if err != nil {
			return nil, err
		}
		out := make([]commissioner.Record, 0, len(events))
		for _, event := range events {
			out = append(out, *event.Payload.Commissioner)
		}
		return out, nil
	}
	l, f := identities()
	clear(l)
	f.rows = map[int][]history.Feature{}
	e := evidence{weeks: map[int][]source.PlayerWeek{}, ids: map[string]string{}}
	in := Inputs{Owner: map[string]string{}, Contracts: map[string]state.PlayerState{}}
	for i, p := range plan.Seed.Players[:regulars] {
		l[p.ID] = normalize.PlayerFacts{Name: p.ID, Position: domain.PosWR}
		f.rows[0] = append(f.rows[0], history.Feature{PlayerID: p.ID, Measure: "context.birth_date", Text: "2000-03-01"}, history.Feature{PlayerID: p.ID, Measure: "context.rookie_season", Value: 2022})
		e.ids[p.ID] = gsisOf(i)
		// CY *.1, 16 played weeks; whole-number PPG avoids irrelevant summation noise.
		f.rows[season-1] = append(f.rows[season-1], history.Feature{PlayerID: p.ID, Season: season - 1, Measure: "outcome.fantasy_points", Value: float64((2 + i) * 16)})
		for w := 1; w <= 16; w++ {
			e.weeks[season-1] = append(e.weeks[season-1], receivingWeek(gsisOf(i), season-1, w, int64(20+10*i)))
			f.rows[season-1] = append(f.rows[season-1], history.Feature{PlayerID: p.ID, Season: season - 1, Week: w, Measure: "exposure.offense_snaps", Value: 40})
		}
		in.Owner[p.ID] = p.Franchise
		in.Contracts[p.ID] = state.PlayerState{MFLID: p.ID, Salary: p.Salary, ExpirationYear: p.ExpirationYear}
	}
	src := Source{Facts: modelrun.Held{Listing: l, Features: f}, Held: e, Records: records, Params: params.DefaultSet, Now: func() time.Time { return migrationtest.ReadAt }}
	before, err := records(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	got, err := src.Read(t.Context(), league, "", season)
	if err != nil {
		t.Fatal(err)
	}
	runner, err := modelrun.New(rosterState{in}, src.Facts, f, "parity")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(t.Context(), modelrun.Spec{Season: season, AsOf: migrationtest.ReadAt, LastWeek: lastWeek, Params: params.DefaultSet()}); err != nil {
		t.Fatal(err)
	}
	wants := map[string]history.ModelScore{}
	for _, score := range f.written[0].Scores {
		wants[score.MFLID] = score
	}
	checked, maxNow, maxDyn := 0, 0.0, 0.0
	for _, p := range got.Players {
		want, ok := wants[p.Player]
		if !ok {
			continue
		}
		if !p.Available || want.Unavailable {
			t.Fatalf("unavailable panel player %s: %+v / %+v", p.Player, p, want)
		}
		maxNow = math.Max(maxNow, math.Abs(p.NowPPG-want.NowPPG))
		maxDyn = math.Max(maxDyn, math.Abs(p.DynastyPP-want.DynastyPPG))
		checked++
	}
	t.Logf("14432 imported paid-roster panel: players=%d max Now difference=%.17g max dynasty difference=%.17g", checked, maxNow, maxDyn)
	if checked != regulars || maxNow != 0 || maxDyn != 0 {
		t.Fatalf("parity: %d players; Now %.17g dynasty %.17g", checked, maxNow, maxDyn)
	}
	t.Run("cache", func(t *testing.T) { checkSourceCache(t, src, league, got) })
	t.Run("saved", func(t *testing.T) { checkSavedPass(t, src, league, got) })
	t.Run("preparation", func(t *testing.T) { checkPreparation(t, src, league, got) })
	after, err := records(t.Context(), league)
	if err != nil || len(after) != len(before) {
		t.Fatalf("value read wrote ledger: %d -> %d, %v", len(before), len(after), err)
	}
}
