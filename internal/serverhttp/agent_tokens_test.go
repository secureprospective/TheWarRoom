package serverhttp_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestHumanIssuesAndRevokesAgentTokenOverHTTPS(t *testing.T) {
	f := newHostedFixture(t)
	human, proof := f.directCommissioner(t)
	scope := `{"protocol":1,"scope":{"league":"native.hosted","franchise":"franchise.01","commands":["` + commissioner.NotePost + `","commish.ledger.read"],"modes":["preview","read"]}}`

	// Recent authenticator proof is required, and an agent token cannot mint another.
	f.post(t, serverhttp.AgentTokenPath, scope, human, "", 403)
	body := f.post(t, serverhttp.AgentTokenPath, scope, human, proof, 200)
	var issued serverhttp.AgentTokenResponse
	if err := json.Unmarshal(body, &issued); err != nil || issued.Bearer == "" || issued.ID == "" || issued.ExpiresAt == nil {
		t.Fatalf("issued %v", err)
	}
	if strings.Contains(issued.String(), issued.Bearer) || strings.Contains(issued.GoString(), issued.Bearer) {
		t.Fatal("token response leaks its secret when printed")
	}
	f.post(t, serverhttp.AgentTokenPath, scope, issued.Bearer, "", 401)
	f.post(t, serverhttp.LedgerPath, `{"protocol":1,"league":"native.hosted"}`, issued.Bearer, "", 200)

	// Undelegable commands and unknown fields are refused before anything is issued.
	f.post(t, serverhttp.AgentTokenPath, strings.Replace(scope, commissioner.NotePost, "commish.role.grant", 1), human, proof, 400)
	f.post(t, serverhttp.AgentTokenPath, strings.Replace(scope, `"protocol":1`, `"protocol":1,"bearer":"x"`, 1), human, proof, 400)

	revoke := `{"protocol":1,"id":"` + issued.ID + `"}`
	f.post(t, serverhttp.AgentTokenRevokePath, revoke, issued.Bearer, "", 401)
	f.post(t, serverhttp.AgentTokenRevokePath, revoke, human, "", 200)
	f.post(t, serverhttp.LedgerPath, `{"protocol":1,"league":"native.hosted"}`, issued.Bearer, "", 401)
}
