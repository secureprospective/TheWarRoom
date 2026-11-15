package offers

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
)

func TestBidOnlyOnePlanOpenPerPlayer(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	first, err := a.TargetBidDraft(rawBid("5", 3))
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.TargetBidDraft(rawBid("6", 2))
	if err == nil || !strings.Contains(err.Error(), "already have an open bid plan on Smith, John (WR, DAL) (ready)") {
		t.Fatalf("second plan on the same player: %v", err)
	}
	if n := a.TargetBidPlans("0001").Count; n != 1 {
		t.Errorf("the refused plan was saved: %d plans", n)
	}
	// A different player is still free, and so is the same player once the plan is over.
	other := rawBid("5", 3)
	other.Player = "13595"
	if _, err := a.TargetBidDraft(other); err != nil {
		t.Fatalf("a different player: %v", err)
	}
	if _, err := a.TargetBidWithdraw(first.Receipt.CorrelationID); err != nil {
		t.Fatal(err)
	}
	if _, err := a.TargetBidDraft(rawBid("6", 2)); err != nil {
		t.Fatalf("after the first plan was withdrawn: %v", err)
	}
	// The pool row on him names the open plan, so the screen offers it instead of a second.
	rows := a.openBidPlansByPlayer(a.core.Ctx, "0001")
	if got := rows["13593"]; got.State != "Ready" || got.ID == "" || got.ID == first.Receipt.CorrelationID {
		t.Errorf("open plan on him = %+v", got)
	}
}

func TestBidHandedOffPlanCanBeWithdrawnAndItsFactsStayFrozen(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	raw := rawBid("5", 3)
	raw.Snipe, raw.Beat, raw.Evidence = true, "5.500", "thread"
	plan, err := a.TargetBidDraft(raw)
	if err != nil {
		t.Fatal(err)
	}
	id := plan.Receipt.CorrelationID
	handed, err := a.TargetBidHandOff(id)
	if err != nil || actionIDs(handed) != "copy,posted,unposted" {
		t.Fatalf("handed off: %v, actions %s", err, actionIDs(handed))
	}
	before, err := a.core.Moves.BidFacts(a.core.Ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	gone, err := a.TargetBidWithdraw(id)
	if err != nil || gone.Receipt.State != envelope.Withdrawn || gone.State != "Withdrawn" {
		t.Fatalf("withdrawn = %v, %v", gone.Receipt.State, err)
	}
	last := gone.Receipt.Audit[len(gone.Receipt.Audit)-1]
	if last.From != envelope.HandedOff || last.Event != envelope.Withdraw || last.Note != gone.Status ||
		!strings.Contains(gone.Status, "did not post it") || !strings.HasSuffix(gone.Status, ".") {
		t.Errorf("history entry = %+v, status %q", last, gone.Status)
	}
	if len(gone.Actions) != 0 {
		t.Errorf("a withdrawn plan still offers %s", actionIDs(gone))
	}
	after, err := a.core.Moves.BidFacts(a.core.Ctx, id)
	if err != nil || *after.Beat != *before.Beat || after.Salary != before.Salary || after.Years != before.Years ||
		after.Points != before.Points || after.Snipe != before.Snipe || after.Evidence != before.Evidence ||
		after.Closes != before.Closes {
		t.Fatalf("facts changed by the withdrawal: %+v then %+v, %v", before, after, err)
	}
	if err := a.core.Moves.SaveBid(a.core.Ctx, mustGet(t, a, id), envelope.BidFacts{Player: before.Player, Salary: 1,
		Years: 1}); err == nil {
		t.Error("the facts of a withdrawn plan were rewritten")
	}
	for name, try := range map[string]func() error{
		"withdraw again": func() error { _, err := a.TargetBidWithdraw(id); return err },
		"posted":         func() error { _, err := a.TargetBidPosted(id); return err },
		"hand off":       func() error { _, err := a.TargetBidHandOff(id); return err },
		"revise":         func() error { _, err := a.TargetBidRevise(id, raw); return err },
	} {
		if err := try(); err == nil {
			t.Errorf("%s worked on a withdrawn plan", name)
		}
	}
	if got := a.TargetBidPlans("0001").Value.Summary; got != "0 open of 1 bid plans" {
		t.Errorf("summary = %q", got)
	}
	if _, err := a.TargetBidDraft(rawBid("5", 3)); err != nil {
		t.Errorf("the player could not be planned again: %v", err)
	}
}

func TestBidPreviewSaysWhatTheClosingTimeDoes(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	none := a.TargetBidPreview(rawBid("5", 3))
	if !strings.HasPrefix(none.ClosesNote, "No closing time is set") {
		t.Errorf("without a closing time: %q", none.ClosesNote)
	}
	set := rawBid("5", 3)
	set.Closes = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	got := a.TargetBidPreview(set)
	if !strings.Contains(got.ClosesNote, "awarded off-app") || got.ClosesNote == none.ClosesNote {
		t.Errorf("with a closing time: %q", got.ClosesNote)
	}
}
