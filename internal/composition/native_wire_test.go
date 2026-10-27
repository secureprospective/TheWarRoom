package composition

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

// nativeLeagueWindows opens every owner market window the native goldens drive: cuts, claims, bids and trades.
const nativeLeagueWindows = `{"native":{"start":{"month":1,"day":1},"opens":["cut","waiver_claim","ufa_bid","trade"]}}`

// nativeMarketApp is the bid league with trades open too, until a deadline after the fixture's clock.
func nativeMarketApp(t *testing.T) *bidFixture {
	t.Helper()
	f := bidApp(t)
	f.set(t, "calendar.windows", nativeLeagueWindows, "now")
	f.set(t, "calendar.tradeDeadline", `"2026-12-01T00:00:00Z"`, "now")
	f.set(t, "trades.dotSeats", `4`, "now")
	trades := &tradeFixture{f.waiverFixture}
	for _, who := range []string{"owner.two", "dot.a", "dot.b", "dot.c"} {
		trades.role(t, who, governance.DOT)
	}
	return f
}

// wireReadings collects binding results by name, failing on the first binding that refuses.
type wireReadings map[string]any

func (r wireReadings) add(t *testing.T, name string, v any, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	r[name] = v
}

func (r wireReadings) check(t *testing.T) {
	t.Helper()
	for name, v := range r {
		checkGolden(t, "native", name, v)
	}
}

// TestNativeMarketWireMatchesGolden pins the owner doors and readings of one native league: a cut, a bid
// and a trade offer through their owner bindings, the waiver, bid and trade desks, every legality check,
// the house rules, a player score, the directory and the assets.
func TestNativeMarketWireMatchesGolden(t *testing.T) {
	t.Parallel()
	f := nativeMarketApp(t)
	readings := wireReadings{}
	cut, err := f.owner(t, "owner.two", commissioner.RosterCut, commissioner.RawWaiverArgs{Player: "1001",
		Franchise: "franchise.02"})
	readings.add(t, "waiver-action", cut, err)
	bid, err := f.ownerBid(t, "owner.one", commissioner.BidPlace, commissioner.RawBidArgs{Player: "5004",
		Franchise: "franchise.01", Salary: moneyOf(5 * million), Years: 2})
	readings.add(t, "bid-action", bid, err)
	f.app.core.CommissionerSession = testCommissionerSession{"owner.two"}
	offer, err := newCommissionerAPI(f.app).TargetNativeTradeAction(f.raw(t, commissioner.TradeOffer,
		commissioner.RawArgs{Trade: new(tradePlayerOffer())}, "league"))
	readings.add(t, "trade-action", offer, err)
	f.app.core.CommissionerSession = testCommissionerSession{"owner.one"}
	waivers, err := newCommissionerAPI(f.app).TargetNativeWaivers(legalityLeague, "franchise.01", f.now)
	readings.add(t, "waivers", waivers, err)
	bids, err := newCommissionerAPI(f.app).TargetNativeBids(legalityLeague, "franchise.01")
	readings.add(t, "bids", bids, err)
	trades, err := newCommissionerAPI(f.app).TargetNativeTrades(legalityLeague, "franchise.01")
	readings.add(t, "trades", trades, err)
	capCheck, err := newCommissionerAPI(f.app).TargetNativeCapCheck(legalityLeague, "America/Los_Angeles")
	readings.add(t, "cap-check", capCheck, err)
	roster, err := newCommissionerAPI(f.app).TargetNativeRosterCheck(legalityLeague)
	readings.add(t, "roster-check", roster, err)
	reserve, err := newCommissionerAPI(f.app).TargetNativeReserveCheck(legalityLeague)
	readings.add(t, "reserve-check", reserve, err)
	lineup, err := newCommissionerAPI(f.app).TargetNativeLineupCheck(commissioner.RawNativeLineup{League: legalityLeague,
		Franchise: "franchise.02", Submitted: true, Starters: firstStarters(1)})
	readings.add(t, "lineup-check", lineup, err)
	houses, err := newCommissionerAPI(f.app).TargetNativeHouseRules(legalityLeague)
	readings.add(t, "house-rules", houses, err)
	score, err := newCommissionerAPI(f.app).TargetNativePlayerScore(legalityLeague, "1000", 2026, 1)
	readings.add(t, "player-score", score, err)
	directory, err := newCommissionerAPI(f.app).TargetNativeDirectory(legalityLeague)
	readings.add(t, "directory", directory, err)
	assets, err := newCommissionerAPI(f.app).TargetNativeAssets(legalityLeague)
	if err == nil && assets.Snapshot != nil {
		// A dozen players and picks keep the golden reviewable; the full snapshot has its own wire file.
		assets.Snapshot.Players = assets.Snapshot.Players[:min(12, len(assets.Snapshot.Players))]
		assets.Snapshot.Picks = assets.Snapshot.Picks[:min(12, len(assets.Snapshot.Picks))]
		assets.Snapshot.RosterAt = time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)
	}
	readings.add(t, "assets", assets, err)
	readings.check(t)
}

// TestNativeLeagueWireMatchesGolden pins the remaining native bindings, each on the fixture its own tests
// use: results, standings, a score preview, the bracket and the statistics queue; a lineup set and read;
// the franchise schedule and week; a rookie pick and the draft; the starting snapshot; a contract quote.
func TestNativeLeagueWireMatchesGolden(t *testing.T) {
	t.Parallel()
	readings := wireReadings{}
	scores := scoreApp(t)
	scores.app.core.History.SetNativeLineups(nativeResultLineups{})
	raw := commissioner.RawCommand{League: legalityLeague, Command: commissioner.ScheduleGenerate,
		Args: commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 13}}, Expected: 2,
		Request: "generate-results", Effective: "now", Visibility: "league"}
	executeCommissionerTest(t, scores.app, raw)
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 3, "publish-results"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	executeCommissionerTest(t, scores.app, raw)
	results, err := newCommissionerAPI(scores.app).TargetNativeResults(legalityLeague, 2026, 1)
	if err == nil {
		results.Games = results.Games[:min(2, len(results.Games))]
	}
	readings.add(t, "results", results, err)
	standings, err := newCommissionerAPI(scores.app).TargetNativeStandings(legalityLeague, 2026)
	if err == nil {
		standings.Computed = standings.Computed[:min(4, len(standings.Computed))]
	}
	readings.add(t, "standings", standings, err)
	change := commissioner.RawCommand{League: legalityLeague, Command: commissioner.ScorePlayer,
		Args: commissioner.RawArgs{Score: &commissioner.RawScoreChange{Player: "1000", Season: 2026, Week: 1,
			Delta: scoreDelta(10000)}}, Expected: 4, Request: "preview-results", Reason: "Correct this score",
		Effective: "now", Visibility: "league"}
	preview, err := newCommissionerAPI(scores.app).TargetNativeScorePreview(encodedCommand(t, change))
	readings.add(t, "score-preview", preview, err)
	bracket, err := newCommissionerAPI(scores.app).TargetNativeBracket(legalityLeague, 2026)
	readings.add(t, "bracket", bracket, err)
	queue, err := newCommissionerAPI(scores.app).TargetStatQueue(legalityLeague, 2026)
	readings.add(t, "stat-queue", queue, err)

	lineups := lineupAppTest(t)
	lineups.execute(t, commissioner.RawCommand{Command: commissioner.RulesSet,
		Args: commissioner.RawArgs{Field: "calendar.boundary", Value: json.RawMessage(`"start_of_day"`)}})
	set, err := newCommissionerAPI(lineups.app).TargetNativeLineupSet(lineups.setBody(t, "franchise.02", 1, fixtureStarters(0, 21)))
	readings.add(t, "lineup-set", set, err)
	lineup, err := newCommissionerAPI(lineups.app).TargetNativeLineup(legalityLeague, "franchise.02", 1, "")
	readings.add(t, "lineup", lineup, err)

	at := time.Date(2026, 9, 10, 18, 0, 0, 0, time.UTC)
	calendar, _ := calendarTestApp(t, at)
	schedule := seedScheduleTest(t, calendar)
	schedule.Args.Schedule.Weeks, schedule.Args.Schedule.Pattern = 2, ""
	executeCommissionerTest(t, calendar, schedule)
	schedule.Command, schedule.Expected, schedule.Request = commissioner.SchedulePublish, 7, "publish-owner"
	schedule.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	executeCommissionerTest(t, calendar, schedule)
	first := time.Date(2026, 9, 11, 0, 20, 0, 0, time.UTC)
	heldNativeWeek(t, calendar, 1, first, false)
	heldNativeWeek(t, calendar, 2, first.AddDate(0, 0, 7), false)
	season, err := newCommissionerAPI(calendar).TargetNativeFranchiseSchedule(nativeOwnerScope())
	readings.add(t, "franchise-schedule", season, err)
	week, err := newCommissionerAPI(calendar).TargetNativeWeek(RawNativeWeekScope{RawNativeScheduleScope: nativeOwnerScope(), At: at})
	readings.add(t, "week", week, err)

	draft := draftApp(t)
	draft.scheduled(t, 3, 2, 1)
	draft.seated(t, 1, 3)
	draft.now = draftStartAt().Add(time.Hour)
	draft.asOwner(3)
	rev := draft.revision(t)
	pick, err := draft.tryPick("franchise.03", "5001", fmt.Sprintf("pick.%d", rev), rev)
	readings.add(t, "draft-pick", pick, err)
	reading, err := newCommissionerAPI(draft.app).TargetNativeDraft(commissioner.RawNativeDraftScope{League: draftLeague,
		Season: draftSeason, Franchise: "franchise.01", At: draft.now})
	readings.add(t, "draft", reading, err)

	seeded := nativeSeedTestApp(t)
	seed, err := newCommissionerAPI(seeded).TargetNativeSeed(createNativeSeedLeague(t, seeded, commissioner.PresetLegacy))
	readings.add(t, "seed", seed, err)

	quotes := quoteApp(t)
	quotes.app.core.ClockNow = quoteClock
	quote, err := newCommissionerAPI(quotes.app).TargetNativeContractQuote(market.RawContractQuote{League: legalityLeague,
		Player: "0099", Action: "cut", ViewerZone: "UTC"})
	readings.add(t, "contract-quote", quote, err)
	readings.check(t)
}

// TestNativeDenialsMatchGolden pins the exact words each refused caller reads at the native owner doors
// before the native logic moves out of package main: a stranger and an owner acting for another franchise
// on every owner act and owner reading, an owner on the commissioner's starting snapshot and overrides, a
// shut window, a passed trade deadline, an unreadable session, and every malformed request.
func TestNativeDenialsMatchGolden(t *testing.T) {
	t.Parallel()
	denials := map[string]string{}
	f := nativeMarketApp(t)
	bid := f.place(t, "owner.one", "franchise.01", "5004", 5*million, 2)
	f.app.core.CommissionerSession = testCommissionerSession{"owner.two"}
	offer, err := newCommissionerAPI(f.app).TargetNativeTradeAction(f.raw(t, commissioner.TradeOffer,
		commissioner.RawArgs{Trade: new(tradePlayerOffer())}, "league"))
	if err != nil {
		t.Fatal(err)
	}
	ownerActs := func(prefix string) {
		cut := commissioner.RawWaiverArgs{Player: "1000", Franchise: "franchise.02"}
		_, err := newCommissionerAPI(f.app).TargetNativeWaiverAction(f.raw(t, commissioner.RosterCut,
			commissioner.RawArgs{Waiver: &cut}, "league"))
		denials[prefix+"/cut"] = errorWords(err)
		claim := commissioner.RawWaiverArgs{Window: "window", Franchise: "franchise.02"}
		_, err = newCommissionerAPI(f.app).TargetNativeWaiverAction(f.raw(t, commissioner.WaiverClaim,
			commissioner.RawArgs{Waiver: &claim}, "private"))
		denials[prefix+"/claim"] = errorWords(err)
		place := commissioner.RawBidArgs{Player: "5005", Franchise: "franchise.02", Salary: moneyOf(5 * million), Years: 1}
		_, err = newCommissionerAPI(f.app).TargetNativeBidAction(f.raw(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &place}, "private"))
		denials[prefix+"/bid-place"] = errorWords(err)
		raise := commissioner.RawBidArgs{Bid: bid.Entry, Franchise: "franchise.01", Salary: moneyOf(6 * million), Years: 2}
		_, err = newCommissionerAPI(f.app).TargetNativeBidAction(f.raw(t, commissioner.BidRaise, commissioner.RawArgs{Bid: &raise}, "private"))
		denials[prefix+"/bid-raise-franchise.01"] = errorWords(err)
		_, err = newCommissionerAPI(f.app).TargetNativeTradeAction(f.raw(t, commissioner.TradeOffer,
			commissioner.RawArgs{Trade: new(tradePlayerOffer())}, "league"))
		denials[prefix+"/trade-offer"] = errorWords(err)
		accept := commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"}
		_, err = newCommissionerAPI(f.app).TargetNativeTradeAction(f.raw(t, commissioner.TradeAccept,
			commissioner.RawArgs{Trade: &accept}, "league"))
		denials[prefix+"/trade-accept-franchise.01"] = errorWords(err)
		bids, err := newCommissionerAPI(f.app).TargetNativeBids(legalityLeague, "franchise.02")
		denials[prefix+"/bids-franchise.02"] = errorWords(err) + " " + jsonWords(t, len(bids.Players))
		waivers, err := newCommissionerAPI(f.app).TargetNativeWaivers(legalityLeague, "franchise.02", f.now)
		denials[prefix+"/waivers-franchise.02"] = errorWords(err) + " " + jsonWords(t, len(waivers.Claims))
		trades, err := newCommissionerAPI(f.app).TargetNativeTrades(legalityLeague, "franchise.02")
		denials[prefix+"/trades-franchise.02"] = errorWords(err) + " " + jsonWords(t, len(trades.Offers))
	}
	for _, who := range []string{"stranger", "owner.three"} {
		f.app.core.CommissionerSession = testCommissionerSession{who}
		ownerActs(who)
		for name, raw := range map[string]commissioner.RawCommand{
			"bids-override":  f.raw(t, commissioner.BidsOverride, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Bid: bid.Entry, Franchise: "franchise.03"}}, "league"),
			"bids-process":   f.raw(t, commissioner.BidsProcess, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5004"}}, "league"),
			"trade-override": f.raw(t, commissioner.TradeOverride, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: offer.Entry, Decision: "veto"}}, "league"),
		} {
			_, err := newCommissionerAPI(f.app).TargetCommissionerExecute(encodedCommand(t, raw))
			denials[who+"/"+name] = errorWords(err)
		}
	}
	f.app.core.CommissionerSession = failingCommissionerSession{}
	ownerActs("no-session")
	malformed := map[string]func() error{
		"waiver-outcome": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeWaiverAction(f.raw(t, commissioner.RosterCut, commissioner.RawArgs{
				WaiverOutcome: &commissioner.WaiverOutcome{}}, "league"))
			return err
		},
		"waiver-not-an-owner-act": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeWaiverAction(f.raw(t, commissioner.WaiverProcess, commissioner.RawArgs{}, "league"))
			return err
		},
		"waiver-no-details": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeWaiverAction(f.raw(t, commissioner.RosterCut, commissioner.RawArgs{}, "league"))
			return err
		},
		"bid-not-an-owner-act": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeBidAction(f.raw(t, commissioner.BidsOverride, commissioner.RawArgs{}, "league"))
			return err
		},
		"bid-no-details": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeBidAction(f.raw(t, commissioner.BidPlace, commissioner.RawArgs{}, "private"))
			return err
		},
		"trade-not-an-owner-act": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeTradeAction(f.raw(t, commissioner.TradeReview, commissioner.RawArgs{}, "league"))
			return err
		},
		"trade-no-details": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeTradeAction(f.raw(t, commissioner.TradeOffer, commissioner.RawArgs{}, "league"))
			return err
		},
		"waivers-no-time": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeWaivers(legalityLeague, "franchise.01", time.Time{})
			return err
		},
		"cap-check-zone": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeCapCheck(legalityLeague, "Local")
			return err
		},
		"cap-check-league": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeCapCheck("native.", "UTC")
			return err
		},
		"lineup-check-input": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeLineupCheck(commissioner.RawNativeLineup{League: legalityLeague})
			return err
		},
		"player-score-scope": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativePlayerScore(legalityLeague, "", 2026, 0)
			return err
		},
		"contract-quote-input": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeContractQuote(market.RawContractQuote{League: legalityLeague})
			return err
		},
		"lineup-scope": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeLineup(" native.seed", "franchise.01", 1, "")
			return err
		},
		"lineup-week": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeLineup(legalityLeague, "franchise.01", 23, "")
			return err
		},
		"lineup-at": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeLineup(legalityLeague, "franchise.01", 1, "Sunday")
			return err
		},
		"lineup-set-body": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeLineupSet(`{"league":"native.seed"}`)
			return err
		},
		"lineup-set-two-bodies": func() error { _, err := newCommissionerAPI(f.app).TargetNativeLineupSet(`{} {}`); return err },
		"lineup-set-unknown": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeLineupSet(`{"league":"native.seed","starters":[],"request":"x","extra":1}`)
			return err
		},
		"results-season": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeResults(legalityLeague, 1800, 1)
			return err
		},
		"results-week": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeResults(legalityLeague, 2026, 23)
			return err
		},
		"results-league": func() error { _, err := newCommissionerAPI(f.app).TargetNativeResults("legacy", 2026, 1); return err },
		"standings":      func() error { _, err := newCommissionerAPI(f.app).TargetNativeStandings("native.", 2026); return err },
		"bracket":        func() error { _, err := newCommissionerAPI(f.app).TargetNativeBracket(legalityLeague, 0); return err },
		"stat-queue":     func() error { _, err := newCommissionerAPI(f.app).TargetStatQueue("native.seed ", 2026); return err },
		"score-preview":  func() error { _, err := newCommissionerAPI(f.app).TargetNativeScorePreview(`{`); return err },
		"schedule-zone": func() error {
			scope := nativeOwnerScope()
			scope.LocalZone = "Local"
			_, err := newCommissionerAPI(f.app).TargetNativeFranchiseSchedule(scope)
			return err
		},
		"schedule-franchise": func() error {
			scope := nativeOwnerScope()
			scope.Franchise = " franchise.01"
			_, err := newCommissionerAPI(f.app).TargetNativeFranchiseSchedule(scope)
			return err
		},
		"schedule-unknown-zone": func() error {
			scope := nativeOwnerScope()
			scope.LocalZone = "Mars/Olympus"
			_, err := newCommissionerAPI(f.app).TargetNativeFranchiseSchedule(scope)
			return err
		},
		"week-no-time": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeWeek(RawNativeWeekScope{RawNativeScheduleScope: nativeOwnerScope()})
			return err
		},
		"seed-request": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeSeed(RawNativeSeedRequest{League: legalityLeague, Expected: 1, Request: "x"})
			return err
		},
		"draft-pick-input": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeDraftPick(commissioner.DraftPickInput{League: legalityLeague})
			return err
		},
		"draft-scope": func() error {
			_, err := newCommissionerAPI(f.app).TargetNativeDraft(commissioner.RawNativeDraftScope{League: "legacy"})
			return err
		},
	}
	f.app.core.CommissionerSession = testCommissionerSession{"owner.one"}
	for name, read := range malformed {
		denials["malformed/"+name] = errorWords(read())
	}
	// A shut window: the league keeps only cuts open, and the rightful owners ask for the rest.
	f.set(t, "calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["cut"]}}`, "now")
	f.app.core.CommissionerSession = testCommissionerSession{"owner.one"}
	place := commissioner.RawBidArgs{Player: "5005", Franchise: "franchise.01", Salary: moneyOf(5 * million), Years: 1}
	_, err = newCommissionerAPI(f.app).TargetNativeBidAction(f.raw(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &place}, "private"))
	denials["closed-window/bid-place"] = errorWords(err)
	claim := commissioner.RawWaiverArgs{Window: "window", Franchise: "franchise.01"}
	_, err = newCommissionerAPI(f.app).TargetNativeWaiverAction(f.raw(t, commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &claim}, "private"))
	denials["closed-window/claim"] = errorWords(err)
	accept := commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.01"}
	_, err = newCommissionerAPI(f.app).TargetNativeTradeAction(f.raw(t, commissioner.TradeAccept, commissioner.RawArgs{Trade: &accept}, "league"))
	denials["closed-window/trade-accept"] = errorWords(err)
	f.set(t, "calendar.windows", nativeLeagueWindows, "now")
	f.now = time.Date(2026, 12, 2, 12, 0, 0, 0, time.UTC)
	f.app.core.CommissionerSession = testCommissionerSession{"owner.one"}
	_, err = newCommissionerAPI(f.app).TargetNativeTradeAction(f.raw(t, commissioner.TradeAccept, commissioner.RawArgs{Trade: &accept}, "league"))
	denials["past-deadline/trade-accept"] = errorWords(err)

	lineupDenials(t, denials)
	draftDenials(t, denials)
	seed := nativeSeedTestApp(t)
	input := createNativeSeedLeague(t, seed, commissioner.PresetLegacy)
	for _, who := range []string{"stranger", "owner.one"} {
		seed.core.CommissionerSession = testCommissionerSession{who}
		_, err := newCommissionerAPI(seed).TargetNativeSeed(input)
		denials[who+"/seed"] = errorWords(err)
	}
	seed.core.CommissionerSession = failingCommissionerSession{}
	_, err = newCommissionerAPI(seed).TargetNativeSeed(input)
	denials["no-session/seed"] = errorWords(err)
	checkGolden(t, "native", "denials", denials)
}

// lineupDenials refuses lineups: a stranger's, one for another owner's franchise, one with no session, one
// in a shut window and one that moves a player whose game has started.
func lineupDenials(t *testing.T, denials map[string]string) {
	t.Helper()
	f := lineupAppTest(t)
	legal := fixtureStarters(0, 21)
	f.execute(t, commissioner.RawCommand{Command: commissioner.RulesSet,
		Args: commissioner.RawArgs{Field: "calendar.boundary", Value: json.RawMessage(`"start_of_day"`)}})
	for who, franchise := range map[string]string{"stranger": "franchise.02", "creator": "franchise.03"} {
		f.app.core.CommissionerSession = testCommissionerSession{who}
		_, err := newCommissionerAPI(f.app).TargetNativeLineupSet(f.setBody(t, franchise, 1, legal))
		denials[who+"/lineup-set-"+franchise] = errorWords(err)
	}
	f.app.core.CommissionerSession = failingCommissionerSession{}
	_, err := newCommissionerAPI(f.app).TargetNativeLineupSet(f.setBody(t, "franchise.02", 1, legal))
	denials["no-session/lineup-set"] = errorWords(err)
	f.app.core.CommissionerSession = testCommissionerSession{"creator"}
	if _, err := newCommissionerAPI(f.app).TargetNativeLineupSet(f.setBody(t, "franchise.02", 1, legal)); err != nil {
		t.Fatal(err)
	}
	f.at = f.kickoff
	swapped := append([]string{}, legal...)
	swapped[3] = legalityID(1000, 21)
	_, err = newCommissionerAPI(f.app).TargetNativeLineupSet(f.setBody(t, "franchise.02", 1, swapped))
	denials["locked/lineup-set"] = errorWords(err)
	f.execute(t, commissioner.RawCommand{Command: commissioner.WindowRemove,
		Args: commissioner.RawArgs{Window: &commissioner.RawWindowArgs{ID: "lineups"}}})
	_, err = newCommissionerAPI(f.app).TargetNativeLineupSet(f.setBody(t, "franchise.02", 1, legal))
	denials["closed-window/lineup-set"] = errorWords(err)
}

// draftDenials refuses rookie picks: before the draft opens, a stranger's, another owner's for the
// franchise on the clock, a pick out of turn and one with no session.
func draftDenials(t *testing.T, denials map[string]string) {
	t.Helper()
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.seated(t, 1, 3)
	try := func(name, franchise string) {
		rev := f.revision(t)
		_, err := f.tryPick(franchise, "5001", fmt.Sprintf("denied.%s", name), rev)
		denials[name] = errorWords(err)
	}
	f.asOwner(3)
	try("closed-window/draft-pick", "franchise.03")
	f.now = draftStartAt().Add(time.Hour)
	f.actAs("stranger")
	try("stranger/draft-pick-franchise.03", "franchise.03")
	f.asOwner(1)
	try("owner1/draft-pick-franchise.03", "franchise.03")
	try("owner1/draft-pick-out-of-turn", "franchise.01")
	f.app.core.CommissionerSession = failingCommissionerSession{}
	try("no-session/draft-pick", "franchise.03")
}

// TestNativeSimultaneousOwnerActsLandOnce races two of an owner's bids made at the same league version, with
// readers of their desks alongside: the gate, the version check and the ledger write share one store
// transaction, so exactly one bid lands and the other reads as stale.
func TestNativeSimultaneousOwnerActsLandOnce(t *testing.T) {
	t.Parallel()
	f := nativeMarketApp(t)
	first := f.raw(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5004",
		Franchise: "franchise.01", Salary: moneyOf(5 * million), Years: 2}}, "private")
	second := f.raw(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5005",
		Franchise: "franchise.01", Salary: moneyOf(5 * million), Years: 2}}, "private")
	second.Request = "race.second"
	f.app.core.CommissionerSession = testCommissionerSession{"owner.one"}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, raw := range []commissioner.RawCommand{first, second} {
		wg.Go(func() { _, errs[i] = newCommissionerAPI(f.app).TargetNativeBidAction(raw) })
	}
	for range 4 {
		wg.Go(func() {
			_, _ = newCommissionerAPI(f.app).TargetNativeBids(legalityLeague, "franchise.01")
			_, _ = newCommissionerAPI(f.app).TargetNativeWaivers(legalityLeague, "franchise.01", f.now)
		})
	}
	wg.Wait()
	if (errs[0] == nil) == (errs[1] == nil) {
		t.Fatalf("two bids at one version must land exactly once: %v / %v", errs[0], errs[1])
	}
	lost := errs[0]
	if lost == nil {
		lost = errs[1]
	}
	if !strings.Contains(lost.Error(), "League version is ") {
		t.Fatalf("the losing bid must read as stale: %v", lost)
	}
	records, err := f.app.leagueRecords(t.Context(), legalityLeague)
	if err != nil || int64(len(records)) != first.Expected+1 {
		t.Fatalf("ledger holds %d records, want %d: %v", len(records), first.Expected+1, err)
	}
}
