package commissioner

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

const createBody = `{"league":"native.test","command":"commish.league.create","args":{"name":"Test","preset":"empty","season":2026,"franchiseCount":2},"reason":"start","expected":0,"request":"first","effective":"now","visibility":"league"}`

func TestDecodeCommissionerBoundary(t *testing.T) {
	raw, err := Decode(createBody)
	if err != nil || raw.Command != Create {
		t.Fatalf("decode: %+v %v", raw, err)
	}
	for _, body := range []string{
		strings.Replace(createBody, `"args":{"name":"Test","preset":"empty","season":2026,"franchiseCount":2}`, `"args":{"name":"Test","preset":"empty","season":2026,"franchiseCount":2,"actor":"other"}`, 1),
		strings.Replace(createBody, `"expected":0`, `"expected":-1`, 1),
		strings.Replace(createBody, `"reason":"start"`, `"reason":" \t"`, 1),
		strings.Replace(createBody, `"expected":0`, `"expected":0,"principal":"other"`, 1),
		strings.Replace(createBody, `"expected":0`, `"expected":0,"role":"commissioner"`, 1),
		strings.Replace(createBody, `"effective":"now"`, `"effective":"never"`, 1),
		strings.Replace(createBody, `"visibility":"league"`, `"visibility":"public"`, 1),
		strings.Replace(createBody, `"command":"commish.league.create"`, `"command":"commish.unknown.run"`, 1),
		createBody + " {}",
	} {
		if _, err := Decode(body); err == nil {
			t.Fatalf("hostile body accepted: %s", body)
		}
	}
	canonical, err := raw.Canonical("creator")
	if err != nil {
		t.Fatal(err)
	}
	raw.Gravity = "G3"
	explicit, err := raw.Canonical("creator")
	if err != nil || explicit != canonical {
		t.Fatalf("canonical gravity: %s %v", explicit, err)
	}
	other, err := raw.Canonical("other")
	if err != nil || other == canonical {
		t.Fatal("principal missing from retry identity")
	}
}

func TestRefusalReasonIsTheInnermostSentence(t *testing.T) {
	leaf := errors.New("this league already has a co-commissioner: revoke them first")
	wrapped := fmt.Errorf("check league change: %w", Refused(fmt.Errorf("people change refused: change 19: %w", leaf)))
	var r *Refusal
	if !errors.As(wrapped, &r) || r.Reason() != "This league already has a co-commissioner: revoke them first" {
		t.Fatalf("reason: %v", wrapped)
	}
	if !errors.Is(wrapped, leaf) || Refused(r) != error(r) || Refused(nil) != nil {
		t.Fatal("a refusal must keep its cause, mark once and leave nil alone")
	}
	if errors.As(fmt.Errorf("record league change: %w", leaf), &r) {
		t.Fatal("an unmarked failure is not a refusal")
	}
}
