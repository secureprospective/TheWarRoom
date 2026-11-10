package serverhttp_test

import (
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
	// Issued the way the desktop issues them: the catalog's league reads ride along (platform.DelegableCommands).
	commands = append(commands, "native.directory.read", "native.assets.read")
	token, err := f.tokens.Issue(t.Context(), accounts.RawSession{Bearer: human}, accounts.RawTokenScope{League: "native.hosted", Franchise: "franchise.01", Commands: commands, Modes: modes})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// tokenWorkflow is the change as twr sends it: a catalog workflow carrying the server's football digest.
func (f *hostedFixture) tokenWorkflow(t *testing.T, token accounts.AgentToken, raw commissioner.RawCommand, mode string) string {
	t.Helper()
	read := workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, "")
	readBody, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &read})
	if err != nil {
		t.Fatal(err)
	}
	var league serverhttp.WorkflowResponse
	if err := json.Unmarshal(f.post(t, serverhttp.WorkflowPath, string(readBody), token.Bearer, "", 200), &league); err != nil {
		t.Fatal(err)
	}
	pin := league.Football
	args, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	request := workflowRequest(t, raw.Command, mode, string(args), pin)
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &request})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func (f *hostedFixture) events(t *testing.T) int {
	t.Helper()
	events, err := f.history.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	return len(events)
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
	f := newHostedFixture(t, true)
	human, proof := f.directCommissioner(t)
	token := f.agentToken(t, human, []string{"commish.franchise.edit", commissioner.SeasonRollover, "commish.ledger.read"}, "read", "preview", "execute")
	raw := commissioner.RawCommand{League: "native.hosted", Command: "commish.franchise.edit", Args: commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.01", Name: "Caretaker team", Abbreviation: "CT"}}, Expected: f.agentRevision(t, token), Request: "agent-name", Effective: "now", Visibility: "league"}
	base := f.events(t)
	body := f.tokenWorkflow(t, token, raw, "execute")
	f.postConsent(t, body, token.Bearer, "", 409)
	f.revision(t, base)
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
	changed := raw
	changed.Args.People = &commissioner.RawPeople{Franchise: "franchise.01", Name: "Different team", Abbreviation: "CT"}
	f.postConsent(t, f.tokenWorkflow(t, token, changed, "execute"), token.Bearer, consent.Confirmation, 409)
	f.revision(t, base)
	f.postConsent(t, body, token.Bearer, consent.Confirmation, 200)
	f.revision(t, base+1)
	g3 := commissioner.RawCommand{League: raw.League, Command: commissioner.SeasonRollover, Args: commissioner.RawArgs{Rollover: &commissioner.RawRolloverArgs{To: 2027}}, Expected: f.agentRevision(t, token), Request: "roll", Reason: "New season", Effective: "now", Visibility: "league"}
	// Even a forged confirmation cannot authorize a different command.
	f.postConsent(t, f.tokenWorkflow(t, token, g3, "execute"), token.Bearer, consent.Confirmation, 409)
	f.revision(t, base+1)
}

func (f *hostedFixture) postConsent(t *testing.T, body, bearer, consent string, want int) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.server.URL+serverhttp.WorkflowPath, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+bearer)
	if consent != "" {
		request.Header.Set("X-TWR-Human-Confirmation", consent)
	}
	response, err := f.server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != want {
		t.Fatalf("token workflow: %d want %d", response.StatusCode, want)
	}
}

func TestAgentScopeFamiliesAndExpiredRetry(t *testing.T) {
	f := newHostedFixture(t, true)
	human, _ := f.directCommissioner(t)
	seat := commissioner.RawCommand{League: "native.hosted", Command: "commish.seat.replace", Args: commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.01", Principal: f.principal}}, Reason: "Seat the commissioner", Request: "seat", Expected: int64(f.events(t)), Effective: "now", Visibility: "league"}
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
	base := f.events(t)
	raw := commissioner.RawCommand{League: "native.hosted", Command: commissioner.NotePost, Args: commissioner.RawArgs{Note: &commissioner.RawNote{Text: "Expires later"}}, Request: "expiry", Expected: f.agentRevision(t, token), Effective: "now", Visibility: "league"}
	f.postConsent(t, f.tokenWorkflow(t, token, raw, "execute"), token.Bearer, "", 200)
	body := f.tokenWorkflow(t, token, raw, "execute")
	f.now = f.now.Add(accounts.TokenExecuteLifetime)
	f.postConsent(t, body, token.Bearer, "", 401)
	f.revision(t, base+1)
}
