package commissioner

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func TestPeopleBoundary(t *testing.T) {
	raw := RawCommand{League: "native.test", Command: "commish.role.grant",
		Args:   RawArgs{People: &RawPeople{Principal: "cc", Role: governance.CoCommissioner}},
		Reason: "Authorize our deputy", Request: "grant", Effective: "now", Visibility: "private"}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(string(body)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		strings.Replace(string(body), `"visibility":"private"`, `"visibility":"league"`, 1),
		strings.Replace(string(body), `"reason":"Authorize our deputy"`, `"reason":""`, 1),
		strings.Replace(string(body), `"expected":0`, `"expected":0,"minimumRole":"owner"`, 1),
		strings.Replace(string(body), `"expected":0`, `"expected":0,"cOnly":false`, 1),
		strings.Replace(string(body), `"expected":0`, `"expected":0,"gravity":"G1"`, 1),
		strings.Replace(string(body), `"people":{`, `"people":{"ability":"trades",`, 1),
	} {
		if _, err := Decode(bad); err == nil {
			t.Fatalf("forged metadata/private command accepted: %s", bad)
		}
	}
	raw.Command = "commish.franchise.edit"
	raw.Args.People = &RawPeople{Franchise: "f1", Name: "Club", Abbreviation: "CLB"}
	raw.Visibility, raw.Reason = "league", ""
	if err := raw.Validate(); err != nil {
		t.Fatalf("G2 unnecessarily requires a reason: %v", err)
	}
	raw.Gravity = "G3"
	if err := raw.Validate(); err == nil {
		t.Fatal("caller chose gravity")
	}
}

func TestDeputyCommandReplayAndCommissionerOnlyPowers(t *testing.T) {
	b := newLedger(t, PresetEmpty)
	for _, p := range []string{"z", "a"} {
		raw := RawCommand{Command: "commish.role.grant", Args: RawArgs{People: &RawPeople{Principal: p, Role: governance.CoCommissioner}}}
		if err := CheckGovernanceAppend(raw, "creator", b.records, seasonMoment(2)); err != nil {
			t.Fatal(err)
		}
		b.add(raw, seasonMoment(2))
	}
	state, err := GovernanceState(b.records)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"z", "a"} {
		for _, command := range []string{"commish.role.grant", "commish.role.revoke", ContractActionSet, TradeOverride} {
			if _, err := (GovernanceAuthorizer{State: state}).Allow("native.test", p, command); err == nil {
				t.Fatalf("%s gained %s", p, command)
			}
		}
	}
	raw := RawCommand{Command: "commish.role.revoke", Args: RawArgs{People: &RawPeople{Principal: "z", Role: governance.CoCommissioner}}}
	b.add(raw, seasonMoment(3))
	raw = RawCommand{Command: "commish.succession.stepdown", Args: RawArgs{People: &RawPeople{Principal: "creator"}}}
	if err := CheckGovernanceAppend(raw, "creator", b.records, seasonMoment(4)); err != nil {
		t.Fatal(err)
	}
	b.add(raw, seasonMoment(4))
	state, err = GovernanceState(b.records)
	if err != nil || !state.HasRole("a", governance.Commissioner) || state.HasRole("z", governance.CoCommissioner) {
		t.Fatalf("wrong succession after command replay: %v", err)
	}
}
