package offers

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
)

func TestMFLPublicationStaysOnDesktop(t *testing.T) {
	commands, err := catalog.Publication(MFLCommands())
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 9 {
		t.Fatalf("missing pool or plan operations: %d", len(commands))
	}
	methods := map[string]bool{}
	for _, c := range commands {
		if c.Availability != "desktop" || c.Permission != "local-desktop-owner" || c.Role != "gm,commish,admin" || !strings.Contains(c.Safeguards, "no MFL submissions") {
			t.Fatalf("local ownership boundary lost: %+v", c)
		}
		if c.Gravity == "G2" && !strings.Contains(c.Safeguards, "explicit-owner-action") {
			t.Fatal(c.ID)
		}
		if slices.Contains(c.Modes, catalog.Read) && c.Undo != "none" {
			t.Fatal(c.ID)
		}
		methods[c.Method] = true
	}
	for _, method := range []string{"TargetPool", "TargetPoolRefresh", "TargetBidPlans", "TargetBidPreview", "TargetBidDraft", "TargetBidRevise", "TargetBidHandOff", "TargetBidPosted", "TargetBidWithdraw"} {
		if !methods[method] {
			t.Fatal("missing", method)
		}
	}
	for _, c := range Commands() {
		if methods[c.Method] && c.Version != "1" {
			t.Fatalf("binding lost publication: %+v", c)
		}
	}
	if !reflect.DeepEqual(commands[3].Arguments, catalog.Shape(reflect.TypeFor[RawBidInput](), "RawBidInput.Validate")) {
		t.Fatal("bid schema drift")
	}
}

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
