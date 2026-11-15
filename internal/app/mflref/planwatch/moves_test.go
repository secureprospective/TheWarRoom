package planwatch

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
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
