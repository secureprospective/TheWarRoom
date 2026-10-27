package nativecheck

import (
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Starter is one player in a lineup: who, where he plays, and the points he scored.
type Starter struct {
	ID       string
	Position leaguerules.Position
	Points   leaguerules.Decimal
}

// LineupStatus is how a lineup stands against the league's limits.
type LineupStatus string

const (
	LineupValid    LineupStatus = "valid"
	LineupInvalid  LineupStatus = "invalid"
	LineupMissed   LineupStatus = "missed"
	LineupUnjudged LineupStatus = "unjudged"
)

// LineupProblem is one limit a lineup breaks.
type LineupProblem struct {
	Subject string `json:"subject"`
	Words   string `json:"words"`
}

// LineupVerdict is the plain-words result of judging one lineup.
type LineupVerdict struct {
	Status   LineupStatus    `json:"status"`
	Words    string          `json:"words"`
	Problems []LineupProblem `json:"problems"`
	// Flagged is true when a missed lineup is reported to the commissioner. Nothing else happens to it.
	Flagged bool `json:"flagged"`
	// Penalty says what an invalid lineup costs under this league's rule; empty for any other status.
	Penalty string `json:"penalty"`
}

// JudgeLineup judges a lineup against the position and group limits. A lineup that was never set is
// missed: it is flagged if the league flags them, and never penalised.
func JudgeLineup(rules leaguerules.LineupRules, submitted bool, starters []Starter) LineupVerdict {
	v := LineupVerdict{Problems: []LineupProblem{}}
	if !submitted {
		v.Status, v.Words = LineupMissed, "No lineup was set"
		switch {
		case rules.MissedFlag.State != leaguerules.IsKnown:
			v.Words += ". Needs a ruling: whether a missed lineup is flagged"
		case rules.MissedFlag.Value:
			v.Flagged = true
			v.Words += ". It is flagged to the commissioner; nothing is zeroed"
		}
		return v
	}
	if rules.Positions.State != leaguerules.IsKnown && rules.Groups.State != leaguerules.IsKnown {
		v.Status, v.Words = LineupUnjudged, "Needs a ruling: starting lineup limits"
		return v
	}
	v.Problems = lineupProblems(rules, starters)
	if len(v.Problems) == 0 {
		v.Status, v.Words = LineupValid, "Valid lineup"
		return v
	}
	v.Status = LineupInvalid
	subjects := make([]string, len(v.Problems))
	for i, p := range v.Problems {
		subjects[i] = p.Words
	}
	v.Words = "Invalid lineup: " + strings.Join(subjects, "; ")
	v.Penalty = penaltyWords(rules.InvalidPenalty)
	return v
}

// lineupProblems lists every limit the starters break: position by position, then the groups.
func lineupProblems(rules leaguerules.LineupRules, starters []Starter) []LineupProblem {
	problems := []LineupProblem{}
	counts, offense, defense := map[leaguerules.Position]int{}, 0, 0
	for _, s := range starters {
		counts[s.Position]++
		switch sideOf(s.Position) {
		case leaguerules.Offense:
			offense++
		case leaguerules.Defense:
			defense++
		case leaguerules.Total:
		}
	}
	if rules.Positions.State == leaguerules.IsKnown {
		problems = append(problems, positionProblems(rules.Positions.Value, counts)...)
	}
	if rules.Groups.State == leaguerules.IsKnown {
		groups := rules.Groups.Value
		for _, g := range []struct {
			group leaguerules.LineupGroup
			name  string
			n     int
		}{{leaguerules.Offense, "Offense", offense}, {leaguerules.Defense, "Defense", defense},
			{leaguerules.Total, "Total", len(starters)}} {
			if bounds, ok := groups[g.group]; ok {
				problems = appendBound(problems, g.name, g.n, bounds)
			}
		}
	}
	return problems
}

// positionProblems checks each limited position, then any starter at a position with no limit.
func positionProblems(limits map[leaguerules.Position]leaguerules.Bounds, counts map[leaguerules.Position]int) []LineupProblem {
	problems := []LineupProblem{}
	for _, position := range sortedPositions(limits) {
		problems = appendBound(problems, string(position), counts[position], limits[position])
	}
	for _, position := range sortedPositions(counts) {
		if _, ok := limits[position]; ok {
			continue
		}
		words := fmt.Sprintf("%s: %d starting, not a lineup position", position, counts[position])
		if position == "" {
			words = fmt.Sprintf("Position unknown: %d starting", counts[position])
		}
		problems = append(problems, LineupProblem{Subject: string(position), Words: words})
	}
	return problems
}

// sideOf is the football fact that splits a lineup: the kicker and the O-Line card are on neither side,
// so each counts toward the total only (the offense count is the skill players and their flex).
func sideOf(p leaguerules.Position) leaguerules.LineupGroup {
	switch p {
	case leaguerules.QB, leaguerules.RB, leaguerules.WR, leaguerules.TE:
		return leaguerules.Offense
	case leaguerules.DT, leaguerules.DE, leaguerules.LB, leaguerules.CB, leaguerules.S:
		return leaguerules.Defense
	case leaguerules.PK, leaguerules.OL:
		return leaguerules.Total
	}
	return leaguerules.Total
}

// appendBound adds a problem when the count falls outside bounds; an unset end is no limit.
func appendBound(problems []LineupProblem, name string, n int, b leaguerules.Bounds) []LineupProblem {
	low, high := b.Min, b.Max
	exact := low.State == leaguerules.IsKnown && high.State == leaguerules.IsKnown && low.Value == high.Value
	var words string
	switch {
	case low.State == leaguerules.IsKnown && n < low.Value && exact:
		words = fmt.Sprintf("%s: %d starting, needs exactly %d", name, n, low.Value)
	case low.State == leaguerules.IsKnown && n < low.Value:
		words = fmt.Sprintf("%s: %d starting, needs at least %d", name, n, low.Value)
	case high.State == leaguerules.IsKnown && n > high.Value && exact:
		words = fmt.Sprintf("%s: %d starting, needs exactly %d", name, n, high.Value)
	case high.State == leaguerules.IsKnown && n > high.Value:
		words = fmt.Sprintf("%s: %d starting, at most %d", name, n, high.Value)
	default:
		return problems
	}
	return append(problems, LineupProblem{Subject: name, Words: words})
}
