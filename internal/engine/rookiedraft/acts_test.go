package rookiedraft

import (
	"strings"
	"testing"
	"time"
)

func actsDraft(t *testing.T, at time.Time, acts ...Act) Draft {
	t.Helper()
	r := testRules().Draft
	d, err := ReplayActs(r, testOrder(t, r), fixedStart(), at, acts)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func actsError(t *testing.T, at time.Time, acts ...Act) string {
	t.Helper()
	r := testRules().Draft
	if _, err := ReplayActs(r, testOrder(t, r), fixedStart(), at, acts); err == nil {
		t.Fatalf("acts accepted: %+v", acts)
	} else {
		return err.Error()
	}
	return ""
}

func first() PickRef  { return PickRef{Round: 1, Original: "a"} }
func second() PickRef { return PickRef{Round: 1, Original: "b"} }

func TestACommissionerSkipPassesTheTurnAtOnce(t *testing.T) {
	start := fixedStart()
	d := actsDraft(t, start.Add(time.Hour), Act{Kind: Skip, At: start.Add(time.Hour), Pick: first()})
	if !d.Picks[0].Skipped || d.Picks[0].How != HowSkipped || d.OnClock == nil || d.OnClock.Slot != 2 ||
		!d.OnClock.Deadline.Equal(start.Add(7*time.Hour)) {
		t.Fatalf("a skip did not pass the turn: %+v", d.OnClock)
	}
	if msg := actsError(t, start.Add(time.Hour), Act{Kind: Skip, At: start.Add(time.Hour), Pick: second()}); !strings.Contains(
		msg, "only the pick on the clock") {
		t.Fatalf("skipped a pick that is not on the clock: %s", msg)
	}
}

func TestACommissionerAssignsTheClockPickOrOverridesASkip(t *testing.T) {
	start := fixedStart()
	assigned := Act{Kind: Assign, At: start.Add(time.Hour), Pick: first(), Player: "0001"}
	d := actsDraft(t, start.Add(time.Hour), assigned)
	if d.Picks[0].Player != "0001" || d.Picks[0].How != HowCommissioner || d.Picks[0].Contract == nil ||
		d.OnClock.Slot != 2 {
		t.Fatalf("assign did not use the pick: %+v", d.Picks[0])
	}
	// The clock runs out on the first pick at 6h; the commissioner then overrides the skip at 7h.
	override := Act{Kind: Assign, At: start.Add(7 * time.Hour), Pick: first(), Player: "0002"}
	d = actsDraft(t, start.Add(7*time.Hour), override)
	if d.Picks[0].Skipped || d.Picks[0].Player != "0002" || d.Picks[0].How != HowOverrode {
		t.Fatalf("a skipped pick was not overridden: %+v", d.Picks[0])
	}
	if d.OnClock.Slot != 2 || !d.OnClock.Deadline.Equal(start.Add(12*time.Hour)) {
		t.Fatalf("overriding a skip moved the clock: %+v", d.OnClock)
	}
	for _, bad := range []Act{
		{Kind: Assign, At: start.Add(time.Hour), Pick: second(), Player: "0001"},
		{Kind: Assign, At: start.Add(time.Hour), Pick: first(), Player: "9"},
	} {
		actsError(t, start.Add(time.Hour), bad)
	}
}

func TestAnInsertedPickTakesNoTurnAndKeepsItsOwnContract(t *testing.T) {
	start := fixedStart()
	insert := Act{Kind: Insert, At: start.Add(time.Hour), Pick: first(), Owner: "b", Player: "0003"}
	d := actsDraft(t, start.Add(time.Hour), insert)
	if len(d.Picks) != 11 || d.Picks[1].Extra != 1 || d.Picks[1].Owner != "b" || d.Picks[1].Player != "0003" ||
		d.Picks[1].How != HowInserted || d.Picks[1].Contract == nil || d.Picks[1].Contract.Salary != 300000000 {
		t.Fatalf("the inserted pick: %+v", d.Picks[:3])
	}
	if d.OnClock.Original != "a" || d.OnClock.Slot != 1 || !d.OnClock.Deadline.Equal(start.Add(6*time.Hour)) {
		t.Fatalf("an inserted pick took the clock: %+v", d.OnClock)
	}
	again := insert
	again.Player, again.At = "0004", start.Add(2*time.Hour)
	d = actsDraft(t, again.At, insert, again)
	if d.Picks[2].Extra != 2 || d.Picks[2].Player != "0004" {
		t.Fatalf("a second insertion: %+v", d.Picks[:4])
	}
	if next, ok := d.NextPick("b"); !ok || next.Round != 1 || next.Slot != 1 {
		t.Fatalf("the franchise's own turn went to its inserted pick: %+v", next)
	}
}

func TestChangingAPickNeedsTheRookieTheCommissionerSaw(t *testing.T) {
	start := fixedStart()
	made := Act{Kind: Select, At: start.Add(time.Hour), Pick: first(), Owner: "b", Player: "0001"}
	changed := Act{Kind: Change, At: start.Add(2 * time.Hour), Pick: first(), Replaces: "0001", Player: "0002"}
	d := actsDraft(t, start.Add(2*time.Hour), made, changed)
	if d.Picks[0].Player != "0002" || !d.Picks[0].Changed || d.Picks[0].Contract.Salary != 300000000 {
		t.Fatalf("the change: %+v", d.Picks[0])
	}
	stale := changed
	stale.Replaces = "0003"
	if msg := actsError(t, start.Add(2*time.Hour), made, stale); !strings.Contains(msg, "different player") {
		t.Fatalf("a change retargeted silently: %s", msg)
	}
	unmade := changed
	unmade.Pick = second()
	actsError(t, start.Add(2*time.Hour), made, unmade)
	taken := changed
	taken.Player = "0001"
	actsError(t, start.Add(2*time.Hour), made, taken)
}

func TestTheClockCanBePausedResumedAndSet(t *testing.T) {
	start := fixedStart()
	pause := Act{Kind: Pause, At: start.Add(time.Hour)}
	d := actsDraft(t, start.Add(20*time.Hour), pause)
	if d.OnClock == nil || !d.OnClock.Paused || d.OnClock.Slot != 1 || d.OnClock.Remaining != 5*time.Hour ||
		d.Picks[0].Skipped {
		t.Fatalf("a paused clock ran on: %+v", d.OnClock)
	}
	resume := Act{Kind: Resume, At: start.Add(10 * time.Hour)}
	d = actsDraft(t, start.Add(11*time.Hour), pause, resume)
	if d.OnClock.Paused || !d.OnClock.Deadline.Equal(start.Add(15*time.Hour)) || d.OnClock.Remaining != 4*time.Hour {
		t.Fatalf("resume did not give back the time left: %+v", d.OnClock)
	}
	set := Act{Kind: SetClock, At: start.Add(2 * time.Hour), Duration: 30 * time.Minute}
	d = actsDraft(t, start.Add(2*time.Hour), set)
	if !d.OnClock.Deadline.Equal(start.Add(2*time.Hour+30*time.Minute)) || d.OnClock.Remaining != 30*time.Minute {
		t.Fatalf("set did not set the time left: %+v", d.OnClock)
	}
	d = actsDraft(t, start.Add(3*time.Hour), set)
	if !d.Picks[0].Skipped || d.OnClock.Slot != 2 {
		t.Fatalf("a shortened clock did not run out: %+v", d.OnClock)
	}
	select1 := Act{Kind: Select, At: start.Add(2 * time.Hour), Pick: first(), Owner: "b", Player: "0001"}
	if msg := actsError(t, start.Add(2*time.Hour), pause, select1); !strings.Contains(msg, "paused") {
		t.Fatalf("an owner picked on a paused clock: %s", msg)
	}
	actsError(t, start.Add(2*time.Hour), resume)
	actsError(t, start.Add(2*time.Hour), pause, pause)
	actsError(t, start.Add(2*time.Hour), Act{Kind: SetClock, At: start.Add(time.Hour)})
}

func TestARestartPutsTheDraftBackWithoutLettingTheMissedTimeSkipPicks(t *testing.T) {
	start := fixedStart()
	made := Act{Kind: Select, At: start.Add(time.Hour), Pick: first(), Owner: "b", Player: "0001"}
	// Nine hours later the commissioner restores the draft to just after the first pick. The second pick's
	// clock would have run out at 7h; it restarts at the restore instead of skipping.
	restart := Act{Kind: Restart, At: start.Add(10 * time.Hour), Since: start.Add(time.Hour)}
	d := actsDraft(t, start.Add(10*time.Hour), made, restart)
	if d.Picks[1].Skipped || d.OnClock.Slot != 2 || !d.OnClock.Deadline.Equal(start.Add(16*time.Hour)) {
		t.Fatalf("a restart let the missed time skip a pick: %+v", d.OnClock)
	}
	d = actsDraft(t, start.Add(17*time.Hour), made, restart)
	if !d.Picks[1].Skipped || d.OnClock.Round != 2 {
		t.Fatalf("the restarted clock did not run: %+v", d.OnClock)
	}
}

func TestARestartKeepsWhatTheClockSkippedBeforeItsCheckpoint(t *testing.T) {
	start := fixedStart()
	// The first pick ran out at 6h with no one recording it. A restore to the draft's start-and-a-bit at 7h
	// must keep that skip, and give only the second pick a fresh clock at the restore.
	restart := Act{Kind: Restart, At: start.Add(12 * time.Hour), Since: start.Add(7 * time.Hour)}
	d := actsDraft(t, start.Add(12*time.Hour), restart)
	if !d.Picks[0].Skipped || d.Picks[0].How != HowExpired || d.OnClock.Slot != 2 ||
		!d.OnClock.Deadline.Equal(start.Add(18*time.Hour)) {
		t.Fatalf("a restart undid a skip that happened before its checkpoint: %+v", d.OnClock)
	}
}

func TestClosingSealsTheDraftAndSkipsWhatIsLeft(t *testing.T) {
	start := fixedStart()
	made := Act{Kind: Select, At: start.Add(time.Hour), Pick: first(), Owner: "b", Player: "0001"}
	d := actsDraft(t, start.Add(2*time.Hour), made, Act{Kind: Close, At: start.Add(2 * time.Hour)})
	if !d.Exhausted || d.OnClock != nil || d.Picks[0].Player != "0001" || !d.Picks[1].Skipped ||
		d.Picks[9].Player != "" || !d.Picks[9].Skipped {
		t.Fatalf("a closed draft: exhausted=%v onClock=%v", d.Exhausted, d.OnClock)
	}
}
