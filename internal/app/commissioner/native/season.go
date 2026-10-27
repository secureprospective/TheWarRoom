package native

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// WeekAt is the week in play: the first held week not yet finished, or the last held week once every one
// is. With no week held the app cannot say.
func WeekAt(weeks []leagueweek.Week, at time.Time) nativecheck.BidSeason {
	played := make([]leagueweek.Week, 0, len(weeks))
	for _, w := range weeks {
		if len(w.Games) != 0 {
			played = append(played, w)
		}
	}
	if len(played) == 0 {
		return nativecheck.BidSeason{}
	}
	first := leagueweek.FirstLock(played[0])
	for _, w := range played[1:] {
		if kickoff := leagueweek.FirstLock(w); kickoff.Before(first) {
			first = kickoff
		}
	}
	if at.Before(first) {
		return nativecheck.BidSeason{Offseason: true}
	}
	if current, found := leagueweek.LineupWeek(at, played...); found {
		return nativecheck.BidSeason{Week: current.Number, WeekKnown: true}
	}
	last := 0
	for _, w := range played {
		last = max(last, w.Number)
	}
	return nativecheck.BidSeason{Week: last, WeekKnown: true}
}

// PlayoffField treats absence as unknown until the native reading has a complete field.
func PlayoffField(bracket commissioner.NativeBracket, directory commissioner.NativeDirectory) map[string]nativecheck.PlayoffStanding {
	field := map[string]nativecheck.PlayoffStanding{}
	if len(bracket.Fields) == 0 {
		return field
	}
	for _, franchise := range directory.Franchises {
		field[franchise.Franchise] = nativecheck.PlayoffOut
	}
	for _, conference := range bracket.Fields {
		for _, seed := range conference.Seeds {
			field[seed.Franchise] = nativecheck.PlayoffIn
		}
	}
	return field
}

// ScheduleDates reads local reference bodies only; its caller bounds the resource lifetime.
func ScheduleDates(ctx context.Context, bodies func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error), season int, at time.Time) ([]leagueweek.Week, error) {
	var weeks []leagueweek.Week
	seen := make(map[int]bool)
	_, err := bodies(ctx, "TYPE="+nflschedule.Export,
		func(src string, body []byte, _ time.Time) bool {
			if !ingestion.SameLeaguelessExport(src, nflschedule.Export, season) {
				return false
			}
			raw, err := nflschedule.Parse(body)
			if err != nil {
				return false
			}
			week, err := nflschedule.ToWeek(raw, at)
			if err == nil && !seen[week.Number] {
				weeks = append(weeks, week)
				seen[week.Number] = true
			}
			return false
		})
	if err != nil {
		return nil, fmt.Errorf("read held NFL week dates: %w", err)
	}
	return weeks, nil
}

// HeldLineupFacts uses only the selected week's held locks; later duplicates retain precedence.
func HeldLineupFacts(weeks []leagueweek.Week, week int, facts nativecheck.Facts, zone *time.Location) commissioner.LineupFacts {
	out := commissioner.LineupFacts{Facts: facts, Zone: zone}
	for _, w := range weeks {
		if w.Number == week {
			out.Locks, out.ScheduleHeld = leagueweek.Locks(w), true
		}
	}
	return out
}

type BidSource struct {
	Records  func(context.Context, string) ([]commissioner.Record, error)
	Schedule func(context.Context, int, time.Time) ([]leagueweek.Week, error)
	Bracket  func(context.Context, string, int) (commissioner.NativeBracket, error)
}

func ReadBidEnvironment(ctx context.Context, source BidSource, league string, at time.Time) (market.BidEnvironment, error) {
	records, err := source.Records(ctx, league)
	if err != nil {
		return market.BidEnvironment{}, err
	}
	season, err := commissioner.SeasonAt(records, commissioner.ReadingInstant(records, at))
	if err != nil {
		return market.BidEnvironment{}, fmt.Errorf("read the league season: %w", err)
	}
	weeks, err := source.Schedule(ctx, season, at)
	if err != nil {
		return market.BidEnvironment{}, err
	}
	phase := WeekAt(weeks, at)
	if !phase.WeekKnown && !phase.Offseason {
		phase.Offseason, err = market.BidOffseasonAt(records, at, season)
		if err != nil {
			return market.BidEnvironment{}, fmt.Errorf("read bidding offseason: %w", err)
		}
	}
	field := map[string]nativecheck.PlayoffStanding{}
	if phase.WeekKnown {
		bracket, err := source.Bracket(ctx, league, season)
		if err != nil {
			return market.BidEnvironment{}, fmt.Errorf("read bidding playoff field: %w", err)
		}
		directory, err := commissioner.NativeDirectoryReading(league, records, at)
		if err != nil {
			return market.BidEnvironment{}, fmt.Errorf("read bidding franchises: %w", err)
		}
		field = PlayoffField(bracket, directory)
	}
	return market.BidEnvironment{Season: phase, Playoff: field}, nil
}

func OwnerZone(zone *time.Location) *time.Location {
	if zone != nil {
		return zone
	}
	return time.Local
}
