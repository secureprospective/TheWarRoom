package serverhttp_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
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

func TestAgentPrivatePlayerCommandsUseOnlyIssuerSeat(t *testing.T) {
	f := newHostedFixture(t, true)
	human, _ := f.directCommissioner(t)
	commands := []string{leagueservice.PlayerNoteRead, leagueservice.PlayerNoteSet, leagueservice.PlayerWatchRead,
		leagueservice.PlayerWatchSet, leagueservice.PlayerWatchOrder}
	token := f.agentToken(t, human, commands, "read", "execute")
	invoke := func(command, mode, arguments string) serverhttp.WorkflowResponse {
		t.Helper()
		workflow := workflowRequest(t, command, mode, arguments, "")
		body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &workflow})
		if err != nil {
			t.Fatal(err)
		}
		var response serverhttp.WorkflowResponse
		if err := json.Unmarshal(f.post(t, serverhttp.WorkflowPath, string(body), token.Bearer, "", 200), &response); err != nil {
			t.Fatal(err)
		}
		return response
	}
	invoke(leagueservice.PlayerNoteSet, "execute", `{"league":"native.hosted","player":"0101","text":"agent private note"}`)
	invoke(leagueservice.PlayerWatchSet, "execute", `{"league":"native.hosted","player":"0101","watched":true}`)
	invoke(leagueservice.PlayerWatchSet, "execute", `{"league":"native.hosted","player":"0102","watched":true}`)
	invoke(leagueservice.PlayerWatchOrder, "execute", `{"league":"native.hosted","player":"0102","with":"0101"}`)
	response := invoke(leagueservice.PlayerNoteRead, "read", `{"league":"native.hosted","player":"0101"}`)
	var notes leagueservice.PlayerNotes
	if err := json.Unmarshal(response.Data, &notes); err != nil || notes.Count != 1 || notes.Notes[0].Text != "agent private note" || !notes.Watched {
		t.Fatalf("agent private reading = %+v, %v", notes, err)
	}
	response = invoke(leagueservice.PlayerWatchRead, "read", `{"league":"native.hosted"}`)
	var watches leagueservice.PlayerWatchList
	if err := json.Unmarshal(response.Data, &watches); err != nil || len(watches.Players) != 2 ||
		watches.Players[0] != "0102" || watches.Players[1] != "0101" {
		t.Fatalf("agent watch reading = %+v, %v", watches, err)
	}
	other := hostedWorkflowClient{t: t, f: f, bearer: f.login(t, "owner", "private-password")}
	other.invoke(leagueservice.PlayerWatchSet, "execute", []byte(`{"league":"native.hosted","player":"0202","watched":true}`), 200)
	invoke(leagueservice.PlayerWatchSet, "execute", `{"league":"native.hosted","player":"0202","watched":false}`)
	response = other.invoke(leagueservice.PlayerWatchRead, "read", []byte(`{"league":"native.hosted"}`), 200)
	if err := json.Unmarshal(response.Data, &watches); err != nil || len(watches.Players) != 1 || watches.Players[0] != "0202" {
		t.Fatalf("team A token changed team B's targets: %+v, %v", watches, err)
	}

	if _, err := f.tokens.Issue(t.Context(), accounts.RawSession{Bearer: human}, accounts.RawTokenScope{
		League: "native.hosted", Franchise: "franchise.02", Commands: commands, Modes: []string{"read", "execute"},
	}); err == nil {
		t.Fatal("commissioner delegated private player data for another franchise")
	}
}

func TestContractDecisionTokenReadAndWriteStayOnItsFranchise(t *testing.T) {
	f := newHostedFixture(t, true)
	human, _ := f.directCommissioner(t)
	commands := []string{leagueservice.ContractDecisionsRead, commissioner.ContractDecisionSet,
		"native.assets.read", "native.directory.read"}
	token, err := f.tokens.Issue(t.Context(), accounts.RawSession{Bearer: human}, accounts.RawTokenScope{
		League: "native.hosted", Franchise: "franchise.01", Commands: commands,
		Modes: []string{"read", "preview", "execute"},
	})
	if err != nil {
		t.Fatal(err)
	}
	read := workflowRequest(t, leagueservice.ContractDecisionsRead, "read",
		`{"league":"native.hosted","viewerZone":"UTC"}`, "")
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &read})
	if err != nil {
		t.Fatal(err)
	}
	var response serverhttp.WorkflowResponse
	if err := json.Unmarshal(f.post(t, serverhttp.WorkflowPath, string(body), token.Bearer, "", 200), &response); err != nil {
		t.Fatal(err)
	}
	var reading struct {
		Franchise string `json:"franchise"`
	}
	if err := json.Unmarshal(response.Data, &reading); err != nil || reading.Franchise != "franchise.01" {
		t.Fatalf("private decision read used the wrong seat: %+v %v", reading, err)
	}

	before := f.events(t)
	raw, err := (commissioner.RawCommand{League: "native.hosted", Command: commissioner.ContractDecisionSet,
		Args: commissioner.RawArgs{ContractDecision: &commissioner.RawContractDecision{Franchise: "franchise.02",
			Player: "0102", Action: "rfa_tender", Level: 1}},
		Expected: int64(before), Request: "cross-team-contract-decision", Reason: "Try another team",
		Effective: "now", Visibility: "private"}).Prepared()
	if err != nil {
		t.Fatal(err)
	}
	body = []byte(f.tokenWorkflow(t, token, raw, "preview"))
	f.post(t, serverhttp.WorkflowPath, string(body), token.Bearer, "", 409)
	if got := f.events(t); got != before {
		t.Fatalf("cross-team decision wrote %d events", got-before)
	}
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
	f.advanceTime(accounts.TokenExecuteLifetime)
	f.postConsent(t, body, token.Bearer, "", 401)
	f.revision(t, base+1)
}
