package market

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/pickvalue"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
)

// Native is the native value model as the asset board reads it: each player's dynasty value from
// the latest model run, and Pick Value v1 for picks, both in dynasty points per game.
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
}

// NewNative builds the native model. Pick Value v1 is fitted on the records' draft capital and the
// run's values; when it cannot be (no draft history or no recent class) players are still valued
// and picks say why.
func NewNative(in NativeInput) Native {
	n := Native{values: in.Values, season: in.Season, teams: in.Teams, seasons: in.Seasons}
	history, err := pickvalue.History()
	if err != nil {
		n.pickWhy = err.Error()
		return n
	}
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

// Pick values a pick by its round with Pick Value v1; the slot is not known until the order is.
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
