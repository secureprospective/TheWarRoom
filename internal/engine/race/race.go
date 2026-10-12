// Package race is the league race's arithmetic as pure functions: standings from MFL's weekly
// results, all-play, streaks and games back, the league's tiebreak order applied step by step,
// playoff seeding, the first round and the bubble.
package race

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

// Franchise places a franchise in its division and conference ("" when the league has none).
type Franchise struct {
	ID, Division, Conference string
}

// Record is wins, losses and ties.
type Record struct{ W, L, T int }

func (r Record) Games() int { return r.W + r.L + r.T }

func (r *Record) add(res leaguefeed.Result) {
	switch res {
	case "W":
		r.W++
	case "L":
		r.L++
	case "T":
		r.T++
	}
}

// comparePct orders by win percentage, a tie counting half; no games reads as .000.
func comparePct(a, b Record) int {
	an, ad := int64(2*a.W+a.T), int64(2*max(a.Games(), 1))
	bn, bd := int64(2*b.W+b.T), int64(2*max(b.Games(), 1))
	switch l, r := an*bd, bn*ad; {
	case l > r:
		return 1
	case l < r:
		return -1
	}
	return 0
}

// Streak is the run of identical results a franchise's regular season ends on.
type Streak struct {
	Result leaguefeed.Result
	Count  int
}

// Line is one franchise's season. Records count regular-season games by MFL's result; All-play
// and For count every week MFL scored, playoffs included, as MFL's own standings do.
type Line struct {
	Franchise
	Overall, InDivision, InConference Record
	AllPlay                           Record
	For, Against                      leaguefeed.Points
	Streak                            Streak
	Power                             int64 // MFL's power rating in hundredths; HasPower says it is held
	HasPower                          bool
	Remaining                         []string // regular-season opponents still to play, in week order
}

// Table is a season's standings and the head-to-head results the tiebreaks read.
type Table struct {
	lines map[string]*Line
	ids   []string
	h2h   map[[2]string]Record
	keys  []Key
	// RegularWeeks is how many regular-season weeks have every game scored.
	RegularWeeks int
}

// Build counts the season. A franchise in the results that the league does not list is an error,
// so a stale league can never silently drop a team from the race.
func Build(franchises []Franchise, weeks []leaguefeed.WeekResult, keys []Key, power map[string]int64) (Table, error) {
	t := Table{lines: map[string]*Line{}, h2h: map[[2]string]Record{}, keys: keys}
	for _, f := range franchises {
		t.lines[f.ID] = &Line{Franchise: f, Remaining: []string{}}
		t.ids = append(t.ids, f.ID)
		if p, ok := power[f.ID]; ok {
			t.lines[f.ID].Power, t.lines[f.ID].HasPower = p, true
		}
	}
	slices.Sort(t.ids)
	ordered := slices.Clone(weeks)
	slices.SortFunc(ordered, func(a, b leaguefeed.WeekResult) int { return a.Week - b.Week })
	for _, w := range ordered {
		if err := t.addWeek(w); err != nil {
			return Table{}, err
		}
	}
	return t, nil
}

func (t *Table) addWeek(w leaguefeed.WeekResult) error {
	scored := []leaguefeed.Side{}
	complete, regular := true, false
	for _, g := range w.Games {
		for _, s := range []leaguefeed.Side{g.Home, g.Away} {
			if t.lines[s.Franchise] == nil {
				return fmt.Errorf("race: week %d lists franchise %s, which the league does not", w.Week, s.Franchise)
			}
		}
		regular = regular || g.Regular
		if g.Regular {
			complete = complete && g.Home.Scored
			t.addGame(g)
		}
		if g.Home.Scored {
			scored = append(scored, g.Home, g.Away)
		}
	}
	for _, s := range w.Others {
		if t.lines[s.Franchise] == nil {
			return fmt.Errorf("race: week %d lists franchise %s, which the league does not", w.Week, s.Franchise)
		}
		if s.Scored {
			scored = append(scored, s)
		}
	}
	if regular && complete {
		t.RegularWeeks++
	}
	t.addScores(scored)
	return nil
}

// addScores counts a week's points for and its all-play: each score against every other.
func (t *Table) addScores(scored []leaguefeed.Side) {
	for _, s := range scored {
		l := t.lines[s.Franchise]
		l.For += s.Points
		for _, o := range scored {
			switch {
			case o.Franchise == s.Franchise:
			case s.Points > o.Points:
				l.AllPlay.W++
			case s.Points < o.Points:
				l.AllPlay.L++
			default:
				l.AllPlay.T++
			}
		}
	}
}

func (t *Table) addGame(g leaguefeed.Game) {
	home, away := t.lines[g.Home.Franchise], t.lines[g.Away.Franchise]
	if !g.Home.Scored {
		home.Remaining = append(home.Remaining, away.ID)
		away.Remaining = append(away.Remaining, home.ID)
		return
	}
	for _, side := range []struct {
		me, them *Line
		res      leaguefeed.Result
		against  leaguefeed.Points
	}{{home, away, g.Home.Result, g.Away.Points}, {away, home, g.Away.Result, g.Home.Points}} {
		l := side.me
		l.Overall.add(side.res)
		if l.Division != "" && l.Division == side.them.Division {
			l.InDivision.add(side.res)
		}
		if l.Conference != "" && l.Conference == side.them.Conference {
			l.InConference.add(side.res)
		}
		l.Against += side.against
		if l.Streak.Result == side.res {
			l.Streak.Count++
		} else {
			l.Streak = Streak{Result: side.res, Count: 1}
		}
		pair := t.h2h[[2]string{l.ID, side.them.ID}]
		pair.add(side.res)
		t.h2h[[2]string{l.ID, side.them.ID}] = pair
	}
}

// Line is one franchise's season; ok is false for a franchise the league does not list.
func (t Table) Line(id string) (Line, bool) {
	l, ok := t.lines[id]
	if !ok {
		return Line{}, false
	}
	return *l, true
}

// IDs lists every franchise in MFL's franchise order.
func (t Table) IDs() []string { return slices.Clone(t.ids) }

// HeadToHead is a's regular-season record against b.
func (t Table) HeadToHead(a, b string) Record { return t.h2h[[2]string{a, b}] }

// HalfGamesBack is how far r trails lead, in half games: a win and a loss are one game apart.
func HalfGamesBack(lead, r Record) int {
	return (lead.W - r.W) + (r.L - lead.L)
}
