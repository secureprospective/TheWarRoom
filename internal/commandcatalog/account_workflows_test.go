package catalog

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
)

type homeLayoutRepositoryFixture struct {
	accountID   string
	input       accounts.RawHomeLayoutSet
	state       accounts.HomeLayoutState
	snoozeInput accounts.RawJobSnoozeSet
	wakeInput   accounts.RawJobSnoozeWake
	snoozes     accounts.JobSnoozesState
}

func (f *homeLayoutRepositoryFixture) ReadHomeLayout(_ context.Context, accountID string) (accounts.HomeLayoutState, error) {
	f.accountID = accountID
	return f.state, nil
}

func (f *homeLayoutRepositoryFixture) SetHomeLayout(_ context.Context, accountID string,
	input accounts.RawHomeLayoutSet) (accounts.HomeLayoutState, error) {
	f.accountID, f.input = accountID, input
	return f.state, nil
}

func (f *homeLayoutRepositoryFixture) ReadJobSnoozes(_ context.Context, accountID string) (accounts.JobSnoozesState, error) {
	f.accountID = accountID
	return f.snoozes, nil
}

func (f *homeLayoutRepositoryFixture) SetJobSnooze(_ context.Context, accountID string,
	input accounts.RawJobSnoozeSet) (accounts.JobSnoozesState, error) {
	f.accountID, f.snoozeInput = accountID, input
	return f.snoozes, nil
}

func (f *homeLayoutRepositoryFixture) WakeJobSnooze(_ context.Context, accountID string,
	input accounts.RawJobSnoozeWake) (accounts.JobSnoozesState, error) {
	f.accountID, f.wakeInput = accountID, input
	return f.snoozes, nil
}

func TestHomeLayoutWorkflowsUseAccountScopeWithoutLeagueInput(t *testing.T) {
	repository := &homeLayoutRepositoryFixture{state: accounts.HomeLayoutState{
		Layout: json.RawMessage(`{"mode":"default","tiles":[],"over":[],"pins":{}}`), Revision: 4}}
	read := RawWorkflow{Command: "home.layout.read", Version: "1", Mode: "read", Arguments: json.RawMessage(`{}`)}
	if league, err := read.League(); err != nil || league != "" {
		t.Fatalf("account preference league = %q err=%v", league, err)
	}
	data, err := RunAccountWorkflow(t.Context(), repository, "trusted-account", read)
	if err != nil || repository.accountID != "trusted-account" {
		t.Fatalf("read account=%q err=%v", repository.accountID, err)
	}
	var state accounts.HomeLayoutState
	if err := json.Unmarshal(data, &state); err != nil || state.Revision != 4 {
		t.Fatalf("read workflow data = %+v err=%v", state, err)
	}
	if _, err := RunWorkflow(t.Context(), nil, read); err == nil {
		t.Fatal("account preference ran without its authenticated owner scope")
	}

	set := RawWorkflow{Command: "home.layout.set", Version: "1", Mode: "execute",
		Arguments: json.RawMessage(`{"layout":{"mode":"default","tiles":[],"over":[],"pins":{}},"expectedRevision":4}`)}
	if _, err := RunAccountWorkflow(t.Context(), repository, "trusted-account", set); err != nil {
		t.Fatal(err)
	}
	if repository.accountID != "trusted-account" || repository.input.ExpectedRevision != 4 {
		t.Fatalf("write scope = %q %+v", repository.accountID, repository.input)
	}
	set.Arguments = json.RawMessage(`{"layout":{"mode":"default","tiles":[],"over":[],"pins":{}},"expectedRevision":4,"accountId":"attacker"}`)
	if _, err := RunAccountWorkflow(t.Context(), repository, "trusted-account", set); err == nil {
		t.Fatal("request-selected account identity accepted")
	}
}
