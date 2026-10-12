package main

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/engine/race"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
)

// shareBar draws value on a shared axis from zero to max, so bars compare down a column.
func shareBar(value, scale float64, text string) *MarketRange {
	value = math.Max(value, 0)
	return &MarketRange{Interval: lab.Interval{Value: value, Lo: 0, Hi: value}, Max: math.Max(scale, value), Text: text}
}

func moveText(r PowerRow) string {
	switch {
	case !r.DeltaOK:
		return "—"
	case r.RankDelta > 0:
		return fmt.Sprintf("Up %d", r.RankDelta)
	case r.RankDelta < 0:
		return fmt.Sprintf("Down %d", -r.RankDelta)
	}
	return "No change"
}

func aggText(res PowerRankingsResult) string {
	switch res.AggMode {
	case m2service.AggLineup:
		return "the best lineup the league's starter rules allow"
	case m2service.AggSum:
		return "the whole roster"
	}
	return fmt.Sprintf("its top %d players", res.StarterN)
}

func powerHow(res PowerRankingsResult) string {
	if res.View == ViewFranchise {
		return fmt.Sprintf("Each roster's dynasty value (%s, from the app's model run #%d), with an older roster "+
			"counting against it: age weighs %.1f against value, both on one scale. That weight was fitted on the "+
			"league's own 2022–2025 seasons, where older rosters scored worse over the next two seasons than their "+
			"dynasty value alone said. This season's results do not count. The blend runs from 0 (last) to 1 (first); "+
			"moves compare with the board from the previous model run.", aggText(res), res.ModelRunID, res.AgeWeight)
	}
	perf := map[string]string{m2service.PerfAllPlay: "all-play win percentage",
		m2service.PerfPointsFor: "points for, as a share of the league's best"}[res.Performance]
	if perf == "" {
		perf = "none yet, so the roster alone ranks"
	}
	roster := int(math.Round(res.Weight * 100))
	return fmt.Sprintf("Each roster's on-field value now (%s, from the app's model run #%d) blended with this "+
		"season's results (%s). After %d weeks of results the roster counts %d%% and results %d%%: the roster's "+
		"share is %.0f ÷ (%.0f + weeks played), fitted on the league's own 2022–2025 seasons. Both parts are put "+
		"on one scale before blending, so the shares are real. The blend runs from 0 (last) to 1 (first); moves "+
		"compare with the board from the previous model run.", aggText(res), res.ModelRunID, perf,
		res.WeeksScored, roster, 100-roster, powerrankings.RosterCredibilityWeeks, powerrankings.RosterCredibilityWeeks)
}

func (v raceView) power(view string) Report {
	id, title, columns := "M-008", "Power rankings: this season",
		[]string{"Rank", "Franchise", "Move", "Record", "Roster value", "Results", "Blend"}
	if view == ViewFranchise {
		id, title, columns = "C-02", "Power rankings: the franchise",
			[]string{"Rank", "Franchise", "Move", "Roster value", "Roster age", "Blend"}
	}
	r := reportShell(id, title, "", v.in.held.standings, v.in.zone, columns...)
	r.Source = "Source: MFL standings and the app's model"
	res, held := v.in.held.power[view]
	switch {
	case r.Status == "failed":
		return r
	case !held || !res.OK:
		r.Status, r.Note = "failed", strings.TrimSpace("Power rankings unavailable. "+res.Error)
		return r
	case len(res.Rows) == 0:
		r.Status, r.Note = "empty", "No model run is held for this season; run the app's model first."
		return r
	}
	r.How = powerHow(res)
	for _, p := range res.Rows {
		value := fmt.Sprintf("%.1f", p.RosterValue)
		cells := []string{strconv.Itoa(p.Rank), v.name(p.FranchiseID), moveText(p)}
		if view == ViewFranchise {
			age := "—"
			if p.Age > 0 {
				age = fmt.Sprintf("%.1f", p.Age)
			}
			cells = append(cells, value, age)
		} else {
			result := strings.TrimPrefix(fmt.Sprintf("%.3f", p.Results), "0")
			if res.Performance == m2service.PerfPointsFor {
				result = fmt.Sprintf("%.0f%% of the best", p.Results*100)
			}
			cells = append(cells, recordText(race.Record{W: p.H2HW, L: p.H2HL, T: p.H2HT}), value, result)
		}
		blend := fmt.Sprintf("%.2f", p.PowerScore)
		row := v.row(p.FranchiseID, append(cells, blend)...)
		row.Range = shareBar(p.PowerScore, 1, "Blend "+blend+" of 1")
		r.Rows = append(r.Rows, row)
	}
	return r
}

func (v raceView) allPlay() Report {
	r := v.weeksReport("M-161", "All-play",
		"Each week, every franchise's score is compared with every other franchise's score that week: higher is "+
			"an all-play win, lower a loss, equal a tie. Every week MFL scored counts, playoff weeks included, as "+
			"MFL's own all-play does. Wins against all-play pace are actual regular-season wins minus the wins "+
			"the all-play percentage gives over the same games. Every bar runs from 0 to 1.",
		"Franchise", "All-play record", "Record", "Wins against all-play pace", "All-play %")
	ids := v.table.IDs()
	slices.SortStableFunc(ids, func(a, b string) int {
		la, _ := v.table.Line(a)
		lb, _ := v.table.Line(b)
		return cmp.Or(-cmpPct(la.AllPlay, lb.AllPlay), cmp.Compare(lb.For, la.For))
	})
	for _, id := range ids {
		l, _ := v.table.Line(id)
		if l.AllPlay.Games() == 0 {
			continue
		}
		share := float64(2*l.AllPlay.W+l.AllPlay.T) / float64(2*l.AllPlay.Games())
		pace := float64(l.Overall.W) + float64(l.Overall.T)/2 - share*float64(l.Overall.Games())
		row := v.row(id, v.name(id), recordText(l.AllPlay), recordText(l.Overall), fmt.Sprintf("%+.1f", pace), pctText(l.AllPlay))
		row.Range = shareBar(share, 1, "All-play "+pctText(l.AllPlay))
		r.Rows = append(r.Rows, row)
	}
	return r
}

func cmpPct(a, b race.Record) int {
	return cmp.Compare(float64(2*a.W+a.T)/float64(2*max(a.Games(), 1)), float64(2*b.W+b.T)/float64(2*max(b.Games(), 1)))
}

// scoredSides is every franchise week MFL scored, in week order.
func (v raceView) scoredSides(team string) []struct {
	week int
	side leaguefeed.Side
} {
	out := []struct {
		week int
		side leaguefeed.Side
	}{}
	for _, w := range v.weeks {
		sides := slices.Clone(w.Others)
		for _, g := range w.Games {
			sides = append(sides, g.Home, g.Away)
		}
		for _, s := range sides {
			if s.Scored && (team == "" || s.Franchise == team) {
				out = append(out, struct {
					week int
					side leaguefeed.Side
				}{w.Week, s})
			}
		}
	}
	return out
}

func (v raceView) weeklyPoints() Report {
	r := v.weeksReport("M-071", "Weekly points",
		"Each franchise's score in every week MFL scored, most points first. All bars share one scale, from zero "+
			"to the highest score any franchise posted in a week, so they compare down the column.",
		"Franchise", "Points for", "Per week", "Points by week")
	scored := []int{}
	for _, w := range v.weeks {
		if weekScored(w) {
			scored = append(scored, w.Week)
			r.BarLabels = append(r.BarLabels, "Week "+strconv.Itoa(w.Week))
		}
	}
	byTeam := map[string]map[int]leaguefeed.Points{}
	for _, s := range v.scoredSides("") {
		if byTeam[s.side.Franchise] == nil {
			byTeam[s.side.Franchise] = map[int]leaguefeed.Points{}
		}
		byTeam[s.side.Franchise][s.week] = s.side.Points
		r.BarMax = math.Max(r.BarMax, float64(s.side.Points)/100)
	}
	ids := v.table.IDs()
	slices.SortStableFunc(ids, func(a, b string) int {
		la, _ := v.table.Line(a)
		lb, _ := v.table.Line(b)
		return cmp.Compare(lb.For, la.For)
	})
	for _, id := range ids {
		l, _ := v.table.Line(id)
		weeks := byTeam[id]
		if len(weeks) == 0 {
			continue
		}
		row := v.row(id, v.name(id), scoreText(l.For), scoreText(l.For/leaguefeed.Points(len(weeks))), "")
		best := scored[0]
		for _, wk := range scored {
			row.Bars = append(row.Bars, math.Max(float64(weeks[wk])/100, 0))
			if weeks[wk] > weeks[best] {
				best = wk
			}
		}
		row.Cells[3] = fmt.Sprintf("Best: week %d, %s", best, scoreText(weeks[best]))
		r.Rows = append(r.Rows, row)
	}
	if r.BarMax <= 0 {
		r.BarMax, r.BarLabels = 0, nil
		for i := range r.Rows {
			r.Rows[i].Bars = nil
		}
	}
	return r
}

// player names a player from the held players directory, never by id.
func (v raceView) player(id playerid.PlayerID) (name, position string) {
	if f, ok := v.in.lookup.Facts(id.String()); ok {
		return f.Name, string(f.Position)
	}
	if n, ok := v.in.lookup.Name(id.String()); ok && n != "" {
		return n, "Team unit"
	}
	return "A player MFL's directory does not list", "Position unknown"
}

func (v raceView) directoryNote(r *Report) {
	if !v.in.hasLookup && r.Status != "failed" {
		r.Note = strings.TrimSpace(r.Note + " Player names are not loaded yet; refresh from MFL.")
	}
}

// positions is the league's starter positions in its own order, then any other position seen.
func (v raceView) positions(seen map[string]bool) []string {
	out := []string{}
	for _, p := range v.in.config.Starters.Positions {
		if pos, _ := normalize.PositionFromMFL(p.Name); !slices.Contains(out, string(pos)) {
			out = append(out, string(pos))
		}
	}
	extra := []string{}
	for p := range seen {
		if !slices.Contains(out, p) {
			extra = append(extra, p)
		}
	}
	slices.Sort(extra)
	return append(out, extra...)
}

func (v raceView) positionPoints() Report {
	totals, seen := map[string]map[string]leaguefeed.Points{}, map[string]bool{}
	for _, s := range v.scoredSides("") {
		for _, p := range s.side.Players {
			if !p.Starter {
				continue
			}
			_, pos := v.player(p.ID)
			if totals[s.side.Franchise] == nil {
				totals[s.side.Franchise] = map[string]leaguefeed.Points{}
			}
			totals[s.side.Franchise][pos] += p.Points
			seen[pos] = true
		}
	}
	positions := v.positions(seen)
	r := v.weeksReport("M-076", "Starter points by position",
		"Points scored by each franchise's starters, by position, in every week MFL scored. Bench points do not "+
			"count, and neither do the home-field points MFL adds in playoff games, so a playoff host's total sits "+
			"just under its points for. Most starter points first.", append([]string{"Franchise", "Starter points"}, positions...)...)
	v.directoryNote(&r)
	ids := slices.Collect(func(yield func(string) bool) {
		for id := range totals {
			if !yield(id) {
				return
			}
		}
	})
	sum := func(id string) (t leaguefeed.Points) {
		for _, p := range totals[id] {
			t += p
		}
		return t
	}
	slices.SortFunc(ids, func(a, b string) int { return cmp.Or(cmp.Compare(sum(b), sum(a)), strings.Compare(a, b)) })
	for _, id := range ids {
		row := v.row(id, v.name(id), scoreText(sum(id)))
		for _, pos := range positions {
			row.Cells = append(row.Cells, scoreText(totals[id][pos]))
		}
		r.Rows = append(r.Rows, row)
	}
	return r
}

func (v raceView) playerPoints() Report {
	r, ok := v.needTeam(v.weeksReport("M-077", "Starter points by player",
		"Every player the franchise started in a week MFL scored, with his points in those starts. Bench weeks do "+
			"not count. Most points first.", "Player", "Position", "Starts", "Points as starter", "Per start"))
	if !ok {
		return r
	}
	v.directoryNote(&r)
	type tally struct {
		id     playerid.PlayerID
		starts int
		points leaguefeed.Points
	}
	byID := map[string]*tally{}
	for _, s := range v.scoredSides(v.team) {
		for _, p := range s.side.Players {
			if p.Starter {
				if byID[p.ID.String()] == nil {
					byID[p.ID.String()] = &tally{id: p.ID}
				}
				byID[p.ID.String()].starts++
				byID[p.ID.String()].points += p.Points
			}
		}
	}
	rows := slices.Collect(func(yield func(*tally) bool) {
		for _, t := range byID {
			if !yield(t) {
				return
			}
		}
	})
	slices.SortFunc(rows, func(a, b *tally) int {
		return cmp.Or(cmp.Compare(b.points, a.points), strings.Compare(a.id.String(), b.id.String()))
	})
	for _, t := range rows {
		name, pos := v.player(t.id)
		r.Rows = append(r.Rows, v.row("", name, pos, strconv.Itoa(t.starts), scoreText(t.points),
			scoreText(t.points/leaguefeed.Points(t.starts))))
	}
	return r
}

// weekSides is the chosen week's franchise sides.
func (v raceView) weekSides() (leaguefeed.WeekResult, []leaguefeed.Side) {
	for _, w := range v.weeks {
		if w.Week == v.week {
			sides := slices.Clone(w.Others)
			for _, g := range w.Games {
				sides = append(sides, g.Home, g.Away)
			}
			return w, sides
		}
	}
	return leaguefeed.WeekResult{}, nil
}

// starters lists a side's starters in the league's position order.
func (v raceView) starters(s leaguefeed.Side) []leaguefeed.WeekPlayer {
	order := v.positions(nil)
	out := []leaguefeed.WeekPlayer{}
	for _, p := range s.Players {
		if p.Starter {
			out = append(out, p)
		}
	}
	rank := func(p leaguefeed.WeekPlayer) int {
		_, pos := v.player(p.ID)
		if i := slices.Index(order, pos); i >= 0 {
			return i
		}
		return len(order)
	}
	slices.SortStableFunc(out, func(a, b leaguefeed.WeekPlayer) int { return cmp.Compare(rank(a), rank(b)) })
	return out
}

func (v raceView) lineups() Report {
	r := v.weeksReport("M-009", "Starting lineups",
		"Every franchise's starters in the chosen week, by position, with each starter's points. Most points first.",
		"Franchise", "Points", "Starters")
	v.directoryNote(&r)
	if r.Status != "ok" && r.Status != "stale" {
		return r
	}
	_, sides := v.weekSides()
	r.Summary = append(r.Summary, ReportRow{Cells: []string{"Week", "Week " + strconv.Itoa(v.week)}})
	slices.SortStableFunc(sides, func(a, b leaguefeed.Side) int { return cmp.Compare(b.Points, a.Points) })
	for _, s := range sides {
		lines := []string{}
		for _, p := range v.starters(s) {
			name, pos := v.player(p.ID)
			lines = append(lines, pos+" · "+name+" · "+scoreText(p.Points))
		}
		if len(lines) == 0 {
			lines = append(lines, "No lineup listed")
		}
		r.Rows = append(r.Rows, v.row(s.Franchise, v.name(s.Franchise), scoreText(s.Points), strings.Join(lines, "\n")))
	}
	return r
}

func (v raceView) lineupDetail() Report {
	r, ok := v.needTeam(v.weeksReport("M-010", "Lineup detail",
		"The franchise's starters in the chosen week with each one's points, then any bench player MFL's best "+
			"lineup would have started. Best lineup is the most the same roster could have scored under the "+
			"league's starter rules, as MFL counts it.", "Position", "Player", "Points", "Lineup call"))
	if !ok {
		return r
	}
	v.directoryNote(&r)
	w, sides := v.weekSides()
	idx := slices.IndexFunc(sides, func(s leaguefeed.Side) bool { return s.Franchise == v.team })
	r.Summary = append(r.Summary, ReportRow{Cells: []string{"Week", "Week " + strconv.Itoa(v.week)}})
	if idx < 0 {
		r.Status, r.Note = "empty", strings.TrimSpace(r.Note+" MFL lists no lineup for this franchise in this week.")
		return r
	}
	me := sides[idx]
	r.Summary = append(r.Summary, ReportRow{Cells: []string{"Points", scoreText(me.Points)}},
		ReportRow{Cells: []string{"Best lineup", scoreText(me.Best) + " · " + scoreText(me.Best-me.Points) + " left on the bench"}})
	if g, found := findGame(w, v.team); found {
		them := g.Home
		if them.Franchise == v.team {
			them = g.Away
		}
		r.Summary = append(r.Summary, ReportRow{Cells: []string{"Opponent", v.name(them.Franchise) + " · " + scoreText(them.Points)}})
	}
	for _, p := range v.starters(me) {
		name, pos := v.player(p.ID)
		call := "In the best lineup"
		if !p.ShouldStart {
			call = "The best lineup started someone else"
		}
		r.Rows = append(r.Rows, v.row("", pos, name, scoreText(p.Points), call))
	}
	for _, p := range me.Players {
		if !p.Starter && p.ShouldStart {
			name, pos := v.player(p.ID)
			r.Rows = append(r.Rows, v.row("", pos, name, scoreText(p.Points), "Benched; the best lineup started him"))
		}
	}
	return r
}
