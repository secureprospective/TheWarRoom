package serverclient

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestDesktopViewIdentityGeneration(t *testing.T) {
	bearer := strings.Repeat("a", 42) + "A"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case serverhttp.LoginPath:
			_, _ = fmt.Fprintf(w, `{"protocol":1,"bearer":%q,"session":{"account":{"id":"11111111-1111-4111-8111-111111111111","username":"owner"},"expires_at":"2030-01-01T00:00:00Z"}}`, bearer)
		case serverhttp.PeoplePath:
			_, _ = fmt.Fprint(w, `{"League":"native.test","Name":"Test league","YourRole":"Owner","YourFranchise":"","Seats":[]}`)
		case serverhttp.RevokePath:
			_, _ = fmt.Fprint(w, `{"protocol":1}`)
		case serverhttp.InvalidationPath:
			<-r.Context().Done()
		}
	}))
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	d, err := NewDesktop(server.URL, "native.test", roots, &testCredentials{})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	before, err := d.Act(t.Context(), RawDesktop{Action: "status"})
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"login", "enter", "forget", "login", "revoke-device", "login", "revoke-all"} {
		raw := RawDesktop{Action: action}
		if action == "login" {
			raw.Username, raw.Password = "owner", "owner-password"
		}
		view, actErr := d.Act(t.Context(), raw)
		if actErr != nil {
			t.Fatalf("%s: %v", action, actErr)
		}
		if view.AuthGeneration == before.AuthGeneration {
			t.Fatalf("%s kept generation %s", action, view.AuthGeneration)
		}
		if _, parseErr := strconv.ParseUint(view.AuthGeneration, 10, 64); parseErr != nil || strings.Contains(view.AuthGeneration, bearer) {
			t.Fatalf("unsafe generation %q", view.AuthGeneration)
		}
		change := d.changesAfter(t.Context(), 0, time.Millisecond)
		request := httptest.NewRequest(http.MethodPost, DesktopChangesPath, strings.NewReader(`{"after":0}`))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		d.Handler().ServeHTTP(recorder, request)
		if decodeErr := json.Unmarshal(recorder.Body.Bytes(), &change); decodeErr != nil {
			t.Fatal(decodeErr)
		}
		if change.AuthGeneration != view.AuthGeneration {
			t.Fatalf("%s: stream=%s view=%s", action, change.AuthGeneration, view.AuthGeneration)
		}
		before = view
	}
}
