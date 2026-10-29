package offers

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/offers/offerstest"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
)

func TestBidPreviewShowsGoFigures(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	p := a.TargetBidPreview(rawBid("5", 3))
	if p.Problem != "" || p.Points != "7.000" || p.Salary.Cents != 500_000_000 || p.Blocked ||
		p.Player != "Smith, John (WR, DAL)" || p.Snapped != "" || len(p.Rules) != 11 {
		t.Fatalf("preview = %+v", p)
	}
	if want := "Player: Smith, John (WR, DAL)\nContract: 3 years at $5,000,000 a year, flat\n" +
		"Cap hit: $5,000,000 a year\nPoints: 7.000"; p.Post != want {
		t.Errorf("post text:\n%s", p.Post)
	}
	snapped := a.TargetBidPreview(rawBid("5.004", 1))
	if snapped.Salary.Cents != 500_000_000 || !strings.Contains(snapped.Snapped, "Entered $5,004,000") ||
		snapped.Points != "5.000" {
		t.Errorf("snapped preview = %+v", snapped)
	}
	long := a.TargetBidPreview(rawBid("5", 5))
	if !long.Blocked || long.Points != "Not scored" || !strings.Contains(long.Note, "A bid runs 1 to 4 years, not 5") {
		t.Errorf("5-year preview = %+v", long)
	}
	for _, bad := range []RawBidInput{rawBid("", 1), rawBid("abc", 1), {FranchiseID: "1", Player: "13593", Salary: "5", Years: 1},
		{FranchiseID: "0001", Player: "99999", Salary: "5", Years: 1},
		{FranchiseID: "0001", Player: "13593", Salary: "5", Years: 1, Beat: "6.5551"},
		{FranchiseID: "0001", Player: "13593", Salary: "5", Years: 1, Closes: "tomorrow"}} {
		if got := a.TargetBidPreview(bad); got.Problem == "" || got.Rules == nil {
			t.Errorf("bad input %+v gave %+v", bad, got)
		}
	}
}

func TestBidPreviewMinimumAndCapRoom(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	// 13594 has no draft year: the rookie minimum applies, and the preview says why.
	low := a.TargetBidPreview(RawBidInput{FranchiseID: "0001", Player: "13594", Salary: "0.32", Years: 1})
	if !low.Blocked || !strings.Contains(low.Note, "Below the $330,000 minimum salary (no draft year on file") {
		t.Errorf("rookie minimum: %+v", low)
	}
	// 13595 has 16 years: $630,000.
	vet := a.TargetBidPreview(RawBidInput{FranchiseID: "0001", Player: "13595", Salary: "0.6", Years: 1})
	if !vet.Blocked || !strings.Contains(vet.Note, "Below the $630,000 minimum") {
		t.Errorf("veteran minimum: %+v", vet)
	}
	room := a.TargetBidPreview(rawBid("5", 2))
	last := room.Rules[len(room.Rules)-1]
	if last.Rule != "cap-room" || !strings.Contains(last.Reason, "$5,000,000 a year against $105,000,000 of room leaves $100,000,000") {
		t.Errorf("cap room = %+v", last)
	}
	over := a.TargetBidPreview(rawBid("110", 3))
	if !over.Blocked || !strings.Contains(over.Note, "room") {
		t.Errorf("cap room did not block the plan: %s", over.Note)
	}
}

func TestBidBlockedByPlayoffStatusKeepsTheActOff(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	a.core.Week = leagueweek.Week{Number: 13}
	plan, err := a.TargetBidDraft(rawBid("5", 1))
	if err != nil || plan.Receipt.State != envelope.Blocked {
		t.Fatalf("draft = %v, %v", plan.Receipt.State, err)
	}
	if !strings.Contains(plan.Status, "this app cannot tell whether yours is one") {
		t.Errorf("status = %q", plan.Status)
	}
	if got := ruleWords(plan, "late-season"); !strings.HasPrefix(got, "Can't tell:") {
		t.Errorf("late-season = %q", got)
	}
	var handOff BidAction
	for _, action := range plan.Actions {
		if action.ID == "hand-off" {
			handOff = action
		}
	}
	if !handOff.Disabled || !strings.Contains(handOff.Reason, "cannot tell whether yours is one") {
		t.Errorf("hand-off action = %+v", handOff)
	}
	if _, err := a.TargetBidHandOff(plan.Receipt.CorrelationID); err == nil {
		t.Error("a blocked plan was handed off")
	}
}

func TestBidWithoutAHeldScheduleCannotCheckTheWeekLimits(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	a.core.WeekFetchedAt = time.Time{}
	plan, err := a.TargetBidDraft(rawBid("5", 1))
	if err != nil || plan.Receipt.State != envelope.Blocked || !strings.Contains(plan.Status, "holds no NFL week") {
		t.Fatalf("plan = %v %q, %v", plan.Receipt.State, plan.Status, err)
	}
}

func TestBidActiveLimitOfThirty(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	var extra []players.RawPlayer // one player per plan: a second open plan on a player is refused
	for i := range 32 {
		extra = append(extra, players.RawPlayer{ID: fmt.Sprint(20000 + i), Name: fmt.Sprintf("Player, N%d", i),
			Position: "WR", Team: "DAL", DraftYear: "2015"})
	}
	a.core.Lookup = offerstest.Lookup(t, extra...)
	on := func(i int) RawBidInput { r := rawBid("1", 1); r.Player = fmt.Sprint(20000 + i); return r }
	var first string
	for i := range 30 {
		plan, err := a.TargetBidDraft(on(i))
		if err != nil || plan.Receipt.State != envelope.Ready {
			t.Fatalf("plan %d = %v, %v", i, plan.Receipt.State, err)
		}
		if i == 0 {
			first = plan.Receipt.CorrelationID
		}
	}
	over, err := a.TargetBidDraft(on(30))
	if err != nil || over.Receipt.State != envelope.Blocked ||
		!strings.Contains(over.Status, "You already have 30 active bid plans") {
		t.Fatalf("31st plan = %v %q, %v", over.Receipt.State, over.Status, err)
	}
	if _, err := a.TargetBidWithdraw(first); err != nil {
		t.Fatal(err)
	}
	again, err := a.TargetBidDraft(on(31))
	if err != nil || again.Receipt.State != envelope.Ready {
		t.Fatalf("plan after a withdrawal = %v, %v", again.Receipt.State, err)
	}
}

func TestBidReviseAndWithdraw(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	plan, err := a.TargetBidDraft(rawBid("5", 3))
	if err != nil {
		t.Fatal(err)
	}
	id := plan.Receipt.CorrelationID
	revised, err := a.TargetBidRevise(id, rawBid("6", 2))
	if err != nil || revised.Receipt.State != envelope.Ready || revised.Points != "7.200" {
		t.Fatalf("revised = %+v, %v", revised.Receipt.State, err)
	}
	var states []string
	for _, entry := range revised.Receipt.Audit {
		states = append(states, string(entry.To))
	}
	if strings.Join(states, ",") != "ready,stale,draft,ready" {
		t.Errorf("revision route = %v", states)
	}
	if _, err := a.TargetBidRevise(id, RawBidInput{FranchiseID: "0001", Player: "13594", Salary: "6", Years: 2}); err == nil {
		t.Error("a plan changed its player")
	}
	broken, err := a.TargetBidRevise(id, rawBid("6", 5))
	if err != nil || broken.Receipt.State != envelope.Blocked {
		t.Fatalf("revised into a block = %v, %v", broken.Receipt.State, err)
	}
	fixed, err := a.TargetBidRevise(id, rawBid("6", 4))
	if err != nil || fixed.Receipt.State != envelope.Ready || fixed.Facts.Years != 4 {
		t.Fatalf("fixed = %v, %v", fixed.Receipt.State, err)
	}
	gone, err := a.TargetBidWithdraw(id)
	if err != nil || gone.Receipt.State != envelope.Withdrawn || gone.State != "Withdrawn" ||
		gone.Status != "Removed by you before it was handed off." || len(gone.Actions) != 0 {
		t.Fatalf("withdrawn = %+v, %v", gone, err)
	}
	if _, err := a.TargetBidRevise(id, rawBid("6", 4)); err == nil {
		t.Error("a withdrawn plan was revised")
	}
}

func TestBidFactsFreezeOnceHandedOff(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	plan, err := a.TargetBidDraft(rawBid("5", 3))
	if err != nil {
		t.Fatal(err)
	}
	id := plan.Receipt.CorrelationID
	handed, err := a.TargetBidHandOff(id)
	if err != nil || handed.Receipt.State != envelope.HandedOff {
		t.Fatalf("hand-off = %v, %v", handed.Receipt.State, err)
	}
	if !strings.Contains(handed.Status, "nothing was posted for you") || len(handed.Rules) != 0 {
		t.Errorf("handed off: %q rules %d", handed.Status, len(handed.Rules))
	}
	for _, try := range []func() error{
		func() error { _, err := a.TargetBidRevise(id, rawBid("9", 3)); return err },
		func() error { _, err := a.TargetBidHandOff(id); return err },
	} {
		if err := try(); err == nil {
			t.Error("a handed-off plan was changed")
		}
	}
	posted, err := a.TargetBidPosted(id)
	if err != nil || posted.Receipt.State != envelope.BidPending || !strings.Contains(posted.Status, "owner reports") {
		t.Fatalf("posted = %v, %v", posted.Receipt.State, err)
	}
	select {
	case <-a.core.MovesWake:
	default:
		t.Error("the watcher was not woken")
	}
}

// landedBid drafts, hands off and posts a 3-year $5M bid on 13593, then lets the watcher look at
// a roster refresh that sets the mirror to extra.
func TestBlockedPlanThatWouldPassNowStillExplainsItself(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	a.core.Week = leagueweek.Week{Number: 13}
	plan, err := a.TargetBidDraft(rawBid("5", 1))
	if err != nil || plan.Receipt.State != envelope.Blocked {
		t.Fatalf("plan = %v, %v", plan.Receipt.State, err)
	}
	a.core.Week = leagueweek.Week{Number: 5}
	board := a.TargetBidPlans("0001").Value.Plans[0]
	var handOff BidAction
	for _, action := range board.Actions {
		if action.ID == "hand-off" {
			handOff = action
		}
	}
	if !handOff.Disabled || !strings.Contains(handOff.Reason, "cannot tell whether yours is one") ||
		!strings.HasSuffix(handOff.Reason, "change the figures to check it again.") {
		t.Errorf("hand-off = %+v", handOff)
	}
}

func TestSimultaneousBidDraftsLandOnce(t *testing.T) {
	a := bidTestApp(t)
	results := make(chan error, 2)
	for range 2 {
		go func() { _, err := a.TargetBidDraft(rawBid("5", 3)); results <- err }()
	}
	successes := 0
	for range 2 {
		if err := <-results; err == nil {
			successes++
		}
	}
	if successes != 1 || a.TargetBidPlans("0001").Count != 1 {
		t.Fatal("duplicate concurrent draft", successes)
	}
}
