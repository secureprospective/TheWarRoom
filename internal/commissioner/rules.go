package commissioner

import (
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const RulesSet = "commish.rules.set"

func ruleCommand(command string) bool {
	return command == RulesSet || command == ContractActionSet
}

// Definition derives gravity from the command and, for rules, from the field's census row and timing.
func (r RawCommand) Definition() (Definition, error) {
	if !ruleCommand(r.Command) {
		return DefinitionFor(r.Command)
	}
	if r.Command == ContractActionSet {
		if r.Args.ContractAction == nil {
			return Definition{}, fmt.Errorf("choose a contract action")
		}
		field, err := r.Args.ContractAction.field()
		if err != nil {
			return Definition{}, err
		}
		return (RawCommand{Command: RulesSet, Effective: r.Effective,
			Args: RawArgs{Field: string(field)}}).Definition()
	}
	field, err := catalogueField(r.Args.Field)
	if err != nil {
		return Definition{}, err
	}
	owner, err := ownerFor(field.Path)
	if err != nil {
		return Definition{}, err
	}
	if r.Effective == "now" && field.Competitive {
		return Definition{Gravity: "G3"}, nil
	}
	return Definition{Gravity: owner.gravity}, nil
}

func (r RawCommand) validateRulePlacement() error {
	if r.Visibility != "league" {
		return fmt.Errorf("rule changes are shown to every owner")
	}
	if r.Effective != "now" && r.Effective != "next_rollover" {
		return fmt.Errorf("choose now or the next rollover; a chosen start date is not available yet")
	}
	return nil
}

func (a RawArgs) validateRule() error {
	if a.Name != "" || a.Entry != "" || a.People != nil {
		return fmt.Errorf("send only the rule and its new value")
	}
	if _, err := leaguerules.DecodeSetting(leaguerules.FieldPath(a.Field), a.Value); err != nil {
		return Refused(fmt.Errorf("rule change refused: %w", err))
	}
	return nil
}

// authorizeRule checks the census row that owns the field; the caller never names it.
func authorizeRule(auth GovernanceAuthorizer, raw RawCommand, principal string) (string, error) {
	owner, err := ownerFor(leaguerules.FieldPath(raw.Args.Field))
	if err != nil {
		return "", err
	}
	row, ok := governance.CensusControl(owner.control)
	if !ok {
		return "", fmt.Errorf("this league rule is not available")
	}
	role, err := governance.Allow(auth.State, governance.Principal(principal), governance.CommandRef{
		ID: row.ID, MinimumRole: row.MinimumRole, COnly: row.COnly})
	if err != nil {
		return "", fmt.Errorf("league permission: %w", err)
	}
	if role == governance.CoCommissioner {
		return "cocommissioner", nil
	}
	return string(role), nil
}

// resolveRecords reads the rules in force at the ledger's last entry. Readings that run on the
// clock call RulesNow instead.
func resolveRecords(records []Record) (leaguerules.RuleSet, error) {
	return RulesNow(records, time.Time{})
}

// CheckRulesAppend refuses a change whose result the league's rules could not hold together.
func CheckRulesAppend(raw RawCommand, records []Record, at time.Time) error {
	trial := make([]Record, len(records), len(records)+1)
	copy(trial, records)
	trial = append(trial, Record{Command: raw, Receipt: Receipt{Entry: "proposed." + raw.Request, RecordedAt: at}})
	if _, err := resolveRecords(trial); err != nil {
		return Refused(fmt.Errorf("rule change refused: %w", err))
	}
	return checkRuleChangesSomething(raw, records, at)
}

// ruleValue reads one rule as the plain JSON a change is sent in; "null" while it needs a ruling.
func ruleValue(records []Record, field string) (string, error) {
	rules, err := resolveRecords(records)
	if err != nil {
		return "", err
	}
	current := leaguerules.Current(rules)
	for i, f := range leaguerules.Catalogue() {
		if string(f.Path) == field {
			return string(current[i]), nil
		}
	}
	return "", fmt.Errorf("choose a recognised league rule")
}

// withWaitingInForce counts the newest change waiting on the next rollover for field as already in force.
// That is what a second change from the next rollover has to differ from.
func withWaitingInForce(records []Record, field string) ([]Record, bool) {
	active := Replay(records).Active
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		if r.Command.Command != RulesSet || r.Command.Args.Field != field || !active[r.Receipt.Entry] ||
			r.Command.Effective != "next_rollover" {
			continue
		}
		counted := append([]Record(nil), records...)
		r.Command.Effective = "now"
		r.Receipt.Entry = "waiting." + r.Receipt.Entry
		// Rules resolve by recorded instant, so the waiting change is counted at the newest one:
		// an immediate change recorded after it must not shadow it.
		for _, later := range records[i+1:] {
			if later.Receipt.RecordedAt.After(r.Receipt.RecordedAt) {
				r.Receipt.RecordedAt = later.Receipt.RecordedAt
			}
		}
		return append(counted, r), true
	}
	return records, false
}

// checkRuleChangesSomething refuses a rule change that leaves the rule exactly as it already is, whether
// it is in force or already waiting for the next rollover. Preview and execute share it.
func checkRuleChangesSomething(raw RawCommand, records []Record, at time.Time) error {
	if raw.Command != RulesSet {
		return nil
	}
	base, waiting := records, false
	if raw.Effective == "next_rollover" {
		base, waiting = withWaitingInForce(records, raw.Args.Field)
	}
	before, err := ruleValue(base, raw.Args.Field)
	if err != nil {
		return Refused(fmt.Errorf("rule change refused: %w", err))
	}
	trial := raw
	trial.Effective = "now"
	after, err := ruleValue(append(append([]Record(nil), base...), Record{Command: trial,
		Receipt: Receipt{Entry: "proposed." + raw.Request, RecordedAt: at}}), raw.Args.Field)
	if err != nil {
		return Refused(fmt.Errorf("rule change refused: %w", err))
	}
	if before != after {
		return nil
	}
	if waiting {
		return Refused(errors.New("this change is already waiting for the next rollover. Nothing would change"))
	}
	return Refused(errors.New("this is already the rule. Nothing would change"))
}

func RuleValueWords(rules leaguerules.RuleSet, path string) string {
	fields, values := leaguerules.Catalogue(), leaguerules.ReadSettings(rules)
	for i, f := range fields {
		if string(f.Path) == path {
			return values[i].Words
		}
	}
	return "Needs a ruling"
}

func timingWords(effective string) string {
	if effective == "next_rollover" {
		return "from the next rollover"
	}
	return "in force now"
}

// ruleLabel says, for the ledger, label: before → after and when it takes effect.
func ruleLabel(r Record, before []Record) (string, error) {
	r, err := ruleRecord(r)
	if err != nil {
		return "", err
	}
	field, err := catalogueField(r.Command.Args.Field)
	if err != nil {
		return "", err
	}
	rules, err := resolveRecords(before)
	if err != nil {
		return "", err
	}
	setting, err := leaguerules.DecodeSetting(field.Path, r.Command.Args.Value)
	if err != nil {
		return "", fmt.Errorf("describe recorded rule: %w", err)
	}
	return fmt.Sprintf("%s: %s → %s, %s", field.Label, RuleValueWords(rules, string(field.Path)),
		leaguerules.DescribeField(field.Path, setting).Words, timingWords(r.Command.Effective)), nil
}

// revertRuleLabel describes a reversal whose original change was a rule change; ok is false otherwise.
func revertRuleLabel(records []Record, i int, target Record) (string, bool, error) {
	var err error
	target, err = ruleRecord(target)
	if err != nil {
		return "", false, err
	}
	if target.Command.Command != RulesSet {
		return "", false, nil
	}
	field, err := catalogueField(target.Command.Args.Field)
	if err != nil {
		return "", false, err
	}
	if target.Command.Effective == "next_rollover" {
		verb := "Withdrew"
		if Replay(records[:i+1]).Active[target.Receipt.Entry] {
			verb = "Restored"
		}
		return verb + " a waiting rule change to " + field.Label + ", " + timingWords("next_rollover"), true, nil
	}
	before, err := resolveRecords(records[:i])
	if err != nil {
		return "", false, err
	}
	after, err := resolveRecords(records[:i+1])
	if err != nil {
		return "", false, err
	}
	return fmt.Sprintf("Reversed a rule change. %s: %s → %s, in force now", field.Label,
		RuleValueWords(before, string(field.Path)), RuleValueWords(after, string(field.Path))), true, nil
}
