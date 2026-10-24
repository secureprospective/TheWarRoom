package accounts_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestLocalBootstrapAuthenticationAndLeagueIsolation(t *testing.T) {
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "server.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	ledger := history.New(pools, registry)
	if err := ledger.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	raw := commissioner.RawCommand{League: "native.test", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Test league", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2},
		Reason: "Start a fixture league", Request: "create", Effective: "now", Visibility: "league"}
	if _, err := ledger.ExecuteCommissioner(t.Context(), raw, "league-creator", nil); err != nil {
		t.Fatal(err)
	}
	before, err := ledger.CommissionerLedger(t.Context(), raw.League, "league-creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	store := accountstore.New(pools)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	service := accounts.New(store)
	credentials := accounts.RawCredentials{Username: "Host.Operator", Password: "sentinel-private-bootstrap-password"}
	account, err := service.LocalBootstrap(t.Context(), credentials)
	if err != nil {
		t.Fatal(err)
	}
	if account.Username != "host.operator" {
		t.Fatal("username is not canonical")
	}
	principal, err := account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	if len(account.ID) != 36 || account.ID[14] != '4' || !strings.HasPrefix(string(principal), "acct.") {
		t.Fatal("not a canonical random account UUID")
	}
	credentials.Username = "HOST.OPERATOR"
	got, err := service.Authenticate(t.Context(), credentials)
	if err != nil || got != account {
		t.Fatalf("correct password: %v", err)
	}
	for _, wrong := range []accounts.RawCredentials{
		{Username: "host.operator", Password: "wrong-password"},
		{Username: "missing", Password: credentials.Password},
		{Username: "host.operator", Password: ""},
	} {
		if got, err := service.Authenticate(t.Context(), wrong); !errors.Is(err, accounts.ErrInvalidCredentials) || got != (accounts.Account{}) {
			t.Fatalf("bad login: %v", err)
		}
	}
	if _, err := service.LocalBootstrap(t.Context(), credentials); !errors.Is(err, accounts.ErrBootstrapClosed) {
		t.Fatalf("second bootstrap: %v", err)
	}
	people, err := ledger.CommissionerPeople(t.Context(), raw.League, string(principal))
	if err != nil {
		t.Fatal(err)
	}
	if people.YourRole != "Visitor" || people.YourFranchise != "" || people.MayGrantRoles || people.MayChangeRules || people.MayPostNotices {
		t.Fatal("host bootstrap acquired league rights")
	}
	for _, role := range people.Roles {
		if role.Person == string(principal) {
			t.Fatal("host bootstrap acquired a league role")
		}
	}
	for _, seat := range people.Seats {
		if seat.Owner == string(principal) {
			t.Fatal("host bootstrap acquired a franchise")
		}
	}
	after, err := ledger.CommissionerLedger(t.Context(), raw.League, "league-creator", nil)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("account operations changed league ledger: %v", err)
	}
	stored, err := store.Credential(t.Context(), account.Username)
	if err != nil {
		t.Fatal(err)
	}
	assertNoSecrets(t, credentials, stored, after)
}

func assertNoSecrets(t *testing.T, raw accounts.RawCredentials, stored accounts.Credential, ledger commissioner.Ledger) {
	t.Helper()
	for _, value := range []string{fmt.Sprintf("%v %+v %#v", raw, raw, raw), fmt.Sprintf("%v %+v %#v", stored, stored, stored)} {
		if strings.Contains(value, raw.Password) || strings.Contains(value, stored.PasswordHash) {
			t.Fatal("credentials exposed by log formatting")
		}
	}
	body, err := json.Marshal(struct {
		Raw    accounts.RawCredentials
		Stored accounts.Credential
		Ledger commissioner.Ledger
	}{raw, stored, ledger})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), raw.Password) || strings.Contains(string(body), stored.PasswordHash) {
		t.Fatal("credentials exposed in JSON or ledger")
	}
}
