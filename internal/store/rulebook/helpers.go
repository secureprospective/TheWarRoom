package rulebook

import "github.com/secureprospective/TheWarRoom/internal/ingestion/league"

// cloneConfig deep-copies every nested slice of a config so a value returned to a
// caller shares no backing array with the store's in-memory active snapshot. The
// scalar fields are value-copied by the struct assignment; only the slices alias.
func cloneConfig(c league.RawConfig) league.RawConfig {
	c.ScoringRules = cloneScoring(c.ScoringRules)
	c.RosterLimits = cloneLimits(c.RosterLimits)
	c.Starters.Positions = cloneLimits(c.Starters.Positions)
	return c
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
	}
}
