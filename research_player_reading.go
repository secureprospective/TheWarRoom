package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

const researchNewsHow = "Posted times are estimated from MFL’s elapsed wording at the fetch time, in your local time zone. Minutes show a minute, hours show about an hour, and days show only a date. Unrecognized wording or a missing fetch time has no posted time."

type ResearchPlayerReading struct {
	ResearchPlayer
	Report   Report            `json:"report"`
	Articles []ResearchArticle `json:"articles"`
	HowLabel string            `json:"howLabel"`
}

func researchHeldTime(at time.Time, loss string, zone *time.Location) string {
	if at.IsZero() {
		if loss != "" {
			return loss
		}
		return "No held copy available"
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
		AsOf: "As of: each feed’s time below", How: "Depth slots match only through the held NFL identity crosswalk. Points per game uses finished weeks with a held score, including zero. Season points sum the same held finished-week scores used by the card; an unfinished week is excluded. Missing facts are not predictions. " + researchNewsHow,
		Columns: []string{"Reading", "Held fact", "Feed time"}, Rows: []ReportRow{}}
	r.Rows = append(r.Rows, ReportRow{Cells: []string{"Owner", p.Owner, "Held league roster"}})
	v.playerInjury(&r, id)
	v.playerDepth(&r, id)
	v.playerPoints(&r, p)
	v.playerForm(&r, p)
	v.playerCutCharge(&r, id)
	news := v.newsHeld(id)
	articles := []ResearchArticle{}
	newsReport := Report{Rows: []ReportRow{}}
	v.appendNews(&newsReport, &articles, id, news.value)
	sortResearchNews(&newsReport, articles)
	for _, row := range newsReport.Rows {
		r.Rows = append(r.Rows, ReportRow{Cells: []string{"News · " + row.Cells[3], row.Cells[2], newsHeldTime(news.fetchedAt, news.refreshError, v.zone)}})
	}
	if len(news.value) == 0 {
		text := "No MFL news for this player"
		if news.fetchedAt.IsZero() {
			text = "not held · My players’ news reads rostered players"
		}
		r.Rows = append(r.Rows, ReportRow{Cells: []string{"News", text, newsHeldTime(news.fetchedAt, news.refreshError, v.zone)}})
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
	last := max(v.week-1, 0)
	week := v.feed("playerScores", strconv.Itoa(last))
	text := "not held"
	rows, _, latest := v.finishedScores()
	sum, games, rank := 0.0, 0, 1
	if row := rows[p.ID]; row != nil {
		sum, games = row.sum, row.games
	}
	for _, row := range rows {
		if row.position == p.Position && row.sum > sum {
			rank++
		}
	}
	for _, e := range researchRows(week) {
		if e.ID.String() == p.ID {
			text = fmt.Sprintf("%.2f", e.Value)
		}
	}
	season := "not held · no finished weekly score for this player"
	if games > 0 {
		season = fmt.Sprintf("%.2f · %d finished weeks with scores · position rank %d", sum, games, rank)
	}
	r.Rows = append(r.Rows, ReportRow{Cells: []string{"Season fantasy points", season, researchHeldTime(latest, week.refreshError, v.zone)}})
	r.Rows = append(r.Rows, ReportRow{Cells: []string{"Last finished week · " + researchLastLabel(last), text, researchHeldTime(week.fetchedAt, week.refreshError, v.zone)}})
	ppg := "not held"
	if games > 0 {
		ppg = fmt.Sprintf("%.2f · %d finished weeks with scores", sum/float64(games), games)
	}
	if games < last {
		ppg += " · other weekly scores not held"
	}
	r.Rows = append(r.Rows, ReportRow{Cells: []string{"Points per game", ppg, researchHeldTime(latest, week.refreshError, v.zone)}})
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

type researchNewsHealth struct {
	fetched, losses int
	oldest          time.Time
	loss            string
}

func (v researchView) newsHeld(id string) heldSeasonFeed[[]researchfeeds.News] {
	h := v.held.news[id]
	now := v.now
	if now.IsZero() {
		now = time.Now()
	}
	if !h.fetchedAt.IsZero() && h.refreshError == "" &&
		h.fetchedAt.In(v.zone).Format("2006-01-02") != now.In(v.zone).Format("2006-01-02") {
		h.refreshError = "Archived copy; refresh pending"
	}
	return h
}

func (v researchView) newsHealth(ids []string) researchNewsHealth {
	health := researchNewsHealth{}
	for _, id := range ids {
		h := v.newsHeld(id)
		if h.fetchedAt.IsZero() || h.refreshError != "" {
			health.losses++
			if h.refreshError != "" {
				health.loss = h.refreshError
			}
		}
		if h.fetchedAt.IsZero() {
			continue
		}
		health.fetched++
		if health.oldest.IsZero() || h.fetchedAt.Before(health.oldest) {
			health.oldest = h.fetchedAt
		}
	}
	return health
}

func newsHeldTime(at time.Time, loss string, zone *time.Location) string {
	if at.IsZero() { // a finished read with no copy says why, like every other held fact
		return researchHeldTime(at, loss, zone)
	}
	text := "held · fetched " + reportTime(at, zone)
	if loss != "" {
		text += " · " + loss
	}
	return text
}

func newsPosted(wording string, fetched time.Time, zone *time.Location) (string, float64) {
	if strings.TrimSpace(wording) == "" {
		return "Posted time not supplied", 0
	}
	age, ok := researchNewsAge(wording)
	if !ok {
		return "Posted time unknown: MFL wording not recognized", 0
	}
	if fetched.IsZero() {
		return "Posted time unknown: fetch time not held", 0
	}
	at := fetched.Add(-age)
	text := reportTime(at, zone)
	switch strings.Fields(wording)[1] {
	case "hour", "hours":
		text = at.In(zone).Format("Jan 2, 2006, about 3 PM")
	case "day", "days":
		text = at.In(zone).Format("Jan 2, 2006")
	}
	return text, float64(at.Unix())
}

func sortResearchNews(r *Report, articles []ResearchArticle) {
	order := make([]int, len(r.Rows))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := r.Rows[order[i]], r.Rows[order[j]]
		return a.Sort[0] > b.Sort[0]
	})
	rows := append([]ReportRow{}, r.Rows...)
	links := append([]ResearchArticle{}, articles...)
	for i, idx := range order {
		r.Rows[i], articles[i] = rows[idx], links[idx]
		r.Rows[i].Sort = nil
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

func (v researchView) playerCutCharge(r *Report, id string) {
	text := "not held · no roster contract for this player"
	source := "No held copy available"
	for _, p := range v.mirror.Players {
		if p.MFLID != id {
			continue
		}
		source = "MFL roster contract"
		charge, available := heldCutCharge(p, v.mirror.Season)
		if !available {
			text = "Cannot price a cut: contract expiration is not held"
			continue
		}
		text = snapshot.ContractAmountOf(charge).Text +
			" if unclaimed · future salaries held flat; restructure history not held"
	}
	r.Rows = append(r.Rows, ReportRow{Cells: []string{"Dead cap if cut today", text, source}})
}

func (v researchView) playerForm(r *Report, p ResearchPlayer) {
	rows, _, latest := v.finishedScores()
	season := rows[p.ID]
	text := "not held · no finished weekly scores for this player"
	if season != nil && season.games > 0 {
		weeks, total := []string{}, 0.0
		for w := max(v.week-3, 1); w < v.week; w++ {
			for _, score := range researchRows(v.feed("playerScores", strconv.Itoa(w))) {
				if score.ID.String() == p.ID {
					weeks = append(weeks, fmt.Sprintf("W%d: %.2f", w, score.Value))
					total += score.Value
				}
			}
		}
		if len(weeks) > 0 {
			text = fmt.Sprintf("%s · %.2f PPG over %d of the last 3 finished weeks; season %.2f PPG",
				strings.Join(weeks, ", "), total/float64(len(weeks)), len(weeks), season.sum/float64(season.games))
		} else {
			text = "None of the last 3 finished weekly scores held for this player"
		}
	}
	r.Rows = append(r.Rows, ReportRow{Cells: []string{"Form", text, researchHeldTime(latest, "", v.zone)}})
}
