package market

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func principalRecords(t *testing.T, records []commissioner.Record, aliases ...string) ([]commissioner.Record, string) {
	t.Helper()
	const account = "acct.12345678-1234-4234-8234-123456789abc"
	verified := make([]governance.VerifiedAlias, 0, len(aliases))
	for _, alias := range aliases {
		verified = append(verified, governance.VerifiedAlias{League: "native.test", Alias: governance.Principal(alias), Account: account})
	}
	principals, err := governance.NewPrincipals("native.test", verified)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := commissioner.ResolvePrincipals(commissioner.WithPrincipals(t.Context(), principals), "native.test", records)
	if err != nil {
		t.Fatal(err)
	}
	return resolved, account
}

func TestAliasPrivateWaiverClaims(t *testing.T) {
	b := newLedger(t, commissioner.PresetEmpty)
	records, account := principalRecords(t, b.records, "legacy-owner", "other-install")
	window := commissioner.WaiverWindow{ID: "window", Claims: []commissioner.WaiverBookClaim{{ID: "claim", Franchise: "one", Principal: "legacy-owner"}}}
	claims := appendOwnWaiverClaims(nil, window, "one", account, records)
	if len(claims) != 1 || claims[0].ID != "claim" {
		t.Fatalf("own alias claim hidden: %+v", claims)
	}
	if got := appendOwnWaiverClaims(nil, window, "one", "unverified", records); len(got) != 0 {
		t.Fatal("claim leaked")
	}
}
