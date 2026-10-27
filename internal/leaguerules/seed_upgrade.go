package leaguerules

import (
	"bytes"
	"maps"
	"slices"
)

// seedUpgrade fills or replaces settings in a league's recorded preset. Steps are versioned and
// append only; each decides from the seed as recorded, before any step ran, whether it applies.
type seedUpgrade struct {
	version int
	applies func(recorded map[FieldPath]Setting) bool
	values  func() RuleSet
	fields  []FieldPath
}

func scoringKindPaths() []FieldPath {
	return []FieldPath{"scoring.extras", "scoring.ranges", "scoring.scales", "scoring.totals", "scoring.lineUnit"}
}

// seedUpgrades:
//   - version 1 (2026-10-26) records the scoring rule kinds as empty for a seed recorded before they
//     existed, so an older league scores exactly as it did and has nothing to rule on.
//   - version 2 (2026-10-26, owner ruling) brings a Legacy seed recorded before the kinds up to MFL
//     parity: the catalogue and kinds a new Legacy league gets (#UT other returns, #BF/#BP IDP
//     blocked-kick TDs, MFL's per-rule ranges, the other kinds empty), so its scoring equals a new
//     Legacy league's. A seed is Legacy when its catalogue is exactly the
//     Legacy catalogue as recorded then; any other league keeps its own catalogue.
func seedUpgrades() []seedUpgrade {
	return []seedUpgrade{
		{version: 1, fields: scoringKindPaths(),
			applies: func(recorded map[FieldPath]Setting) bool {
				return recorded["scoring.catalogue"] != nil && !anyRecorded(recorded, scoringKindPaths())
			},
			values: func() RuleSet {
				var r RuleSet
				emptyScoringKinds(&r.Scoring, "MFL league scoring settings; recorded empty for a league seeded before these settings")
				return r
			}},
		{version: 2, fields: append([]FieldPath{"scoring.catalogue"}, scoringKindPaths()...),
			applies: func(recorded map[FieldPath]Setting) bool {
				return !anyRecorded(recorded, scoringKindPaths()) && sameStored(recorded["scoring.catalogue"],
					cited[map[StatID]Stat](legacyCatalogueBeforeKinds(), "MFL league scoring settings"))
			},
			values: LegacyNFL},
		{version: 3, fields: []FieldPath{"taxi.autoPromote", "ir.autoReturn"},
			applies: func(recorded map[FieldPath]Setting) bool {
				return !anyRecorded(recorded, []FieldPath{"taxi.autoPromote", "ir.autoReturn"})
			},
			values: LegacyNFL},
	}
}

// legacyCatalogueBeforeKinds is the Legacy catalogue a league recorded before the scoring kinds:
// today's, less the three kick-return touchdown rows added with them.
func legacyCatalogueBeforeKinds() map[StatID]Stat {
	catalogue := maps.Clone(legacyCatalogue())
	for _, id := range []StatID{"other_return_td", "blocked_fg_td", "blocked_punt_td"} {
		delete(catalogue, id)
	}
	return catalogue
}

func sameStored(recorded, want Setting) bool {
	if recorded == nil {
		return false
	}
	a, err := EncodeStored(recorded)
	if err != nil {
		return false
	}
	b, err := EncodeStored(want)
	return err == nil && bytes.Equal(a, b)
}

// UpgradeSeed returns a recorded preset with every step that applies to it, each value recorded at
// the seed's own instant. Stored records are never rewritten; this runs on every read.
func UpgradeSeed(seed []Change) []Change {
	if len(seed) == 0 {
		return seed
	}
	recorded := make(map[FieldPath]Setting, len(seed))
	for _, c := range seed {
		recorded[c.Field] = c.New
	}
	result := seed
	for _, step := range seedUpgrades() {
		if !step.applies(recorded) {
			continue
		}
		values := step.values()
		for _, f := range ruleFields() {
			for _, path := range step.fields {
				if f.pathName() == path {
					result = setSeedValue(result, Change{LeagueID: seed[0].LeagueID, Field: path, New: f.read(values),
						Effective: seed[0].Effective, RecordedAt: seed[0].RecordedAt, Season: seed[0].Season})
				}
			}
		}
	}
	return result
}

// setSeedValue replaces the seed's value for the change's field, or appends it, on a copy so the
// caller's recorded seed is never altered.
func setSeedValue(seed []Change, c Change) []Change {
	result := slices.Clone(seed)
	if i := slices.IndexFunc(result, func(s Change) bool { return s.Field == c.Field }); i >= 0 {
		result[i] = c
		return result
	}
	return append(result, c)
}

func anyRecorded(recorded map[FieldPath]Setting, fields []FieldPath) bool {
	for _, path := range fields {
		if recorded[path] != nil {
			return true
		}
	}
	return false
}
