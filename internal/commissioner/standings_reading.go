package commissioner

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// StandingRow is one franchise's place. ComputedRecord is set only where a commissioner record differs.
type StandingRow struct {
	Rank           int                 `json:"rank"`
	Franchise      string              `json:"franchise"`
	Name           string              `json:"name"`
	Division       string              `json:"division,omitempty"`
	Wins           int                 `json:"wins"`
	Losses         int                 `json:"losses"`
	Ties           int                 `json:"ties"`
	Record         string              `json:"record"`
	ComputedRecord string              `json:"computedRecord,omitempty"`
	PointsFor      leaguerules.Decimal `json:"pointsFor"`
	PointsWords    string              `json:"pointsWords"`
	Why            []string            `json:"why"`
}

// StandingsOverrideView is what the commissioner has set to stand, shown beside the computed standings.
type StandingsOverrideView struct {
	Words   string                    `json:"words"`
	Rows    []StandingRow             `json:"rows"`
	Order   *OrderOverride            `json:"order,omitempty"`
	Records map[string]AdjustedRecord `json:"records"`
}

// NativeStandings ranks a season's decided regular-season games under the league's own tie-breakers.
// Computed is always present; an override never replaces it.
type NativeStandings struct {
	League   string                 `json:"league"`
	Name     string                 `json:"name"`
	Season   int                    `json:"season"`
	Revision int64                  `json:"revision"`
	Status   string                 `json:"status"`
	Words    string                 `json:"words"`
	Through  int                    `json:"through"`
	Decided  int                    `json:"decided"`
	Pending  int                    `json:"pending"`
	Criteria []string               `json:"criteria"`
	NoData   []string               `json:"noData"`
	Computed []StandingRow          `json:"computed"`
	Override *StandingsOverrideView `json:"override,omitempty"`
	Changes  []ScoreChange          `json:"changes"`
	// level and overrideLevel are the teams left level by the computed ranking and by the ranking under the
	// commissioner's records; the draft's first round reads them to apply its own tie rule.
	level, overrideLevel [][]string
	regularWeeks         int
}

// seasonGames is every decided game and complete score of the counted weeks.
type seasonGames struct {
	games   []rankGame
	allPlay map[string]tally
	points  map[string]leaguerules.Decimal
	decided int
	pending int
}

// NativeStandingsReading counts games through the last regular-season week and no later.
func NativeStandingsReading(ctx context.Context, league string, records []Record, src ResultSources, season int,
	now time.Time) (NativeStandings, error) {
	if season < 1920 || season > 9999 {
		return NativeStandings{}, fmt.Errorf("choose a season")
	}
	out := NativeStandings{League: league, Season: season, Revision: int64(len(records)), Status: "Unavailable",
		Criteria: []string{}, NoData: []string{}, Computed: []StandingRow{}, Changes: []ScoreChange{}}
	if len(records) == 0 {
		return out, fmt.Errorf("this app league has not been created")
	}
	out.Name = Replay(records).Name
	if !seeded(records) {
		out.Words = "Record the starting snapshot first"
		return out, nil
	}
	rules, err := RulesInForce(records, season, ReadingInstant(records, now))
	if err != nil {
		return out, err
	}
	published, err := resultSchedule(records, season, now)
	if err != nil {
		return out, err
	}
	adjustments, err := StandingsAdjustmentsReading(records, season)
	if err != nil {
		return out, err
	}
	out.Changes = adjustments.Changes
	switch {
	case rules.Standings.Criteria.State != leaguerules.IsKnown || len(rules.Standings.Criteria.Value) == 0:
		out.Status, out.Words = "Needs a ruling", "Needs a ruling: how teams are ranked and ties broken"
		return out, nil
	case rules.Schedule.RegularWeeks.State != leaguerules.IsKnown:
		out.Status, out.Words = "Needs a ruling", "Needs a ruling: the number of regular-season weeks"
		return out, nil
	case len(published) == 0:
		out.Words = "The league schedule is not published yet, so there are no games to rank"
		return out, nil
	}
	for _, c := range rules.Standings.Criteria.Value {
		out.Criteria = append(out.Criteria, criterionLabel(c))
	}
	directory, err := NativeDirectoryReading(league, records, now)
	if err != nil {
		return out, err
	}
	out.regularWeeks = rules.Schedule.RegularWeeks.Value
	out.Through = min(rules.Schedule.RegularWeeks.Value, len(published))
	counted, err := countSeason(ctx, league, records, src, rules.Lineup, directory, published[:out.Through], season)
	if err != nil {
		return out, err
	}
	return out.rank(rules, directory, counted, adjustments, conferencesFor(records, rules, directoryIDs(directory)))
}

// countSeason reads each counted week's results once.
func countSeason(ctx context.Context, league string, records []Record, src ResultSources, lineup leaguerules.LineupRules,
	directory NativeDirectory, weeks []ScheduleWeek, season int) (seasonGames, error) {
	out := seasonGames{allPlay: map[string]tally{}, points: map[string]leaguerules.Decimal{}}
	for _, week := range weeks {
		results, err := weekResults(ctx, league, records, src, lineup, directory, weeks, season, week.Week)
		if err != nil {
			return out, err
		}
		for _, g := range results.Games {
			if g.Status == GameNoResult {
				out.pending++
				continue
			}
			out.decided++
			out.games = append(out.games, rankGame{Home: g.Home, Away: g.Away, HomePoints: *g.HomePoints, AwayPoints: *g.AwayPoints})
		}
		for _, score := range results.Scores {
			if score.Points != nil {
				if out.points[score.Franchise], err = addScoreDelta(out.points[score.Franchise], *score.Points); err != nil {
					return out, err
				}
			}
		}
		out.addAllPlay(results.Scores)
	}
	return out, nil
}

// addAllPlay counts a week only when every franchise that played has a complete score, so no one is
// compared against a missing one.
func (s *seasonGames) addAllPlay(scores []FranchiseWeekScore) {
	for _, mine := range scores {
		if mine.Points == nil {
			return
		}
	}
	for _, mine := range scores {
		for _, theirs := range scores {
			if mine.Franchise != theirs.Franchise {
				s.allPlay[mine.Franchise] = resultOf(*mine.Points, *theirs.Points).add(s.allPlay[mine.Franchise])
			}
		}
	}
}

func (out NativeStandings) rank(rules leaguerules.RuleSet, directory NativeDirectory, counted seasonGames,
	adjustments StandingsAdjustments, conferences conferenceLayout) (NativeStandings, error) {
	teams := make([]rankTeam, 0, len(directory.Franchises))
	for _, seat := range directory.Franchises {
		division := ""
		if rules.League.Alignment.State == leaguerules.IsKnown {
			division = rules.League.Alignment.Value[seat.Franchise]
		}
		teams = append(teams, rankTeam{ID: seat.Franchise, Name: seat.Name, Division: division,
			Conference: conferences.Of[seat.Franchise]})
	}
	input := rankInput{teams: teams, games: counted.games, allPlay: counted.allPlay, points: counted.points,
		criteria: rules.Standings.Criteria.Value}
	computed := rankStandings(input)
	out.Computed, out.level = standingRows(computed, teams, counted.points, nil), computed.level
	out.NoData = computed.skipped
	out.Decided, out.Pending, out.Status = counted.decided, counted.pending, "Ranked"
	out.Words = fmt.Sprintf("Standings through week %d: %d games have a result, %d do not yet", out.Through,
		counted.decided, counted.pending)
	if out.Through < rules.Schedule.RegularWeeks.Value {
		out.Words += fmt.Sprintf("; the published schedule covers only %d of %d weeks", out.Through,
			rules.Schedule.RegularWeeks.Value)
	}
	if len(adjustments.Records) == 0 && adjustments.Order == nil {
		return out, nil
	}
	input.stands = map[string]tally{}
	for id, r := range adjustments.Records {
		input.stands[id] = tally{W: r.Wins, L: r.Losses, T: r.Ties}
	}
	stands := rankStandings(input)
	view := &StandingsOverrideView{Records: adjustments.Records, Order: adjustments.Order}
	view.Rows = standingRows(stands, teams, counted.points, &computed)
	view.Words = "The commissioner's records stand beside the computed standings"
	if adjustments.Order != nil {
		view.Rows = overrideOrderRows(view.Rows, *adjustments.Order)
		view.Words = "The commissioner set the order by hand: " + adjustments.Order.Reason
	}
	out.Override, out.overrideLevel = view, stands.level
	return out, nil
}

// standingRows lists the ranking's order with each record and the lines behind its ties. When a computed
// ranking is given, a record the commissioner changed shows the computed one beside it.
func standingRows(r ranking, teams []rankTeam, points map[string]leaguerules.Decimal, computed *ranking) []StandingRow {
	byID := map[string]rankTeam{}
	for _, t := range teams {
		byID[t.ID] = t
	}
	rows := make([]StandingRow, 0, len(r.order))
	for i, id := range r.order {
		record := r.records[id]
		row := StandingRow{Rank: i + 1, Franchise: id, Name: byID[id].Name, Division: byID[id].Division, Wins: record.W,
			Losses: record.L, Ties: record.T, Record: record.words(), PointsFor: points[id],
			PointsWords: scorePointsWords(points[id]) + " points scored", Why: append([]string{}, r.why[id]...)}
		if computed != nil && computed.records[id] != record {
			row.ComputedRecord = computed.records[id].words()
		}
		rows = append(rows, row)
	}
	return rows
}

// overrideOrderRows puts the rows in the commissioner's order and says why each sits where it does.
func overrideOrderRows(rows []StandingRow, override OrderOverride) []StandingRow {
	byID := map[string]StandingRow{}
	for _, row := range rows {
		byID[row.Franchise] = row
	}
	ordered := make([]StandingRow, 0, len(override.Order))
	for i, id := range override.Order {
		row := byID[id]
		row.Rank = i + 1
		row.Why = []string{"The commissioner placed this team here: " + override.Reason}
		ordered = append(ordered, row)
	}
	return ordered
}

func directoryIDs(d NativeDirectory) []string {
	ids := make([]string, len(d.Franchises))
	for i, seat := range d.Franchises {
		ids[i] = seat.Franchise
	}
	return ids
}
