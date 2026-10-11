package main

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

// nflGame is one NFL game of the chosen week, away team first.
type nflGame struct {
	away, home string
	scores     [2]int // away, home
	kickoff    time.Time
	started    bool
	final      bool
}

// nflGames maps each NFL team playing in the held week to its game, graded at the supplied instant.
func nflGames(weeks []nflschedule.RawWeek, at time.Time) map[string]nflGame {
	out := map[string]nflGame{}
	if len(weeks) != 1 {
		return out
	}
	for _, m := range weeks[0].Matchups {
		home := 1
		if m.Home[0] {
			home = 0
		}
		secs, _ := strconv.ParseInt(m.Kickoff, 10, 64)
		g := nflGame{away: m.Teams[1-home], home: m.Teams[home], kickoff: time.Unix(secs, 0).UTC()}
		g.started = !at.Before(g.kickoff) && m.Scores[0] != "" && m.Scores[1] != ""
		g.final = g.started && m.GameSecondsRemaining == "0"
		g.scores[0], _ = strconv.Atoi(m.Scores[1-home])
		g.scores[1], _ = strconv.Atoi(m.Scores[home])
		out[g.away], out[g.home] = g, g
	}
	return out
}

func (g nflGame) label() string { return g.away + " at " + g.home }

func (g nflGame) result() string {
	switch {
	case !g.started:
		return "Not played yet"
	case !g.final:
		return fmt.Sprintf("In progress: %s %d, %s %d", g.away, g.scores[0], g.home, g.scores[1])
	case g.scores[0] == g.scores[1]:
		return fmt.Sprintf("Tied %d–%d", g.scores[0], g.scores[1])
	case g.scores[0] > g.scores[1]:
		return fmt.Sprintf("%s won %d–%d", g.away, g.scores[0], g.scores[1])
	}
	return fmt.Sprintf("%s won %d–%d", g.home, g.scores[1], g.scores[0])
}

func (v nowView) nflReport(id, title, how string, columns ...string) Report {
	r := reportShell(id, title, how, v.schedule, v.in.zone, columns...)
	r.Source = "Source: MFL NFL schedule"
	r.Summary = append(r.Summary, ReportRow{Cells: []string{"Week", weekLabel(v.shown)}})
	if r.Status == "failed" {
		r.Note = "The NFL schedule for this week is not held yet; refresh from MFL."
	}
	return r
}

func (v nowView) team(p leaguefeed.WeekPlayer) string {
	if f, ok := v.in.lookup.Facts(p.ID.String()); ok {
		return f.Team
	}
	return ""
}

// byeTeams lists the NFL teams on bye in the chosen week, by the held bye weeks.
func (v nowView) byeTeams() []string {
	out := []string{}
	for _, b := range v.now.held.byes.value {
		if b.ByeWeek() == v.week {
			out = append(out, b.Team)
		}
	}
	slices.Sort(out)
	return out
}

// nflGameText is where a player's NFL team plays this week, from his side of the game.
func (v nowView) nflGameText(team string) string {
	g, ok := v.games[team]
	switch {
	case len(v.schedule.value) == 0:
		return "NFL schedule not fetched yet"
	case team == "":
		return "No NFL team"
	case !ok && slices.Contains(v.byeTeams(), team):
		return "Bye"
	case !ok:
		return "No game"
	case team == g.home:
		return "vs " + g.away + " · " + g.result()
	}
	return "at " + g.home + " · " + g.result()
}

func (v nowView) myPlayers() []leaguefeed.WeekPlayer {
	me, _, _, ok := v.myGame()
	if !ok {
		for _, s := range v.shown.Others {
			if s.Franchise == v.in.mine {
				return s.Players
			}
		}
		return nil
	}
	return me.Players
}

func (v nowView) nflWeek() Report {
	r := v.nflReport("M-015", "NFL schedule",
		"The week's NFL games in kickoff order, with the teams on bye. My players are those on my roster for the "+
			"week, by their NFL team in MFL's player directory today.", "Kickoff", "Game", "Result", "My players")
	if r.Status == "failed" {
		return r
	}
	if byes := v.byeTeams(); len(byes) > 0 {
		r.Summary = append(r.Summary, ReportRow{Cells: []string{"On bye", strings.Join(byes, ", ")}})
	} else if v.now.held.byes.fetchedAt.IsZero() {
		r.Summary = append(r.Summary, ReportRow{Cells: []string{"On bye", "Bye weeks not fetched yet"}})
	} else {
		r.Summary = append(r.Summary, ReportRow{Cells: []string{"On bye", "No team"}})
	}
	mine := map[string][]string{}
	for _, p := range v.myPlayers() {
		name, _ := v.player(p.ID)
		mine[v.team(p)] = append(mine[v.team(p)], name)
	}
	if v.in.mine != "" && len(mine) == 0 {
		r.Note = strings.TrimSpace(r.Note + " MFL lists no roster for my franchise this week yet.")
	}
	byeNames := []string{}
	for _, team := range v.byeTeams() {
		byeNames = append(byeNames, mine[team]...)
	}
	if len(byeNames) > 0 {
		r.Summary = append(r.Summary, ReportRow{Cells: []string{"My players on bye", strings.Join(byeNames, "; ")}})
	}
	games := []nflGame{}
	for team, g := range v.games {
		if team == g.home {
			games = append(games, g)
		}
	}
	slices.SortFunc(games, func(a, b nflGame) int {
		return cmp.Or(a.kickoff.Compare(b.kickoff), strings.Compare(a.label(), b.label()))
	})
	for _, g := range games {
		players := append(slices.Clone(mine[g.away]), mine[g.home]...)
		cell := strings.Join(players, "\n")
		if cell == "" {
			cell = "—"
		}
		row := v.row("", reportTime(g.kickoff, v.in.zone), g.label(), g.result(), cell)
		row.Mine = len(players) > 0
		r.Rows = append(r.Rows, row)
	}
	return r
}

func roleText(p leaguefeed.WeekPlayer) string {
	if p.Starter {
		return "Started"
	}
	return "Bench"
}

func (v nowView) nflBoxScore() Report {
	r := v.nflReport("M-016", "NFL box score",
		"My players and my opponent's, grouped by the NFL game they played in, with each one's fantasy points. "+
			"A player's NFL team is his team in MFL's player directory today. Games in kickoff order.",
		"Player", "Franchise", "Role", "Points")
	v.directoryNote(&r)
	me, them, ok := v.needMine(&r)
	if !ok || r.Status == "failed" {
		return r
	}
	type line struct {
		side leaguefeed.Side
		p    leaguefeed.WeekPlayer
	}
	byTeam := map[string][]line{}
	for _, s := range []leaguefeed.Side{me, them} {
		for _, p := range s.Players {
			byTeam[v.team(p)] = append(byTeam[v.team(p)], line{side: s, p: p})
		}
	}
	type group struct {
		heading string
		order   time.Time
		lines   []line
	}
	groups := map[string]*group{}
	for team, lines := range byTeam {
		heading, order := "Not playing this week", time.Unix(1<<40, 0)
		if g, found := v.games[team]; found {
			heading, order = g.label()+" · "+g.result(), g.kickoff
		}
		if groups[heading] == nil {
			groups[heading] = &group{heading: heading, order: order}
		}
		groups[heading].lines = append(groups[heading].lines, lines...)
	}
	sorted := slices.Collect(func(yield func(*group) bool) {
		for _, g := range groups {
			if !yield(g) {
				return
			}
		}
	})
	slices.SortFunc(sorted, func(a, b *group) int {
		return cmp.Or(a.order.Compare(b.order), strings.Compare(a.heading, b.heading))
	})
	for _, g := range sorted {
		slices.SortStableFunc(g.lines, func(a, b line) int {
			return cmp.Or(cmp.Compare(b.p.Points, a.p.Points), strings.Compare(a.p.ID.String(), b.p.ID.String()))
		})
		for _, l := range g.lines {
			name, _ := v.player(l.p.ID)
			row := v.row(l.side.Franchise, name, v.name(l.side.Franchise), roleText(l.p), v.pointsText(l.side, l.p.Points))
			row.Group = g.heading
			r.Rows = append(r.Rows, row)
		}
	}
	if len(r.Rows) == 0 {
		r.Status, r.Note = "empty", strings.TrimSpace(r.Note+" MFL lists no rosters for this game yet.")
	}
	return r
}

func (v nowView) boxScore() Report {
	r := v.shownReport("M-072", "Fantasy box score",
		"My game, starter by starter: each one's NFL game and fantasy points, and whether MFL's best lineup (the "+
			"most the same roster could have scored under the league's starter rules) started him; then any bench "+
			"player it would have started. MFL's stat-by-stat scoring lines are not held, so each line is the "+
			"player's points.", "Slot", "Player", "NFL game", "Points", "Lineup call")
	v.directoryNote(&r)
	me, them, ok := v.needMine(&r)
	if !ok {
		return r
	}
	for _, s := range []leaguefeed.Side{me, them} {
		group := v.name(s.Franchise) + " · " + v.pointsText(s, s.Points)
		if s.Scored && s.Best > 0 {
			group += " · best lineup " + scoreText(s.Best)
		}
		for _, p := range v.starters(s) {
			r.Rows = append(r.Rows, v.boxRow(s, p, group))
		}
		for _, p := range s.Players {
			if !p.Starter && p.ShouldStart && s.Scored {
				r.Rows = append(r.Rows, v.boxRow(s, p, group))
			}
		}
	}
	if len(r.Rows) == 0 {
		r.Status, r.Note = "empty", strings.TrimSpace(r.Note+" Neither lineup is set yet.")
	}
	return r
}

func (v nowView) boxRow(s leaguefeed.Side, p leaguefeed.WeekPlayer, group string) ReportRow {
	name, pos := v.player(p.ID)
	call := "Not scored yet"
	switch {
	case !s.Scored:
	case !p.Starter:
		call = "Benched; the best lineup started him"
	case p.ShouldStart:
		call = "In the best lineup"
	default:
		call = "The best lineup started someone else"
	}
	row := v.row(s.Franchise, pos, name, v.nflGameText(v.team(p)), v.pointsText(s, p.Points), call)
	row.Group = group
	return row
}

func (v nowView) previews() Report {
	r := reportShell("M-018", "Fantasy previews",
		"Each game's projected totals: the sum of MFL's projections for the starters each side has set. A "+
			"starter MFL does not project counts zero. My game first.",
		v.now.projections, v.in.zone, "Away", "Projected", "Home", "Projected", "Favored")
	r.Summary = append(r.Summary, ReportRow{Cells: []string{"Week", weekLabel(v.shown)}})
	if r.Status == "failed" {
		r.Note = "MFL projections are not held yet; refresh from MFL."
		return r
	}
	feeds := v.now.projections.value
	if len(feeds) != 1 || feeds[0].Week != v.week {
		r.Status, r.Note = "empty", fmt.Sprintf("MFL projections are held for week %d only; choose that week.", v.now.current)
		return r
	}
	projected := map[string]leaguefeed.Points{}
	for _, p := range feeds[0].Players {
		projected[p.ID.String()] = hundredths(p.Value)
	}
	missing := 0
	total := func(s leaguefeed.Side) (leaguefeed.Points, bool) {
		var sum leaguefeed.Points
		starters := v.starters(s)
		for _, p := range starters {
			pts, ok := projected[p.ID.String()]
			if !ok {
				missing++
			}
			sum += pts
		}
		return sum, len(starters) > 0
	}
	for _, g := range v.mineFirst(v.shown.Games) {
		away, awaySet := total(g.Away)
		home, homeSet := total(g.Home)
		row := v.row("", v.name(g.Away.Franchise), projText(away, awaySet), v.name(g.Home.Franchise),
			projText(home, homeSet), v.favored(g, away, home, awaySet && homeSet))
		row.Mine = v.isMine(g)
		r.Rows = append(r.Rows, row)
	}
	r.Summary = append(r.Summary, ReportRow{Cells: []string{"Starters MFL does not project", strconv.Itoa(missing)}})
	return r
}

func projText(p leaguefeed.Points, set bool) string {
	if !set {
		return "No lineup set"
	}
	return scoreText(p)
}

func (v nowView) favored(g leaguefeed.Game, away, home leaguefeed.Points, set bool) string {
	switch {
	case !set:
		return "—"
	case away == home:
		return "Even"
	case away > home:
		return v.name(g.Away.Franchise) + " by " + scoreText(away-home)
	}
	return v.name(g.Home.Franchise) + " by " + scoreText(home-away)
}
