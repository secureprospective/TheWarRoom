package commissioner

import (
	"strings"
	"testing"
)

const createBody = `{"league":"native.test","command":"commish.league.create","args":{"name":"Test","franchises":["f1"]},"reason":"start","expected":0,"request":"first","effective":"now","visibility":"league"}`

func TestDecodeCommissionerBoundary(t *testing.T) {
	raw, err := Decode(createBody)
	if err != nil || raw.Command != Create {
		t.Fatalf("decode: %+v %v", raw, err)
	}
	for _, body := range []string{
		strings.Replace(createBody, `"args":{"name":"Test","franchises":["f1"]}`, `"args":{"name":"Test","franchises":["f1"],"actor":"other"}`, 1),
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
