package commissioner_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

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

func (f *marketFixture) act(t *testing.T, command string, args commissioner.RawArgs, principal string) string {
	t.Helper()
	visibility := "league"
	if commissioner.OwnerBidAction(command) != "" || command == commissioner.WaiverClaim {
		visibility = "private"
	}
	input := commissioner.RawCommand{League: rolloverLeague, Command: command, Args: args, Request: fmt.Sprintf("request.%d", len(f.b.records)), Reason: "Recorded change", Effective: "now", Visibility: visibility}
	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := commissioner.Decode(string(encoded))
	if err != nil {
		t.Fatalf("%s decode: %v", command, err)
	}
	switch {
	case commissioner.BidCommand(command):
		err = market.PrepareBidAppend(&raw, f.b.records, f.now, f.facts, time.UTC, market.BidEnvironment{Season: nativecheck.BidSeason{Week: 5, WeekKnown: true}})
	case commissioner.WaiverCommand(command):
		err = market.PrepareWaiverAppend(&raw, f.b.records, f.now, f.facts, time.UTC)
	case commissioner.TradeCommand(command):
		err = market.PrepareTradeAppend(&raw, principal, f.b.records, f.now, f.facts, time.UTC)
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

func TestMarketClosedBooksRoundTrip(t *testing.T) {
	f := newMarketFixture(t)
	f.setup = true
	f.act(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5001", Franchise: "franchise.01", Salary: 500000000, Years: 2}}, "owner.one")
	window := f.act(t, commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0003", Franchise: "franchise.02"}}, "owner.two")
	f.act(t, commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Window: window, Franchise: "franchise.01"}}, "owner.one")
	f.act(t, commissioner.TradeOffer, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{Franchise: "franchise.02", Parties: []string{"franchise.02", "franchise.01"}, Assets: []commissioner.TradeAsset{{From: "franchise.02", To: "franchise.01", Season: 2028, Round: 1, Original: "franchise.01"}}}}, "owner.two")
	f.b.rollover(2027, rolloverMoment())
	raw, err := commissioner.CloseBooks(f.b.records)
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
	var decoded commissioner.RawClosedBooks
	if err := json.Unmarshal(before, &decoded); err != nil {
		t.Fatal(err)
	}
	decoded.Native, decoded.Waivers, err = commissioner.CloneClosedNativeForTest(decoded)
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
