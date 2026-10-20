package research

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type Scope struct {
	Mine     string `json:"mine"`
	Search   string `json:"search"`
	Position string `json:"position"`
	Team     string `json:"team"`
	Group    string `json:"group"`
	Trend    string `json:"trend"`
	Depth    string `json:"depth"`
	Injuries string `json:"injuries"`
}

func (r Scope) Validate() error {
	if r.Mine != "" {
		if err := ingestion.FeedFranchise(r.Mine); err != nil {
			return fmt.Errorf("research franchise: %w", err)
		}
	}
	if len(r.Search) > 100 {
		return fmt.Errorf("research: search is too long")
	}
	if r.Team != "" && (len(r.Team) > 4 || strings.IndexFunc(r.Team, func(c rune) bool { return c < 'A' || c > 'Z' }) >= 0) {
		return fmt.Errorf("research: invalid NFL team")
	}
	for _, choice := range []struct {
		Value   string
		options []string
	}{
		{r.Position, researchPositions()}, {r.Group, []string{"QB", "RB", "WR/TE", "K", "IDP", "DST"}},
		{r.Trend, []string{"topAdds", "topDrops", "topStarters", "topOwns", "topTrades"}},
		{r.Depth, []string{"team", "grid"}}, {r.Injuries, []string{"all", "games"}},
	} {
		if choice.Value != "" && !slices.Contains(choice.options, choice.Value) {
			return fmt.Errorf("research: invalid report choice")
		}
	}
	return nil
}

func researchPositions() []string {
	return []string{"QB", "RB", "WR", "TE", "K", "DT", "DE", "LB", "CB", "S", "DST"}
}

type PlayerInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Position    string `json:"position"`
	Team        string `json:"team"`
	FranchiseID string `json:"franchiseId"`
	Owner       string `json:"owner"`
}
type Article struct {
	ID       string `json:"id"`
	Article  string `json:"article"`
	Name     string `json:"name"`
	Headline string `json:"headline"`
}
type Board struct {
	Board     report.Board  `json:"board"`
	News      report.Report `json:"news"`
	Articles  []Article     `json:"articles"`
	Players   []PlayerInfo  `json:"players"`
	Teams     []string      `json:"teams"`
	Positions []string      `json:"positions"`
	Scope     Scope         `json:"scope"`
}

// Snapshot is a detached reading input. Published feed payloads and Lookup are immutable;
// the root copies mutable map indexes while holding its locks before calling Build.
type Snapshot struct {
	Finished   map[string]*Score
	Latest     time.Time
	CutCharges map[string]domain.Money
	Held       Held
	Lookup     normalize.Lookup
	Mirror     state.MirrorSnapshot
	Injuries   Feed[[]leaguefeed.Injury]
	Schedules  map[int]Feed[[]nflschedule.RawWeek]
	Config     league.RawConfig
	Week       int
	Zone       *time.Location
	Now        time.Time
}
type researchView struct {
	Snapshot
	scope  Scope
	mine   map[string]bool
	owners map[string]bool
}

func Build(in Snapshot, scope Scope) Board {
	if scope.Group == "" {
		scope.Group = "QB"
	}
	if scope.Trend == "" {
		scope.Trend = "topAdds"
	}
	if scope.Depth == "" {
		scope.Depth = "team"
	}
	if scope.Injuries == "" {
		scope.Injuries = "all"
	}
	v := researchView{Snapshot: in, scope: scope, mine: map[string]bool{}, owners: map[string]bool{}}
	for _, p := range in.Mirror.Players {
		v.owners[p.MFLID] = true
		if scope.Mine != "" && p.FranchiseID == scope.Mine {
			v.mine[p.MFLID] = true
		}
	}
	players, teams := v.search()
	reports := []report.Report{v.stats("M-038", true, false), v.stats("M-037", false, false), v.stats("M-040", false, true), v.injuryReport("M-057", false), v.injuryReport("M-051", true), v.depthReport(), v.trends(), v.allowedReport("M-053", true), v.allowedReport("M-054", false)}
	news, articles := v.newsReport(false)
	mineNews, _ := v.newsReport(true)
	reports = append(reports, mineNews)
	return Board{Board: report.Board{Title: "Player research", Loading: "Reading player research…", Refresh: "Refresh from sources", HowLabel: "How this is counted", Reports: reports},
		News: news, Articles: articles, Players: players, Teams: teams, Positions: researchPositions(), Scope: scope}
}
func (v researchView) player(id string) PlayerInfo {
	f, raw, ok := v.Lookup.DisplayFacts(id)
	if !ok {
		return PlayerInfo{ID: id, Name: "A player MFL’s directory does not list", Position: "Unknown", Team: "Unknown"}
	}
	position := string(f.Position)
	if position == "FLAG" {
		position = "Unknown"
	}
	if position == "" {
		position = raw
		if raw == "Def" {
			position = "DST"
		}
	}
	if position == "PK" {
		position = "K"
	}
	owner := ""
	for _, p := range v.Mirror.Players {
		if p.MFLID == id {
			owner = p.FranchiseID
			break
		}
	}
	label := "Free agent"
	if owner != "" {
		for _, f := range v.Config.Franchises {
			if f.ID == owner {
				label = f.Name
			}
		}
		if label == "Free agent" {
			label = "Franchise " + owner
		}
	}
	return PlayerInfo{ID: id, Name: f.Name, Position: position, Team: Team(f.Team), FranchiseID: owner, Owner: label}
}
func (v researchView) search() ([]PlayerInfo, []string) {
	players := []PlayerInfo{}
	teams := map[string]bool{}
	for _, id := range v.Lookup.IDs() {
		p := v.player(id)
		if p.Team != "" {
			teams[p.Team] = true
		}
		hay := strings.ToLower(p.Name + " " + p.Position + " " + p.Team)
		if strings.TrimSpace(v.scope.Search) != "" && strings.Contains(hay, strings.ToLower(strings.TrimSpace(v.scope.Search))) {
			players = append(players, p)
		}
	}
	sort.Slice(players, func(i, j int) bool { return players[i].Name < players[j].Name })
	if len(players) > 40 {
		players = players[:40]
	}
	options := make([]string, 0, len(teams))
	for t := range teams {
		options = append(options, t)
	}
	sort.Strings(options)
	return players, options
}
func (v researchView) feed(kind, week string) Feed[[]researchfeeds.Feed] {
	return v.Held.Feeds[Key(kind, week)]
}
func researchRows(feed Feed[[]researchfeeds.Feed]) []researchfeeds.Entry {
	if len(feed.Value) == 0 {
		return nil
	}
	return feed.Value[0].Players
}
func (v researchView) matches(p PlayerInfo) bool {
	return (v.scope.Position == "" || v.scope.Position == p.Position) && (v.scope.Team == "" || v.scope.Team == p.Team)
}
func (v researchView) inGroup(p PlayerInfo) bool {
	switch v.scope.Group {
	case "WR/TE":
		return p.Position == "WR" || p.Position == "TE"
	case "IDP":
		return slices.Contains([]string{"DT", "DE", "LB", "CB", "S"}, p.Position)
	}
	return p.Position == v.scope.Group
}
func (v researchView) stats(id string, mine, group bool) report.Report {
	title := map[string]string{"M-037": "Player stats", "M-038": "My top performers", "M-040": "Position leaders"}[id]
	feed := v.feed("playerScores", "YTD")
	r := researchShell(id, title, "MFL fantasy points under this league’s scoring, not raw passing or rushing yards. Season totals are MFL’s YTD export; weekly values are its separate weekly fantasy points. A dash means a missing score, not zero. No points enter the scouting prior.", feed, v.Zone, "Player", "Pos", "NFL", "Season")
	r.Source = "Source: MFL fantasy points, this league’s scoring"
	weekly := v.weekScores(&r)
	last := max(v.Week-1, 0)
	if mine && v.scope.Mine == "" {
		r.Status = "empty"
		r.Note = "Choose your franchise to see your top performers."
		return r
	}
	rows := append([]researchfeeds.Entry{}, researchRows(feed)...)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Value > rows[j].Value })
	missing := 0
	for _, entry := range rows {
		p := v.player(entry.ID.String())
		if p.Position != "Unknown" && !slices.Contains(researchPositions(), p.Position) {
			continue
		}
		if (!group && !v.matches(p)) || (mine && !v.mine[p.ID]) || (group && !v.inGroup(p)) {
			continue
		}
		if p.Position == "Unknown" {
			missing++
		}
		r.Rows = append(r.Rows, v.statsRow(p, entry.Value, weekly))
	}
	researchTextSort(r.Rows, 3)
	r.SortAscending = "Sort ascending"
	r.SortDescending = "Sort descending"
	r.Note += " Last finished week: " + researchLastLabel(last) + "."
	if missing > 0 {
		r.Note += fmt.Sprintf(" Directory: %d scored players cannot be named.", missing)
	}
	if group {
		r.Note += v.slotNote()
	}
	researchEmpty(&r)
	return r
}
func (v researchView) weekScores(r *report.Report) map[int]map[string]float64 {
	weekly := map[int]map[string]float64{}
	var differ []string // per Week: its held time when it differs from week 1's, else ""
	for w := 1; w <= v.Week; w++ {
		r.Columns = append(r.Columns, fmt.Sprintf("W%d", w))
		h := v.feed("playerScores", strconv.Itoa(w))
		weekly[w] = map[string]float64{}
		for _, p := range researchRows(h) {
			weekly[w][p.ID.String()] = p.Value
		}
		base := v.feed("playerScores", "1")
		if w == 1 {
			r.Source += fmt.Sprintf(" · weeks 1–%d %s", v.Week, HeldTime(h.FetchedAt, h.RefreshError, v.Zone))
			r.AsOf = ""
		}
		text := HeldTime(h.FetchedAt, h.RefreshError, v.Zone)
		if text == HeldTime(base.FetchedAt, base.RefreshError, v.Zone) {
			text = ""
		}
		differ = append(differ, text)
		if (h.FetchedAt.IsZero() || h.RefreshError != "") && r.Status == "ok" {
			r.Status = "stale"
		}
	}
	if strings.Contains(r.Source, "Archived copy") { // the source line already says so
		r.Note = strings.Replace(r.Note, "Archived copy held; refresh pending.", "", 1)
	}
	r.Note += weekRuns(differ)
	return weekly
}

// weekRuns names each run of consecutive weeks whose held time differs from week 1's, once:
// "Weeks 2–5 · as of … ." rather than one repeated line per week.
func weekRuns(differ []string) string {
	out := ""
	for i := 0; i < len(differ); {
		j := i
		for j+1 < len(differ) && differ[j+1] == differ[i] {
			j++
		}
		switch {
		case differ[i] == "":
		case i == j:
			out += fmt.Sprintf(" Week %d · %s.", i+1, differ[i])
		default:
			out += fmt.Sprintf(" Weeks %d–%d · %s.", i+1, j+1, differ[i])
		}
		i = j + 1
	}
	return out
}
func scoreCell(scores map[string]float64, id string) (string, float64) {
	if n, ok := scores[id]; ok {
		return fmt.Sprintf("%.2f", n), n
	}
	return "—", -1e9
}
func (v researchView) statsRow(p PlayerInfo, total float64, weekly map[int]map[string]float64) report.Row {
	row := report.Row{Cells: []string{p.Name, p.Position, p.Team, fmt.Sprintf("%.2f", total)}, Sort: []float64{0, 0, 0, total}, Mine: v.mine[p.ID]}
	for w := 1; w <= v.Week; w++ {
		text, key := scoreCell(weekly[w], p.ID)
		row.Cells = append(row.Cells, text)
		row.Sort = append(row.Sort, key)
	}
	return row
}
func (v researchView) slotNote() string {
	code := map[string]string{"K": "PK", "DST": "Def"}[v.scope.Group]
	if code == "" {
		return ""
	}
	for _, p := range v.Config.Starters.Positions {
		if p.Name == code && p.Limit != "0" && p.Limit != "0-0" {
			return ""
		}
	}
	return " This league has no " + v.scope.Group + " starter slots; MFL’s numbers are still shown."
}

func researchEmpty(r *report.Report) {
	if r.Status == "ok" && len(r.Rows) == 0 {
		r.Status = "empty"
		r.Note += " No players match these choices."
	}
}
func researchLoss(r *report.Report, label string, at time.Time, err string, zone *time.Location) {
	text := "As of " + report.Time(at, zone)
	switch {
	case at.IsZero():
		text = "Unavailable; refresh from sources"
	case strings.Contains(strings.ToLower(err), "archived copy"):
		text += " · archived copy; refresh pending"
	case err != "":
		text += " · refresh failed; showing last held copy"
	}
	r.Summary = append(r.Summary, report.Row{Cells: []string{label, text}})
	if (at.IsZero() || err != "") && r.Status == "ok" {
		r.Status = "stale"
	}
}
func (v researchView) trends() report.Report {
	feed := v.feed(v.scope.Trend, "")
	r := researchShell("M-052", "MFL-wide activity", "The order and percentages are exactly MFL’s selected export. This is activity across MFL, not transactions in this league. Percentages are as MFL supplies them; no league-specific inference.", feed, v.Zone, "Rank", "Player", "Pos", "NFL", "Percent")
	r.Source = "Source: MFL " + FeedLabel(v.scope.Trend) + " · MFL-wide, not this league"
	for i, p := range researchRows(feed) {
		name := v.player(p.ID.String())
		r.Rows = append(r.Rows, report.Row{Cells: []string{strconv.Itoa(i + 1), name.Name, name.Position, name.Team, fmt.Sprintf("%.2f%%", p.Value)}, Mine: v.mine[name.ID]})
	}
	return r
}
func (v researchView) allowedReport(id string, position bool) report.Report {
	feed := v.feed("pointsAllowed", "")
	title := "Points allowed per NFL team"
	if position {
		title = "Points allowed by position"
	}
	r := researchShell(id, title, "MFL season fantasy points conceded under this league’s scoring. All bars share one scale within this report, from zero to the largest shown total. These are totals, not per-game averages.", feed, v.Zone, "NFL team", "Position", "Points allowed")
	r.Source = "Source: MFL points allowed · this league’s scoring"
	if len(feed.Value) == 0 {
		return r
	}
	totals := v.allowedTotals(feed.Value[0].Allowed, position)
	keys := make([]string, 0, len(totals))
	for k := range totals {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if totals[keys[i]] == totals[keys[j]] {
			return keys[i] < keys[j]
		}
		return totals[keys[i]] > totals[keys[j]]
	})
	maxValue := 0.0
	for _, n := range totals {
		if n > maxValue {
			maxValue = n
		}
	}
	for _, k := range keys {
		parts := strings.Split(k, "|")
		n := totals[k]
		row := report.Row{Cells: []string{parts[0], parts[1], fmt.Sprintf("%.2f", n)}}
		if n >= 0 && maxValue > 0 {
			row.Range = &report.Range{Interval: lab.Interval{Value: n, Lo: 0, Hi: n}, Max: maxValue, Text: fmt.Sprintf("%.2f points allowed", n)}
		}
		if position {
			row.Group = parts[1]
		}
		r.Rows = append(r.Rows, row)
	}
	if position {
		sort.SliceStable(r.Rows, func(i, j int) bool { return r.Rows[i].Group < r.Rows[j].Group })
	}
	researchEmpty(&r)
	return r
}

func researchTextSort(rows []report.Row, columns int) {
	for col := 0; col < columns; col++ {
		ordered := make([]string, 0, len(rows))
		for _, row := range rows {
			ordered = append(ordered, row.Cells[col])
		}
		sort.Strings(ordered)
		ranks := map[string]float64{}
		for i, s := range ordered {
			ranks[s] = float64(i)
		}
		for i := range rows {
			rows[i].Sort[col] = ranks[rows[i].Cells[col]]
		}
	}
}

func (v researchView) allowedTotals(rows []researchfeeds.Allowed, position bool) map[string]float64 {
	totals := map[string]float64{}
	for _, p := range rows {
		pos := p.Position
		if pos == "PK" {
			pos = "K"
		}
		if v.scope.Team != "" && v.scope.Team != Team(p.Team) || v.scope.Position != "" && v.scope.Position != pos {
			continue
		}
		team := Team(p.Team)
		key := team + "|" + pos
		if !position {
			key = team + "|All positions"
		}
		totals[key] += p.Points
	}
	return totals
}

func Team(raw string) string {
	aliases := map[string]string{"ARZ": "ARI", "LA": "LAR", "STL": "LAR", "OAK": "LV", "SD": "LAC", "JAC": "JAX"}
	if team := aliases[raw]; team != "" {
		return team
	}
	return raw
}

func researchLastLabel(week int) string {
	if week == 0 {
		return "No previous NFL week held"
	}
	return fmt.Sprintf("Week %d", week)
}

// Player resolves a held directory/roster identity without a reading or live state.
func Player(lookup normalize.Lookup, mirror state.MirrorSnapshot, config league.RawConfig, id string) PlayerInfo {
	return (researchView{Snapshot: Snapshot{Lookup: lookup, Mirror: mirror, Config: config}}).player(id)
}

// ReadPlayer constructs the held player reading from a detached snapshot.
func ReadPlayer(in Snapshot, id string, names map[string]string, how string) PlayerReading {
	return (researchView{Snapshot: in}).playerReading(id, names, how)
}
