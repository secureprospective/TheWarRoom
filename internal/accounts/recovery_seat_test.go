package accounts_test

import (
	"errors"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestCommissionerSeatReassignmentCannotResetAccount(t *testing.T) {
	f, recovery, codes, proof := recoveryFixture(t)
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	ledger := history.New(f.pools, registry)
	if err := ledger.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	principal, err := f.account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	create := commissioner.RawCommand{League: "native.recovery", Command: commissioner.Create, Args: commissioner.RawArgs{Name: "Recovery fixture", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2}, Reason: "Create fixture", Request: "create", Effective: "now", Visibility: "league"}
	if _, err := ledger.ExecuteCommissioner(t.Context(), create, "fixture.commissioner", nil); err != nil {
		t.Fatal(err)
	}
	assign := create
	assign.Command = "commish.seat.replace"
	assign.Args = commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.01", Principal: string(principal)}}
	assign.Request = "assign"
	assign.Expected = 1
	if _, err := ledger.ExecuteCommissioner(t.Context(), assign, "fixture.commissioner", nil); err != nil {
		t.Fatal(err)
	}
	assign.Args.People = &commissioner.RawPeople{Franchise: "franchise.01", Principal: "replacement.owner"}
	assign.Request = "replace"
	assign.Expected = 2
	if _, err := ledger.ExecuteCommissioner(t.Context(), assign, "fixture.commissioner", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := accounts.New(f.store).Authenticate(t.Context(), f.credentials); err != nil {
		t.Fatalf("seat replacement changed account: %v", err)
	}
	if err := f.store.CheckRecentAuth(t.Context(), f.account.ID, digestToken(proof.Token)); err != nil {
		t.Fatalf("seat replacement removed proof: %v", err)
	}
	if err := recovery.ConsoleRecover(t.Context(), accounts.RawConsoleRecovery{Username: f.account.Username, Password: "commissioner-chosen-password", Reason: "Reassigned seat"}, nil); !errors.Is(err, accounts.ErrRecoveryRefused) {
		t.Fatalf("seat authority reset account: %v", err)
	}
	generation, err := f.store.AuthGeneration(t.Context(), f.account.ID)
	if err != nil || generation != 0 {
		t.Fatalf("seat replacement revoked account credentials: %d %v", generation, err)
	}
	// The original account retains recovery authority even after losing its seat.
	if err := recovery.Recover(t.Context(), accounts.RawRecovery{Username: f.account.Username, Password: "owner-chosen-password", Code: codes.Codes[0]}); err != nil {
		t.Fatal(err)
	}
	reading, err := ledger.CommissionerPeople(t.Context(), create.League, "replacement.owner")
	if err != nil || reading.YourFranchise != "franchise.01" {
		t.Fatalf("account recovery changed the seat: %+v %v", reading, err)
	}
}
