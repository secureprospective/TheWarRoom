package serverhttp_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

const kickoffRosterReason = "Practice free agent is not on this franchise's active roster at kickoff, so cannot start"

func hostedLineupCorrection(t *testing.T, f *hostedFixture) (string, string, string) {
	t.Helper()
	f.skew.Add(int64(26 * time.Hour))
	bearer, proof := f.directCommissioner(t)
	ledgerBody := f.post(t, serverhttp.LedgerPath, `{"protocol":1,"league":"native.hosted"}`, bearer, proof, http.StatusOK)
	var ledger commissioner.Ledger
	if err := json.Unmarshal(ledgerBody, &ledger); err != nil {
		t.Fatal(err)
	}
	raw := commissioner.RawCommand{League: "native.hosted", Command: commissioner.LineupCorrect,
		Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: 2026, Franchise: "franchise.01",
			Week: 1, Starters: []string{"6001"}}}, Expected: ledger.Revision,
		Reason: "Correct the lineup", Request: "kickoff-roster-correction", Effective: "now", Visibility: "league"}
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Command: &raw})
	if err != nil {
		t.Fatal(err)
	}
	return string(body), bearer, proof
}

func TestHostedPreviewRefusesCorrectionOutsideKickoffRoster(t *testing.T) {
	f := newHostedFixture(t, true)
	body, bearer, proof := hostedLineupCorrection(t, f)
	result := f.post(t, serverhttp.PreviewPath, body, bearer, proof, http.StatusOK)
	var preview commissioner.Preview
	if err := json.Unmarshal(result, &preview); err != nil {
		t.Fatal(err)
	}
	if !containsText(preview.Blockers, kickoffRosterReason) {
		t.Fatalf("preview blockers=%q, want %q", preview.Blockers, kickoffRosterReason)
	}
}

func TestHostedExecuteReturnsKickoffRosterRefusal(t *testing.T) {
	f := newHostedFixture(t, true)
	body, bearer, proof := hostedLineupCorrection(t, f)
	result := f.post(t, serverhttp.ExecutePath, body, bearer, proof, http.StatusConflict)
	if got := strings.TrimSpace(string(result)); got != kickoffRosterReason {
		t.Fatalf("execute refusal=%q, want %q", got, kickoffRosterReason)
	}
}

func containsText(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
