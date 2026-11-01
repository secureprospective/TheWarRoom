package commissioner

import "github.com/secureprospective/TheWarRoom/internal/leaguerules"

// MirrorRulesReading words an MFL-mirror league's held settings in the same shape as RulesReading.
// A mirror league has no ledger, so nothing is pending, nothing changed here, and every row cites MFL.
// revision is the held roster time, shared by every mirror read of the same refresh.
func MirrorRulesReading(league, name string, rules leaguerules.RuleSet, revision int64) Rules {
	values, fields, current := leaguerules.ReadSettings(rules), leaguerules.Catalogue(), leaguerules.Current(rules)
	out := Rules{League: league, Name: name, Revision: revision, Rollover: "Rules change on MFL; the app reads them at each refresh.",
		Sections: []RuleSection{}}
	for i, field := range fields {
		value := values[i]
		row := RuleRow{Field: string(field.Path), Label: field.Label, Value: value.Words, State: value.State,
			Citation: "Held from MFL", Schema: field.Schema, Competitive: field.Competitive,
			NowOnly: shapeField(string(field.Path)) || field.Path == poolsField, Pending: []RulePending{},
			Current: current[i], Unit: unitFor(field.Path), Safeguards: safeguardsFor(string(field.Path))}
		if len(out.Sections) == 0 || out.Sections[len(out.Sections)-1].Label != field.Section {
			out.Sections = append(out.Sections, RuleSection{Label: field.Section, Rows: []RuleRow{}})
		}
		last := &out.Sections[len(out.Sections)-1]
		last.Rows = append(last.Rows, row)
	}
	return out
}
