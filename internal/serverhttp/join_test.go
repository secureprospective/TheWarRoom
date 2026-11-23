package serverhttp_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leagueservice"

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

// The commissioner's join link lists franchises; each claim seats an ordinary owner and the same link
// stays live for the next open seat (ruling 2026-11-23). Taken picks, wrong, rotated and revoked links
// are refused; release reopens a franchise.
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
	first := f.lastCommissionerEvent(t)
	if first.Command.Command != "commish.seat.replace" || first.Command.Args.People.Franchise != "franchise.04" || first.Principal != f.principal {
		t.Fatalf("claim event = %s %+v by %s", first.Command.Command, first.Command.Args.People, first.Principal)
	}
	// Person B opens the same link after A claimed: B sees the league, A's seat taken, the rest open.
	// (Service calls here keep the test under the doors' 10-a-minute limit.)
	seen, err := f.join.Franchises(t.Context(), accounts.RawJoin{League: "native.hosted", Code: code.Code})
	if err != nil {
		t.Fatalf("second holder opening the link after a claim: %v", err)
	}
	for _, franchise := range seen {
		if franchise.ID == "franchise.05" && franchise.Taken || franchise.ID == "franchise.04" && franchise.ClaimedBy != "fourth" {
			t.Fatalf("second holder sees %+v", franchise)
		}
	}
	if _, err := f.join.Claim(t.Context(), accounts.RawJoinClaim{RawJoin: accounts.RawJoin{League: "native.hosted", Code: code.Code},
		Franchise: "franchise.04", Username: "thief2", Password: "private-password"}); !errors.Is(err, accounts.ErrFranchiseTaken) {
		t.Fatalf("second holder took a claimed seat: %v", err)
	}
	f.post(t, serverhttp.JoinClaimPath, joinBody(code.Code, "franchise.05", "fifth"), "", "", http.StatusCreated)
	claimed := f.lastCommissionerEvent(t)
	if claimed.Command.Args.People.Franchise != "franchise.05" || !strings.HasPrefix(claimed.Command.Request, "join.claim.") {
		t.Fatalf("second claim event = %+v request %q", claimed.Command.Args.People, claimed.Command.Request)
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
	if listed["franchise.06"].Taken || listed["franchise.06"].ClaimedBy != "" {
		t.Fatalf("open franchise = %+v, want no claimant", listed["franchise.06"])
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

// Two holders of the one link race for the same franchise: one owner, one 409, one account.
func TestJoinLinkConcurrentClaimsConsumeOneSeatOnly(t *testing.T) {
	f := newHostedFixture(t, true)
	code, err := f.join.Rotate(t.Context(), "native.hosted", governance.Principal(f.principal))
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	statuses := make(chan int, 2)
	var attempts sync.WaitGroup
	for i, franchise := range []string{"franchise.04", "franchise.04"} {
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
		case http.StatusConflict:
			refused++
		default:
			t.Fatalf("concurrent claim status = %d, want 201 or 409", status)
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
	if !franchises["franchise.04"].Taken || !strings.HasPrefix(franchises["franchise.04"].ClaimedBy, "concurrent.") {
		t.Fatalf("concurrent claims state = %+v; want franchise.04 owned by the winner", franchises["franchise.04"])
	}
}

// Once every franchise has an owner the one link says so plainly, both on opening and on claiming,
// and creates no account. Releasing a seat lets the same link fill it again.
func TestJoinLinkSaysLeagueIsFull(t *testing.T) {
	f := newHostedFixture(t, true)
	commish := governance.Principal(f.principal)
	code, err := f.join.Rotate(t.Context(), "native.hosted", commish)
	if err != nil {
		t.Fatal(err)
	}
	for id, taken := range f.openFranchises(t, code.Code) {
		if taken {
			continue
		}
		claim := accounts.RawJoinClaim{RawJoin: accounts.RawJoin{League: "native.hosted", Code: code.Code}, Franchise: id,
			Username: "owner." + strings.TrimPrefix(id, "franchise."), Password: "private-password"}
		if _, err := f.join.Claim(t.Context(), claim); err != nil {
			t.Fatalf("claim %s with the shared link: %v", id, err)
		}
	}
	const full = "Every franchise in this league already has an owner. Ask your commissioner if you think this is wrong."
	if body := string(f.post(t, serverhttp.JoinFranchisesPath, joinBody(code.Code, "", ""), "", "", http.StatusGone)); strings.TrimSpace(body) != full {
		t.Fatalf("full league listing = %q", body)
	}
	if body := string(f.post(t, serverhttp.JoinClaimPath, joinBody(code.Code, "franchise.04", "late"), "", "", http.StatusGone)); strings.TrimSpace(body) != full {
		t.Fatalf("full league claim = %q", body)
	}
	var late int
	if err := f.pools.Read().QueryRowContext(t.Context(), `SELECT count(*) FROM accounts WHERE username = 'late'`).Scan(&late); err != nil || late != 0 {
		t.Fatalf("full league created an account: count=%d err=%v", late, err)
	}
	if err := f.join.Release(t.Context(), "native.hosted", "franchise.04", commish); err != nil {
		t.Fatal(err)
	}
	f.post(t, serverhttp.JoinClaimPath, joinBody(code.Code, "franchise.04", "late"), "", "", http.StatusCreated)
}

// The commissioner can also own a team on the same sign-in (11-14 ruling): taking an open franchise records
// the commissioner seating themself, the people read then names both the role and the franchise, and a
// second team seat or a non-commissioner is refused.
func TestCommissionerTakesAnOpenFranchise(t *testing.T) {
	f := newHostedFixture(t, false) // the commissioner holds no team, like CT 257
	commish := governance.Principal(f.principal)
	// The console's no-login door seats whoever holds the commissioner role, found from the league itself.
	if found, err := (leagueservice.JoinSeats{Store: f.history}).Commissioner(t.Context(), "native.hosted"); err != nil ||
		found != commish {
		t.Fatalf("league commissioner = %q, %v; want %q", found, err, commish)
	}
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

// A join-link owner must read their own lineup; another franchise's lineup stays refused.
func TestJoinedOwnerReadsOwnLineup(t *testing.T) {
	f := newHostedFixture(t, true)
	commish := governance.Principal(f.principal)
	code, err := f.join.Rotate(t.Context(), "native.hosted", commish)
	if err != nil {
		t.Fatal(err)
	}
	f.post(t, serverhttp.JoinClaimPath, joinBody(code.Code, "franchise.04", "fourth"), "", "", http.StatusCreated)
	bearer := f.login(t, "fourth", "private-password")
	read := func(franchise string) (int, string) {
		t.Helper()
		raw := workflowRequest(t, "lineup.read", "read", fmt.Sprintf(`{"league":"native.hosted","franchise":%q,"week":1,"at":""}`, franchise), "")
		body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
		if err != nil {
			t.Fatal(err)
		}
		status, result := f.send(t, serverhttp.WorkflowPath, string(body), bearer, "")
		return status, string(result)
	}
	if status, result := read("franchise.04"); status != http.StatusOK || strings.Contains(result, "private to its team") {
		t.Fatalf("own lineup: status=%d body=%s", status, result)
	}
	if status, result := read("franchise.05"); status == http.StatusOK {
		t.Fatalf("other lineup readable: body=%s", result)
	}
}
