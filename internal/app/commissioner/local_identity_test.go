package commissioner

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
)

type fixedLocalIdentity struct{ value string }

func (s fixedLocalIdentity) Principal() (string, error) { return s.value, nil }

func TestLocalIdentityStableScopedAndLazyFallback(t *testing.T) {
	first, err := LocalPrincipal("installation", 42)
	if err != nil {
		t.Fatal(err)
	}
	second, err := LocalPrincipal("installation", 42)
	if err != nil || first != second || !strings.HasPrefix(first, "local.") {
		t.Fatal(first, second, err)
	}
	other, err := LocalPrincipal("installation", 43)
	if err != nil || first == other {
		t.Fatal(other, err)
	}
	if _, err := LocalPrincipal("", 42); err == nil {
		t.Fatal("missing installation accepted")
	}
	got, err := ResolvePrincipal(fixedLocalIdentity{"injected"}, func() (string, error) { t.Fatal("eager fallback"); return "", nil })
	if err != nil || got != "injected" {
		t.Fatal(got, err)
	}
	if _, err := ResolvePrincipal(fixedLocalIdentity{"  "}, nil); err == nil {
		t.Fatal("blank identity accepted")
	}
}

func TestLocalSeatSignInRereadsAndRefusesBeforeResources(t *testing.T) {
	calls := 0
	s := LocalSeat{Ready: func() error { return nil }, Principal: func() (string, error) { return "local", nil }, Context: context.Background,
		Status: func(context.Context, string) (accounts.SeatStatus, error) { calls++; return accounts.SeatStatus{}, nil },
		SignIn: func(_ context.Context, _ accounts.RawCredentials, local string) (accounts.Account, error) {
			if local != "local" {
				t.Fatal(local)
			}
			calls++
			return accounts.Account{}, nil
		}}
	if _, err := s.Login(accounts.RawCredentials{}, s.Read); err != nil || calls != 2 {
		t.Fatal(err, calls)
	}
	failure := errors.New("startup failed")
	s.Ready = func() error { return failure }
	s.Principal = func() (string, error) { t.Fatal("identity read before ready"); return "", nil }
	if _, err := s.Read(); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if _, err := s.Login(accounts.RawCredentials{}, s.Read); !errors.Is(err, failure) {
		t.Fatal(err)
	}
}
