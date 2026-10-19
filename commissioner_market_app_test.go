package main

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func marketOf(t *testing.T, f *waiverFixture, command, who string) commissioner.Market {
	t.Helper()
	f.app.commissionerSession = testCommissionerSession{who}
	reading, err := f.app.TargetCommissionerMarket(legalityLeague, command)
	if err != nil {
		t.Fatal(err)
	}
	return reading
}

func marketPreview(t *testing.T, f *waiverFixture, command string, args commissioner.RawArgs, who string) {
	t.Helper()
	f.app.commissionerSession = testCommissionerSession{who}
	body, err := json.Marshal(f.raw(t, command, args, "league"))
	if err != nil {
		t.Fatal(err)
	}
	preview, err := f.app.TargetCommissionerPreview(string(body))
	if err != nil || len(preview.Blockers) != 0 || len(preview.Fields) == 0 {
		t.Fatalf("market target rejected by %s: %+v %v", command, preview, err)
	}
}

func TestCommissionerMarketPermissionAndSession(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	f.app.commissionerSession = testCommissionerSession{"owner.one"}
	for _, command := range []string{commissioner.WaiverProcess, commissioner.WaiverOverride,
		commissioner.BidsProcess, commissioner.BidsOverride, commissioner.TradeReview,
		commissioner.TradeAppealRule, commissioner.TradeOverride} {
		_, err := f.app.TargetCommissionerMarket(legalityLeague, command)
		records, readErr := f.app.leagueRecords(t.Context(), legalityLeague)
		if readErr != nil {
			t.Fatal(readErr)
		}
		_, want := commissioner.AuthorizeRecords(commissioner.RawCommand{League: legalityLeague, Command: command}, "owner.one", records)
		if want == nil || err == nil || err.Error() != "commissioner market: "+want.Error() {
			t.Fatalf("permission words changed: %s: %v / %v", command, err, want)
		}
	}
	actor, err := f.app.TargetCommissionerActor()
	if err != nil || actor != "owner.one" {
		t.Fatalf("actor: %q %v", actor, err)
	}
	f.app.commissionerSession = testCommissionerSession{"creator"}
	people, err := f.app.TargetCommissionerPeople(legalityLeague)
	if err != nil || people.YourFranchise != "" {
		t.Fatal("commissioner unexpectedly owns a franchise", err)
	}
	for _, command := range []string{commissioner.WaiverProcess, commissioner.BidsProcess, commissioner.TradeOverride} {
		marketOf(t, f, command, "creator")
	}
	f.command(t, "commish.role.grant", commissioner.RawArgs{People: &commissioner.RawPeople{
		Principal: "cc", Role: governance.CoCommissioner}}, "private")
	marketOf(t, f, commissioner.WaiverOverride, "cc")
	f.app.commissionerSession = testCommissionerSession{"cc"}
	if _, err := f.app.TargetCommissionerMarket(legalityLeague, commissioner.TradeOverride); err == nil {
		t.Fatal("co-commissioner got commissioner-only override targets")
	}
}

func TestCommissionerMarketWaiversRoundTrip(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	cut := f.cut(t, "1000")
	claim := f.claim(t, cut.Entry, "owner.one", "franchise.01")
	other := f.claim(t, cut.Entry, "owner.three", "franchise.03")
	reading := marketOf(t, f, commissioner.WaiverProcess, "creator")
	if len(reading.Windows) != 1 || reading.Windows[0].ID != cut.Entry || len(reading.Claims) != 0 ||
		!strings.Contains(reading.Windows[0].Words, "Clock running") || !strings.Contains(reading.Windows[0].Words, "cut by") {
		t.Fatalf("running waiver/private claim: %+v", reading)
	}
	f.now = f.now.Add(25 * time.Hour)
	reading = marketOf(t, f, commissioner.WaiverProcess, "creator")
	if !strings.Contains(reading.Windows[0].Words, "Ready to process") {
		t.Fatal(reading)
	}
	marketPreview(t, f, commissioner.WaiverProcess, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Window: reading.Windows[0].ID}}, "creator")
	f.process(t, cut.Entry)
	reading = marketOf(t, f, commissioner.WaiverOverride, "creator")
	if len(reading.Windows) != 0 || len(reading.Claims) != 2 ||
		!slices.ContainsFunc(reading.Claims, func(c commissioner.MarketChoice) bool { return c.ID == claim.Entry }) ||
		!slices.ContainsFunc(reading.Claims, func(c commissioner.MarketChoice) bool { return c.ID == other.Entry }) {
		t.Fatalf("processed claims: %+v", reading)
	}
	for _, c := range reading.Claims {
		if !strings.Contains(c.Words, "Awarded to") {
			t.Fatal("award team missing", c)
		}
		marketPreview(t, f, commissioner.WaiverOverride, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
			Claim: c.ID, Franchise: map[string]string{claim.Entry: "franchise.01", other.Entry: "franchise.03"}[c.ID]}}, "creator")
	}
}

func TestCommissionerMarketBidsRoundTripAndBlindPrivacy(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.set(t, "bids.ufa", `"blind"`, "now")
	first := f.place(t, "owner.one", "franchise.01", "5001", million, 2)
	second := f.place(t, "owner.two", "franchise.02", "5001", million, 2)
	reading := marketOf(t, f.waiverFixture, commissioner.BidsProcess, "creator")
	body, err := json.Marshal(reading)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{first.Entry, second.Entry, "owner.one", "owner.two", "franchise.01", "franchise.02", "$", "salary", "points"} {
		if strings.Contains(string(body), secret) {
			t.Fatalf("blind market leaked %q: %s", secret, body)
		}
	}
	if len(reading.Bids) != 0 || len(reading.Players) != 1 || !strings.Contains(reading.Players[0].Words, "Clock running") {
		t.Fatalf("undecided market: %+v", reading)
	}
	f.advance(25 * time.Hour)
	reading = marketOf(t, f.waiverFixture, commissioner.BidsProcess, "creator")
	if !strings.Contains(reading.Players[0].Words, "Tie to settle") {
		t.Fatal(reading)
	}
	for _, p := range reading.Players {
		marketPreview(t, f.waiverFixture, commissioner.BidsProcess, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: p.ID}}, "creator")
	}
	f.decide(t, "5001")
	reading = marketOf(t, f.waiverFixture, commissioner.BidsOverride, "creator")
	if len(reading.Bids) != 2 || len(reading.Players) != 0 {
		t.Fatalf("decided market: %+v", reading)
	}
	for _, b := range reading.Bids {
		if !strings.Contains(b.Words, "year") {
			t.Fatal("contract words missing", b)
		}
		marketPreview(t, f.waiverFixture, commissioner.BidsOverride, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{
			Bid: b.ID, Franchise: "franchise.01"}}, "creator")
	}
}

func TestCommissionerMarketTradesRoundTrip(t *testing.T) {
	t.Parallel()
	f := tradeApp(t)
	offer := f.offer(t)
	if got := marketOf(t, f.waiverFixture, commissioner.TradeReview, "dot.a"); len(got.Trades) != 0 {
		t.Fatal("unaccepted offer was actionable", got)
	}
	f.accept(t, offer)
	if got := marketOf(t, f.waiverFixture, commissioner.TradeReview, "owner.two"); len(got.Trades) != 0 {
		t.Fatal("a party reviewer received its own trade", got)
	}
	reading := marketOf(t, f.waiverFixture, commissioner.TradeReview, "dot.a")
	if len(reading.Trades) != 1 || !strings.Contains(reading.Trades[0].Words, "review") {
		t.Fatal(reading)
	}
	marketPreview(t, f.waiverFixture, commissioner.TradeReview, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{
		ID: reading.Trades[0].ID, Decision: "approve"}}, "dot.a")
	f.veto(t, offer)
	reading = marketOf(t, f.waiverFixture, commissioner.TradeOverride, "creator")
	marketPreview(t, f.waiverFixture, commissioner.TradeOverride, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{
		ID: reading.Trades[0].ID, Decision: "approve"}}, "creator")
	f.mustAction(t, "owner.one", commissioner.TradeAppeal, commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"})
	reading = marketOf(t, f.waiverFixture, commissioner.TradeAppealRule, "creator")
	if len(reading.Trades) != 1 || !strings.Contains(reading.Trades[0].Words, "appeal") {
		t.Fatal(reading)
	}
	marketPreview(t, f.waiverFixture, commissioner.TradeAppealRule, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{
		ID: reading.Trades[0].ID, Decision: "approve"}}, "creator")
	f.mustAction(t, "creator", commissioner.TradeAppealRule, commissioner.RawTradeArgs{ID: offer.Entry, Decision: "approve"})
	reading = marketOf(t, f.waiverFixture, commissioner.TradeOverride, "creator")
	if len(reading.Trades) != 1 || !strings.Contains(reading.Trades[0].Words, "Approved") {
		t.Fatal(reading)
	}
	marketPreview(t, f.waiverFixture, commissioner.TradeOverride, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{
		ID: reading.Trades[0].ID, Decision: "veto"}}, "creator")
}

func TestCommissionerMarketWire(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	cut := f.cut(t, "1000")
	f.claim(t, cut.Entry, "owner.one", "franchise.01")
	f.place(t, "owner.one", "franchise.01", "5001", million, 2)
	f.place(t, "owner.one", "franchise.01", "5002", million, 2)
	f.advance(25 * time.Hour)
	pending := marketOf(t, f.waiverFixture, commissioner.WaiverProcess, "creator")
	waitingBids := marketOf(t, f.waiverFixture, commissioner.BidsProcess, "creator")
	f.process(t, cut.Entry)
	f.decide(t, "5001")
	claims := marketOf(t, f.waiverFixture, commissioner.WaiverOverride, "creator")
	bids := marketOf(t, f.waiverFixture, commissioner.BidsOverride, "creator")
	trade := tradeApp(t)
	offer := trade.offer(t)
	trade.accept(t, offer)
	trades := marketOf(t, trade.waiverFixture, commissioner.TradeReview, "dot.a")
	wire := map[string]commissioner.Market{"pending": pending, "waitingBids": waitingBids, "claims": claims, "bids": bids, "trades": trades}
	body, err := json.MarshalIndent(wire, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := "frontend/src/app/commissioner/market.wire.testdata.json"
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(path)
	if err != nil || string(held) != string(body) {
		t.Fatal("market wire stale", err)
	}
}

func TestCommissionerMarketIgnoresForeignLedgerAndPermission(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	cut := f.cut(t, "1000")
	f.claim(t, cut.Entry, "owner.one", "franchise.01")
	f.place(t, "owner.two", "franchise.02", "5001", million, 2)
	records, err := f.app.leagueRecords(t.Context(), legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	foreignBody := strings.ReplaceAll(strings.ReplaceAll(string(body), legalityLeague, "native.foreign"), "act.", "foreign.")
	foreignBody = strings.ReplaceAll(foreignBody, "creator", "foreign.commissioner")
	var foreign []commissioner.Record
	if err := json.Unmarshal([]byte(foreignBody), &foreign); err != nil {
		t.Fatal(err)
	}
	combined := append(slices.Clone(records), foreign...)
	for _, command := range []string{commissioner.WaiverProcess, commissioner.BidsProcess} {
		want, err := commissioner.MarketReading(legalityLeague, command, "creator", records, f.now, time.UTC, f.app.nativeFacts())
		if err != nil {
			t.Fatal(err)
		}
		got, err := commissioner.MarketReading(legalityLeague, command, "creator", combined, f.now, time.UTC, f.app.nativeFacts())
		if err != nil {
			t.Fatal(err)
		}
		wantBody, marshalErr := json.Marshal(want)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		gotBody, marshalErr := json.Marshal(got)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if string(wantBody) != string(gotBody) {
			t.Fatal("foreign data changed the market", string(gotBody))
		}
		if _, err := commissioner.MarketReading(legalityLeague, command, "foreign.commissioner", combined, f.now, time.UTC, nil); err == nil {
			t.Fatal("foreign commissioner authorized in this league")
		}
	}
}

func TestCommissionerScheduleDraftAndPublishedRoundTrip(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	f.command(t, commissioner.ScheduleGenerate, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 2}}, "league")
	f.app.commissionerSession = testCommissionerSession{"owner.one"}
	if _, err := f.app.TargetCommissionerSchedule(legalityLeague); err == nil {
		t.Fatal("owner read unpublished games")
	}
	f.app.commissionerSession = testCommissionerSession{"creator"}
	draft, err := f.app.TargetCommissionerSchedule(legalityLeague)
	if err != nil || len(draft.Draft) != 2 || len(draft.Published) != 0 {
		t.Fatal(draft, err)
	}
	for _, week := range draft.Draft {
		for _, game := range week.Games {
			game.Home, game.Away = game.Away, game.Home
			marketPreview(t, f, commissioner.ScheduleEdit, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{
				Week: week.Week, Games: []commissioner.ScheduleGame{game}}}, "creator")
		}
	}
	f.command(t, commissioner.SchedulePublish, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}, "league")
	f.command(t, commissioner.ScheduleGenerate, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 3}}, "league")
	held, err := f.app.TargetCommissionerSchedule(legalityLeague)
	if err != nil || len(held.Draft) != 3 || len(held.Published) != 2 {
		t.Fatal(held, err)
	}
	game := held.Published[0].Games[0]
	game.Home, game.Away = game.Away, game.Home
	marketPreview(t, f, commissioner.ScheduleEdit, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{
		Week: 1, Published: true, Games: []commissioner.ScheduleGame{game}}}, "creator")
}

func TestCommissionerMarketExcludesBidAwardsSupersededByANewAuction(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.set(t, "bids.ufa", `"blind"`, "now")
	f.place(t, "owner.one", "franchise.01", "5001", million, 2)
	f.place(t, "owner.two", "franchise.02", "5001", million, 2)
	f.advance(25 * time.Hour)
	f.decide(t, "5001")
	if got := marketOf(t, f.waiverFixture, commissioner.BidsOverride, "creator"); len(got.Bids) != 2 {
		t.Fatal("decided tie lost its corrective targets", got)
	}
	f.place(t, "owner.three", "franchise.03", "5001", 2*million, 2)
	got := marketOf(t, f.waiverFixture, commissioner.BidsOverride, "creator")
	if len(got.Bids) != 0 || len(got.Players) != 1 {
		t.Fatal("an older auction cannot be corrected once a new auction exists", got)
	}
}
