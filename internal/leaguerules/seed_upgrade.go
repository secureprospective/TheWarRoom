package leaguerules

// seedUpgrade fills settings added after a league's preset was recorded. Steps are versioned and
// append only; each applies to a seed that holds its trigger field but none of the step's fields.
type seedUpgrade struct {
	version int
	trigger FieldPath
	values  func() RuleSet
	fields  []FieldPath
}

// seedUpgrades: version 1 (2026-10-26) records the scoring rule kinds as empty for a seed recorded
// before they existed, so an older league scores exactly as it did and has nothing to rule on.
func seedUpgrades() []seedUpgrade {
	return []seedUpgrade{{version: 1, trigger: "scoring.catalogue",
		fields: []FieldPath{"scoring.extras", "scoring.ranges", "scoring.scales", "scoring.totals", "scoring.lineUnit"},
		values: func() RuleSet {
			var r RuleSet
			emptyScoringKinds(&r.Scoring, "MFL league scoring settings; recorded empty for a league seeded before these settings")
			return r
		}}}
}

// UpgradeSeed returns a recorded preset with every later setting it predates, recorded as the step
// says, at the seed's own instant. Stored records are never rewritten; this runs on every read.
func UpgradeSeed(seed []Change) []Change {
	if len(seed) == 0 {
		return seed
	}
	present := make(map[FieldPath]bool, len(seed))
	for _, c := range seed {
		present[c.Field] = true
	}
	result := seed
	for _, step := range seedUpgrades() {
		if !present[step.trigger] || anyPresent(present, step.fields) {
			continue
		}
		values := step.values()
		for _, f := range ruleFields() {
			for _, path := range step.fields {
				if f.pathName() != path {
					continue
				}
				first := seed[0]
				result = append(result, Change{LeagueID: first.LeagueID, Field: path, New: f.read(values),
					Effective: first.Effective, RecordedAt: first.RecordedAt, Season: first.Season})
				present[path] = true
			}
		}
	}
	return result
}

func anyPresent(present map[FieldPath]bool, fields []FieldPath) bool {
	for _, path := range fields {
		if present[path] {
			return true
		}
	}
	return false
}
