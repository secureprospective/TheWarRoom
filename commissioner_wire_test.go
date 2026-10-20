package main

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

type failingCommissionerSession struct{}

func (failingCommissionerSession) Principal() (string, error) { return "", errors.New("no session") }

func encodedCommand(t *testing.T, raw commissioner.RawCommand) string {
	t.Helper()
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func errorWords(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}

// TestCommissionerWireMatchesGolden pins every Commissioner binding's result for one seeded league.
func TestCommissionerWireMatchesGolden(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	f.cut(t, "1000")
	rules := f.command(t, commissioner.RulesSet, commissioner.RawArgs{Field: "roster.limits",
		Value: json.RawMessage(`{"min":0,"max":90}`)}, "league")
	f.app.commissionerSession = testCommissionerSession{"creator"}
	readings := map[string]any{"execute": rules}
	add := func(name string, v any, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		readings[name] = v
	}
	ledger, err := f.app.TargetCommissionerLedger(legalityLeague)
	add("ledger", ledger, err)
	filtered, err := f.app.TargetCommissionerLedgerFiltered(legalityLeague, `{"kind":"rules"}`)
	add("ledger-filtered", filtered, err)
	people, err := f.app.TargetCommissionerPeople(legalityLeague)
	add("people", people, err)
	leagues, err := f.app.TargetCommissionerLeagues()
	add("leagues", leagues, err)
	ruleBook, err := f.app.TargetCommissionerRules(legalityLeague)
	add("rules", ruleBook, err)
	jobs, err := f.app.TargetCommissionerJobs(legalityLeague)
	add("jobs", jobs, err)
	readings["controls"] = f.app.TargetCommissionerControls()
	notices, err := f.app.TargetCommissionerNotices(legalityLeague)
	add("notices", notices, err)
	digests, err := f.app.TargetCommissionerDigests(legalityLeague)
	add("digests", digests, err)
	corrections, err := f.app.TargetCorrections(legalityLeague)
	add("corrections", corrections, err)
	market, err := f.app.TargetCommissionerMarket(legalityLeague, commissioner.WaiverProcess)
	add("market", market, err)
	actor, err := f.app.TargetCommissionerActor()
	add("actor", actor, err)
	managed, err := f.app.TargetCommissionerSchedule(legalityLeague)
	add("managed-schedule", managed, err)
	preview, err := f.app.TargetCommissionerPreview(encodedCommand(t, f.raw(t, commissioner.RulesSet,
		commissioner.RawArgs{Field: "roster.limits", Value: json.RawMessage(`{"min":0,"max":80}`)}, "league")))
	add("preview", preview, err)
	calendar, err := f.app.TargetLeagueCalendar(RawLeagueCalendarScope{League: legalityLeague, Season: 2026,
		LocalZone: "America/Los_Angeles", Franchise: "franchise.01", Action: "waiver_claim"})
	add("calendar", calendar, err)
	schedule, err := f.app.TargetLeagueSchedule(RawLeagueScheduleScope{League: legalityLeague})
	add("schedule", schedule, err)
	f.app.commissionerSession = testCommissionerSession{"owner.one"}
	ownerLedger, err := f.app.TargetCommissionerLedger(legalityLeague)
	add("ledger-owner", ownerLedger, err)
	ownerPeople, err := f.app.TargetCommissionerPeople(legalityLeague)
	add("people-owner", ownerPeople, err)
	for name, v := range readings {
		checkGolden(t, "commissioner", name, v)
	}
}

// TestCommissionerDenialsMatchGolden pins the exact words each refused caller reads: a stranger, an
// owner on commissioner acts, a co-commissioner on commissioner-only acts, a wrong franchise, a
// closed window, a stale version, an unreadable session and every act routed to its own gated path.
func TestCommissionerDenialsMatchGolden(t *testing.T) {
	t.Parallel()
	f := waiverApp(t)
	f.command(t, "commish.role.grant", commissioner.RawArgs{People: &commissioner.RawPeople{
		Principal: "cc", Role: governance.CoCommissioner}}, "private")
	denials := map[string]string{}
	rulesSet := f.raw(t, commissioner.RulesSet, commissioner.RawArgs{Field: "roster.limits",
		Value: json.RawMessage(`{"min":0,"max":90}`)}, "league")
	seat := f.raw(t, "commish.seat.replace", commissioner.RawArgs{People: &commissioner.RawPeople{
		Franchise: "franchise.02", Principal: "owner.one"}}, "league")
	grant := f.raw(t, "commish.role.grant", commissioner.RawArgs{People: &commissioner.RawPeople{
		Principal: "owner.two", Role: governance.CoCommissioner}}, "private")
	grant.Request = "denied.grant"
	for _, who := range []string{"stranger", "owner.one", "cc"} {
		f.app.commissionerSession = testCommissionerSession{who}
		if who != "cc" {
			_, err := f.app.TargetCommissionerExecute(encodedCommand(t, rulesSet))
			denials[who+"/execute-rules"] = errorWords(err)
			_, err = f.app.TargetCommissionerExecute(encodedCommand(t, seat))
			denials[who+"/execute-seat"] = errorWords(err)
		}
		_, err := f.app.TargetCommissionerExecute(encodedCommand(t, grant))
		denials[who+"/execute-grant"] = errorWords(err)
		for _, command := range []string{commissioner.WaiverProcess, commissioner.BidsOverride, commissioner.TradeOverride} {
			_, err = f.app.TargetCommissionerMarket(legalityLeague, command)
			denials[who+"/market-"+command] = errorWords(err)
		}
		_, err = f.app.TargetCommissionerSchedule(legalityLeague)
		denials[who+"/managed-schedule"] = errorWords(err)
		_, err = f.app.TargetCommissionerDigests(legalityLeague)
		denials[who+"/digests"] = errorWords(err)
		preview, err := f.app.TargetCommissionerPreview(encodedCommand(t, rulesSet))
		denials[who+"/preview-rules"] = errorWords(err) + " " + jsonWords(t, preview.Blockers)
		preview, err = f.app.TargetCommissionerPreview(encodedCommand(t, grant))
		denials[who+"/preview-grant"] = errorWords(err) + " " + jsonWords(t, preview.Blockers)
		ledger, err := f.app.TargetCommissionerLedger(legalityLeague)
		denials[who+"/ledger-entries"] = errorWords(err) + " " + jsonWords(t, len(ledger.Entries))
	}
	f.app.commissionerSession = testCommissionerSession{"owner.one"}
	exception := f.raw(t, commissioner.CalendarException, commissioner.RawArgs{Exception: &commissioner.RawWindowException{
		Franchise: "franchise.99", Action: "waiver_claim", Until: f.now.AddDate(0, 0, 1)}}, "league")
	f.app.commissionerSession = testCommissionerSession{"creator"}
	_, err := f.app.TargetCommissionerExecute(encodedCommand(t, exception))
	denials["creator/exception-wrong-franchise"] = errorWords(err)
	exception.Args.Exception.Franchise, exception.Args.Exception.Until = "franchise.01", f.now.AddDate(0, 0, -1)
	_, err = f.app.TargetCommissionerExecute(encodedCommand(t, exception))
	denials["creator/exception-expired"] = errorWords(err)
	stale := rulesSet
	stale.Expected, stale.Request = stale.Expected-1, "denied.stale"
	_, err = f.app.TargetCommissionerExecute(encodedCommand(t, stale))
	denials["creator/execute-stale"] = errorWords(err)
	for _, routed := range routedCommands(t, f) {
		_, err = f.app.TargetCommissionerExecute(encodedCommand(t, routed))
		denials["creator/execute-routed-"+routed.Command] = errorWords(err)
	}
	for _, scope := range []RawLeagueCalendarScope{
		{League: legalityLeague, Season: 2026, LocalZone: "America/Los_Angeles", Franchise: "franchise.99", Action: "waiver_claim"},
		{League: legalityLeague, Season: 2026, LocalZone: "Local", Franchise: "franchise.01", Action: "waiver_claim"},
		{League: "native.", Season: 2026, LocalZone: "America/Los_Angeles", Franchise: "franchise.01", Action: "waiver_claim"},
	} {
		_, err = f.app.TargetLeagueCalendar(scope)
		denials["calendar/"+scope.League+"/"+scope.Franchise+"/"+scope.LocalZone] = errorWords(err)
	}
	closed, err := f.app.TargetLeagueCalendar(RawLeagueCalendarScope{League: legalityLeague, Season: 2026,
		LocalZone: "America/Los_Angeles", Franchise: "franchise.01", Action: "ufa_bid"})
	denials["calendar/closed-window"] = errorWords(err) + " " + jsonWords(t, closed.Decision)
	for _, league := range []string{"legacy", "native.", " native.seed", "native.missing"} {
		_, err = f.app.TargetLeagueSchedule(RawLeagueScheduleScope{League: league})
		denials["schedule/"+league] = errorWords(err)
		_, err = f.app.TargetCommissionerLedger(league)
		denials["ledger/"+league] = errorWords(err)
		_, err = f.app.TargetCommissionerPeople(league)
		denials["people/"+league] = errorWords(err)
		for name, read := range map[string]func(string) error{
			"rules":   func(l string) error { _, err := f.app.TargetCommissionerRules(l); return err },
			"jobs":    func(l string) error { _, err := f.app.TargetCommissionerJobs(l); return err },
			"notices": func(l string) error { _, err := f.app.TargetCommissionerNotices(l); return err },
			"digests": func(l string) error { _, err := f.app.TargetCommissionerDigests(l); return err },
		} {
			denials[name+"/"+league] = errorWords(read(league))
		}
	}
	_, err = f.app.TargetCommissionerMarket(legalityLeague, commissioner.RulesSet)
	denials["creator/market-not-a-market-act"] = errorWords(err)
	f.app.commissionerSession = failingCommissionerSession{}
	_, err = f.app.TargetCommissionerExecute(encodedCommand(t, rulesSet))
	denials["no-session/execute"] = errorWords(err)
	_, err = f.app.TargetCommissionerActor()
	denials["no-session/actor"] = errorWords(err)
	_, err = f.app.TargetCommissionerMarket(legalityLeague, commissioner.WaiverProcess)
	denials["no-session/market"] = errorWords(err)
	_, err = f.app.TargetCommissionerPreview(encodedCommand(t, rulesSet))
	denials["no-session/preview"] = errorWords(err)
	checkGolden(t, "commissioner", "denials", denials)
}

func jsonWords(t *testing.T, v any) string {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

// routedCommands are well-formed owner acts that the commissioner door sends to their own gated doors.
func routedCommands(t *testing.T, f *waiverFixture) []commissioner.RawCommand {
	t.Helper()
	out := []commissioner.RawCommand{
		f.raw(t, commissioner.TradeOffer, commissioner.RawArgs{Trade: new(tradePlayerOffer())}, "league"),
		f.raw(t, commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
			Player: "1001", Franchise: "franchise.02"}}, "league"),
		f.raw(t, commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
			Window: "window", Franchise: "franchise.01"}}, "private"),
		f.raw(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "1001",
			Franchise: "franchise.01", Salary: moneyOf(1000000), Years: 1}}, "private"),
		f.raw(t, commissioner.NativeSeed, commissioner.RawArgs{Native: heldSeedSnapshot(t)}, "league"),
		f.raw(t, commissioner.LineupSet, commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: 2026,
			Franchise: "franchise.01", Week: 1, Starters: []string{"1001"}}}, "private"),
		f.raw(t, commissioner.DraftPick, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: 2026,
			Pick: &commissioner.RawDraftSelection{Franchise: "franchise.01", Player: "5001", EntrySeason: 2026,
				Round: 1, Slot: 1, Original: "franchise.01", Terms: rookiedraft.Terms{Salary: 1, Years: 1}}}}, "league"),
	}
	for i := range out {
		out[i].Request = "routed." + out[i].Command
	}
	return out
}

func heldSeedSnapshot(t *testing.T) *commissioner.RawNativeSnapshot {
	t.Helper()
	body, err := os.ReadFile("internal/commissioner/testdata/native-assets.wire.json")
	if err != nil {
		t.Fatal(err)
	}
	var wire commissioner.NativeAssets
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	return wire.Snapshot
}
