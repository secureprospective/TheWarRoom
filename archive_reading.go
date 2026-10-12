package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

type archiveView struct {
	cache       archiveCache
	views       facts.Views
	names       map[string]string
	players     map[string]string
	season      int
	team, other string
	stale       bool
}

func archiveValue[T any](p *T) (v T) {
	if p != nil {
		return *p
	}
	return v
}
func archiveYear(y int64) string { return strconv.FormatInt(y, 10) }
func archivePoints(p *float64) string {
	if p == nil {
		return "Not recorded"
	}
	return fmt.Sprintf("%.2f", *p)
}
func archiveRecord(w, l, t *int64) string {
	if w == nil {
		return "Not recorded"
	}
	return fmt.Sprintf("%d-%d-%d", *w, archiveValue(l), archiveValue(t))
}
func archiveKey(y int64, id string) string { return archiveYear(y) + ":" + id }

func buildArchive(cache archiveCache, stale bool, season int, team, other string) (ArchiveBoard, error) {
	v := archiveView{cache: cache, views: cache.facts.Views(), names: map[string]string{}, players: map[string]string{}, season: season, team: team, other: other, stale: stale}
	fs := cache.facts.FranchiseSeason
	latest := map[string]string{}
	for i, y := range fs.Year {
		name := archiveValue(fs.Name[i])
		if name == "" {
			name = "Unnamed franchise"
		}
		v.names[archiveKey(y, fs.Fid[i])] = name
		latest[fs.Fid[i]] = name
	}
	for i, pid := range cache.facts.Player.Pid {
		v.players[pid] = cache.facts.Player.Name[i]
	}
	b := ArchiveBoard{Season: season, Team: team, Other: other, Seasons: []RaceChoice{}, Teams: []RaceChoice{}, Board: ReportBoard{Title: "Pools / Archive", Loading: "Reading the league archive…", Refresh: "Reread the imported archive", HowLabel: "How this is counted", Reports: []Report{}}}
	for _, y := range cache.facts.Season.Year {
		b.Seasons = append(b.Seasons, RaceChoice{Value: archiveYear(y), Label: archiveYear(y)})
	}
	if season == 0 && len(b.Seasons) > 0 {
		v.season = int(slices.Max(cache.facts.Season.Year))
		b.Season = v.season
	}
	if err := v.validateChoices(latest); err != nil {
		return ArchiveBoard{}, err
	}
	for id, name := range latest {
		b.Teams = append(b.Teams, RaceChoice{Value: id, Label: name})
	}
	slices.SortFunc(b.Teams, func(a, b RaceChoice) int {
		return cmp.Or(strings.Compare(a.Label, b.Label), strings.Compare(a.Value, b.Value))
	})
	b.Board.Reports = append(b.Board.Reports, v.history(), v.champions(), v.eras(), v.trophies(), v.franchise("M-105", "All-time franchise record"), v.franchiseBests(), v.franchiseSummary(), v.series(), v.matchupRecords(), v.seasonRecords(), v.careerRecords(), v.streaks(), v.playerRecords(), v.scoring(), v.playerChanges(), v.awards(), v.oddities(), v.pools())
	return b, nil
}

func (v archiveView) validateChoices(latest map[string]string) error {
	if v.season != 0 && v.cache.sha != "" && !slices.Contains(v.cache.facts.Season.Year, int64(v.season)) {
		return fmt.Errorf("archive: season is not in the snapshot")
	}
	for _, id := range []string{v.team, v.other} {
		if id != "" && v.cache.sha != "" && latest[id] == "" {
			return fmt.Errorf("archive: franchise is not in the snapshot")
		}
	}
	return nil
}

func (v archiveView) shell(id, title, how string, columns ...string) Report {
	r := Report{ID: id, Title: title, Status: "ok", Source: "Source: imported MFL archive · snapshot cutoff " + v.cache.cutoff, How: how, Columns: columns, Rows: []ReportRow{}, Summary: []ReportRow{}}
	if v.cache.sha == "" {
		r.Status = "failed"
		r.Note = "The compiled league archive is not available. Import the league archive in Control Room › Data and sources."
		r.Source = "Source: no compiled league archive"
	} else if v.stale {
		r.Status = "stale"
		r.Note = "Showing the last verified snapshot; the newer import has not passed compilation. Check Control Room › Data and sources."
	}
	return r
}
func (v archiveView) name(y int64, id string) string {
	if n := v.names[archiveKey(y, id)]; n != "" {
		return n
	}
	return "Franchise name not recorded"
}
func (v archiveView) player(id string) string {
	if n := v.players[id]; n != "" {
		return n
	}
	return "Player name not recorded"
}
func (v archiveView) needTeam(r Report) (Report, bool) {
	if r.Status == "failed" {
		return r, false
	}
	if v.team == "" {
		r.Status = "empty"
		r.Note = "Choose a franchise to see this report."
		return r, false
	}
	return r, true
}

type archiveFinal struct {
	year                   int64
	name, title, win, lose string
	winner, loser          float64
}

func (v archiveView) finals() []archiveFinal {
	b, g := v.cache.facts.PlayoffBracket, v.cache.facts.PlayoffGame
	out := []archiveFinal{}
	for i, y := range b.Year {
		last := int64(0)
		for j, gy := range g.Year {
			if gy == y && g.BracketID[j] == b.BracketID[i] {
				last = max(last, archiveValue(g.Week[j]))
			}
		}
		indices := []int{}
		for j, gy := range g.Year {
			if gy == y && g.BracketID[j] == b.BracketID[i] && archiveValue(g.Week[j]) == last {
				indices = append(indices, j)
			}
		}
		if final, ok := v.scoredFinal(i, indices); ok {
			out = append(out, final)
		}
	}
	return out
}
func (v archiveView) scoredFinal(bracket int, indices []int) (archiveFinal, bool) {
	if len(indices) != 2 {
		return archiveFinal{}, false
	}
	b, g := v.cache.facts.PlayoffBracket, v.cache.facts.PlayoffGame
	a, z := indices[0], indices[1]
	if g.Points[a] == nil || g.Points[z] == nil || *g.Points[a] == *g.Points[z] || archiveValue(g.Fid[a]) == "" || archiveValue(g.Fid[z]) == "" {
		return archiveFinal{}, false
	}
	if *g.Points[a] < *g.Points[z] {
		a, z = z, a
	}
	return archiveFinal{b.Year[bracket], archiveValue(b.Name[bracket]), archiveValue(b.WinnerTitle[bracket]), *g.Fid[a], *g.Fid[z], *g.Points[a], *g.Points[z]}, true
}

func superBowl(name string) bool {
	return strings.Contains(strings.ReplaceAll(strings.ToLower(name), " ", ""), "superbowl")
}

func (v archiveView) history() Report {
	r := v.shell("M-080", "League history", "Champions and runners-up are the two scored sides of the final round of the league's Super Bowl, exactly as the reference season cards count them. Unplayed, tied or missing finals are not assigned a winner. Regular-season leaders have the best win percentage; tied leaders are all named, not resolved by an invented tiebreak.", "Season", "Champion", "Runner-up", "Regular-season leader", "Final")
	finals := v.finals()
	for _, y := range v.cache.facts.Season.Year {
		champion, runner, score := "Final not recorded", "Final not recorded", "Not recorded"
		if y == slices.Max(v.cache.facts.Season.Year) {
			champion, runner, score = "Season in progress", "Season in progress", "Not played yet"
		}
		for _, f := range finals {
			if f.year == y && superBowl(f.name) {
				champion, runner, score = v.name(y, f.win), v.name(y, f.lose), fmt.Sprintf("%.2f–%.2f", f.winner, f.loser)
			}
		}
		best := float64(-1)
		leaders := []string{}
		ts := v.views.TeamSeason
		for i, sy := range ts.Year {
			games := archiveValue(ts.RegW[i]) + archiveValue(ts.RegL[i]) + archiveValue(ts.RegT[i])
			if sy != y || games == 0 {
				continue
			}
			pct := float64(2*archiveValue(ts.RegW[i])+archiveValue(ts.RegT[i])) / float64(2*games)
			label := v.name(y, ts.Fid[i]) + " (" + archiveRecord(ts.RegW[i], ts.RegL[i], ts.RegT[i]) + ")"
			if pct > best {
				best = pct
				leaders = []string{label}
			} else if pct == best {
				leaders = append(leaders, label)
			}
		}
		leader := strings.Join(leaders, "\n")
		if leader == "" {
			leader = "No scored regular-season games"
		}
		r.Rows = append(r.Rows, draftRow(archiveYear(y), champion, runner, leader, score))
	}
	return r
}

func (v archiveView) champions() Report {
	r := v.shell("M-114", "League champions", "Each season's Super Bowl winner, from the final round of MFL's recorded playoff bracket. A season without a scored, untied final has no champion here.", "Season", "Champion", "Final")
	for _, f := range v.finals() {
		if superBowl(f.name) {
			r.Rows = append(r.Rows, draftRow(archiveYear(f.year), v.name(f.year, f.win), fmt.Sprintf("%.2f–%.2f over %s", f.winner, f.loser, v.name(f.year, f.lose))))
		}
	}
	if len(r.Rows) == 0 && r.Status == "ok" {
		r.Status, r.Note = "empty", "No scored championship final is recorded in this snapshot."
	}
	return r
}

func (v archiveView) eras() Report {
	r := v.shell("M-182", "League rules by season", "The reference season cards name regular-season weeks, salary cap, roster size, taxi squad, injured reserve and starting lineup. Rules are shown as recorded for each season; repeated setups are grouped. Stale bracket titles are kept as MFL recorded them, not used to change a season's date.", "Seasons", "Regular season", "Salary cap", "Roster size", "Taxi / injured reserve", "Starting lineup")
	s := v.cache.facts.Season
	groups := map[string]int{}
	for i, y := range s.Year {
		weeks := []int64{}
		for j, wy := range v.views.RegularWeek.Year {
			if wy == y {
				weeks = append(weeks, v.views.RegularWeek.Week[j])
			}
		}
		span := "Not recorded"
		if len(weeks) > 0 {
			span = fmt.Sprintf("Weeks %d–%d", slices.Min(weeks), slices.Max(weeks))
		}
		salaryCap := "Not recorded"
		if s.SalaryCap[i] != nil {
			salaryCap = fmt.Sprintf("$%.2fM", *s.SalaryCap[i])
		}
		cells := []string{span, salaryCap, fmt.Sprint(archiveValue(s.RosterSize[i])), archiveUnquote(archiveValue(s.TaxiSquad[i])) + " / " + archiveUnquote(archiveValue(s.InjuredReserve[i])), archiveLineup(archiveValue(s.Starters[i]))}
		key := strings.Join(cells, "\x00")
		if at, ok := groups[key]; ok {
			r.Rows[at].Cells[0] += ", " + archiveYear(y)
		} else {
			groups[key] = len(r.Rows)
			r.Rows = append(r.Rows, draftRow(append([]string{archiveYear(y)}, cells...)...))
		}
	}
	return r
}

// archiveUnquote reads a setting MFL recorded as a JSON string ("8") as its words (8).
func archiveUnquote(raw string) string {
	var text string
	if json.Unmarshal([]byte(raw), &text) == nil {
		return text
	}
	return raw
}

// archiveLineup writes MFL's starting-lineup setting in words: each position's limit in MFL's
// order, then the starter counts it records. Anything it cannot read is shown as "Not recorded".
func archiveLineup(raw string) string {
	var lineup struct {
		Count    string `json:"count"`
		IDP      string `json:"idp_starters"`
		Offense  string `json:"iop_starters"`
		Position []struct {
			Name  string `json:"name"`
			Limit string `json:"limit"`
		} `json:"position"`
	}
	if raw == "" || json.Unmarshal([]byte(raw), &lineup) != nil || len(lineup.Position) == 0 {
		return "Not recorded"
	}
	parts := []string{}
	for _, p := range lineup.Position {
		parts = append(parts, p.Name+" "+strings.ReplaceAll(p.Limit, "-", "–"))
	}
	text := strings.Join(parts, " · ")
	for _, c := range []struct{ label, value string }{{"starters", lineup.Count}, {"offensive", lineup.Offense}, {"defensive", lineup.IDP}} {
		if c.value != "" {
			text += " · " + strings.ReplaceAll(c.value, "-", "–") + " " + c.label
		}
	}
	return text
}

func (v archiveView) franchise(id, title string) Report {
	r, ok := v.needTeam(v.shell(id, title, "Regular records come from the reference's season reading. Playoff records count scored head-to-head games outside the regular weeks; byes and unscored placeholders never count. All-play includes every scored week, including playoff weeks, with MFL's published all-play taking precedence. Trophy winners require a scored, untied two-team final.", "Season", "Franchise", "Regular W-L-T", "Playoff W-L-T", "Points · all weeks", "All-play W-L-T"))
	if !ok {
		return r
	}
	ts := v.views.TeamSeason
	reg := map[[2]int64]bool{}
	for i, y := range v.views.RegularWeek.Year {
		reg[[2]int64{y, v.views.RegularWeek.Week[i]}] = true
	}
	var rw, rl, rt, pw, pl, pt int64
	var best float64
	bestWhen := "Not recorded"
	g := v.views.PlayedGame
	for i, y := range ts.Year {
		if ts.Fid[i] != v.team {
			continue
		}
		w, l, t := v.playoffRecord(y, reg)
		rw += archiveValue(ts.RegW[i])
		rl += archiveValue(ts.RegL[i])
		rt += archiveValue(ts.RegT[i])
		pw += w
		pl += l
		pt += t
		r.Rows = append(r.Rows, draftRow(archiveYear(y), v.name(y, v.team), archiveRecord(ts.RegW[i], ts.RegL[i], ts.RegT[i]), archiveRecord(&w, &l, &t), archivePoints(ts.AllWeeksPf[i]), archiveRecord(ts.AllPlayW[i], ts.AllPlayL[i], ts.AllPlayT[i])))
	}
	for i, y := range g.Year {
		if g.Fid[i] == v.team && (bestWhen == "Not recorded" || *g.Score[i] > best) {
			best = *g.Score[i]
			bestWhen = fmt.Sprintf("%s week %d", archiveYear(y), g.Week[i])
		}
	}
	r.Summary = append(r.Summary, draftRow("All-time regular record", archiveRecord(&rw, &rl, &rt)), draftRow("All-time playoff record", archiveRecord(&pw, &pl, &pt)), draftRow("Highest weekly score", fmt.Sprintf("%.2f · %s", best, bestWhen)))
	trophies := []string{}
	for _, f := range v.finals() {
		if f.win == v.team {
			trophies = append(trophies, fmt.Sprintf("%d · %s (%s) · %.2f–%.2f", f.year, f.name, f.title, f.winner, f.loser))
		}
	}
	if len(trophies) == 0 {
		trophies = append(trophies, "No scored bracket championship recorded")
	}
	r.Summary = append(r.Summary, draftRow("Trophy case", strings.Join(trophies, "\n")))
	return r
}

// teamFinals are the scored bracket finals the chosen franchise won.
func (v archiveView) teamFinals() []archiveFinal {
	out := []archiveFinal{}
	for _, f := range v.finals() {
		if f.win == v.team {
			out = append(out, f)
		}
	}
	return out
}

func (v archiveView) trophies() Report {
	r, ok := v.needTeam(v.shell("M-104", "Trophy case", "Every bracket final the franchise won, as MFL's playoff brackets record it. A final needs two scored sides and no tie; unplayed or tied finals award nothing.", "Season", "Title", "Final"))
	if !ok {
		return r
	}
	for _, f := range v.teamFinals() {
		r.Rows = append(r.Rows, draftRow(archiveYear(f.year), trophyTitle(f.name, f.title), fmt.Sprintf("%.2f–%.2f over %s", f.winner, f.loser, v.name(f.year, f.lose))))
	}
	if len(r.Rows) == 0 {
		r.Status, r.Note = "empty", v.name(slices.Max(v.cache.facts.Season.Year), v.team)+" has no recorded bracket championship."
	}
	return r
}

// franchiseBests are the chosen franchise's own records: its best and worst weeks and seasons.
func (v archiveView) franchiseBests() Report {
	r, ok := v.needTeam(v.shell("M-023", "Franchise records", "The franchise's own highs and lows across every recorded season: scored weeks (playoffs included) and regular seasons. Ties for a record are all shown.", "Record", "When", "Value"))
	if !ok {
		return r
	}
	weeks, seasons := []archiveRecordEntry{}, []archiveRecordEntry{}
	g := v.views.PlayedGame
	for i, y := range g.Year {
		if g.Fid[i] == v.team && g.Score[i] != nil {
			when := fmt.Sprintf("%d week %d", y, g.Week[i])
			weeks = append(weeks, archiveRecordEntry{"Highest weekly score", "", when, *g.Score[i]}, archiveRecordEntry{"Lowest weekly score", "", when, -*g.Score[i]})
		}
	}
	ts := v.views.TeamSeason
	for i, y := range ts.Year {
		if ts.Fid[i] != v.team || y == slices.Max(v.cache.facts.Season.Year) {
			continue
		}
		when := archiveYear(y) + " · " + archiveRecord(ts.RegW[i], ts.RegL[i], ts.RegT[i])
		seasons = append(seasons, archiveRecordEntry{"Most regular-season wins", "", when, float64(archiveValue(ts.RegW[i]))}, archiveRecordEntry{"Most regular-season losses", "", when, float64(archiveValue(ts.RegL[i]))})
		if ts.AllWeeksPf[i] != nil {
			seasons = append(seasons, archiveRecordEntry{"Most season points · all weeks", "", archiveYear(y), *ts.AllWeeksPf[i]}, archiveRecordEntry{"Fewest season points · all weeks", "", archiveYear(y), -*ts.AllWeeksPf[i]})
		}
	}
	appendArchiveExtremes(&r, append(weeks, seasons...))
	for i := range r.Rows {
		r.Rows[i].Cells = []string{r.Rows[i].Cells[0], r.Rows[i].Cells[2], strings.TrimPrefix(r.Rows[i].Cells[3], "-")}
	}
	return r
}

func (v archiveView) franchiseSummary() Report {
	r, ok := v.needTeam(v.shell("M-110", "Franchise summary", "One franchise at a glance: its seasons and names, all-time records and titles. The full season table is the All-time franchise record.", "Fact", "Value"))
	if !ok {
		return r
	}
	full := v.franchise("M-105", "")
	ts := v.views.TeamSeason
	names, first, last, seasons := []string{}, int64(0), int64(0), 0
	for i, y := range ts.Year {
		if ts.Fid[i] != v.team {
			continue
		}
		seasons++
		if first == 0 {
			first = y
		}
		last = y
		if name := v.name(y, v.team); !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	r.Rows = append(r.Rows, draftRow("Seasons", fmt.Sprintf("%d · %d–%d", seasons, first, last)), draftRow("Names", strings.Join(names, ", ")))
	r.Rows = append(r.Rows, full.Summary[:2]...)
	titles := []string{}
	for _, f := range v.teamFinals() {
		titles = append(titles, f.title)
	}
	if len(titles) == 0 {
		titles = append(titles, "None recorded")
	}
	r.Rows = append(r.Rows, draftRow("Titles", strings.Join(titles, " · ")))
	return r
}

func (v archiveView) playoffRecord(year int64, regular map[[2]int64]bool) (w, l, t int64) {
	g := v.views.PlayedGame
	for i, y := range g.Year {
		if y != year || g.Fid[i] != v.team || g.Opp[i] == nil || regular[[2]int64{y, g.Week[i]}] {
			continue
		}
		switch archiveValue(g.Result[i]) {
		case "W":
			w++
		case "L":
			l++
		case "T":
			t++
		}
	}
	return w, l, t
}

func (v archiveView) series() Report {
	r, ok := v.needTeam(v.shell("M-019", "All-time series", "Every scored meeting, in season and week order. The selected franchise is shown first. Unscored placeholders are not ties.", "Season", "Week", "Franchise", "Opponent", "Score", "Result"))
	if !ok {
		return r
	}
	if v.other == "" || v.other == v.team {
		r.Status = "empty"
		r.Note = "Choose two different franchises to see every meeting."
		return r
	}
	g := v.views.PlayedGame
	theirs := map[string]*float64{}
	for i, y := range g.Year {
		if g.Fid[i] == v.other {
			theirs[fmt.Sprintf("%d:%d", y, g.Week[i])] = g.Score[i]
		}
	}
	w, l, t := 0, 0, 0
	for i, y := range g.Year {
		if g.Fid[i] != v.team || archiveValue(g.Opp[i]) != v.other {
			continue
		}
		result := archiveValue(g.Result[i])
		switch result {
		case "W":
			w++
		case "L":
			l++
		case "T":
			t++
		}
		r.Rows = append(r.Rows, draftRow(archiveYear(y), fmt.Sprint(g.Week[i]), v.name(y, v.team), v.name(y, v.other), archivePoints(g.Score[i])+"–"+archivePoints(theirs[fmt.Sprintf("%d:%d", y, g.Week[i])]), result))
	}
	r.Summary = append(r.Summary, draftRow("Series W-L-T", fmt.Sprintf("%d-%d-%d", w, l, t)))
	if len(r.Rows) == 0 {
		r.Status = "empty"
		r.Note = "No scored meetings recorded between these franchises."
	}
	return r
}

// trophyTitle names a trophy once: "NFC Conference Champ", not "NFC · NFC Conference Champ".
func trophyTitle(name, title string) string {
	switch {
	case name == "" || strings.Contains(strings.ToLower(title), strings.ToLower(name)):
		return title
	case title == "" || strings.Contains(strings.ToLower(name), strings.ToLower(title)):
		return name
	}
	return name + " · " + title
}
