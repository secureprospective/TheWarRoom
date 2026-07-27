package rulebook

import (
	"fmt"
	"sort"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
)

// diffConfigs reports every value difference between an active config and a freshly
// fetched candidate: scalar settings, the salary cap, scoring rules, and roster /
// starter limits. The result is deterministic (sorted by field) so the ChangeSet a
// commissioner reviews is stable across runs. It is comparison only — no value is
// interpreted or normalized, matching the store's pure-data contract.
func diffConfigs(old, cand league.RawConfig) []RuleDelta {
	var d []RuleDelta
	d = append(d, diffScalars(old, cand)...)
	d = append(d, diffScoring(old, cand)...)
	d = append(d, diffLimits("rosterLimit", old.RosterLimits, cand.RosterLimits)...)
	d = append(d, diffLimits("starterLimit", old.Starters.Positions, cand.Starters.Positions)...)
	sort.Slice(d, func(i, j int) bool { return d[i].Field < d[j].Field })
	return d
}

// diffScalars compares the flat single-value fields.
func diffScalars(old, cand league.RawConfig) []RuleDelta {
	pairs := []struct {
		field    string
		old, new string
	}{
		{"salaryCapAmount", old.SalaryCapAmount, cand.SalaryCapAmount},
		{"rosterSize", old.RosterSize, cand.RosterSize},
		{"taxiSquad", old.TaxiSquad, cand.TaxiSquad},
		{"injuredReserve", old.InjuredReserve, cand.InjuredReserve},
		{"keeperType", old.KeeperType, cand.KeeperType},
		{"usesSalaries", old.UsesSalaries, cand.UsesSalaries},
		{"usesContractYear", old.UsesContractYear, cand.UsesContractYear},
		{"startWeek", old.StartWeek, cand.StartWeek},
		{"endWeek", old.EndWeek, cand.EndWeek},
		{"lastRegularSeasonWeek", old.LastRegularSeasonWeek, cand.LastRegularSeasonWeek},
		{"includeTaxiWithSalary", old.IncludeTaxiWithSalary, cand.IncludeTaxiWithSalary},
		{"includeIRWithSalary", old.IncludeIRWithSalary, cand.IncludeIRWithSalary},
		{"includeTaxiWithContractYear", old.IncludeTaxiWithContractYear, cand.IncludeTaxiWithContractYear},
		{"starters.count", old.Starters.Count, cand.Starters.Count},
		{"starters.iop", old.Starters.IOPStarters, cand.Starters.IOPStarters},
		{"starters.idp", old.Starters.IDPStarters, cand.Starters.IDPStarters},
	}
	var out []RuleDelta
	for _, p := range pairs {
		if p.old != p.new {
			out = append(out, RuleDelta{Field: p.field, Kind: KindChanged, Old: p.old, New: p.new})
		}
	}
	return out
}

// diffScoring compares the position-additive scoring rules. Each rule is keyed by
// position group + event + range, so a points change, a new event, or a removed
// event is detected precisely.
func diffScoring(old, cand league.RawConfig) []RuleDelta {
	oldM := scoringMap(old.ScoringRules)
	candM := scoringMap(cand.ScoringRules)
	var out []RuleDelta
	for k, ov := range oldM {
		nv, ok := candM[k]
		switch {
		case !ok:
			out = append(out, RuleDelta{Field: "scoring:" + k, Kind: KindRemoved, Old: ov})
		case nv != ov:
			out = append(out, RuleDelta{Field: "scoring:" + k, Kind: KindChanged, Old: ov, New: nv})
		}
	}
	for k, nv := range candM {
		if _, ok := oldM[k]; !ok {
			out = append(out, RuleDelta{Field: "scoring:" + k, Kind: KindAdded, New: nv})
		}
	}
	return out
}

// scoringMap flattens position rule sets into a "positions/event[range]" -> points
// map. A duplicate key (same group+event+range twice) keeps the last — MFL does not
// emit duplicates, and the store does not interpret them.
func scoringMap(sets []league.PositionRuleSet) map[string]string {
	out := map[string]string{}
	for _, s := range sets {
		for _, r := range s.Rules {
			out[fmt.Sprintf("%s/%s[%s]", s.Positions, r.Event, r.Range)] = r.Points
		}
	}
	return out
}

// diffLimits compares two position-keyed limit lists (roster or starter) by name.
func diffLimits(prefix string, old, cand []league.PositionLimit) []RuleDelta {
	oldM := limitMap(old)
	candM := limitMap(cand)
	var out []RuleDelta
	for name, ov := range oldM {
		nv, ok := candM[name]
		switch {
		case !ok:
			out = append(out, RuleDelta{Field: prefix + ":" + name, Kind: KindRemoved, Old: ov})
		case nv != ov:
			out = append(out, RuleDelta{Field: prefix + ":" + name, Kind: KindChanged, Old: ov, New: nv})
		}
	}
	for name, nv := range candM {
		if _, ok := oldM[name]; !ok {
			out = append(out, RuleDelta{Field: prefix + ":" + name, Kind: KindAdded, New: nv})
		}
	}
	return out
}

// limitMap keys position limits by name.
func limitMap(in []league.PositionLimit) map[string]string {
	out := make(map[string]string, len(in))
	for _, p := range in {
		out[p.Name] = p.Limit
	}
	return out
}
