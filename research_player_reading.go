package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ResearchPlayerReading struct {
	ResearchPlayer
	Report   Report            `json:"report"`
	Articles []ResearchArticle `json:"articles"`
	HowLabel string            `json:"howLabel"`
}

func researchHeldTime(at time.Time, loss string, zone *time.Location) string {
	if at.IsZero() {
		return "not read yet"
	}
	text := "as of " + reportTime(at, zone)
	if loss != "" {
		text += " · " + loss
	}
	return text
}
func (v researchView) playerReading(id string, names map[string]string, how string) ResearchPlayerReading {
	p := v.player(id)
	if p.FranchiseID != "" {
		p.Owner = names[p.FranchiseID]
		if p.Owner == "" {
			p.Owner = "Franchise " + p.FranchiseID
		}
	}
	r := Report{ID: "M-062", Title: "Held player research", Status: "ok", Source: "Source: held MFL and nflverse reports",
		AsOf: "As of: each feed’s time below", How: "Depth slots match only through the held NFL identity crosswalk. Points per game uses finished weeks with a held score, including zero. Position rank uses this league’s season fantasy points for all players at the same MFL position. Missing facts are not predictions.",
		Columns: []string{"Reading", "Held fact", "Feed time"}, Rows: []ReportRow{}}
	r.Rows = append(r.Rows, ReportRow{Cells: []string{"Owner", p.Owner, "Held league roster"}})
	v.playerInjury(&r, id)
	v.playerDepth(&r, id)
	v.playerPoints(&r, p)
	news := v.held.news[id]
	articles := []ResearchArticle{}
	for _, n := range news.value {
		r.Rows = append(r.Rows, ReportRow{Cells: []string{"News · " + n.Published, n.Headline, researchHeldTime(news.fetchedAt, news.refreshError, v.zone)}})
		articles = append(articles, ResearchArticle{ID: id, Article: n.Article, Name: p.Name, Headline: n.Headline})
	}
	if len(news.value) == 0 {
		text := "No MFL news for this player"
		if news.fetchedAt.IsZero() {
			text = "not held · My players’ news reads rostered players"
		}
		r.Rows = append(r.Rows, ReportRow{Cells: []string{"News", text, researchHeldTime(news.fetchedAt, news.refreshError, v.zone)}})
	}
	return ResearchPlayerReading{ResearchPlayer: p, Report: r, Articles: articles, HowLabel: how}
}
func (v researchView) playerInjury(r *Report, id string) {
	text := "not held"
	if !v.injuries.fetchedAt.IsZero() {
		text = "Not on MFL’s injury report"
	}
	for _, i := range v.injuries.value {
		if i.ID.String() == id {
			body, back := i.Details, i.ExpReturn
			if body == "" {
				body = "body part not held"
			}
			if back == "" {
				back = "expected return not held"
			}
			text = i.Status + " · " + body + " · " + back
		}
	}
	r.Rows = append(r.Rows, ReportRow{Cells: []string{"Injury", text, researchHeldTime(v.injuries.fetchedAt, v.injuries.refreshError, v.zone)}})
}
func (v researchView) playerDepth(r *Report, id string) {
	found := false
	h := v.held.depth
	for _, d := range h.value {
		if depthID(d, v.held.links) != id {
			continue
		}
		found = true
		r.Rows = append(r.Rows, ReportRow{Cells: []string{"NFL depth", researchTeam(d.Team) + " · " + d.Group + " · " + d.Position + " · depth " + fmt.Sprint(d.Rank), researchHeldTime(h.fetchedAt, h.refreshError, v.zone)}})
	}
	if !found {
		r.Rows = append(r.Rows, ReportRow{Cells: []string{"NFL depth", "not held", researchHeldTime(h.fetchedAt, h.refreshError, v.zone)}})
	}
}
func (v researchView) playerPoints(r *Report, p ResearchPlayer) {
	h := v.feed("playerScores", "YTD")
	total, rank, held := 0.0, 1, false
	for _, e := range researchRows(h) {
		if e.ID.String() == p.ID {
			total, held = e.Value, true
		}
	}
	season := "not held"
	if held {
		for _, e := range researchRows(h) {
			if v.player(e.ID.String()).Position == p.Position && e.Value > total {
				rank++
			}
		}
		season = fmt.Sprintf("%.2f · position rank %d in this league’s scoring", total, rank)
	}
	r.Rows = append(r.Rows, ReportRow{Cells: []string{"Season fantasy points", season, researchHeldTime(h.fetchedAt, h.refreshError, v.zone)}})
	last := max(v.week-1, 0)
	week := v.feed("playerScores", strconv.Itoa(last))
	text := "not held"
	sum, games := 0.0, 0
	missing := false
	for w := 1; w <= last; w++ {
		found := false
		for _, e := range researchRows(v.feed("playerScores", strconv.Itoa(w))) {
			if e.ID.String() == p.ID {
				sum += e.Value
				games++
				found = true
				if w == last {
					text = fmt.Sprintf("%.2f", e.Value)
				}
			}
		}
		if !found {
			missing = true
		}
	}
	r.Rows = append(r.Rows, ReportRow{Cells: []string{"Last finished week · " + researchLastLabel(last), text, researchHeldTime(week.fetchedAt, week.refreshError, v.zone)}})
	ppg := "not held"
	if games > 0 {
		ppg = fmt.Sprintf("%.2f · %d finished weeks with scores", sum/float64(games), games)
	}
	if missing {
		ppg += " · other weekly scores not held"
	}
	r.Rows = append(r.Rows, ReportRow{Cells: []string{"Points per game", ppg, researchHeldTime(week.fetchedAt, week.refreshError, v.zone)}})
}

func researchNewsAge(wording string) (time.Duration, bool) {
	fields := strings.Fields(wording)
	if len(fields) != 2 {
		return 0, false
	}
	n, err := strconv.Atoi(fields[0])
	if err != nil || n < 0 || n > 365000 {
		return 0, false
	}
	unit := map[string]time.Duration{"minute": time.Minute, "minutes": time.Minute, "hour": time.Hour, "hours": time.Hour, "day": 24 * time.Hour, "days": 24 * time.Hour}[fields[1]]
	if unit == 0 {
		return 0, false
	}
	return time.Duration(n) * unit, true
}
func sortResearchNews(r *Report, articles []ResearchArticle) {
	order := make([]int, len(r.Rows))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, aok := researchNewsAge(r.Rows[order[i]].Cells[3])
		b, bok := researchNewsAge(r.Rows[order[j]].Cells[3])
		if aok != bok {
			return aok
		}
		return aok && a < b
	})
	rows := append([]ReportRow{}, r.Rows...)
	links := append([]ResearchArticle{}, articles...)
	for i, idx := range order {
		r.Rows[i], articles[i] = rows[idx], links[idx]
	}
}

func researchShell[T any](id, title, how string, feed heldSeasonFeed[[]T], zone *time.Location, columns ...string) Report {
	r := reportShell(id, title, how, feed, zone, columns...)
	if strings.Contains(strings.ToLower(feed.refreshError), "archived copy") && !feed.fetchedAt.IsZero() {
		r.Note = "Archived copy held; refresh pending."
	}
	return r
}

func (v researchView) depthMissNote(starters bool) string {
	line, special, other := 0, 0, 0
	for _, d := range v.held.depth.value {
		if starters && d.Rank != 1 || v.scope.Team != "" && researchTeam(d.Team) != v.scope.Team || depthID(d, v.held.links) != "" {
			continue
		}
		switch d.Position {
		case "LT", "LG", "C", "RG", "RT":
			line++
		case "P", "H", "LS":
			special++
		default:
			other++
		}
	}
	return fmt.Sprintf(" Of those slots: %d offensive line, %d punter/holder/long snapper, %d other. These are NFL depth slots, including repeated formations, not missing fantasy roster players; names remain visible but roster ownership cannot be joined without an identity link.", line, special, other)
}
