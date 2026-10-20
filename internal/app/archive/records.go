package archive

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
)

type archiveRecordEntry struct {
	label, who, when string
	value            float64
}

func appendArchiveExtremes(r *report.Report, entries []archiveRecordEntry) {
	groups := map[string][]archiveRecordEntry{}
	labels := []string{}
	for _, e := range entries {
		if _, ok := groups[e.label]; !ok {
			labels = append(labels, e.label)
		}
		groups[e.label] = append(groups[e.label], e)
	}
	for _, label := range labels {
		best := math.Inf(-1)
		for _, e := range groups[label] {
			best = max(best, e.value)
		}
		for _, e := range groups[label] {
			if e.value == best {
				r.Rows = append(r.Rows, report.Cells(e.label, e.who, e.when, archiveRecordValue(e)))
			}
		}
	}
	if len(r.Rows) == 0 && r.Status != "failed" {
		r.Status = "empty"
		r.Note = "No scored records are available in this snapshot."
	}
}

// archiveRecordValue writes counts (wins, losses, games in a streak) as whole numbers and points
// with two decimals.
func archiveRecordValue(e archiveRecordEntry) string {
	label := strings.ToLower(e.label)
	if strings.Contains(label, "wins") || strings.Contains(label, "losses") || strings.Contains(label, "streak") {
		return fmt.Sprintf("%.0f", e.value)
	}
	return report.ScoreText(report.Hundredths(e.value))
}

func (v archiveView) matchupRecords() report.Report {
	r := v.shell("M-083", "Matchup records", "Highest weekly franchise score, highest combined game score and largest winning margin. Opponents are paired by season and week. A game is counted once, on the winning side (or one side of a tie). Every tied record holder is shown.", "Record", "Who", "When", "Points")
	g := v.views.PlayedGame
	lookup := map[string]int{}
	for i, y := range g.Year {
		lookup[fmt.Sprintf("%d:%d:%s", y, g.Week[i], g.Fid[i])] = i
	}
	entries := []archiveRecordEntry{}
	for i, y := range g.Year {
		when := fmt.Sprintf("%d week %d", y, g.Week[i])
		entries = append(entries, archiveRecordEntry{"Highest weekly score", v.name(y, g.Fid[i]), when, *g.Score[i]})
		opp := archiveValue(g.Opp[i])
		j, ok := lookup[fmt.Sprintf("%d:%d:%s", y, g.Week[i], opp)]
		if !ok || opp == "" || *g.Score[i] < *g.Score[j] || (*g.Score[i] == *g.Score[j] && g.Fid[i] > opp) {
			continue
		}
		game := v.name(y, g.Fid[i]) + " vs " + v.name(y, opp)
		margin := v.name(y, g.Fid[i]) + " over " + v.name(y, opp)
		entries = append(entries, archiveRecordEntry{"Highest combined game", game, when, *g.Score[i] + *g.Score[j]}, archiveRecordEntry{"Largest winning margin", margin, when, *g.Score[i] - *g.Score[j]})
	}
	appendArchiveExtremes(&r, entries)
	return r
}

func (v archiveView) seasonRecords() report.Report {
	r := v.shell("M-084", "Single-season records", "Regular-season wins and losses use the reference season definition. Points include all weeks MFL scored. A partial season is labelled and is not extrapolated. Every tied holder is shown.", "Record", "Who", "When", "Value")
	ts := v.views.TeamSeason
	entries := []archiveRecordEntry{}
	for i, y := range ts.Year {
		who, when := v.name(y, ts.Fid[i]), archiveYear(y)
		if y == slices.Max(v.cache.Facts.Season.Year) {
			when += " · through cutoff"
		}
		if ts.RegW[i] != nil {
			entries = append(entries, archiveRecordEntry{"Most regular-season wins", who, when, float64(*ts.RegW[i])}, archiveRecordEntry{"Most regular-season losses", who, when, float64(archiveValue(ts.RegL[i]))})
		}
		if ts.AllWeeksPf[i] != nil {
			entries = append(entries, archiveRecordEntry{"Most season points · all weeks", who, when, *ts.AllWeeksPf[i]})
		}
	}
	appendArchiveExtremes(&r, entries)
	return r
}

func (v archiveView) careerRecords() report.Report {
	r := v.shell("M-085", "Career franchise records", "Franchise identity is MFL's id across seasons, not its changing name. Regular wins, losses and points are summed over recorded seasons. No unplayed games count. Every tied record holder is shown.", "Record", "Who", "When", "Value")
	type tally struct {
		wins, losses, points float64
		first, last          int64
	}
	tallies := map[string]tally{}
	ts := v.views.TeamSeason
	for i, y := range ts.Year {
		t := tallies[ts.Fid[i]]
		if t.first == 0 {
			t.first = y
		}
		t.last = y
		t.wins += float64(archiveValue(ts.RegW[i]))
		t.losses += float64(archiveValue(ts.RegL[i]))
		t.points += archiveValue(ts.AllWeeksPf[i])
		tallies[ts.Fid[i]] = t
	}
	ids := []string{}
	for id := range tallies {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	entries := []archiveRecordEntry{}
	for _, id := range ids {
		t := tallies[id]
		who, when := v.name(t.last, id), fmt.Sprintf("%d–%d · through cutoff", t.first, t.last)
		entries = append(entries, archiveRecordEntry{"Most regular-season wins", who, when, t.wins}, archiveRecordEntry{"Most regular-season losses", who, when, t.losses}, archiveRecordEntry{"Most career points · all weeks", who, when, t.points})
	}
	appendArchiveExtremes(&r, entries)
	return r
}

func (v archiveView) streaks() report.Report {
	r := v.shell("M-086", "Winning and losing streaks", "Consecutive scored head-to-head games, regular season and playoffs, across season boundaries. A tie ends both streaks; a bye or an unscored game does not extend or end one. Every tied longest streak is shown with its starting and ending game.", "Record", "Who", "When", "Games")
	g := v.views.PlayedGame
	order := make([]int, len(g.Year))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int {
		return cmp.Or(cmp.Compare(g.Year[a], g.Year[b]), cmp.Compare(g.Week[a], g.Week[b]), strings.Compare(g.Fid[a], g.Fid[b]))
	})
	type run struct {
		result, start string
		count         int
	}
	runs := map[string]run{}
	entries := []archiveRecordEntry{}
	for _, i := range order {
		if g.Opp[i] == nil {
			continue
		}
		id, result := g.Fid[i], archiveValue(g.Result[i])
		when := fmt.Sprintf("%d week %d", g.Year[i], g.Week[i])
		held := runs[id]
		if result != "W" && result != "L" {
			delete(runs, id)
			continue
		}
		if result != held.result {
			held = run{result: result, start: when}
		}
		held.count++
		runs[id] = held
		label := "Longest winning streak"
		if result == "L" {
			label = "Longest losing streak"
		}
		entries = append(entries, archiveRecordEntry{label, v.name(g.Year[i], id), held.start + " → " + when, float64(held.count)})
	}
	appendArchiveExtremes(&r, entries)
	return r
}

func (v archiveView) playerRecords() report.Report {
	r := v.shell("M-082", "Player scoring records", "MFL's recorded weekly player scores under this league's scoring, not NFL statistics or a rescoring formula. Season and career totals include every recorded scored week; missing scores are not zero. Every tied record holder is shown.", "Record", "Player", "When", "Points")
	p := v.cache.Facts.PlayerWeek
	entries := []archiveRecordEntry{}
	seasons := map[string]float64{}
	careers := map[string]float64{}
	years := map[string]int64{}
	ids := map[string]string{}
	for i, y := range p.Year {
		if p.Score[i] == nil {
			continue
		}
		pid := p.Pid[i]
		entries = append(entries, archiveRecordEntry{"Highest single game", v.player(pid), fmt.Sprintf("%d week %d", y, p.Week[i]), *p.Score[i]})
		key := archiveKey(y, pid)
		seasons[key] += *p.Score[i]
		careers[pid] += *p.Score[i]
		years[key] = y
		ids[key] = pid
	}
	keys := []string{}
	for key := range seasons {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		entries = append(entries, archiveRecordEntry{"Most season points", v.player(ids[key]), archiveYear(years[key]) + " · recorded weeks", seasons[key]})
	}
	keys = keys[:0]
	for pid := range careers {
		keys = append(keys, pid)
	}
	slices.Sort(keys)
	for _, pid := range keys {
		entries = append(entries, archiveRecordEntry{"Most career points", v.player(pid), "All archived seasons · through cutoff", careers[pid]})
	}
	appendArchiveExtremes(&r, entries)
	return r
}

func (v archiveView) scoring() report.Report {
	r := v.shell("M-073", "Scoring history", "Points are the sum of franchise weekly scores, all weeks MFL scored. A partial season is not extrapolated. Era boundaries follow a change in the recorded scoring rules; the reference's regular-season calendar changes separately from 12 to 13 weeks.", "Season", "Scoring era", "League points", "Scored franchise weeks", "Regular season")
	rules := v.cache.Facts.ScoringRule
	fingerprints := map[int64]string{}
	for i, y := range rules.Year {
		fingerprints[y] += strings.Join([]string{rules.Positions[i], rules.Event[i], archiveValue(rules.Range[i]), archiveValue(rules.Points[i])}, "|") + "\n"
	}
	g := v.views.PlayedGame
	era := ""
	last := ""
	totals := map[string]float64{}
	order := []string{}
	years := map[string][]string{}
	for _, y := range v.cache.Facts.Season.Year {
		if era == "" || fingerprints[y] != last {
			era = archiveYear(y) + " scoring rules"
			order = append(order, era)
		}
		last = fingerprints[y]
		total, count := 0.0, 0
		for i, gy := range g.Year {
			if gy == y {
				total += *g.Score[i]
				count++
			}
		}
		weeks := []int64{}
		for i, wy := range v.views.RegularWeek.Year {
			if wy == y {
				weeks = append(weeks, v.views.RegularWeek.Week[i])
			}
		}
		span := "Not recorded"
		if len(weeks) > 0 {
			span = fmt.Sprintf("Weeks %d–%d", slices.Min(weeks), slices.Max(weeks))
		}
		row := report.Cells(archiveYear(y), era, report.ScoreText(report.Hundredths(total)), fmt.Sprint(count), span)
		row.Group = era
		r.Rows = append(r.Rows, row)
		totals[era] += total
		years[era] = append(years[era], archiveYear(y))
	}
	for _, era := range order {
		r.Summary = append(r.Summary, report.Cells(strings.Join(years[era], ", ")+" · all scored weeks · total points", report.ScoreText(report.Hundredths(totals[era]))))
	}
	return r
}

func (v archiveView) playerChanges() report.Report {
	r := v.shell("M-079", "Player additions and position changes", "The selected season only. Additions are the reference's roster moves with an add role (free agents, waivers and roster loads). Position changes compare consecutive recorded seasonal player positions; a missing year is not assigned a date. Transaction dates are MFL's recorded dates.", "When", "Change", "Player", "Franchise / positions")
	moves := v.views.RosterMove
	for i, y := range moves.Year {
		if y == int64(v.season) && moves.Role[i] == "add" && moves.Pid[i] != nil {
			r.Rows = append(r.Rows, report.Cells(archiveValue(moves.Day[i]), "Player added", v.player(*moves.Pid[i]), v.name(y, archiveValue(moves.Fid[i]))))
		}
	}
	ps := v.cache.Facts.PlayerSeason
	type heldPosition struct {
		year     int64
		position string
	}
	prior := map[string]heldPosition{}
	order := make([]int, len(ps.Year))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int {
		return cmp.Or(cmp.Compare(ps.Year[a], ps.Year[b]), strings.Compare(ps.Pid[a], ps.Pid[b]))
	})
	for _, i := range order {
		pid, pos := ps.Pid[i], archiveValue(ps.Position[i])
		if pos == "" {
			continue
		}
		held, seen := prior[pid]
		if seen && held.position != pos && ps.Year[i] == int64(v.season) {
			r.Rows = append(r.Rows, report.Cells(fmt.Sprintf("%d → %d", held.year, ps.Year[i]), "Position changed", v.player(pid), held.position+" → "+pos))
		}
		prior[pid] = heldPosition{ps.Year[i], pos}
	}
	if len(r.Rows) == 0 && r.Status != "failed" {
		r.Status = "empty"
		r.Note = "No additions or position changes recorded this season."
	}
	return r
}

func (v archiveView) awards() report.Report {
	r := v.shell("M-081", "Awards", "Only awards recorded by MFL qualify; bracket championships appear in the Trophy case, not as invented awards.", "Award", "Recipient", "Season")
	if r.Status != "failed" {
		r.Status = "empty"
		r.Note = "Deferred to ring 4: the pinned archive has no awards report. This is not evidence that the league has no awards; MFL's awards page still needs an archived capture."
	}
	return r
}

func (v archiveView) oddities() report.Report {
	r := v.shell("M-089", "Recorded oddities", "The selected season only. This is not league lore. Only facts already flagged by the archived results are shown: scored ties and zero-point starters. A zero score does not say whether the player was injured, benched or simply scored zero.", "When", "Recorded oddity", "Who")
	g, l := v.views.PlayedGame, v.cache.Facts.Lineup
	groups := map[string][]string{}
	for i, y := range g.Year {
		if y == int64(v.season) && archiveValue(g.Result[i]) == "T" && g.Opp[i] != nil && g.Fid[i] < *g.Opp[i] {
			key := fmt.Sprintf("%d week %d · Scored tie", y, g.Week[i])
			groups[key] = append(groups[key], v.name(y, g.Fid[i])+" / "+v.name(y, *g.Opp[i])+" · "+archivePoints(g.Score[i]))
		}
	}
	for i, y := range l.Year {
		if y == int64(v.season) && l.Started[i] == 1 && l.Score[i] != nil && *l.Score[i] == 0 {
			key := fmt.Sprintf("%d week %d · Zero-point starter", y, l.Week[i])
			groups[key] = append(groups[key], v.name(y, l.Fid[i])+" · "+v.player(l.Pid[i]))
		}
	}
	keys := []string{}
	for key := range groups {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b string) int {
		var ay, aw, by, bw int
		_, _ = fmt.Sscanf(a, "%d week %d", &ay, &aw)
		_, _ = fmt.Sscanf(b, "%d week %d", &by, &bw)
		return cmp.Or(cmp.Compare(ay, by), cmp.Compare(aw, bw), strings.Compare(a, b))
	})
	for _, key := range keys {
		when, kind, _ := strings.Cut(key, " · ")
		if len(groups[key]) > 1 {
			kind = fmt.Sprintf("%s · %d", kind, len(groups[key]))
		}
		r.Rows = append(r.Rows, report.Cells(when, kind, oddityNames(groups[key])))
	}
	if r.Status != "failed" {
		r.Note = "These are recorded results, not explanations of why they happened."
	}
	return r
}

func oddityNames(names []string) string {
	groups := map[string][]string{}
	for _, name := range names {
		team, player, _ := strings.Cut(name, " · ")
		groups[team] = append(groups[team], player)
	}
	teams := []string{}
	for team := range groups {
		teams = append(teams, team)
	}
	slices.Sort(teams)
	lines := []string{}
	for _, team := range teams {
		slices.Sort(groups[team])
		lines = append(lines, team+": "+strings.Join(groups[team], "; "))
	}
	return strings.Join(lines, "\n")
}
