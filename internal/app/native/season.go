package native

import (
	"time"

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
