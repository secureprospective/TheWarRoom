package commissioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const RulesSet = "commish.rules.set"

// ruleOwner maps a rule field (exact path, or a section ending in ".") to the census row that owns it.
type ruleOwner struct{ prefix, control, gravity string }

func ruleOwners() []ruleOwner {
	return []ruleOwner{
		{"league.teams", "CM-001", "G2"}, {"league.conferences", "CM-001", "G2"},
		{"league.divisions", "CM-001", "G2"}, {"league.format", "CM-001", "G2"},
		{"league.initialRosters", "CM-001", "G2"}, {"league.alignment", "CM-002", "G2"},
		{"league.dynasty", "CM-005", "G2"}, {"league.carry", "CM-005", "G2"},
		{"league.keepers", "CM-005", "G2"}, {"roster.", "CM-003", "G2"},
		{"lineup.", "CM-004", "G2"}, {"scoring.", "CM-006", "G2"},
		{"standings.", "CM-007", "G2"}, {"cap.", "CM-008", "G2"},
		{"contracts.", "CM-009", "G2"}, {"ir.", "CM-010", "G2"},
		{"taxi.", "CM-011", "G2"}, {"waivers.", "CM-012", "G2"},
		{"bids.", "CM-013", "G2"}, {"trades.", "CM-014", "G2"},
		{"schedule.", "CM-015", "G2"}, {"playoffs.", "CM-016", "G2"},
		{"house.", "CM-017", "G1"}, {"calendar.windows", "CM-023", "G2"},
		{"calendar.timezone", "CM-024", "G2"}, {"calendar.boundary", "CM-024", "G2"},
		{"calendar.rollover", "CM-027", "G3"}, {"calendar.rolloverDate", "CM-027", "G3"},
		{"calendar.tradeDeadline", "CM-025", "G2"}, {"calendar.tradeDeadlineWeek", "CM-025", "G2"},
		{"draft.mode", "CM-040", "G2"}, {"draft.clock", "CM-040", "G2"},
		{"draft.onExpire", "CM-040", "G2"}, {"draft.timeouts", "CM-040", "G2"},
		{"draft.rounds", "CM-040", "G2"}, {"draft.maximumPicks", "CM-040", "G2"},
		{"draft.forfeitExtra", "CM-040", "G2"}, {"draft.pricing", "CM-040", "G2"},
		{"draft.lengths", "CM-040", "G2"}, {"draft.rfaAfter", "CM-040", "G2"},
		{"draft.orderSource", "CM-041", "G2"}, {"draft.order", "CM-041", "G2"},
		{"draft.laterPattern", "CM-041", "G2"}, {"draft.tieRule", "CM-041", "G2"},
	}
}

func (o ruleOwner) owns(path leaguerules.FieldPath) bool {
	if strings.HasSuffix(o.prefix, ".") {
		return strings.HasPrefix(string(path), o.prefix)
	}
	return string(path) == o.prefix
}

// ownerFor refuses a field that no row, or more than one row, owns.
func ownerFor(path leaguerules.FieldPath) (ruleOwner, error) {
	var found []ruleOwner
	for _, owner := range ruleOwners() {
		if owner.owns(path) {
			found = append(found, owner)
		}
	}
	if len(found) != 1 {
		return ruleOwner{}, fmt.Errorf("choose a recognised league rule")
	}
	return found[0], nil
}

func catalogueField(path string) (leaguerules.CatalogueField, error) {
	for _, f := range leaguerules.Catalogue() {
		if string(f.Path) == path {
			return f, nil
		}
	}
	return leaguerules.CatalogueField{}, fmt.Errorf("choose a recognised league rule")
}

// Definition derives gravity from the command and, for rules, from the field's census row and timing.
func (r RawCommand) Definition() (Definition, error) {
	if r.Command != RulesSet {
		return DefinitionFor(r.Command)
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
	return nil
}

func fieldWords(rules leaguerules.RuleSet, path string) string {
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
	return fmt.Sprintf("%s: %s → %s, %s", field.Label, fieldWords(rules, string(field.Path)),
		leaguerules.DescribeField(field.Path, setting).Words, timingWords(r.Command.Effective)), nil
}

// revertRuleLabel describes a reversal whose original change was a rule change; ok is false otherwise.
func revertRuleLabel(records []Record, i int) (string, bool, error) {
	target := records[i]
	for target.Command.Command == Revert {
		found := false
		for _, r := range records[:i] {
			if r.Receipt.Entry == target.Command.Args.Entry {
				target, found = r, true
				break
			}
		}
		if !found {
			return "", false, nil
		}
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
		fieldWords(before, string(field.Path)), fieldWords(after, string(field.Path))), true, nil
}
