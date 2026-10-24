package serverhttp_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/serverclient"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func openStream(t *testing.T, f *hostedFixture, bearer, proof string, status int, leagues ...string) *http.Response {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, f.server.URL+serverhttp.InvalidationPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	league := "native.hosted"
	if len(leagues) > 0 {
		league = leagues[0]
	}
	request.Header.Set("X-TWR-League", league)
	request.Header.Set("Authorization", "Bearer "+bearer)
	request.Header.Set("X-TWR-Recent-Auth", proof)
	response, err := f.server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	if response.StatusCode != status {
		t.Fatalf("stream status=%d want=%d", response.StatusCode, status)
	}
	return response
}

func streamEvents(response *http.Response) <-chan string {
	events := make(chan string, 16)
	go func() {
		defer close(events)
		scanner := bufio.NewScanner(response.Body)
		frame := ""
		for scanner.Scan() {
			frame += scanner.Text() + "\n"
			if scanner.Text() == "" {
				events <- frame
				frame = ""
			}
		}
	}()
	return events
}

func expectInvalidation(t *testing.T, events <-chan string) {
	t.Helper()
	select {
	case event := <-events:
		if event != "event: invalidate\ndata: {\"protocol\":1}\n\n" {
			t.Fatalf("unexpected event: %q", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("missing invalidation")
	}
}
func expectQuiet(t *testing.T, events <-chan string) {
	t.Helper()
	select {
	case event, ok := <-events:
		t.Fatalf("hidden/failed change emitted %q (open=%v)", event, ok)
	case <-time.After(600 * time.Millisecond):
	}
}

func appendStreamNote(t *testing.T, f *hostedFixture, visibility string) {
	t.Helper()
	events, err := f.history.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	raw := commissioner.RawCommand{League: "native.hosted", Command: commissioner.NotePost, Args: commissioner.RawArgs{Note: &commissioner.RawNote{Text: "Stream note"}}, Expected: int64(len(events)), Reason: "Exercise committed visibility", Request: fmt.Sprintf("stream.%d", len(events)), Effective: "now", Visibility: visibility}
	if _, err := f.history.ExecuteCommissioner(t.Context(), raw, f.principal, nil); err != nil {
		t.Fatal(err)
	}
}

func streamOwner(t *testing.T, f *hostedFixture) string {
	t.Helper()
	password := accounts.RawCredentials{Username: "stream.owner", Password: "stream-owner-password"}
	code, err := f.enrollment.Issue(t.Context(), "native.hosted", password.Username)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := f.enrollment.Redeem(t.Context(), accounts.RawEnrollment{Code: code.Code, Username: password.Username, Password: password.Password})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := owner.Principal()
	if err != nil {
		t.Fatal(err)
	}
	seat := commissioner.RawCommand{League: "native.hosted", Command: "commish.seat.replace", Args: commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.01", Principal: string(principal)}}, Expected: 1, Reason: "Assign stream owner", Request: "stream.seat", Effective: "now", Visibility: "league"}
	if _, err := f.history.ExecuteCommissioner(t.Context(), seat, f.principal, nil); err != nil {
		t.Fatal(err)
	}
	return f.login(t, password.Username, password.Password)
}

func TestSSECommittedVisibilityRollbackReconnectAndRevoke(t *testing.T) {
	f := newHostedFixture(t)
	bearer := streamOwner(t, f)
	response := openStream(t, f, bearer, "", 200)
	defer response.Body.Close()
	events := streamEvents(response)
	expectInvalidation(t, events)
	appendPrivateRole(t, f)
	expectQuiet(t, events)
	f.post(t, serverhttp.ExecutePath, `{"protocol":1,"command":{"league":"native.hosted","command":"commish.whatsnew.post","args":{"note":{"text":"Refused"}},"reason":"No authority","expected":1,"request":"refused","effective":"now","visibility":"league"}}`, bearer, "", 409)
	f.revision(t, 3)
	expectQuiet(t, events)
	appendStreamNote(t, f, "league")
	expectInvalidation(t, events)
	// No missed-event replay: a new connection always asks for a fresh authorized read.
	second := openStream(t, f, bearer, "", 200)
	defer second.Body.Close()
	reconnect := streamEvents(second)
	expectInvalidation(t, reconnect)
	denied := openStream(t, f, bearer, "", 429)
	defer denied.Body.Close()
	start := time.Now()
	if err := f.sessions.Revoke(t.Context(), accounts.RawSession{Bearer: bearer}, false); err != nil {
		t.Fatal(err)
	}
	for _, stream := range []<-chan string{events, reconnect} {
		select {
		case event, ok := <-stream:
			if ok {
				t.Fatalf("revoke sent event %q", event)
			}
		case <-time.After(time.Until(start.Add(time.Second))):
			t.Fatal("revoke did not close SSE within one second")
		}
	}
	revoked := openStream(t, f, bearer, "", 401)
	defer revoked.Body.Close()
}

func TestSSERejectsMissingCredentialsAndPrivileges(t *testing.T) {
	f := newHostedFixture(t)
	missing := openStream(t, f, "", "", 401)
	defer missing.Body.Close()
	bearer := f.login(t, "commissioner", "private-password")
	unprivileged := openStream(t, f, bearer, "", 403)
	defer unprivileged.Body.Close()
	proof := f.privilegedProof(t)
	response := openStream(t, f, bearer, proof, 200)
	if response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatal("missing SSE type")
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	f.revision(t, 1)
}

func TestSSEDesktopRereadsAfterDroppedConnection(t *testing.T) {
	f := newHostedFixture(t)
	streamOwner(t, f)
	desktop := desktopFor(t, f)
	if _, err := desktop.Act(t.Context(), serverclient.RawDesktop{Action: "login", Username: "stream.owner", Password: "stream-owner-password"}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	reads := make(chan struct{}, 4)
	done := make(chan error, 1)
	go func() {
		done <- desktop.WatchInvalidations(ctx, func(ctx context.Context) error {
			if _, err := desktop.Act(ctx, serverclient.RawDesktop{Action: "enter"}); err != nil {
				return err
			}
			reads <- struct{}{}
			return nil
		})
	}()
	select {
	case <-reads:
	case <-time.After(2 * time.Second):
		t.Fatal("initial reread missing")
	}
	f.server.CloseClientConnections()
	appendStreamNote(t, f, "league")
	select {
	case <-reads:
	case <-time.After(3 * time.Second):
		t.Fatal("reconnect reread missing")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not stop")
	}
}

type streamBidEnvironment struct{}

func (streamBidEnvironment) BidEnvironment(context.Context, string, time.Time) (commissioner.BidEnvironment, error) {
	return commissioner.BidEnvironment{Season: nativecheck.BidSeason{Week: 5, WeekKnown: true}}, nil
}
func prepareHiddenBid(t *testing.T, f *hostedFixture) func() {
	t.Helper()
	f.history.SetBidEnvironment(streamBidEnvironment{})
	facts := nativecheck.Facts{"5001": {Position: leaguerules.QB, DesignationHeld: true}}
	execute := func(command string, args commissioner.RawArgs, principal, visibility string) {
		t.Helper()
		events, err := f.history.Events(t.Context(), "native.hidden", commissioner.Stream, 0)
		if err != nil {
			t.Fatal(err)
		}
		raw := commissioner.RawCommand{League: "native.hidden", Command: command, Args: args, Expected: int64(len(events)), Request: fmt.Sprintf("bid.%d", len(events)), Reason: "Exercise hidden bid stream", Effective: "now", Visibility: visibility}
		if _, err := f.history.ExecuteCommissionerWithFacts(t.Context(), raw, principal, nil, facts, time.UTC); err != nil {
			t.Fatal(err)
		}
	}
	execute(commissioner.Create, commissioner.RawArgs{Name: "Hidden league", Preset: commissioner.PresetLegacy, Season: 2026, FranchiseCount: 32}, f.principal, "league")
	at := time.Now().UTC()
	seed := &commissioner.RawNativeSnapshot{Season: 2026, RosterAt: at, DraftAt: at, FutureAt: at, Players: []commissioner.NativePlayer{{ID: "0099", Franchise: "franchise.01", Status: domain.RosterActive, Salary: 100000000, ExpirationYear: 2028, ContractStatus: domain.CStatusUFA}}}
	for n := 1; n <= 32; n++ {
		seed.Franchises = append(seed.Franchises, commissioner.NativeFranchise{ID: fmt.Sprintf("franchise.%02d", n), HeldID: fmt.Sprintf("%04d", n), Name: fmt.Sprintf("Club %d", n)})
	}
	execute(commissioner.NativeSeed, commissioner.RawArgs{Native: seed}, f.principal, "league")
	for _, rule := range []struct{ field, value string }{
		{"roster.limits", `{"min":0,"max":100}`}, {"cap.amount", `100000000000`},
		{"calendar.timezone", `"UTC"`}, {"calendar.boundary", `"start_of_day"`},
		{"calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["ufa_bid"]}}`},
		{"bids.ufa", `"blind"`},
	} {
		execute(commissioner.RulesSet, commissioner.RawArgs{Field: rule.field, Value: json.RawMessage(rule.value)}, f.principal, "league")
	}
	execute("commish.seat.replace", commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.02", Principal: "hidden.owner"}}, f.principal, "league")
	return func() {
		execute(commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5001", Franchise: "franchise.02", Salary: 500000000, Years: 1}}, "hidden.owner", "private")
	}
}

func appendPrivateRole(t *testing.T, f *hostedFixture) {
	t.Helper()
	raw := commissioner.RawCommand{League: "native.hosted", Command: "commish.role.grant", Args: commissioner.RawArgs{People: &commissioner.RawPeople{Principal: "private.deputy", Role: governance.CoCommissioner}}, Expected: 2, Reason: "Assign private deputy", Request: "private.role", Effective: "now", Visibility: "private"}
	if _, err := f.history.ExecuteCommissioner(t.Context(), raw, f.principal, nil); err != nil {
		t.Fatal(err)
	}
}
func TestSSEHiddenBidHasNoActivityCursor(t *testing.T) {
	f := newHostedFixture(t)
	hiddenBid := prepareHiddenBid(t, f)
	bearer := f.login(t, "commissioner", "private-password")
	proof := f.privilegedProof(t)
	response := openStream(t, f, bearer, proof, 200, "native.hidden")
	defer response.Body.Close()
	events := streamEvents(response)
	expectInvalidation(t, events)
	hiddenBid()
	expectQuiet(t, events)
}

func TestSSEOutlivesOrdinaryRequestBudget(t *testing.T) {
	f := newHostedFixture(t)
	bearer := streamOwner(t, f)
	response := openStream(t, f, bearer, "", 200)
	defer response.Body.Close()
	events := streamEvents(response)
	expectInvalidation(t, events)
	select {
	case event, ok := <-events:
		t.Fatalf("idle stream ended or emitted %q (open=%v)", event, ok)
	case <-time.After(5500 * time.Millisecond):
	}
	appendStreamNote(t, f, "league")
	expectInvalidation(t, events)
}
