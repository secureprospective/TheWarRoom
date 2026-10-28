package commissioner

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func TestOLinePoolsAndCloseFollowPoolSetting(t *testing.T) {
	for _, pools := range []int{1, 2} {
		t.Run(fmt.Sprintf("%d pools", pools), func(t *testing.T) {
			b := newLedger(t, PresetDefault32)
			if pools == 2 {
				b.set(poolsField, `"per_side"`, "now", seasonMoment(2))
			}
			seed, err := NativeStart("native.test", b.records, seasonMoment(3))
			if err != nil {
				t.Fatal(err)
			}
			if len(seed.Pool) != 32*pools {
				t.Fatalf("lots=%d, want %d", len(seed.Pool), 32*pools)
			}
			if err := checkCardStart(b.records, &seed); err != nil {
				t.Fatal(err)
			}
			bad := seed
			bad.Pool = slices.Clone(seed.Pool)
			bad.Pool[1] = bad.Pool[0]
			if err := checkCardStart(b.records, &bad); err == nil {
				t.Fatal("duplicate side lot admitted")
			}
			b.add(RawCommand{League: "native.test", Command: NativeSeed, Args: RawArgs{Native: &seed}}, seasonMoment(3))
			if !StartingCardLot(b.records, seed.Pool[0].Player) {
				t.Fatal("side lots lost starting auction rules")
			}
			raw := RawCommand{League: "native.test", Command: OLineAuctionClose}
			if err := PrepareOLineClose(&raw, b.records, seasonMoment(4)); err != nil {
				t.Fatal(err)
			}
			if len(raw.Args.OLineOutcome.Assigned) != 32 || len(raw.Args.OLineOutcome.Pending) != 0 {
				t.Fatalf("close=%+v", raw.Args.OLineOutcome)
			}
			seen := map[string]bool{}
			for _, p := range raw.Args.OLineOutcome.Assigned {
				key := seed.SideOf(p.Franchise) + ":" + p.ID
				if seen[key] || p.ExpirationYear != seed.Season || p.Salary <= 0 {
					t.Fatalf("bad copy/contract: %+v", p)
				}
				seen[key] = true
			}
			b.add(raw, seasonMoment(4))
			a, err := NativeAssetsReading("native.test", b.records)
			if err != nil || len(a.Snapshot.Players) != 32 {
				t.Fatalf("replay: %+v %v", a, err)
			}
			if err := PrepareOLineClose(&raw, b.records, seasonMoment(5)); err == nil {
				t.Fatal("close assigned twice")
			}
		})
	}
}

func TestOLineCapReservationsAndRetryArePerSide(t *testing.T) {
	for _, pools := range []int{1, 2} {
		t.Run(fmt.Sprintf("%d pools", pools), func(t *testing.T) {
			b := newLedger(t, PresetDefault32)
			if pools == 2 {
				b.set(poolsField, `"per_side"`, "now", seasonMoment(2))
			}
			seed, err := NativeStart("native.test", b.records, seasonMoment(3))
			if err != nil {
				t.Fatal(err)
			}
			rules, err := RulesNow(b.records, seasonMoment(3))
			if err != nil {
				t.Fatal(err)
			}
			seed.Players = []NativePlayer{{ID: "1000", Franchise: "franchise.01", Status: domain.RosterActive,
				Salary: domain.Money(rules.Cap.Amount.Value), ExpirationYear: 2026}}
			b.add(RawCommand{League: "native.test", Command: NativeSeed, Args: RawArgs{Native: &seed}}, seasonMoment(3))
			raw := RawCommand{League: "native.test", Command: OLineAuctionClose}
			if err := PrepareOLineClose(&raw, b.records, seasonMoment(4)); err != nil {
				t.Fatal(err)
			}
			out := raw.Args.OLineOutcome
			if len(out.Assigned) != 31 || len(out.Pending) != 1 || out.Pending[0].Franchise != "franchise.01" {
				t.Fatalf("cap blocked the wrong side: %+v", out)
			}
			if pools == 2 && !slices.ContainsFunc(out.Assigned, func(p NativePlayer) bool {
				return seed.SideOf(p.Franchise) == "NFC" && p.ID == out.Pending[0].ID
			}) {
				t.Fatal("the other side could not use its own copy of the reserved card")
			}
			if jobs := olineJobs(MarketBook{OLine: out}, &seed); len(jobs) != 1 || !strings.Contains(jobs[0].Words, "cut a player") {
				t.Fatalf("no scoped cut prompt: %+v", jobs)
			}
			b.add(raw, seasonMoment(4))
			if due, err := OLineAssignmentsDue("native.test", b.records, seasonMoment(5)); err != nil || due {
				t.Fatalf("full cap retried assignment: %t %v", due, err)
			}
			// A recorded roster correction removes the cap-filling player, as a completed cut would.
			b.add(RawCommand{League: "native.test", Command: CorrectRoster, Args: RawArgs{Correction: &RawCorrection{
				Franchise: "franchise.01", Player: "1000", Action: "remove"}}}, seasonMoment(5))
			if due, err := OLineAssignmentsDue("native.test", b.records, seasonMoment(6)); err != nil || !due {
				t.Fatalf("room did not unblock reserved copy: %t %v", due, err)
			}
			if err := PrepareOLineClose(&raw, b.records, seasonMoment(6)); err != nil {
				t.Fatal(err)
			}
			if len(raw.Args.OLineOutcome.Assigned) != 1 || raw.Args.OLineOutcome.Assigned[0].ID != out.Pending[0].ID ||
				len(raw.Args.OLineOutcome.Pending) != 0 {
				t.Fatal("retry lost its reserved copy", raw.Args.OLineOutcome)
			}
			b.add(raw, seasonMoment(6))
			a, err := NativeAssetsReading("native.test", b.records)
			if err != nil || len(a.Snapshot.Players) != 32 {
				t.Fatalf("retry replay: %v", err)
			}
		})
	}
}

func TestOldUnscopedOLineSeedReadsPerSideWithoutRewritingIt(t *testing.T) {
	b := newLedger(t, PresetDefault32)
	seed, err := NativeStart("native.test", b.records, seasonMoment(2))
	if err != nil {
		t.Fatal(err)
	}
	b.add(RawCommand{League: "native.test", Command: NativeSeed, Args: RawArgs{Native: &seed}}, seasonMoment(2))
	b.set(poolsField, `"per_side"`, "now", seasonMoment(3))
	before, err := json.Marshal(b.records)
	if err != nil {
		t.Fatal(err)
	}
	a, err := NativeAssetsReading("native.test", b.records)
	if err != nil || len(a.Snapshot.Pool) != 64 {
		t.Fatalf("old seed pool: %+v %v", a, err)
	}
	a.Snapshot.Pool[0].Name = "changed reading"
	after, err := json.Marshal(b.records)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("reading rewrote old seed", err)
	}
}
