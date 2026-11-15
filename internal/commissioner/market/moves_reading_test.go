package market

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNativeMoveReceiptAdapter(t *testing.T) {
	m := nativeMove("native.test", "franchise.01", "event.1", "trade.offer", "needs_you", "Accept or counter this offer", seasonMoment(3))
	body, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"correlationId":`, `"spec":`, `"state":`, `"audit":`, `"kind":"native"`} {
		if !strings.Contains(string(body), key) {
			t.Fatalf("receipt missing %s: %s", key, body)
		}
	}
	if m.Spec.Target.Kind != "native" || len(m.Audit) != 1 || m.Audit[0].To != m.State || m.Audit[0].Event != "recorded" {
		t.Fatalf("native record was disguised as an MFL transition: %+v", m)
	}
}

func TestNativeMoveScopeValidation(t *testing.T) {
	for _, scope := range []RawNativeMovesScope{
		{League: "native.test", Franchise: "franchise.01"},
		{League: "native.", Franchise: "franchise.01"},
		{League: "12345", Franchise: "franchise.01"},
		{League: "native.test", Franchise: " "},
		{League: "native.test ", Franchise: "franchise.01"},
	} {
		valid := scope.League == "native.test" && scope.Franchise == "franchise.01"
		if err := scope.Validate(); (err == nil) != valid {
			t.Fatalf("scope %+v: %v", scope, err)
		}
	}
}
