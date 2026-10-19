package commissioner

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// Starter is one player who counts for a franchise in a week.
type Starter struct {
	Player string `json:"player"`
	Name   string `json:"name"`
}

// WeekLineup is the lineup that counts for one franchise in one week. Set is false until a lineup exists.
type WeekLineup struct {
	Set      bool
	Starters []Starter
}

// ResultSources supplies what a week's scores are made of. Lineup and PlayerScore read the given ledger, so the
// same week can be scored as it would be after a proposed change.
type ResultSources interface {
	Lineup(ctx context.Context, records []Record, franchise string, week int) (WeekLineup, error)
	PlayerScore(ctx context.Context, records []Record, player string, week int) (NativePlayerScore, error)
}

// StarterScore is one starter's contribution to a franchise's week.
type StarterScore struct {
	Player   string               `json:"player"`
	Name     string               `json:"name"`
	Position leaguerules.Position `json:"position,omitempty"`
	Words    string               `json:"words"`
	Points   *leaguerules.Decimal `json:"points,omitempty"`
}

// FranchiseWeekScore is a franchise's score for one week: its starters' points plus any franchise
// adjustment. Points is present only when every starter's score is complete.
type FranchiseWeekScore struct {
	Franchise  string               `json:"franchise"`
	Name       string               `json:"name"`
	Status     string               `json:"status"`
	Points     *leaguerules.Decimal `json:"points,omitempty"`
	Words      string               `json:"words"`
	Starters   []StarterScore       `json:"starters"`
	Adjustment leaguerules.Decimal  `json:"adjustment"`
	Changes    []ScoreChange        `json:"changes"`
	// Lineup is the league's judgement of the lineup: a missed one is flagged and never penalised, an invalid
	// one costs what the league's penalty rule says. Zeroed names the starter whose points that took away.
	Lineup nativecheck.LineupVerdict `json:"lineup"`
	Zeroed string                    `json:"zeroed,omitempty"`
}

func franchiseWeekScore(ctx context.Context, records []Record, src ResultSources, rules leaguerules.LineupRules,
	seat NativeDirectorySeat, season, week int) (FranchiseWeekScore, error) {
	out := FranchiseWeekScore{Franchise: seat.Franchise, Name: seat.Name, Status: "Incomplete",
		Starters: []StarterScore{}, Changes: []ScoreChange{}}
	lineup, err := src.Lineup(ctx, records, seat.Franchise, week)
	if err != nil {
		return out, fmt.Errorf("read %s lineup: %w", seat.Name, err)
	}
	if !lineup.Set || len(lineup.Starters) == 0 {
		out.Lineup = nativecheck.JudgeLineup(rules, false, nil)
		out.Words = fmt.Sprintf("%s has no lineup set for week %d", seat.Name, week)
		if out.Lineup.Flagged {
			out.Words += "; it is flagged to the commissioner as a missed lineup, and nothing is zeroed"
		}
		out.Words += "."
		return out, nil
	}
	changes, err := FranchiseScoreChanges(records, seat.Franchise, season, week)
	if err != nil {
		return out, err
	}
	out.Adjustment, out.Changes = changes.Delta, changes.Changes
	total, missing, err := out.addStarters(ctx, records, src, lineup, week)
	if err != nil {
		return out, err
	}
	if len(missing) != 0 {
		out.Lineup = nativecheck.LineupVerdict{Status: nativecheck.LineupUnjudged, Problems: []nativecheck.LineupProblem{},
			Words: "The lineup is judged once every starter's score is complete"}
		out.Words = fmt.Sprintf("%s's score is incomplete: %s", seat.Name, strings.Join(missing, "; "))
		return out, nil
	}
	if total, err = out.judge(rules, total); err != nil {
		return out, err
	}
	if total, err = addScoreDelta(total, changes.Delta); err != nil {
		return out, err
	}
	out.Status, out.Points = "Complete", &total
	out.Words = fmt.Sprintf("%s scored %s points", seat.Name, scorePointsWords(total))
	if out.Zeroed != "" {
		out.Words += fmt.Sprintf("; the lineup is invalid, so %s counts for nothing", out.zeroedName())
	}
	return out, nil
}

func (out FranchiseWeekScore) zeroedName() string {
	for _, s := range out.Starters {
		if s.Player == out.Zeroed {
			return s.Name
		}
	}
	return "a starter"
}

// judge holds complete starters to the league's lineup limits. An invalid lineup loses what the penalty
// rule takes, through the same function the native lineup check uses; any other lineup counts in full.
func (out *FranchiseWeekScore) judge(rules leaguerules.LineupRules, total leaguerules.Decimal) (leaguerules.Decimal, error) {
	starters := make([]nativecheck.Starter, len(out.Starters))
	for i, s := range out.Starters {
		if s.Points == nil || s.Position == "" {
			out.Lineup = nativecheck.LineupVerdict{Status: nativecheck.LineupUnjudged, Problems: []nativecheck.LineupProblem{},
				Words: "Can't tell: where a starter plays is not held"}
			return total, nil
		}
		starters[i] = nativecheck.Starter{ID: s.Player, Position: s.Position, Points: *s.Points}
	}
	out.Lineup = nativecheck.JudgeLineup(rules, true, starters)
	if out.Lineup.Status != nativecheck.LineupInvalid {
		return total, nil
	}
	penalised := nativecheck.ApplyPenalty(rules.InvalidPenalty, starters)
	if penalised.Zeroed == "" {
		return total, nil
	}
	out.Zeroed = penalised.Zeroed
	var kept leaguerules.Decimal
	for i, s := range penalised.Starters {
		points := s.Points
		out.Starters[i].Points = &points
		if s.ID == penalised.Zeroed {
			out.Starters[i].Words += "; zeroed because the lineup is invalid"
		}
		var err error
		if kept, err = addScoreDelta(kept, points); err != nil {
			return 0, err
		}
	}
	return kept, nil
}

// addStarters scores each starter once and sums those that are complete.
func (out *FranchiseWeekScore) addStarters(ctx context.Context, records []Record, src ResultSources, lineup WeekLineup,
	week int) (leaguerules.Decimal, []string, error) {
	var total leaguerules.Decimal
	missing, seen := []string{}, map[string]bool{}
	for _, starter := range lineup.Starters {
		if seen[starter.Player] {
			return 0, nil, fmt.Errorf("a lineup lists player %s twice", starter.Player)
		}
		seen[starter.Player] = true
		name := starter.Name
		if name == "" {
			name = "player " + starter.Player
		}
		score, err := src.PlayerScore(ctx, records, starter.Player, week)
		if err != nil {
			return 0, nil, fmt.Errorf("score %s: %w", name, err)
		}
		item := StarterScore{Player: starter.Player, Name: name, Words: score.Words, Position: score.Position}
		if score.Status != "Complete" || score.Points == nil {
			missing = append(missing, name+": "+score.Words)
		} else {
			item.Points = score.Points
			if total, err = addScoreDelta(total, *score.Points); err != nil {
				return 0, nil, err
			}
		}
		out.Starters = append(out.Starters, item)
	}
	return total, missing, nil
}
