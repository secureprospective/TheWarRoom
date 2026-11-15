package commissioner

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// StatMove is one statistic whose points differ between the last live line and the counted one.
type StatMove struct {
	Stat  leaguerules.StatID  `json:"stat"`
	Label string              `json:"label"`
	Was   leaguerules.Decimal `json:"was"`
	Now   leaguerules.Decimal `json:"now"`
}

// StatRevision says how the final statistics moved a player's points off the last live line this app
// held. It is read from held evidence only and is never a result of its own.
type StatRevision struct {
	Stats []StatMove          `json:"stats"`
	Was   leaguerules.Decimal `json:"was"`
	Now   leaguerules.Decimal `json:"now"`
}

// lastLive is the newest live line held for a week whose counted line is the final feed's.
func lastLive(versions []source.LineRevision, counted *source.LineRevision) *source.LineRevision {
	if counted == nil || counted.Held.Authority != source.NFLVerse {
		return nil
	}
	for i := len(versions) - 1; i >= 0; i-- {
		v := &versions[i]
		if v.Held.Week != nil && v.Held.Authority != source.NFLVerse && v.Held.Authority != source.MFLLive {
			return v
		}
	}
	return nil
}

// withRevision compares a complete score with the same rules applied to the last live line. A line that
// needs a ruling, or one whose points did not move, has no revision.
func (out NativePlayerScore) withRevision(rules leaguerules.ScoringRules, live *source.LineRevision,
	line fantasyscore.StatLine, delta leaguerules.Decimal) (NativePlayerScore, error) {
	if live == nil || out.Status != "Complete" || out.Points == nil {
		return out, nil
	}
	before, was, err := fantasyscore.Score(rules, out.Position, live.Held.Week.Line)
	if err != nil {
		return out, fmt.Errorf("count the last live statistics: %w", err)
	}
	if before.State != leaguerules.IsKnown {
		return out, nil
	}
	total, err := addScoreDelta(before.Value, delta)
	if err != nil || total == *out.Points {
		return out, err
	}
	_, now, err := fantasyscore.Score(rules, out.Position, line)
	if err != nil {
		return out, fmt.Errorf("count held player statistics: %w", err)
	}
	out.Revision = &StatRevision{Stats: statMoves(was, now), Was: total, Now: *out.Points}
	return out, nil
}

type statSide struct {
	label            string
	quantity, points leaguerules.Decimal
}

func statSides(items []fantasyscore.LineItem) map[leaguerules.StatID]statSide {
	out := map[leaguerules.StatID]statSide{}
	for _, item := range items {
		side := out[item.Stat]
		side.label, side.quantity = item.Label, item.Quantity
		if !item.Unused && item.Points.State == leaguerules.IsKnown {
			side.points += item.Points.Value
		}
		out[item.Stat] = side
	}
	return out
}

// statMoves lists the statistics whose points differ, in statistic order.
func statMoves(was, now []fantasyscore.LineItem) []StatMove {
	before, after := statSides(was), statSides(now)
	moves := []StatMove{}
	for id, a := range after {
		if b := before[id]; b.points != a.points {
			moves = append(moves, StatMove{Stat: id, Label: a.label, Was: b.quantity, Now: a.quantity})
		}
	}
	for id, b := range before {
		if _, held := after[id]; !held && b.points != 0 {
			moves = append(moves, StatMove{Stat: id, Label: b.label, Was: b.quantity})
		}
	}
	slices.SortFunc(moves, func(a, b StatMove) int { return cmp.Compare(a.Stat, b.Stat) })
	return moves
}
