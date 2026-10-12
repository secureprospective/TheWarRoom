package main

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

type archiveRecordEntry struct {
	label, who, when string
	value            float64
}

func appendArchiveExtremes(r *Report, entries []archiveRecordEntry) {
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
				r.Rows = append(r.Rows, draftRow(e.label, e.who, e.when, archiveRecordValue(e)))
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
	return fmt.Sprintf("%.2f", e.value)
}

func (v archiveView) matchupRecords() Report {
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

func (v archiveView) seasonRecords() Report {
	r := v.shell("M-084", "Single-season records", "Regular-season wins and losses use the reference season definition. Points include all weeks MFL scored. A partial season is labelled and is not extrapolated. Every tied holder is shown.", "Record", "Who", "When", "Value")
	ts := v.views.TeamSeason
	entries := []archiveRecordEntry{}
	for i, y := range ts.Year {
		who, when := v.name(y, ts.Fid[i]), archiveYear(y)
		if y == slices.Max(v.cache.facts.Season.Year) {
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

func (v archiveView) careerRecords() Report {
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

func (v archiveView) streaks() Report {
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

func (v archiveView) playerRecords() Report {
	r := v.shell("M-082", "Player scoring records", "MFL's recorded weekly player scores under this league's scoring, not NFL statistics or a rescoring formula. Season and career totals include every recorded scored week; missing scores are not zero. Every tied record holder is shown.", "Record", "Player", "When", "Points")
	p := v.cache.facts.PlayerWeek
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

func (v archiveView) scoring() Report {
	r := v.shell("M-073", "Scoring history", "Points are the sum of franchise weekly scores, all weeks MFL scored. A partial season is not extrapolated. Era boundaries follow a change in the recorded scoring rules; the reference's regular-season calendar changes separately from 12 to 13 weeks.", "Season", "Scoring era", "League points", "Scored franchise weeks", "Regular season")
	rules := v.cache.facts.ScoringRule
	fingerprints := map[int64]string{}
	for i, y := range rules.Year {
		fingerprints[y] += strings.Join([]string{rules.Positions[i], rules.Event[i], archiveValue(rules.Range[i]), archiveValue(rules.Points[i])}, "|") + "\n"
	}
	g := v.views.PlayedGame
	era := ""
	last := ""
	totals := map[string]float64{}
	order := []string{}
	for _, y := range v.cache.facts.Season.Year {
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
		row := draftRow(archiveYear(y), era, fmt.Sprintf("%.2f", total), fmt.Sprint(count), span)
		row.Group = era
		r.Rows = append(r.Rows, row)
		totals[era] += total
	}
	for _, era := range order {
		r.Summary = append(r.Summary, draftRow(era+" · total points", fmt.Sprintf("%.2f", totals[era])))
	}
	return r
}

func (v archiveView) playerChanges() Report {
	r := v.shell("M-079", "Player additions and position changes", "The selected season only. Additions are the reference's roster moves with an add role (free agents, waivers and roster loads). Position changes compare consecutive recorded seasonal player positions; a missing year is not assigned a date. Transaction dates are MFL's recorded dates.", "When", "Change", "Player", "Franchise / positions")
	moves := v.views.RosterMove
	for i, y := range moves.Year {
		if y == int64(v.season) && moves.Role[i] == "add" && moves.Pid[i] != nil {
			r.Rows = append(r.Rows, draftRow(archiveValue(moves.Day[i]), "Player added", v.player(*moves.Pid[i]), v.name(y, archiveValue(moves.Fid[i]))))
		}
	}
	ps := v.cache.facts.PlayerSeason
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
			r.Rows = append(r.Rows, draftRow(fmt.Sprintf("%d → %d", held.year, ps.Year[i]), "Position changed", v.player(pid), held.position+" → "+pos))
		}
		prior[pid] = heldPosition{ps.Year[i], pos}
	}
	if len(r.Rows) == 0 && r.Status != "failed" {
		r.Status = "empty"
		r.Note = "No additions or position changes recorded this season."
	}
	return r
}

func (v archiveView) awards() Report {
	r := v.shell("M-081", "Awards", "Only awards recorded by MFL qualify; bracket championships appear in the Trophy case, not as invented awards.", "Award", "Recipient", "Season")
	if r.Status != "failed" {
		r.Status = "empty"
		r.Note = "Deferred to ring 4: the pinned archive has no awards report. This is not evidence that the league has no awards; MFL's awards page still needs an archived capture."
	}
	return r
}

func (v archiveView) oddities() Report {
	r := v.shell("M-089", "Recorded oddities", "The selected season only. This is not league lore. Only facts already flagged by the archived results are shown: scored ties and zero-point starters. A zero score does not say whether the player was injured, benched or simply scored zero.", "When", "Recorded oddity", "Who")
	g, l := v.views.PlayedGame, v.cache.facts.Lineup
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
			groups[key] = append(groups[key], v.player(l.Pid[i])+" · "+v.name(y, l.Fid[i]))
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
		r.Rows = append(r.Rows, draftRow(when, kind, strings.Join(groups[key], "; ")))
	}
	if r.Status != "failed" {
		r.Note = "League lore and explanations are deferred to ring 4. These are recorded oddities only."
	}
	return r
}

func (v archiveView) pools() Report {
	r := v.shell("M-111", "Pools", "Picks are MFL's recorded NFL pick'em entries for the selected season. Confidence weights are preserved. Standings are counted only for NFL games with recorded final scores; equal NFL scores are ties, not wins. Survivor setup is reported separately. This app makes no picks.", "Franchise", "Week", "Matchup", "Pick", "Confidence", "Result")
	if r.Status == "failed" {
		return r
	}
	pool, ok := v.cache.pools[int64(v.season)]
	if !ok {
		r.Status = "empty"
		r.Note = "The selected season's pool exports are not pinned in this snapshot. Import the league archive in Control Room › Data and sources."
		return r
	}
	if len(pool.Picks) == 0 {
		r.Status = "empty"
		r.Note = "No pools are running this season."
		return r
	}
	wins, losses, pending, points := map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}
	for _, p := range pool.Picks {
		result := v.poolResult(p)
		switch result {
		case "Won":
			wins[p.Franchise]++
			points[p.Franchise] += int(p.Rank)
		case "Lost":
			losses[p.Franchise]++
		}
		if result == "Not scored in the archive" {
			pending[p.Franchise]++
		}
		r.Rows = append(r.Rows, draftRow(v.name(int64(v.season), p.Franchise), fmt.Sprint(p.Week), strings.ReplaceAll(p.Matchup, ",", " vs "), p.Pick, fmt.Sprint(p.Rank), result))
	}
	ids := []string{}
	for _, p := range pool.Picks {
		if !slices.Contains(ids, p.Franchise) {
			ids = append(ids, p.Franchise)
		}
	}
	slices.SortFunc(ids, func(a, b string) int { return cmp.Or(cmp.Compare(points[b], points[a]), strings.Compare(a, b)) })
	for i, id := range ids {
		r.Summary = append(r.Summary, draftRow(fmt.Sprintf("Standing %d · %s", i+1, v.name(int64(v.season), id)), fmt.Sprintf("%d confidence points · %d won, %d lost, %d awaiting scores", points[id], wins[id], losses[id], pending[id])))
	}
	r.Note = "Only recorded picks count; standings are derived from held NFL final scores, not an MFL pool standings export. Survivor: " + pool.Survivor
	return r
}

func (v archiveView) poolResult(p facts.PoolPick) string {
	nfl := v.cache.facts.NflGame
	for i, y := range nfl.Year {
		if y != int64(v.season) || nfl.Week[i] != p.Week || nfl.Team[i] != p.Pick || nfl.Score[i] == nil {
			continue
		}
		for j, oy := range nfl.Year {
			if oy != y || nfl.Week[j] != p.Week || nfl.Team[j] != archiveValue(nfl.Opp[i]) || nfl.Score[j] == nil {
				continue
			}
			switch {
			case *nfl.Score[i] > *nfl.Score[j]:
				return "Won"
			case *nfl.Score[i] < *nfl.Score[j]:
				return "Lost"
			default:
				return "Tied"
			}
		}
	}
	return "Not scored in the archive"
}
