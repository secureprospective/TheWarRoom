package serverclient

import (
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFixtureClientRejectsInvalidResponses(t *testing.T) {
	for _, body := range []string{
		`{"protocol":2,"ledger":{}}`,
		`{"protocol":1,"ledger":{}}`,
		`not JSON`,
		strings.Repeat("x", (256<<10)+1),
	} {
		t.Run(body[:8], func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if _, err := w.Write([]byte(body)); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			roots := x509.NewCertPool()
			roots.AddCert(server.Certificate())
			client, err := NewFixture(server.URL, roots)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			if _, err := client.FixtureLedger(t.Context()); err == nil {
				t.Fatal("invalid response accepted")
			}
		})
	}
}

func TestFixtureClientRejectsUnsafeOrigins(t *testing.T) {
	for _, address := range []string{"http://127.0.0.1", "https://example.com", "https://0.0.0.0", "https://127.0.0.1/path", "https://user@127.0.0.1", "https://127.0.0.1?principal=creator"} {
		if client, err := NewFixture(address, x509.NewCertPool()); err == nil {
			client.Close()
			t.Fatalf("accepted %s", address)
		}
	}
	if _, err := NewFixture("https://127.0.0.1", nil); err == nil {
		t.Fatal("accepted implicit trust")
	}
}

func TestFixtureClientDoesNotFollowRedirects(t *testing.T) {
	targetCalled := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirected" {
			targetCalled = true
		}
		http.Redirect(w, r, "/redirected", http.StatusFound)
	}))
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	client, err := NewFixture(server.URL, roots)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.FixtureLedger(t.Context()); err == nil {
		t.Fatal("redirect accepted")
	}
	if targetCalled {
		t.Fatal("followed redirect")
	}
}
