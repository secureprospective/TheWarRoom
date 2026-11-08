package serverhttp_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func joinBody(code, franchise, username string) string {
	if franchise == "" {
		return fmt.Sprintf(`{"protocol":1,"join":{"league":"native.hosted","code":%q}}`, code)
	}
	return fmt.Sprintf(`{"protocol":1,"claim":{"league":"native.hosted","code":%q,"franchise":%q,"username":%q,"password":"private-password"}}`, code, franchise, username)
}

func (f *hostedFixture) openFranchises(t *testing.T, code string) map[string]bool {
	t.Helper()
	var list serverhttp.JoinFranchisesResponse
	if err := json.Unmarshal(f.post(t, serverhttp.JoinFranchisesPath, joinBody(code, "", ""), "", "", 200), &list); err != nil {
		t.Fatal(err)
	}
	taken := map[string]bool{}
	for _, franchise := range list.Franchises {
		if franchise.Name == "" {
			t.Fatalf("franchise %s has no name", franchise.ID)
		}
		taken[franchise.ID] = franchise.Taken
	}
	return taken
}

func (f *hostedFixture) lastCommissionerEvent(t *testing.T) commissioner.Record {
	t.Helper()
	events, err := f.history.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
	if err != nil || len(events) == 0 {
		t.Fatal(err)
	}
	last := events[len(events)-1]
	if last.Payload.Commissioner == nil {
		t.Fatal("last event is not a commissioner change")
	}
	return *last.Payload.Commissioner
}

// The commissioner's join link lists franchises; a holder claims an open one, which is recorded in
// the league ledger as the issuing commissioner seating the new owner; the new account signs in as
// an ordinary owner. Taken, wrong, rotated and revoked links are refused; release reopens a franchise.
func TestJoinLinkClaimReleaseAndRotation(t *testing.T) {
	f := newHostedFixture(t, true)
	commish := governance.Principal(f.principal)
	if _, err := f.join.Rotate(t.Context(), "native.hosted", governance.Principal("acct.00000000-0000-4000-8000-000000000000")); err == nil {
		t.Fatal("a non-commissioner issued a join link")
	}
	code, err := f.join.Rotate(t.Context(), "native.hosted", commish)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(fmt.Sprintf("%v %#v", code, code), code.Code) {
		t.Fatal("join code printed by fmt")
	}
	taken := f.openFranchises(t, code.Code)
	if len(taken) != 32 || !taken["franchise.02"] || taken["franchise.04"] {
		t.Fatalf("franchises = %v", taken)
	}
	wrong := strings.Repeat("A", 43)
	f.post(t, serverhttp.JoinFranchisesPath, joinBody(wrong, "", ""), "", "", http.StatusForbidden)
	f.post(t, serverhttp.JoinClaimPath, joinBody(code.Code, "franchise.02", "thief"), "", "", http.StatusConflict)
	f.post(t, serverhttp.JoinClaimPath, joinBody(code.Code, "franchise.04", "fourth"), "", "", http.StatusCreated)
	claimed := f.lastCommissionerEvent(t)
	if claimed.Command.Command != "commish.seat.replace" || claimed.Command.Args.People.Franchise != "franchise.04" || claimed.Principal != f.principal {
		t.Fatalf("claim event = %s %+v by %s", claimed.Command.Command, claimed.Command.Args.People, claimed.Principal)
	}
	bearer := f.login(t, "fourth", "private-password")
	f.post(t, serverhttp.LedgerPath, `{"protocol":1,"league":"native.hosted"}`, bearer, "", 200)
	if f.openFranchises(t, code.Code)["franchise.04"] != true {
		t.Fatal("claimed franchise still open")
	}

	if err := f.join.Release(t.Context(), "native.hosted", "franchise.04", commish); err != nil {
		t.Fatal(err)
	}
	if released := f.lastCommissionerEvent(t); released.Command.Command != "commish.seat.remove" {
		t.Fatalf("release event = %s", released.Command.Command)
	}
	rotated, err := f.join.Rotate(t.Context(), "native.hosted", commish)
	if err != nil {
		t.Fatal(err)
	}
	f.post(t, serverhttp.JoinFranchisesPath, joinBody(code.Code, "", ""), "", "", http.StatusForbidden)
	if f.openFranchises(t, rotated.Code)["franchise.04"] {
		t.Fatal("released franchise still taken")
	}
	if err := f.join.Revoke(t.Context(), "native.hosted", commish); err != nil {
		t.Fatal(err)
	}
	if _, err := f.join.Franchises(t.Context(), accounts.RawJoin{League: "native.hosted", Code: rotated.Code}); !errors.Is(err, accounts.ErrJoinRefused) {
		t.Fatalf("revoked link = %v", err)
	}
}

// Two people claim the same open franchise at once: exactly one is seated and gets an account; the
// other is told it is taken and has no account.
func TestJoinClaimRaceExactlyOneWins(t *testing.T) {
	f := newHostedFixture(t, true)
	code, err := f.join.Rotate(t.Context(), "native.hosted", governance.Principal(f.principal))
	if err != nil {
		t.Fatal(err)
	}
	statuses := make([]int, 2)
	var wg sync.WaitGroup
	for i := range statuses {
		wg.Go(func() {
			statuses[i], _ = f.send(t, serverhttp.JoinClaimPath, joinBody(code.Code, "franchise.05", fmt.Sprintf("racer%d", i)), "", "")
		})
	}
	wg.Wait()
	won, lost := 0, 0
	for _, status := range statuses {
		switch status {
		case http.StatusCreated:
			won++
		case http.StatusConflict:
			lost++
		}
	}
	if won != 1 || lost != 1 {
		t.Fatalf("statuses = %v, want one 201 and one 409", statuses)
	}
	accountsMade := 0
	for i := range statuses {
		if _, err := accounts.New(f.accounts).Authenticate(t.Context(), accounts.RawCredentials{Username: fmt.Sprintf("racer%d", i), Password: "private-password"}); err == nil {
			accountsMade++
		}
	}
	seats := 0
	events, err := f.history.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if c := event.Payload.Commissioner; c != nil && c.Command.Command == "commish.seat.replace" && c.Command.Args.People.Franchise == "franchise.05" {
			seats++
		}
	}
	if accountsMade != 1 || seats != 1 {
		t.Fatalf("accounts=%d seat events=%d, want 1 and 1", accountsMade, seats)
	}
}
