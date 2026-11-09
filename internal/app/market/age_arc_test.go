package market

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

type ageArcFixture struct {
	records []playerrecord.Record
	book    playerrecord.Book
	current params.Set
	games   int
}

func newAgeArcFixture(t *testing.T) ageArcFixture {
	t.Helper()
	records := []playerrecord.Record{
		{ID: "a", MFL: "0001", Listed: true, Position: domain.PosWR, Birth: time.Date(2001, 1, 3, 0, 0, 0, 0, time.UTC), Rookie: 2024, Draft: playerrecord.Draft{Overall: 10}},
		{ID: "b", MFL: "0002", Listed: true, Position: domain.PosRB, Birth: time.Date(1998, 6, 4, 0, 0, 0, 0, time.UTC), Rookie: 2020, Draft: playerrecord.Draft{Overall: 30}},
		{ID: "c", MFL: "0003", Listed: true, Position: domain.PosDE, Birth: time.Date(1995, 9, 5, 0, 0, 0, 0, time.UTC), Rookie: 2017, Draft: playerrecord.Draft{Overall: 50}},
	}
	obs := []model.Obs{{Player: "0003", Season: 2025, Measure: "context.contract_cap_pct", Value: 0.06}, {Player: "0003", Season: 2025, Measure: "context.contract_years", Value: 4}}
	book, err := playerrecord.NewBook(records, obs, nil, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	return ageArcFixture{records: records, book: book, current: params.DefaultSet(), games: 16}
}

func (f ageArcFixture) source(run params.Set, haveRun bool) AgeArcSource {
	return AgeArcSource{Season: 2026,
		Params:    func() params.Set { return f.current },
		RunParams: func(context.Context, string) (params.Set, bool, error) { return run, haveRun, nil },
		Games:     func(context.Context, int) (int, error) { return f.games, nil },
		Book:      func(context.Context, playerrecord.Query) (playerrecord.Book, error) { return f.book, nil },
		LatestRun: func(context.Context) (history.Run, bool, error) { return history.Run{ID: 1}, true, nil },
		Scores: func(context.Context, int64) ([]history.ModelScore, error) {
			return []history.ModelScore{
				{MFLID: "0001", Value: model.Value{Now: 0.8}}, {MFLID: "0002", Value: model.Value{Now: 0.6}}, {MFLID: "0003", Value: model.Value{Now: 0.7}},
			}, nil
		},
	}
}

func arcParams(t *testing.T, set params.Set, pos domain.Position) model.Params {
	t.Helper()
	p, err := model.ParamsFrom(pos, func(k string) (float64, error) { return set.GetPosition(k, string(pos)) })
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAgeArcRead(t *testing.T) {
	f := newAgeArcFixture(t)
	src := f.source(f.current, true)
	for _, r := range f.records {
		t.Run(r.MFL, func(t *testing.T) {
			got, err := ReadAgeArc(t.Context(), src, r.MFL)
			if err != nil {
				t.Fatal(err)
			}
			pl := f.book.Player(r.ID)
			p := arcParams(t, f.current, pl.Position)
			if got.Age != pl.AgeAt(2026) || len(got.Points) != 8 || got.ParamsFallback != "" {
				t.Fatalf("bad age, points or fallback: %+v", got)
			}
			for i, pt := range got.Points {
				if pt.Age != pl.AgeAt(pt.Season) || pt.Survival != p.Survives(pt.Age, pl.DraftPick, float64(f.games), pt.Level, pl.TenureAt(pt.Season)) {
					t.Fatalf("bad point: %+v", pt)
				}
				if i > 0 {
					prev := got.Points[i-1]
					step := 0.0
					if pl.Rookie == 0 || prev.Season >= pl.Rookie {
						step = p.ArcStep(prev.Age, pl.Experience(prev.Season))
					}
					want := model.Level(model.Score(prev.Level) + step)
					if math.Abs(pt.Level-want) > 1e-10 {
						t.Fatalf("season %d: got %g want %g", pt.Season, pt.Level, want)
					}
				}
			}
			if got.PeakStart > got.PeakEnd {
				t.Fatalf("invalid peak: %+v", got)
			}
		})
	}
}

func TestAgeArcSurvivalFollowsGamesPerSeason(t *testing.T) {
	f := newAgeArcFixture(t)
	for _, r := range f.records {
		t.Run(r.MFL, func(t *testing.T) {
			f.games = 16
			sixteen, err := ReadAgeArc(t.Context(), f.source(f.current, true), r.MFL)
			if err != nil {
				t.Fatal(err)
			}
			f.games = 15
			fifteen, err := ReadAgeArc(t.Context(), f.source(f.current, true), r.MFL)
			if err != nil {
				t.Fatal(err)
			}
			pl := f.book.Player(r.ID)
			p := arcParams(t, f.current, pl.Position)
			changed := false
			for i, pt := range sixteen.Points {
				if pt.Survival != p.Survives(pt.Age, pl.DraftPick, 16, pt.Level, pl.TenureAt(pt.Season)) {
					t.Fatalf("16 games: bad survival: %+v", pt)
				}
				if fifteen.Points[i].Survival != p.Survives(pt.Age, pl.DraftPick, 15, pt.Level, pl.TenureAt(pt.Season)) {
					t.Fatalf("15 games: bad survival: %+v", fifteen.Points[i])
				}
				changed = changed || pt.Survival != fifteen.Points[i].Survival
			}
			if !changed {
				t.Fatal("changing the games per season did not change survival")
			}
		})
	}
}

func TestAgeArcReadsRunParams(t *testing.T) {
	f := newAgeArcFixture(t)
	values := f.current.Values()
	for k := range values {
		if strings.HasPrefix(k, "model.arc.age@") {
			values[k] += 0.5
		}
	}
	runSet := params.SetOf(values)
	if runSet.Values()["model.arc.age@WR"] == f.current.Values()["model.arc.age@WR"] {
		t.Fatal("test parameter differs from current")
	}
	r := f.records[0]
	got, err := ReadAgeArc(t.Context(), f.source(runSet, true), r.MFL)
	if err != nil {
		t.Fatal(err)
	}
	fallback, err := ReadAgeArc(t.Context(), f.source(runSet, false), r.MFL)
	if err != nil {
		t.Fatal(err)
	}
	if got.ParamsFallback != "" {
		t.Fatalf("run params marked as fallback: %q", got.ParamsFallback)
	}
	if fallback.ParamsFallback == "" {
		t.Fatal("missing run params not reported")
	}
	if got.Points[3].Level == fallback.Points[3].Level {
		t.Fatal("run parameters did not change the arc from current parameters")
	}
	pl := f.book.Player(r.ID)
	runParams := arcParams(t, runSet, pl.Position)
	step := runParams.ArcStep(got.Points[2].Age, pl.Experience(got.Points[2].Season))
	if want := model.Level(model.Score(got.Points[2].Level) + step); math.Abs(got.Points[3].Level-want) > 1e-10 {
		t.Fatalf("level %g not from run params; want %g", got.Points[3].Level, want)
	}
	current := arcParams(t, f.current, pl.Position)
	step = current.ArcStep(fallback.Points[2].Age, pl.Experience(fallback.Points[2].Season))
	if want := model.Level(model.Score(fallback.Points[2].Level) + step); math.Abs(fallback.Points[3].Level-want) > 1e-10 {
		t.Fatalf("fallback level %g not from current params; want %g", fallback.Points[3].Level, want)
	}
}
