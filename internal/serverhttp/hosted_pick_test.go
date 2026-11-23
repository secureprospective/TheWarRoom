package serverhttp_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

// TestHostedOwnerRookiePickAdmitted is B7: the on-clock franchise's rookie pick through the owner door
// (PickPath) must be admitted. The rookie's entry season comes from the league's pinned football, so the
// door has to judge the pick by that evidence, as the workflow reads it, not by an empty host.
func TestHostedOwnerRookiePickAdmitted(t *testing.T) {
	f := newHostedFixture(t, true)
	// The seed starts the 2027 rookie draft two seconds after it was built; the clock passes that start here.
	f.skew.Add(int64(3 * time.Second))
	bearer, proof := f.directCommissioner(t)
	// The owner's expected revision is the league version its own reads return (the ledger read here).
	ledgerBody := f.post(t, serverhttp.LedgerPath, `{"protocol":1,"league":"native.hosted"}`, bearer, proof, http.StatusOK)
	var ledger commissioner.Ledger
	if err := json.Unmarshal(ledgerBody, &ledger); err != nil {
		t.Fatal(err)
	}
	pick := commissioner.DraftPickInput{League: "native.hosted", Season: 2027, Franchise: "franchise.01", Player: "5001",
		Request: "hosted-pick.5001", Expected: ledger.Revision}
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Pick: &pick})
	if err != nil {
		t.Fatal(err)
	}
	f.post(t, serverhttp.PickPath, string(body), bearer, proof, http.StatusOK)
	assets, err := f.history.NativeAssets(t.Context(), "native.hosted")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(assets.Snapshot.Players, func(p commissioner.NativePlayer) bool {
		return p.ID == "5001" && p.Franchise == "franchise.01"
	}) {
		t.Fatal("admitted pick did not put rookie 5001 on franchise.01")
	}
}
