package nativecheck

import (
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// PenaltyResult is a lineup's points after the league's invalid-lineup penalty.
type PenaltyResult struct {
	// Starters carries each starter's points after the penalty, in the order they were given.
	Starters []Starter
	// Zeroed is the starter whose points were set to zero; empty when none were.
	Zeroed string
	Words  string
}

// penaltyWords says, in advance, what an invalid lineup costs.
func penaltyWords(p leaguerules.Value[leaguerules.Penalty]) string {
	if p.State != leaguerules.IsKnown {
		return "Needs a ruling: the invalid lineup penalty"
	}
	switch p.Value {
	case leaguerules.ZeroTopStarter:
		return "The highest-scoring starter's points are set to zero"
	case leaguerules.ZeroSlot:
		return "Needs a ruling: which starting slot is zeroed"
	case leaguerules.NoPenalty:
		return "No penalty"
	}
	return "Needs a ruling: the invalid lineup penalty"
}

// ApplyPenalty applies the league's penalty to the starters of a lineup already judged invalid. The
// highest-scoring starter is zeroed; of two with the same points, the lower player id.
func ApplyPenalty(p leaguerules.Value[leaguerules.Penalty], starters []Starter) PenaltyResult {
	result := PenaltyResult{Starters: slices.Clone(starters), Words: penaltyWords(p)}
	if p.State != leaguerules.IsKnown || p.Value != leaguerules.ZeroTopStarter || len(starters) == 0 {
		return result
	}
	top := 0
	for i, s := range result.Starters {
		best := result.Starters[top]
		if s.Points > best.Points || (s.Points == best.Points && strings.Compare(s.ID, best.ID) < 0) {
			top = i
		}
	}
	result.Zeroed = result.Starters[top].ID
	result.Starters[top].Points = 0
	return result
}
