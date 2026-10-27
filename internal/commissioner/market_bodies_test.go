package commissioner

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestMain(m *testing.M) {
	update := flag.Bool("update", false, "rewrite market body goldens")
	flag.Parse()
	if *update {
		if err := os.Setenv("TWR_WRITE_MARKET_BODIES", "1"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	os.Exit(m.Run())
}

type marketFixture struct {
	b     *ledgerBuilder
	facts nativecheck.Facts
	now   time.Time
	setup bool
}

func newMarketFixture(t *testing.T) *marketFixture {
	t.Helper()
	b := rolloverLedger(t)
	now := seasonMoment(13)
	for i := 1; i <= 32; i++ {
		b.records[1].Command.Args.Native.StandingsOrder = append(b.records[1].Command.Args.Native.StandingsOrder, fmt.Sprintf("franchise.%02d", i))
	}
	for _, rule := range [][2]string{
		{"roster.limits", `{"min":0,"max":100}`},
		{"cap.amount", `100000000000`},
		{"calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["cut","waiver_claim","ufa_bid","trade","ir"]}}`},
		{"calendar.tradeDeadline", `"2026-12-01T00:00:00Z"`},
		{"trades.dotSeats", `4`},
	} {
		b.set(rule[0], rule[1], "now", seasonMoment(5))
	}
	return &marketFixture{b: b, now: now, facts: nativecheck.Facts{"5001": {Name: "Free Agent"}}}
}

func (f *marketFixture) act(t *testing.T, command string, args RawArgs, principal string) string {
	t.Helper()
	visibility := "league"
	if OwnerBidAction(command) != "" || command == WaiverClaim {
		visibility = "private"
	}
	input := RawCommand{League: rolloverLeague, Command: command, Args: args, Request: fmt.Sprintf("request.%d", len(f.b.records)), Reason: "Recorded change", Effective: "now", Visibility: visibility}
	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Decode(string(encoded))
	if err != nil {
		t.Fatalf("%s decode: %v", command, err)
	}
	switch {
	case BidCommand(command):
		err = PrepareBidAppend(&raw, f.b.records, f.now, f.facts, time.UTC, BidEnvironment{Season: nativecheck.BidSeason{Week: 5, WeekKnown: true}})
	case WaiverCommand(command):
		err = PrepareWaiverAppend(&raw, f.b.records, f.now, f.facts, time.UTC)
	case TradeCommand(command):
		err = PrepareTradeAppend(&raw, principal, f.b.records, f.now, f.facts, time.UTC)
	}
	if err != nil {
		t.Fatalf("%s prepare: %v", command, err)
	}
	if f.setup {
		entry := f.b.add(raw, f.now)
		f.b.records[len(f.b.records)-1].Principal = principal
		return entry
	}
	got, err := json.MarshalIndent(raw.Args, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := "testdata/market-bodies/" + command + ".json"
	if os.Getenv("TWR_WRITE_MARKET_BODIES") == "1" {
		if err := os.WriteFile(path, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s: appended body differs from golden", command)
	}
	entry := f.b.add(raw, f.now)
	f.b.records[len(f.b.records)-1].Principal = principal
	return entry
}

func TestMarketBodies(t *testing.T) {
	f := newMarketFixture(t)
	order := make([]string, 32)
	for i := range order {
		order[i] = fmt.Sprintf("franchise.%02d", 32-i)
	}
	f.act(t, WaiverOrderSet, RawArgs{Waiver: &RawWaiverArgs{Order: order}}, "creator")
	bid := f.act(t, BidPlace, RawArgs{Bid: &RawBidArgs{Player: "5001", Franchise: "franchise.01", Salary: 500000000, Years: 2}}, "owner.one")
	raised := f.act(t, BidRaise, RawArgs{Bid: &RawBidArgs{Bid: bid, Franchise: "franchise.01", Salary: 600000000, Years: 2}}, "owner.one")
	beforeWithdraw := len(f.b.records)
	f.act(t, BidWithdraw, RawArgs{Bid: &RawBidArgs{Bid: raised, Franchise: "franchise.01"}}, "owner.one")
	f.b.records = f.b.records[:beforeWithdraw]
	f.setup = true
	f.act(t, BidPlace, RawArgs{Bid: &RawBidArgs{Player: "5001", Franchise: "franchise.02", Salary: 700000000, Years: 2}}, "owner.two")
	f.setup = false
	f.act(t, BidsProcess, RawArgs{Bid: &RawBidArgs{Player: "5001"}}, "creator")
	f.act(t, BidsOverride, RawArgs{Bid: &RawBidArgs{Bid: raised, Franchise: "franchise.01"}}, "creator")
	f.act(t, RosterReserve, RawArgs{Waiver: &RawWaiverArgs{Player: "0002", Franchise: "franchise.02", Bucket: "active"}}, "owner.two")
	window := f.act(t, RosterCut, RawArgs{Waiver: &RawWaiverArgs{Player: "0003", Franchise: "franchise.02"}}, "owner.two")
	claim := f.act(t, WaiverClaim, RawArgs{Waiver: &RawWaiverArgs{Window: window, Franchise: "franchise.01"}}, "owner.one")
	f.setup = true
	f.act(t, WaiverClaim, RawArgs{Waiver: &RawWaiverArgs{Window: window, Franchise: "franchise.03"}}, "owner.three")
	f.setup = false
	f.now = f.now.Add(7 * 24 * time.Hour)
	f.act(t, WaiverProcess, RawArgs{Waiver: &RawWaiverArgs{Window: window}}, "creator")
	f.act(t, WaiverOverride, RawArgs{Waiver: &RawWaiverArgs{Claim: claim, Franchise: "franchise.01"}}, "creator")
	f.act(t, WaiverOrderAuto, RawArgs{Waiver: &RawWaiverArgs{}}, "creator")
	for _, who := range []string{"dot.a", "dot.b", "dot.c", "dot.d"} {
		f.b.add(RawCommand{League: rolloverLeague, Command: roleGrant, Args: RawArgs{People: &RawPeople{Principal: who, Role: governance.DOT}}, Effective: "now", Visibility: "private"}, seasonMoment(5))
	}
	offer := f.act(t, TradeOffer, RawArgs{Trade: &RawTradeArgs{Franchise: "franchise.02", Parties: []string{"franchise.02", "franchise.01"}, Assets: []TradeAsset{{From: "franchise.02", To: "franchise.01", Season: 2028, Round: 1, Original: "franchise.01"}}}}, "owner.two")
	f.act(t, TradeAccept, RawArgs{Trade: &RawTradeArgs{ID: offer, Franchise: "franchise.01"}}, "owner.one")
	f.act(t, TradeReview, RawArgs{Trade: &RawTradeArgs{ID: offer, Decision: "veto"}}, "dot.a")
	f.setup = true
	f.act(t, TradeReview, RawArgs{Trade: &RawTradeArgs{ID: offer, Decision: "veto"}}, "dot.b")
	f.act(t, TradeReview, RawArgs{Trade: &RawTradeArgs{ID: offer, Decision: "veto"}}, "dot.c")
	f.setup = false
	f.act(t, TradeOverride, RawArgs{Trade: &RawTradeArgs{ID: offer, Decision: "approve"}}, "creator")
}

func TestMarketClosedBooksRoundTrip(t *testing.T) {
	f := newMarketFixture(t)
	f.setup = true
	f.act(t, BidPlace, RawArgs{Bid: &RawBidArgs{Player: "5001", Franchise: "franchise.01", Salary: 500000000, Years: 2}}, "owner.one")
	window := f.act(t, RosterCut, RawArgs{Waiver: &RawWaiverArgs{Player: "0003", Franchise: "franchise.02"}}, "owner.two")
	f.act(t, WaiverClaim, RawArgs{Waiver: &RawWaiverArgs{Window: window, Franchise: "franchise.01"}}, "owner.one")
	f.act(t, TradeOffer, RawArgs{Trade: &RawTradeArgs{Franchise: "franchise.02", Parties: []string{"franchise.02", "franchise.01"}, Assets: []TradeAsset{{From: "franchise.02", To: "franchise.01", Season: 2028, Round: 1, Original: "franchise.01"}}}}, "owner.two")
	f.b.rollover(2027, rolloverMoment())
	raw, err := CloseBooks(f.b.records)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw.Waivers.Bids.Auctions) != 1 || raw.Waivers.Bids.Auctions[0].Decided ||
		len(raw.Waivers.Bids.Auctions[0].Bids) != 1 || len(raw.Waivers.Windows) != 1 ||
		len(raw.Waivers.Windows[0].Claims) != 1 || len(raw.Waivers.Trades) != 1 || raw.Waivers.Trades[0].Status != "offered" {
		t.Fatal("market state missing from closed books")
	}
	before, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	var decoded RawClosedBooks
	if err := json.Unmarshal(before, &decoded); err != nil {
		t.Fatal(err)
	}
	decoded.Native, decoded.Waivers, err = cloneClosedNative(decoded)
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("closed books changed after round-trip and clone")
	}
}
