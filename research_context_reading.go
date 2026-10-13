package main

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

func (v researchView) injuryReport(id string, team bool) Report {
	title := "NFL injuries"
	if team {
		title = "Injuries by NFL team"
	}
	r := researchShell(id, title, "MFL’s NFL injury report: its status, body part and expected return are preserved as supplied, not medical predictions. The game view includes only NFL teams in this week’s held schedule; byes are excluded.", v.injuries, v.zone, "Player", "Pos", "NFL", "Status", "Body part", "Expected return")
	r.Source = "Source: MFL injuries"
	playing, ok := v.injuryGames(&r, team)
	if !ok {
		return r
	}
	for _, injury := range v.injuries.value {
		p := v.player(injury.ID.String())
		if v.scope.Team != "" && p.Team != v.scope.Team || v.scope.Position != "" && p.Position != v.scope.Position {
			continue
		}
		if !team && v.scope.Injuries == "games" && !playing[p.Team] {
			continue
		}
		r.Rows = append(r.Rows, v.injuryRow(p, injury, team))
	}
	sort.SliceStable(r.Rows, func(i, j int) bool {
		a, b := r.Rows[i], r.Rows[j]
		if a.Group != b.Group {
			return a.Group < b.Group
		}
		return a.Cells[0] < b.Cells[0]
	})
	researchEmpty(&r)
	return r
}
func depthID(r researchfeeds.RawDepth, links map[string]string) string {
	if r.GSIS != nil {
		if id := links["gsis:"+*r.GSIS]; id != "" {
			return id
		}
	}
	if r.ESPN != nil {
		return links["espn:"+*r.ESPN]
	}
	return ""
}
func (v researchView) depthName(r researchfeeds.RawDepth) (string, bool, bool) {
	id := depthID(r, v.held.links)
	name := r.Name
	if name == "" && id != "" {
		name = v.player(id).Name
	}
	if name == "" {
		name = "Unnamed depth slot"
	}
	if id == "" {
		return name + " · MFL identity not linked", false, false
	}
	mine := v.mine[id]

	return name, mine, true
}
func (v researchView) depthReport() Report {
	r := researchShell("M-048", "NFL depth charts", "Only the newest nflverse snapshot is shown. Formation groups follow the source’s formation order, then slot and depth rank. ▸ marks your players. Identity joins use gsis, then ESPN through the held crosswalk; an unmatched player stays visible by nflverse’s name and is labelled. The grid shows rank-one players at each position, including different formations.", v.held.depth, v.zone, "Player", "Position", "Depth")
	r.Source = "Source: nflverse depth charts · latest snapshot"
	if v.held.linkError != "" {
		r.Note += " Crosswalk unavailable: " + v.held.linkError
	}
	if len(v.held.depth.value) > 0 {
		r.Summary = append(r.Summary, ReportRow{Cells: []string{"Snapshot", depthTime(v.held.depth.value[0].DT, v.zone)}})
	}
	if v.scope.Depth == "grid" {
		return v.depthGrid(r)
	}
	misses := 0
	for _, d := range v.held.depth.value {
		if v.scope.Team != "" && researchTeam(d.Team) != v.scope.Team {
			continue
		}
		name, mine, matched := v.depthName(d)
		if !matched {
			misses++
		}
		r.Rows = append(r.Rows, ReportRow{Cells: []string{name, d.Position, fmt.Sprint(d.Rank)}, Mine: mine, Group: researchTeam(d.Team) + " · " + d.Group})
	}
	r.Note += fmt.Sprintf(" %d displayed depth slots have no held MFL identity link (not a roster status).", misses)
	r.Note += v.depthMissNote(false)
	researchEmpty(&r)
	return r
}
func (v researchView) depthGrid(r Report) Report {
	cells := map[string]map[string][]string{}
	positions := map[string]bool{}
	misses := 0
	for _, d := range v.held.depth.value {
		if d.Rank != 1 || v.scope.Team != "" && researchTeam(d.Team) != v.scope.Team {
			continue
		}
		name, mine, matched := v.depthName(d)
		if mine {
			name = "▸ " + name
		}
		if !matched {
			misses++
		}
		team := researchTeam(d.Team)
		if cells[team] == nil {
			cells[team] = map[string][]string{}
		}
		positions[d.Position] = true
		names := cells[team][d.Position]
		if !slices.Contains(names, name) {
			cells[team][d.Position] = append(names, name)
		}
	}
	cols := []string{}
	for p := range positions {
		cols = append(cols, p)
	}
	sort.Strings(cols)
	r.Columns = append([]string{"NFL team"}, cols...)
	teams := []string{}
	for t := range cells {
		teams = append(teams, t)
	}
	sort.Strings(teams)
	for _, team := range teams {
		row := ReportRow{Cells: []string{team}, Titles: []string{team}}
		for _, p := range cols {
			names := cells[team][p]
			text := "—"
			if len(names) > 0 {
				text = strings.Join(names, " · ")
			}
			row.Cells = append(row.Cells, shortDepthNames(names))
			row.Titles = append(row.Titles, text)
		}
		r.Rows = append(r.Rows, row)
	}
	r.Note += fmt.Sprintf(" %d starter slots have no held MFL identity link (*). ▸ marks your players.", misses) + v.depthMissNote(true)
	researchEmpty(&r)
	return r
}
func (v researchView) newsReport(mine bool) (Report, []ResearchArticle) {
	id, title := "M-058", "Rostered players’ news"
	if mine {
		id, title = "M-060", "My players’ news"
	}
	r := Report{ID: id, Title: title, Source: "Source: MFL player news · rostered players only", AsOf: "As of: not fetched", Status: "ok", Rows: []ReportRow{}, Columns: []string{"Player", "Age", "Headline", "Published"}, How: "Profiles are fetched only for rostered players, one second apart, and cached per local calendar day. Published is MFL’s literal relative wording, not a reconstructed timestamp. Age is MFL’s bio age. Open article hands off to MFL in your browser; nothing is submitted."}
	articles := []ResearchArticle{}
	if mine && v.scope.Mine == "" {
		r.Status = "empty"
		r.Note = "Choose your franchise to see its news."
		return r, articles
	}
	ids := v.newsIDs(mine)
	health := v.newsHealth(ids)
	for _, player := range ids {
		v.appendNews(&r, &articles, player, v.held.news[player].value)
	}
	if !health.oldest.IsZero() {
		r.AsOf = "held · fetched " + reportTime(health.oldest, v.zone)
	}
	if health.losses > 0 {
		at := health.oldest
		if health.loss == "" {
			at = time.Time{}
		}
		researchLoss(&r, fmt.Sprintf("Player news · %d profiles need refreshing", health.losses), at, health.loss, v.zone)
	}
	sortResearchNews(&r, articles)
	r.Note = fmt.Sprintf("%d of %d rostered profiles held today or from the last fetch. Published is MFL’s wording.", health.fetched, len(ids))
	if health.fetched == 0 && len(ids) > 0 {
		r.Status = "failed"
	} else {
		researchEmpty(&r)
	}
	return r, articles
}

func (v researchView) newsIDs(mine bool) []string {
	ids := []string{}
	for player := range v.owners {
		_, position, _ := v.lookup.DisplayFacts(player)
		if (!mine || v.mine[player]) && !researchfeeds.TeamUnit(position) {
			ids = append(ids, player)
		}
	}
	sort.Strings(ids)
	return ids
}

func (v researchView) injuryGames(r *Report, team bool) (map[string]bool, bool) {
	playing := map[string]bool{}
	if !team && v.scope.Injuries == "games" {
		schedule := v.schedules[v.week]
		researchLoss(r, "This week’s NFL schedule", schedule.fetchedAt, schedule.refreshError, v.zone)
		r.Note += fmt.Sprintf(" Week %d games.", v.week)
		for _, w := range schedule.value {
			for _, m := range w.Matchups {
				for _, t := range m.Teams {
					playing[researchTeam(t)] = true
				}
			}
		}
		if schedule.fetchedAt.IsZero() {
			r.Status = "failed"
			r.Note += " Game injuries unavailable without this week’s schedule."
			return playing, false
		}
	}
	return playing, true
}

func (v researchView) injuryRow(p ResearchPlayer, injury leaguefeed.Injury, team bool) ReportRow {
	body, back := injury.Details, injury.ExpReturn
	if body == "" {
		body = "Not supplied"
	}
	if back == "" {
		back = "Not supplied"
	}
	row := ReportRow{Cells: []string{p.Name, p.Position, p.Team, injury.Status, body, back}, Mine: v.mine[p.ID]}
	if team {
		row.Group = p.Team
	}
	return row
}

func (v researchView) appendNews(r *Report, articles *[]ResearchArticle, player string, rows []researchfeeds.News) {
	name := v.player(player).Name
	for _, news := range rows {
		age, published := news.Age, news.Published
		if age == "" {
			age = "Not supplied"
		}
		if published == "" {
			published = "Not supplied"
		}
		r.Rows = append(r.Rows, ReportRow{Cells: []string{name, age, news.Headline, published}, Mine: v.mine[player], Sort: []float64{newsPublishedAt(published, v.held.news[player].fetchedAt)}})
		*articles = append(*articles, ResearchArticle{ID: player, Article: news.Article, Name: name, Headline: news.Headline})
	}
}

func depthTime(raw string, zone *time.Location) string {
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return "Snapshot time unavailable"
	}
	return reportTime(at, zone)
}

func shortDepthNames(names []string) string {
	if len(names) == 0 {
		return "—"
	}
	short := make([]string, 0, len(names))
	for _, name := range names {
		marker := ""
		if strings.HasPrefix(name, "▸ ") {
			marker = "▸ "
			name = strings.TrimPrefix(name, "▸ ")
		}
		unmatched := strings.HasSuffix(name, " · MFL identity not linked")
		name = strings.TrimSuffix(name, " · MFL identity not linked")
		words := strings.Fields(name)
		if len(words) > 1 {
			name = string([]rune(words[0])[0]) + ". " + strings.Join(words[1:], " ")
		}
		if unmatched {
			name += "*"
		}
		short = append(short, marker+name)
	}
	return strings.Join(short, " · ")
}
