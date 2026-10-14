package commissioner

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
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

// ResultSources supplies what a week's scores are made of and the ledger does not hold. PlayerScore reads
// the given ledger, so the same week can be scored as it would be after a proposed change.
type ResultSources interface {
	Lineup(ctx context.Context, franchise string, week int) (WeekLineup, error)
	PlayerScore(ctx context.Context, records []Record, player string, week int) (NativePlayerScore, error)
}

// StarterScore is one starter's contribution to a franchise's week.
type StarterScore struct {
	Player string               `json:"player"`
	Name   string               `json:"name"`
	Words  string               `json:"words"`
	Points *leaguerules.Decimal `json:"points,omitempty"`
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
}

func franchiseWeekScore(ctx context.Context, records []Record, src ResultSources, seat NativeDirectorySeat,
	season, week int) (FranchiseWeekScore, error) {
	out := FranchiseWeekScore{Franchise: seat.Franchise, Name: seat.Name, Status: "Incomplete",
		Starters: []StarterScore{}, Changes: []ScoreChange{}}
	lineup, err := src.Lineup(ctx, seat.Franchise, week)
	if err != nil {
		return out, fmt.Errorf("read %s lineup: %w", seat.Name, err)
	}
	if !lineup.Set || len(lineup.Starters) == 0 {
		out.Words = fmt.Sprintf("%s has no lineup set for week %d.", seat.Name, week)
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
		out.Words = fmt.Sprintf("%s's score is incomplete: %s", seat.Name, strings.Join(missing, "; "))
		return out, nil
	}
	if total, err = addScoreDelta(total, changes.Delta); err != nil {
		return out, err
	}
	out.Status, out.Points = "Complete", &total
	out.Words = fmt.Sprintf("%s scored %s points", seat.Name, scorePointsWords(total))
	return out, nil
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
		item := StarterScore{Player: starter.Player, Name: name, Words: score.Words}
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
