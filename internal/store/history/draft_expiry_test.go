package history

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestHumanExpiryAutomaticRankingAndWait(t *testing.T) {
	for _, policy := range []string{"autopick", "wait"} {
		t.Run(policy, func(t *testing.T) {
			facts := testFacts(6)
			f := newAutoPickFixture(t, "native.expiry"+policy,
				commissioner.PresetDefault32, 2, 1, "",
				[]string{"franchise.01", "franchise.02"}, facts)
			f.seat("franchise.01", "owner.one")
			f.setRule("draft.onExpire", `"`+policy+`"`)
			f.setRule("draft.clock", "10000000000")
			f.openStartup()
			rank := fixedDraftRanking{order: []string{"1001", "1002"}}
			process := func() {
				t.Helper()
				if err := f.s.ProcessDueAutoPicks(t.Context(), f.league, f.c.now, facts, rank); err != nil {
					t.Fatal(err)
				}
			}
			process()
			if f.draftReading("franchise.01").Draft.Picks[0].Player != "" {
				t.Fatal("picked before deadline")
			}
			f.c.now = f.c.now.Add(11 * time.Second)
			process()
			d := f.draftReading("franchise.01")
			if policy == "autopick" {
				if d.Draft.Picks[0].Player != "1001" || d.Draft.Picks[1].Player != "1002" {
					t.Fatalf("ranking or vacant selection lost: %+v", d.Draft.Picks)
				}
			} else if d.Draft.OnClock == nil || d.Draft.OnClock.Slot != 1 || d.Draft.Picks[0].Skipped {
				t.Fatalf("wait expired: %+v", d.Draft)
			}
		})
	}
}

func TestOwedPickUsesOriginalSlotAfterTurnsEnd(t *testing.T) {
	facts := testFacts(6)
	f := newAutoPickFixture(t, "native.owed", commissioner.PresetDefault32,
		2, 1, "", []string{"franchise.01", "franchise.02"}, facts)
	f.seat("franchise.01", "owner.one")
	f.setRule("draft.clock", "10000000000")
	f.openStartup()
	f.c.now = f.c.now.Add(11 * time.Second)
	if err := f.s.ProcessDueAutoPicks(t.Context(), f.league, f.c.now, facts,
		fixedDraftRanking{order: []string{"1001", "1002"}}); err != nil {
		t.Fatal(err)
	}
	recs, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	if due := commissioner.DueDrafts(recs, f.c.now, nil); len(due) != 0 {
		t.Fatalf("owed draft auto-closes: %+v", due)
	}
	reading := f.draftReading("franchise.01")
	if len(reading.Draft.Picks) == 0 || reading.Draft.Picks[0].Contract == nil ||
		reading.Draft.Picks[0].Round != 1 || reading.Draft.Picks[0].Slot != 1 {
		t.Fatalf("owner reading lacks the oldest owed slot's terms: %+v", reading.Draft.Picks)
	}
	terms := *reading.Draft.Picks[0].Contract
	other := f.draftReading("franchise.02")
	if other.Draft.Picks[0].Contract != nil {
		t.Fatalf("another franchise received the owed price: %+v", other.Draft.Picks[0])
	}
	gate, err := commissioner.NativeActionGate(f.league, recs, "franchise.01", "draft_pick", f.c.now, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	permit, ok := gate.Permit()
	if !ok {
		t.Fatalf("gate closed: %s", gate.Sentence)
	}
	if _, err := f.s.RecordDraftPick(t.Context(), permit, commissioner.DraftPickInput{
		League: f.league, Season: 2026, Franchise: "franchise.01", Player: "1002",
		Request: "late", Expected: int64(len(recs)), Startup: true,
	}, "owner.one", facts, time.UTC); err != nil {
		t.Fatal(err)
	}
	d := f.draftReading("franchise.01")
	p := d.Draft.Picks[0]
	if p.Player != "1002" || p.Round != 1 || p.Slot != 1 || p.Contract == nil {
		t.Fatalf("wrong owed selection: %+v", p)
	}
	if *p.Contract != terms {
		t.Fatalf("recorded terms %+v differ from owed-read terms %+v", *p.Contract, terms)
	}
}

func TestVacantFranchiseAutopickUsesLiveSlotAndLeavesOwedPick(t *testing.T) {
	facts := testFacts(8)
	f := newAutoPickFixture(t, "native.owedautopick", commissioner.PresetDefault32, 2, 2, "",
		[]string{"franchise.01", "franchise.02"}, facts)
	f.seat("franchise.01", "owner.one")
	f.setRule("draft.clock", "10000000000")
	f.openStartup()
	f.c.now = f.c.now.Add(11 * time.Second)

	// The old human deadline is remembered even though the other franchise is vacant.
	if _, err := f.s.ProcessDueAutoPicksUpTo(t.Context(), f.league, f.c.now, facts,
		fixedDraftRanking{order: []string{"1001", "1002", "1003"}}, 2); err != nil {
		t.Fatal(err)
	}
	reading := f.draftReading("franchise.01")
	if reading.Draft.Picks[0].How != "expired" || reading.Draft.OnClock == nil ||
		reading.Draft.OnClock.Owner != "franchise.01" || reading.Draft.OnClock.Round != 2 {
		t.Fatalf("expected an owed round-one pick and live round-two turn: %+v", reading.Draft)
	}
	if reading.Draft.Picks[0].Contract == nil || reading.Draft.Picks[0].Round != 1 {
		t.Fatalf("oldest owed slot terms are missing beside the live turn: %+v", reading.Draft.Picks[0])
	}
	liveTerms := reading.Draft.OnClock.Contract
	// Releasing the seat makes its current turn eligible for computer selection.
	f.exec("creator", "commish.seat.remove", commissioner.RawArgs{
		People: &commissioner.RawPeople{Franchise: "franchise.01"}})
	before, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.ProcessDueAutoPicksUpTo(t.Context(), f.league, f.c.now, facts,
		fixedDraftRanking{order: []string{"1001", "1002", "1003"}}, 1); err != nil {
		t.Fatal(err)
	}
	after, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before)+1 {
		t.Fatalf("made %d records, want one: before=%d after=%d", len(after)-len(before), len(before), len(after))
	}
	pick := after[len(after)-1].Command.Args.Draft.Pick
	if pick == nil || pick.Round != 2 || pick.Slot != 2 {
		t.Fatalf("automatic pick did not use the live slot: %+v", pick)
	}
	if pick.Terms != liveTerms {
		t.Fatalf("automatic pick used terms %+v, live slot terms are %+v", pick.Terms, liveTerms)
	}
	reading = f.draftReading("franchise.01")
	if reading.Draft.Picks[0].Player != "" || !reading.Draft.Picks[0].Skipped ||
		reading.Draft.Picks[3].Player != pick.Player || !reading.Draft.Exhausted ||
		reading.Draft.OnClock != nil {
		t.Fatalf("automatic selection did not preserve the owed slot and fill the live turn: %+v", reading.Draft)
	}
}

func TestClosedStartupDraftKeepsExpiryPolicyAtClose(t *testing.T) {
	facts := testFacts(8)
	f := newAutoPickFixture(t, "native.closedexpiry", commissioner.PresetDefault32,
		2, 1, "", []string{"franchise.01", "franchise.02"}, facts)
	f.seat("franchise.01", "owner.one")
	f.seat("franchise.02", "owner.two")
	f.setRule("calendar.rollover", `"2027-03-01T05:00:00Z"`)
	f.setRule("draft.clock", "10000000000")
	f.openStartup()
	f.c.now = f.c.now.Add(11 * time.Second)
	pick := func(franchise, owner, player, request string) {
		t.Helper()
		records, err := f.s.nativeRecords(t.Context(), f.league)
		if err != nil {
			t.Fatal(err)
		}
		gate, err := commissioner.NativeActionGate(f.league, records, franchise,
			"draft_pick", f.c.now, time.UTC)
		if err != nil {
			t.Fatal(err)
		}
		permit, ok := gate.Permit()
		if !ok {
			t.Fatalf("pick gate closed: %s", gate.Sentence)
		}
		_, err = f.s.RecordDraftPick(t.Context(), permit, commissioner.DraftPickInput{
			League: f.league, Season: 2026, Franchise: franchise,
			Player: player, Request: request, Expected: int64(len(records)), Startup: true,
		}, owner, facts, time.UTC)
		if err != nil {
			t.Fatal(err)
		}
	}
	pick("franchise.02", "owner.two", "1001", "later-live-pick")
	pick("franchise.01", "owner.one", "1002", "late-owed-pick")
	if err := f.s.CloseFinishedDrafts(t.Context(), f.league, facts); err != nil {
		t.Fatal(err)
	}

	records, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	raw := commissioner.RawCommand{League: f.league, Command: commissioner.RulesSet,
		Args:     commissioner.RawArgs{Field: "draft.onExpire", Value: json.RawMessage(`"wait"`)},
		Expected: int64(len(records)), Request: "future-expiry", Reason: "Test setup",
		Effective: "next_rollover", Visibility: "league"}
	if _, err := f.s.ExecuteCommissionerWithFacts(t.Context(), raw, "creator", nil,
		facts, time.UTC); err != nil {
		t.Fatal(err)
	}
	f.c.now = time.Date(2027, 4, 1, 12, 0, 0, 0, time.UTC)
	records, err = f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	rules, err := commissioner.RulesNow(records, f.c.now)
	if err != nil || string(rules.Draft.OnExpire.Value) != "wait" {
		t.Fatalf("next-rollover expiry setting not in force: %+v %v", rules.Draft.OnExpire, err)
	}
	d := f.draftReading("franchise.01")
	if d.Draft.Picks[0].Player != "1002" || d.Draft.Picks[1].Player != "1001" ||
		d.Closed == nil {
		t.Fatalf("closed draft history changed after expiry policy rollover: %+v", d.Draft)
	}
}
