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
	// ParamsFallback says why the arc could not use its run's stored parameters, when it could not.
	ParamsFallback string `json:"paramsFallback,omitempty"`
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
	RunParams func(context.Context, string) (params.Set, bool, error)
	Params    func() params.Set
	Games     func(context.Context, int) (int, error)
	Season    int
}

// ReadAgeArc resolves the MFL ID without changing it and reads the parameters its run scored with.
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
	if src.Games == nil {
		return AgeArc{}, fmt.Errorf("age arc: no NFL schedule games source")
	}
	games, err := src.Games(ctx, src.Season)
	if err != nil {
		return AgeArc{}, fmt.Errorf("age arc: NFL games per season: %w", err)
	}
	if games < 1 {
		return AgeArc{}, fmt.Errorf("age arc: NFL schedule has no games for the season")
	}
	pl, err := readArcPlayer(ctx, src, id, games)
	if err != nil {
		return AgeArc{}, err
	}
	set, fallback, err := runParamSet(ctx, src, run)
	if err != nil {
		return AgeArc{}, fmt.Errorf("age arc: run parameters: %w", err)
	}
	p, err := model.ParamsFrom(pl.Position, func(k string) (float64, error) { return set.GetPosition(k, string(pl.Position)) })
	if err != nil {
		return AgeArc{}, fmt.Errorf("age arc: parameters: %w", err)
	}
	arc := buildAgeArc(pl, p, src.Season, score.Now, games)
	arc.ParamsFallback = fallback
	return arc, nil
}

// readArcPlayer reads the player record for the run's season at the season's games count.
func readArcPlayer(ctx context.Context, src AgeArcSource, id string, games int) (model.Player, error) {
	book, err := src.Book(ctx, modelrun.BookQuery(modelrun.Spec{Season: src.Season, AsOf: time.Now(), LastWeek: games}))
	if err != nil {
		return model.Player{}, fmt.Errorf("age arc: player record: %w", err)
	}
	key, ok := book.ID(id)
	if !ok {
		return model.Player{}, fmt.Errorf("age arc: player record not found")
	}
	pl := book.Player(key)
	if math.IsNaN(pl.AgeAt(src.Season)) {
		return model.Player{}, fmt.Errorf("age arc: birth date unknown")
	}
	if !slices.Contains(model.Positions(), pl.Position) {
		return model.Player{}, fmt.Errorf("age arc: position not modeled")
	}
	return pl, nil
}

// runParamSet reads the parameters the run scored with. A run stored without its parameter set
// falls back to the current fitted set, and the arc says so.
func runParamSet(ctx context.Context, src AgeArcSource, run history.Run) (params.Set, string, error) {
	if src.RunParams != nil {
		set, ok, err := src.RunParams(ctx, run.ParamSetID)
		if err != nil {
			return params.Set{}, "", err
		}
		if ok {
			return set, "", nil
		}
	}
	if src.Params == nil {
		return params.Set{}, "", fmt.Errorf("the run stored no parameter set and no current set is available")
	}
	return src.Params(), "this run stored no parameter set; the arc uses current parameters", nil
}

// StoredParams reads a stored parameter set and returns it in the form the arc reads.
func StoredParams(ctx context.Context, read func(context.Context, string) (history.ParamSet, bool, error), id string) (params.Set, bool, error) {
	stored, ok, err := read(ctx, id)
	if err != nil {
		return params.Set{}, false, fmt.Errorf("read run parameter set: %w", err)
	}
	if !ok {
		return params.Set{}, false, nil
	}
	return params.SetOf(stored.Params), true, nil
}

// peakWindowShare is how close to the peak a season's level must be to sit inside the peak
// window: 90% of the peak, approved by Christopher for the HQ age arc.
const peakWindowShare = 0.90

// The peak window is the contiguous run containing the maximum whose levels are at least
// peakWindowShare of it. Past bars are an arc backcast, not historical production. Survival and
// the player record use the season's games count from the NFL schedule, not a fixed season length.
func buildAgeArc(pl model.Player, p model.Params, season int, now float64, games int) AgeArc {
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
		points[i] = AgeArcPoint{Season: year, Age: age, Level: level, Survival: p.Survives(age, pl.DraftPick, float64(games), level, pl.TenureAt(year))}
		if level > levels[peak] {
			peak = i
		}
	}
	first, last := peak, peak
	for first > 0 && levels[first-1] >= levels[peak]*peakWindowShare {
		first--
	}
	for last+1 < len(levels) && levels[last+1] >= levels[peak]*peakWindowShare {
		last++
	}
	return AgeArc{Age: pl.AgeAt(season), Points: points, PeakStart: points[first].Season, PeakEnd: points[last].Season}
}
