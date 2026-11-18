package commissioner

import (
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func permissionTestLedger(t *testing.T) []Record {
	t.Helper()
	b := newLedger(t, PresetEmpty)
	for _, grant := range []struct {
		person string
		role   governance.Role
	}{
		{"full-deputy", governance.Deputy},
		{"empty-deputy", governance.Deputy},
		{"co-commissioner", governance.CoCommissioner},
	} {
		b.add(RawCommand{League: "native.test", Command: "commish.role.grant",
			Args: RawArgs{People: &RawPeople{Principal: grant.person, Role: grant.role}}}, seasonMoment(2))
	}
	b.add(RawCommand{League: "native.test", Command: DeputyPermissionsSet,
		Args: RawArgs{People: &RawPeople{Principal: "full-deputy", Permissions: governance.AllDeputyPermissions()}}}, seasonMoment(3))
	b.add(RawCommand{League: "native.test", Command: "commish.seat.replace",
		Args: RawArgs{People: &RawPeople{Franchise: "franchise.01", Principal: "full-deputy"}}}, seasonMoment(4))
	return b.records
}

func permissionTestCommand(command string) RawCommand {
	raw := RawCommand{League: "native.test", Command: command}
	switch command {
	case RulesSet:
		raw.Args.Field = "roster.limits"
	case LineupSet, LineupCorrect:
		raw.Args.Lineup = &RawLineupChange{Season: 2026, Franchise: "franchise.01", Week: 1, Starters: []string{"1001"}}
	}
	return raw
}

func TestDeputyPermissionGatesNonTradeCatalogCommands(t *testing.T) {
	records := permissionTestLedger(t)
	groups := make(map[governance.DeputyPermission]bool)
	for _, rule := range DeputyCommandRules() {
		if rule.Class != DeputyDelegable {
			continue
		}
		groups[rule.Permission] = true
		raw := permissionTestCommand(rule.Command)
		if TradeCommand(rule.Command) {
			continue // Trade-review authorization needs a real recorded trade; covered separately.
		}
		if role, err := AuthorizeRecords(raw, "full-deputy", records); err != nil || role != "deputy" {
			t.Errorf("ticked deputy could not run %s: role=%q err=%v", rule.Command, role, err)
		}
		if _, err := AuthorizeRecords(raw, "empty-deputy", records); err == nil {
			t.Errorf("unticked deputy ran %s", rule.Command)
		}
	}
	for _, permission := range governance.AllDeputyPermissions() {
		if !groups[permission] {
			t.Errorf("permission %q has no catalog commands", permission)
		}
	}
}

func TestCoCommissionerHasAllSixPermissionGroups(t *testing.T) {
	records := permissionTestLedger(t)
	state, err := GovernanceState(records)
	if err != nil {
		t.Fatal(err)
	}
	if got := effectiveDeputyPermissions(state, "co-commissioner"); !slices.Equal(got, governance.AllDeputyPermissions()) {
		t.Fatalf("co-commissioner's effective permissions = %v", got)
	}
}

func TestDeputyEditLineupsKeepsExistingLineupConstraints(t *testing.T) {
	records := permissionTestLedger(t)
	set := permissionTestCommand(LineupSet)
	set.Args.Lineup.Franchise = "franchise.02"
	if _, err := AuthorizeRecords(set, "full-deputy", records); err == nil || err.Error() != "you may set a lineup only for your own franchise" {
		t.Fatalf("deputy set another franchise's lineup: %v", err)
	}
	off := append(append([]Record(nil), records...), Record{Receipt: Receipt{Entry: "lineups-off"}, Principal: "commissioner",
		Command: RawCommand{League: "native.test", Command: "commish.abilities.set",
			Args: RawArgs{People: &RawPeople{Franchise: "franchise.01", Ability: "lineups"}}}})
	if _, err := AuthorizeRecords(permissionTestCommand(LineupSet), "full-deputy", off); err == nil ||
		!strings.Contains(err.Error(), "turned off lineup changes") {
		t.Fatalf("deputy bypassed the franchise's lineup setting: %v", err)
	}
}

func TestChangeRulesTickPreservesCommissionerOnlyRuleFields(t *testing.T) {
	records := permissionTestLedger(t)
	raw := permissionTestCommand(RulesSet)
	raw.Args.Field = "league.teams"
	if _, err := AuthorizeRecords(raw, "full-deputy", records); err == nil {
		t.Fatal("Change rules tick delegated a commissioner-only league rule")
	}
}

func TestDeputyWithEveryTickCannotCloseOrTiebreakSuccessionVote(t *testing.T) {
	records := permissionTestLedger(t)
	for _, command := range []string{"commish.vote.close", "commish.vote.tiebreak"} {
		raw := RawCommand{League: "native.test", Command: command,
			Args: RawArgs{People: &RawPeople{ID: "election.2026", Principal: "full-deputy", Choice: "commissioner"}}}
		if _, err := AuthorizeRecords(raw, "full-deputy", records); err == nil {
			t.Errorf("deputy with all six permissions ran %s", command)
		}
	}
}

func TestDeputyWithEveryTickCannotRunCommissionerOnlyCatalogCommands(t *testing.T) {
	records := permissionTestLedger(t)
	for _, rule := range DeputyCommandRules() {
		if rule.Class != DeputyCommissionerOnly {
			continue
		}
		if _, err := AuthorizeRecords(permissionTestCommand(rule.Command), "full-deputy", records); err == nil {
			t.Errorf("deputy with all six permissions ran commissioner-only %s", rule.Command)
		}
	}
}

func TestCoCommissionerTradeOverrideRemainsCommissionerOnly(t *testing.T) {
	records := permissionTestLedger(t)
	raw := permissionTestCommand(TradeOverride)
	if _, err := AuthorizeRecords(raw, "co-commissioner", records); err == nil || err.Error() != "only the commissioner may override a trade" {
		t.Fatalf("co-commissioner override refusal = %v", err)
	}
}

func TestCoCommissionerCannotUseCommissionerOnlyControls(t *testing.T) {
	records := permissionTestLedger(t)
	commands := []string{"commish.role.grant", "commish.role.revoke", DeputyOrder, DeputyPermissionsSet,
		"commish.succession.inactivity.set", "commish.succession.periods.set", "commish.vote.close", "commish.vote.tiebreak"}
	for _, command := range commands {
		if _, err := AuthorizeRecords(permissionTestCommand(command), "co-commissioner", records); err == nil {
			t.Errorf("co-commissioner ran commissioner-only %s", command)
		}
	}
}
