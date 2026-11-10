package commissioner

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// RawRecordedResults holds the source's final team results, not player scoring evidence.
// CutoverWeek is the first week scored by the app, including a week still in play at import.
type RawRecordedResults struct {
	Season      int                  `json:"season"`
	CutoverWeek int                  `json:"cutoverWeek"`
	Teams       []RecordedTeamResult `json:"teams"`
}

type RecordedTeamResult struct {
	Franchise string              `json:"franchise"`
	Week      int                 `json:"week"`
	Points    leaguerules.Decimal `json:"points"`
	Result    string              `json:"result,omitempty"`
}

func (r RawRecordedResults) Validate() error {
	if r.Season < 1920 || r.Season > 9999 || r.CutoverWeek < 1 || r.CutoverWeek > 23 {
		return fmt.Errorf("recorded results need a season and a cutover week between 1 and 23")
	}
	seen := map[struct {
		franchise string
		week      int
	}]bool{}
	for _, team := range r.Teams {
		key := struct {
			franchise string
			week      int
		}{team.Franchise, team.Week}
		if team.Franchise == "" || team.Week < 1 || team.Week >= r.CutoverWeek || seen[key] {
			return fmt.Errorf("recorded results need unique franchises and weeks before cutover")
		}
		if team.Result != "" && team.Result != "W" && team.Result != "L" && team.Result != "T" {
			return fmt.Errorf("recorded result must be W, L or T")
		}
		seen[key] = true
	}
	return nil
}

// RecordedResults reads the latest active evidence for this season. A later schedule
// edit without evidence does not erase the imported results or move the boundary.
func RecordedResults(records []Record, season int) *RawRecordedResults {
	active := Replay(records).Active
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		if !active[r.Receipt.Entry] || r.Command.Command != ScheduleGenerate || r.Command.Args.Schedule == nil {
			continue
		}
		held := r.Command.Args.Schedule.Recorded
		if held != nil && held.Season == season {
			return held
		}
	}
	return nil
}

func recordedWeekScore(records []Record, seat NativeDirectorySeat, season, week int) (FranchiseWeekScore, bool) {
	held := RecordedResults(records, season)
	if held == nil || week >= held.CutoverWeek {
		return FranchiseWeekScore{}, false
	}
	out := FranchiseWeekScore{Franchise: seat.Franchise, Name: seat.Name, Status: "Incomplete",
		Words:    "The source's recorded result is not held for this pre-cutover week",
		Starters: []StarterScore{}, Changes: []ScoreChange{}}
	for _, team := range held.Teams {
		if team.Franchise == seat.Franchise && team.Week == week {
			points := team.Points
			out.Status, out.Points, out.RecordedResult = "Complete", &points, team.Result
			out.Words = fmt.Sprintf("%s scored %s points as recorded before cutover", seat.Name, scorePointsWords(points))
			return out, true
		}
	}
	return out, true
}

// RecordedStarters lists, for each franchise whose week is the source's recorded result, who started and what
// each starter scores under the league's rules. The franchise's total stays the recorded one: the starters
// explain the week, they never re-decide it. A lineup that is not held or cannot be scored is left unlisted.
func RecordedStarters(ctx context.Context, records []Record, src ResultSources, results *NativeResults) {
	for i := range results.Scores {
		score := &results.Scores[i]
		if score.RecordedResult == "" {
			continue
		}
		lineup, err := src.Lineup(ctx, records, score.Franchise, results.Week)
		if err != nil || !lineup.Set {
			continue
		}
		listed := FranchiseWeekScore{Starters: []StarterScore{}}
		if _, _, err := listed.addStarters(ctx, records, src, lineup, results.Week); err == nil {
			score.Starters = listed.Starters
		}
	}
}

func checkRecordedAppend(held RawRecordedResults, records []Record) error {
	rules, err := resolveRecords(records)
	if err != nil {
		return err
	}
	if held.Season != rules.Season {
		return fmt.Errorf("recorded results must match the current season")
	}
	if previous := RecordedResults(records, held.Season); previous != nil && previous.CutoverWeek != held.CutoverWeek {
		return fmt.Errorf("the recorded scoring boundary cannot move on refresh")
	}
	franchises := map[string]bool{}
	for _, id := range LeagueFranchises(records) {
		franchises[id] = true
	}
	for _, team := range held.Teams {
		if !franchises[team.Franchise] {
			return fmt.Errorf("recorded results name an unknown franchise")
		}
	}
	return nil
}

// The source may have settled a level score using its own recorded decision.
func applyRecordedVerdict(game *NativeGame, home, away string) bool {
	switch {
	case home == "W" && away == "L":
		game.Status, game.Winner = GameFinal, game.Home
	case home == "L" && away == "W":
		game.Status, game.Winner = GameFinal, game.Away
	case home == "T" && away == "T":
		game.Status = GameTie
	default:
		return false
	}
	game.Words = fmt.Sprintf("%s %s, %s %s, as recorded before cutover (%s–%s points)",
		game.HomeName, home, game.AwayName, away, scorePointsWords(*game.HomePoints), scorePointsWords(*game.AwayPoints))
	return true
}
