package history

import (
	"errors"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestAutoPickRetainsTurnAcrossDurabilityPause(t *testing.T) {
	facts := testFacts(6)
	f := newAutoPickFixture(t, "native.lostpickrepro", commissioner.PresetOneSeason10, 2, 1, "", []string{"franchise.01", "franchise.02"}, facts)
	f.setRule("draft.clock", "10000000000")
	f.openStartup()
	refused := errors.New("changes are paused because the off-site backup is not current")
	f.s.pools.SetWriteGuard(func() error { return refused })
	rank := fixedDraftRanking{order: []string{"1001", "1002"}}
	if err := f.s.ProcessDueAutoPicks(t.Context(), f.league, f.c.now, facts, rank); !errors.Is(err, refused) {
		t.Fatalf("expected durability refusal, got %v", err)
	}
	f.c.now = f.c.now.Add(11 * time.Second)
	f.s.pools.SetWriteGuard(nil)
	if err := f.s.ProcessDueAutoPicks(t.Context(), f.league, f.c.now, facts, rank); err != nil {
		t.Fatal(err)
	}
	d := f.draftReading("franchise.01")
	if d.Draft.Picks[0].Player != "1001" || d.Draft.Picks[0].Skipped {
		t.Fatalf("durability pause lost vacant team's pick: %+v", d.Draft.Picks[0])
	}
}

func TestHumanTurnStillExpiresAcrossDurabilityPause(t *testing.T) {
	facts := testFacts(6)
	f := newAutoPickFixture(t, "native.humanpause", commissioner.PresetOneSeason10, 2, 1, "", []string{"franchise.01", "franchise.02"}, facts)
	f.setRule("draft.clock", "10000000000")
	f.seat("franchise.01", "owner.one")
	f.openStartup()
	refused := errors.New("writes paused")
	f.s.pools.SetWriteGuard(func() error { return refused })
	f.c.now = f.c.now.Add(11 * time.Second)
	f.s.pools.SetWriteGuard(nil)
	if err := f.s.ProcessDueAutoPicks(t.Context(), f.league, f.c.now, facts, fixedDraftRanking{order: []string{"1001", "1002"}}); err != nil {
		t.Fatal(err)
	}
	p := f.draftReading("franchise.01").Draft.Picks[0]
	if !p.Skipped || p.Player != "" || p.How != "expired" {
		t.Fatalf("human timeout changed: %+v", p)
	}
}
