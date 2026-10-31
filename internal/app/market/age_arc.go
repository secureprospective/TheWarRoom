package market

import (
	"context"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/modelrun"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// AgeArc is the eight-season talent arc, anchored to the latest held model score.
// Survival is the conditional probability of playing the following season, not a cumulative probability.
type AgeArc struct {
	Age       float64       `json:"age"`
	Points    []AgeArcPoint `json:"points"`
	PeakStart int           `json:"peakStart"`
	PeakEnd   int           `json:"peakEnd"`
}

type AgeArcPoint struct {
	Season   int     `json:"season"`
	Age      float64 `json:"age"`
	Level    float64 `json:"level"`
	Survival float64 `json:"survival"`
}

// AgeArcSource reads held inputs; it never refreshes the model or the player record.
type AgeArcSource struct {
	Book      func(context.Context, playerrecord.Query) (playerrecord.Book, error)
	LatestRun func(context.Context) (history.Run, bool, error)
	Scores    func(context.Context, int64) ([]history.ModelScore, error)
	Params    func() params.Set
	Season    int
}

// ReadAgeArc resolves the MFL ID without changing it and reads current fitted parameters.
func ReadAgeArc(ctx context.Context, src AgeArcSource, id string) (AgeArc, error) {
	run, ok, err := src.LatestRun(ctx)
	if err != nil {
		return AgeArc{}, fmt.Errorf("age arc: latest run: %w", err)
	}
	if !ok {
		return AgeArc{}, fmt.Errorf("age arc: no model run for this season")
	}
	scores, err := src.Scores(ctx, run.ID)
	if err != nil {
		return AgeArc{}, fmt.Errorf("age arc: scores: %w", err)
	}
	i := slices.IndexFunc(scores, func(s history.ModelScore) bool { return s.MFLID == id })
	if i < 0 {
		return AgeArc{}, fmt.Errorf("age arc: player not in the model run")
	}
	score := scores[i]
	if score.Unavailable {
		return AgeArc{}, fmt.Errorf("age arc: %s", score.Reason)
	}
	book, err := src.Book(ctx, modelrun.BookQuery(modelrun.Spec{Season: src.Season, AsOf: time.Now(), LastWeek: 17}))
	if err != nil {
		return AgeArc{}, fmt.Errorf("age arc: player record: %w", err)
	}
	key, ok := book.ID(id)
	if !ok {
		return AgeArc{}, fmt.Errorf("age arc: player record not found")
	}
	pl := book.Player(key)
	if math.IsNaN(pl.AgeAt(src.Season)) {
		return AgeArc{}, fmt.Errorf("age arc: birth date unknown")
	}
	if !slices.Contains(model.Positions(), pl.Position) {
		return AgeArc{}, fmt.Errorf("age arc: position not modeled")
	}
	set := src.Params()
	p, err := model.ParamsFrom(pl.Position, func(k string) (float64, error) { return set.GetPosition(k, string(pl.Position)) })
	if err != nil {
		return AgeArc{}, fmt.Errorf("age arc: parameters: %w", err)
	}
	return buildAgeArc(pl, p, src.Season, score.Now), nil
}

// The peak window is the contiguous run containing the maximum whose levels are at least 90% of it.
// Past bars are an arc backcast, not historical production. All seasons assume 17 games.
func buildAgeArc(pl model.Player, p model.Params, season int, now float64) AgeArc {
	points := make([]AgeArcPoint, 8)
	levels := make([]float64, 8)
	levels[2] = now
	for i := 1; i >= 0; i-- {
		year := season + i - 2
		step := 0.0
		if pl.Rookie == 0 || year >= pl.Rookie {
			step = p.ArcStep(pl.AgeAt(year), pl.Experience(year))
		}
		levels[i] = model.Level(model.Score(levels[i+1]) - step)
	}
	for i := 3; i < len(points); i++ {
		year := season + i - 3
		step := 0.0
		if pl.Rookie == 0 || year >= pl.Rookie {
			step = p.ArcStep(pl.AgeAt(year), pl.Experience(year))
		}
		levels[i] = model.Level(model.Score(levels[i-1]) + step)
	}
	peak := 0
	for i, level := range levels {
		year := season + i - 2
		age := pl.AgeAt(year)
		points[i] = AgeArcPoint{Season: year, Age: age, Level: level, Survival: p.Survives(age, pl.DraftPick, 17, level, pl.TenureAt(year))}
		if level > levels[peak] {
			peak = i
		}
	}
	first, last := peak, peak
	for first > 0 && levels[first-1] >= levels[peak]*0.9 {
		first--
	}
	for last+1 < len(levels) && levels[last+1] >= levels[peak]*0.9 {
		last++
	}
	return AgeArc{Age: pl.AgeAt(season), Points: points, PeakStart: points[first].Season, PeakEnd: points[last].Season}
}
