package history

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func TestRuleRowsCarryWhatTheFormNeeds(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetLegacy, 0), "creator", nil); err != nil {
		t.Fatal(err)
	}
	if row := ruleRow(t, s, "roster.limits"); string(row.Current) != `{"Max":80,"Min":18}` ||
		row.Safeguards != (commissioner.RuleSafeguards{NextRollover: "hold", Now: "reason"}) || row.Unit != "" {
		t.Fatalf("roster limits row: %+v", row)
	}
	if row := ruleRow(t, s, "cap.irShare"); row.Unit != "percent" || string(row.Current) == "null" {
		t.Fatalf("a share is typed as a percentage: %+v", row)
	}
	if row := ruleRow(t, s, "calendar.rollover"); row.Safeguards.NextRollover != "reason" {
		t.Fatalf("the rollover is always a typed reason: %+v", row.Safeguards)
	}
	if row := ruleRow(t, s, "house.texts"); row.Safeguards != (commissioner.RuleSafeguards{NextRollover: "confirm", Now: "confirm"}) {
		t.Fatalf("house rules are a plain confirm: %+v", row.Safeguards)
	}
}

func TestRuleRowsOfAnEmptyLeagueSendNoCurrentValue(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetEmpty, 4), "creator", nil); err != nil {
		t.Fatal(err)
	}
	if row := ruleRow(t, s, "roster.limits"); string(row.Current) != "null" || row.State != "Needs a ruling" {
		t.Fatalf("an unset rule is not sent as a value: %+v", row)
	}
}

func undoOf(t *testing.T, s *Store, principal string) map[string]string {
	t.Helper()
	ledger, err := s.CommissionerLedger(t.Context(), "native.test", principal, nil)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, e := range ledger.Entries {
		out[e.ID] = e.Undo
	}
	return out
}

func TestLedgerSaysWhichEntriesThisPrincipalMayUndo(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	created, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetLegacy, 0), "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	grant := executePeople(t, s, "creator", "commish.role.grant",
		commissioner.RawPeople{Principal: "cc", Role: governance.CoCommissioner}, 1)
	rule, err := s.ExecuteCommissioner(t.Context(),
		ruleCommand("roster.limits", `{"min":20,"max":70}`, "next_rollover", "", 2), "cc", nil)
	if err != nil {
		t.Fatal(err)
	}
	creator, cc, owner := undoOf(t, s, "creator"), undoOf(t, s, "cc"), undoOf(t, s, "owner")
	if creator[created.Entry] != "" || creator[grant.Entry] != "private" || creator[rule.Entry] != "league" {
		t.Fatalf("the commissioner may undo the grant privately and the rule openly: %v", creator)
	}
	if cc[grant.Entry] != "" || cc[rule.Entry] != "league" {
		t.Fatalf("a co-commissioner may not undo a role grant: %v", cc)
	}
	for id, undo := range owner {
		if undo != "" {
			t.Fatalf("an owner is offered an undo for %s", id)
		}
	}
	undone, err := s.ExecuteCommissioner(t.Context(), reverseCommand(rule.Entry, 3, "undo.rule"), "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	after := undoOf(t, s, "creator")
	if after[rule.Entry] != "" || after[undone.Entry] != "league" {
		t.Fatalf("a reversed entry is spent; its reversal may be undone: %v", after)
	}
}

func TestPeopleReadingSaysWhatYouMayStart(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetEmpty, 4), "creator", nil); err != nil {
		t.Fatal(err)
	}
	executePeople(t, s, "creator", "commish.role.grant",
		commissioner.RawPeople{Principal: "cc", Role: governance.CoCommissioner}, 1)
	for principal, want := range map[string]commissioner.People{
		"creator": {YourRole: "Commissioner", MayChangeRules: true, MayGrantRoles: true},
		"cc":      {YourRole: "Co-commissioner", MayChangeRules: true},
		"nobody":  {YourRole: "Visitor"},
	} {
		got, err := s.CommissionerPeople(t.Context(), "native.test", principal)
		if err != nil || got.YourRole != want.YourRole || got.MayChangeRules != want.MayChangeRules ||
			got.MayGrantRoles != want.MayGrantRoles {
			t.Fatalf("%s: %+v %v", principal, got, err)
		}
	}
}
