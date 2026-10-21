package composition

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

func mustGet(t *testing.T, a *App, id string) envelope.Envelope {
	t.Helper()
	e, err := a.core.Moves.Get(a.core.Ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestBidNotYetDoneAndPendingCanBeWithdrawn(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	past := time.Now().Add(-time.Hour)
	a.core.MovesNow = func() time.Time { return past }
	plan, err := offersTestAPI(a).TargetBidDraft(rawBid("5", 3))
	if err != nil {
		t.Fatal(err)
	}
	id := plan.Receipt.CorrelationID
	if _, err := offersTestAPI(a).TargetBidHandOff(id); err != nil {
		t.Fatal(err)
	}
	a.core.MovesNow = func() time.Time { return time.Now().Add(time.Minute) }
	a.core.MovesRefresh = func(context.Context, envelope.Source) error { setRosters(t, a); return nil }
	if err := a.checkMoves(a.core.Ctx); err != nil {
		t.Fatal(err)
	}
	waiting := offersTestAPI(a).TargetBidPlans("0001").Value.Plans[0]
	if waiting.Receipt.State != envelope.NotYetDone || actionIDs(waiting) != "copy,posted,unposted" {
		t.Fatalf("after a roster check: %v %s", waiting.Receipt.State, actionIDs(waiting))
	}
	if gone, err := offersTestAPI(a).TargetBidWithdraw(id); err != nil || gone.Receipt.State != envelope.Withdrawn {
		t.Fatalf("withdraw from not yet done: %v, %v", gone.Receipt.State, err)
	}

	again := rawBid("5", 3)
	posted, err := offersTestAPI(a).TargetBidDraft(again)
	if err != nil {
		t.Fatal(err)
	}
	pid := posted.Receipt.CorrelationID
	if _, err := offersTestAPI(a).TargetBidHandOff(pid); err != nil {
		t.Fatal(err)
	}
	pending, err := offersTestAPI(a).TargetBidPosted(pid)
	if err != nil || pending.Receipt.State != envelope.BidPending || actionIDs(pending) != "copy,withdraw" {
		t.Fatalf("posted = %v %s, %v", pending.Receipt.State, actionIDs(pending), err)
	}
	if gone, err := offersTestAPI(a).TargetBidWithdraw(pid); err != nil || gone.Receipt.State != envelope.Withdrawn || !strings.Contains(gone.Status, "cannot remove a ProBoards post") {
		t.Fatalf("withdraw pending local bid: %+v %v", gone, err)
	}
	if now := mustGet(t, a, pid); now.State() != envelope.Withdrawn {
		t.Errorf("pending local plan did not end: %v", now.State())
	}
}

func TestPoolRowOffersTheOpenPlan(t *testing.T) {
	t.Parallel()
	a, _ := poolApp(t)
	if err := offersTestAPI(a).TargetPoolRefresh(true); err != nil {
		t.Fatal(err)
	}
	plan, err := offersTestAPI(a).TargetBidDraft(rawBid("5", 3))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range offersTestAPI(a).TargetPool(RawPoolScope{FranchiseID: "0001", Sort: "name"}).Value.Rows {
		if row.Name == "Smith, John" && (row.PlanID != plan.Receipt.CorrelationID || row.Plan != "Ready") {
			t.Errorf("Smith's row = %+v", row)
		}
		if row.Name != "Smith, John" && row.PlanID != "" {
			t.Errorf("%s shows a plan %s", row.Name, row.PlanID)
		}
	}
}
