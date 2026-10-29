package research

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

const NewsHow = "Posted times are estimated from MFL’s elapsed wording at the fetch time, in your local time zone. Minutes show a minute, hours show about an hour, and days show only a date. Unrecognized wording or a missing fetch time has no posted time."

type PlayerReading struct {
	PlayerInfo
	Report   report.Report `json:"report"`
	Articles []Article     `json:"articles"`
	HowLabel string        `json:"howLabel"`
}

func HeldTime(at time.Time, loss string, zone *time.Location) string {
	if at.IsZero() {
		return "No held copy available"
	}
	return "as of " + report.Time(at, zone) + refreshQualifier(loss)
}
func (v researchView) playerReading(id string, names map[string]string, how string) PlayerReading {
	p := v.player(id)
	if p.FranchiseID != "" {
		p.Owner = names[p.FranchiseID]
		if p.Owner == "" {
			p.Owner = "Franchise " + p.FranchiseID
		}
	}
	r := report.Report{ID: "M-062", Title: "Held player research", Status: "ok", Source: "Source: held MFL and nflverse reports",
		AsOf: "As of: each feed’s time below", How: "Depth slots match only through the held NFL identity crosswalk. Points per game uses finished weeks with a held score, including zero. Season points sum the same held finished-week scores used by the card; an unfinished week is excluded. Missing facts are not predictions. " + NewsHow,
		Columns: []string{"Reading", "Held fact", "Feed time"}, Rows: []report.Row{}}
	r.Rows = append(r.Rows, report.Row{Cells: []string{"Owner", p.Owner, "Held league roster"}})
	v.playerInjury(&r, id)
	v.playerDepth(&r, id)
	v.playerPoints(&r, p)
	v.playerForm(&r, p)
	v.playerCutCharge(&r, id)
	news := v.newsHeld(id)
	articles := []Article{}
	newsReport := report.Report{Rows: []report.Row{}}
	v.appendNews(&newsReport, &articles, id, news.Value)
	sortResearchNews(&newsReport, articles)
	for _, row := range newsReport.Rows {
		r.Rows = append(r.Rows, report.Row{Cells: []string{"News · " + row.Cells[3], row.Cells[2], newsHeldTime(news.FetchedAt, news.RefreshError, v.Zone)}})
	}
	if len(news.Value) == 0 {
		text := "No MFL news for this player"
		if news.FetchedAt.IsZero() {
			text = "not held · My players’ news reads rostered players"
		}
		r.Rows = append(r.Rows, report.Row{Cells: []string{"News", text, newsHeldTime(news.FetchedAt, news.RefreshError, v.Zone)}})
	}
	return PlayerReading{PlayerInfo: p, Report: r, Articles: articles, HowLabel: how}
}
func (v researchView) playerInjury(r *report.Report, id string) {
	text := "not held"
	if !v.Injuries.FetchedAt.IsZero() {
		text = "Not on MFL’s injury report"
	}
	for _, i := range v.Injuries.Value {
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
	r.Rows = append(r.Rows, report.Row{Cells: []string{"Injury", text, HeldTime(v.Injuries.FetchedAt, v.Injuries.RefreshError, v.Zone)}})
}
func (v researchView) playerDepth(r *report.Report, id string) {
	found := false
	h := v.Held.Depth
	for _, d := range h.Value {
		if DepthID(d, v.Held.Links) != id {
			continue
		}
		found = true
		r.Rows = append(r.Rows, report.Row{Cells: []string{"NFL depth", Team(d.Team) + " · " + d.Group + " · " + d.Position + " · depth " + fmt.Sprint(d.Rank), HeldTime(h.FetchedAt, h.RefreshError, v.Zone)}})
	}
	if !found {
		r.Rows = append(r.Rows, report.Row{Cells: []string{"NFL depth", "not held", HeldTime(h.FetchedAt, h.RefreshError, v.Zone)}})
	}
}
func (v researchView) playerPoints(r *report.Report, p PlayerInfo) {
	last := max(v.Week-1, 0)
	week := v.feed("playerScores", strconv.Itoa(last))
	text := "not held"
	rows, latest := v.Finished, v.Latest
	sum, games, rank := 0.0, 0, 1
	if row := rows[p.ID]; row != nil {
		sum, games = row.Sum, row.Games
	}
	for _, row := range rows {
		if row.Position == p.Position && row.Sum > sum {
			rank++
		}
	}
	for _, e := range Rows(week) {
		if e.ID.String() == p.ID {
			text = fmt.Sprintf("%.2f", e.Value)
		}
	}
	season := "not held · no finished weekly score for this player"
	if games > 0 {
		season = fmt.Sprintf("%.2f · %d finished weeks with scores · position rank %d", sum, games, rank)
	}
	r.Rows = append(r.Rows, report.Row{Cells: []string{"Season fantasy points", season, HeldTime(latest, week.RefreshError, v.Zone)}})
	r.Rows = append(r.Rows, report.Row{Cells: []string{"Last finished week · " + researchLastLabel(last), text, HeldTime(week.FetchedAt, week.RefreshError, v.Zone)}})
	ppg := "not held"
	if games > 0 {
		ppg = fmt.Sprintf("%.2f · %d finished weeks with scores", sum/float64(games), games)
	}
	if games < last {
		ppg += " · other weekly scores not held"
	}
	r.Rows = append(r.Rows, report.Row{Cells: []string{"Points per game", ppg, HeldTime(latest, week.RefreshError, v.Zone)}})
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

func (v researchView) newsHeld(id string) Feed[[]researchfeeds.News] {
	h := v.Held.News[id]
	now := v.Now
	if !h.FetchedAt.IsZero() && h.RefreshError == "" &&
		h.FetchedAt.In(v.Zone).Format("2006-01-02") != now.In(v.Zone).Format("2006-01-02") {
		h.RefreshError = "Archived copy; refresh pending"
	}
	return h
}

func (v researchView) newsHealth(ids []string) researchNewsHealth {
	health := researchNewsHealth{}
	for _, id := range ids {
		h := v.newsHeld(id)
		if h.FetchedAt.IsZero() || h.RefreshError != "" {
			health.losses++
			if h.RefreshError != "" {
				health.loss = h.RefreshError
			}
		}
		if h.FetchedAt.IsZero() {
			continue
		}
		health.fetched++
		if health.oldest.IsZero() || h.FetchedAt.Before(health.oldest) {
			health.oldest = h.FetchedAt
		}
	}
	return health
}

func newsHeldTime(at time.Time, loss string, zone *time.Location) string {
	if at.IsZero() { // a finished read with no copy says why, like every other held fact
		return HeldTime(at, loss, zone)
	}
	return "held · fetched " + report.Time(at, zone) + refreshQualifier(loss)
}

func refreshQualifier(loss string) string {
	if loss == "" {
		return ""
	}
	if strings.Contains(strings.ToLower(loss), "archived copy") {
		return " · archived copy; refresh pending"
	}
	return " · refresh needed"
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
	text := report.Time(at, zone)
	switch strings.Fields(wording)[1] {
	case "hour", "hours":
		text = at.In(zone).Format("Jan 2, 2006, about 3 PM")
	case "day", "days":
		text = at.In(zone).Format("Jan 2, 2006")
	}
	return text, float64(at.Unix())
}

func sortResearchNews(r *report.Report, articles []Article) {
	order := make([]int, len(r.Rows))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := r.Rows[order[i]], r.Rows[order[j]]
		return a.Sort[0] > b.Sort[0]
	})
	rows := append([]report.Row{}, r.Rows...)
	links := append([]Article{}, articles...)
	for i, idx := range order {
		r.Rows[i], articles[i] = rows[idx], links[idx]
		r.Rows[i].Sort = nil
	}
}

func researchShell[T any](id, title, how string, feed Feed[[]T], zone *time.Location, columns ...string) report.Report {
	r := report.Shell(id, title, how, report.Feed{FetchedAt: feed.FetchedAt, RefreshError: feed.RefreshError, ReadPending: feed.ReadPending}, len(feed.Value), zone, columns...)
	if strings.Contains(strings.ToLower(feed.RefreshError), "archived copy") && !feed.FetchedAt.IsZero() {
		r.Note = "Archived copy held; refresh pending."
	}
	return r
}

func (v researchView) depthMissNote(starters bool) string {
	line, special, other := 0, 0, 0
	for _, d := range v.Held.Depth.Value {
		if starters && d.Rank != 1 || v.scope.Team != "" && Team(d.Team) != v.scope.Team || DepthID(d, v.Held.Links) != "" {
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

func (v researchView) playerCutCharge(r *report.Report, id string) {
	text := "not held · no roster contract for this player"
	source := "No held copy available"
	for _, p := range v.Mirror.Players {
		if p.MFLID != id {
			continue
		}
		source = "MFL roster contract"
		charge, available := v.CutCharges[p.MFLID]
		if !available {
			text = "Cannot price a cut: contract expiration is not held"
			continue
		}
		text = snapshot.ContractAmountOf(charge).Text +
			" if unclaimed · future salaries held flat; restructure history not held"
	}
	r.Rows = append(r.Rows, report.Row{Cells: []string{"Dead cap if cut today", text, source}})
}

func (v researchView) playerForm(r *report.Report, p PlayerInfo) {
	rows, latest := v.Finished, v.Latest
	season := rows[p.ID]
	text := "not held · no finished weekly scores for this player"
	if season != nil && season.Games > 0 {
		weeks, total := []string{}, 0.0
		for w := max(v.Week-3, 1); w < v.Week; w++ {
			for _, score := range Rows(v.feed("playerScores", strconv.Itoa(w))) {
				if score.ID.String() == p.ID {
					weeks = append(weeks, fmt.Sprintf("W%d: %.2f", w, score.Value))
					total += score.Value
				}
			}
		}
		if len(weeks) > 0 {
			text = fmt.Sprintf("%s · %.2f PPG over %d of the last 3 finished weeks; season %.2f PPG",
				strings.Join(weeks, ", "), total/float64(len(weeks)), len(weeks), season.Sum/float64(season.Games))
		} else {
			text = "None of the last 3 finished weekly scores held for this player"
		}
	}
	r.Rows = append(r.Rows, report.Row{Cells: []string{"Form", text, HeldTime(latest, "", v.Zone)}})
}

// Feed is a published immutable payload plus its held read state, not a cache owner.
type Feed[T any] = report.HeldFeed[T]

// Score is the shared finished-week calculation supplied by its root owner.
type Score struct {
	Position string
	Sum      float64
	Games    int
}

// Key identifies one kind/window of a held feed.
func Key(kind, week string) string { return kind + ":" + week }

// FeedLabel is the display name of a Research export.
func FeedLabel(kind string) string {
	return map[string]string{"playerScores": "fantasy points", "topAdds": "most added players",
		"topDrops": "most dropped players", "topStarters": "most started players", "topOwns": "most owned players",
		"topTrades": "most traded players", "pointsAllowed": "points allowed"}[kind]
}

// Kinds is the ordered set of Research exports read by the root refresh coordinator.
func Kinds() []string {
	return []string{"playerScores", "topAdds", "topDrops", "topStarters", "topOwns", "topTrades", "pointsAllowed"}
}
