// Package leagueweek derives lineup weeks, team locks and season phases from supplied facts.
package leagueweek

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

type Game struct {
	Teams   [2]string
	Kickoff time.Time
	Final   bool
}

type Week struct {
	Number int
	Games  []Game
}

type Bounds struct{ Start, LastRegular, End int }

// ErrSeasonOver requires a person to confirm rollover, never a same-season offseason.
var ErrSeasonOver = errors.New("leagueweek: season over")

// LineupWeek is the lowest supplied week with a game still to finish. When every supplied week
// is final it reports false: the caller fetches the next week rather than guessing its number.
func LineupWeek(at time.Time, weeks ...Week) (Week, bool) {
	var chosen Week
	found := false
	for _, w := range weeks {
		if finished(at, w) || (found && w.Number >= chosen.Number) {
			continue
		}
		chosen, found = w, true
	}
	return chosen, found
}

// finished holds when every game is final. A game whose kickoff is still ahead is never final,
// whatever its flag says.
func finished(at time.Time, w Week) bool {
	for _, g := range w.Games {
		if !g.Final || at.Before(g.Kickoff) {
			return false
		}
	}
	return true
}

// Locks maps each NFL team to its game's kickoff (league setting: players lock at kickoff of
// their game). A team on bye has no game and so no entry.
func Locks(w Week) map[string]time.Time {
	locks := make(map[string]time.Time, 2*len(w.Games))
	for _, g := range w.Games {
		for _, team := range g.Teams {
			locks[team] = g.Kickoff
		}
	}
	return locks
}

// FirstLock requires a validated, nonempty week.
func FirstLock(w Week) time.Time {
	first := w.Games[0].Kickoff
	for _, g := range w.Games[1:] {
		if g.Kickoff.Before(first) {
			first = g.Kickoff
		}
	}
	return first
}

// LastLock requires a validated, nonempty week.
func LastLock(w Week) time.Time {
	last := w.Games[0].Kickoff
	for _, g := range w.Games[1:] {
		if g.Kickoff.After(last) {
			last = g.Kickoff
		}
	}
	return last
}

func (b Bounds) validate() error {
	if b.Start < 1 || b.Start > b.LastRegular || b.LastRegular > b.End {
		return fmt.Errorf("leagueweek: invalid bounds %+v", b)
	}
	return nil
}

// Phase is the phase a league week implies. Past the last week it returns ErrSeasonOver:
// rollover expires contracts, so a person confirms it.
func Phase(week int, b Bounds) (domain.Phase, error) {
	if err := b.validate(); err != nil {
		return "", err
	}
	if week < 1 {
		return "", fmt.Errorf("leagueweek: invalid week %d", week)
	}
	switch {
	case week < b.Start:
		return domain.PhaseOffseason, nil
	case week <= b.LastRegular:
		return domain.PhaseRegularSeason, nil
	case week <= b.End:
		return domain.PhasePlayoffs, nil
	default:
		return "", fmt.Errorf("leagueweek: week %d exceeds end %d: %w", week, b.End, ErrSeasonOver)
	}
}

func ParseBounds(start, lastRegular, end string) (Bounds, error) {
	names := [3]string{"startWeek", "lastRegularSeasonWeek", "endWeek"}
	values := [3]int{}
	for i, raw := range []string{start, lastRegular, end} {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return Bounds{}, fmt.Errorf("leagueweek: parse %s %q: %w", names[i], raw, err)
		}
		values[i] = value
	}
	b := Bounds{Start: values[0], LastRegular: values[1], End: values[2]}
	if err := b.validate(); err != nil {
		return Bounds{}, err
	}
	return b, nil
}
