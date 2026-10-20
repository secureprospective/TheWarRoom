package now

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

type weekStarter struct {
	franchise string
	player    leaguefeed.WeekPlayer
}

// weekStarters is every starter of the week, most points first; ties keep name order.
func (v nowView) weekStarters() []weekStarter {
	out := []weekStarter{}
	for _, s := range allSides(v.shown) {
		for _, p := range s.Players {
			if p.Starter {
				out = append(out, weekStarter{franchise: s.Franchise, player: p})
			}
		}
	}
	slices.SortStableFunc(out, func(a, b weekStarter) int {
		an, _ := v.player(a.player.ID)
		bn, _ := v.player(b.player.ID)
		return cmp.Or(cmp.Compare(b.player.Points, a.player.Points), strings.Compare(an, bn))
	})
	return out
}

func (v nowView) sideText(s leaguefeed.Side) string {
	return v.name(s.Franchise) + " · " + scoreText(s.Points)
}

func (v nowView) beatText(g leaguefeed.Game) string {
	win, lose, tied := winnerFirst(g)
	if tied {
		return v.name(win.Franchise) + " and " + v.name(lose.Franchise) + " tied at " + scoreText(win.Points)
	}
	return v.name(win.Franchise) + " beat " + v.name(lose.Franchise) + " by " + scoreText(win.Points-lose.Points)
}

func (v nowView) weekSummary() report.Report {
	r, ok := v.scoredReport("M-007", "Weekly summary",
		"The week at a glance from MFL's scores: the high and low score, the league average, the closest game, "+
			"the biggest win and the most points left on the bench (the best lineup MFL found from the same "+
			"roster, less the points scored). Then the week's ten best starters.",
		"Player", "Position", "Franchise", "Points")
	if !ok {
		return r
	}
	v.directoryNote(&r)
	sides := []leaguefeed.Side{}
	for _, s := range allSides(v.shown) {
		if s.Scored {
			sides = append(sides, s)
		}
	}
	byPoints := slices.Clone(sides)
	slices.SortStableFunc(byPoints, func(a, b leaguefeed.Side) int { return cmp.Compare(b.Points, a.Points) })
	var total leaguefeed.Points
	for _, s := range sides {
		total += s.Points
	}
	r.Summary = append(r.Summary, report.Row{Cells: []string{"High score", v.sideText(byPoints[0])}},
		report.Row{Cells: []string{"Low score", v.sideText(byPoints[len(byPoints)-1])}},
		report.Row{Cells: []string{"League average", scoreText((total + leaguefeed.Points(len(sides)/2)) / leaguefeed.Points(len(sides)))}})
	if len(v.shown.Games) > 0 {
		margin := func(g leaguefeed.Game) leaguefeed.Points {
			win, lose, _ := winnerFirst(g)
			return win.Points - lose.Points
		}
		games := slices.Clone(v.shown.Games)
		slices.SortStableFunc(games, func(a, b leaguefeed.Game) int { return cmp.Compare(margin(a), margin(b)) })
		r.Summary = append(r.Summary, report.Row{Cells: []string{"Closest game", v.beatText(games[0])}},
			report.Row{Cells: []string{"Biggest win", v.beatText(games[len(games)-1])}})
	}
	bench := slices.Clone(sides)
	slices.SortStableFunc(bench, func(a, b leaguefeed.Side) int { return cmp.Compare(b.Best-b.Points, a.Best-a.Points) })
	if left := bench[0].Best - bench[0].Points; left > 0 {
		r.Summary = append(r.Summary, report.Row{Cells: []string{"Most left on the bench",
			v.name(bench[0].Franchise) + " · " + scoreText(left)}})
	}
	for _, s := range v.weekStarters()[:min(10, len(v.weekStarters()))] {
		name, pos := v.player(s.player.ID)
		r.Rows = append(r.Rows, v.row(s.franchise, name, pos, v.name(s.franchise), scoreText(s.player.Points)))
	}
	return r
}

// topStarter is the side's highest-scoring starter; ok is false for a side with none listed.
func (v nowView) topStarter(s leaguefeed.Side) (leaguefeed.WeekPlayer, bool) {
	var top leaguefeed.WeekPlayer
	found := false
	for _, p := range v.starters(s) {
		if !found || p.Points > top.Points {
			top, found = p, true
		}
	}
	return top, found
}

// recap writes a scored game in plain sentences from its numbers alone.
func (v nowView) recap(g leaguefeed.Game) string {
	win, lose, tied := winnerFirst(g)
	parts := []string{}
	if !g.Regular {
		parts = append(parts, "Playoff game.")
	}
	if tied {
		parts = append(parts, fmt.Sprintf("%s and %s tied at %s.", v.name(win.Franchise), v.name(lose.Franchise),
			scoreText(win.Points)))
	} else {
		parts = append(parts, fmt.Sprintf("%s beat %s %s to %s, a margin of %s.", v.name(win.Franchise),
			v.name(lose.Franchise), scoreText(win.Points), scoreText(lose.Points), scoreText(win.Points-lose.Points)))
	}
	for _, s := range []leaguefeed.Side{win, lose} {
		if p, ok := v.topStarter(s); ok {
			name, pos := v.player(p.ID)
			parts = append(parts, fmt.Sprintf("%s top starter was %s (%s) with %s.", Possessive(v.name(s.Franchise)), name, pos,
				scoreText(p.Points)))
		}
	}
	if left := lose.Best - lose.Points; left > 0 && !tied {
		verdict := "would still have lost"
		if lose.Best > win.Points {
			verdict = "would have won"
		}
		parts = append(parts, fmt.Sprintf("%s left %s on the bench; its best lineup, %s, %s.", v.name(lose.Franchise),
			scoreText(left), scoreText(lose.Best), verdict))
	} else if lose.Best > 0 && !tied {
		parts = append(parts, v.name(lose.Franchise)+" started its best lineup.")
	}
	return strings.Join(parts, " ")
}

func (v nowView) recaps() report.Report {
	r, ok := v.scoredReport("M-021", "Recaps",
		"Each game in plain sentences, written only from MFL's numbers: the score and margin, each side's top "+
			"starter, and what the loser's best lineup (MFL's best lineup from the same roster) would have scored. "+
			"My game first, then the highest winning score.", "Game", "Recap")
	if !ok {
		return r
	}
	v.directoryNote(&r)
	for _, g := range v.resultOrder() {
		row := v.row("", v.name(g.Away.Franchise)+" at "+v.name(g.Home.Franchise), v.recap(g))
		row.Mine = v.isMine(g)
		r.Rows = append(r.Rows, row)
	}
	return r
}

func (v nowView) topByPosition() report.Report {
	r, ok := v.scoredReport("M-022", "Top scorers by position",
		"The week's three best starters at each position, in the league's position order. Ties keep name order.",
		"Player", "Franchise", "Points")
	if !ok {
		return r
	}
	v.directoryNote(&r)
	byPos, seen := map[string][]weekStarter{}, map[string]bool{}
	for _, s := range v.weekStarters() {
		_, pos := v.player(s.player.ID)
		byPos[pos], seen[pos] = append(byPos[pos], s), true
	}
	for _, pos := range v.positions(seen) {
		for _, s := range byPos[pos][:min(3, len(byPos[pos]))] {
			name, _ := v.player(s.player.ID)
			row := v.row(s.franchise, name, v.name(s.franchise), scoreText(s.player.Points))
			row.Group = pos
			r.Rows = append(r.Rows, row)
		}
	}
	return r
}

// scoreChange is one player's points for a week, changed since the first held copy.
type scoreChange struct {
	franchise  string
	player     leaguefeed.WeekPlayer
	first, now leaguefeed.Points
}

// weekChanges compares a week's first held scores with its latest: every rostered player whose
// points moved, and every side whose result changed.
func weekChanges(first, latest leaguefeed.WeekResult) (players []scoreChange, flips []leaguefeed.Side) {
	before := map[string]leaguefeed.Side{}
	for _, s := range allSides(first) {
		before[s.Franchise] = s
	}
	for _, s := range allSides(latest) {
		was, ok := before[s.Franchise]
		if !ok {
			continue
		}
		points := map[string]leaguefeed.Points{}
		for _, p := range was.Players {
			points[p.ID.String()] = p.Points
		}
		for _, p := range s.Players {
			if old, held := points[p.ID.String()]; held && old != p.Points {
				players = append(players, scoreChange{franchise: s.Franchise, player: p, first: old, now: p.Points})
			}
		}
		if was.Result != s.Result {
			flips = append(flips, s)
		}
	}
	return players, flips
}

func allSides(w leaguefeed.WeekResult) []leaguefeed.Side {
	sides := slices.Clone(w.Others)
	for _, g := range w.Games {
		sides = append(sides, g.Home, g.Away)
	}
	return sides
}

func (v nowView) statChanges() report.Report {
	r := v.weeksReport("M-088", "Official stat changes",
		"MFL corrects stats after games. For each scored week, the app compares the first copy of MFL's weekly "+
			"results it held with the latest, player by player; a change can flip a result. Weeks MFL scored before "+
			"the app first read it compare from that first read. Newest week first.",
		"Player", "Franchise", "First held", "Now", "Change")
	v.directoryNote(&r)
	if r.Status == "failed" {
		return r
	}
	firsts := v.now.Held.Firsts
	r.Summary = append(r.Summary, report.Row{Cells: []string{"Copies compared", strconv.Itoa(firsts.Copies)}})
	weeks := slices.Clone(v.weeks)
	slices.Reverse(weeks)
	for _, w := range weeks {
		first, ok := firsts.Weeks[w.Week]
		if !ok || !weekScored(w) {
			continue
		}
		players, flips := weekChanges(first.Week, w)
		group := "Week " + strconv.Itoa(w.Week) + " · first held " + reportTime(first.At, v.in.Zone)
		for _, c := range players {
			name, _ := v.player(c.player.ID)
			row := v.row(c.franchise, name, v.name(c.franchise), scoreText(c.first), scoreText(c.now),
				signedText(c.now-c.first))
			row.Group = group
			r.Rows = append(r.Rows, row)
		}
		for _, s := range flips {
			r.Summary = append(r.Summary, report.Row{Cells: []string{"Week " + strconv.Itoa(w.Week) + " result changed",
				v.name(s.Franchise) + " now " + resultWord(s.Result)}})
		}
	}
	if len(firsts.Weeks) == 0 {
		r.Status, r.Note = "empty", strings.TrimSpace(r.Note+
			" No held copy of the weekly results has a scored week yet; changes show once MFL has been read again.")
	} else if len(r.Rows) == 0 {
		r.Status, r.Note = "empty", strings.TrimSpace(r.Note+" No score has changed since the first held copy.")
	}
	return r
}

func resultWord(r leaguefeed.Result) string {
	return map[leaguefeed.Result]string{"W": "won", "L": "lost", "T": "tied"}[r]
}

func signedText(p leaguefeed.Points) string {
	if p > 0 {
		return "+" + scoreText(p)
	}
	return scoreText(p)
}

// Possessive is a franchise name's Possessive: "Buffalo Bills'", "Miami Heat's".
func Possessive(name string) string {
	if strings.HasSuffix(name, "s") {
		return name + "'"
	}
	return name + "'s"
}
