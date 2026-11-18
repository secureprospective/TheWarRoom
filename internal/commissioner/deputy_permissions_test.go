package commissioner

import (
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
	return b.records
}

func permissionTestCommand(command string) RawCommand {
	raw := RawCommand{League: "native.test", Command: command}
	if command == RulesSet {
		raw.Args.Field = "roster.limits"
	}
	return raw
}

func TestMayRunEveryDelegatedCatalogCommandByPermission(t *testing.T) {
	records := permissionTestLedger(t)
	groups := make(map[governance.DeputyPermission]bool)
	for _, rule := range DeputyCommandRules() {
		if rule.Class != DeputyDelegable {
			continue
		}
		groups[rule.Permission] = true
		raw := permissionTestCommand(rule.Command)
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
	for _, rule := range DeputyCommandRules() {
		if rule.Class != DeputyDelegable {
			continue
		}
		if role, err := AuthorizeRecords(permissionTestCommand(rule.Command), "co-commissioner", records); err != nil || role != "cocommissioner" {
			t.Errorf("co-commissioner could not run %s: role=%q err=%v", rule.Command, role, err)
		}
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
