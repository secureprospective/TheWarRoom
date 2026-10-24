package accounts_test

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestAccountEnrollmentDoesNotGrantOrReplaceSeat(t *testing.T) {
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "server.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	store := accountstore.New(pools)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	issuer, err := accounts.New(store).LocalBootstrap(t.Context(), accounts.RawCredentials{Username: "commissioner", Password: "private-issuer-password"})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := issuer.Principal()
	if err != nil {
		t.Fatal(err)
	}
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	ledger := history.New(pools, registry)
	if err := ledger.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	create := commissioner.RawCommand{League: "native.test", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Test league", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2},
		Reason: "Start fixture league", Request: "create", Effective: "now", Visibility: "league"}
	if _, err := ledger.ExecuteCommissioner(t.Context(), create, string(principal), nil); err != nil {
		t.Fatal(err)
	}
	invite := commissioner.RawCommand{League: create.League, Command: "commish.seat.invite", Args: commissioner.RawArgs{People: &commissioner.RawPeople{ID: "seat-invite", Franchise: "franchise.01", Principal: "existing.owner"}},
		Reason: "Invite existing owner", Request: "invite", Effective: "now", Visibility: "private", Expected: 1}
	if _, err := ledger.ExecuteCommissioner(t.Context(), invite, string(principal), nil); err != nil {
		t.Fatal(err)
	}
	redeem := invite
	redeem.Command = "commish.seat.invite.redeem"
	redeem.Args.People = &commissioner.RawPeople{ID: "seat-invite", Principal: "existing.owner"}
	redeem.Request = "seat-redeem"
	redeem.Expected = 2
	if _, err := ledger.ExecuteCommissioner(t.Context(), redeem, "existing.owner", nil); err != nil {
		t.Fatal(err)
	}
	before, err := ledger.CommissionerLedger(t.Context(), create.League, string(principal), nil)
	if err != nil {
		t.Fatal(err)
	}
	peopleBefore, err := ledger.CommissionerPeople(t.Context(), create.League, string(principal))
	if err != nil {
		t.Fatal(err)
	}
	policy, err := governance.Apply(governance.State{}, governance.LeagueCreated{Creator: principal, Franchises: []governance.FranchiseID{"franchise.01", "franchise.02"}})
	if err != nil {
		t.Fatal(err)
	}
	service := accounts.NewEnrollment(store, &enrollmentAuthority{policy: policy, principal: principal})
	code, err := service.Issue(t.Context(), create.League, "new.owner")
	if err != nil {
		t.Fatal(err)
	}
	raw := accounts.RawEnrollment{Code: code.Code, Username: "existing.owner", Password: "new-owner-password"}
	if _, err := service.Redeem(t.Context(), raw); !errors.Is(err, accounts.ErrEnrollmentRefused) {
		t.Fatalf("wrong identity: %v", err)
	}
	raw.Username = "new.owner"
	account, err := service.Redeem(t.Context(), raw)
	if err != nil {
		t.Fatal(err)
	}
	after, err := ledger.CommissionerLedger(t.Context(), create.League, string(principal), nil)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("enrollment changed ledger: %v", err)
	}
	peopleAfter, err := ledger.CommissionerPeople(t.Context(), create.League, string(principal))
	if err != nil || !reflect.DeepEqual(peopleBefore, peopleAfter) {
		t.Fatalf("enrollment changed seat or grants: %v", err)
	}
	enrolledPrincipal, err := account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	reading, err := ledger.CommissionerPeople(t.Context(), create.League, string(enrolledPrincipal))
	if err != nil || reading.YourRole != "Visitor" || reading.YourFranchise != "" {
		t.Fatalf("enrollment acquired league rights: %v", err)
	}
	// A seat invitation cannot silently overwrite an occupied seat.
	stale := invite
	stale.Args.People = &commissioner.RawPeople{ID: "new-seat-invite", Franchise: "franchise.01", Principal: string(enrolledPrincipal)}
	stale.Request = "occupied-seat"
	stale.Expected = 3
	if _, err := ledger.ExecuteCommissioner(t.Context(), stale, string(principal), nil); err == nil {
		t.Fatal("occupied seat accepted a new invitation")
	}
	final, err := ledger.CommissionerLedger(t.Context(), create.League, string(principal), nil)
	if err != nil || !reflect.DeepEqual(after, final) {
		t.Fatalf("occupied seat refusal changed ledger: %v", err)
	}
}
