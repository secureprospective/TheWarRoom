package commissioner

import (
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func TestNativeOverrideBoundary(t *testing.T) {
	for _, tc := range []struct{ command, value, gravity string }{
		{ContractEdit, `{"player":"0099","years":4,"salaryCents":0}`, "G2"},
		{CapAdjust, `{"franchise":"franchise.02","amountCents":-100,"season":2027}`, "G2"},
		{CapRelief, `{"player":"0099","amountCents":0}`, "G2"},
		{ReserveException, `{"player":"0099","bucket":"ir"}`, "G3"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			raw := RawCommand{League: "native.test", Command: tc.command, Args: RawArgs{Value: json.RawMessage(tc.value)},
				Reason: "Correct held facts", Request: "correction", Effective: "now", Visibility: "league"}
			if err := raw.Validate(); err != nil {
				t.Fatal(err)
			}
			definition, err := raw.Definition()
			if err != nil || definition.Gravity != tc.gravity {
				t.Fatalf("gravity: %+v %v", definition, err)
			}
			for _, change := range []func(*RawCommand){
				func(r *RawCommand) { r.Reason = " " },
				func(r *RawCommand) { r.Gravity = "G1" },
				func(r *RawCommand) { r.Visibility = "private" },
				func(r *RawCommand) { r.Effective = "next_rollover" },
				func(r *RawCommand) { r.Args.People = &RawPeople{} },
			} {
				bad := raw
				change(&bad)
				if err := bad.Validate(); err == nil {
					t.Fatalf("accepted invalid correction: %+v", bad)
				}
			}
		})
	}
	for _, tc := range []struct{ command, value string }{
		{ContractEdit, `{"player":"99","years":1,"salaryCents":0}`},
		{ContractEdit, `{"player":"0099","years":0,"salaryCents":0}`},
		{ContractEdit, `{"player":"0099","years":1,"salaryCents":-1}`},
		{ContractEdit, `{"player":"0099","years":1}`},
		{ContractEdit, `{"player":"0099","years":1,"salaryCents":0,"amountCents":1}`},
		{CapAdjust, `{"franchise":"franchise.02","amountCents":1,"season":0}`},
		{CapRelief, `{"player":"0099","amountCents":-1}`},
		{CapRelief, `{"player":"0099","amountCents":1,"years":1}`},
		{ReserveException, `{"player":"0099","bucket":"active"}`},
		{ReserveException, `{"player":"0099","bucket":"ir","until":"tomorrow"}`},
		{ReserveException, `{"player":"0099","bucket":"ir","unknown":1}`},
		{ReserveException, `{"player":"0099","bucket":"ir"} {}`},
	} {
		if _, err := decodeOverride(tc.command, json.RawMessage(tc.value)); err == nil {
			t.Fatalf("accepted invalid value: %s", tc.value)
		}
	}
	if changed, words := PendingTradeCapCheck(RawCommand{}, nil); changed || words != "No pending trades" {
		t.Fatalf("unexpected pending trade ruling: %t %s", changed, words)
	}
}

func TestNativeOverridePermissions(t *testing.T) {
	state, err := governance.Replay([]governance.Event{
		governance.LeagueCreated{Creator: "creator", Franchises: []governance.FranchiseID{"f1"}},
		governance.SeatReplaced{Franchise: "f1", Principal: "owner"},
		governance.RoleGranted{Principal: "cc", Role: governance.CoCommissioner},
		governance.RoleGranted{Principal: "dot", Role: governance.DOT},
	})
	if err != nil {
		t.Fatal(err)
	}
	auth := GovernanceAuthorizer{State: state}
	for _, command := range []string{ContractEdit, CapAdjust, CapRelief, ReserveException} {
		for _, principal := range []string{"creator", "cc", "dot", "owner", "stranger"} {
			_, err := auth.Allow("native.test", principal, command)
			allowed := principal == "creator" || principal == "cc"
			if (err == nil) != allowed {
				t.Fatalf("%s permission for %s: %v", command, principal, err)
			}
		}
	}
}

func TestNativeChargeSeasonBoundary(t *testing.T) {
	for _, season := range []int{-1, 0, 2027, 9999, 10000} {
		seed := RawNativeSnapshot{Adjustments: []NativeCharge{{ID: "charge", Franchise: "f1", Season: season}}}
		err := seed.validateCharges(map[string]bool{"f1": true})
		if (err == nil) != (season >= 0 && season <= 9999) {
			t.Fatalf("season %d: %v", season, err)
		}
	}
}
