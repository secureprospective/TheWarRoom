package leagueservice

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type viewTestHost struct {
	StoredHost
	clock *dueTestClock
	facts nativecheck.Facts
}

func (h viewTestHost) CommissionerNow() time.Time     { return h.clock.Now() }
func (h viewTestHost) NativeFacts() nativecheck.Facts { return h.facts }

func viewService(t *testing.T, f *dueFixture, principal string) *Service {
	t.Helper()
	return NewHosted(viewTestHost{StoredHost: StoredHost{Store: f.store, Context: func() context.Context { return t.Context() },
		Principal: principal, Registry: "id,name,gm_question,governs\n"}, clock: f.clock, facts: f.facts},
		admissionFunc(func(context.Context, commissioner.Access) error { return nil }))
}

func viewTranscript(t *testing.T, service *Service) []string {
	t.Helper()
	out := []string{}
	for _, tc := range []struct {
		name string
		args []any
	}{
		{"TargetCommissionerLedger", []any{dueLeague}},
		{"TargetCommissionerLedgerFiltered", []any{dueLeague, `{"kind":"rules"}`}},
		{"TargetCommissionerPeople", []any{dueLeague}},
		{"TargetCommissionerRules", []any{dueLeague}},
		{"TargetCommissionerJobs", []any{dueLeague}},
		{"TargetCommissionerNotices", []any{dueLeague}},
		{"TargetCommissionerDigests", []any{dueLeague}},
		{"TargetCorrections", []any{dueLeague}},
		{"TargetNativeBids", []any{dueLeague, "franchise.02"}},
		{"TargetNativeWaivers", []any{dueLeague, "franchise.02", service.host.CommissionerNow()}},
		{"TargetNativeTrades", []any{dueLeague, "franchise.02"}},
		{"TargetNativeDirectory", []any{dueLeague}},
		{"TargetNativeAssets", []any{dueLeague}},
		{"TargetNativeCapCheck", []any{dueLeague, "UTC"}},
		{"TargetNativeRosterCheck", []any{dueLeague}},
		{"TargetNativeReserveCheck", []any{dueLeague}},
		{"TargetNativeHouseRules", []any{dueLeague}},
	} {
		args := make([]reflect.Value, len(tc.args))
		for i, arg := range tc.args {
			args[i] = reflect.ValueOf(arg)
		}
		result := reflect.ValueOf(service).MethodByName(tc.name).Call(args)
		if !result[1].IsNil() {
			principal, err := service.host.CommissionerPrincipal()
			if err != nil || principal != "creator" || (tc.name != "TargetNativeBids" && tc.name != "TargetNativeWaivers" && tc.name != "TargetNativeTrades") {
				t.Fatalf("%s: %v", tc.name, result[1].Interface())
			}
			out = append(out, tc.name+": "+result[1].Interface().(error).Error())
			continue
		}
		body, err := json.Marshal(result[0].Interface())
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, string(body))
	}
	return out
}

func TestHiddenBidPreservesRecipientBodiesTokensAndErrors(t *testing.T) {
	f := newDueFixture(t)
	f.execute(t, "creator", commissioner.RulesSet, commissioner.RawArgs{Field: "bids.ufa", Value: json.RawMessage(`"blind"`)})
	owner, admin := viewService(t, f, "owner.two"), viewService(t, f, "creator")
	before := viewTranscript(t, owner)
	beforeAdmin := viewTranscript(t, admin)
	ownBefore, err := viewService(t, f, "owner.one").TargetNativeBids(dueLeague, "franchise.01")
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := owner.TargetCommissionerLedger(dueLeague)
	if err != nil {
		t.Fatal(err)
	}
	cut := f.raw(t, commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0099", Franchise: "franchise.02"}})
	cut.Expected = ledger.Revision
	stale := cut
	stale.Request, stale.Expected = "stale-cut", 1
	_, refusal := owner.Execute(t.Context(), stale)
	if refusal == nil {
		t.Fatal("stale token accepted")
	}
	preview, err := owner.Preview(stale)
	if err != nil {
		t.Fatal(err)
	}
	bid := f.bid(t)
	afterAdmin := viewTranscript(t, admin)
	if !reflect.DeepEqual(before, viewTranscript(t, owner)) || !reflect.DeepEqual(beforeAdmin, afterAdmin) {
		t.Fatal("hidden bid changed an unauthorized response or token")
	}
	ownAfter, err := viewService(t, f, "owner.one").TargetNativeBids(dueLeague, "franchise.01")
	if err != nil || ownAfter.Revision == ownBefore.Revision || len(ownAfter.Players) != 1 {
		t.Fatalf("own bid not visible: %+v %v", ownAfter, err)
	}
	for _, command := range []string{commissioner.BidRaise, commissioner.BidWithdraw} {
		args := &commissioner.RawBidArgs{Bid: bid.Entry, Franchise: "franchise.01"}
		if command == commissioner.BidRaise {
			args.Salary, args.Years = 600000000, 1
		}
		bid = f.execute(t, "owner.one", command, commissioner.RawArgs{Bid: args})
		if !reflect.DeepEqual(before, viewTranscript(t, owner)) || !reflect.DeepEqual(beforeAdmin, viewTranscript(t, admin)) {
			t.Fatal("hidden bid update changed unauthorized transcript")
		}
	}
	count := len(f.events(t))
	_, afterRefusal := owner.Execute(t.Context(), stale)
	afterPreview, err := owner.Preview(stale)
	if afterRefusal == nil || refusal.Error() != afterRefusal.Error() || err != nil || !reflect.DeepEqual(preview, afterPreview) || len(f.events(t)) != count {
		t.Fatal("hidden bid changed errors/preview or refused writes appended")
	}
	receipt, err := owner.Execute(t.Context(), cut)
	if err != nil {
		t.Fatalf("invisible event conflicted: %v", err)
	}
	retry, err := owner.Execute(t.Context(), cut)
	if err != nil || !reflect.DeepEqual(receipt, retry) || len(f.events(t)) != count+1 {
		t.Fatalf("retry: %+v %v", retry, err)
	}
	fresh, err := owner.TargetCommissionerLedger(dueLeague)
	if err != nil || fresh.Revision == ledger.Revision || fresh.Revision != receipt.Revision {
		t.Fatalf("visible change token: %+v %v", fresh, err)
	}
}

func TestOpenBidChangesRecipientTokenAndConflictsWithoutAppend(t *testing.T) {
	f := newDueFixture(t)
	owner := viewService(t, f, "owner.two")
	before, err := owner.TargetNativeBids(dueLeague, "franchise.02")
	if err != nil {
		t.Fatal(err)
	}
	f.bid(t)
	after, err := owner.TargetNativeBids(dueLeague, "franchise.02")
	if err != nil || after.Revision == before.Revision || len(after.Players) != 1 {
		t.Fatalf("open bid: %+v %v", after, err)
	}
	cut := f.raw(t, commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0099", Franchise: "franchise.02"}})
	cut.Expected = before.Revision
	count := len(f.events(t))
	if _, err := owner.Execute(t.Context(), cut); err == nil || len(f.events(t)) != count {
		t.Fatal("visible change did not conflict atomically")
	}
	cut.Expected = after.Revision
	if _, err := owner.Execute(t.Context(), cut); err != nil {
		t.Fatalf("fresh token refused: %v", err)
	}
}

func TestHiddenBidRequestKeyCannotAffectAnotherRecipient(t *testing.T) {
	f := newDueFixture(t)
	f.execute(t, "creator", commissioner.RulesSet, commissioner.RawArgs{Field: "bids.ufa", Value: json.RawMessage(`"blind"`)})
	one, two := viewService(t, f, "owner.one"), viewService(t, f, "owner.two")
	before, err := two.TargetCommissionerLedger(dueLeague)
	if err != nil {
		t.Fatal(err)
	}
	bid := f.raw(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5001", Franchise: "franchise.01", Salary: 500000000, Years: 1}})
	own, err := one.TargetCommissionerLedger(dueLeague)
	if err != nil {
		t.Fatal(err)
	}
	bid.Expected, bid.Request = own.Revision, "same-key"
	hidden, err := one.Execute(t.Context(), bid)
	if err != nil {
		t.Fatal(err)
	}
	cut := f.raw(t, commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0099", Franchise: "franchise.02"}})
	cut.Expected, cut.Request = before.Revision, bid.Request
	visible, err := two.Execute(t.Context(), cut)
	if err != nil || hidden.Entry == visible.Entry {
		t.Fatalf("request-key oracle: %+v %v", visible, err)
	}
	for _, retry := range []struct {
		service *Service
		raw     commissioner.RawCommand
		want    commissioner.Receipt
	}{{one, bid, hidden}, {two, cut, visible}} {
		got, err := retry.service.Execute(t.Context(), retry.raw)
		if err != nil || !reflect.DeepEqual(got, retry.want) {
			t.Fatalf("scoped retry: %+v %v", got, err)
		}
	}
}

func TestRecipientTokensSurviveColdReplayWithoutChangingHistory(t *testing.T) {
	f := newDueFixture(t)
	one, two := viewService(t, f, "owner.one"), viewService(t, f, "owner.two")
	first, err := one.TargetCommissionerLedger(dueLeague)
	if err != nil {
		t.Fatal(err)
	}
	second, err := two.TargetCommissionerLedger(dueLeague)
	if err != nil || first.Revision == second.Revision || first.Revision > (1<<53)-1 || second.Revision < 2 {
		t.Fatalf("recipient scope: %+v %v", second, err)
	}
	before, err := json.Marshal(f.events(t))
	if err != nil {
		t.Fatal(err)
	}
	transcript := viewTranscript(t, two)
	if err := f.pools.Close(); err != nil {
		t.Fatal(err)
	}
	f.open(t)
	if !reflect.DeepEqual(transcript, viewTranscript(t, viewService(t, f, "owner.two"))) {
		t.Fatal("cold replay changed recipient response")
	}
	after, err := json.Marshal(f.events(t))
	if err != nil || string(before) != string(after) {
		t.Fatalf("reading rewrote durable history: %v", err)
	}
}
