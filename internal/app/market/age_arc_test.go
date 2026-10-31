package market

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

func TestAgeArcRead(t *testing.T) {
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
	set := params.DefaultSet()
	src := AgeArcSource{Season: 2026, Params: func() params.Set { return set },
		Book:      func(context.Context, playerrecord.Query) (playerrecord.Book, error) { return book, nil },
		LatestRun: func(context.Context) (history.Run, bool, error) { return history.Run{ID: 1}, true, nil },
		Scores: func(context.Context, int64) ([]history.ModelScore, error) {
			return []history.ModelScore{
				{MFLID: "0001", Value: model.Value{Now: 0.8}}, {MFLID: "0002", Value: model.Value{Now: 0.6}}, {MFLID: "0003", Value: model.Value{Now: 0.7}},
			}, nil
		},
	}
	for _, r := range records {
		t.Run(r.MFL, func(t *testing.T) {
			got, err := ReadAgeArc(t.Context(), src, r.MFL)
			if err != nil {
				t.Fatal(err)
			}
			pl := book.Player(r.ID)
			p, err := model.ParamsFrom(pl.Position, func(k string) (float64, error) { return set.GetPosition(k, string(pl.Position)) })
			if err != nil {
				t.Fatal(err)
			}
			if got.Age != pl.AgeAt(2026) || len(got.Points) != 8 {
				t.Fatalf("bad age or points: %+v", got)
			}
			for i, pt := range got.Points {
				if pt.Age != pl.AgeAt(pt.Season) || pt.Survival != p.Survives(pt.Age, pl.DraftPick, 17, pt.Level, pl.TenureAt(pt.Season)) {
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
