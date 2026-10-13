package main

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

// NowBoard is League Pulse › Now: the week shown, the GM's own matchup first, then the week's
// reports. Week is the week Go accepted.
type NowBoard struct {
	ReportWeek  int          `json:"reportWeek"`
	ReportWeeks []RaceChoice `json:"reportWeeks"`
	Week        int          `json:"week"`
	Weeks       []RaceChoice `json:"weeks"`
	Matchup     Report       `json:"matchup"`
	Board       ReportBoard  `json:"board"`
}

type nowInput struct {
	race        raceInput
	held        nowHeld
	live        heldSeasonFeed[leaguefeed.Lineups]
	projections heldSeasonFeed[[]marketfeeds.Feed]
	at          time.Time
	reportWeek  int
	current     int
	week        int
}

// nowView is the race's view of the season plus the chosen week as shown: MFL's scored week, or
// its held live scores while MFL has not scored it.
type nowView struct {
	raceView
	now      nowInput
	shown    leaguefeed.WeekResult
	live     bool
	schedule heldSeasonFeed[[]nflschedule.RawWeek]
	games    map[string]nflGame
}

func buildNow(in nowInput) (NowBoard, error) {
	v, err := newNowView(in)
	if err != nil {
		return NowBoard{}, err
	}
	board := NowBoard{Week: v.week, Weeks: v.nowWeekChoices(), Matchup: v.myMatchup(),
		Board: ReportBoard{Title: "This week's reports", Loading: "Reading the week…",
			Refresh: "Refresh from MFL", HowLabel: "How this is counted", Reports: []Report{}}}
	if in.reportWeek > max(v.week, in.current) {
		return NowBoard{}, fmt.Errorf("now: report week %d has not begun", in.reportWeek)
	}
	reports := v
	reportWeek := in.reportWeek
	if reportWeek == 0 {
		for _, w := range v.weeks {
			if weekScored(w) && w.Week <= v.week {
				reportWeek = w.Week
			}
		}
	}
	if reportWeek != 0 {
		reportInput := in
		reportInput.week = reportWeek
		reports, err = newNowView(reportInput)
		if err != nil {
			return NowBoard{}, err
		}
	}
	board.ReportWeek = reports.week
	board.ReportWeeks = []RaceChoice{}
	for _, w := range v.weeks {
		if w.Week <= max(v.week, in.current) {
			board.ReportWeeks = append(board.ReportWeeks, RaceChoice{Value: strconv.Itoa(w.Week), Label: weekLabel(w)})
		}
	}
	board.Board.Reports = append(board.Board.Reports, v.matchups(), reports.results(), reports.weekSummary(), reports.recaps(),
		reports.topByPosition(), v.boxScore(), v.lastThree(), v.previews(), v.nflWeek(), v.nflBoxScore(), v.statChanges(), v.liveScoring())
	v.inspectable(&board.Matchup)
	for i := range board.Board.Reports {
		v.inspectable(&board.Board.Reports[i])
	}
	return board, nil
}

func newNowView(in nowInput) (nowView, error) {
	in.race.week, in.race.team = 0, ""
	rv, err := newRaceView(in.race)
	if err != nil {
		return nowView{}, fmt.Errorf("now: %w", err)
	}
	v := nowView{raceView: rv, now: in}
	if err := v.chooseNowWeek(in.week); err != nil {
		return nowView{}, err
	}
	for _, w := range v.weeks {
		if w.Week == v.week {
			v.shown = w
		}
	}
	if v.week != 0 && !weekScored(v.shown) && !in.live.fetchedAt.IsZero() && in.live.value.Week == v.week {
		v.shown, v.live = liveWeek(in.live.value, v.shown), true
	}
	v.schedule = in.held.schedules[v.week]
	v.games = nflGames(v.schedule.value, in.at)
	return v, nil
}

// chooseNowWeek accepts any week the league's schedule lists. 0 is the current NFL week, else the
// latest scored week, else the first listed.
func (v *nowView) chooseNowWeek(week int) error {
	listed := func(n int) bool {
		return slices.ContainsFunc(v.weeks, func(w leaguefeed.WeekResult) bool { return w.Week == n })
	}
	switch {
	case week != 0 && !listed(week):
		return fmt.Errorf("now: week %d is not on the league's schedule", week)
	case week != 0:
		v.week = week
	case v.now.current != 0 && listed(v.now.current):
		v.week = v.now.current
	case v.week == 0 && len(v.weeks) > 0:
		v.week = v.weeks[0].Week
	}
	return nil
}

func (v nowView) nowWeekChoices() []RaceChoice {
	out := []RaceChoice{}
	for _, w := range v.weeks {
		out = append(out, RaceChoice{Value: strconv.Itoa(w.Week), Label: weekLabel(w)})
	}
	return out
}

// liveWeek is the week as MFL's live scoring holds it: its games, each side's points so far and
// its starters and bench with their points.
func liveWeek(l leaguefeed.Lineups, base leaguefeed.WeekResult) leaguefeed.WeekResult {
	sides := map[string]leaguefeed.Side{}
	for _, f := range l.Franchises {
		scores := map[string]leaguefeed.Points{}
		for _, p := range f.Players {
			scores[p.ID.String()] = hundredths(p.Score)
		}
		s := leaguefeed.Side{Franchise: f.Franchise, Points: hundredths(f.Score)}
		for _, id := range f.Starters {
			s.Players = append(s.Players, leaguefeed.WeekPlayer{ID: id, Starter: true, Points: scores[id.String()]})
		}
		for _, id := range f.NonStarters {
			s.Players = append(s.Players, leaguefeed.WeekPlayer{ID: id, Points: scores[id.String()]})
		}
		sides[f.Franchise] = s
	}
	out := leaguefeed.WeekResult{Week: base.Week}
	for _, m := range l.Matchups {
		home, away := sides[m.Home], sides[m.Away]
		home.Franchise, away.Franchise = m.Home, m.Away
		regular := true
		if g, ok := findGame(base, m.Home); ok {
			regular = g.Regular
		}
		out.Games = append(out.Games, leaguefeed.Game{Regular: regular, Home: home, Away: away})
	}
	return out
}

func hundredths(points float64) leaguefeed.Points {
	return leaguefeed.Points(math.Round(points * 100))
}

// shownReport is a report on the chosen week as shown; held live scores carry their own time.
func (v nowView) shownReport(id, title, how string, columns ...string) Report {
	r := v.weeksReport(id, title, how, columns...)
	if v.live && r.Status != "failed" {
		r.AsOf = "As of: " + reportTime(v.now.live.fetchedAt, v.in.zone)
		r.Note = strings.TrimSpace(r.Note + " Held live scores, not final; they change when MFL is refreshed.")
	}
	r.Summary = append(r.Summary, ReportRow{Cells: []string{"Week", weekLabel(v.shown)}})
	return r
}

// scoredReport is a report that needs MFL's final scores for the chosen week.
func (v nowView) scoredReport(id, title, how string, columns ...string) (Report, bool) {
	r := v.weeksReport(id, title, how, columns...)
	if r.Status == "failed" {
		return r, false
	}
	r.Summary = append(r.Summary, ReportRow{Cells: []string{"Week", weekLabel(v.shown)}})
	if v.live || !weekScored(v.shown) {
		r.Status = "empty"
		r.Note = strings.TrimSpace(r.Note + fmt.Sprintf(" MFL has not scored week %d yet.", v.week))
		return r, false
	}
	return r, true
}

func (v nowView) pointsText(s leaguefeed.Side, p leaguefeed.Points) string {
	if s.Scored || v.live {
		return scoreText(p)
	}
	return "—"
}

// playerPointsText is a starter's points, or why there are none yet. With the week's NFL schedule
// held, a team missing from it has no game: on bye, or no NFL team at all for an NFL free agent.
func (v nowView) playerPointsText(s leaguefeed.Side, p leaguefeed.WeekPlayer) string {
	if !s.Scored {
		team := v.team(p)
		g, ok := v.games[team]
		switch {
		case ok && v.now.at.Before(g.kickoff):
			return "yet to play · " + g.kickoff.In(v.in.zone).Format("Mon 3:04 PM")
		case !ok && len(v.games) > 0:
			return noGameText(team)
		}
	}
	return v.pointsText(s, p.Points)
}

func noGameText(team string) string {
	if team == "" || team == "FA" {
		return "no NFL team"
	}
	return "bye"
}

// kickedOff reports whether any NFL game of the held week has started; unknown counts as started.
func (v nowView) kickedOff() bool {
	if len(v.games) == 0 {
		return true
	}
	for _, g := range v.games {
		if !v.now.at.Before(g.kickoff) {
			return true
		}
	}
	return false
}

func (v nowView) gameState(g leaguefeed.Game) string {
	switch {
	case g.Home.Scored:
		return v.gameResult(g)
	case v.live && !v.kickedOff():
		return "Not played yet"
	case v.live:
		// My side reads first, as in my matchup's columns; otherwise MFL's away-then-home order.
		first, second := g.Away, g.Home
		if second.Franchise == v.in.mine {
			first, second = second, first
		}
		return "In progress: " + v.name(first.Franchise) + " " + scoreText(first.Points) + ", " +
			v.name(second.Franchise) + " " + scoreText(second.Points)
	}
	return "Not played yet"
}

// mineFirst orders the week's games with the GM's own first, the rest as MFL lists them.
func (v nowView) mineFirst(games []leaguefeed.Game) []leaguefeed.Game {
	out := slices.Clone(games)
	slices.SortStableFunc(out, func(a, b leaguefeed.Game) int {
		return cmp.Compare(v.gameRank(a), v.gameRank(b))
	})
	return out
}

func (v nowView) gameRank(g leaguefeed.Game) int {
	if v.isMine(g) {
		return 0
	}
	return 1
}

func (v nowView) isMine(g leaguefeed.Game) bool {
	return v.in.mine != "" && (g.Home.Franchise == v.in.mine || g.Away.Franchise == v.in.mine)
}

// playerText names a player with his NFL team, never an id.
func (v nowView) playerText(p leaguefeed.WeekPlayer) string {
	name, _ := v.player(p.ID)
	if f, ok := v.in.lookup.Facts(p.ID.String()); ok && f.Team != "" {
		return name + " · " + f.Team
	}
	return name
}

// classicRows pairs two sides' starters slot by slot in the league's position order, then totals.
func (v nowView) classicRows(left, right leaguefeed.Side) []ReportRow {
	seen := map[string]bool{}
	slots := func(s leaguefeed.Side) map[string][]leaguefeed.WeekPlayer {
		out := map[string][]leaguefeed.WeekPlayer{}
		for _, p := range v.starters(s) {
			_, pos := v.player(p.ID)
			out[pos] = append(out[pos], p)
			seen[pos] = true
		}
		return out
	}
	l, r := slots(left), slots(right)
	if len(seen) == 0 {
		return []ReportRow{{Cells: []string{"Lineups", "No lineup set yet", "—", "No lineup set yet", "—"}}}
	}
	cell := func(s leaguefeed.Side, ps []leaguefeed.WeekPlayer, i int) (string, string) {
		if i >= len(ps) {
			return "—", ""
		}
		return v.playerText(ps[i]), v.playerPointsText(s, ps[i])
	}
	rows := []ReportRow{}
	for _, pos := range v.positions(seen) {
		for i := range max(len(l[pos]), len(r[pos])) {
			ln, lp := cell(left, l[pos], i)
			rn, rp := cell(right, r[pos], i)
			rows = append(rows, ReportRow{Cells: []string{pos, ln, lp, rn, rp}})
		}
	}
	return append(rows, ReportRow{Cells: []string{"Total", "", v.pointsText(left, left.Points), "",
		v.pointsText(right, right.Points)}})
}

// myGame is the GM's game in the week shown, the GM's side first.
func (v nowView) myGame() (me, them leaguefeed.Side, g leaguefeed.Game, ok bool) {
	if v.in.mine == "" {
		return me, them, g, false
	}
	if g, ok = findGame(v.shown, v.in.mine); !ok {
		return me, them, g, false
	}
	if g.Home.Franchise == v.in.mine {
		return g.Home, g.Away, g, true
	}
	return g.Away, g.Home, g, true
}

// needMine fills the GM's game into the report's summary, or says why there is none.
func (v nowView) needMine(r *Report) (me, them leaguefeed.Side, ok bool) {
	if r.Status == "failed" {
		return me, them, false
	}
	me, them, g, ok := v.myGame()
	switch {
	case v.in.mine == "":
		r.Status, r.Note = "empty", strings.TrimSpace(r.Note+" Choose your franchise to see your matchup.")
	case !ok:
		r.Status, r.Note = "empty", strings.TrimSpace(r.Note+" "+v.name(v.in.mine)+" has no game this week.")
	default:
		r.Summary = append(r.Summary, ReportRow{Cells: []string{"Opponent", v.name(them.Franchise)}},
			ReportRow{Cells: []string{"Status", v.gameState(g)}})
	}
	return me, them, ok
}

func (v nowView) myMatchup() Report {
	r := v.shownReport("X-03", "My matchup",
		"Both lineups side by side, starter against starter in the league's slot order, with each starter's "+
			"points and the team totals as MFL scores them. A week MFL has not scored shows the held live scores, "+
			"or the lineups set so far.", "Slot", "Mine", "Points", "Opponent", "Points")
	v.directoryNote(&r)
	me, them, ok := v.needMine(&r)
	if !ok {
		return r
	}
	r.Columns = []string{"Slot", v.name(me.Franchise), "Points", v.name(them.Franchise), "Points"}
	r.Rows = v.classicRows(me, them)
	for i := range r.Rows {
		r.Rows[i].Mine = false
	}
	return r
}

func (v nowView) matchups() Report {
	r := v.shownReport("M-005", "Every matchup",
		"Each game of the week, away team on the left, starter against starter in the league's slot order "+
			"with each one's points and the totals. My game first.", "Slot", "Away", "Points", "Home", "Points")
	v.directoryNote(&r)
	if r.Status == "failed" {
		return r
	}
	if len(v.shown.Games) == 0 {
		r.Status, r.Note = "empty", strings.TrimSpace(r.Note+" MFL lists no games this week.")
	}
	for _, g := range v.mineFirst(v.shown.Games) {
		group := v.name(g.Away.Franchise) + " at " + v.name(g.Home.Franchise) + " · " + v.gameState(g)
		for _, row := range v.classicRows(g.Away, g.Home) {
			row.Group, row.Mine = group, v.isMine(g)
			r.Rows = append(r.Rows, row)
		}
	}
	return r
}

// winnerFirst is a scored game's winner and loser; tied is true when MFL called it a tie.
func winnerFirst(g leaguefeed.Game) (win, lose leaguefeed.Side, tied bool) {
	if g.Away.Result == "W" {
		return g.Away, g.Home, false
	}
	return g.Home, g.Away, g.Home.Result != "W"
}

func (v nowView) results() Report {
	r, ok := v.scoredReport("M-003", "Weekly results",
		"Every game's score and winner as MFL scored it. Playoff scores include the home-field points MFL adds. "+
			"My game first, then the highest winning score.", "Winner", "Points", "Loser", "Points", "Margin")
	if !ok {
		return r
	}
	for _, g := range v.resultOrder() {
		win, lose, tied := winnerFirst(g)
		margin := scoreText(win.Points - lose.Points)
		if tied {
			margin = "Tied"
		}
		row := v.row("", v.name(win.Franchise), scoreText(win.Points), v.name(lose.Franchise), scoreText(lose.Points), margin)
		row.Mine = v.isMine(g)
		r.Rows = append(r.Rows, row)
	}
	if others := v.othersText(v.shown); others != "" {
		r.Summary = append(r.Summary, ReportRow{Cells: []string{"Scored without a game", others}})
	}
	return r
}

// resultOrder is the week's games with my game first, then the highest winning score.
func (v nowView) resultOrder() []leaguefeed.Game {
	games := slices.Clone(v.shown.Games)
	slices.SortStableFunc(games, func(a, b leaguefeed.Game) int {
		wa, _, _ := winnerFirst(a)
		wb, _, _ := winnerFirst(b)
		return cmp.Or(cmp.Compare(v.gameRank(a), v.gameRank(b)), cmp.Compare(wb.Points, wa.Points))
	})
	return games
}

func (v nowView) othersText(w leaguefeed.WeekResult) string {
	parts := []string{}
	for _, s := range w.Others {
		if s.Scored {
			parts = append(parts, v.name(s.Franchise)+" "+scoreText(s.Points))
		}
	}
	return strings.Join(parts, ", ")
}

// lastThree is each franchise's last three scored weeks up to the week shown.
func (v nowView) lastThree() Report {
	weeks := []leaguefeed.WeekResult{}
	for _, w := range v.weeks {
		if w.Week <= v.week && weekScored(w) {
			weeks = append(weeks, w)
		}
	}
	weeks = weeks[max(len(weeks)-3, 0):]
	columns := []string{"Franchise"}
	for _, w := range weeks {
		columns = append(columns, "Week "+strconv.Itoa(w.Week))
	}
	r := v.weeksReport("M-004", "Last 3 weeks",
		"Each franchise's result and points in the last three weeks MFL scored, up to the week chosen, then its "+
			"points and record over those weeks. A week without a game counts its points but no result. Most "+
			"points first.", append(columns, "Points", "Record")...)
	if r.Status == "failed" {
		return r
	}
	if len(weeks) == 0 {
		r.Status, r.Note = "empty", strings.TrimSpace(r.Note+" MFL has scored no week up to this one.")
		return r
	}
	type form struct {
		id     string
		cells  []string
		points leaguefeed.Points
		w, l   int
		t      int
	}
	forms := []*form{}
	for _, id := range v.table.IDs() {
		f := &form{id: id}
		for _, w := range weeks {
			f.cells = append(f.cells, v.formCell(w, id, &f.points, &f.w, &f.l, &f.t))
		}
		forms = append(forms, f)
	}
	slices.SortStableFunc(forms, func(a, b *form) int { return cmp.Compare(b.points, a.points) })
	for _, f := range forms {
		rec := fmt.Sprintf("%d-%d", f.w, f.l)
		if f.t > 0 {
			rec += fmt.Sprintf("-%d", f.t)
		}
		cells := append(append([]string{v.name(f.id)}, f.cells...), scoreText(f.points), rec)
		r.Rows = append(r.Rows, v.row(f.id, cells...))
	}
	return r
}

func (v nowView) formCell(w leaguefeed.WeekResult, id string, points *leaguefeed.Points, won, lost, tied *int) string {
	if g, ok := findGame(w, id); ok {
		me := g.Home
		if g.Away.Franchise == id {
			me = g.Away
		}
		*points += me.Points
		switch me.Result {
		case "W":
			*won++
		case "L":
			*lost++
		default:
			*tied++
		}
		return string(me.Result) + " " + scoreText(me.Points)
	}
	for _, s := range w.Others {
		if s.Franchise == id && s.Scored {
			*points += s.Points
			return "No game, " + scoreText(s.Points)
		}
	}
	return "—"
}

func (v nowView) liveScoring() Report {
	r := v.shownReport("M-001", "Live fantasy scoring", "MFL liveScoring, refreshed every 30 seconds while Now is open and a held NFL game has kicked off but is not final. Errors back off and keep the held scores. My matchup reads first.", "Matchup", "Franchise", "Points")
	if !v.live {
		r.Note += " Off-game view: held weekly scores; no live poll."
	}
	for _, g := range v.mineFirst(v.shown.Games) {
		sides := []leaguefeed.Side{g.Away, g.Home}
		if g.Home.Franchise == v.in.mine {
			sides[0], sides[1] = sides[1], sides[0]
		}
		for _, side := range sides {
			r.Rows = append(r.Rows, ReportRow{Cells: []string{v.gameState(g), v.name(side.Franchise), v.pointsText(side, side.Points)}, Mine: side.Franchise == v.in.mine})
		}
	}
	if v.live && v.now.live.refreshError != "" {
		r.Status = "stale"
		r.Note += " " + v.now.live.refreshError
	}
	return r
}
