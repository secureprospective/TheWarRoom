package platform

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestMoveURLSecondLock(t *testing.T) {
	t.Parallel()
	for _, target := range []string{
		"https://example.com/", "http://myfantasyleague.com/",
		"https://owner@myfantasyleague.com/", "https://myfantasyleague.com.evil.test/",
		"https://evilmyfantasyleague.com/", "https:///options", ":bad",
	} {
		if err := ValidateMoveURL(target); err == nil {
			t.Fatal("accepted", target)
		}
	}
	for _, target := range []string{
		"https://myfantasyleague.com/", "https://www47.myfantasyleague.com/options?O=18",
	} {
		if err := ValidateMoveURL(target); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTaxiPredicateRegistered(t *testing.T) {
	t.Parallel()
	predicate := MovePredicate("roster.taxi")
	if _, ok := predicate.(envelope.TaxiPredicate); !ok {
		t.Fatalf("taxi registry: %T", predicate)
	}
}

func TestMovePolicies(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		intent string
		want   envelope.Predicate
	}{
		{"roster.ir", envelope.IRPredicate{}}, {"roster.taxi", envelope.TaxiPredicate{}},
		{"mfl.lineup.set", envelope.LineupPredicate{}}, {"mfl.trade.accept", envelope.TradePredicate{}},
		{"trade.propose", envelope.TradeActPredicate{}}, {"trade.reject", envelope.TradeActPredicate{}},
		{"trade.revoke", envelope.TradeActPredicate{}}, {"bid.submit", envelope.BidPredicate{}},
		{envelope.CutIntent, envelope.ReleasePredicate{}}, {envelope.BuyoutIntent, envelope.ReleasePredicate{}},
		{"rfa.tender", envelope.ContractPredicate{}}, {"fa.resign", envelope.ContractPredicate{}},
		{"contract.tag", envelope.ContractPredicate{}}, {"contract.extend", envelope.ContractPredicate{}},
		{"contract.restructure", envelope.ContractPredicate{}}, {"unknown", nil},
	} {
		if got := MovePredicate(tc.intent); reflect.TypeOf(got) != reflect.TypeOf(tc.want) {
			t.Errorf("%s: %T, want %T", tc.intent, got, tc.want)
		}
	}
	sources := map[envelope.Source]bool{envelope.Rosters: true, envelope.Contracts: true}
	if !SkipMoveRefresh(envelope.Contracts, sources) || SkipMoveRefresh(envelope.Rosters, sources) || !SkipMoveRefresh(envelope.Lineups, sources) {
		t.Fatal("roster refresh must cover contracts, not missing sources")
	}
	delete(sources, envelope.Rosters)
	if SkipMoveRefresh(envelope.Contracts, sources) {
		t.Fatal("contracts must refresh without rosters")
	}
}

func TestRosterMoveTarget(t *testing.T) {
	t.Parallel()
	missing, blocks := RosterMoveTarget("", 2026, "league", "18")
	if missing.Kind != envelope.Unmapped || missing.URL != "" || !reflect.DeepEqual(blocks, []string{"MFL host not yet known"}) {
		t.Fatalf("missing: %+v %v", missing, blocks)
	}
	got, blocks := RosterMoveTarget("discovered.test", 2026, "a&b", "18")
	if got.Kind != envelope.Mapped || got.URL != "https://discovered.test/2026/options?L=a%26b&O=18" || blocks != nil {
		t.Fatalf("mapped: %+v %v", got, blocks)
	}
}

func TestBuildLabel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ version, commit, want string }{{"dev", "", "dev"}, {"v1", "abc1234", "v1 (abc1234)"}} {
		if got := BuildLabel(tc.version, tc.commit); got != tc.want {
			t.Errorf("label = %q, want %q", got, tc.want)
		}
	}
}

func TestMoveEvidenceAndWatchability(t *testing.T) {
	t.Parallel()
	at := time.Unix(1700000000, 0)
	makeMove := func(intent string, state envelope.State, note string) envelope.Envelope {
		t.Helper()
		first, err := playerid.New("0042")
		if err != nil {
			t.Fatal(err)
		}
		second, err := playerid.New("0099")
		if err != nil {
			t.Fatal(err)
		}
		spec := envelope.Spec{Intent: intent, LeagueID: "1", FranchiseID: "0001", Gravity: envelope.G2, Undo: envelope.Reversible, Target: envelope.Target{Kind: envelope.Manual}, Subject: envelope.Subject{Players: []playerid.PlayerID{first}}, Expected: envelope.ExpectedChange{Player: first, RosterStatus: domain.RosterIR}}
		if envelope.IsTrade(intent) {
			spec.Subject.Players = append(spec.Subject.Players, second)
			offering, accepting := "0002", "0001"
			if intent == "trade.propose" {
				offering, accepting = "0001", "0002"
			}
			spec.Expected = envelope.ExpectedChange{Trade: &envelope.ExpectedTrade{TradeID: "77", Offering: offering, Accepting: accepting, OfferingGives: []string{"0042"}, AcceptingGives: []string{"0099"}}}
		}
		audit := []envelope.AuditEntry{{At: at, From: envelope.Draft, Event: envelope.ChecksPass, To: envelope.Ready, Note: "ready"}, {At: at, From: envelope.Ready, Event: envelope.HandOff, To: envelope.HandedOff, Note: note}}
		if state == envelope.Landed {
			audit = append(audit, envelope.AuditEntry{At: at, From: envelope.HandedOff, Event: envelope.Match, To: state, Note: note})
		}
		e, err := envelope.Restore("move", at, spec, audit)
		if err != nil {
			t.Fatal(err)
		}
		return e
	}

	ir := makeMove("roster.ir", envelope.HandedOff, "old")
	contract := makeMove("contract.extend", envelope.HandedOff, "old")
	if !WatchableMove(ir, envelope.HandedOff) || WatchableMove(ir, envelope.Ready) || WatchableMove(contract, envelope.HandedOff) {
		t.Fatal("watchability changed")
	}
	for _, tc := range []struct {
		intent string
		next   envelope.State
		note   string
		want   bool
	}{
		{"roster.ir", envelope.Landed, "old", true}, {"roster.ir", envelope.HandedOff, "new", false},
		{"mfl.trade.accept", envelope.HandedOff, "new", false}, {"trade.propose", envelope.HandedOff, "new", true},
		{"trade.reject", envelope.HandedOff, "old", false},
	} {
		before := makeMove(tc.intent, envelope.HandedOff, "old")
		after := makeMove(tc.intent, tc.next, tc.note)
		if got := SaveMoveEvidence(before, after); got != tc.want {
			t.Errorf("%+v: %v", tc, got)
		}
	}
}
