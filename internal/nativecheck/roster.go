// Package nativecheck judges a native league's rosters, lineups and reserve squads against the rules
// the commissioner set. It is pure: every input arrives as a value, so the same rules and players
// always give the same plain-words verdict. It describes what this league's rules allow; it never
// says what MFL would accept.
package nativecheck

import (
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// RosterPlayer is one player on a native franchise: who, and where he sits.
type RosterPlayer struct {
	Exceptions []DecisionException
	ID         string
	Status     domain.RosterStatus
}

// RosterVerdict is one franchise's roster against the league's size and position limits.
type RosterVerdict struct {
	Franchise string `json:"franchise"`
	// Counted is how many players the size limit counts: the active roster, plus IR and taxi only when
	// the league counts them.
	Counted int `json:"counted"`
	// Fits is true only when the size is judged and within its limits and no position limit is broken.
	Fits  bool   `json:"fits"`
	Words string `json:"words"`
	// Problems lists each position limit the roster breaks, in plain words.
	Problems []string `json:"problems"`
	// Notes say what could not be judged, and why.
	Notes []string `json:"notes"`
}

// CheckRoster counts a franchise's players against the roster rules. Positions come from facts and
// are only read when the league sets position limits.
func CheckRoster(rules leaguerules.RosterRules, franchise string, players []RosterPlayer, facts Facts, exceptions ...DecisionException) RosterVerdict {
	v := RosterVerdict{Franchise: franchise, Problems: []string{}, Notes: []string{}}
	if rules.ReserveExcluded.State != leaguerules.IsKnown {
		v.Words = "Needs a ruling: whether IR and taxi players count toward roster size"
		return v
	}
	label := "Active roster"
	if !rules.ReserveExcluded.Value {
		label = "Whole roster"
	}
	counted := countedPlayers(rules.ReserveExcluded.Value, players)
	v.Counted = len(counted)
	sizeOK := judgeSize(&v, label, rules.Limits)
	if why, allowed := ExceptionReason(exceptions, "roster.limits"); allowed {
		sizeOK, v.Words = true, why
	}
	positionsOK := judgePositions(&v, rules.PositionBounds, counted, facts)
	if why, allowed := ExceptionReason(exceptions, "roster.positionBounds"); allowed {
		positionsOK, v.Problems = true, nil
		v.Notes = append(v.Notes, why)
	}
	v.Fits = sizeOK && positionsOK
	return v
}

// countedPlayers keeps the players the size limit counts.
func countedPlayers(reserveExcluded bool, players []RosterPlayer) []RosterPlayer {
	counted := make([]RosterPlayer, 0, len(players))
	for _, p := range players {
		if p.Status == domain.RosterActive || !reserveExcluded {
			counted = append(counted, p)
		}
	}
	return counted
}

// judgeSize writes the headline and says whether the size is judged and within its limits.
func judgeSize(v *RosterVerdict, label string, limits leaguerules.Value[leaguerules.Bounds]) bool {
	if limits.State != leaguerules.IsKnown || (limits.Value.Min.State != leaguerules.IsKnown &&
		limits.Value.Max.State != leaguerules.IsKnown) {
		v.Words = "Needs a ruling: roster size limits"
		return false
	}
	low, high := limits.Value.Min, limits.Value.Max
	switch {
	case high.State == leaguerules.IsKnown && v.Counted > high.Value:
		v.Words = fmt.Sprintf("%s %d of %d: %d over", label, v.Counted, high.Value, v.Counted-high.Value)
		return false
	case low.State == leaguerules.IsKnown && v.Counted < low.Value:
		v.Words = fmt.Sprintf("%s %d of %d: %d under", label, v.Counted, low.Value, low.Value-v.Counted)
		return false
	}
	v.Words = "Within " + boundsPhrase(low, high)
	return true
}

// boundsPhrase reads a range whose ends may be unset.
func boundsPhrase(low, high leaguerules.Value[int]) string {
	switch {
	case low.State == leaguerules.IsKnown && high.State == leaguerules.IsKnown:
		return fmt.Sprintf("%d to %d", low.Value, high.Value)
	case low.State == leaguerules.IsKnown:
		return fmt.Sprintf("the minimum of %d", low.Value)
	default:
		return fmt.Sprintf("the maximum of %d", high.Value)
	}
}

// judgePositions checks the optional per-position limits and reports whether every one is met.
func judgePositions(v *RosterVerdict, bounds leaguerules.Value[map[leaguerules.Position]leaguerules.Bounds],
	counted []RosterPlayer, facts Facts) bool {
	if bounds.State != leaguerules.IsKnown {
		v.Notes = append(v.Notes, "Needs a ruling: roster position limits")
		return false
	}
	if len(bounds.Value) == 0 {
		return true
	}
	byPosition, unplaced := map[leaguerules.Position]int{}, 0
	for _, p := range counted {
		if f, ok := facts.Of(p.ID); ok && f.Position != "" {
			byPosition[f.Position]++
		} else {
			unplaced++
		}
	}
	if unplaced > 0 {
		v.Notes = append(v.Notes, fmt.Sprintf("Position is not held for %d players, so position limits are not fully judged", unplaced))
	}
	ok := true
	for _, position := range sortedPositions(bounds.Value) {
		if line := positionLine(position, byPosition[position], bounds.Value[position], unplaced > 0); line != "" {
			v.Problems = append(v.Problems, line)
			ok = false
		}
	}
	return ok && unplaced == 0
}

// positionLine says how a position breaks its limit, or is empty when it does not. With players whose
// position is not held, a shortfall is only a possible one, so only an excess is reported.
func positionLine(position leaguerules.Position, n int, b leaguerules.Bounds, partial bool) string {
	switch {
	case b.Max.State == leaguerules.IsKnown && n > b.Max.Value:
		return fmt.Sprintf("%s %d of %d: %d over", position, n, b.Max.Value, n-b.Max.Value)
	case !partial && b.Min.State == leaguerules.IsKnown && n < b.Min.Value:
		return fmt.Sprintf("%s %d of %d: %d under", position, n, b.Min.Value, b.Min.Value-n)
	}
	return ""
}

func sortedPositions[V any](m map[leaguerules.Position]V) []leaguerules.Position {
	keys := make([]leaguerules.Position, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b leaguerules.Position) int {
		if ra, rb := positionRank(a), positionRank(b); ra != rb {
			return ra - rb
		}
		return strings.Compare(string(a), string(b))
	})
	return keys
}

// DecisionException has already been matched to one decision's scope and instant by the ledger reader.
// The rule itself consumes it; callers must not maintain a parallel version of the decision.
type DecisionException struct {
	Rule   string
	Reason string
}

func ExceptionReason(exceptions []DecisionException, rule string) (string, bool) {
	for _, exception := range exceptions {
		if exception.Rule == rule && exception.Reason != "" {
			return exception.Reason, true
		}
	}
	return "", false
}
