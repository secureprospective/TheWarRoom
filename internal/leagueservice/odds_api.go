package leagueservice

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/appvalue"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// oddsHost reads the odds from the league's own records once its values are in hand.
type oddsHost interface {
	NativeOdds(ctx context.Context, league string, season int, values commissioner.OddsValues) (commissioner.NativeOdds, error)
}

// TargetNativeOdds reads win chance, power and playoff odds for any app league from its own results, schedule,
// rosters and values, under its own rules. Values that cannot be read leave those numbers empty with the reason;
// the standings still read.
func (a *Service) TargetNativeOdds(league string, season int) (commissioner.NativeOdds, error) {
	if err := (RawNativeResultsScope{League: league, Season: season, Week: 1}).Validate(); err != nil {
		return commissioner.NativeOdds{}, err
	}
	if err := a.admitRead(league, "odds.read", ""); err != nil {
		return commissioner.NativeOdds{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeOdds{}, fmt.Errorf("native odds startup: %w", err)
	}
	host, ok := a.host.(oddsHost)
	if !ok {
		return commissioner.NativeOdds{}, fmt.Errorf("native odds: league results are not available on this host")
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	reading, err := host.NativeOdds(ctx, league, season, a.oddsValues(ctx, league, season))
	if err != nil {
		return commissioner.NativeOdds{}, fmt.Errorf("native odds: %w", err)
	}
	return reading, nil
}

// oddsValues is the league's value read, every franchise's players, in the form the odds reading counts.
func (a *Service) oddsValues(ctx context.Context, league string, season int) commissioner.OddsValues {
	host, ok := a.host.(valueHost)
	if !ok {
		return commissioner.OddsValues{Words: "Player values are not available on this host"}
	}
	reading, err := host.NativeValues(ctx, league, "", season)
	return OddsValuesOf(reading, err)
}

// OddsValuesOf turns a value read into the odds reading's input; a read still preparing or failed gives words.
func OddsValuesOf(reading appvalue.Reading, err error) commissioner.OddsValues {
	switch {
	case err != nil:
		return commissioner.OddsValues{Words: "Player values could not be read: " + err.Error()}
	case reading.Preparing:
		return commissioner.OddsValues{Words: "Player values are still being prepared"}
	}
	out := commissioner.OddsValues{Players: make([]commissioner.OddsPlayer, 0, len(reading.Players))}
	for _, p := range reading.Players {
		out.Players = append(out.Players, commissioner.OddsPlayer{Franchise: p.Franchise, Position: p.Position,
			NowPPG: p.NowPPG, Age: p.Age, AgeKnown: p.AgeKnown, Available: p.Available})
	}
	return out
}
