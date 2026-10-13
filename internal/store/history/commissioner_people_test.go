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
	if strings.HasPrefix(command, "commish.role.") || strings.HasPrefix(command, "commish.seat.invite") {
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
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err == nil ||
		!strings.Contains(err.Error(), "This league already has a co-commissioner: revoke them first") {
		t.Fatalf("second CC: %v", err)
	}
	raw = peopleCommand("commish.franchise.edit",
		commissioner.RawPeople{Franchise: "f1", Name: "Falcons", Abbreviation: "FAL"}, 2)
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "owner", nil); err == nil {
		t.Fatal("owner changed a franchise identity")
	}
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "cc", nil); err != nil {
		t.Fatalf("CC refused on CC control: %v", err)
	}
	reading, err := s.CommissionerPeople(t.Context(), "native.test", "creator")
	if err != nil || reading.Revision != 3 || reading.Seats[0].Name != "Falcons" || len(reading.Roles) != 2 {
		t.Fatalf("wrong authorized projection or refusal appended: %+v %v", reading, err)
	}
}

func TestCommissionerPeoplePrivacyAndRevert(t *testing.T) {
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
		t.Fatal(err)
	}
	executePeople(t, s, "creator", "commish.role.grant",
		commissioner.RawPeople{Principal: "secret-cc", Role: governance.CoCommissioner}, 1)
	invite := executePeople(t, s, "creator", "commish.seat.invite",
		commissioner.RawPeople{ID: "secret-invite", Franchise: "f1", Principal: "owner"}, 2)
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
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
		t.Fatal(err)
	}
	executePeople(t, s, "creator", "commish.seat.invite",
		commissioner.RawPeople{ID: "invite", Franchise: "f1", Principal: "owner"}, 1)
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
	s, _, pools := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
		t.Fatal(err)
	}
	changes := []struct {
		command string
		args    commissioner.RawPeople
	}{
		{"commish.seat.replace", commissioner.RawPeople{Franchise: "f1", Principal: "owner"}},
		{"commish.seat.remove", commissioner.RawPeople{Franchise: "f1"}},
		{"commish.abilities.set", commissioner.RawPeople{Franchise: "f1", Ability: "trades", On: false}},
		{"commish.visibility.set", commissioner.RawPeople{Report: "finances", Audience: "commish"}},
		{"commish.role.grant", commissioner.RawPeople{Principal: "judge", Role: governance.DOT}},
		{"commish.role.revoke", commissioner.RawPeople{Principal: "judge", Role: governance.DOT}},
		{"commish.seat.invite", commissioner.RawPeople{ID: "inv", Franchise: "f1", Principal: "owner"}},
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
	f, _ := state.Franchise("f1")
	if f.Seat != "" || f.Abilities["trades"] || state.Audience("finances") != "commish" ||
		state.HasRole("judge", governance.DOT) {
		t.Fatalf("commands missing from projection: %+v", f)
	}
	before, err := s.CommissionerPeople(t.Context(), "native.test", "creator")
	if err != nil || before.DOTStatus != "Needs a ruling: DOT has 0 of 4 seats" ||
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
	reopened, err := db.Open(t.Context(), path)
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
