package commissioner

import (
	"encoding/json"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// RulePending is a recorded change that is not in force yet.
type RulePending struct {
	Entry string `json:"entry"`
	Value string `json:"value"`
	When  string `json:"when"`
}

// RuleSafeguards says what Go asks of the person confirming a change, for each time it can take effect:
// "confirm" (a plain button), "hold" (hold to confirm, a reason optional) or "reason" (a typed reason).
type RuleSafeguards struct {
	NextRollover string `json:"nextRollover"`
	Now          string `json:"now"`
}

// RuleRow carries what the rules form needs: the value in words, its source and how it may be edited.
type RuleRow struct {
	Field       string                  `json:"field"`
	Label       string                  `json:"label"`
	Value       string                  `json:"value"`
	State       string                  `json:"state"`
	Citation    string                  `json:"citation"`
	Schema      leaguerules.ValueSchema `json:"schema"`
	Competitive bool                    `json:"competitive"`
	Pending     []RulePending           `json:"pending"`
	// Current is the value in force as the plain JSON a change is sent in; null while it Needs a ruling.
	Current json.RawMessage `json:"current"`
	// Unit is "percent" where the value is a share a GM types as a percentage.
	Unit       string         `json:"unit"`
	Safeguards RuleSafeguards `json:"safeguards"`
}
type RuleSection struct {
	Label string    `json:"label"`
	Rows  []RuleRow `json:"rows"`
}
type Rules struct {
	League   string        `json:"league"`
	Name     string        `json:"name"`
	Revision int64         `json:"revision"`
	Rollover string        `json:"rollover"`
	Sections []RuleSection `json:"sections"`
}

const (
	rolloverWords = "Needs a ruling: the next rollover has no date until the season rollover is built"
	waitingWords  = "Waits for the next rollover"
	unsetWords    = "Not set when the league was created"
)

// RulesReading is public: every owner may read the league's rules and the changes waiting on them.
func RulesReading(league string, records []Record) (Rules, error) {
	if len(records) == 0 {
		return Rules{}, fmt.Errorf("this app league has not been created")
	}
	rules, err := resolveRecords(records)
	if err != nil {
		return Rules{}, err
	}
	sources, pending, err := ruleHistory(records)
	if err != nil {
		return Rules{}, err
	}
	values, fields, current := leaguerules.ReadSettings(rules), leaguerules.Catalogue(), leaguerules.Current(rules)
	out := Rules{League: league, Name: Replay(records).Name, Revision: int64(len(records)),
		Rollover: rolloverWords, Sections: []RuleSection{}}
	for i, field := range fields {
		value := values[i]
		row := RuleRow{Field: string(field.Path), Label: field.Label, Value: value.Words, State: value.State,
			Citation: value.Citation, Schema: field.Schema, Competitive: field.Competitive,
			Pending: pending[string(field.Path)], Current: current[i], Unit: unitFor(field.Path),
			Safeguards: safeguardsFor(string(field.Path))}
		if source, ok := sources[row.Field]; ok {
			row.Citation = source
		}
		if row.Citation == "" {
			row.Citation = unsetWords
		}
		if row.Pending == nil {
			row.Pending = []RulePending{}
		}
		if len(out.Sections) == 0 || out.Sections[len(out.Sections)-1].Label != field.Section {
			out.Sections = append(out.Sections, RuleSection{Label: field.Section, Rows: []RuleRow{}})
		}
		last := &out.Sections[len(out.Sections)-1]
		last.Rows = append(last.Rows, row)
	}
	return out, nil
}

// ruleHistory finds where each in-force value came from and which recorded changes still wait.
func ruleHistory(records []Record) (map[string]string, map[string][]RulePending, error) {
	active := Replay(records).Active
	sources, pending := map[string]string{}, map[string][]RulePending{}
	for _, r := range records {
		if r.Command.Command != RulesSet || !active[r.Receipt.Entry] {
			continue
		}
		field := r.Command.Args.Field
		if r.Command.Effective == "now" {
			sources[field] = "Changed in the league ledger on " + r.Receipt.RecordedAt.Format("2 Jan 2006")
			continue
		}
		setting, err := leaguerules.DecodeSetting(leaguerules.FieldPath(field), r.Command.Args.Value)
		if err != nil {
			return nil, nil, fmt.Errorf("read waiting rule: %w", err)
		}
		pending[field] = append(pending[field], RulePending{Entry: r.Receipt.Entry,
			Value: leaguerules.DescribeField(leaguerules.FieldPath(field), setting).Words, When: waitingWords})
	}
	return sources, pending, nil
}

func unitFor(path leaguerules.FieldPath) string {
	if leaguerules.IsPercent(path) {
		return "percent"
	}
	return ""
}

// safeguardsFor asks the same Definition a submitted change gets, so the form never guesses.
func safeguardsFor(field string) RuleSafeguards {
	words := func(effective string) string {
		d, err := RawCommand{Command: RulesSet, Effective: effective, Args: RawArgs{Field: field}}.Definition()
		if err != nil {
			return "reason"
		}
		switch d.Gravity {
		case "G3":
			return "reason"
		case "G2":
			return "hold"
		default:
			return "confirm"
		}
	}
	return RuleSafeguards{NextRollover: words("next_rollover"), Now: words("now")}
}
