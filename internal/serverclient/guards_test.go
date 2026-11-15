package serverclient

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDesktopViewNeverSerializesCredentials(t *testing.T) {
	desktop, err := NewDesktop("https://example.invalid", "native.test", nil, &testCredentials{unavailable: true})
	if err != nil {
		t.Fatal(err)
	}
	defer desktop.Close()
	desktop.bearer, desktop.proof = "bearer-sentinel", "proof-sentinel"
	view, err := desktop.Act(t.Context(), RawDesktop{Action: "status"})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(view)
	if err != nil || strings.Contains(string(body), "sentinel") {
		t.Fatalf("secret crossed webview: %s %v", body, err)
	}
}
