package commissioner

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// RulePending is a recorded change that is not in force yet.
type RulePending struct {
	Entry      string `json:"entry"`
	Value      string `json:"value"`
	When       string `json:"when"`
	ReplacedAt string `json:"replacedAt,omitempty"`
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
	ChangedAt   string                  `json:"changedAt"`
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
	rolloverWords = "The next rollover has no date yet; the commissioner sets it."
	waitingWords  = "Waits for the next rollover"
)

// RulesReading is public: every owner may read the league's rules and the changes waiting on them.
func RulesReading(league string, records []Record, now time.Time) (Rules, error) {
	if len(records) == 0 {
		return Rules{}, fmt.Errorf("this app league has not been created")
	}
	rules, err := RulesNow(records, now)
	if err != nil {
		return Rules{}, err
	}
	sources, pending, err := ruleHistory(records, ReadingInstant(records, now))
	if err != nil {
		return Rules{}, err
	}
	values, fields, current := leaguerules.ReadSettings(rules), leaguerules.Catalogue(), leaguerules.Current(rules)
	out := Rules{League: league, Name: Replay(records).Name, Revision: int64(len(records)),
		Rollover: rolloverWords, Sections: []RuleSection{}}
	preset := records[0].Command.Args.Preset
	for i, field := range fields {
		value := values[i]
		row := RuleRow{Field: string(field.Path), Label: field.Label, Value: value.Words, State: value.State,
			Citation: startingSource(preset, value.State, value.Citation), Schema: field.Schema,
			Competitive: field.Competitive, Pending: pending[string(field.Path)], Current: current[i],
			Unit:       unitFor(field.Path),
			Safeguards: safeguardsFor(string(field.Path))}
		if source, ok := sources[row.Field]; ok {
			row.Citation = "Changed in the league ledger"
			row.ChangedAt = source
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

// ruleHistory finds where each in-force value came from and which recorded changes still wait. A
// next-rollover change stops waiting once its rollover has come.
func ruleHistory(records []Record, at time.Time) (map[string]string, map[string][]RulePending, error) {
	starts, err := rolloverInstants(records)
	if err != nil {
		return nil, nil, err
	}
	active := Replay(records).Active
	sources, pending := map[string]string{}, map[string][]RulePending{}
	latest := map[string]time.Time{}
	for _, r := range records {
		if windowCommand(r.Command.Command) && active[r.Receipt.Entry] {
			sources[string(windowsField)] = r.Receipt.RecordedAt.Format(time.RFC3339Nano)
			continue
		}
		if r.Command.Command != RulesSet || !active[r.Receipt.Entry] {
			continue
		}
		field, startedAt := r.Command.Args.Field, r.Receipt.RecordedAt
		if rollover, ok := nextRollover(starts, startedAt); r.Command.Effective != "now" {
			if !ok || at.Before(rollover) {
				change, err := waitingChange(r)
				if err != nil {
					return nil, nil, err
				}
				pending[field] = append(pending[field], change)
				continue
			}
			startedAt = rollover
		}
		if previous, ok := latest[field]; !ok || !startedAt.Before(previous) {
			latest[field] = startedAt
			sources[field] = startedAt.Format(time.RFC3339Nano)
		}
	}
	if err := markReplaced(records, pending); err != nil {
		return nil, nil, err
	}
	return sources, pending, nil
}

// waitingChange describes one recorded change that still waits for its rollover.
func waitingChange(r Record) (RulePending, error) {
	field := leaguerules.FieldPath(r.Command.Args.Field)
	setting, err := leaguerules.DecodeSetting(field, r.Command.Args.Value)
	if err != nil {
		return RulePending{}, fmt.Errorf("read waiting rule: %w", err)
	}
	return RulePending{Entry: r.Receipt.Entry, Value: leaguerules.DescribeField(field, setting).Words,
		When: waitingWords}, nil
}

// markReplaced marks each waiting change a later one for the same rule and rollover replaced.
func markReplaced(records []Record, pending map[string][]RulePending) error {
	replaced, err := replacedWaiting(records)
	if err != nil {
		return err
	}
	for _, changes := range pending {
		for i := range changes {
			if at, ok := replaced[changes[i].Entry]; ok {
				changes[i].When, changes[i].ReplacedAt = "Replaced by a later change", at
			}
		}
	}
	return nil
}

// replacedWaiting names each waiting change that a later one for the same rule and the same rollover
// replaced, with when it was replaced. Only the newest takes effect (Christopher, 10-14). It does not
// depend on the clock, so the ledger keeps the mark after the rollover has passed.
func replacedWaiting(records []Record) (map[string]string, error) {
	starts, err := rolloverInstants(records)
	if err != nil {
		return nil, err
	}
	active := Replay(records).Active
	groups, order := map[string][]Record{}, []string{}
	for _, r := range records {
		if r.Command.Command != RulesSet || !active[r.Receipt.Entry] || r.Command.Effective == "now" {
			continue
		}
		key := r.Command.Args.Field + "|unscheduled"
		if rollover, ok := nextRollover(starts, r.Receipt.RecordedAt); ok {
			key = r.Command.Args.Field + "|" + rollover.Format(time.RFC3339Nano)
		}
		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}
		groups[key] = append(groups[key], r)
	}
	replaced := map[string]string{}
	for _, key := range order {
		changes := groups[key]
		newest := changes[len(changes)-1].Receipt.RecordedAt.Format(time.RFC3339Nano)
		for _, r := range changes[:len(changes)-1] {
			replaced[r.Receipt.Entry] = newest
		}
	}
	return replaced, nil
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

// startingSource translates catalogue provenance without changing the stored preset.
func startingSource(preset, state, citation string) string {
	if state == "Needs a ruling" {
		if _, detail, ok := strings.Cut(citation, ":"); ok && strings.TrimSpace(detail) != "" {
			return "Needs a ruling: " + strings.TrimSpace(detail)
		}
		return "Needs a ruling"
	}
	if preset == PresetEmpty {
		return "Empty league default"
	}
	source := "Legacy NFL rules"
	if strings.Contains(citation, "MFL league scoring settings") {
		return source + " (the league’s MFL scoring)"
	}
	if _, section, ok := strings.Cut(citation, "Rulebook §"); ok {
		number := strings.TrimLeftFunc(section, unicode.IsSpace)
		end := strings.IndexFunc(number, func(r rune) bool { return !unicode.IsDigit(r) })
		if end >= 0 {
			number = number[:end]
		}
		if number != "" {
			source += " (rulebook section " + number + ")"
		}
	}
	return source
}
