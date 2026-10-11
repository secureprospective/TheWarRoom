package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
)

func actionIDs(plan BidPlan) string {
	ids := []string{}
	for _, a := range plan.Actions {
		ids = append(ids, a.ID)
	}
	return strings.Join(ids, ",")
}

func TestBidOnlyOnePlanOpenPerPlayer(t *testing.T) {
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
	rows := a.openBidPlansByPlayer(a.ctx, "0001")
	if got := rows["13593"]; got.State != "Ready" || got.ID == "" || got.ID == first.Receipt.CorrelationID {
		t.Errorf("open plan on him = %+v", got)
	}
}

func TestBidBlockedPlanStillCountsAsTheOpenPlan(t *testing.T) {
	a := bidTestApp(t)
	if plan, err := a.TargetBidDraft(rawBid("5", 5)); err != nil || plan.Receipt.State != envelope.Blocked {
		t.Fatalf("blocked plan = %v, %v", plan.Receipt.State, err)
	}
	if _, err := a.TargetBidDraft(rawBid("5", 3)); err == nil || !strings.Contains(err.Error(), "(blocked)") {
		t.Errorf("a blocked plan did not hold the player: %v", err)
	}
}

func TestBidHandedOffPlanCanBeWithdrawnAndItsFactsStayFrozen(t *testing.T) {
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
	before, err := a.moves.BidFacts(a.ctx, id)
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
	after, err := a.moves.BidFacts(a.ctx, id)
	if err != nil || *after.Beat != *before.Beat || after.Salary != before.Salary || after.Years != before.Years ||
		after.Points != before.Points || after.Snipe != before.Snipe || after.Evidence != before.Evidence ||
		after.Closes != before.Closes {
		t.Fatalf("facts changed by the withdrawal: %+v then %+v, %v", before, after, err)
	}
	if err := a.moves.SaveBid(a.ctx, mustGet(t, a, id), envelope.BidFacts{Player: before.Player, Salary: 1,
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

func mustGet(t *testing.T, a *App, id string) envelope.Envelope {
	t.Helper()
	e, err := a.moves.Get(a.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestBidNotYetDoneCanBeWithdrawnButAPostedBidCannot(t *testing.T) {
	a := bidTestApp(t)
	past := time.Now().Add(-time.Hour)
	a.movesNow = func() time.Time { return past }
	plan, err := a.TargetBidDraft(rawBid("5", 3))
	if err != nil {
		t.Fatal(err)
	}
	id := plan.Receipt.CorrelationID
	if _, err := a.TargetBidHandOff(id); err != nil {
		t.Fatal(err)
	}
	a.movesNow = func() time.Time { return time.Now().Add(time.Minute) }
	a.movesRefresh = func(context.Context, envelope.Source) error { setRosters(t, a); return nil }
	if err := a.checkMoves(a.ctx); err != nil {
		t.Fatal(err)
	}
	waiting := a.TargetBidPlans("0001").Value.Plans[0]
	if waiting.Receipt.State != envelope.NotYetDone || actionIDs(waiting) != "copy,posted,unposted" {
		t.Fatalf("after a roster check: %v %s", waiting.Receipt.State, actionIDs(waiting))
	}
	if gone, err := a.TargetBidWithdraw(id); err != nil || gone.Receipt.State != envelope.Withdrawn {
		t.Fatalf("withdraw from not yet done: %v, %v", gone.Receipt.State, err)
	}

	again := rawBid("5", 3)
	posted, err := a.TargetBidDraft(again)
	if err != nil {
		t.Fatal(err)
	}
	pid := posted.Receipt.CorrelationID
	if _, err := a.TargetBidHandOff(pid); err != nil {
		t.Fatal(err)
	}
	pending, err := a.TargetBidPosted(pid)
	if err != nil || pending.Receipt.State != envelope.BidPending || actionIDs(pending) != "copy" {
		t.Fatalf("posted = %v %s, %v", pending.Receipt.State, actionIDs(pending), err)
	}
	if _, err := a.TargetBidWithdraw(pid); err == nil ||
		!strings.Contains(err.Error(), "a posted bid cannot be withdrawn here") ||
		!strings.Contains(err.Error(), "ProBoards") {
		t.Errorf("withdrawing a posted bid: %v", err)
	}
	if now := mustGet(t, a, pid); now.State() != envelope.BidPending {
		t.Errorf("the refused withdrawal changed the plan to %v", now.State())
	}
}

func TestBidPlanWordsForTheScreens(t *testing.T) {
	a := bidTestApp(t)
	plan, err := a.TargetBidDraft(rawBid("5", 3))
	if err != nil {
		t.Fatal(err)
	}
	if plan.RuleSummary != "10 rules pass" || len(plan.Rules) != 11 {
		t.Errorf("summary %q over %d rules", plan.RuleSummary, len(plan.Rules))
	}
	if got := ruleWords(plan, "flat-salary"); got != "OK: $5,000,000 a year, the same in each of the 3 years" {
		t.Errorf("flat salary = %q", got)
	}
	if got := ruleWords(plan, "tier-years"); got != "OK: Up to $12,000,000 needs at least 1 year" {
		t.Errorf("tier = %q", got)
	}
	one := rawBid("5", 1)
	one.Player = "13595"
	short, err := a.TargetBidDraft(one)
	if err != nil {
		t.Fatal(err)
	}
	if got := ruleWords(short, "flat-salary"); got != "OK: $5,000,000 a year for its 1 year" {
		t.Errorf("one-year flat salary = %q", got)
	}
	for _, p := range []BidPlan{plan, short} {
		if !strings.HasSuffix(p.Status, ".") {
			t.Errorf("status %q does not end in a period", p.Status)
		}
		for _, entry := range p.Receipt.Audit {
			if !strings.HasSuffix(entry.Note, ".") {
				t.Errorf("history note %q does not end in a period", entry.Note)
			}
		}
	}
	blocked, err := a.TargetBidDraft(RawBidInput{FranchiseID: "0001", Player: "13594", Salary: "0.32", Years: 1})
	if err != nil || !strings.HasSuffix(blocked.Status, ".") || blocked.RuleSummary == "" {
		t.Fatalf("blocked = %q %q, %v", blocked.Status, blocked.RuleSummary, err)
	}
	for _, action := range blocked.Actions {
		if action.Reason != "" && !strings.HasSuffix(action.Reason, ".") {
			t.Errorf("reason %q does not end in a period", action.Reason)
		}
	}
}

func TestBidPreviewSaysWhatTheClosingTimeDoes(t *testing.T) {
	a := bidTestApp(t)
	none := a.TargetBidPreview(rawBid("5", 3))
	if !strings.HasPrefix(none.ClosesNote, "No closing time is set") || none.RuleSummary != "10 rules pass" {
		t.Errorf("without a closing time: %q, %q", none.ClosesNote, none.RuleSummary)
	}
	set := rawBid("5", 3)
	set.Closes = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	got := a.TargetBidPreview(set)
	if !strings.Contains(got.ClosesNote, "awarded off-app") || got.ClosesNote == none.ClosesNote {
		t.Errorf("with a closing time: %q", got.ClosesNote)
	}
}

func TestPoolRowOffersTheOpenPlan(t *testing.T) {
	a, _ := poolApp(t)
	if err := a.TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	plan, err := a.TargetBidDraft(rawBid("5", 3))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range a.TargetPool(RawPoolScope{FranchiseID: "0001", Sort: "name"}).Value.Rows {
		if row.Name == "Smith, John" && (row.PlanID != plan.Receipt.CorrelationID || row.Plan != "Ready") {
			t.Errorf("Smith's row = %+v", row)
		}
		if row.Name != "Smith, John" && row.PlanID != "" {
			t.Errorf("%s shows a plan %s", row.Name, row.PlanID)
		}
	}
}
