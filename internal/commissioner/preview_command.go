package commissioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// CommandPreview describes the proposed change for a GM. The checks that may refuse it run elsewhere and
// are added as blockers by the caller; a description is still returned when they refuse.
func CommandPreview(raw RawCommand, records []Record, catalogue PreviewCatalogue, now time.Time) (Preview, error) {
	preview, err := commandPreview(raw, records, catalogue, now)
	if err != nil {
		return preview, err
	}
	preview.Affects, err = PreviewAffectedFranchises(raw, records, now, preview.Affects)
	return preview, err
}

func commandPreview(raw RawCommand, records []Record, catalogue PreviewCatalogue, now time.Time) (Preview, error) {
	switch raw.Command {
	case ContractActionSet:
		field, value, err := raw.Args.ContractAction.ruleValue()
		if err != nil {
			return Preview{}, err
		}
		raw.Command, raw.Args = RulesSet, RawArgs{Field: string(field), Value: value}
		return RulePreview(raw, records, catalogue, now)
	case RulesSet:
		return RulePreview(raw, records, catalogue, now)
	case Create:
		return createPreview(raw, catalogue)
	case Revert:
		return undoPreview(raw, records, catalogue)
	case roleGrant, roleRevoke:
		return rolePreview(raw, records, catalogue)
	case NoticePublish, NotePost:
		return notePreview(raw, records, catalogue)
	}
	return operationPreview(raw, records, catalogue, now)
}

const (
	roleGrant  = "commish.role.grant"
	roleRevoke = "commish.role.revoke"
	proposed   = "Your proposed change"
)

func safeguardOf(raw RawCommand) (string, error) {
	definition, err := raw.EffectiveDefinition()
	if err != nil {
		return "", err
	}
	return safeguardFor(definition.Gravity), nil
}

// Nothing in an existing league changes when a new one starts, so its affected count is a known zero.
func createPreview(raw RawCommand, catalogue PreviewCatalogue) (Preview, error) {
	safeguard, err := safeguardOf(raw)
	if err != nil {
		return Preview{}, err
	}
	rules := "Every rule waits for a ruling"
	if raw.Args.Preset == PresetLegacy {
		rules = "Legacy NFL rules"
	}
	none := 0
	return Preview{Question: catalogue.questions["CM-001"], Safeguard: safeguard, Blockers: []string{},
		Target: PreviewTarget{League: raw.League, Name: raw.Args.Name, Version: raw.Expected},
		Fields: []PreviewField{
			{"League name", "No league yet", "Not started", raw.Args.Name, proposed, "now"},
			{"Franchises", "None", "Not started", fmt.Sprintf("%d franchises", raw.Args.FranchiseCount),
				proposed, "now"},
			{"Starting rules", "None", "Not started", rules, proposed, "now"}},
		Affects: PreviewAffects{Readings: []string{}, Count: &none,
			Words: "No existing league changes. This starts a new one."}}, nil
}

func seatWords(role governance.Role) string {
	switch role {
	case governance.DOT:
		return "Trade review seat"
	case governance.Commissioner, governance.CoCommissioner, governance.Owner:
		return roleLabel(role)
	}
	return roleLabel(role)
}

// personWords names a person as the ledger does: this desktop's own sign-in reads as the local operator.
func personWords(principal string) string {
	if strings.HasPrefix(principal, "local.") {
		return "Local operator"
	}
	return principal
}

func rolePreview(raw RawCommand, records []Record, catalogue PreviewCatalogue) (Preview, error) {
	if len(records) == 0 {
		return Preview{}, fmt.Errorf("this app league has not been created")
	}
	state, err := GovernanceState(records)
	if err != nil {
		return Preview{}, err
	}
	safeguard, err := safeguardOf(raw)
	if err != nil {
		return Preview{}, err
	}
	person, role := raw.Args.People.Principal, raw.Args.People.Role
	held := state.HasRole(governance.Principal(person), role)
	words := func(holds bool) string {
		if holds {
			return "Holds the role"
		}
		return "Does not hold the role"
	}
	return Preview{Question: catalogue.questions["CM-021"], Safeguard: safeguard, Blockers: []string{},
		Target: PreviewTarget{League: raw.League, Name: Replay(records).Name, Version: raw.Expected},
		Fields: []PreviewField{{Label: seatWords(role) + " for " + personWords(person),
			Before: words(held), BeforeSource: "Recorded league roles",
			After: words(raw.Command == roleGrant), AfterSource: proposed, When: "now"}},
		Affects: catalogue.affects("CM-021")}, nil
}

// controlOf is the census row that owns a recorded change, so undoing it affects what it affected.
func controlOf(r Record) string {
	if r.Command.Command == RulesSet {
		if owner, err := ownerFor(leaguerules.FieldPath(r.Command.Args.Field)); err == nil {
			return owner.control
		}
		return ""
	}
	if d, ok := peopleDefinitionFor(r.Command.Command); ok {
		return d.control
	}
	if d, ok := noteDefinitionFor(r.Command.Command); ok {
		return d.control
	}
	return ""
}

func undoPreview(raw RawCommand, records []Record, catalogue PreviewCatalogue) (Preview, error) {
	safeguard, err := safeguardOf(raw)
	if err != nil {
		return Preview{}, err
	}
	for i, r := range records {
		if r.Receipt.Entry != raw.Args.Entry {
			continue
		}
		what, err := entryLabel(records, i)
		if err != nil {
			return Preview{}, err
		}
		after := "Undone: this change stops counting. The original stays in the ledger."
		if !Replay(records).Active[r.Receipt.Entry] {
			after = "Restored: this change counts again."
		}
		return Preview{Question: catalogue.questions["CM-039"], Safeguard: safeguard, Blockers: []string{},
			Target: PreviewTarget{League: raw.League, Name: Replay(records).Name, Version: raw.Expected},
			Fields: []PreviewField{{Label: "Recorded change", Before: what,
				BeforeSource: "Recorded by " + personWords(r.Principal) + " in the league ledger",
				After:        after, AfterSource: proposed, When: "now"}},
			Affects: catalogue.affects(controlOf(r))}, nil
	}
	return Preview{}, fmt.Errorf("that recorded change is not in this league")
}

// notePreview shows the words exactly as owners will read them. Nothing is posted until it is confirmed.
func notePreview(raw RawCommand, records []Record, catalogue PreviewCatalogue) (Preview, error) {
	if len(records) == 0 {
		return Preview{}, fmt.Errorf("this app league has not been created")
	}
	d, _ := noteDefinitionFor(raw.Command)
	safeguard, err := safeguardOf(raw)
	if err != nil {
		return Preview{}, err
	}
	where := "At the top of the league notices"
	if !raw.Args.Note.Pin {
		where = "In the league notices, newest first"
	}
	return Preview{Question: catalogue.questions[d.control], Safeguard: safeguard, Blockers: []string{},
		Target: PreviewTarget{League: raw.League, Name: Replay(records).Name, Version: raw.Expected},
		Fields: []PreviewField{{Label: d.title, Before: "Not posted",
			BeforeSource: "Nothing is shown to owners until you confirm", After: raw.Args.Note.Text,
			AfterSource: proposed + ". " + where, When: "now"}},
		Affects: catalogue.affects(d.control)}, nil
}
