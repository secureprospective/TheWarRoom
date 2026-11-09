package offers

import (
	"reflect"
	"strings"
	"testing"
)

func TestMFLInvalidPlanActionsPreservePlans(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	saved, err := a.TargetBidDraft(rawBid("5", 1))
	if err != nil {
		t.Fatal(err)
	}
	before := a.TargetBidPlans("0001")
	for _, id := range []string{"", " ", strings.Repeat("x", 129), "missing-plan"} {
		for _, run := range []func(string) (BidPlan, error){a.TargetBidHandOff, a.TargetBidPosted, a.TargetBidWithdraw} {
			if _, err := run(id); err == nil {
				t.Fatalf("accepted invalid plan %q", id)
			}
		}
		if _, err := a.TargetBidRevise(id, rawBid("6", 1)); err == nil {
			t.Fatal("revised absent plan")
		}
	}
	after := a.TargetBidPlans("0001")
	if len(after.Value.Plans) != 1 || after.Value.Plans[0].Receipt.CorrelationID != saved.Receipt.CorrelationID ||
		!reflect.DeepEqual(before.Value, after.Value) {
		t.Fatal("refusal changed local state")
	}
}
