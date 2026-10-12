package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/race"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// RaceBoard is League Pulse › Race: the standings on top, then the reports. Week and Team are the
// choices the lineup and one-team reports follow, as Go accepted them.
type RaceBoard struct {
	Standings Report       `json:"standings"`
	Board     ReportBoard  `json:"board"`
	Week      int          `json:"week"`
	Weeks     []RaceChoice `json:"weeks"`
	Team      string       `json:"team"`
	Teams     []RaceChoice `json:"teams"`
}

type RaceChoice struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type raceInput struct {
	held      raceHeld
	config    league.RawConfig
	lookup    normalize.Lookup
	week      int
	team      string
	mine      string
	zone      *time.Location
	hasLookup bool
}

// raceView is everything the reports read, resolved once: the season's table, names, the
// league's groupings and the accepted choices.
type raceView struct {
	in          raceInput
	table       race.Table
	sortNote    string
	weeks       []leaguefeed.WeekResult
	names       map[string]string
	abbrev      map[string]string
	conferences []league.Conference
	divisions   []league.Division
	week        int
	team        string
}

func buildRace(in raceInput) (RaceBoard, error) {
	v, err := newRaceView(in)
	if err != nil {
		return RaceBoard{}, err
	}
	board := RaceBoard{Standings: v.standings(), Week: v.week, Team: v.team,
		Weeks: v.weekChoices(), Teams: v.teamChoices(),
		Board: ReportBoard{Title: "League race reports", Loading: "Reading the league race…",
			Refresh: "Refresh from MFL", HowLabel: "How this is counted", Reports: []Report{}}}
	board.Board.Reports = append(board.Board.Reports, v.power(ViewSeason), v.power(ViewFranchise), v.seeding(),
		v.bubble(), v.tiebreakers(), v.brackets(), v.allPlay(), v.weeklyPoints(), v.positionPoints(),
		v.playerPoints(), v.scheduleByWeek(), v.scheduleByTeam(), v.scheduleGrid(), v.lineups(), v.lineupDetail())
	return board, nil
}

func newRaceView(in raceInput) (raceView, error) {
	v := raceView{in: in, names: map[string]string{}, abbrev: map[string]string{},
		conferences: in.config.Conferences, divisions: in.config.Divisions}
	v.weeks = slices.Clone(in.held.weeks.value)
	slices.SortFunc(v.weeks, func(a, b leaguefeed.WeekResult) int { return a.Week - b.Week })
	confOf := map[string]string{}
	for _, d := range v.divisions {
		confOf[d.ID] = d.Conference
	}
	teams := []race.Franchise{}
	for _, f := range in.config.Franchises {
		v.names[f.ID], v.abbrev[f.ID] = f.Name, f.Abbrev
		teams = append(teams, race.Franchise{ID: f.ID, Division: f.Division, Conference: confOf[f.Division]})
	}
	for _, id := range []string{in.team, in.mine} {
		if _, ok := v.names[id]; id != "" && !ok {
			return raceView{}, fmt.Errorf("race: franchise %q is not in the league", id)
		}
	}
	v.team = in.team
	keys, err := race.ParseSort(in.config.StandingsSort)
	if err != nil {
		keys, v.sortNote = []race.Key{race.KeyPct}, "MFL's tiebreak order is not held, so ties keep MFL's franchise order; refresh from MFL."
	}
	if v.table, err = race.Build(teams, v.weeks, keys, mflPower(in.held.standings.value)); err != nil {
		return raceView{}, fmt.Errorf("race: %w", err)
	}
	return v, v.chooseWeek(in.week)
}

// chooseWeek accepts a scored week, or 0 for the latest one.
func (v *raceView) chooseWeek(week int) error {
	for _, w := range v.weeks {
		if weekScored(w) && (week == 0 || week == w.Week) {
			v.week = w.Week
		}
	}
	if week != 0 && v.week != week {
		return fmt.Errorf("race: week %d has no scored games", week)
	}
	return nil
}

func weekScored(w leaguefeed.WeekResult) bool {
	return slices.ContainsFunc(w.Games, func(g leaguefeed.Game) bool { return g.Home.Scored }) ||
		slices.ContainsFunc(w.Others, func(s leaguefeed.Side) bool { return s.Scored })
}

// mflPower is MFL's power rating per franchise, in hundredths, for the PWR tiebreak.
func mflPower(standings []leaguestandings.RawStanding) map[string]int64 {
	out := map[string]int64{}
	for _, s := range standings {
		if v, err := strconv.ParseFloat(leaguestandings.SanitizeNumeric(s.Pwr), 64); err == nil && s.Pwr != "" {
			out[s.FranchiseID] = int64(v*100 + 0.5)
		}
	}
	return out
}

func (v raceView) weekChoices() []RaceChoice {
	out := []RaceChoice{}
	for _, w := range v.weeks {
		if weekScored(w) {
			out = append(out, RaceChoice{Value: strconv.Itoa(w.Week), Label: "Week " + strconv.Itoa(w.Week)})
		}
	}
	return out
}

func (v raceView) teamChoices() []RaceChoice {
	out := []RaceChoice{}
	for _, id := range v.table.IDs() {
		out = append(out, RaceChoice{Value: id, Label: v.name(id)})
	}
	slices.SortStableFunc(out, func(a, b RaceChoice) int { return strings.Compare(a.Label, b.Label) })
	return out
}

func (v raceView) name(id string) string {
	switch {
	case v.names[id] != "":
		return v.names[id]
	case v.abbrev[id] != "":
		return v.abbrev[id]
	}
	return "Unnamed franchise"
}

func (v raceView) short(id string) string {
	if v.abbrev[id] != "" {
		return v.abbrev[id]
	}
	return v.name(id)
}

func (v raceView) row(id string, cells ...string) ReportRow {
	return ReportRow{Cells: cells, Mine: id != "" && id == v.in.mine}
}

func (v raceView) weeksReport(id, title, how string, columns ...string) Report {
	return reportShell(id, title, how, v.in.held.weeks, v.in.zone, columns...)
}

func recordText(r race.Record) string {
	if r.T > 0 {
		return fmt.Sprintf("%d-%d-%d", r.W, r.L, r.T)
	}
	return fmt.Sprintf("%d-%d", r.W, r.L)
}

func pctText(r race.Record) string {
	if r.Games() == 0 {
		return "—"
	}
	s := fmt.Sprintf("%.3f", float64(2*r.W+r.T)/float64(2*r.Games()))
	return strings.TrimPrefix(s, "0")
}

func scoreText(p leaguefeed.Points) string {
	sign, n := "", int64(p)
	if n < 0 {
		sign, n = "-", -n
	}
	whole := strconv.FormatInt(n/100, 10)
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	return fmt.Sprintf("%s%s.%02d", sign, whole, n%100)
}

func streakText(s race.Streak) string {
	words := map[leaguefeed.Result]string{"W": "Won", "L": "Lost", "T": "Tied"}
	if s.Count == 0 {
		return "—"
	}
	return fmt.Sprintf("%s %d", words[s.Result], s.Count)
}

func halfGamesText(half int) string {
	if half%2 == 0 {
		return strconv.Itoa(half / 2)
	}
	return fmt.Sprintf("%.1f", float64(half)/2)
}

func (v raceView) divisionName(id string) string {
	for _, d := range v.divisions {
		if d.ID == id {
			return d.Name
		}
	}
	return "No division"
}

func (v raceView) standings() Report {
	r := v.weeksReport("M-002", "Standings",
		"Records count regular-season games by MFL's result; a tie counts half a win in win percentage. "+
			"Points for counts every week MFL scored, playoff weeks included, as MFL's own standings do; points "+
			"against counts regular-season games. Games back trail the division leader: a win and a loss are one "+
			"game apart. Each division is in the league's tiebreak order (Tiebreakers shows each step).",
		"Franchise", "Record", "Win %", "Points for", "Points against", "Streak", "Games back")
	r.Note = strings.TrimSpace(r.Note + " " + v.sortNote)
	if r.Status == "failed" {
		return r
	}
	groups := slices.Clone(v.divisions)
	if len(groups) == 0 {
		groups = []league.Division{{}}
		r.Note = strings.TrimSpace(r.Note + " MFL's league settings list no divisions; refresh from MFL to group the standings.")
	}
	for _, d := range groups {
		order := v.table.Division(d.ID).Order
		for i, id := range order {
			l, _ := v.table.Line(id)
			lead, _ := v.table.Line(order[0])
			back := "—"
			if i > 0 {
				back = halfGamesText(race.HalfGamesBack(lead.Overall, l.Overall))
			}
			row := v.row(id, v.name(id), recordText(l.Overall), pctText(l.Overall),
				scoreText(l.For), scoreText(l.Against), streakText(l.Streak), back)
			if d.ID != "" {
				row.Group = v.divisionName(d.ID)
			}
			r.Rows = append(r.Rows, row)
		}
	}
	return r
}

func weekLabel(w leaguefeed.WeekResult) string {
	label := "Week " + strconv.Itoa(w.Week)
	if len(w.Games) > 0 && !w.Games[0].Regular {
		label += " · playoffs"
	}
	return label
}

func (v raceView) gameResult(g leaguefeed.Game) string {
	if !g.Home.Scored {
		return "Not played yet"
	}
	win, lose := g.Home, g.Away
	if g.Away.Result == "W" {
		win, lose = g.Away, g.Home
	}
	if win.Result != "W" {
		return "Tied " + scoreText(win.Points) + " to " + scoreText(lose.Points)
	}
	return v.name(win.Franchise) + " won " + scoreText(win.Points) + " to " + scoreText(lose.Points)
}

func (v raceView) scheduleByWeek() Report {
	r := v.weeksReport("M-011", "Fantasy schedule by week",
		"Every game MFL lists, week by week, away team first. Played games show MFL's score and winner.",
		"Week", "Away", "Home", "Result")
	for _, w := range v.weeks {
		for _, g := range w.Games {
			row := v.row("", weekLabel(w), v.name(g.Away.Franchise), v.name(g.Home.Franchise), v.gameResult(g))
			row.Mine = v.in.mine != "" && (g.Home.Franchise == v.in.mine || g.Away.Franchise == v.in.mine)
			r.Rows = append(r.Rows, row)
		}
	}
	return r
}

func (v raceView) needTeam(r Report) (Report, bool) {
	if r.Status == "failed" {
		return r, false
	}
	if v.team == "" {
		r.Status, r.Note = "empty", "Choose a franchise to see this report."
		return r, false
	}
	r.Summary = append(r.Summary, ReportRow{Cells: []string{"Franchise", v.name(v.team)}})
	return r, true
}

func (v raceView) scheduleByTeam() Report {
	r, ok := v.needTeam(v.weeksReport("M-012", "Schedule by franchise",
		"One franchise's games in week order. A week without a game is a playoff bye or a week after its season ended.",
		"Week", "Opponent", "Where", "Result"))
	if !ok {
		return r
	}
	for _, w := range v.weeks {
		game, found := findGame(w, v.team)
		if !found {
			r.Rows = append(r.Rows, v.row("", weekLabel(w), "—", "—", "No game this week"))
			continue
		}
		me, them, where := game.Home, game.Away, "Home"
		if game.Away.Franchise == v.team {
			me, them, where = game.Away, game.Home, "Away"
		}
		result := "Not played yet"
		if me.Scored {
			words := map[leaguefeed.Result]string{"W": "Won", "L": "Lost", "T": "Tied"}
			result = words[me.Result] + " " + scoreText(me.Points) + " to " + scoreText(them.Points)
		}
		r.Rows = append(r.Rows, v.row("", weekLabel(w), v.name(them.Franchise), where, result))
	}
	return r
}

func findGame(w leaguefeed.WeekResult, team string) (leaguefeed.Game, bool) {
	for _, g := range w.Games {
		if g.Home.Franchise == team || g.Away.Franchise == team {
			return g, true
		}
	}
	return leaguefeed.Game{}, false
}

func (v raceView) scheduleGrid() Report {
	columns := []string{"Franchise"}
	for _, w := range v.weeks {
		columns = append(columns, "W"+strconv.Itoa(w.Week))
	}
	r := v.weeksReport("M-013", "Schedule grid",
		"One row per franchise in standings order, one column per week. \"@\" marks an away game; W, L or T is "+
			"MFL's result once the game is played. Team short names are the league's own.", columns...)
	if r.Status == "failed" {
		return r
	}
	for _, id := range v.standingsOrder() {
		row := v.row(id, v.name(id))
		for _, w := range v.weeks {
			cell := ""
			if g, ok := findGame(w, id); ok {
				me, them, at := g.Home, g.Away, ""
				if g.Away.Franchise == id {
					me, them, at = g.Away, g.Home, "@"
				}
				cell = strings.TrimSpace(at + v.short(them.Franchise) + " " + string(me.Result))
			}
			row.Cells = append(row.Cells, cell)
		}
		r.Rows = append(r.Rows, row)
	}
	return r
}

// standingsOrder is every franchise division by division, as the standings list them.
func (v raceView) standingsOrder() []string {
	if len(v.divisions) == 0 {
		return v.table.Division("").Order
	}
	out := []string{}
	for _, d := range v.divisions {
		out = append(out, v.table.Division(d.ID).Order...)
	}
	return out
}
