package rulebook

import (
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
)

// cloneConfig deep-copies the slices so a caller never shares a backing array with the
// in-memory snapshot.
func cloneConfig(c league.RawConfig) league.RawConfig {
	c.ScoringRules = cloneScoring(c.ScoringRules)
	c.RosterLimits = cloneLimits(c.RosterLimits)
	c.Starters.Positions = cloneLimits(c.Starters.Positions)
	c.Franchises = append([]league.Franchise(nil), c.Franchises...)
	return c
}

// FranchiseNames returns the active franchise directory as id -> name. Blank names are left
// out, so callers fall back to the id.
func (s *Store) FranchiseNames() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.active.Franchises))
	for _, f := range s.active.Franchises {
		if f.Name != "" {
			out[f.ID] = f.Name
		}
	}
	return out
}

// cloneScoring deep-copies the position rule sets and each block's rule slice.
func cloneScoring(in []league.PositionRuleSet) []league.PositionRuleSet {
	if in == nil {
		return nil
	}
	out := make([]league.PositionRuleSet, len(in))
	for i, set := range in {
		out[i] = league.PositionRuleSet{
			Positions: set.Positions,
			Rules:     append([]league.ScoringRule(nil), set.Rules...),
		}
	}
	return out
}

// cloneLimits copies a position-limit slice (elements are value types).
func cloneLimits(in []league.PositionLimit) []league.PositionLimit {
	if in == nil {
		return nil
	}
	return append([]league.PositionLimit(nil), in...)
}

// settingsMap projects a config's scalar settings into a key -> value map.
func settingsMap(c league.RawConfig) map[string]string {
	return map[string]string{
		"rosterSize":                  c.RosterSize,
		"taxiSquad":                   c.TaxiSquad,
		"injuredReserve":              c.InjuredReserve,
		"keeperType":                  c.KeeperType,
		"usesSalaries":                c.UsesSalaries,
		"usesContractYear":            c.UsesContractYear,
		"startWeek":                   c.StartWeek,
		"endWeek":                     c.EndWeek,
		"lastRegularSeasonWeek":       c.LastRegularSeasonWeek,
		"includeTaxiWithSalary":       c.IncludeTaxiWithSalary,
		"includeIRWithSalary":         c.IncludeIRWithSalary,
		"includeTaxiWithContractYear": c.IncludeTaxiWithContractYear,
	}
}

// capPercent parses an MFL percentage ("100", "0" or empty) into 0-100. Empty or unparseable
// means 100, the behavior before the field existed. It clamps because a setting override is
// not range-checked.
func capPercent(raw string) float64 {
	v := strings.TrimSpace(raw)
	if v == "" {
		return 100
	}
	pct, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 100
	}
	switch {
	case pct < 0:
		return 0
	case pct > 100:
		return 100
	default:
		return pct
	}
}

// TaxiCapPercent is the share of a taxi player's salary that counts against the cap.
func (s *Store) TaxiCapPercent() float64 {
	v, _ := s.GetSetting("includeTaxiWithSalary")
	return capPercent(v)
}

// IRCapPercent is the share of an IR player's salary that counts against the cap.
func (s *Store) IRCapPercent() float64 {
	v, _ := s.GetSetting("includeIRWithSalary")
	return capPercent(v)
}
