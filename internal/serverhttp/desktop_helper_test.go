package serverhttp_test

import (
	"crypto/x509"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverclient"
)

// The whole helper loop a commissioner runs: connect a helper from the desktop, let it preview a
// bigger change, confirm that exact change, let it record it, then disconnect it.
func TestDesktopConnectsConfirmsAndRevokesAnAIHelper(t *testing.T) {
	f := newHostedFixture(t)
	desktop := desktopFor(t, f)
	desktop.SetHelperCommands([]string{"commish.franchise.edit", "commish.ledger.read"})
	credentials := serverclient.RawDesktop{Action: "login", Username: "commissioner", Password: "private-password"}
	if _, err := desktop.Act(t.Context(), credentials); err != nil {
		t.Fatal(err)
	}
	issue := serverclient.RawDesktop{Action: "helper-issue", Helper: "record", Franchise: "franchise.01"}
	if _, err := desktop.Act(t.Context(), issue); err == nil || !strings.Contains(err.Error(), "authenticator") {
		t.Fatalf("helper without recent verification: %v", err)
	}
	credentials.Action = "totp-begin"
	setup, err := desktop.Act(t.Context(), credentials)
	if err != nil {
		t.Fatal(err)
	}
	credentials.Action, credentials.Code = "totp-confirm", testTOTP(t, setup.Seed, f.now)
	if _, err := desktop.Act(t.Context(), credentials); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(30 * time.Second)
	credentials.Action, credentials.Code = "totp-step-up", testTOTP(t, setup.Seed, f.now)
	if _, err := desktop.Act(t.Context(), credentials); err != nil {
		t.Fatal(err)
	}
	if _, err := desktop.Act(t.Context(), issue); err == nil || !strings.Contains(err.Error(), "league access first") {
		t.Fatalf("helper before entering the league: %v", err)
	}
	entered, err := desktop.Act(t.Context(), serverclient.RawDesktop{Action: "enter"})
	if err != nil || len(entered.Seats) == 0 {
		t.Fatalf("enter: %+v %v", entered, err)
	}
	connected, err := desktop.Act(t.Context(), issue)
	if err != nil || connected.Connection == "" || connected.HelperID == "" {
		t.Fatalf("connect helper: %+v %v", connected, err)
	}
	id, bearer, err := serverclient.ParseHelperLink(connected.Connection)
	if err != nil || id != connected.HelperID {
		t.Fatalf("connection line: %v", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(f.server.Certificate())
	agent, err := serverclient.NewAgent(f.server.URL, "native.hosted", bearer, roots)
	if err != nil {
		t.Fatal(err)
	}
	change, err := agent.Prepare(t.Context(), commissioner.RawCommand{Command: "commish.franchise.edit",
		Args: commissioner.RawArgs{People: &commissioner.RawPeople{
			Franchise: "franchise.01", Name: "Helper team", Abbreviation: "HT"}},
		Reason: "Use the new team name", Request: "helper-name", Effective: "now", Visibility: "league"})
	if err != nil {
		t.Fatal(err)
	}
	previewBody, err := agent.Dispatch(t.Context(), change, "preview", "")
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if _, err := agent.Dispatch(t.Context(), change, "execute", ""); !errors.Is(err, serverclient.ErrRefused) {
		t.Fatalf("unconfirmed G2 = %v", err)
	}
	request, err := serverclient.EncodeConfirmRequest(id, change)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := desktop.Act(t.Context(), serverclient.RawDesktop{Action: "helper-check", Change: request})
	var preview commissioner.Preview
	if err := json.Unmarshal(previewBody, &preview); err != nil {
		t.Fatal(err)
	}
	if err != nil || checked.Confirmation != "" || preview.Question == "" ||
		!strings.Contains(checked.HelperPreview, preview.Question) ||
		!strings.Contains(checked.HelperPreview,
			"Franchise 1 name changes from “Franchise 1 (F1)” to “Helper team (HT)”.") ||
		!strings.Contains(checked.HelperPreview, "Helper: "+id) ||
		!strings.Contains(checked.HelperPreview, "Reason: “Use the new team name”") ||
		!strings.Contains(checked.HelperPreview, "1 franchise affected: Franchise 1") {
		t.Fatalf("check: %+v %v", checked, err)
	}
	confirmed, err := desktop.Act(t.Context(), serverclient.RawDesktop{Action: "helper-confirm", Change: request})
	if err != nil || confirmed.Confirmation == "" {
		t.Fatalf("confirm: %+v %v", confirmed, err)
	}
	if _, err := agent.Dispatch(t.Context(), change, "execute", confirmed.Confirmation); err != nil {
		t.Fatalf("confirmed execute: %v", err)
	}
	f.revision(t, 2)
	change.Request = "stale-helper-name"
	request, err = serverclient.EncodeConfirmRequest(id, change)
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"helper-check", "helper-confirm"} {
		stale, err := desktop.Act(t.Context(), serverclient.RawDesktop{Action: action, Change: request})
		if err == nil || stale.Confirmation != "" || !strings.Contains(strings.ToLower(err.Error()), "reread") {
			t.Fatalf("stale %s: %+v %v", action, stale, err)
		}
	}
	f.revision(t, 2)

	if _, err := desktop.Act(t.Context(), serverclient.RawDesktop{Action: "helper-revoke", Code: id}); err != nil {
		t.Fatal(err)
	}
	if _, err := agent.Ledger(t.Context()); err == nil || !strings.Contains(err.Error(), "revoked") {
		t.Fatalf("revoked helper read the ledger: %v", err)
	}
}
