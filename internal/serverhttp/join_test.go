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

func (f *hostedFixture) franchises(t *testing.T, code string) map[string]accounts.JoinFranchise {
	t.Helper()
	var list serverhttp.JoinFranchisesResponse
	if err := json.Unmarshal(f.post(t, serverhttp.JoinFranchisesPath, joinBody(code, "", ""), "", "", 200), &list); err != nil {
		t.Fatal(err)
	}
	byID := map[string]accounts.JoinFranchise{}
	for _, franchise := range list.Franchises {
		if franchise.Name == "" {
			t.Fatalf("franchise %s has no name", franchise.ID)
		}
		byID[franchise.ID] = franchise
	}
	return byID
}

func (f *hostedFixture) openFranchises(t *testing.T, code string) map[string]bool {
	t.Helper()
	taken := map[string]bool{}
	for id, franchise := range f.franchises(t, code) {
		taken[id] = franchise.Taken
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

// The commissioner's join link lists franchises; one successful claim seats an ordinary owner and
// consumes the link. Taken, wrong, rotated, revoked and reused links are refused; release reopens a franchise.
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
	f.post(t, serverhttp.JoinClaimPath, joinBody(code.Code, "franchise.05", "fifth"), "", "", http.StatusForbidden)
	f.post(t, serverhttp.JoinFranchisesPath, joinBody(code.Code, "", ""), "", "", http.StatusForbidden)
	var secondAccount int
	if err := f.pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM accounts WHERE username = 'fifth'`).Scan(&secondAccount); err != nil || secondAccount != 0 {
		t.Fatalf("reused link created second account: count=%d err=%v", secondAccount, err)
	}
	claimed := f.lastCommissionerEvent(t)
	if claimed.Command.Command != "commish.seat.replace" || claimed.Command.Args.People.Franchise != "franchise.04" || claimed.Principal != f.principal {
		t.Fatalf("claim event = %s %+v by %s", claimed.Command.Command, claimed.Command.Args.People, claimed.Principal)
	}
	bearer := f.login(t, "fourth", "private-password")
	f.post(t, serverhttp.LedgerPath, `{"protocol":1,"league":"native.hosted"}`, bearer, "", 200)
	inspection, err := f.join.Rotate(t.Context(), "native.hosted", commish)
	if err != nil {
		t.Fatal(err)
	}
	listed := f.franchises(t, inspection.Code)
	if !listed["franchise.04"].Taken || listed["franchise.04"].ClaimedBy != "fourth" {
		t.Fatalf("claimed franchise = %+v, want claimedBy fourth", listed["franchise.04"])
	}
	if listed["franchise.05"].Taken || listed["franchise.05"].ClaimedBy != "" {
		t.Fatalf("open franchise = %+v, want no claimant", listed["franchise.05"])
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
	released := f.franchises(t, rotated.Code)
	if released["franchise.04"].Taken || released["franchise.04"].ClaimedBy != "" {
		t.Fatalf("released franchise = %+v, want open with no claimant", released["franchise.04"])
	}
	if err := f.join.Revoke(t.Context(), "native.hosted", commish); err != nil {
		t.Fatal(err)
	}
	if _, err := f.join.Franchises(t.Context(), accounts.RawJoin{League: "native.hosted", Code: rotated.Code}); !errors.Is(err, accounts.ErrJoinRefused) {
		t.Fatalf("revoked link = %v", err)
	}
}

func TestJoinLinkConcurrentClaimsConsumeOneSeatOnly(t *testing.T) {
	f := newHostedFixture(t, true)
	code, err := f.join.Rotate(t.Context(), "native.hosted", governance.Principal(f.principal))
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	statuses := make(chan int, 2)
	var attempts sync.WaitGroup
	for i, franchise := range []string{"franchise.04", "franchise.05"} {
		attempts.Add(1)
		go func(franchise, username string) {
			defer attempts.Done()
			<-start
			status, _ := f.send(t, serverhttp.JoinClaimPath, joinBody(code.Code, franchise, username), "", "")
			statuses <- status
		}(franchise, fmt.Sprintf("concurrent.%d", i))
	}
	close(start)
	attempts.Wait()
	close(statuses)

	created, refused := 0, 0
	for status := range statuses {
		switch status {
		case http.StatusCreated:
			created++
		case http.StatusForbidden:
			refused++
		default:
			t.Fatalf("concurrent claim status = %d, want 201 or 403", status)
		}
	}
	if created != 1 || refused != 1 {
		t.Fatalf("concurrent claims: created=%d refused=%d, want one each", created, refused)
	}
	var owners int
	if err := f.pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM accounts WHERE username LIKE 'concurrent.%'`).Scan(&owners); err != nil || owners != 1 {
		t.Fatalf("accounts created by concurrent claims = %d, want one (%v)", owners, err)
	}
	inspection, err := f.join.Rotate(t.Context(), "native.hosted", governance.Principal(f.principal))
	if err != nil {
		t.Fatal(err)
	}
	franchises := f.franchises(t, inspection.Code)
	if franchises["franchise.04"].Taken == franchises["franchise.05"].Taken {
		t.Fatalf("concurrent claims state = %v, %v; want exactly one seat", franchises["franchise.04"].Taken, franchises["franchise.05"].Taken)
	}
}

// The commissioner can also own a team on the same sign-in (11-14 ruling): taking an open franchise records
// the commissioner seating themself, the people read then names both the role and the franchise, and a
// second team seat or a non-commissioner is refused.
func TestCommissionerTakesAnOpenFranchise(t *testing.T) {
	f := newHostedFixture(t, false) // the commissioner holds no team, like CT 257
	commish := governance.Principal(f.principal)
	if err := f.join.Take(t.Context(), "native.hosted", "franchise.02", governance.Principal("acct.00000000-0000-4000-8000-000000000000")); err == nil {
		t.Fatal("a non-commissioner took a franchise")
	}
	if err := f.join.Take(t.Context(), "native.hosted", "franchise.02", commish); err != nil {
		t.Fatal(err)
	}
	if err := f.join.Take(t.Context(), "native.hosted", "franchise.01", commish); err == nil {
		t.Fatal("the commissioner took a second team seat")
	}
	taken := f.lastCommissionerEvent(t)
	if taken.Command.Command != "commish.seat.replace" || taken.Command.Args.People.Franchise != "franchise.02" ||
		taken.Command.Args.People.Principal != f.principal || taken.Principal != f.principal {
		t.Fatalf("take event = %s %+v by %s", taken.Command.Command, taken.Command.Args.People, taken.Principal)
	}
	var people struct {
		YourRole      string `json:"yourRole"`
		YourFranchise string `json:"yourFranchise"`
	}
	bearer := f.login(t, "commissioner", "private-password")
	if err := json.Unmarshal(f.post(t, serverhttp.PeoplePath, `{"protocol":1,"league":"native.hosted"}`, bearer, "", 200), &people); err != nil {
		t.Fatal(err)
	}
	if people.YourRole != "Commissioner" || people.YourFranchise != "franchise.02" {
		t.Fatalf("people = %+v, want the commissioner holding franchise.02", people)
	}
}
