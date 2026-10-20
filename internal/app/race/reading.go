package race

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/engine/race"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// RaceBoard is League Pulse › Race: the standings on top, then the reports. Week and Team are the
// choices the lineup and one-team reports follow, as Go accepted them.
type Board struct {
	Standings report.Report   `json:"standings"`
	Board     report.Board    `json:"board"`
	Week      int             `json:"week"`
	Weeks     []report.Choice `json:"weeks"`
	Team      string          `json:"team"`
	Teams     []report.Choice `json:"teams"`
}

type Snapshot struct {
	LiveWeeks map[int]leaguefeed.WeekResult
	Live      Feed[leaguefeed.Lineups]
	// liveUnplayed: no NFL game of the live week has kicked off, so its 0.00 scores are not a game in progress.
	LiveUnplayed bool
	Held         Held
	Config       league.RawConfig
	Lookup       normalize.Lookup
	Week         int
	Team         string
	Mine         string
	Zone         *time.Location
	HasLookup    bool
}

// raceView is everything the reports read, resolved once: the season's table, names, the
// league's groupings and the accepted choices.
type View struct {
	in          Snapshot
	Table       race.Table
	SortNote    string
	Weeks       []leaguefeed.WeekResult
	Names       map[string]string
	Abbrev      map[string]string
	Conferences []league.Conference
	Divisions   []league.Division
	Week        int
	Team        string
}

func Build(in Snapshot) (Board, error) {
	v, err := NewView(in)
	if err != nil {
		return Board{}, err
	}
	board := Board{Standings: v.Standings(), Week: v.Week, Team: v.Team,
		Weeks: v.WeekChoices(), Teams: v.TeamChoices(),
		Board: report.Board{Title: "League race reports", Loading: "Reading the league race…",
			Refresh: "Refresh from MFL", HowLabel: "How this is counted", Reports: []report.Report{}}}
	board.Board.Reports = append(board.Board.Reports, v.Power(season.ViewSeason), v.Power(season.ViewFranchise), v.Seeding(),
		v.Bubble(), v.Tiebreakers(), v.Brackets(), v.AllPlay(), v.WeeklyPoints(), v.PositionPoints(),
		v.PlayerPoints(), v.ScheduleByWeek(), v.ScheduleByTeam(), v.ScheduleGrid(), v.Lineups(), v.LineupDetail())
	return board, nil
}

func NewView(in Snapshot) (View, error) {
	v := View{in: in, Names: map[string]string{}, Abbrev: map[string]string{},
		Conferences: in.Config.Conferences, Divisions: in.Config.Divisions}
	v.Weeks = slices.Clone(in.Held.Weeks.Value)
	slices.SortFunc(v.Weeks, func(a, b leaguefeed.WeekResult) int { return a.Week - b.Week })
	confOf := map[string]string{}
	for _, d := range v.Divisions {
		confOf[d.ID] = d.Conference
	}
	teams := []race.Franchise{}
	for _, f := range in.Config.Franchises {
		v.Names[f.ID], v.Abbrev[f.ID] = f.Name, f.Abbrev
		teams = append(teams, race.Franchise{ID: f.ID, Division: f.Division, Conference: confOf[f.Division]})
	}
	for _, id := range []string{in.Team, in.Mine} {
		if _, ok := v.Names[id]; id != "" && !ok {
			return View{}, fmt.Errorf("race: franchise %q is not in the league", id)
		}
	}
	v.Team = in.Team
	keys, err := race.ParseSort(in.Config.StandingsSort)
	if err != nil {
		keys, v.SortNote = []race.Key{race.KeyPct}, "MFL's tiebreak order is not held, so ties keep MFL's franchise order; refresh from MFL."
	}
	if v.Table, err = race.Build(teams, v.Weeks, keys, mflPower(in.Held.Standings.Value)); err != nil {
		return View{}, fmt.Errorf("race: %w", err)
	}
	return v, v.ChooseWeek(in.Week)
}

// chooseWeek accepts a scored week, or 0 for the latest one.
func (v *View) ChooseWeek(week int) error {
	for _, w := range v.Weeks {
		if WeekScored(w) && (week == 0 || week == w.Week) {
			v.Week = w.Week
		}
	}
	if week != 0 && v.Week != week {
		return fmt.Errorf("race: week %d has no scored games", week)
	}
	return nil
}

func WeekScored(w leaguefeed.WeekResult) bool {
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

func (v View) WeekChoices() []report.Choice {
	out := []report.Choice{}
	for _, w := range v.Weeks {
		if WeekScored(w) {
			out = append(out, report.Choice{Value: strconv.Itoa(w.Week), Label: "Week " + strconv.Itoa(w.Week)})
		}
	}
	return out
}

func (v View) TeamChoices() []report.Choice {
	out := []report.Choice{}
	for _, id := range v.Table.IDs() {
		out = append(out, report.Choice{Value: id, Label: v.Name(id)})
	}
	slices.SortStableFunc(out, func(a, b report.Choice) int { return strings.Compare(a.Label, b.Label) })
	return out
}

func (v View) Name(id string) string {
	switch {
	case v.Names[id] != "":
		return v.Names[id]
	case v.Abbrev[id] != "":
		return v.Abbrev[id]
	}
	return "Unnamed franchise"
}

func (v View) Short(id string) string {
	if v.Abbrev[id] != "" {
		return v.Abbrev[id]
	}
	return v.Name(id)
}

func (v View) Row(id string, cells ...string) report.Row {
	return report.Row{Cells: cells, Mine: id != "" && id == v.in.Mine}
}

func (v View) WeeksReport(id, title, how string, columns ...string) report.Report {
	return shell(id, title, how, v.in.Held.Weeks, v.in.Zone, columns...)
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

func scoreText(p leaguefeed.Points) string { return report.ScoreText(p) }

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

func (v View) DivisionName(id string) string {
	for _, d := range v.Divisions {
		if d.ID == id {
			return d.Name
		}
	}
	return "No division"
}

func (v View) Standings() report.Report {
	r := v.WeeksReport("M-002", "Standings",
		"Records count regular-season games by MFL's result; a tie counts half a win in win percentage. "+
			"Points for counts every week MFL scored, playoff weeks included, as MFL's own standings do; points "+
			"against counts regular-season games. Games back trail the division leader: a win and a loss are one "+
			"game apart. Each division is in the league's tiebreak order (Tiebreakers shows each step).",
		"Franchise", "Record", "Win %", "Points for", "Points against", "Streak", "Games back")
	r.Note = strings.TrimSpace(r.Note + " " + v.SortNote)
	if r.Status == "failed" {
		return r
	}
	groups := slices.Clone(v.Divisions)
	if len(groups) == 0 {
		groups = []league.Division{{}}
		r.Note = strings.TrimSpace(r.Note + " MFL's league settings list no divisions; refresh from MFL to group the standings.")
	}
	for _, d := range groups {
		order := v.Table.Division(d.ID).Order
		for i, id := range order {
			l, _ := v.Table.Line(id)
			lead, _ := v.Table.Line(order[0])
			back := "—"
			if i > 0 {
				back = halfGamesText(race.HalfGamesBack(lead.Overall, l.Overall))
			}
			row := v.Row(id, v.Name(id), recordText(l.Overall), pctText(l.Overall),
				scoreText(l.For), scoreText(l.Against), streakText(l.Streak), back)
			if d.ID != "" {
				row.Group = v.DivisionName(d.ID)
			}
			r.Rows = append(r.Rows, row)
		}
	}
	return r
}

func WeekLabel(w leaguefeed.WeekResult) string {
	label := "Week " + strconv.Itoa(w.Week)
	if len(w.Games) > 0 && !w.Games[0].Regular {
		label += " · playoffs"
	}
	return label
}

func (v View) GameResult(g leaguefeed.Game) string {
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
	return v.Name(win.Franchise) + " won " + scoreText(win.Points) + " to " + scoreText(lose.Points)
}

func (v View) ScheduleByWeek() report.Report {
	r := v.WeeksReport("M-011", "Fantasy schedule by week",
		"Every game MFL lists, week by week, away team first. Played games show MFL's score and winner.",
		"Week", "Away", "Home", "Result")
	for _, w := range v.Weeks {
		for _, g := range w.Games {
			row := v.Row("", WeekLabel(w), v.Name(g.Away.Franchise), v.Name(g.Home.Franchise), v.GameResult(g))
			row.Mine = v.in.Mine != "" && (g.Home.Franchise == v.in.Mine || g.Away.Franchise == v.in.Mine)
			r.Rows = append(r.Rows, row)
		}
	}
	return r
}

func (v View) NeedTeam(r report.Report) (report.Report, bool) {
	if r.Status == "failed" {
		return r, false
	}
	if v.Team == "" {
		r.Status, r.Note = "empty", "Choose a franchise to see this report."
		return r, false
	}
	r.Summary = append(r.Summary, report.Row{Cells: []string{"Franchise", v.Name(v.Team)}})
	return r, true
}

func (v View) ScheduleByTeam() report.Report {
	r, ok := v.NeedTeam(v.WeeksReport("M-012", "Schedule by franchise",
		"One franchise's games in week order. A week without a game is a playoff bye or a week after its season ended.",
		"Week", "Opponent", "Where", "Result"))
	if !ok {
		return r
	}
	for _, w := range v.Weeks {
		game, found := FindGame(w, v.Team)
		if !found {
			r.Rows = append(r.Rows, v.Row("", WeekLabel(w), "—", "—", "No game this week"))
			continue
		}
		me, them, where := game.Home, game.Away, "Home"
		if game.Away.Franchise == v.Team {
			me, them, where = game.Away, game.Home, "Away"
		}
		result := "Not played yet"
		if !me.Scored && v.in.Live.Value.Week == w.Week && !v.in.Live.FetchedAt.IsZero() && !v.in.LiveUnplayed {
			if live, found := FindGame(v.in.LiveWeeks[w.Week], v.Team); found {
				left, right := live.Home, live.Away
				if left.Franchise != v.Team {
					left, right = right, left
				}
				result = "In progress · " + scoreText(left.Points) + " to " + scoreText(right.Points)
			}
		}
		if me.Scored {
			words := map[leaguefeed.Result]string{"W": "Won", "L": "Lost", "T": "Tied"}
			result = words[me.Result] + " " + scoreText(me.Points) + " to " + scoreText(them.Points)
		}
		r.Rows = append(r.Rows, v.Row("", WeekLabel(w), v.Name(them.Franchise), where, result))
	}
	return r
}

func FindGame(w leaguefeed.WeekResult, team string) (leaguefeed.Game, bool) {
	for _, g := range w.Games {
		if g.Home.Franchise == team || g.Away.Franchise == team {
			return g, true
		}
	}
	return leaguefeed.Game{}, false
}

func (v View) ScheduleGrid() report.Report {
	columns := []string{"Franchise"}
	for _, w := range v.Weeks {
		columns = append(columns, "W"+strconv.Itoa(w.Week))
	}
	r := v.WeeksReport("M-013", "Schedule grid",
		"One row per franchise in standings order, one column per week. \"@\" marks an away game; W, L or T is "+
			"MFL's result once the game is played. Team short names are the league's own.", columns...)
	if r.Status == "failed" {
		return r
	}
	for _, id := range v.StandingsOrder() {
		row := v.Row(id, v.Name(id))
		for _, w := range v.Weeks {
			cell := ""
			if g, ok := FindGame(w, id); ok {
				me, them, at := g.Home, g.Away, ""
				if g.Away.Franchise == id {
					me, them, at = g.Away, g.Home, "@"
				}
				cell = strings.TrimSpace(at + v.Short(them.Franchise) + " " + string(me.Result))
			}
			row.Cells = append(row.Cells, cell)
		}
		r.Rows = append(r.Rows, row)
	}
	return r
}

// standingsOrder is every franchise division by division, as the standings list them.
func (v View) StandingsOrder() []string {
	if len(v.Divisions) == 0 {
		return v.Table.Division("").Order
	}
	out := []string{}
	for _, d := range v.Divisions {
		out = append(out, v.Table.Division(d.ID).Order...)
	}
	return out
}
