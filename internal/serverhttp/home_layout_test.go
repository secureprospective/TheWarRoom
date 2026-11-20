package serverhttp_test

import (
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func layoutRequest(t *testing.T, command, mode, arguments string) string {
	t.Helper()
	workflow := workflowRequest(t, command, mode, arguments, "")
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &workflow})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func readAccountLayout(t *testing.T, f *hostedFixture, bearer string) accounts.HomeLayoutState {
	t.Helper()
	body := f.post(t, serverhttp.AccountPreferencePath,
		layoutRequest(t, "home.layout.read", "read", `{}`), bearer, "", 200)
	var response serverhttp.WorkflowResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	var state accounts.HomeLayoutState
	if err := json.Unmarshal(response.Data, &state); err != nil {
		t.Fatal(err)
	}
	return state
}

func TestHostedHomeLayoutIsPrivateRevisionedAndNotDelegable(t *testing.T) {
	f := newHostedFixture(t, true)
	commissioner, _ := f.directCommissioner(t)
	owner := f.login(t, "owner", "private-password")
	initial := readAccountLayout(t, f, commissioner)
	if initial.Revision != 0 || string(initial.Layout) != `{"mode":"default","tiles":[],"over":[],"pins":{}}` {
		t.Fatalf("initial layout = %+v", initial)
	}

	custom := `{"mode":"custom","tiles":[["news","1x2"],["wire","1x1"]],"over":[],"pins":{"cap":true}}`
	request := layoutRequest(t, "home.layout.set", "execute",
		`{"layout":`+custom+`,"expectedRevision":0}`)
	body := f.post(t, serverhttp.AccountPreferencePath, request, commissioner, "", 200)
	var response serverhttp.WorkflowResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	var saved accounts.HomeLayoutState
	if err := json.Unmarshal(response.Data, &saved); err != nil || saved.Revision != 1 || saved.Conflict || string(saved.Layout) != custom {
		t.Fatalf("saved layout = %+v err=%v", saved, err)
	}

	other := readAccountLayout(t, f, owner)
	if other.Revision != 0 || string(other.Layout) != string(initial.Layout) {
		t.Fatalf("another account read saved layout: %+v", other)
	}
	ownerLayout := `{"mode":"custom","tiles":[["cap","1x1"]],"over":[],"pins":{}}`
	f.post(t, serverhttp.AccountPreferencePath,
		layoutRequest(t, "home.layout.set", "execute", `{"layout":`+ownerLayout+`,"expectedRevision":0}`), owner, "", 200)
	if got := readAccountLayout(t, f, commissioner); got.Revision != 1 || string(got.Layout) != custom {
		t.Fatalf("another account changed saved layout: %+v", got)
	}

	stale := layoutRequest(t, "home.layout.set", "execute",
		`{"layout":{"mode":"default","tiles":[],"over":[],"pins":{}},"expectedRevision":0}`)
	body = f.post(t, serverhttp.AccountPreferencePath, stale, commissioner, "", 200)
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	var conflict accounts.HomeLayoutState
	if err := json.Unmarshal(response.Data, &conflict); err != nil || !conflict.Conflict || conflict.Revision != 1 || string(conflict.Layout) != custom {
		t.Fatalf("stale write did not return server copy: %+v err=%v", conflict, err)
	}

	invalid := layoutRequest(t, "home.layout.set", "execute",
		`{"layout":{"mode":"custom","tiles":[["unknown","1x1"]],"over":[],"pins":{}},"expectedRevision":1}`)
	f.post(t, serverhttp.AccountPreferencePath, invalid, commissioner, "", 400)
	spoof := layoutRequest(t, "home.layout.set", "execute",
		`{"layout":{"mode":"default","tiles":[],"over":[],"pins":{}},"expectedRevision":1,"accountId":"aaaaaaaa-1234-4234-8234-123456789abc"}`)
	f.post(t, serverhttp.AccountPreferencePath, spoof, commissioner, "", 400)
	if got := readAccountLayout(t, f, commissioner); got.Revision != 1 || string(got.Layout) != custom {
		t.Fatalf("rejected writes changed saved layout: %+v", got)
	}
	f.post(t, serverhttp.AccountPreferencePath,
		layoutRequest(t, "home.layout.read", "read", `{}`), "", "", 401)

	agent := f.agentToken(t, commissioner, []string{"commish.rules.read"}, "read")
	f.post(t, serverhttp.AccountPreferencePath,
		layoutRequest(t, "home.layout.read", "read", `{}`), agent.Bearer, "", 403)
}
