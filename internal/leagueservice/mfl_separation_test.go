package leagueservice

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// TestMflSeparationBidCheckAndPlanSaveAreLocal is the server twin of composition's
// TestMFLWhatIfAndNativeRulesRemainSeparateWithoutMFLWrites (desktop removal PR-B): an owner's
// My MFL moves check and saved plan never call MFL, never touch the native league's ledger and
// never change the held MFL copy.
func TestMflSeparationBidCheckAndPlanSaveAreLocal(t *testing.T) {
	f := mflOwnerFixture(t, nil)
	ctx := t.Context()
	calls := f.mfl.calls.Load()
	events, err := f.store.Events(ctx, f.league, commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	held, ok, err := f.mirror.Read(ctx, "14432")
	if err != nil || !ok {
		t.Fatalf("held MFL copy: ok=%v err=%v", ok, err)
	}
	got, err := f.service.MflMovesBidCheckContext(ctx, RawMflBidCheck{League: f.league, Add: "0531", AmountCents: 100_000_000, Years: 1})
	if err != nil || !got.OK {
		t.Fatalf("bid check = %+v, %v", got, err)
	}
	if _, err := saveBid(t, f, "0531", 1_000_000); err != nil {
		t.Fatal(err)
	}
	if now := f.mfl.calls.Load(); now != calls {
		t.Fatalf("bid check and plan save asked MFL: %d -> %d calls", calls, now)
	}
	after, err := f.store.Events(ctx, f.league, commissioner.Stream, 0)
	if err != nil || len(after) != len(events) {
		t.Fatalf("native ledger changed: %d -> %d events (%v)", len(events), len(after), err)
	}
	reread, _, err := f.mirror.Read(ctx, "14432")
	if err != nil || !reflect.DeepEqual(reread, held) {
		t.Fatalf("held MFL copy changed (%v)", err)
	}
}
