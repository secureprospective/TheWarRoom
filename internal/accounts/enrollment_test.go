package accounts_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
)

type enrollmentAuthority struct {
	policy    governance.State
	principal governance.Principal
	err       error
}

func (a *enrollmentAuthority) EnrollmentPolicy(context.Context, string) (governance.State, governance.Principal, error) {
	return a.policy, a.principal, a.err
}

func TestEnrollmentServiceIdentityHandoffAndIssuerRights(t *testing.T) {
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "accounts.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	// Use the actual account store, including its single writer and password hashing.
	store := accountstore.New(pools)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	bootstrap, err := accounts.New(store).LocalBootstrap(t.Context(), accounts.RawCredentials{Username: "commissioner", Password: "private-bootstrap"})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := bootstrap.Principal()
	if err != nil {
		t.Fatal(err)
	}
	policy, err := governance.Apply(governance.State{}, governance.LeagueCreated{Creator: principal, Franchises: []governance.FranchiseID{"001", "002"}})
	if err != nil {
		t.Fatal(err)
	}
	authority := &enrollmentAuthority{policy: policy, principal: principal}
	service := accounts.NewEnrollment(store, authority)
	issued := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	store.SetClock(func() time.Time { return issued })
	code, err := service.Issue(t.Context(), "native.test", "New.Owner")
	if err != nil || !code.ExpiresAt.Equal(issued.Add(48*time.Hour)) {
		t.Fatalf("issue: %v", err)
	}
	second, err := service.Issue(t.Context(), "native.test", "another.owner")
	if err != nil || second.Code == code.Code {
		t.Fatalf("randomness: %v", err)
	}
	expected := sha256.Sum256([]byte(code.Code))
	var digest string
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT code_hash FROM account_enrollments WHERE username = ?`, "new.owner").Scan(&digest); err != nil || digest != hex.EncodeToString(expected[:]) {
		t.Fatalf("not hash-only: %v", err)
	}
	raw := accounts.RawEnrollment{Code: code.Code, Username: "wrong.owner", Password: "sentinel-enrollment-password"}
	if _, err := service.Redeem(t.Context(), raw); !errors.Is(err, accounts.ErrEnrollmentRefused) {
		t.Fatalf("wrong identity: %v", err)
	}
	raw.Username = "NEW.OWNER"
	account, err := service.Redeem(t.Context(), raw)
	if err != nil || account.Username != "new.owner" || account.ID == bootstrap.ID {
		t.Fatalf("redeem: %v", err)
	}
	if _, err := service.Redeem(t.Context(), raw); !errors.Is(err, accounts.ErrEnrollmentRefused) {
		t.Fatalf("replay: %v", err)
	}
	got, err := accounts.New(store).Authenticate(t.Context(), accounts.RawCredentials{Username: account.Username, Password: raw.Password})
	if err != nil || got != account {
		t.Fatalf("new account login: %v", err)
	}
	for _, value := range []string{fmt.Sprintf("%v %+v %#v", raw, raw, raw), fmt.Sprintf("%v %+v %#v", code, code, code)} {
		if strings.Contains(value, raw.Password) || strings.Contains(value, code.Code) {
			t.Fatal("enrollment secrets in log formatting")
		}
	}
	body, err := json.Marshal(raw)
	if err != nil || strings.Contains(string(body), raw.Password) || strings.Contains(string(body), code.Code) {
		t.Fatalf("JSON leaks enrollment secrets: %v", err)
	}
	authority.principal, err = account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Issue(t.Context(), "native.test", "forbidden.owner"); err == nil {
		t.Fatal("noncommissioner issued code")
	}
	if _, err := accounts.NewEnrollment(store, nil).Issue(t.Context(), "native.test", "forbidden.owner"); err == nil {
		t.Fatal("missing authority issued code")
	}
	authority.err = errors.New("policy unavailable")
	if _, err := service.Issue(t.Context(), "native.test", "forbidden.owner"); err == nil {
		t.Fatal("policy failure issued code")
	}
	var count int
	if err := pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM account_enrollments`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("denied issuance changed store: %d %v", count, err)
	}
}
