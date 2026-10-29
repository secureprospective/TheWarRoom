package season

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/app/report"

	"github.com/secureprospective/TheWarRoom/internal/engine/race"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/pickvalue"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/powerrankings"
)

// shareBar draws value on a shared axis from zero to max, so bars compare down a column.
func shareBar(value, scale float64, text string) *report.Range {
	value = math.Max(value, 0)
	return &report.Range{Interval: report.Interval{Value: value, Lo: 0, Hi: value}, Max: math.Max(scale, value), Text: text}
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
			"moves compare with the board from the previous model run. Picks from the next two drafts add to the counted players at weight %.2g (%s), before the common scale; they do not change roster age. An unreadable pick total is blank and does not change the score.", aggText(res), res.ModelRunID, res.AgeWeight, res.PickWeight, pickvalue.Version)
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

func (v View) Power(view string) report.Report {
	id, title, columns := "M-008", "Power rankings: this season",
		[]string{"Rank", "Franchise", "Move", "Record", "Roster value", "Results", "Blend"}
	if view == ViewFranchise {
		id, title, columns = "C-02", "Power rankings: the franchise",
			[]string{"Rank", "Franchise", "Move", "Roster value", "Picks", "Roster age", "Blend"}
	}
	r := shell(id, title, "", v.in.Held.Standings, v.in.Zone, columns...)
	r.Source = "Source: MFL standings and the app's model"
	res, held := v.in.Held.Power[view]
	switch {
	case r.Status == "failed":
		return r
	case !held || !res.OK:
		r.Status, r.Note = "failed", "Power rankings are unavailable. Try refreshing again."
		return r
	case len(res.Rows) == 0:
		r.Status, r.Note = "empty", "No model run is held for this season; run the app's model first."
		return r
	}
	if res.Error != "" {
		r.Status, r.Note = "stale", "Power rankings could not be updated. Try refreshing again."
	}
	r.How = powerHow(res)
	for _, p := range res.Rows {
		value, picks, age := powerValues(p)
		cells := []string{strconv.Itoa(p.Rank), v.Name(p.FranchiseID), moveText(p)}
		if view == ViewFranchise {
			cells = append(cells, value, picks, age)
		} else {
			result := strings.TrimPrefix(fmt.Sprintf("%.3f", p.Results), "0")
			if res.Performance == m2service.PerfPointsFor {
				result = fmt.Sprintf("%.0f%% of the best", p.Results*100)
			}
			cells = append(cells, recordText(race.Record{W: p.H2HW, L: p.H2HL, T: p.H2HT}), value, result)
		}
		blend := fmt.Sprintf("%.2f", p.PowerScore)
		row := v.Row(p.FranchiseID, append(cells, blend)...)
		row.Range = shareBar(p.PowerScore, 1, "Blend "+blend+" of 1")
		if p.ValueReason != "" {
			row.Titles = make([]string, len(row.Cells))
			for i := range row.Titles {
				row.Titles[i] = p.ValueReason
			}
		}
		r.Rows = append(r.Rows, row)
	}
	return r
}

func powerValues(p PowerRow) (value, picks, age string) {
	value = fmt.Sprintf("%.1f", p.RosterValue)
	if p.Unavailable {
		value = "Unavailable"
	} else if p.ReducedInputs {
		value += " · reduced inputs"
	}
	age = "—"
	if p.Age > 0 {
		age = fmt.Sprintf("%.1f", p.Age)
	}
	picks = ""
	if p.HasPicks {
		picks = fmt.Sprintf("%.1f", p.PickValue)
		if p.ReducedInputs {
			picks += " · reduced inputs"
		}
	}
	if !p.HasPicks && p.ValueReason != "" {
		picks = "Unavailable"
	}

	return value, picks, age
}

func (v View) AllPlay() report.Report {
	r := v.WeeksReport("M-161", "All-play",
		"Each week, every franchise's score is compared with every other franchise's score that week: higher is "+
			"an all-play win, lower a loss, equal a tie. Every week MFL scored counts, playoff weeks included, as "+
			"MFL's own all-play does. Wins against all-play pace are actual regular-season wins minus the wins "+
			"the all-play percentage gives over the same games. Every bar runs from 0 to 1.",
		"Franchise", "All-play record", "Record", "Wins against all-play pace", "All-play %")
	ids := v.Table.IDs()
	slices.SortStableFunc(ids, func(a, b string) int {
		la, _ := v.Table.Line(a)
		lb, _ := v.Table.Line(b)
		return cmp.Or(-cmpPct(la.AllPlay, lb.AllPlay), cmp.Compare(lb.For, la.For))
	})
	for _, id := range ids {
		l, _ := v.Table.Line(id)
		if l.AllPlay.Games() == 0 {
			continue
		}
		share := float64(2*l.AllPlay.W+l.AllPlay.T) / float64(2*l.AllPlay.Games())
		pace := float64(l.Overall.W) + float64(l.Overall.T)/2 - share*float64(l.Overall.Games())
		row := v.Row(id, v.Name(id), recordText(l.AllPlay), recordText(l.Overall), fmt.Sprintf("%+.1f", pace), pctText(l.AllPlay))
		row.Range = shareBar(share, 1, "All-play "+pctText(l.AllPlay))
		r.Rows = append(r.Rows, row)
	}
	return r
}

func cmpPct(a, b race.Record) int {
	return cmp.Compare(float64(2*a.W+a.T)/float64(2*max(a.Games(), 1)), float64(2*b.W+b.T)/float64(2*max(b.Games(), 1)))
}

// scoredSides is every franchise week MFL scored, in week order.
func (v View) ScoredSides(team string) []struct {
	week int
	side leaguefeed.Side
} {
	out := []struct {
		week int
		side leaguefeed.Side
	}{}
	for _, w := range v.Weeks {
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

func (v View) WeeklyPoints() report.Report {
	r := v.WeeksReport("M-071", "Weekly points",
		"Each franchise's score in every week MFL scored, most points first. All bars share one scale, from zero "+
			"to the highest score any franchise posted in a week, so they compare down the column.",
		"Franchise", "Points for", "Per week", "Points by week")
	scored := []int{}
	for _, w := range v.Weeks {
		if WeekScored(w) {
			scored = append(scored, w.Week)
			r.BarLabels = append(r.BarLabels, "Week "+strconv.Itoa(w.Week))
		}
	}
	byTeam := map[string]map[int]leaguefeed.Points{}
	for _, s := range v.ScoredSides("") {
		if byTeam[s.side.Franchise] == nil {
			byTeam[s.side.Franchise] = map[int]leaguefeed.Points{}
		}
		byTeam[s.side.Franchise][s.week] = s.side.Points
		r.BarMax = math.Max(r.BarMax, float64(s.side.Points)/100)
	}
	ids := v.Table.IDs()
	slices.SortStableFunc(ids, func(a, b string) int {
		la, _ := v.Table.Line(a)
		lb, _ := v.Table.Line(b)
		return cmp.Compare(lb.For, la.For)
	})
	for _, id := range ids {
		l, _ := v.Table.Line(id)
		weeks := byTeam[id]
		if len(weeks) == 0 {
			continue
		}
		row := v.Row(id, v.Name(id), scoreText(l.For), scoreText(l.For/leaguefeed.Points(len(weeks))), "")
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
func (v View) Player(id playerid.PlayerID) (name, position string) {
	if f, ok := v.in.Lookup.Facts(id.String()); ok {
		return f.Name, string(f.Position)
	}
	if n, ok := v.in.Lookup.Name(id.String()); ok && n != "" {
		return n, "Team unit"
	}
	return "A player MFL's directory does not list", "Position unknown"
}

func (v View) DirectoryNote(r *report.Report) {
	if !v.in.HasLookup && r.Status != "failed" {
		r.Note = strings.TrimSpace(r.Note + " Player names are not loaded yet; refresh from MFL.")
	}
}

// positions is the league's starter positions in its own order, then any other position seen.
func (v View) Positions(seen map[string]bool) []string {
	out := []string{}
	for _, p := range v.in.Config.Starters.Positions {
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

func (v View) PositionPoints() report.Report {
	totals, seen := map[string]map[string]leaguefeed.Points{}, map[string]bool{}
	for _, s := range v.ScoredSides("") {
		for _, p := range s.side.Players {
			if !p.Starter {
				continue
			}
			_, pos := v.Player(p.ID)
			if totals[s.side.Franchise] == nil {
				totals[s.side.Franchise] = map[string]leaguefeed.Points{}
			}
			totals[s.side.Franchise][pos] += p.Points
			seen[pos] = true
		}
	}
	positions := v.Positions(seen)
	r := v.WeeksReport("M-076", "Starter points by position",
		"Points scored by each franchise's starters, by position, in every week MFL scored. Bench points do not "+
			"count, and neither do the home-field points MFL adds in playoff games, so a playoff host's total sits "+
			"just under its points for. Most starter points first.", append([]string{"Franchise", "Starter points"}, positions...)...)
	v.DirectoryNote(&r)
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
		row := v.Row(id, v.Name(id), scoreText(sum(id)))
		for _, pos := range positions {
			row.Cells = append(row.Cells, scoreText(totals[id][pos]))
		}
		r.Rows = append(r.Rows, row)
	}
	return r
}

func (v View) PlayerPoints() report.Report {
	r, ok := v.NeedTeam(v.WeeksReport("M-077", "Starter points by player",
		"Every player the franchise started in a week MFL scored, with his points in those starts. Bench weeks do "+
			"not count. Most points first.", "Player", "Position", "Starts", "Points as starter", "Per start"))
	if !ok {
		return r
	}
	v.DirectoryNote(&r)
	type tally struct {
		id     playerid.PlayerID
		starts int
		points leaguefeed.Points
	}
	byID := map[string]*tally{}
	for _, s := range v.ScoredSides(v.Team) {
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
		name, pos := v.Player(t.id)
		r.Rows = append(r.Rows, v.Row("", name, pos, strconv.Itoa(t.starts), scoreText(t.points),
			scoreText(t.points/leaguefeed.Points(t.starts))))
	}
	return r
}

// weekSides is the chosen week's franchise sides.
func (v View) WeekSides() (leaguefeed.WeekResult, []leaguefeed.Side) {
	for _, w := range v.Weeks {
		if w.Week == v.Week {
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
func (v View) Starters(s leaguefeed.Side) []leaguefeed.WeekPlayer {
	order := v.Positions(nil)
	out := []leaguefeed.WeekPlayer{}
	for _, p := range s.Players {
		if p.Starter {
			out = append(out, p)
		}
	}
	rank := func(p leaguefeed.WeekPlayer) int {
		_, pos := v.Player(p.ID)
		if i := slices.Index(order, pos); i >= 0 {
			return i
		}
		return len(order)
	}
	slices.SortStableFunc(out, func(a, b leaguefeed.WeekPlayer) int {
		an, _ := v.Player(a.ID)
		bn, _ := v.Player(b.ID)
		return cmp.Or(cmp.Compare(rank(a), rank(b)), strings.Compare(an, bn),
			strings.Compare(a.ID.String(), b.ID.String()))
	})
	return out
}

func (v View) Lineups() report.Report {
	r := v.WeeksReport("M-009", "Starting lineups",
		"Every franchise's starters in the chosen week, by position, with each starter's points. Most points first.",
		"Franchise", "Points", "Starters")
	v.DirectoryNote(&r)
	if r.Status != "ok" && r.Status != "stale" {
		return r
	}
	_, sides := v.WeekSides()
	r.Summary = append(r.Summary, report.Row{Cells: []string{"Week", "Week " + strconv.Itoa(v.Week)}})
	slices.SortStableFunc(sides, func(a, b leaguefeed.Side) int { return cmp.Compare(b.Points, a.Points) })
	for _, s := range sides {
		lines := []string{}
		for _, p := range v.Starters(s) {
			name, pos := v.Player(p.ID)
			lines = append(lines, pos+" · "+name+" · "+scoreText(p.Points))
		}
		if len(lines) == 0 {
			lines = append(lines, "No lineup listed")
		}
		r.Rows = append(r.Rows, v.Row(s.Franchise, v.Name(s.Franchise), scoreText(s.Points), strings.Join(lines, "\n")))
	}
	return r
}

func (v View) LineupDetail() report.Report {
	r, ok := v.NeedTeam(v.WeeksReport("M-010", "Lineup detail",
		"The franchise's starters in the chosen week with each one's points, then any bench player MFL's best "+
			"lineup would have started. Best lineup is the most the same roster could have scored under the "+
			"league's starter rules, as MFL counts it.", "Position", "Player", "Points", "Lineup call"))
	if !ok {
		return r
	}
	v.DirectoryNote(&r)
	w, sides := v.WeekSides()
	idx := slices.IndexFunc(sides, func(s leaguefeed.Side) bool { return s.Franchise == v.Team })
	r.Summary = append(r.Summary, report.Row{Cells: []string{"Week", "Week " + strconv.Itoa(v.Week)}})
	if idx < 0 {
		r.Status, r.Note = "empty", strings.TrimSpace(r.Note+" MFL lists no lineup for this franchise in this week.")
		return r
	}
	me := sides[idx]
	r.Summary = append(r.Summary, report.Row{Cells: []string{"Points", scoreText(me.Points)}},
		report.Row{Cells: []string{"Best lineup", scoreText(me.Best) + " · " + scoreText(me.Best-me.Points) + " left on the bench"}})
	if g, found := FindGame(w, v.Team); found {
		them := g.Home
		if them.Franchise == v.Team {
			them = g.Away
		}
		r.Summary = append(r.Summary, report.Row{Cells: []string{"Opponent", v.Name(them.Franchise) + " · " + scoreText(them.Points)}})
	}
	for _, p := range v.Starters(me) {
		name, pos := v.Player(p.ID)
		call := "In the best lineup"
		if !p.ShouldStart {
			call = "The best lineup started someone else"
		}
		r.Rows = append(r.Rows, v.Row("", pos, name, scoreText(p.Points), call))
	}
	for _, p := range me.Players {
		if !p.Starter && p.ShouldStart {
			name, pos := v.Player(p.ID)
			r.Rows = append(r.Rows, v.Row("", pos, name, scoreText(p.Points), "Benched; the best lineup started him"))
		}
	}
	return r
}
