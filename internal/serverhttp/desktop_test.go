package serverhttp_test

import (
	"crypto/x509"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverclient"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func desktopFor(t *testing.T, f *hostedFixture) *serverclient.Desktop {
	t.Helper()
	roots := x509.NewCertPool()
	roots.AddCert(f.server.Certificate())
	desktop, err := serverclient.NewDesktop(f.server.URL, "native.hosted", roots)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(desktop.Close)
	return desktop
}

func TestDesktopNativeEnrollmentAssignedFranchise(t *testing.T) {
	f := newHostedFixture(t)
	desktop := desktopFor(t, f)
	code, err := f.enrollment.Issue(t.Context(), "native.hosted", "desktop.owner")
	if err != nil {
		t.Fatal(err)
	}
	request := serverclient.RawDesktop{Action: "enroll", Username: "desktop.owner", Password: "desktop-owner-password", Code: code.Code}
	view, err := desktop.Act(t.Context(), request)
	if err != nil || !strings.Contains(view.Message, "Account created") {
		t.Fatalf("enroll: %+v %v", view, err)
	}
	if _, err := desktop.Act(t.Context(), request); err == nil || !strings.Contains(err.Error(), "48 hours") {
		t.Fatalf("used invite recovery: %v", err)
	}
	// Enrollment itself cannot grant league membership or a franchise.
	request.Action, request.Code = "login", ""
	if _, err := desktop.Act(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	if _, err := desktop.Act(t.Context(), serverclient.RawDesktop{Action: "enter"}); err == nil {
		t.Fatal("enrollment granted a league seat")
	}
	account, err := f.accounts.Credential(t.Context(), "desktop.owner")
	if err != nil {
		t.Fatal(err)
	}
	principal, err := account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	for i, raw := range []commissioner.RawCommand{
		{Command: "commish.franchise.edit", Args: commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.01", Name: "Owner club", Abbreviation: "OWN"}}},
		{Command: "commish.seat.replace", Args: commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.01", Principal: string(principal)}}},
	} {
		raw.League, raw.Reason, raw.Request, raw.Effective, raw.Visibility = "native.hosted", "Assign the enrolled owner", raw.Command, "now", "league"
		raw.Expected = int64(i + 1)
		if _, err := f.history.ExecuteCommissioner(t.Context(), raw, f.principal, nil); err != nil {
			t.Fatal(err)
		}
	}
	view, err = desktop.Act(t.Context(), serverclient.RawDesktop{Action: "enter"})
	if err != nil || view.Franchise != "franchise.01" || view.FranchiseName != "Owner club" || view.Role != "Owner" {
		t.Fatalf("assigned franchise: %+v %v", view, err)
	}
	encoded, err := json.Marshal(view)
	if err != nil || strings.Contains(string(encoded), "bearer") || strings.Contains(string(encoded), "revision") || strings.Contains(string(encoded), "password") {
		t.Fatalf("webview secret: %s %v", encoded, err)
	}
	f.revision(t, 3)
	remove := commissioner.RawCommand{League: "native.hosted", Command: "commish.seat.remove",
		Args:     commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.01"}},
		Expected: 3, Reason: "Remove the test seat", Request: "remove", Effective: "now", Visibility: "league"}
	if _, err := f.history.ExecuteCommissioner(t.Context(), remove, f.principal, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := desktop.Act(t.Context(), serverclient.RawDesktop{Action: "enter"}); err == nil {
		t.Fatal("removed owner retained league access")
	}
	f.revision(t, 4)
	if _, err := desktop.Act(t.Context(), serverclient.RawDesktop{Action: "forget"}); err != nil {
		t.Fatal(err)
	}
	if _, err := desktop.Act(t.Context(), serverclient.RawDesktop{Action: "enter"}); err == nil {
		t.Fatal("forgotten session revived")
	}
	f.revision(t, 4)
}

func TestDesktopExpiredInviteAndPrivilegedAuthenticator(t *testing.T) {
	f := newHostedFixture(t)
	desktop := desktopFor(t, f)
	code, err := f.enrollment.Issue(t.Context(), "native.hosted", "late.owner")
	if err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(accounts.EnrollmentLifetime)
	if _, err := desktop.Act(t.Context(), serverclient.RawDesktop{Action: "enroll", Username: "late.owner", Password: "owner-password", Code: code.Code}); err == nil || !strings.Contains(err.Error(), "personally handed") {
		t.Fatalf("expired invite: %v", err)
	}
	credentials := serverclient.RawDesktop{Action: "login", Username: "commissioner", Password: "private-password"}
	if _, err := desktop.Act(t.Context(), credentials); err != nil {
		t.Fatal(err)
	}
	if _, err := desktop.Act(t.Context(), serverclient.RawDesktop{Action: "enter"}); err == nil || !strings.Contains(err.Error(), "authenticator") {
		t.Fatalf("missing privileged factors: %v", err)
	}
	credentials.Action = "totp-begin"
	setup, err := desktop.Act(t.Context(), credentials)
	if err != nil || setup.Seed == "" {
		t.Fatalf("setup: %v", err)
	}
	credentials.Action, credentials.Code = "totp-confirm", testTOTP(t, setup.Seed, f.now)
	if _, err := desktop.Act(t.Context(), credentials); err != nil {
		t.Fatal(err)
	}
	credentials.Action = "totp-step-up"
	if _, err := desktop.Act(t.Context(), credentials); err == nil || !strings.Contains(err.Error(), "fresh code") {
		t.Fatalf("replay recovery: %v", err)
	}
	f.now = f.now.Add(30 * time.Second)
	credentials.Code = testTOTP(t, setup.Seed, f.now)
	if _, err := desktop.Act(t.Context(), credentials); err != nil {
		t.Fatal(err)
	}
	view, err := desktop.Act(t.Context(), serverclient.RawDesktop{Action: "enter"})
	if err != nil || view.Name != "Hosted test" {
		t.Fatalf("privileged entry: %+v %v", view, err)
	}
	f.now = f.now.Add(accounts.RecentAuthLifetime)
	if _, err := desktop.Act(t.Context(), serverclient.RawDesktop{Action: "enter"}); err == nil {
		t.Fatal("expired recent auth admitted")
	}
	// The new read door uses the same missing/spoofed-session policy as every hosted route.
	f.post(t, serverhttp.PeoplePath, `{"protocol":1,"league":"native.hosted"}`, "", "", 401)
	f.revision(t, 1)
}
