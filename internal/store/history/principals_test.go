package history

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func aliasContext(t *testing.T, league string, account governance.Principal, aliases ...string) context.Context {
	t.Helper()
	verified := make([]governance.VerifiedAlias, 0, len(aliases))
	for _, alias := range aliases {
		verified = append(verified, governance.VerifiedAlias{League: league, Alias: governance.Principal(alias), Account: account})
	}
	principals, err := governance.NewPrincipals(league, verified)
	if err != nil {
		t.Fatal(err)
	}
	return commissioner.WithPrincipals(t.Context(), principals)
}

func ledgerBytes(t *testing.T, s *Store) []byte {
	t.Helper()
	rows, err := s.pools.Read().QueryContext(t.Context(), `SELECT actor,payload,input_hash FROM league_event WHERE league=? AND stream=? ORDER BY revision`, "native.test", commissioner.Stream)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var stored []struct{ Actor, Payload, Hash string }
	for rows.Next() {
		var event struct{ Actor, Payload, Hash string }
		if err := rows.Scan(&event.Actor, &event.Payload, &event.Hash); err != nil {
			t.Fatal(err)
		}
		stored = append(stored, event)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestAccountRetriesEveryVerifiedAliasWithoutChangingHistory(t *testing.T) {
	t.Parallel()
	const account = "acct.12345678-1234-4234-8234-123456789abc"
	s, _, _ := newStore(t)
	created, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	grant := peopleCommand("commish.role.grant", commissioner.RawPeople{Principal: "second", Role: governance.CoCommissioner}, 1)
	first, err := s.ExecuteCommissioner(t.Context(), grant, "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	edit := peopleCommand("commish.franchise.edit", commissioner.RawPeople{Franchise: "franchise.01", Name: "Falcons", Abbreviation: "FAL"}, 2)
	// The same account historically acted under two independently verified identities.
	second, err := s.ExecuteCommissioner(t.Context(), edit, "second", nil)
	if err != nil {
		t.Fatal(err)
	}
	before := ledgerBytes(t, s)
	ctx := aliasContext(t, "native.test", account, "creator", "second")
	for _, tc := range []struct {
		raw     commissioner.RawCommand
		receipt commissioner.Receipt
	}{{createCommand(), created}, {grant, first}, {edit, second}} {
		got, err := s.ExecuteCommissioner(ctx, tc.raw, account, nil)
		if err != nil || !reflect.DeepEqual(got, tc.receipt) {
			t.Fatalf("alias retry: %+v %v; want %+v", got, err, tc.receipt)
		}
	}
	changed := edit
	changed.Reason = "different request body"
	if _, err := s.ExecuteCommissioner(ctx, changed, account, nil); err == nil {
		t.Fatal("changed retry accepted")
	}
	unverified := aliasContext(t, "native.test", account, "creator")
	if _, err := s.ExecuteCommissioner(unverified, edit, account, nil); err == nil {
		t.Fatal("unverified alias retry accepted")
	}
	denied := commissioner.WithCredentialAdmission(ctx, denyAliasCredential{})
	if _, err := s.ExecuteCommissioner(denied, edit, account, nil); err == nil {
		t.Fatal("revoked retry accepted")
	}
	if !reflect.DeepEqual(before, ledgerBytes(t, s)) {
		t.Fatal("retry rewrote original history")
	}
}

type denyAliasCredential struct{}

func (denyAliasCredential) Admit(context.Context, commissioner.Access) error {
	return errors.New("revoked")
}

func TestAliasGrantsOwnershipPrivacyAndRevocation(t *testing.T) {
	t.Parallel()
	const account = "acct.12345678-1234-4234-8234-123456789abc"
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
		t.Fatal(err)
	}
	executePeople(t, s, "creator", "commish.role.grant", commissioner.RawPeople{Principal: "legacy-owner", Role: governance.CoCommissioner}, 1)
	executePeople(t, s, "creator", "commish.seat.replace", commissioner.RawPeople{Principal: "legacy-owner", Franchise: "franchise.01"}, 2)
	before := ledgerBytes(t, s)
	ctx := aliasContext(t, "native.test", account, "legacy-owner", "other-install")
	people, err := s.CommissionerPeople(ctx, "native.test", account)
	if err != nil || people.YourRole != "Co-commissioner" || people.YourFranchise != "franchise.01" || len(people.Roles) != 2 {
		t.Fatalf("alias grants/seat: %+v %v", people, err)
	}
	records, err := s.nativeRecords(ctx, "native.test")
	if err != nil {
		t.Fatal(err)
	}
	if err := commissioner.AuthorizeFranchiseOwner(records, account, "franchise.01"); err != nil {
		t.Fatal(err)
	}
	if err := commissioner.AuthorizeFranchiseOwner(records, account, "franchise.02"); err == nil {
		t.Fatal("another franchise allowed")
	}
	ledger, err := s.CommissionerLedger(ctx, "native.test", account, nil)
	if err != nil || len(ledger.Entries) != 3 {
		t.Fatalf("private alias reading: %+v %v", ledger, err)
	}
	outsider, err := s.CommissionerPeople(ctx, "native.test", "unverified")
	if err != nil || len(outsider.Roles) != 0 || outsider.YourFranchise != "" {
		t.Fatalf("outsider: %+v %v", outsider, err)
	}
	wrong := aliasContext(t, "native.other", account, "legacy-owner")
	if _, err := s.CommissionerPeople(wrong, "native.test", account); err == nil {
		t.Fatal("cross-league mapping read allowed")
	}
	edit := peopleCommand("commish.franchise.edit", commissioner.RawPeople{Franchise: "franchise.01", Name: "Falcons", Abbreviation: "FAL"}, 3)
	if _, err := s.ExecuteCommissioner(wrong, edit, account, nil); err == nil {
		t.Fatal("cross-league mapping write allowed")
	}
	if !reflect.DeepEqual(before, ledgerBytes(t, s)) {
		t.Fatal("read/refusal changed history")
	}
	if _, err := s.ExecuteCommissioner(ctx, edit, account, nil); err != nil {
		t.Fatalf("account mutation using legacy grant: %v", err)
	}
	events, err := s.Events(t.Context(), "native.test", commissioner.Stream, 3)
	if err != nil || len(events) != 1 || events[0].Actor != account || events[0].Payload.Commissioner.Principal != account {
		t.Fatalf("new account author not recorded: %+v %v", events, err)
	}
	revoke := peopleCommand("commish.role.revoke", commissioner.RawPeople{Principal: account, Role: governance.CoCommissioner}, 4)
	if _, err := s.ExecuteCommissioner(ctx, revoke, "creator", nil); err != nil {
		t.Fatal(err)
	}
	after := ledgerBytes(t, s)
	edit.Expected, edit.Request = 5, "after-revocation"
	if _, err := s.ExecuteCommissioner(ctx, edit, account, nil); err == nil {
		t.Fatal("alias retained revoked grant")
	}
	if !reflect.DeepEqual(after, ledgerBytes(t, s)) {
		t.Fatal("revoked grant appended")
	}
}
