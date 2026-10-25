package commissioner

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	domain "github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/hostedfixture"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
)

type releaseTokenFixture struct {
	app      *ledgerFixture
	store    *accountstore.Store
	sessions *accounts.SessionService
	tokens   *accounts.TokenService
	creator  accounts.DeviceSession
}

func newReleaseTokenFixture(t *testing.T) *releaseTokenFixture {
	t.Helper()
	app := commissionerTestApp(t)
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "accounts.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	store := accountstore.New(pools)
	credentials := accounts.RawCredentials{Username: "commissioner", Password: "release-matrix-password"}
	principal, err := hostedfixture.Bootstrap(t.Context(), store, credentials)
	if err != nil {
		t.Fatal(err)
	}
	app.session = string(principal)
	executeCommissionerTest(t, app, domain.RawCommand{League: "native.hosted", Command: domain.Create,
		Args:    domain.RawArgs{Name: "Matrix league", Preset: domain.PresetEmpty, Season: 2026, FranchiseCount: 4},
		Request: "create", Reason: "Create disposable league", Effective: "now", Visibility: "league"})
	sessions := accounts.NewSessions(accounts.New(store), store)
	creator, err := sessions.Login(t.Context(), credentials, "matrix commissioner")
	if err != nil {
		t.Fatal(err)
	}
	return &releaseTokenFixture{app: app, store: store, sessions: sessions, creator: creator,
		tokens: accounts.NewTokens(store, serverhttp.TokenPolicy{Store: app.history})}
}

func (f *releaseTokenFixture) createOtherLeague(t *testing.T, account accounts.Account) {
	t.Helper()
	principal, err := account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.history.ExecuteCommissioner(t.Context(), domain.RawCommand{
		League: "native.other", Command: domain.Create,
		Args:    domain.RawArgs{Name: "Other matrix league", Preset: domain.PresetEmpty, Season: 2026, FranchiseCount: 4},
		Request: "create", Reason: "Create separate league", Effective: "now", Visibility: "league",
	}, string(principal), nil); err != nil {
		t.Fatal(err)
	}
}

func (f *releaseTokenFixture) change(t *testing.T, command string, people domain.RawPeople) {
	t.Helper()
	records := f.records(t, "native.hosted")
	visibility := "league"
	if strings.HasPrefix(command, "commish.role.") {
		visibility = "private"
	}
	_, err := f.app.history.ExecuteCommissioner(t.Context(), domain.RawCommand{
		League: "native.hosted", Command: command, Args: domain.RawArgs{People: &people},
		Expected: int64(len(records)), Request: command + "." + strconv.Itoa(len(records)),
		Reason: "Set disposable matrix authority", Effective: "now", Visibility: visibility,
	}, f.app.session, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func (f *releaseTokenFixture) records(t *testing.T, league string) []domain.Record {
	t.Helper()
	records, err := (leagueservice.StoredHost{Store: f.app.history}).CommissionerRecords(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	return records
}

func (f *releaseTokenFixture) enroll(t *testing.T, username string) accounts.DeviceSession {
	t.Helper()
	records := f.records(t, "native.hosted")
	state, err := domain.GovernanceState(records)
	if err != nil {
		t.Fatal(err)
	}
	enrollment := accounts.NewEnrollment(f.store, releaseEnrollment{state: state, principal: governance.Principal(f.app.session)})
	code, err := enrollment.Issue(t.Context(), "native.hosted", username)
	if err != nil {
		t.Fatal(err)
	}
	credentials := accounts.RawCredentials{Username: username, Password: "release-matrix-password"}
	if _, err := enrollment.Redeem(t.Context(), accounts.RawEnrollment{Code: code.Code, Username: username, Password: credentials.Password}); err != nil {
		t.Fatal(err)
	}
	session, err := f.sessions.Login(t.Context(), credentials, "matrix "+username)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func TestReleaseMatrixCatalogTokenCeilingsAndRevocation(t *testing.T) {
	t.Parallel()
	f := newReleaseTokenFixture(t)
	f.change(t, "commish.seat.replace", domain.RawPeople{Franchise: "franchise.01", Principal: f.app.session})
	other := f.enroll(t, "other-commissioner")
	f.createOtherLeague(t, other.Account)
	principal, err := other.Account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	before, otherBefore := f.records(t, "native.hosted"), f.records(t, "native.other")
	for _, command := range NativeCommands() {
		if !slices.Contains(command.Modes, catalog.Execute) {
			continue
		}
		t.Run(command.ID, func(t *testing.T) {
			for _, catalogMode := range command.Modes {
				mode := string(catalogMode)
				t.Run(mode, func(t *testing.T) {
					scope := accounts.RawTokenScope{League: "native.hosted", Franchise: "franchise.01", Commands: []string{command.ID}, Modes: []string{mode}}
					token, err := f.tokens.Issue(t.Context(), f.creator.RawSession, scope)
					if !accounts.DelegableCommand(command.ID) {
						if err == nil {
							t.Fatal("undelegable command received a token")
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					authenticated, err := f.tokens.Authenticate(t.Context(), token.RawSession)
					if err != nil || authenticated.ID != token.ID || !reflect.DeepEqual(authenticated.Scope, scope) {
						t.Fatalf("issued token did not retain its exact ceiling: %v", err)
					}
					gate := f.tokens.Admission(token.RawSession)
					otherMode := "execute"
					if mode == "execute" {
						otherMode = "preview"
					}
					for name, access := range map[string]domain.Access{
						"other-league":    {Principal: f.app.session, League: "native.other", Operation: command.ID, Mode: mode},
						"other-franchise": {Principal: f.app.session, League: scope.League, Operation: command.ID, Mode: mode, Franchise: "franchise.02"},
						"other-mode":      {Principal: f.app.session, League: scope.League, Operation: command.ID, Mode: otherMode},
						"other-command":   {Principal: f.app.session, League: scope.League, Operation: "commish.ledger.read", Mode: mode},
						"other-principal": {Principal: string(principal), League: scope.League, Operation: command.ID, Mode: mode},
						"raw-league":      {Principal: f.app.session, League: scope.League, Operation: command.ID, Mode: mode, Raw: domain.RawCommand{League: "native.other", Command: command.ID}},
					} {
						if err := gate.Admit(t.Context(), access); !errors.Is(err, accounts.ErrTokenRefused) {
							t.Fatalf("%s: expected token-scope refusal, got %v", name, err)
						}
					}
					if err := f.tokens.Revoke(t.Context(), f.creator.RawSession, token.ID); err != nil {
						t.Fatal(err)
					}
					if err := gate.Admit(t.Context(), domain.Access{Principal: f.app.session, League: scope.League, Operation: command.ID, Mode: mode}); !errors.Is(err, accounts.ErrTokenRefused) {
						t.Fatalf("revoked token: expected credential refusal, got %v", err)
					}
				})
			}
			if !reflect.DeepEqual(before, f.records(t, "native.hosted")) || !reflect.DeepEqual(otherBefore, f.records(t, "native.other")) {
				t.Fatal("token boundary checks changed a league ledger")
			}
		})
	}
}

type releaseEnrollment struct {
	state     governance.State
	principal governance.Principal
}

func (e releaseEnrollment) EnrollmentPolicy(context.Context, string) (governance.State, governance.Principal, error) {
	return e.state, e.principal, nil
}
