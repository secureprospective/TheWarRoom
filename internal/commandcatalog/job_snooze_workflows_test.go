package catalog

import (
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
)

func TestJobSnoozeWorkflowsUseOnlyTheAuthenticatedAccount(t *testing.T) {
	repository := &homeLayoutRepositoryFixture{accountID: "", snoozes: accounts.JobSnoozesState{
		Snoozes: []accounts.JobSnooze{{League: "league.a", JobID: "trade.review", Until: "2026-11-19T12:00:00Z"}},
	}}
	read := RawWorkflow{Command: "job.snoozes.read", Version: "1", Mode: "read", Arguments: json.RawMessage(`{}`)}
	if league, err := read.League(); err != nil || league != "" {
		t.Fatalf("account snooze league = %q err=%v", league, err)
	}
	data, err := RunAccountWorkflow(t.Context(), repository, "trusted-account", read)
	if err != nil || repository.accountID != "trusted-account" {
		t.Fatalf("read account=%q err=%v", repository.accountID, err)
	}
	var state accounts.JobSnoozesState
	if err := json.Unmarshal(data, &state); err != nil || len(state.Snoozes) != 1 {
		t.Fatalf("read workflow data = %+v err=%v", state, err)
	}
	if _, err := RunWorkflow(t.Context(), nil, read); err == nil {
		t.Fatal("account snoozes were read without authenticated owner scope")
	}

	set := RawWorkflow{Command: "job.snooze.set", Version: "1", Mode: "execute",
		Arguments: json.RawMessage(`{"league":"league.a","jobId":"trade.review","until":"2026-11-19T12:00:00Z"}`)}
	if _, err := RunAccountWorkflow(t.Context(), repository, "trusted-account", set); err != nil {
		t.Fatal(err)
	}
	if repository.accountID != "trusted-account" || repository.snoozeInput.JobID != "trade.review" {
		t.Fatalf("set scope = %q %+v", repository.accountID, repository.snoozeInput)
	}
	set.Arguments = json.RawMessage(`{"league":"league.a","jobId":"trade.review","until":"2026-11-19T12:00:00Z","accountId":"attacker"}`)
	if _, err := RunAccountWorkflow(t.Context(), repository, "trusted-account", set); err == nil {
		t.Fatal("request-selected account identity accepted")
	}

	wake := RawWorkflow{Command: "job.snooze.wake", Version: "1", Mode: "execute",
		Arguments: json.RawMessage(`{"league":"league.a","jobId":"trade.review"}`)}
	if _, err := RunAccountWorkflow(t.Context(), repository, "trusted-account", wake); err != nil {
		t.Fatal(err)
	}
	if repository.accountID != "trusted-account" || repository.wakeInput.JobID != "trade.review" {
		t.Fatalf("wake scope = %q %+v", repository.accountID, repository.wakeInput)
	}
	if !IsAccountWorkflow("job.snoozes.read") || !IsAccountWorkflow("job.snooze.set") || !IsAccountWorkflow("job.snooze.wake") {
		t.Fatal("a job preference command was not marked account-scoped")
	}
}
