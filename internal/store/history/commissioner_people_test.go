package history

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func peopleCommand(command string, args commissioner.RawPeople, revision int64) commissioner.RawCommand {
	visibility := "league"
	if strings.HasPrefix(command, "commish.role.") || strings.HasPrefix(command, "commish.seat.invite") ||
		command == commissioner.DeputyPermissionsSet {
		visibility = "private"
	}
	return commissioner.RawCommand{League: "native.test", Command: command,
		Args: commissioner.RawArgs{People: &args}, Reason: "Make the agreed people change",
		Expected: revision, Request: fmt.Sprintf("request.%d", revision), Effective: "now", Visibility: visibility}
}

func executePeople(t *testing.T, s *Store, principal, command string, args commissioner.RawPeople,
	revision int64) commissioner.Receipt {
	t.Helper()
	receipt, err := s.ExecuteCommissioner(t.Context(), peopleCommand(command, args, revision), principal, nil)
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func TestCommissionerGovernancePermissions(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
		t.Fatal(err)
	}
	grant := executePeople(t, s, "creator", "commish.role.grant",
		commissioner.RawPeople{Principal: "cc", Role: governance.CoCommissioner}, 1)
	for _, command := range []string{"commish.role.grant", "commish.role.revoke"} {
		raw := peopleCommand(command, commissioner.RawPeople{Principal: "cc", Role: governance.CoCommissioner}, 2)
		if _, err := s.ExecuteCommissioner(t.Context(), raw, "cc", nil); err == nil {
			t.Fatalf("CC accepted on C-only command %s", command)
		}
	}
	raw := reverseCommand(grant.Entry, 2, "cc.revert")
	raw.Visibility = "private"
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "cc", nil); err == nil {
		t.Fatal("CC reversed a C-only role grant")
	}
	raw = peopleCommand("commish.role.grant",
		commissioner.RawPeople{Principal: "second", Role: governance.CoCommissioner}, 2)
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
		t.Fatalf("second CC: %v", err)
	}
	raw = peopleCommand("commish.franchise.edit",
		commissioner.RawPeople{Franchise: "franchise.01", Name: "Falcons", Abbreviation: "FAL"}, 3)
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "owner", nil); err == nil {
		t.Fatal("owner changed a franchise identity")
	}
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "cc", nil); err != nil {
		t.Fatalf("CC refused on CC control: %v", err)
	}
	reading, err := s.CommissionerPeople(t.Context(), "native.test", "creator")
	if err != nil || reading.Revision != 4 || reading.Seats[0].Name != "Falcons" || len(reading.Roles) != 3 {
		t.Fatalf("wrong authorized projection or refusal appended: %+v %v", reading, err)
	}
}

func TestDeputyPermissionsPreviewReplayRevocationAndPrivacy(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
		t.Fatal(err)
	}
	executePeople(t, s, "creator", "commish.role.grant",
		commissioner.RawPeople{Principal: "deputy", Role: governance.Deputy}, 1)

	raw := ruleCommand("roster.limits", `{"min":20,"max":70}`, "next_rollover", "", 2)
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "deputy", nil); err == nil {
		t.Fatal("unticked deputy changed a rule")
	}
	if got := revision(t, s); got != 2 {
		t.Fatalf("refused rule change advanced revision to %d", got)
	}

	permissionArgs := commissioner.RawPeople{Principal: "deputy", Permissions: []governance.DeputyPermission{governance.ChangeRules}}
	permissionCommand := peopleCommand(commissioner.DeputyPermissionsSet, permissionArgs, 2)
	permissionPreview, err := s.PreviewCommissioner(t.Context(), permissionCommand, "creator", nil, commissioner.PreviewCatalogue{})
	if err != nil || len(permissionPreview.Fields) != 1 || permissionPreview.Fields[0].Before != "No permissions" ||
		permissionPreview.Fields[0].After != "Change rules" {
		t.Fatalf("deputy permission change preview: %+v %v", permissionPreview, err)
	}
	executePeople(t, s, "creator", commissioner.DeputyPermissionsSet, permissionArgs, 2)
	records, err := s.HeldCommissionerRecords(t.Context(), "native.test")
	if err != nil {
		t.Fatal(err)
	}
	state, err := commissioner.GovernanceState(records)
	if err != nil || !state.HasDeputyPermission("deputy", governance.ChangeRules) {
		t.Fatalf("permission did not replay from ledger: %v", err)
	}
	previewRaw := ruleCommand("roster.limits", `{"min":20,"max":70}`, "next_rollover", "", 3)
	preview, err := s.PreviewCommissioner(t.Context(), previewRaw, "deputy", nil, commissioner.PreviewCatalogue{})
	if err != nil || len(preview.Blockers) != 0 {
		t.Fatalf("ticked deputy could not preview the rule change: %+v %v", preview, err)
	}
	if _, err := s.ExecuteCommissioner(t.Context(), previewRaw, "deputy", nil); err != nil {
		t.Fatalf("ticked deputy could not execute the rule change: %v", err)
	}
	if len(ruleRow(t, s, "roster.limits").Pending) != 1 {
		t.Fatal("deputy rule change was not read back")
	}

	commissionerReading, err := s.CommissionerPeople(t.Context(), "native.test", "creator")
	if err != nil {
		t.Fatal(err)
	}
	if len(commissionerReading.YourPermissions) != len(governance.AllDeputyPermissions()) {
		t.Fatalf("commissioner effective permissions: %v", commissionerReading.YourPermissions)
	}
	found := false
	for _, actor := range commissionerReading.Actors {
		if actor.Person == "deputy" {
			found = len(actor.EffectivePermissions) == 1 && actor.EffectivePermissions[0] == governance.ChangeRules
		}
	}
	if !found {
		t.Fatalf("commissioner reading omitted deputy permissions: %+v", commissionerReading.Actors)
	}
	deputyReading, err := s.CommissionerPeople(t.Context(), "native.test", "deputy")
	if err != nil || len(deputyReading.YourPermissions) != 1 || deputyReading.YourPermissions[0] != governance.ChangeRules ||
		!deputyReading.MayChangeRules || len(deputyReading.Actors) != 0 {
		t.Fatalf("deputy read exposed wrong effective permissions: %+v %v", deputyReading, err)
	}
	visitorReading, err := s.CommissionerPeople(t.Context(), "native.test", "visitor")
	if err != nil || len(visitorReading.Actors) != 0 || len(visitorReading.Roles) != 0 {
		t.Fatalf("deputy grants leaked to a visitor: %+v %v", visitorReading, err)
	}

	executePeople(t, s, "creator", commissioner.DeputyPermissionsSet,
		commissioner.RawPeople{Principal: "deputy", Permissions: []governance.DeputyPermission{}}, revision(t, s))

	raw = ruleCommand("roster.limits", `{"min":22,"max":70}`, "next_rollover", "", revision(t, s))
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "deputy", nil); err == nil {
		t.Fatal("revoked tick remained effective")
	}
	if got := revision(t, s); got != 5 {
		t.Fatalf("refused post-revocation change advanced revision to %d", got)
	}
	deputyReading, err = s.CommissionerPeople(t.Context(), "native.test", "deputy")
	if err != nil || len(deputyReading.YourPermissions) != 0 || deputyReading.MayChangeRules {
		t.Fatalf("revoked permissions remained visible: %+v %v", deputyReading, err)
	}
}

func TestCommissionerPeoplePrivacyAndRevert(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
		t.Fatal(err)
	}
	executePeople(t, s, "creator", "commish.role.grant",
		commissioner.RawPeople{Principal: "secret-cc", Role: governance.CoCommissioner}, 1)
	invite := executePeople(t, s, "creator", "commish.seat.invite",
		commissioner.RawPeople{ID: "secret-invite", Franchise: "franchise.01", Principal: "owner"}, 2)
	for _, principal := range []string{"creator", "secret-cc", "owner"} {
		people, err := s.CommissionerPeople(t.Context(), "native.test", principal)
		if err != nil {
			t.Fatal(err)
		}
		ledger, err := s.CommissionerLedger(t.Context(), "native.test", principal, nil)
		if err != nil {
			t.Fatal(err)
		}
		if principal == "owner" {
			if len(people.Roles) != 0 || len(people.Invites) != 0 || len(ledger.Entries) != 1 {
				t.Fatalf("owner private leak: %+v %+v", people, ledger)
			}
		} else if len(people.Roles) != 2 || len(people.Invites) != 1 || len(ledger.Entries) != 3 {
			t.Fatalf("C/CC private reading: %+v %+v", people, ledger)
		}
	}
	raw := reverseCommand(invite.Entry, 3, "undo.invite")
	raw.Visibility = "private"
	undo, err := s.ExecuteCommissioner(t.Context(), raw, "secret-cc", nil)
	if err != nil {
		t.Fatal(err)
	}
	people, err := s.CommissionerPeople(t.Context(), "native.test", "creator")
	if err != nil || len(people.Invites) != 0 {
		t.Fatalf("compensated invitation remains: %+v %v", people, err)
	}
	raw = reverseCommand(undo.Entry, 4, "redo.invite")
	raw.Visibility = "private"
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
		t.Fatal(err)
	}
	people, err = s.CommissionerPeople(t.Context(), "native.test", "creator")
	if err != nil || len(people.Invites) != 1 {
		t.Fatalf("reversed reversal missing invitation: %+v %v", people, err)
	}
}

func TestCommissionerInviteSessionAndRetry(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
		t.Fatal(err)
	}
	executePeople(t, s, "creator", "commish.seat.invite",
		commissioner.RawPeople{ID: "invite", Franchise: "franchise.01", Principal: "owner"}, 1)
	raw := peopleCommand("commish.seat.invite.redeem", commissioner.RawPeople{ID: "invite", Principal: "owner"}, 2)
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err == nil ||
		!strings.Contains(err.Error(), "Other owners act from their own app; this desktop can only act as you") {
		t.Fatalf("forged redeemer: %v", err)
	}
	first, err := s.ExecuteCommissioner(t.Context(), raw, "owner", nil)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := s.ExecuteCommissioner(t.Context(), raw, "owner", nil)
	if err != nil || retry != first {
		t.Fatalf("successful redemption is not retryable: %+v %v", retry, err)
	}
	raw.Expected, raw.Request = 3, "used-invite"
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "owner", nil); err == nil {
		t.Fatal("consumed invitation accepted again")
	}
	people, err := s.CommissionerPeople(t.Context(), "native.test", "owner")
	if err != nil || people.Revision != 3 || people.Seats[0].Owner != "owner" {
		t.Fatalf("redeemed seat: %+v %v", people, err)
	}
}

func TestCommissionerPeopleCommandsAndRestart(t *testing.T) {
	t.Parallel()
	s, _, pools := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
		t.Fatal(err)
	}
	changes := []struct {
		command string
		args    commissioner.RawPeople
	}{
		{"commish.seat.replace", commissioner.RawPeople{Franchise: "franchise.01", Principal: "owner"}},
		{"commish.seat.remove", commissioner.RawPeople{Franchise: "franchise.01"}},
		{"commish.abilities.set", commissioner.RawPeople{Franchise: "franchise.01", Ability: "trades", On: false}},
		{"commish.visibility.set", commissioner.RawPeople{Report: "finances", Audience: "commish"}},
		{"commish.role.grant", commissioner.RawPeople{Principal: "judge", Role: governance.DOT}},
		{"commish.role.revoke", commissioner.RawPeople{Principal: "judge", Role: governance.DOT}},
		{"commish.seat.invite", commissioner.RawPeople{ID: "inv", Franchise: "franchise.01", Principal: "owner"}},
		{"commish.seat.invite.revoke", commissioner.RawPeople{ID: "inv"}},
	}
	for i, change := range changes {
		executePeople(t, s, "creator", change.command, change.args, int64(i+1))
	}
	events, err := s.Events(t.Context(), "native.test", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	records, err := commissionerRecords(events)
	if err != nil {
		t.Fatal(err)
	}
	state, err := commissioner.GovernanceState(records)
	if err != nil {
		t.Fatal(err)
	}
	f, _ := state.Franchise("franchise.01")
	if f.Seat != "" || f.Abilities["trades"] || state.Audience("finances") != "commish" ||
		state.HasRole("judge", governance.DOT) {
		t.Fatalf("commands missing from projection: %+v", f)
	}
	before, err := s.CommissionerPeople(t.Context(), "native.test", "creator")
	if err != nil || before.DOTStatus != "Needs a ruling: trade review needs at least one seat" ||
		len(before.Invites) != 1 || before.Invites[0].Status != "Revoked" {
		t.Fatalf("people projection: %+v %v", before, err)
	}
	ledger, err := s.CommissionerLedger(t.Context(), "native.test", "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	var path string
	if err := pools.Write().QueryRowContext(t.Context(),
		`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&path); err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := db.Open(t.Context(), path, SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	s.pools = reopened
	after, err := s.CommissionerPeople(t.Context(), "native.test", "creator")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("restart people differ: %+v %+v %v", before, after, err)
	}
	afterLedger, err := s.CommissionerLedger(t.Context(), "native.test", "creator", nil)
	if err != nil || !reflect.DeepEqual(ledger, afterLedger) {
		t.Fatalf("restart ledger differs: %+v %v", afterLedger, err)
	}
}
