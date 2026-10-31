package composition

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"
)

func movesFor(t *testing.T, f *waiverFixture, who, franchise string) market.NativeMoves {
	t.Helper()
	f.app.core.CommissionerSession = testCommissionerSession{who}
	r, err := newCommissionerAPI(f.app).TargetNativeMoves(legalityLeague, franchise)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Planned) != 0 || r.Planned == nil || r.Reason != "native plans not yet stored" {
		t.Fatalf("invented native plans: %+v", r)
	}
	return r
}

func moveIn(t *testing.T, r market.NativeMoves, id, state string) market.NativeMove {
	t.Helper()
	for _, m := range r.Moves {
		if m.CorrelationID == id {
			if m.State != state {
				t.Fatalf("move %s: got %s, want %s", id, m.State, state)
			}
			return m
		}
	}
	t.Fatalf("move %s missing: %+v", id, r)
	return market.NativeMove{}
}

func noMove(t *testing.T, r market.NativeMoves, id string) {
	t.Helper()
	body, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), id) {
		t.Fatalf("private or inactive move leaked: %s", body)
	}
}

func TestNativeMovesTradeLifecycleAndPairIsolation(t *testing.T) {
	f := tradeApp(t)
	if r := movesFor(t, f.waiverFixture, "owner.one", "franchise.01"); len(r.Moves) != 0 {
		t.Fatalf("planned became a submitted move: %+v", r)
	}
	offer := f.offer(t)
	moveIn(t, movesFor(t, f.waiverFixture, "owner.one", "franchise.01"), offer.Entry, "needs_you")
	moveIn(t, movesFor(t, f.waiverFixture, "owner.two", "franchise.02"), offer.Entry, "handed_off")
	noMove(t, movesFor(t, f.waiverFixture, "owner.three", "franchise.03"), offer.Entry)
	f.now = f.now.Add(time.Minute)
	f.accept(t, offer)
	m := moveIn(t, movesFor(t, f.waiverFixture, "owner.one", "franchise.01"), offer.Entry, "handed_off")
	if m.Spec.Deadline != nil || !m.Audit[0].At.Equal(f.now) || !strings.Contains(m.Audit[0].Note, "review") {
		t.Fatalf("review has wrong evidence: %+v", m)
	}
	for _, who := range []string{"dot.a", "dot.b"} {
		f.vote(t, offer, who, "approve")
	}
	moveIn(t, movesFor(t, f.waiverFixture, "owner.one", "franchise.01"), offer.Entry, "handed_off")
	f.now = f.now.Add(time.Minute)
	approval := f.vote(t, offer, "dot.c", "approve")
	landed := moveIn(t, movesFor(t, f.waiverFixture, "owner.one", "franchise.01"), offer.Entry, "landed")
	if !landed.Audit[0].At.Equal(approval.RecordedAt) || f.ownerOf(t, "1000") != "franchise.01" {
		t.Fatal("landing lacks committed asset transfer")
	}
	f.restart(t)
	if after := moveIn(t, movesFor(t, f.waiverFixture, "owner.one", "franchise.01"), offer.Entry, "landed"); !reflect.DeepEqual(landed, after) {
		t.Fatal("move changed after restart")
	}
	f.revert(t, approval)
	moveIn(t, movesFor(t, f.waiverFixture, "owner.one", "franchise.01"), offer.Entry, "handed_off")
}

func TestNativeMovesBidsLifecycleSettingsAndIsolation(t *testing.T) {
	f := bidApp(t)
	bid := f.place(t, "owner.one", "franchise.01", "5001", 5*million, 1)
	before := moveIn(t, movesFor(t, f.waiverFixture, "owner.one", "franchise.01"), bid.Entry, "handed_off")
	if before.Spec.Deadline == nil {
		t.Fatal("open bid lacks its configured close")
	}
	f.set(t, "bids.standFor", `172800000000000`, "now")
	after := moveIn(t, movesFor(t, f.waiverFixture, "owner.one", "franchise.01"), bid.Entry, "handed_off")
	if after.Spec.Deadline == nil || after.Spec.Deadline.Sub(*before.Spec.Deadline) != 24*time.Hour {
		t.Fatal("bid close ignored changed stand time")
	}
	f.set(t, "bids.ufa", `"blind"`, "now")
	blind := moveIn(t, movesFor(t, f.waiverFixture, "owner.one", "franchise.01"), bid.Entry, "handed_off")
	if blind.Spec.Deadline != nil {
		t.Fatal("blind auction leaked a close")
	}
	other := movesFor(t, f.waiverFixture, "owner.two", "franchise.02")
	noMove(t, other, bid.Entry)
	if len(other.Moves) != 0 {
		t.Fatal("blind auction existence leaked")
	}
	f.now = f.now.Add(48 * time.Hour)
	award := f.decide(t, "5001")
	landed := moveIn(t, movesFor(t, f.waiverFixture, "owner.one", "franchise.01"), bid.Entry, "landed")
	if !landed.Audit[0].At.Equal(award.RecordedAt) || !strings.Contains(landed.Audit[0].Note, award.Entry) {
		t.Fatal("won bid lacks ledger evidence")
	}
	f.revert(t, award)
	moveIn(t, movesFor(t, f.waiverFixture, "owner.one", "franchise.01"), bid.Entry, "handed_off")
}

func TestNativeMovesWaiverLifecycleSettingsAndIsolation(t *testing.T) {
	f := waiverApp(t)
	cut := f.cut(t, "1000")
	moveIn(t, movesFor(t, f, "owner.two", "franchise.02"), cut.Entry, "landed")
	claim := f.claim(t, cut.Entry, "owner.one", "franchise.01")
	before := moveIn(t, movesFor(t, f, "owner.one", "franchise.01"), claim.Entry, "handed_off")
	noMove(t, movesFor(t, f, "owner.three", "franchise.03"), claim.Entry)
	f.set(t, "waivers.duration", `172800000000000`, "now")
	after := moveIn(t, movesFor(t, f, "owner.one", "franchise.01"), claim.Entry, "handed_off")
	if before.Spec.Deadline == nil || after.Spec.Deadline == nil || after.Spec.Deadline.Sub(*before.Spec.Deadline) != 24*time.Hour {
		t.Fatal("waiver close ignored changed duration")
	}
	f.now = f.now.Add(48 * time.Hour)
	award := f.process(t, cut.Entry)
	landed := moveIn(t, movesFor(t, f, "owner.one", "franchise.01"), claim.Entry, "landed")
	if !landed.Audit[0].At.Equal(award.RecordedAt) || !strings.Contains(landed.Audit[0].Note, award.Entry) {
		t.Fatal("claim award lacks committed evidence")
	}
	f.revert(t, award)
	moveIn(t, movesFor(t, f, "owner.one", "franchise.01"), claim.Entry, "handed_off")
}

func TestNativeMovesRefusesOtherFranchiseAndLeague(t *testing.T) {
	f := waiverApp(t)
	f.app.core.CommissionerSession = testCommissionerSession{"owner.one"}
	for _, scope := range []struct{ league, franchise string }{{legalityLeague, "franchise.02"}, {"native.other", "franchise.01"}, {"12345", "franchise.01"}} {
		if _, err := newCommissionerAPI(f.app).TargetNativeMoves(scope.league, scope.franchise); err == nil {
			t.Fatalf("unauthorized scope admitted: %+v", scope)
		}
	}
}

func TestNativeMovesWire(t *testing.T) {
	f := nativeMarketApp(t)
	bid := f.place(t, "owner.one", "franchise.01", "5001", 5*million, 1)
	f.cut(t, "1001")
	f.claim(t, f.reading(t, "owner.one", "franchise.01").Players[0].Window, "owner.one", "franchise.01")
	trades := &tradeFixture{f.waiverFixture}
	trades.offer(t)
	reading := movesFor(t, f.waiverFixture, "owner.one", "franchise.01")
	moveIn(t, reading, bid.Entry, "handed_off")
	body, err := json.MarshalIndent(reading, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	const path = "frontend/src/app/data/native-moves.wire.testdata.json"
	if os.Getenv("UPDATE_NATIVE_MOVES_WIRE") == "1" {
		if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(want)) != string(body) {
		t.Fatal("native moves wire stale")
	}
}
