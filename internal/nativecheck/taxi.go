package nativecheck

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// gamesPerSeason is the length of an NFL regular season. A league that measures taxi experience by
// career games reads every full season of games as one year.
const gamesPerSeason = 17

// experience is how long a player has been in the NFL by the league's own measure, and how to say it.
type experience struct {
	years  int
	words  string
	limit  string
	known  bool
	reason string
}

// taxiExperience measures a player by the rule's basis: calendar years counts the season he was drafted
// as his first; career games counts each season's worth of games played, rounding up.
func taxiExperience(rules leaguerules.TaxiRules, f PlayerFacts, season int, exceptions ...DecisionException) experience {
	if why, allowed := ExceptionReason(exceptions, "taxi.maximumYears"); allowed {
		return experience{known: true, words: why}
	}
	if rules.Basis.State != leaguerules.IsKnown || rules.MaximumYears.State != leaguerules.IsKnown {
		return experience{reason: "Needs a ruling: the taxi experience limit"}
	}
	e := experience{limit: fmt.Sprintf("the limit is %d", rules.MaximumYears.Value)}
	switch rules.Basis.Value {
	case leaguerules.CalendarYears:
		if !f.DraftYearHeld || season < 1 || f.DraftYear < 1 || f.DraftYear > season {
			e.reason = "his draft year is not held"
			return e
		}
		e.years = season - f.DraftYear + 1
		e.words = fmt.Sprintf("in his %s NFL season", ordinal(e.years))
	case leaguerules.CareerGames:
		if !f.CareerGamesHeld || f.CareerGames < 0 {
			e.reason = "his career games played are not held"
			return e
		}
		e.years = (f.CareerGames + gamesPerSeason - 1) / gamesPerSeason
		e.words = fmt.Sprintf("with %d career games, which is %d seasons of %d games", f.CareerGames, e.years, gamesPerSeason)
	}
	e.known = true
	return e
}

func taxiStanding(rules leaguerules.TaxiRules, season int, id string, f PlayerFacts, exceptions ...DecisionException) ReserveVerdict {
	v := ReserveVerdict{Player: id, Name: f.Name, Squad: "Taxi"}
	e := taxiExperience(rules, f, season, exceptions...)
	switch {
	case !e.known:
		v.Standing, v.Words = Unclear, "Can't tell: "+e.reason
	case e.years > rules.MaximumYears.Value:
		v.Standing, v.Words = MustLeave, fmt.Sprintf("Must come off the taxi squad: %s; %s", e.words, e.limit)
	default:
		v.Standing, v.Words = Stays, fmt.Sprintf("May stay on the taxi squad: %s; %s", e.words, e.limit)
	}
	return v
}

// CanEnterTaxi says whether a player may be placed on the taxi squad now, with used spots already taken.
func CanEnterTaxi(rules leaguerules.TaxiRules, f PlayerFacts, season, used int, exceptions ...DecisionException) Entry {
	e := taxiExperience(rules, f, season, exceptions...)
	switch {
	case !e.known:
		return Entry{Words: "Can't tell: " + e.reason}
	case e.years > rules.MaximumYears.Value:
		return Entry{Known: true, Words: fmt.Sprintf("Cannot go on the taxi squad: %s; %s", e.words, e.limit)}
	case rules.Slots.State != leaguerules.IsKnown:
		return Entry{Words: "Needs a ruling: taxi spots"}
	case used >= rules.Slots.Value:
		return Entry{Known: true, Words: fmt.Sprintf("Cannot go on the taxi squad: all %d taxi spots are taken", rules.Slots.Value)}
	}
	return Entry{Allowed: true, Known: true, Words: fmt.Sprintf("May go on the taxi squad: %s; %s", e.words, e.limit)}
}
