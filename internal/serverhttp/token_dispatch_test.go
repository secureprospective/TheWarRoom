package serverhttp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func (f *hostedFixture) agentToken(t *testing.T, human string, commands []string, modes ...string) accounts.AgentToken {
	t.Helper()
	token, err := f.tokens.Issue(t.Context(), accounts.RawSession{Bearer: human}, accounts.RawTokenScope{League: "native.hosted", Franchise: "franchise.01", Commands: commands, Modes: modes})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestAgentSharedDispatchDefaultsToPreviewAndAuditsAppend(t *testing.T) {
	f := newHostedFixture(t)
	human, proof := f.directCommissioner(t)
	token := f.agentToken(t, human, []string{commissioner.NotePost, "commish.ledger.read"}, "preview", "execute", "read")
	raw := commissioner.RawCommand{League: "native.hosted", Command: commissioner.NotePost, Args: commissioner.RawArgs{Note: &commissioner.RawNote{Text: "Agent's league note"}}, Request: "agent-note", Expected: 1, Effective: "now", Visibility: "league"}
	raw.Expected = f.agentRevision(t, token)
	body := tokenCommandBody(t, raw)
	f.post(t, serverhttp.DispatchPath, body, token.Bearer, "", 200)
	f.revision(t, 1)
	f.post(t, serverhttp.ExecutePath, body, token.Bearer, "", 200)
	f.post(t, serverhttp.ExecutePath, body, token.Bearer, "", 200)
	f.revision(t, 2)
	events, err := f.history.Events(t.Context(), raw.League, commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	recorded := events[1].Payload.Commissioner
	if recorded.TokenAuditID != token.ID || recorded.Principal != f.principal {
		t.Fatalf("missing trusted token audit: %+v", recorded)
	}
	ledger := f.post(t, serverhttp.LedgerPath, `{"protocol":1,"league":"native.hosted"}`, human, proof, 200)
	if !bytes.Contains(ledger, []byte("via AI/CLI, acting for "+f.principal)) || bytes.Contains(ledger, []byte(token.Bearer)) {
		t.Fatal("ledger lost attribution or leaked bearer")
	}
	if err := f.tokens.Revoke(t.Context(), accounts.RawSession{Bearer: human}, token.ID); err != nil {
		t.Fatal(err)
	}
	f.post(t, serverhttp.ExecutePath, body, token.Bearer, "", 401)
	f.revision(t, 2)
}

func tokenCommandBody(t *testing.T, raw commissioner.RawCommand) string {
	t.Helper()
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Command: &raw})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func (f *hostedFixture) agentRevision(t *testing.T, token accounts.AgentToken) int64 {
	t.Helper()
	body := f.post(t, serverhttp.LedgerPath, `{"protocol":1,"league":"native.hosted"}`, token.Bearer, "", 200)
	var ledger commissioner.Ledger
	if err := json.Unmarshal(body, &ledger); err != nil {
		t.Fatal(err)
	}
	return ledger.Revision
}

func TestAgentG2G3RequireExactHumanConsent(t *testing.T) {
	f := newHostedFixture(t)
	human, proof := f.directCommissioner(t)
	token := f.agentToken(t, human, []string{"commish.franchise.edit", commissioner.SeasonRollover, "commish.ledger.read"}, "read", "preview", "execute")
	raw := commissioner.RawCommand{League: "native.hosted", Command: "commish.franchise.edit", Args: commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.01", Name: "Caretaker team", Abbreviation: "CT"}}, Expected: f.agentRevision(t, token), Request: "agent-name", Effective: "now", Visibility: "league"}
	body := tokenCommandBody(t, raw)
	f.post(t, serverhttp.ExecutePath, body, token.Bearer, proof, 409)
	f.revision(t, 1)
	consentBody, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Command: &raw, TokenID: token.ID})
	if err != nil {
		t.Fatal(err)
	}
	f.post(t, serverhttp.ConfirmPath, string(consentBody), token.Bearer, proof, 401)
	f.post(t, serverhttp.ConfirmPath, string(consentBody), human, "", 403)
	result := f.post(t, serverhttp.ConfirmPath, string(consentBody), human, proof, 200)
	var consent serverhttp.ConfirmationResponse
	if err := json.Unmarshal(result, &consent); err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(body, "Caretaker team", "Different team", 1)
	f.postConsent(t, changed, token.Bearer, consent.Confirmation, 409)
	f.revision(t, 1)
	f.postConsent(t, body, token.Bearer, consent.Confirmation, 200)
	f.revision(t, 2)
	g3 := commissioner.RawCommand{League: raw.League, Command: commissioner.SeasonRollover, Args: commissioner.RawArgs{Rollover: &commissioner.RawRolloverArgs{To: 2027}}, Expected: f.agentRevision(t, token), Request: "roll", Reason: "New season", Effective: "now", Visibility: "league"}
	// Even a forged confirmation cannot authorize a different command.
	f.postConsent(t, tokenCommandBody(t, g3), token.Bearer, consent.Confirmation, 409)
	f.revision(t, 2)
}

func (f *hostedFixture) postConsent(t *testing.T, body, bearer, consent string, want int) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.server.URL+serverhttp.ExecutePath, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+bearer)
	request.Header.Set("X-TWR-Human-Confirmation", consent)
	response, err := f.server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != want {
		t.Fatalf("consent execute: %d want %d", response.StatusCode, want)
	}
}

func TestAgentScopeFamiliesAndExpiredRetry(t *testing.T) {
	f := newHostedFixture(t)
	human, _ := f.directCommissioner(t)
	seat := commissioner.RawCommand{League: "native.hosted", Command: "commish.seat.replace", Args: commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.01", Principal: f.principal}}, Reason: "Seat the commissioner", Request: "seat", Expected: 1, Effective: "now", Visibility: "league"}
	if _, err := f.history.ExecuteCommissioner(t.Context(), seat, f.principal, nil); err != nil {
		t.Fatal(err)
	}
	families := []string{commissioner.NotePost, commissioner.RulesSet, commissioner.CapAdjust, commissioner.ScoreFranchise, commissioner.LineupSet, commissioner.DraftPick, commissioner.WaiverClaim, commissioner.BidPlace, commissioner.TradeOffer, commissioner.ScheduleEdit, commissioner.BracketSeed, commissioner.CorrectRoster, commissioner.WindowSet, commissioner.SeasonRollover}
	token := f.agentToken(t, human, append(families, "commish.ledger.read"), "read", "preview", "execute")
	gate := f.tokens.Admission(token.RawSession)
	for _, command := range families {
		for name, access := range map[string]commissioner.Access{
			"league":    {Principal: f.principal, League: "native.other", Operation: command, Mode: "execute", Franchise: "franchise.01"},
			"franchise": {Principal: f.principal, League: "native.hosted", Operation: command, Mode: "execute", Franchise: "franchise.02"},
			"command":   {Principal: f.principal, League: "native.hosted", Operation: commissioner.NoticePublish, Mode: "execute", Franchise: "franchise.01"},
		} {
			t.Run(fmt.Sprintf("%s/%s", command, name), func(t *testing.T) {
				if err := gate.Admit(t.Context(), access); err == nil {
					t.Fatal("scope escape admitted")
				}
			})
		}
	}
	f.revision(t, 2)
	raw := commissioner.RawCommand{League: "native.hosted", Command: commissioner.NotePost, Args: commissioner.RawArgs{Note: &commissioner.RawNote{Text: "Expires later"}}, Request: "expiry", Expected: f.agentRevision(t, token), Effective: "now", Visibility: "league"}
	body := tokenCommandBody(t, raw)
	f.post(t, serverhttp.ExecutePath, body, token.Bearer, "", 200)
	f.now = f.now.Add(accounts.TokenExecuteLifetime)
	f.post(t, serverhttp.ExecutePath, body, token.Bearer, "", 401)
	f.revision(t, 3)
}
