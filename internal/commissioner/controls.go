package commissioner

import (
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// ControlField is one league rule a census row owns, named the way the rules screen names it.
type ControlField struct {
	Field string `json:"field"`
	Label string `json:"label"`
}

// ControlEntry is one census row as the landing needs it from Go: where it sits, whether a person can
// use it today, the rules it owns and, for a row that is only partly there, what already works.
// The question and name come from the endpoint registry, which is generated from the census.
type ControlEntry struct {
	ID     string         `json:"id"`
	Built  bool           `json:"built"`
	Note   string         `json:"note"`
	Fields []ControlField `json:"fields"`
}
type ControlGroup struct {
	ID       string         `json:"id"`
	Label    string         `json:"label"`
	Controls []ControlEntry `json:"controls"`
}
type Controls struct {
	Groups []ControlGroup `json:"groups"`
}

// controlGroups gives the census groups the landing shows, each by its first and last row (spec §16.7;
// group F is Playoffs, and the ledger row stands apart from League ops).
func controlGroups() []struct {
	id, label, first, last string
} {
	return []struct {
		id, label, first, last string
	}{
		{"constitution", "Constitution", "CM-001", "CM-017"}, {"people", "People", "CM-018", "CM-022"},
		{"calendar", "Calendar", "CM-023", "CM-026"}, {"season", "Season ops", "CM-027", "CM-039"},
		{"draft", "Draft", "CM-040", "CM-045"}, {"playoffs", "Playoffs", "CM-046", "CM-050"},
		{"ops", "League ops", "CM-051", "CM-055"}, {"ledger", "Ledger", "CM-056", "CM-056"},
	}
}

// partlyThere are rows that own rules a person can set today although the thing the row asks about is
// not built. They are not built, and say what does work.
func partlyThere() map[string]string {
	return map[string]string{
		"CM-015": "The number of regular season weeks and the schedule pattern can be set now.",
		"CM-027": "The rollover date can be set now.",
		"CM-039": "A recorded change can be undone now, from the ledger.",
	}
}

// otherwiseBuilt are rows with a working screen that owns no rule: roles and the ledger.
func otherwiseBuilt() []string { return []string{"CM-021", "CM-056"} }

func ownedFields(control string) []ControlField {
	fields := []ControlField{}
	for _, f := range leaguerules.Catalogue() {
		if owner, err := ownerFor(f.Path); err == nil && owner.control == control {
			fields = append(fields, ControlField{Field: string(f.Path), Label: f.Label})
		}
	}
	return fields
}

// controlBuilt says whether a person can do, today, what the row's question asks.
func controlBuilt(id string, fields []ControlField) bool {
	if _, partly := partlyThere()[id]; partly {
		return false
	}
	return len(fields) > 0 || slices.Contains(otherwiseBuilt(), id)
}

// ControlsReading lists every census row, grouped, in census order.
func ControlsReading() Controls {
	out := Controls{Groups: []ControlGroup{}}
	census := governance.CensusControls()
	for _, g := range controlGroups() {
		group := ControlGroup{ID: g.id, Label: g.label, Controls: []ControlEntry{}}
		for _, row := range census {
			if row.ID < g.first || row.ID > g.last {
				continue
			}
			fields := ownedFields(row.ID)
			group.Controls = append(group.Controls, ControlEntry{ID: row.ID, Built: controlBuilt(row.ID, fields),
				Note: partlyThere()[row.ID], Fields: fields})
		}
		out.Groups = append(out.Groups, group)
	}
	return out
}
