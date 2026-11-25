package history

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestHumanExpiryAutomaticRankingAndWait(t *testing.T) {
	for _, policy := range []string{"autopick", "wait"} {
		t.Run(policy, func(t *testing.T) {
			facts := testFacts(6)
			f := newAutoPickFixture(t, "native.expiry"+policy, commissioner.PresetDefault32, 2, 1, "", []string{"franchise.01", "franchise.02"}, facts)
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
	f := newAutoPickFixture(t, "native.owed", commissioner.PresetDefault32, 2, 1, "", []string{"franchise.01", "franchise.02"}, facts)
	f.seat("franchise.01", "owner.one")
	f.setRule("draft.clock", "10000000000")
	f.openStartup()
	f.c.now = f.c.now.Add(11 * time.Second)
	if err := f.s.ProcessDueAutoPicks(t.Context(), f.league, f.c.now, facts, fixedDraftRanking{order: []string{"1001", "1002"}}); err != nil {
		t.Fatal(err)
	}
	recs, err := f.s.nativeRecords(t.Context(), f.league)
	if err != nil {
		t.Fatal(err)
	}
	if due := commissioner.DueDrafts(recs, f.c.now, nil); len(due) != 0 {
		t.Fatalf("owed draft auto-closes: %+v", due)
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
		League: f.league, Season: 2026, Franchise: "franchise.01", Player: "1002", Request: "late", Expected: int64(len(recs)), Startup: true,
	}, "owner.one", facts, time.UTC); err != nil {
		t.Fatal(err)
	}
	d := f.draftReading("franchise.01")
	p := d.Draft.Picks[0]
	if p.Player != "1002" || p.Round != 1 || p.Slot != 1 || p.Contract == nil {
		t.Fatalf("wrong owed selection: %+v", p)
	}
}
