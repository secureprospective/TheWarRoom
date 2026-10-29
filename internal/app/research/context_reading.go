package research

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

func (v researchView) injuryReport(id string, team bool) report.Report {
	title := "NFL injuries"
	if team {
		title = "Injuries by NFL team"
	}
	r := researchShell(id, title, "MFL’s NFL injury report: its status, body part and expected return are preserved as supplied, not medical predictions. The game view includes only NFL teams in this week’s held schedule; byes are excluded.", v.Injuries, v.Zone, "Player", "Pos", "NFL", "Status", "Body part", "Expected return")
	r.Source = "Source: MFL injuries"
	playing, ok := v.injuryGames(&r, team)
	if !ok {
		return r
	}
	for _, injury := range v.Injuries.Value {
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
func DepthID(r researchfeeds.RawDepth, links map[string]string) string {
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
	id := DepthID(r, v.Held.Links)
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
func (v researchView) depthReport() report.Report {
	r := researchShell("M-048", "NFL depth charts", "Only the newest nflverse snapshot is shown. Formation groups follow the source’s formation order, then slot and depth rank. ▸ marks your players. Identity joins use gsis, then ESPN through the held crosswalk; an unmatched player stays visible by nflverse’s name and is labelled. The grid shows rank-one players at each position, including different formations.", v.Held.Depth, v.Zone, "Player", "Position", "Depth")
	r.Source = "Source: nflverse depth charts · latest snapshot"
	if len(v.Held.Depth.Value) == 0 {
		r.Note = "NFL starters are not available yet; refresh from sources."
		return r
	}
	if v.Held.LinkError != "" {
		r.Note += " Player links are unavailable; NFL names are still shown."
	}
	if len(v.Held.Depth.Value) > 0 {
		r.Summary = append(r.Summary, report.Row{Cells: []string{"Snapshot", depthTime(v.Held.Depth.Value[0].DT, v.Zone)}})
	}
	if v.scope.Depth == "grid" {
		return v.depthGrid(r)
	}
	for _, d := range v.Held.Depth.Value {
		if v.scope.Team != "" && Team(d.Team) != v.scope.Team {
			continue
		}
		name, mine, _ := v.depthName(d)
		r.Rows = append(r.Rows, report.Row{Cells: []string{name, d.Position, fmt.Sprint(d.Rank)}, Mine: mine, Group: Team(d.Team) + " · " + d.Group})
	}
	if len(r.Rows) == 0 {
		r.Status, r.Note = "empty", "No NFL starters match these choices."
	} else {
		r.Note += v.depthMissNote(false)
	}
	return r
}
func (v researchView) depthGrid(r report.Report) report.Report {
	cells := map[string]map[string][]string{}
	positions := map[string]bool{}
	for _, d := range v.Held.Depth.Value {
		if d.Rank != 1 || v.scope.Team != "" && Team(d.Team) != v.scope.Team {
			continue
		}
		name, mine, _ := v.depthName(d)
		if mine {
			name = "▸ " + name
		}
		team := Team(d.Team)
		if cells[team] == nil {
			cells[team] = map[string][]string{}
		}
		positions[d.Position] = true
		names := cells[team][d.Position]
		if !slices.Contains(names, name) {
			cells[team][d.Position] = append(names, name)
		}
	}
	if len(cells) == 0 {
		r.Status, r.Note = "empty", "No NFL starters match these choices."
		return r
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
		row := report.Row{Cells: []string{team}, Titles: []string{team}}
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
	r.Note += v.depthMissNote(true)
	return r
}
func (v researchView) newsReport(mine bool) (report.Report, []Article) {
	id, title := "M-058", "Rostered players’ news"
	if mine {
		id, title = "M-060", "My players’ news"
	}
	r := report.Report{ID: id, Title: title, Source: "Source: MFL player news · rostered players only", AsOf: "As of: not fetched", Status: "ok", Rows: []report.Row{}, Columns: []string{"Player", "Age", "Headline", "Posted"}, How: "Profiles are fetched only for rostered players, one second apart, and cached per local calendar day. Age is MFL’s bio age. Open article hands off to MFL in your browser; nothing is submitted. " + NewsHow}
	articles := []Article{}
	if mine && v.scope.Mine == "" {
		r.Status = "empty"
		r.Note = "Choose your franchise to see its news."
		return r, articles
	}
	ids := v.newsIDs(mine)
	health := v.newsHealth(ids)
	for _, player := range ids {
		v.appendNews(&r, &articles, player, v.Held.News[player].Value)
	}
	if !health.oldest.IsZero() {
		r.AsOf = "held · fetched " + report.Time(health.oldest, v.Zone)
	}
	if health.losses > 0 {
		at := health.oldest
		if health.loss == "" {
			at = time.Time{}
		}
		researchLoss(&r, fmt.Sprintf("Player news · %d profiles need refreshing", health.losses), at, health.loss, v.Zone)
	}
	sortResearchNews(&r, articles)
	r.Note = fmt.Sprintf("%d of %d rostered profiles held today or from the last fetch. Posted times are estimates.", health.fetched, len(ids))
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
		_, position, _ := v.Lookup.DisplayFacts(player)
		if (!mine || v.mine[player]) && !researchfeeds.TeamUnit(position) {
			ids = append(ids, player)
		}
	}
	sort.Strings(ids)
	return ids
}

func (v researchView) injuryGames(r *report.Report, team bool) (map[string]bool, bool) {
	playing := map[string]bool{}
	if !team && v.scope.Injuries == "games" {
		schedule := v.Schedules[v.Week]
		researchLoss(r, "This week’s NFL schedule", schedule.FetchedAt, schedule.RefreshError, v.Zone)
		r.Note += fmt.Sprintf(" Week %d games.", v.Week)
		for _, w := range schedule.Value {
			for _, m := range w.Matchups {
				for _, t := range m.Teams {
					playing[Team(t)] = true
				}
			}
		}
		if schedule.FetchedAt.IsZero() {
			r.Status = "failed"
			r.Note += " Game injuries unavailable without this week’s schedule."
			return playing, false
		}
	}
	return playing, true
}

func (v researchView) injuryRow(p PlayerInfo, injury leaguefeed.Injury, team bool) report.Row {
	body, back := injury.Details, injury.ExpReturn
	if body == "" {
		body = "Not supplied"
	}
	if back == "" {
		back = "Not supplied"
	}
	row := report.Row{Cells: []string{p.Name, p.Position, p.Team, injury.Status, body, back}, Mine: v.mine[p.ID]}
	if team {
		row.Group = p.Team
	}
	return row
}

func (v researchView) appendNews(r *report.Report, articles *[]Article, player string, rows []researchfeeds.News) {
	name := v.player(player).Name
	for _, news := range rows {
		age := news.Age
		posted, order := newsPosted(news.Published, v.Held.News[player].FetchedAt, v.Zone)
		if age == "" {
			age = "Not supplied"
		}
		r.Rows = append(r.Rows, report.Row{Cells: []string{name, age, news.Headline, posted}, Mine: v.mine[player], Sort: []float64{order}})
		*articles = append(*articles, Article{ID: player, Article: news.Article, Name: name, Headline: news.Headline})
	}
}

func depthTime(raw string, zone *time.Location) string {
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return "Snapshot time unavailable"
	}
	return report.Time(at, zone)
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
