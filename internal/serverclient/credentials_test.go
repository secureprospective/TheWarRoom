package serverclient

import (
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/zalando/go-keyring"
)

type testCredentials struct {
	secret      string
	unavailable bool
}

func (s *testCredentials) Get() (string, error) {
	if s.unavailable {
		return "", errors.New("secret-provider-sentinel")
	}
	return s.secret, nil
}
func (s *testCredentials) Set(secret string) error {
	if s.unavailable {
		return errors.New("secret-provider-sentinel")
	}
	s.secret = secret
	return nil
}
func (s *testCredentials) Delete() error {
	if s.unavailable {
		return errors.New("secret-provider-sentinel")
	}
	s.secret = ""
	return nil
}

func TestDeviceKeychainRoundtripAndIsolation(t *testing.T) {
	keyring.MockInit()
	a := newDeviceKeychain("https://a.invalid", "native.a")
	b := newDeviceKeychain("https://b.invalid", "native.a")
	c := newDeviceKeychain("https://a.invalid", "native.b")
	secret := strings.Repeat("a", 42) + "A"
	if err := a.Set(secret); err != nil {
		t.Fatal(err)
	}
	if got, err := a.Get(); err != nil || got != secret {
		t.Fatal("keychain roundtrip", err)
	}
	for _, other := range []CredentialStore{b, c} {
		if _, err := other.Get(); err == nil {
			t.Fatal("cross-origin/league credential")
		}
	}
	if err := a.Delete(); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Get(); err == nil {
		t.Fatal("delete did not remove credential")
	}
	if err := a.Delete(); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialRestoreCloseAndForget(t *testing.T) {
	store := &testCredentials{secret: strings.Repeat("a", 42) + "A"}
	d, err := NewDesktop("https://a.invalid", "native.a", nil, store)
	if err != nil {
		t.Fatal(err)
	}
	view, err := d.Act(t.Context(), RawDesktop{Action: "status"})
	if err != nil || d.bearer != store.secret || strings.Contains(fmt.Sprint(view), store.secret) {
		t.Fatal("restore failed or leaked")
	}
	d.Close()
	if store.secret == "" || d.bearer != "" {
		t.Fatal("close must retain keychain, clear memory")
	}
	d, err = NewDesktop("https://a.invalid", "native.a", nil, store)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := d.Act(t.Context(), RawDesktop{Action: "forget"}); err != nil {
		t.Fatal(err)
	}
	if store.secret != "" || d.bearer != "" {
		t.Fatal("forget retained credential")
	}
}

func TestUnavailableKeychainLoginIsMemoryOnly(t *testing.T) {
	secret := strings.Repeat("a", 42) + "A"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == serverhttp.LoginPath {
			if _, err := fmt.Fprintf(w, `{"protocol":1,"bearer":%q,"session":{"account":{"id":"12345678-1234-4234-8234-123456789abc","username":"owner"},"expires_at":"2026-11-01T00:00:00Z"}}`, secret); err != nil {
				t.Error(err)
			}
			return
		}
		http.Error(w, "refused", http.StatusUnauthorized)
	}))
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	store := &testCredentials{unavailable: true}
	d, err := NewDesktop(server.URL, "native.a", roots, store)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	view, err := d.Act(t.Context(), RawDesktop{Action: "login", Username: "owner", Password: "private-password"})
	if err != nil || !strings.Contains(view.Message, "window only") || d.bearer != secret || store.secret != "" {
		t.Fatalf("memory fallback: %+v %v", view, err)
	}
	if _, err := d.Act(t.Context(), RawDesktop{Action: "enter"}); err == nil || d.bearer != "" {
		t.Fatal("revoked credential survived")
	}
	d.Close()
	if store.secret != "" {
		t.Fatal("fallback persisted secret")
	}
}
