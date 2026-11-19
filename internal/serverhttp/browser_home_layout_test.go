package serverhttp_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestBrowserHomeLayoutUsesSameOriginAccountCookie(t *testing.T) {
	f := newHostedFixture(t, true)
	_, cookie := appLogin(t, f, "commissioner", "private-password", "")
	request := layoutRequest(t, "home.layout.read", "read", `{}`)
	response, body := browserRequest(t, f, http.MethodPost, serverhttp.AccountPreferencePath,
		request, cookie.Value, "", browserOrigin)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("browser layout read: status=%d body=%s", response.StatusCode, body)
	}
	var envelope serverhttp.WorkflowResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	var layout accounts.HomeLayoutState
	if err := json.Unmarshal(envelope.Data, &layout); err != nil || layout.Revision != 0 {
		t.Fatalf("browser layout = %+v err=%v", layout, err)
	}
	custom := `{"mode":"custom","tiles":[["cap","1x1"]],"over":[],"pins":{}}`
	write := layoutRequest(t, "home.layout.set", "execute",
		`{"layout":`+custom+`,"expectedRevision":0}`)
	response, body = browserRequest(t, f, http.MethodPost, serverhttp.AccountPreferencePath,
		write, cookie.Value, "", browserOrigin)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("same-origin browser layout write: status=%d body=%s", response.StatusCode, body)
	}
	response, _ = browserRequest(t, f, http.MethodPost, serverhttp.AccountPreferencePath,
		write, cookie.Value, "", "https://elsewhere.example")
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-origin layout write status=%d, want 403", response.StatusCode)
	}
}
