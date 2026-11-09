package research

import (
	"sort"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
)

const (
	usageWeeks = 4

	usageTeam        = "context.team"
	usageOffenseSnap = "exposure.offense_snap_share"
	usageDefenseSnap = "exposure.defense_snap_share"
	usageTargets     = "opportunity.targets"
	usageCarries     = "opportunity.carries"
)

// UsageMeasures lists the weekly facts TargetPlayerUsage reads from history.
func UsageMeasures() []string {
	return []string{usageTeam, usageOffenseSnap, usageDefenseSnap, usageTargets, usageCarries}
}

// UsageFeature is one stored measure at an NFL week, as the history store hands it over.
type UsageFeature struct {
	Player  string
	Season  int
	Week    int
	Measure string
	Value   float64
	Text    string
}

// UsageSource is the week-grain history read behind the usage read. The root reads it once per
// player request; it never refreshes a source.
type UsageSource struct {
	Season int
	Rows   func() ([]UsageFeature, error)
}

// UsageHost is implemented by the host that can read weekly history for the usage read.
type UsageHost interface {
	UsageSource() UsageSource
}

// PlayerUsage is the HQ drawer's THIS WEEK block: the player's share of his team's work over the
// last four played NFL weeks, and the next opponent's points allowed to his position.
type PlayerUsage struct {
	SnapShare     *float64 `json:"snapShare"`
	TargetShare   *float64 `json:"targetShare"`
	CarryShare    *float64 `json:"carryShare"`
	WeeksUsed     int      `json:"weeksUsed"`
	Opponent      string   `json:"opponent"`
	AllowedPoints *float64 `json:"allowedPoints"`
	AllowedRank   *int     `json:"allowedRank"`
	Teams         int      `json:"teams"`
	Label         string   `json:"label"`
	Reason        string   `json:"reason"`
}

type usageVals map[string]float64

type teamWeek struct {
	targets, carries     float64
	hasTargets, hasCarry bool
}

type usageIndex struct {
	teamOf     map[string]map[int]string
	playerVals map[string]map[int]usageVals
	teamWeeks  map[string]map[int]bool
	teamTotals map[int]map[string]*teamWeek
}

// ReadPlayerUsage derives usage and matchup from stored rows. A missing week shortens the
// window; a zero team total leaves its share null, never zero.
func ReadPlayerUsage(in Snapshot, id, position string, season int, rows []UsageFeature) PlayerUsage {
	index := indexUsage(rows, season)
	team, _ := index.latestTeam(id)
	if team == "" {
		return PlayerUsage{Reason: "NFL team is not held for this player"}
	}
	usage, notes := index.shares(id, position, team)
	opponent, allowed, rank, teams, matchupNotes := usageMatchup(in, team, position)
	usage.Opponent = opponent
	usage.AllowedPoints = allowed
	usage.AllowedRank = rank
	usage.Teams = teams
	if rank != nil {
		usage.Label = usageLabel(*rank, teams)
	}
	notes = append(notes, matchupNotes...)
	usage.Reason = strings.Join(notes, "; ")
	return usage
}

func indexUsage(rows []UsageFeature, season int) usageIndex {
	index := usageIndex{
		teamOf:     map[string]map[int]string{},
		playerVals: map[string]map[int]usageVals{},
		teamWeeks:  map[string]map[int]bool{},
		teamTotals: map[int]map[string]*teamWeek{},
	}
	index.teams(rows, season)
	index.measures(rows, season)
	return index
}

func (index usageIndex) teams(rows []UsageFeature, season int) {
	for _, row := range rows {
		if row.Season != season || row.Measure != usageTeam {
			continue
		}
		team := Team(strings.TrimSpace(row.Text))
		if team == "" {
			continue
		}
		if index.teamOf[row.Player] == nil {
			index.teamOf[row.Player] = map[int]string{}
		}
		index.teamOf[row.Player][row.Week] = team
		if index.teamWeeks[team] == nil {
			index.teamWeeks[team] = map[int]bool{}
		}
		index.teamWeeks[team][row.Week] = true
	}
}

func (index usageIndex) measures(rows []UsageFeature, season int) {
	for _, row := range rows {
		if row.Season != season || !usageMeasure(row.Measure) {
			continue
		}
		if index.playerVals[row.Player] == nil {
			index.playerVals[row.Player] = map[int]usageVals{}
		}
		if index.playerVals[row.Player][row.Week] == nil {
			index.playerVals[row.Player][row.Week] = usageVals{}
		}
		index.playerVals[row.Player][row.Week][row.Measure] = row.Value
		index.total(row, index.teamOf[row.Player][row.Week])
	}
}

func usageMeasure(measure string) bool {
	switch measure {
	case usageOffenseSnap, usageDefenseSnap, usageTargets, usageCarries:
		return true
	}
	return false
}

func (index usageIndex) total(row UsageFeature, team string) {
	if team == "" {
		return
	}
	if index.teamTotals[row.Week] == nil {
		index.teamTotals[row.Week] = map[string]*teamWeek{}
	}
	if index.teamTotals[row.Week][team] == nil {
		index.teamTotals[row.Week][team] = &teamWeek{}
	}
	total := index.teamTotals[row.Week][team]
	switch row.Measure {
	case usageTargets:
		total.targets += row.Value
		total.hasTargets = true
	case usageCarries:
		total.carries += row.Value
		total.hasCarry = true
	}
}

func (index usageIndex) latestTeam(id string) (string, int) {
	latestTeam, latestWeek := "", 0
	for week, team := range index.teamOf[id] {
		if week > latestWeek {
			latestWeek, latestTeam = week, team
		}
	}
	return latestTeam, latestWeek
}

func (index usageIndex) window(team string) []int {
	weeks := make([]int, 0, len(index.teamWeeks[team]))
	for week := range index.teamWeeks[team] {
		weeks = append(weeks, week)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(weeks)))
	if len(weeks) > usageWeeks {
		weeks = weeks[:usageWeeks]
	}
	sort.Ints(weeks)
	return weeks
}

// shares computes the three shares over the team's last four played weeks. A week with no
// player row shortens the window rather than adding a zero.
func (index usageIndex) shares(id, position, team string) (PlayerUsage, []string) {
	usage := PlayerUsage{}
	weeks := index.window(team)
	snapMeasure := usageSnapMeasure(position, index.playerVals[id])
	var snap, target, carry shareAccum
	for _, week := range weeks {
		vals := index.playerVals[id][week]
		if len(vals) == 0 {
			continue
		}
		usage.WeeksUsed++
		if snapMeasure != "" {
			if v, ok := vals[snapMeasure]; ok {
				snap.add(v, 1, true)
			}
		}
		if v, ok := vals[usageTargets]; ok {
			total, held := index.denominator(week, team, usageTargets)
			target.add(v, total, held && total > 0)
		}
		if v, ok := vals[usageCarries]; ok {
			total, held := index.denominator(week, team, usageCarries)
			carry.add(v, total, held && total > 0)
		}
	}
	usage.SnapShare = snap.value()
	usage.TargetShare = target.value()
	usage.CarryShare = carry.value()
	notes := []string{}
	if usage.WeeksUsed == 0 {
		notes = append(notes, "No played NFL weeks are held for this player")
	}
	if target.zero && usage.TargetShare == nil {
		notes = append(notes, "Team targets are zero in the held weeks")
	}
	if carry.zero && usage.CarryShare == nil {
		notes = append(notes, "Team carries are zero in the held weeks")
	}
	return usage, notes
}

type shareAccum struct {
	sum, total float64
	n          int
	zero       bool
}

func (a *shareAccum) add(value, total float64, count bool) {
	if !count {
		a.zero = true
		return
	}
	a.sum += value
	a.total += total
	a.n++
}

func (a shareAccum) value() *float64 {
	if a.n == 0 {
		return nil
	}
	return floatValue(a.sum / a.total)
}

func (index usageIndex) denominator(week int, team, measure string) (float64, bool) {
	total := index.teamTotals[week][team]
	if total == nil {
		return 0, false
	}
	switch measure {
	case usageTargets:
		return total.targets, total.hasTargets
	case usageCarries:
		return total.carries, total.hasCarry
	}
	return 0, false
}

func usageSnapMeasure(position string, weeks map[int]usageVals) string {
	switch position {
	case "QB", "RB", "WR", "TE":
		return usageOffenseSnap
	case "DE", "DT", "LB", "CB", "S":
		return usageDefenseSnap
	}
	for _, vals := range weeks {
		if _, ok := vals[usageOffenseSnap]; ok {
			return usageOffenseSnap
		}
		if _, ok := vals[usageDefenseSnap]; ok {
			return usageDefenseSnap
		}
	}
	return ""
}

// usageMatchup reads points allowed through the existing allowed-totals derivation and ranks
// the opponent against every team that has a total for the position.
func usageMatchup(in Snapshot, team, position string) (string, *float64, *int, int, []string) {
	v := researchView{Snapshot: in, scope: Scope{Position: position}}
	feed := v.feed("pointsAllowed", "")
	if len(feed.Value) == 0 {
		return "", nil, nil, 0, []string{"Points allowed are not held for this season"}
	}
	totals := v.allowedTotals(feed.Value[0].Allowed, true)
	teams := len(totals)
	if teams == 0 {
		return "", nil, nil, 0, []string{"Points allowed to " + position + " are not held"}
	}
	opponent := nextOpponent(in.Schedules, in.Week, team)
	if opponent == "" {
		return "", nil, nil, teams, []string{"No held NFL game is upcoming for this team"}
	}
	rank, points := usageRank(totals, opponent)
	if rank == 0 {
		return opponent, nil, nil, teams, []string{"Points allowed to " + position + " are not held for " + opponent}
	}
	return opponent, floatValue(points), &rank, teams, nil
}

func usageRank(totals map[string]float64, opponent string) (int, float64) {
	type teamScore struct {
		team   string
		points float64
	}
	scores := make([]teamScore, 0, len(totals))
	for key, points := range totals {
		name, _, _ := strings.Cut(key, "|")
		scores = append(scores, teamScore{team: name, points: points})
	}
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].points == scores[j].points {
			return scores[i].team < scores[j].team
		}
		return scores[i].points > scores[j].points
	})
	for i, score := range scores {
		if score.team == opponent {
			return i + 1, score.points
		}
	}
	return 0, 0
}

func nextOpponent(schedules map[int]Feed[[]nflschedule.RawWeek], week int, team string) string {
	type game struct {
		week     int
		opponent string
	}
	games := []game{}
	for key, feed := range schedules {
		for _, raw := range feed.Value {
			number, err := strconv.Atoi(strings.TrimSpace(raw.Week))
			if err != nil {
				number = key
			}
			if number < week {
				continue
			}
			for _, matchup := range raw.Matchups {
				home, away := Team(strings.TrimSpace(matchup.Teams[0])), Team(strings.TrimSpace(matchup.Teams[1]))
				switch team {
				case home:
					games = append(games, game{number, away})
				case away:
					games = append(games, game{number, home})
				}
			}
		}
	}
	sort.Slice(games, func(i, j int) bool {
		if games[i].week == games[j].week {
			return games[i].opponent < games[j].opponent
		}
		return games[i].week < games[j].week
	})
	if len(games) == 0 {
		return ""
	}
	return games[0].opponent
}

// usageLabel divides the held teams into thirds from the count, not from a fixed 32.
func usageLabel(rank, teams int) string {
	if rank < 1 || teams < 1 {
		return ""
	}
	top := (teams + 2) / 3
	bottom := teams - top + 1
	switch {
	case rank <= top:
		return "Soft"
	case rank >= bottom:
		return "Tough"
	default:
		return "Neutral"
	}
}

func floatValue(v float64) *float64 { return &v }
