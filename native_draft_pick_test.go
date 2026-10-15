package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// seated gives each franchise an owner whose session can act for it alone.
func (f *draftFixture) seated(t *testing.T, teams ...int) {
	t.Helper()
	for _, n := range teams {
		f.seat(t, draftTeam(n), fmt.Sprintf("owner%d", n))
	}
}

func (f *draftFixture) asOwner(n int) { f.actAs(fmt.Sprintf("owner%d", n)) }

func (f *draftFixture) player(t *testing.T, id string) commissioner.NativePlayer {
	t.Helper()
	assets, err := f.app.TargetNativeAssets(draftLeague)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range assets.Snapshot.Players {
		if p.ID == id {
			return p
		}
	}
	return commissioner.NativePlayer{}
}

func (f *draftFixture) committed(t *testing.T, franchise string) int64 {
	t.Helper()
	check, err := f.app.TargetNativeCapCheck(draftLeague, "America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range check.Franchises {
		if row.Franchise == franchise && row.CommittedCents != nil {
			return int64(*row.CommittedCents)
		}
	}
	t.Fatalf("no committed salary for %s", franchise)
	return 0
}

func TestOwnerPickGetsTheSlotContractAndMovesTheClock(t *testing.T) {
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.seated(t, 1, 2, 3)
	f.now = draftStartAt().Add(time.Hour)
	capBefore := f.committed(t, "franchise.03")
	f.asOwner(3)
	f.pick(t, "franchise.03", "5001")
	for pass := 0; pass < 2; pass++ {
		got := f.player(t, "5001")
		if got.Franchise != "franchise.03" || got.Salary != 300000000 || got.ExpirationYear != 2031 ||
			got.ContractStatus != "UFA" || !strings.Contains(got.ContractInfo, "round 1 pick 1") {
			t.Fatalf("the rookie contract: %+v", got)
		}
		if spent := f.committed(t, "franchise.03") - capBefore; spent != 300000000 {
			t.Fatalf("the cap moved by %d", spent)
		}
		reading := f.read(t, "franchise.01")
		made := reading.Draft.Picks[0]
		if made.Player != "5001" || made.Contract == nil || made.Contract.Years != 5 || made.MadeAt == nil ||
			!made.MadeAt.Equal(f.now) {
			t.Fatalf("the pick made: %+v", made)
		}
		if onClock := reading.Draft.OnClock; onClock.Original != "franchise.02" || onClock.Owner != "franchise.01" ||
			!onClock.OpenedAt.Equal(f.now) || !onClock.Deadline.Equal(f.now.Add(6*time.Hour)) {
			t.Fatalf("the next pick on the clock: %+v", onClock)
		}
		if next := reading.Next; next == nil || next.Round != 1 || next.Slot != 2 || next.Original != "franchise.02" {
			t.Fatalf("the viewer's own next pick: %+v", reading.Next)
		}
		f.reopen(t)
	}
	f.asOwner(2)
	if _, err := f.tryPick("franchise.02", "5002", "traded", f.revision(t)); err == nil ||
		!strings.Contains(err.Error(), "not your turn") {
		t.Fatalf("the original team picked a pick it traded away: %v", err)
	}
	f.asOwner(1)
	f.pick(t, "franchise.01", "5002")
	if got := f.player(t, "5002"); got.Franchise != "franchise.01" || got.Salary != 300000000 {
		t.Fatalf("the traded pick's contract: %+v", got)
	}
	if on := f.read(t, "franchise.01").Draft.OnClock; on.Original != "franchise.01" || on.Slot != 3 {
		t.Fatalf("the clock after the traded pick: %+v", on)
	}
}

func TestExpiredClockSkipsAndLaterRoundPaysItsOwnPrice(t *testing.T) {
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.seated(t, 1, 3)
	f.now = draftStartAt().Add(6 * time.Hour)
	reading := f.read(t, "franchise.03")
	if !reading.Draft.Picks[0].Skipped || reading.Draft.Picks[0].Player != "" {
		t.Fatalf("a clock that ran out at its deadline did not skip: %+v", reading.Draft.Picks[0])
	}
	if on := reading.Draft.OnClock; on.Slot != 2 || !on.OpenedAt.Equal(f.now) {
		t.Fatalf("the next clock did not open at the deadline: %+v", on)
	}
	f.asOwner(3)
	if _, err := f.tryPick("franchise.03", "5001", "late", f.revision(t)); err == nil ||
		!strings.Contains(err.Error(), "not your turn") {
		t.Fatalf("a skipped pick was made late: %v", err)
	}
	f.now = draftStartAt().Add(32*6*time.Hour + 30*time.Minute)
	reading = f.read(t, "franchise.03")
	for i := 0; i < 32; i++ {
		if !reading.Draft.Picks[i].Skipped || !reading.Draft.Picks[i].MadeAt.Equal(draftStartAt().Add(time.Duration(i+1)*6*time.Hour)) {
			t.Fatalf("round one pick %d: %+v", i+1, reading.Draft.Picks[i])
		}
	}
	if on := reading.Draft.OnClock; on.Round != 2 || on.Slot != 1 || on.Owner != "franchise.03" {
		t.Fatalf("round two did not open: %+v", on)
	}
	f.pick(t, "franchise.03", "5001")
	got := f.player(t, "5001")
	if got.Salary != 142000000 || got.ExpirationYear != 2029 || got.ContractStatus != "RFA" {
		t.Fatalf("a round two pick paid another round's price: %+v", got)
	}
}

func TestPickRefusalsLeaveTheDraftUntouched(t *testing.T) {
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.seated(t, 1, 3)
	revision := f.revision(t)
	try := func(who int, franchise, player, want string) {
		t.Helper()
		if who == 0 {
			f.actAs("creator")
		} else {
			f.asOwner(who)
		}
		_, err := f.tryPick(franchise, player, "refused."+player+franchise, revision)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s for %s as owner %d: %v, want %q", player, franchise, who, err, want)
		}
		if f.revision(t) != revision {
			t.Fatalf("a refused pick changed the league")
		}
	}
	f.now = draftStartAt().Add(-time.Hour)
	try(3, "franchise.03", "5001", "This action opens with the rookie_draft window on Jul 14, 2027 at 9:00 AM PDT.")
	f.now = draftStartAt().Add(time.Hour)
	try(0, "franchise.03", "5001", "Only the owner of this franchise may make its picks")
	try(1, "franchise.03", "5001", "Only the owner of this franchise may make its picks")
	try(1, "franchise.01", "5001", "not your turn")
	try(3, "franchise.03", "1000", "Choose a rookie from this season's player pool")
	try(3, "franchise.03", "9999", "Choose a rookie from this season's player pool")
	try(3, "franchise.03", "12", "Choose a rookie from this season's player pool")
	f.asOwner(3)
	f.pick(t, "franchise.03", "5001")
	revision = f.revision(t)
	try(1, "franchise.01", "5001", "already belongs to a team")
}

func TestPickIsRefusedWhenItBreaksRosterOrCap(t *testing.T) {
	f := draftApp(t)
	f.scheduled(t, 4, 3, 2)
	f.seated(t, 2, 3, 1)
	f.now = draftStartAt().Add(time.Hour)
	f.asOwner(2)
	_, err := f.tryPick("franchise.02", "5001", "over", f.revision(t))
	if err == nil || !strings.Contains(err.Error(), "break your roster limits") || !strings.Contains(err.Error(), "over") {
		t.Fatalf("a pick onto an 82 player roster: %v", err)
	}
	f.skipTo(t, 2)
	f.actAs("creator")
	f.setRule(t, "cap.amount", `100000000`)
	f.asOwner(3)
	_, err = f.tryPick("franchise.03", "5001", "cap", f.revision(t))
	if err == nil || !strings.Contains(err.Error(), "salary cap") {
		t.Fatalf("a $3 million rookie under a $1 million cap: %v", err)
	}
}

// skipTo moves the clock to the opening of the nth pick of the draft, after the picks before it have expired.
func (f *draftFixture) skipTo(t *testing.T, n int) {
	t.Helper()
	f.now = draftStartAt().Add(time.Duration(n-1)*6*time.Hour + time.Minute)
}

func TestPickRetryLandsOnceAndTwoPicksCannotBothLand(t *testing.T) {
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.seated(t, 3)
	f.now = draftStartAt().Add(time.Hour)
	f.asOwner(3)
	rev := f.revision(t)
	first, err := f.tryPick("franchise.03", "5001", "once", rev)
	if err != nil {
		t.Fatal(err)
	}
	again, err := f.tryPick("franchise.03", "5001", "once", rev)
	if err != nil || again != first || f.revision(t) != rev+1 {
		t.Fatalf("a retried pick was recorded again: %v %v", again, err)
	}
	if _, err := f.tryPick("franchise.03", "5002", "once", rev); err == nil ||
		!strings.Contains(err.Error(), "already used for a different change") {
		t.Fatalf("a request key was reused for another pick: %v", err)
	}
	if _, err := f.tryPick("franchise.03", "5003", "stale", rev); err == nil ||
		!strings.Contains(err.Error(), "reread") {
		t.Fatalf("a stale pick was recorded: %v", err)
	}
	// Two different picks for the same slot, sent together: exactly one lands.
	f.skipTo(t, 33)
	rev, wins := f.revision(t), 0
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i, player := range []string{"5003", "5004"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := f.tryPick("franchise.03", player, fmt.Sprintf("race.%d", i), rev); err == nil {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("%d of two simultaneous picks landed", wins)
	}
}

func TestDraftPickCannotBeSentThroughTheCommissionerDoor(t *testing.T) {
	f := draftApp(t)
	f.scheduled(t, 3, 2, 1)
	f.now = draftStartAt().Add(time.Hour)
	_, err := f.tryCommand(commissioner.DraftPick, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{
		Season: draftSeason, Pick: &commissioner.RawDraftSelection{Franchise: "franchise.03", Player: "5001",
			EntrySeason: draftSeason, Round: 1, Slot: 1, Original: "franchise.03",
			Terms: rookiedraft.Terms{Salary: 1, Years: 1}}}}, "")
	if err == nil || !strings.Contains(err.Error(), "franchise on the clock") {
		t.Fatalf("a pick with a contract of the sender's choosing was recorded: %v", err)
	}
}

func TestDraftChangesNeedRolesAndTheRunningDraftNeedsAReason(t *testing.T) {
	f := draftApp(t)
	f.seated(t, 3)
	f.asOwner(3)
	_, err := f.tryCommand(commissioner.DraftOrderSet, commissioner.RawArgs{Draft: &commissioner.RawDraftArgs{
		Season: draftSeason, Order: draftOrder(3)}}, "")
	if err == nil || !strings.Contains(err.Error(), "role") {
		t.Fatalf("an owner set the draft order: %v", err)
	}
	f.actAs("creator")
	orderEntry, startEntry := f.scheduled(t, 3, 2, 1)
	order := func(d commissioner.RawDraftArgs, reason string) error {
		_, err := f.tryCommand(commissioner.DraftOrderSet, commissioner.RawArgs{Draft: &d}, reason)
		return err
	}
	next := commissioner.RawDraftArgs{Season: draftSeason, Order: draftOrder(5)}
	if err := order(commissioner.RawDraftArgs{Season: draftSeason, Order: draftOrder(5), Running: true},
		"early"); err == nil || !strings.Contains(err.Error(), "not running") {
		t.Fatalf("a running claim before the start: %v", err)
	}
	f.now = draftStartAt().Add(time.Hour)
	if err := order(next, ""); err == nil || !strings.Contains(err.Error(), "clock is running") {
		t.Fatalf("an order change while the clock runs, unmarked: %v", err)
	}
	next.Running = true
	if err := order(next, ""); err == nil || !strings.Contains(err.Error(), "give a reason") {
		t.Fatalf("an order change while the clock runs, with no reason: %v", err)
	}
	f.seated(t, 3)
	f.asOwner(3)
	f.pick(t, "franchise.03", "5001")
	f.actAs("creator")
	if err := order(next, "Fix it"); err == nil || !strings.Contains(err.Error(), "contradicts a pick already made") {
		t.Fatalf("an order change that leaves a made pick without its place: %v", err)
	}
	onlyRoundOne := json.RawMessage(`{"1":[{"Slots":{"Min":1,"Max":32},"Salary":1}]}`)
	if _, err := f.tryCommand(commissioner.RulesSet, commissioner.RawArgs{Field: "draft.pricing",
		Value: onlyRoundOne}, "Our ruling"); err == nil || !strings.Contains(err.Error(), "contradicts a pick") {
		t.Fatalf("a price list that leaves later rounds unpriced after a pick was made: %v", err)
	}
	for _, entry := range []string{startEntry.Entry, orderEntry.Entry} {
		_, err := f.tryCommand(commissioner.Revert, commissioner.RawArgs{Entry: entry}, "Undo it")
		if err == nil || !strings.Contains(err.Error(), "contradicts a pick already made") {
			t.Fatalf("reversing a setting a made pick depends on: %v", err)
		}
	}
}

func TestDraftPickIsRefusedOnceTheCalendarClosesTheWindow(t *testing.T) {
	f := draftApp(t)
	f.command(t, commissioner.WindowSet, commissioner.RawArgs{Window: &commissioner.RawWindowArgs{ID: "rookie_draft",
		Start: date(7, 14), End: date(7, 20), Opens: []leaguerules.Action{"draft_pick"}}}, "")
	f.scheduled(t, 3, 2, 1)
	f.seated(t, 3, 29)
	f.now = draftStartAt().Add(time.Hour)
	f.asOwner(3)
	f.pick(t, "franchise.03", "5001")
	f.now = time.Date(2027, 7, 21, 12, 0, 0, 0, time.UTC)
	on := f.read(t, "franchise.01").Draft.OnClock
	if on == nil || on.Owner != "franchise.29" {
		t.Fatalf("expected franchise.29 on the clock: %+v", on)
	}
	f.asOwner(29)
	_, err := f.tryPick("franchise.29", "5002", "late", f.revision(t))
	if err == nil || !strings.Contains(err.Error(), "No league window opens this action now") {
		t.Fatalf("a pick after the calendar closed the draft window: %v", err)
	}
	if got := f.player(t, "5002"); got.ID != "" {
		t.Fatalf("the refused pick still gave a contract: %+v", got)
	}
}

func TestDraftReadingSurvivesReopeningAndFixesTheWire(t *testing.T) {
	f := draftApp(t)
	waiting := f.read(t, "franchise.01")
	f.scheduled(t, 3, 2, 1)
	f.seated(t, 1, 3)
	f.now = draftStartAt().Add(time.Hour)
	f.asOwner(3)
	f.pick(t, "franchise.03", "5001")
	f.now = draftStartAt().Add(7*time.Hour + 30*time.Minute)
	running := f.read(t, "franchise.01")
	before, err := json.MarshalIndent(running, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	f.reopen(t)
	after, err := json.MarshalIndent(f.read(t, "franchise.01"), "", "  ")
	if err != nil || string(before) != string(after) {
		t.Fatalf("replaying the ledger changed the reading: %v", err)
	}
	if running.Draft.Picks[1].Skipped != true || running.Next == nil || running.Next.Slot != 3 {
		t.Fatalf("the skip and the viewer's next pick: %+v", running.Draft.Picks[1])
	}
	ledger, err := f.app.TargetCommissionerLedger(draftLeague)
	if err != nil {
		t.Fatal(err)
	}
	var said []string
	for _, e := range ledger.Entries {
		said = append(said, e.What)
		if strings.HasPrefix(e.What, "Picked player") && e.Undo != "" {
			t.Fatalf("a made pick offers an undo before draft reverts are built: %+v", e)
		}
	}
	for _, want := range []string{"Picked player 5001 for round 1, pick 1 of the 2027 rookie draft",
		"Set the 2027 rookie draft to start at 2027-07-14T16:00:00Z",
		"Set the 2027 rookie draft order for every round"} {
		if !slices.Contains(said, want) {
			t.Fatalf("the ledger does not say %q: %q", want, said)
		}
	}
	body, err := json.MarshalIndent(struct {
		Waiting commissioner.NativeDraft `json:"waiting"`
		Running commissioner.NativeDraft `json:"running"`
	}{waiting, running}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "internal/commissioner/testdata/native-draft.wire.json", body, "native draft")
}
