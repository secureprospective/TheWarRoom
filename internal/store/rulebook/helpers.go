package rulebook

import (
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
)

// cloneConfig deep-copies every nested slice of a config so a value returned to a
// caller shares no backing array with the store's in-memory active snapshot. The
// scalar fields are value-copied by the struct assignment; only the slices alias.
func cloneConfig(c league.RawConfig) league.RawConfig {
	c.ScoringRules = cloneScoring(c.ScoringRules)
	c.RosterLimits = cloneLimits(c.RosterLimits)
	c.Starters.Positions = cloneLimits(c.Starters.Positions)
	c.Franchises = append([]league.Franchise(nil), c.Franchises...)
	return c
}

// FranchiseNames returns the active config's franchise directory as an id->display-name
// map, for the operator UI's rail and trade dropdowns. Only non-empty names are
// included, so a caller can range the map and fall back to the id for any franchise
// absent from it (an older config version stored before franchises were captured, or a
// blank MFL name). Reads are snapshot-consistent under the store's read lock.
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

// settingsMap projects the scalar settings of a config into a key->value map for
// GetSetting. Structured fields (scoring, roster limits, starters) are excluded.
func settingsMap(c league.RawConfig) map[string]string {
	return map[string]string{
		"rosterSize":            c.RosterSize,
		"taxiSquad":             c.TaxiSquad,
		"injuredReserve":        c.InjuredReserve,
		"keeperType":            c.KeeperType,
		"usesSalaries":          c.UsesSalaries,
		"usesContractYear":      c.UsesContractYear,
		"startWeek":             c.StartWeek,
		"endWeek":               c.EndWeek,
		"lastRegularSeasonWeek": c.LastRegularSeasonWeek,
		"includeTaxiWithSalary": c.IncludeTaxiWithSalary,
		"includeIRWithSalary":   c.IncludeIRWithSalary,
	}
}

// capPercent parses a settingsMap-style percentage string (MFL's "100", "0", or an
// empty/unset value) to a 0-100 float, defaulting to 100 (no discount) when the
// raw value is empty or unparseable — matching the historical behavior (every
// rostered player counted 100% toward cap) for configs stored before this field
// existed.
func capPercent(raw string) float64 {
	v := strings.TrimSpace(raw)
	if v == "" {
		return 100
	}
	pct, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 100
	}
	return pct
}

// TaxiCapPercent returns the pct of a taxi-squad player's cap-counting salary that
// counts toward the franchise's cap total, override-aware (GetSetting).
func (s *Store) TaxiCapPercent() float64 {
	v, _ := s.GetSetting("includeTaxiWithSalary")
	return capPercent(v)
}

// IRCapPercent returns the pct of an IR player's cap-counting salary that counts
// toward the franchise's cap total, override-aware (GetSetting).
func (s *Store) IRCapPercent() float64 {
	v, _ := s.GetSetting("includeIRWithSalary")
	return capPercent(v)
}
