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

func TestCreationFranchises(t *testing.T) {
	for _, ids := range [][]string{nil, {""}, {"f1", "f1"}, {" f1"}} {
		raw := RawArgs{Name: "League", Franchises: ids}
		if err := raw.Validate(Create); err == nil {
			t.Fatalf("invalid franchise inventory accepted: %v", ids)
		}
	}
	if err := (RawArgs{Name: "League", Franchises: []string{"a", "b"}}).Validate(Create); err != nil {
		t.Fatal(err)
	}
}
