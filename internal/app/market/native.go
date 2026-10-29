package market

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/pickvalue"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// Native is the native value model as the asset board reads it: each player's dynasty value from
// the latest model run, and Pick Value v2 for picks, both in dynasty points per game.
type Native struct {
	values  map[string]float64 // dynasty points per game by the board's player ID (MFL's)
	picks   *pickvalue.Model   // nil when it could not be fitted; pickWhy says why
	pickWhy string
	season  int
	teams   int
	seasons int
}

// NativeInput is what the native model is built from: the latest model run's dynasty values by
// MFL ID, every player record, and the run's season, horizon and discount, and the league's size.
type NativeInput struct {
	Values   map[string]float64
	Records  []playerrecord.Record
	Season   int
	Seasons  int
	Discount float64
	Teams    int
	History  []pickvalue.Pick
	PickWhy  string
}

// NewNative builds the native model. Pick Value v2 is fitted on the records' draft capital and the
// run's values; when it cannot be (no draft history or no recent class) players are still valued
// and picks say why.
func NewNative(in NativeInput) Native {
	n := Native{values: in.Values, season: in.Season, teams: in.Teams, seasons: in.Seasons}
	if in.PickWhy != "" {
		n.pickWhy = in.PickWhy
		return n
	}
	history, err := pickvalue.History()
	if err != nil {
		n.pickWhy = err.Error()
		return n
	}
	history = append(history, in.History...)
	players := make([]pickvalue.Player, 0, len(in.Records))
	for _, r := range in.Records {
		v, ok := in.Values[r.MFL]
		players = append(players, pickvalue.Player{Draft: r.Draft, Value: v, Valued: ok})
	}
	m, err := pickvalue.Fit(history, players, in.Season, in.Discount)
	if err != nil {
		n.pickWhy = err.Error()
		return n
	}
	n.picks = &m
	return n
}

// Player values a player by his dynasty value in the latest model run.
func (n Native) Player(pid string, _ time.Time) Dynasty {
	v, ok := n.values[pid]
	if !ok {
		return Dynasty{Reason: "Not in this season's model run"}
	}
	return Dynasty{Valued: true, Discounted: v, Seasons: n.seasons, ClassLabel: "Native value"}
}

// Pick values a pick by its round with Pick Value v2; the slot is not known until the order is.
func (n Native) Pick(year, round int, _ string, _ time.Time) Dynasty {
	if n.picks == nil {
		return Dynasty{Reason: pickvalue.Version + " not available: " + n.pickWhy}
	}
	if year < n.season {
		return Dynasty{Reason: fmt.Sprintf("The %d draft is past", year)}
	}
	v, err := n.picks.Pick(year, round, 0, max(n.teams, 1))
	if err != nil {
		return Dynasty{Reason: err.Error()}
	}
	label := fmt.Sprintf("%s, round %d, slot not yet known", pickvalue.Version, round)
	if v.YearsOut > 0 {
		label += fmt.Sprintf(", %d %s out", v.YearsOut, map[bool]string{true: "year", false: "years"}[v.YearsOut == 1])
	}
	return Dynasty{Valued: true, Discounted: v.PPG, Seasons: n.seasons, ClassLabel: label}
}

// NativeSource is what loading the native model reads: the season's latest model run and its
// scores, every player record, the params and the league's size.
type NativeSource struct {
	Ready     func() error // startup came up: the stores below are open
	LatestRun func(context.Context) (history.Run, bool, error)
	Scores    func(context.Context, int64) ([]history.ModelScore, error)
	Records   func(context.Context) ([]playerrecord.Record, error)
	Params    func() params.Set
	Teams     func() int
	Bodies    HeldDraftBodies
	League    string
}

// LoadNative reads the native model's inputs and builds it; ok is false before startup came up or
// before the first model run.
func LoadNative(ctx context.Context, src NativeSource) (Native, bool, error) {
	if err := src.Ready(); err != nil {
		return Native{}, false, nil //nolint:nilerr // Not ready is no failure here: startup reports its own error.
	}
	run, ok, err := src.LatestRun(ctx)
	if err != nil || !ok {
		return Native{}, false, wrapNative("latest model run", err)
	}
	scores, err := src.Scores(ctx, run.ID)
	if err != nil {
		return Native{}, false, wrapNative("model scores", err)
	}
	records, err := src.Records(ctx)
	if err != nil {
		return Native{}, false, wrapNative("player records", err)
	}
	set := src.Params()
	seasons, err := set.GetGlobal(params.KeyDynastySeasons)
	if err != nil {
		return Native{}, false, wrapNative("params", err)
	}
	discount, err := set.GetGlobal(params.KeyDynastyDiscount)
	if err != nil {
		return Native{}, false, wrapNative("params", err)
	}
	values := make(map[string]float64, len(scores))
	for _, sc := range scores {
		values[sc.MFLID] = sc.DynastyPPG
	}
	history, historyErr := HeldPickHistory(ctx, src.Bodies, src.League, run.Season, records)
	why := ""
	if historyErr != nil {
		why = historyErr.Error()
	}
	return NewNative(NativeInput{History: history, PickWhy: why, Values: values, Records: records, Season: run.Season, Seasons: int(math.Round(seasons)),
		Discount: discount, Teams: src.Teams()}), true, nil
}

func wrapNative(step string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("native value model: %s: %w", step, err)
}

// PublishNative publishes a loaded native model, and logs why one could not be loaded.
func PublishNative(native Native, ok bool, err error, publish func(Model), logLine func(string)) {
	if err != nil {
		logLine("the war room: " + err.Error())
	}
	if ok {
		publish(native)
	}
}
