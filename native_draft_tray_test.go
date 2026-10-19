package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// The live draft tray, the restore and the close, proven against the native draft reading (census CM-043,
// CM-044 and CM-045).

func (f *draftFixture) tryTray(command string, act commissioner.RawDraftAct) (commissioner.Receipt, error) {
	return f.tryCommand(command, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: draftSeason,
		Act: &act}}, "Run the draft from the tray")
}

func (f *draftFixture) tray(t *testing.T, command string, act commissioner.RawDraftAct) commissioner.Receipt {
	t.Helper()
	receipt, err := f.tryTray(command, act)
	if err != nil {
		t.Fatalf("%s: %v", command, err)
	}
	return receipt
}

// restore puts the draft back to how it stood after an entry.
func (f *draftFixture) restore(t *testing.T, to string) commissioner.Receipt {
	t.Helper()
	return f.tray(t, commissioner.DraftRevert, commissioner.RawDraftAct{To: to})
}

// checkpoint is the newest point the draft can be restored to.
func (f *draftFixture) checkpoint(t *testing.T) string {
	t.Helper()
	return f.read(t, "franchise.01").Checkpoints[0].Entry
}

func pickOf(round int, team int) commissioner.RawDraftAct {
	return commissioner.RawDraftAct{Round: round, Original: draftTeam(team)}
}

func withPlayer(a commissioner.RawDraftAct, player string) commissioner.RawDraftAct {
	a.Player = player
	return a
}

// governsLive is the survey's round trip for a change to the live draft, whose compensating entry is a
// restore: read, make the change through the real command, read the consequence, reopen the database for
// real, restore the draft to where it stood, read again; then do it all once more from a reopened database.
func (f *draftFixture) governsLive(t *testing.T, name string, read func() string, set func() commissioner.Receipt,
	before, after string) {
	t.Helper()
	for pass, when := range []string{"live", "after reopening the database"} {
		if pass == 1 {
			f.reopen(t)
		}
		if got := read(); got != before {
			t.Fatalf("%s, %s: before the change: %q, want %q", name, when, got, before)
		}
		point := f.checkpoint(t)
		set()
		if got := read(); got != after {
			t.Fatalf("%s, %s: after the change: %q, want %q", name, when, got, after)
		}
		f.reopen(t)
		if got := read(); got != after {
			t.Fatalf("%s, %s: the change did not survive reopening: %q, want %q", name, when, got, after)
		}
		f.restore(t, point)
		if got := read(); got != before {
			t.Fatalf("%s, %s: after the restore: %q, want %q", name, when, got, before)
		}
	}
}

// rookie describes where a rookie stands: on a roster and at what price, or on none.
func (f *draftFixture) rookie(t *testing.T, id string) string {
	t.Helper()
	p := f.player(t, id)
	if p.ID == "" {
		return "on no roster"
	}
	return fmt.Sprintf("%s at %d", p.Franchise, p.Salary)
}

func pickLine(d commissioner.NativeDraft, index int) string {
	p := d.Draft.Picks[index]
	state := "waiting"
	switch {
	case p.Player != "":
		state = "player " + p.Player + " (" + p.How + ")"
		if p.Changed {
			state += " changed"
		}
	case p.Skipped:
		state = "skipped (" + p.How + ")"
	}
	return fmt.Sprintf("round %d pick %d for %s: %s", p.Round, p.Slot, p.Owner, state)
}

func TestTheTraySkipsAndAssignsAndGovernsTheClockCM043(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.now = draftStartAt().Add(time.Hour)
	onClock := func() string { return onClockOf(f.read(t, "franchise.01")) }
	f.governsLive(t, "skip", onClock, func() commissioner.Receipt {
		return f.tray(t, commissioner.DraftSkip, pickOf(1, 3))
	}, "round 1 pick 1: franchise.03 (owned by franchise.03)",
		"round 1 pick 2: franchise.02 (owned by franchise.01)")
	assigned := func() string {
		return pickLine(f.read(t, "franchise.01"), 0) + "; " + f.rookie(t, "5001")
	}
	f.governsLive(t, "assign", assigned, func() commissioner.Receipt {
		return f.tray(t, commissioner.DraftAssign, withPlayer(pickOf(1, 3), "5001"))
	}, "round 1 pick 1 for franchise.03: waiting; on no roster",
		"round 1 pick 1 for franchise.03: player 5001 (commissioner); franchise.03 at 300000000")
}

func TestTheTrayInsertsAndChangesAPickWithoutRetargetingCM043(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.now = draftStartAt().Add(time.Hour)
	f.tray(t, commissioner.DraftAssign, withPlayer(pickOf(1, 3), "5001"))
	inserted := func() string {
		d := f.read(t, "franchise.01")
		return fmt.Sprintf("%d picks; %s; %s", len(d.Draft.Picks), pickLine(d, 1), f.rookie(t, "5002"))
	}
	insert := pickOf(1, 3)
	insert.Franchise, insert.Player = "franchise.07", "5002"
	f.governsLive(t, "insert", inserted, func() commissioner.Receipt {
		return f.tray(t, commissioner.DraftInsert, insert)
	}, "160 picks; round 1 pick 2 for franchise.01: waiting; on no roster",
		"161 picks; round 1 pick 1 for franchise.07: player 5002 (inserted); franchise.07 at 300000000")
	changed := func() string {
		return pickLine(f.read(t, "franchise.01"), 0) + "; " + f.rookie(t, "5001") + "; " + f.rookie(t, "5003")
	}
	swap := withPlayer(pickOf(1, 3), "5003")
	swap.Replaces = "5001"
	f.governsLive(t, "change", changed, func() commissioner.Receipt {
		return f.tray(t, commissioner.DraftChange, swap)
	}, "round 1 pick 1 for franchise.03: player 5001 (commissioner); franchise.03 at 300000000; on no roster",
		"round 1 pick 1 for franchise.03: player 5003 (commissioner) changed; on no roster; franchise.03 at 300000000")
	// A change aimed at a rookie the pick no longer holds is refused, not applied to whatever is there now.
	stale := swap
	stale.Replaces = "5004"
	before := f.revision(t)
	if _, err := f.tryTray(commissioner.DraftChange, stale); err == nil ||
		!strings.Contains(err.Error(), "different player than the one you were shown") {
		t.Fatalf("a change retargeted silently: %v", err)
	}
	if f.revision(t) != before {
		t.Fatal("a refused change was recorded")
	}
}

func TestTheTrayWorksTheClockAndOverridesAnExpiredPickCM043(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	// Read at the instant the clock opens, so a restore (which gives the pick a fresh clock) reads the same.
	f.now = draftStartAt()
	clock := func() string {
		c := f.read(t, "franchise.01").Draft.OnClock
		return fmt.Sprintf("paused=%v deadline=%s left=%s", c.Paused, c.Deadline.UTC().Format("15:04"), c.Remaining)
	}
	f.governsLive(t, "pause", clock, func() commissioner.Receipt {
		return f.tray(t, commissioner.DraftTimer, commissioner.RawDraftAct{Timer: commissioner.TimerPause})
	}, "paused=false deadline=22:00 left=6h0m0s", "paused=true deadline=22:00 left=6h0m0s")
	f.governsLive(t, "set", clock, func() commissioner.Receipt {
		return f.tray(t, commissioner.DraftTimer, commissioner.RawDraftAct{Timer: commissioner.TimerSet, Seconds: 1800})
	}, "paused=false deadline=22:00 left=6h0m0s", "paused=false deadline=16:30 left=30m0s")
	// Hours later a paused clock has run out of nothing, and resumed it has the time it held.
	f.tray(t, commissioner.DraftTimer, commissioner.RawDraftAct{Timer: commissioner.TimerPause})
	f.now = draftStartAt().Add(20 * time.Hour)
	if got := clock(); got != "paused=true deadline=22:00 left=6h0m0s" {
		t.Fatalf("a paused clock ran on: %s", got)
	}
	f.tray(t, commissioner.DraftTimer, commissioner.RawDraftAct{Timer: commissioner.TimerResume})
	if got := clock(); got != "paused=false deadline=18:00 left=6h0m0s" {
		t.Fatalf("a resumed clock lost the time it held: %s", got)
	}
}

func TestTheClockSkipsAnExpiredPickAndTheCommissionerOverridesItCM043(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.now = draftStartAt().Add(7 * time.Hour)
	// The first pick ran out at six hours with no one recording it. A mark in the ledger after that is a
	// point the draft can be restored to with that skip still in place.
	f.tray(t, commissioner.DraftTimer, commissioner.RawDraftAct{Timer: commissioner.TimerPause})
	f.tray(t, commissioner.DraftTimer, commissioner.RawDraftAct{Timer: commissioner.TimerResume})
	first := func() string { return pickLine(f.read(t, "franchise.01"), 0) + "; " + f.rookie(t, "5001") }
	f.governsLive(t, "override", first, func() commissioner.Receipt {
		return f.tray(t, commissioner.DraftAssign, withPlayer(pickOf(1, 3), "5001"))
	}, "round 1 pick 1 for franchise.03: skipped (expired); on no roster",
		"round 1 pick 1 for franchise.03: player 5001 (overrode); franchise.03 at 300000000")
	if on := f.read(t, "franchise.01").Draft.OnClock; on.Slot != 2 {
		t.Fatalf("overriding a skipped pick moved the clock: %+v", on)
	}
}

func TestTheTrayRefusesWhatItShouldNotAndNeedsARoleAndAReasonCM043(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.seated(t, 3)
	f.scheduled(t, 3, 2, 1)
	if _, err := f.tryTray(commissioner.DraftSkip, pickOf(1, 3)); err == nil ||
		!strings.Contains(err.Error(), "has not started") {
		t.Fatalf("a skip before the draft started: %v", err)
	}
	f.now = draftStartAt().Add(time.Hour)
	f.asOwner(3)
	if _, err := f.tryTray(commissioner.DraftSkip, pickOf(1, 3)); err == nil ||
		!strings.Contains(err.Error(), "role") {
		t.Fatalf("an owner worked the tray: %v", err)
	}
	f.actAs("creator")
	skip := commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{Season: draftSeason, Act: &commissioner.RawDraftAct{
		Round: 1, Original: "franchise.03"}}}
	if _, err := f.tryCommand(commissioner.DraftSkip, skip, ""); err == nil ||
		!strings.Contains(err.Error(), "give a reason") {
		t.Fatalf("a skip with no typed reason: %v", err)
	}
	for name, try := range map[string]func() error{
		"skip a pick that is not on the clock": func() error {
			_, err := f.tryTray(commissioner.DraftSkip, pickOf(1, 2))
			return err
		},
		"assign a rookie of another class": func() error {
			_, err := f.tryTray(commissioner.DraftAssign, withPlayer(pickOf(1, 3), "1000"))
			return err
		},
		"change a pick that was never made": func() error {
			a := withPlayer(pickOf(1, 3), "5002")
			a.Replaces = "5001"
			_, err := f.tryTray(commissioner.DraftChange, a)
			return err
		},
		"set a clock of no time": func() error {
			_, err := f.tryTray(commissioner.DraftTimer, commissioner.RawDraftAct{Timer: commissioner.TimerSet})
			return err
		},
		"resume a clock that is running": func() error {
			_, err := f.tryTray(commissioner.DraftTimer, commissioner.RawDraftAct{Timer: commissioner.TimerResume})
			return err
		},
	} {
		before := f.revision(t)
		if err := try(); err == nil {
			t.Fatalf("%s was accepted", name)
		}
		if f.revision(t) != before {
			t.Fatalf("a refused change to %s was recorded", name)
		}
	}
}

// ledgerWhats lists what the ledger says each entry did, newest first, keyed by entry.
func (f *draftFixture) ledgerWhats(t *testing.T) map[string]string {
	t.Helper()
	ledger, err := f.app.TargetCommissionerLedger(draftLeague)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, e := range ledger.Entries {
		out[e.ID] = e.What
	}
	return out
}

func (f *draftFixture) made(t *testing.T) string {
	t.Helper()
	d := f.read(t, "franchise.01")
	var made []string
	for _, p := range d.Draft.Picks {
		if p.Player != "" {
			made = append(made, fmt.Sprintf("%s:%s", p.Player, p.Owner))
		}
	}
	on := "nobody"
	if d.Draft.OnClock != nil {
		on = fmt.Sprintf("round %d pick %d", d.Draft.OnClock.Round, d.Draft.OnClock.Slot)
	}
	return fmt.Sprintf("made %v; %s on the clock; %s, %s", made, on, f.rookie(t, "5001"), f.rookie(t, "5002"))
}

func TestARestoreKeepsBothStatesAndTheLedgerCM044(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.seated(t, 1, 3)
	f.now = draftStartAt().Add(time.Hour)
	f.asOwner(3)
	firstPick := f.pick(t, "franchise.03", "5001")
	f.actAs("creator")
	for pass, when := range []string{"live", "after reopening the database"} {
		if pass == 1 {
			f.reopen(t)
		}
		if got := f.made(t); got != "made [5001:franchise.03]; round 1 pick 2 on the clock; franchise.03 at 300000000, on no roster" {
			t.Fatalf("%s: before the second pick: %s", when, got)
		}
		f.asOwner(1)
		secondPick := f.pick(t, "franchise.01", "5002")
		f.actAs("creator")
		afterPick := f.made(t)
		if afterPick != "made [5001:franchise.03 5002:franchise.01]; round 1 pick 3 on the clock; franchise.03 at 300000000, franchise.01 at 300000000" {
			t.Fatalf("%s: after the second pick: %s", when, afterPick)
		}
		before := f.ledgerWhats(t)
		restore := f.restore(t, firstPick.Entry)
		if got := f.made(t); got != "made [5001:franchise.03]; round 1 pick 2 on the clock; franchise.03 at 300000000, on no roster" {
			t.Fatalf("%s: after the restore: %s", when, got)
		}
		after := f.ledgerWhats(t)
		for entry, what := range before {
			if after[entry] != what {
				t.Fatalf("%s: the restore changed or erased the ledger entry %q (%q)", when, what, after[entry])
			}
		}
		if len(after) != len(before)+1 || !strings.HasPrefix(after[restore.Entry], "Restored the 2027 rookie draft to how it stood after: Picked player 5001") {
			t.Fatalf("%s: the restore is not one new entry that names its checkpoint: %q", when, after[restore.Entry])
		}
		if after[secondPick.Entry] == "" {
			t.Fatalf("%s: the restore deleted the second pick from the ledger", when)
		}
		f.reopen(t)
		if got := f.made(t); got != "made [5001:franchise.03]; round 1 pick 2 on the clock; franchise.03 at 300000000, on no roster" {
			t.Fatalf("%s: the restore did not survive reopening: %s", when, got)
		}
	}
}

func TestARestoreIsRefusedWhereItWouldBreakTheDraftCM044(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	order, start := f.scheduled(t, 3, 2, 1)
	f.seated(t, 3)
	f.now = draftStartAt().Add(time.Hour)
	f.asOwner(3)
	pick := f.pick(t, "franchise.03", "5001")
	f.actAs("creator")
	rule := f.setRule(t, "draft.clock", `7200000000000`)
	for name, to := range map[string]string{"an entry that is not in this draft": rule.Entry,
		"an entry that does not exist": "act.nothing", "a blank point": " "} {
		before := f.revision(t)
		if _, err := f.tryTray(commissioner.DraftRevert, commissioner.RawDraftAct{To: to}); err == nil {
			t.Fatalf("a restore to %s was accepted", name)
		}
		if f.revision(t) != before {
			t.Fatalf("a refused restore to %s was recorded", name)
		}
	}
	// A change to the live draft cannot be undone one entry at a time; only a restore compensates it.
	if _, err := f.tryCommand(commissioner.Revert, commissioner.RawArgs{Entry: pick.Entry}, "Undo it"); err == nil ||
		!strings.Contains(err.Error(), "restore the draft to an earlier point") {
		t.Fatalf("a made pick was undone on its own: %v", err)
	}
	// The draft's own settings can still be restored to.
	f.restore(t, start.Entry)
	f.restore(t, order.Entry)
	if got := f.made(t); !strings.Contains(got, "made []; round 1 pick 1 on the clock") {
		t.Fatalf("restoring to the draft's order did not clear the picks: %s", got)
	}
	if _, err := f.tryCommand(commissioner.DraftRevert, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{
		Season: draftSeason}}, "No point"); err == nil {
		t.Fatal("a restore with no point was accepted")
	}
}

// draftEnds moves the clock on until every pick of a five round draft has been used or has run out.
func (f *draftFixture) draftEnds() { f.now = f.now.Add(160*6*time.Hour + time.Hour) }

// closedWords is what a closed draft did, as the draft room reads it and as the assets hold it.
func (f *draftFixture) closedWords(t *testing.T) string {
	t.Helper()
	d := f.read(t, "franchise.01")
	assets, err := f.app.TargetNativeAssets(draftLeague)
	if err != nil {
		t.Fatal(err)
	}
	if d.Closed == nil {
		return fmt.Sprintf("open; %d rookies available; pool %d; picks %d", len(d.Available), len(assets.Snapshot.Pool),
			len(assets.Snapshot.Picks))
	}
	var pool, issued []string
	for _, p := range d.Closed.Pool {
		pool = append(pool, p.ID)
	}
	for _, i := range d.Closed.Issued {
		issued = append(issued, fmt.Sprintf("%d:%d rounds,%d picks,tradable=%v", i.Season, i.Rounds, i.Picks, i.Tradable))
	}
	held := map[int]int{}
	for _, p := range assets.Snapshot.Picks {
		if p.Season > draftSeason && p.Owner == p.Original && p.Player == "" && !p.Skipped {
			held[p.Season]++
		}
	}
	var poolAssets []string
	for _, p := range assets.Snapshot.Pool {
		poolAssets = append(poolAssets, p.Player)
	}
	return fmt.Sprintf("closed automatic=%v; pool %v; pool assets %v; issued %v; held %v", d.Closed.Automatic, pool,
		poolAssets, issued, held)
}

func TestClosingTheDraftReleasesRookiesAndIssuesPicksCM045(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.seated(t, 1, 3)
	f.now = draftStartAt().Add(time.Hour)
	f.asOwner(3)
	f.pick(t, "franchise.03", "5001")
	f.actAs("creator")
	const open = "open; 5 rookies available; pool 0; picks 160"
	const closed = "closed automatic=true; pool [5002 5003 5004 5005 5006]; pool assets [5002 5003 5004 5005 5006]; " +
		"issued [2028:5 rounds,160 picks,tradable=true 2029:5 rounds,160 picks,tradable=true]; " +
		"held map[2028:160 2029:160]"
	for pass, when := range []string{"live", "after reopening the database"} {
		if pass == 1 {
			f.reopen(t)
		}
		point := f.checkpoint(t)
		f.draftEnds()
		if got := f.closedWords(t); got != open {
			t.Fatalf("%s: the finished draft before it closed: %s", when, got)
		}
		if words := f.read(t, "franchise.01").Words; words != "Every pick has been used; the draft has not been closed." {
			t.Fatalf("%s: the finished draft says %q", when, words)
		}
		f.app.processNativeDraftsDue(t.Context())
		if got := f.closedWords(t); got != closed {
			t.Fatalf("%s: the closed draft: %s\nwant %s", when, got, closed)
		}
		f.reopen(t)
		if got := f.closedWords(t); got != closed {
			t.Fatalf("%s: the close did not survive reopening: %s", when, got)
		}
		// The draft closes once; asking again records nothing.
		before := f.revision(t)
		f.app.processNativeDraftsDue(t.Context())
		if f.revision(t) != before {
			t.Fatalf("%s: a closed draft was closed again", when)
		}
		// The compensating entry: restore the draft to before its last picks, and it is open again.
		f.restore(t, point)
		if got := f.closedWords(t); got != "open; 5 rookies available; pool 0; picks 160" {
			t.Fatalf("%s: after the restore: %s", when, got)
		}
		f.reopen(t)
		if got := f.closedWords(t); got != "open; 5 rookies available; pool 0; picks 160" {
			t.Fatalf("%s: the restore did not survive reopening: %s", when, got)
		}
	}
}

func TestACommissionerCanCloseTheDraftEarlyAndNothingMoreIsPicked(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.seated(t, 1, 3)
	f.now = draftStartAt().Add(time.Hour)
	f.asOwner(3)
	f.pick(t, "franchise.03", "5001")
	f.actAs("creator")
	if _, err := f.tryCommand(commissioner.DraftArchive, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{
		Season: draftSeason}}, ""); err != nil {
		t.Fatalf("a commissioner could not close the draft: %v", err)
	}
	d := f.read(t, "franchise.01")
	if d.Closed == nil || d.Closed.Automatic || len(d.Closed.Pool) != 5 || d.Draft.OnClock != nil || !d.Draft.Exhausted ||
		d.Words != "The rookie draft is closed." {
		t.Fatalf("the early close: %+v", d.Closed)
	}
	if last := d.Draft.Picks[159]; !last.Skipped || last.Player != "" {
		t.Fatalf("a pick left over at the close was not set aside: %+v", last)
	}
	f.asOwner(1)
	if _, err := f.tryPick("franchise.01", "5002", "late", f.revision(t)); err == nil ||
		!strings.Contains(err.Error(), "closed") {
		t.Fatalf("a pick was made after the close: %v", err)
	}
	f.actAs("creator")
	if _, err := f.tryTray(commissioner.DraftSkip, pickOf(1, 2)); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("the tray changed a closed draft: %v", err)
	}
	if _, err := f.tryCommand(commissioner.DraftArchive, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{
		Season: draftSeason}}, ""); err == nil {
		t.Fatal("a closed draft was closed again")
	}
}
