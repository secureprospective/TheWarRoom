package serverhttp_test

import (
	"encoding/json"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverclient"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

// workflowClock runs ahead of the wall clock by skew, which tests advance to pass due deadlines.
type workflowClock struct{ skew *atomic.Int64 }

func (c workflowClock) Now() time.Time { return time.Now().Add(time.Duration(c.skew.Load())) }
func (workflowClock) Trusted() error   { return nil }

func workflowRequest(t *testing.T, command, mode string, args string, football string) serverhttp.RawWorkflow {
	t.Helper()
	return serverhttp.RawWorkflow{Command: command, Version: "1", Mode: mode, Arguments: json.RawMessage(args), Football: football}
}

func TestHostedWorkflowReadsAdmissionAndStrictEnvelope(t *testing.T) {
	f := newHostedFixture(t, true)
	bearer, proof := f.directCommissioner(t)
	raw := workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, "")
	post := func(raw serverhttp.RawWorkflow, bearer, proof string, status int) []byte {
		t.Helper()
		body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
		if err != nil {
			t.Fatal(err)
		}
		return f.post(t, serverhttp.WorkflowPath, string(body), bearer, proof, status)
	}
	post(raw, "", "", 401)
	post(raw, bearer, "", 403)
	var response serverhttp.WorkflowResponse
	if err := json.Unmarshal(post(raw, bearer, proof, 200), &response); err != nil {
		t.Fatal(err)
	}
	if response.Label != "This league is managed here" || !response.Writable || response.Football == "" {
		t.Fatalf("authority: %+v", response)
	}
	for _, bad := range []serverhttp.RawWorkflow{
		{Command: raw.Command, Version: "2", Mode: "read", Arguments: raw.Arguments},
		workflowRequest(t, "native.assets.read", "execute", string(raw.Arguments), ""),
		workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted","principal":"admin"}`, ""),
		workflowRequest(t, "commish.league.seed_held", "execute", `{}`, ""),
	} {
		post(bad, bearer, proof, 400)
	}
	if strings.Contains(string(post(raw, bearer, proof, 200)), bearer) {
		t.Fatal("bearer leaked")
	}
}

func TestDesktopHostedReserveMoveConfirmedRetryAndRefusalPreservesRoster(t *testing.T) {
	f := newHostedFixture(t, true)
	desktop := desktopFor(t, f)
	raw := serverclient.RawDesktop{Action: "login", Username: "commissioner", Password: "private-password"}
	if _, err := desktop.Act(t.Context(), raw); err != nil {
		t.Fatal(err)
	}
	if _, err := desktop.Workflow(t.Context(), workflowRequest(t, "native.assets.read", "read", `{"league":"native.other"}`, "")); err == nil {
		t.Fatal("cross-league desktop request accepted")
	}
	bearer, proof := f.directCommissioner(t)
	invoke := func(r serverhttp.RawWorkflow, status int) serverhttp.WorkflowResponse {
		t.Helper()
		body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &r})
		if err != nil {
			t.Fatal(err)
		}
		result := f.post(t, serverhttp.WorkflowPath, string(body), bearer, proof, status)
		var response serverhttp.WorkflowResponse
		if status == 200 {
			if err := json.Unmarshal(result, &response); err != nil {
				t.Fatal(err)
			}
		}
		return response
	}
	read := func() (commissioner.NativeAssets, string) {
		t.Helper()
		response := invoke(workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, ""), 200)
		var assets commissioner.NativeAssets
		if err := json.Unmarshal(response.Data, &assets); err != nil {
			t.Fatal(err)
		}
		return assets, response.Football
	}
	assets, pin := read()
	command := commissioner.RawCommand{League: "native.hosted", Command: commissioner.RosterReserve, Args: commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0100", Franchise: "franchise.01", Bucket: "ir"}}, Expected: assets.Revision, Request: "reserve.one", Effective: "now", Visibility: "league"}
	body, err := json.Marshal(command)
	if err != nil {
		t.Fatal(err)
	}
	invoke(workflowRequest(t, command.Command, "execute", string(body), ""), 409)
	before, _ := read()
	if before.Revision != assets.Revision {
		t.Fatal("missing pin changed ledger")
	}
	preview := invoke(workflowRequest(t, command.Command, "preview", string(body), pin), 200)
	var check commissioner.Preview
	if err := json.Unmarshal(preview.Data, &check); err != nil {
		t.Fatal(err)
	}
	if len(check.Blockers) > 0 {
		t.Fatalf("eligible reserve blocked: %+v", check)
	}
	afterPreview, _ := read()
	if afterPreview.Revision != assets.Revision {
		t.Fatal("preview mutated roster")
	}
	response := invoke(workflowRequest(t, command.Command, "execute", string(body), pin), 200)
	repeated := invoke(workflowRequest(t, command.Command, "execute", string(body), pin), 200)
	if string(response.Data) != string(repeated.Data) {
		t.Fatal("retry changed receipt")
	}
	after, _ := read()
	if after.Revision == assets.Revision || after.Snapshot.Players[1].Status != "IR" {
		t.Fatalf("move not recorded: %+v", after)
	}
	command.Expected = after.Revision
	command.Request = "ineligible"
	command.Args.Waiver.Player = "0099"
	body, err = json.Marshal(command)
	if err != nil {
		t.Fatal(err)
	}
	invoke(workflowRequest(t, command.Command, "execute", string(body), pin), 409)
	preserved, _ := read()
	if preserved.Revision != after.Revision || preserved.Snapshot.Players[0].Status != "ROSTER" {
		t.Fatal("ineligible IR move changed roster")
	}
}

func TestDesktopHostedLineupDraftReconnectAndRevocation(t *testing.T) {
	f := newHostedFixture(t, true)
	desktop := desktopFor(t, f)
	raw := serverclient.RawDesktop{Action: "login", Username: "commissioner", Password: "private-password"}
	if _, err := desktop.Act(t.Context(), raw); err != nil {
		t.Fatal(err)
	}
	raw.Action = "totp-begin"
	setup, err := desktop.Act(t.Context(), raw)
	if err != nil {
		t.Fatal(err)
	}
	raw.Action = "totp-confirm"
	raw.Code = testTOTP(t, setup.Seed, f.now)
	if _, err := desktop.Act(t.Context(), raw); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(30 * time.Second)
	raw.Action = "totp-step-up"
	raw.Code = testTOTP(t, setup.Seed, f.now)
	if _, err := desktop.Act(t.Context(), raw); err != nil {
		t.Fatal(err)
	}
	call := func(command, mode string, args []byte, pin string) serverhttp.WorkflowResponse {
		t.Helper()
		result, err := desktop.Workflow(t.Context(), serverhttp.RawWorkflow{Command: command, Mode: mode, Version: "1", Arguments: args, Football: pin})
		if err != nil {
			t.Fatalf("%s: %v", command, err)
		}
		return result
	}
	lineup := call("lineup.read", "read", []byte(`{"league":"native.hosted","franchise":"franchise.01","week":1,"at":""}`), "")
	var held commissioner.NativeLineup
	if err := json.Unmarshal(lineup.Data, &held); err != nil {
		t.Fatal(err)
	}
	command := commissioner.RawCommand{League: "native.hosted", Command: commissioner.LineupSet, Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: 2026, Franchise: "franchise.01", Week: 1, Starters: []string{"0099"}}}, Expected: held.Revision, Request: "lineup.one", Effective: "now", Visibility: "private"}
	body, err := json.Marshal(command)
	if err != nil {
		t.Fatal(err)
	}
	call(command.Command, "execute", body, lineup.Football)
	reread := call("lineup.read", "read", []byte(`{"league":"native.hosted","franchise":"franchise.01","week":1,"at":""}`), "")
	if err := json.Unmarshal(reread.Data, &held); err != nil || len(held.Starters) != 1 || held.Starters[0].Player != "0099" {
		t.Fatalf("lineup not saved: %+v %v", held, err)
	}
	// The practice draft deliberately starts two seconds after provisioning.
	time.Sleep(2 * time.Second)
	draftResponse := call("draft.read", "read", []byte(`{"league":"native.hosted","franchise":"franchise.01","season":0,"at":"0001-01-01T00:00:00Z"}`), "")
	var draft commissioner.NativeDraft
	if err := json.Unmarshal(draftResponse.Data, &draft); err != nil {
		t.Fatal(err)
	}
	if draft.Draft.OnClock == nil || draft.Draft.OnClock.Owner != "franchise.01" || len(draft.Available) != 1 {
		t.Fatalf("practice draft not usable: %+v", draft)
	}
	pick := commissioner.DraftPickInput{League: "native.hosted", Season: 2027, Franchise: "franchise.01", Player: "5001", Request: "pick.one", Expected: draft.Revision}
	body, err = json.Marshal(pick)
	if err != nil {
		t.Fatal(err)
	}
	call(commissioner.DraftPick, "execute", body, draftResponse.Football)
	assets := call("native.assets.read", "read", []byte(`{"league":"native.hosted"}`), "")
	var roster commissioner.NativeAssets
	if err := json.Unmarshal(assets.Data, &roster); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range roster.Snapshot.Players {
		if p.ID == "5001" && p.Franchise == "franchise.01" {
			found = true
		}
	}
	if !found {
		t.Fatal("rookie did not reach authoritative roster")
	}
	if _, err := desktop.Act(t.Context(), serverclient.RawDesktop{Action: "revoke-device"}); err != nil {
		t.Fatal(err)
	}
	if _, err := desktop.Workflow(t.Context(), workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, "")); err == nil {
		t.Fatal("revoked device could read")
	}
}

func TestHostedWorkflowFrontendReadings(t *testing.T) {
	f := newHostedFixture(t, true)
	bearer, proof := f.directCommissioner(t)
	readings := map[string]serverhttp.WorkflowResponse{}
	for _, command := range []string{"lineup.read", "native.assets.read", "native.reserve.check", "draft.read", "commish.rules.read", "bids.read", "waivers.read", "trades.read", "results.read", "standings.read"} {
		args := `{"league":"native.hosted"}`
		if command == "lineup.read" {
			args = `{"league":"native.hosted","franchise":"franchise.01","week":1,"at":""}`
		}
		if command == "draft.read" {
			args = `{"league":"native.hosted","franchise":"franchise.01","season":0,"at":"0001-01-01T00:00:00Z"}`
		}
		if command == "bids.read" {
			args = `{"league":"native.hosted","franchise":"franchise.01"}`
		}
		if command == "waivers.read" {
			args = `{"league":"native.hosted","franchise":"franchise.01","at":"2026-10-25T10:00:00Z"}`
		}
		if command == "trades.read" {
			args = `{"league":"native.hosted","viewer":"franchise.01"}`
		}
		if command == "results.read" {
			args = `{"league":"native.hosted","season":2026,"week":1}`
		}
		if command == "standings.read" {
			args = `{"league":"native.hosted","season":2026}`
		}
		raw := workflowRequest(t, command, "read", args, "")
		body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
		if err != nil {
			t.Fatal(err)
		}
		var reading serverhttp.WorkflowResponse
		if err := json.Unmarshal(f.post(t, serverhttp.WorkflowPath, string(body), bearer, proof, 200), &reading); err != nil {
			t.Fatal(err)
		}
		if err := reading.Validate(); err != nil {
			t.Fatal(err)
		}
		readings[command] = reading
	}
	if os.Getenv("TWR_UPDATE_HOSTED_WIRE") == "1" {
		body, err := json.MarshalIndent(readings, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile("../../frontend/src/app/commissioner/hosted.wire.testdata.json", body, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
