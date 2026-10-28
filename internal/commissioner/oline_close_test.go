package commissioner

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestOLineAuctionCloseFillsEveryEmptyRoster(t *testing.T) {
	b := newLedger(t, PresetDefault32)
	seed, err := NativeStart("native.test", b.records, seasonMoment(2))
	if err != nil {
		t.Fatal(err)
	}
	b.add(RawCommand{League: "native.test", Command: NativeSeed, Args: RawArgs{Native: &seed}}, seasonMoment(2))
	raw := RawCommand{League: "native.test", Command: "oline.auction.close", Effective: "now", Visibility: "league"}
	if err := PrepareOLineClose(&raw, b.records, seasonMoment(3)); err != nil {
		t.Fatal(err)
	}
	b.add(raw, seasonMoment(3))
	assets, err := NativeAssetsReading("native.test", b.records)
	if err != nil {
		t.Fatal(err)
	}
	rules, err := RulesNow(b.records, seasonMoment(3))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range assets.Snapshot.Franchises {
		roster := []nativecheck.RosterPlayer{}
		for _, p := range assets.Snapshot.Players {
			if p.Franchise == f.ID {
				roster = append(roster, nativecheck.RosterPlayer{ID: p.ID, Status: p.Status})
			}
		}
		if err := nativecheck.LineCardsHeld(rules.Roster, roster); err != nil {
			t.Fatalf("auction closed with %s missing its unit: %v", f.ID, err)
		}
	}
}

func TestOLineCloseKeepsAnAuctionWinnerAndUsesStableOrder(t *testing.T) {
	b := newLedger(t, PresetDefault32)
	seed, err := NativeStart("native.test", b.records, seasonMoment(2))
	if err != nil {
		t.Fatal(err)
	}
	// An already-awarded unit is not reassigned; the pool may still list its original lot.
	seed.Players = append(seed.Players, NativePlayer{ID: "990001", Name: "ARI O-Line", Franchise: "franchise.02", Status: domain.RosterActive, Salary: 123_000_000, ExpirationYear: 2026})
	slices.Reverse(seed.Pool)
	slices.Reverse(seed.Franchises)
	b.add(RawCommand{League: "native.test", Command: NativeSeed, Args: RawArgs{Native: &seed}}, seasonMoment(2))
	raw := RawCommand{League: "native.test", Command: OLineAuctionClose}
	if err := PrepareOLineClose(&raw, b.records, seasonMoment(3)); err != nil {
		t.Fatal(err)
	}
	outcome := raw.Args.OLineOutcome
	if len(outcome.Assigned) != 31 || len(outcome.Pending) != 0 {
		t.Fatalf("assignments: %+v", outcome)
	}
	if outcome.Assigned[0].Franchise != "franchise.01" || outcome.Assigned[0].Name != "ATL O-Line" {
		t.Fatalf("unstable order: %+v", outcome.Assigned[0])
	}
	b.add(raw, seasonMoment(3))
	assets, err := NativeAssetsReading("native.test", b.records)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets.Snapshot.Players) != 32 || assets.Snapshot.Players[0].Salary != 123_000_000 {
		t.Fatal("the winning contract changed")
	}
	label, err := entryLabel(b.records, len(b.records)-1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(label, "Assigned the ATL O-Line to Franchise 1 at the minimum contract") {
		t.Fatalf("ledger words: %s", label)
	}
	if err := CheckNativeAppend(RawCommand{Command: Revert, Args: RawArgs{Entry: b.records[len(b.records)-1].Receipt.Entry}}, b.records); err == nil {
		t.Fatal("close could be reversed in isolation")
	}
}

func TestOLineCloseRefusesUnsettledBidsWithoutChangingRecords(t *testing.T) {
	b := newLedger(t, PresetDefault32)
	seed, err := NativeStart("native.test", b.records, seasonMoment(2))
	if err != nil {
		t.Fatal(err)
	}
	b.add(RawCommand{League: "native.test", Command: NativeSeed, Args: RawArgs{Native: &seed}}, seasonMoment(2))
	b.add(RawCommand{League: "native.test", Command: BidPlace, Args: RawArgs{Bid: &RawBidArgs{Player: "990001", Franchise: "franchise.01", Salary: 100_000_000, Years: 1}, BidOutcome: &BidOutcome{Player: "990001", Season: 2026}}}, seasonMoment(3))
	before := slices.Clone(b.records)
	raw := RawCommand{League: "native.test", Command: OLineAuctionClose}
	if err := PrepareOLineClose(&raw, b.records, seasonMoment(4)); err == nil || !strings.Contains(err.Error(), "settle any ties") {
		t.Fatalf("unsettled auction: %v", err)
	}
	if raw.Args.OLineOutcome != nil || !reflect.DeepEqual(b.records, before) {
		t.Fatal("refused close changed the ledger")
	}
}

func TestOLineCloseCannotSupplyAssignments(t *testing.T) {
	raw := RawCommand{League: "native.test", Command: OLineAuctionClose, Request: "close", Effective: "now", Visibility: "league", Args: RawArgs{OLineOutcome: &OLineOutcome{}}}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(string(body)); err == nil {
		t.Fatal("caller supplied assignments")
	}
	for _, args := range []RawArgs{{Season: 2026}, {Name: "foreign"}, {Bid: &RawBidArgs{Player: "990001"}}} {
		if err := args.Validate(OLineAuctionClose); err == nil {
			t.Fatalf("foreign close args: %+v", args)
		}
	}
	raw.Args = RawArgs{}
	for _, visibility := range []string{"private", ""} {
		raw.Visibility = visibility
		if err := raw.Validate(); err == nil {
			t.Fatal("private close accepted")
		}
	}
}
