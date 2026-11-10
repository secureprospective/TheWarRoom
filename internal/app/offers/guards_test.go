package offers

import (
	"reflect"
	"strings"
	"testing"
)

func TestMFLPreviewUsesNamesAndNeverSaves(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	before := a.TargetBidPlans("0001")
	preview := a.TargetBidPreview(rawBid("5", 1))
	if preview.Problem != "" || !strings.Contains(preview.Post, "Smith, John") || strings.Contains(preview.Post, "13593") {
		t.Fatalf("not plain English: %+v", preview)
	}
	bad := rawBid("5", 1)
	bad.Player = "99999"
	if got := a.TargetBidPreview(bad); got.Problem == "" || strings.Contains(got.Problem, "99999") {
		t.Fatalf("unknown player leaks an ID: %+v", got)
	}
	if after := a.TargetBidPlans("0001"); !reflect.DeepEqual(before.Value, after.Value) {
		t.Fatal("preview saved a plan")
	}
}
