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

// otherwiseBuilt lists controls with recorded acts or app operations as well as rule settings.
func otherwiseBuilt() []string {
	return []string{"CM-015", "CM-018", "CM-019", "CM-020", "CM-021", "CM-022", "CM-026", "CM-027",
		"CM-028", "CM-029", "CM-030", "CM-031", "CM-032", "CM-033", "CM-034", "CM-035", "CM-036", "CM-037",
		"CM-038", "CM-039", "CM-042", "CM-043", "CM-044", "CM-045", "CM-046", "CM-047", "CM-052", "CM-053",
		"CM-054", "CM-056"}
}

// builtNotes are what a person should know about a row that works, so it never promises more.
func builtNotes() map[string]string {
	return map[string]string{
		"CM-043": "Draft room: skip, choose, add or change picks; pause or set the clock.",
		"CM-044": "Restore a draft to an earlier point from Trade Floor › Draft room. Nothing is erased.",
		"CM-045": "A draft closes itself at its last pick; the Draft room can close it sooner.",
		"CM-051": "The app tells owners inside the app only. It sends no email and no messages.",
		"CM-052": "Notices are shown to owners in the app. The app sends no email and no messages.",
		"CM-054": "Refresh and re-read live in Control Room › Data and sources.",
	}
}

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
			note := builtNotes()[row.ID]
			group.Controls = append(group.Controls, ControlEntry{ID: row.ID, Built: controlBuilt(row.ID, fields),
				Note: note, Fields: fields})
		}
		out.Groups = append(out.Groups, group)
	}
	return out
}
