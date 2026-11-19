package serverhttp_test

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
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

func TestHostedBackupWorkflowPreviewsExecutesAndRefusesOwnerAndAgent(t *testing.T) {
	f := newHostedFixture(t, true)
	bearer, proof := f.directCommissioner(t)
	invoke := func(request serverhttp.RawWorkflow, token, recent string, want int) serverhttp.WorkflowResponse {
		t.Helper()
		body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &request})
		if err != nil {
			t.Fatal(err)
		}
		response := f.post(t, serverhttp.WorkflowPath, string(body), token, recent, want)
		var hosted serverhttp.WorkflowResponse
		if want == 200 {
			if err := json.Unmarshal(response, &hosted); err != nil {
				t.Fatal(err)
			}
		}
		return hosted
	}
	preview := invoke(workflowRequest(t, leagueservice.BackupNowCommand, "preview",
		`{"league":"native.hosted"}`, ""), bearer, proof, 200)
	var description commissioner.Preview
	if err := json.Unmarshal(preview.Data, &description); err != nil || description.Safeguard != "confirm" ||
		description.Target.League != "native.hosted" {
		t.Fatalf("backup preview = %+v, %v", description, err)
	}
	if f.backupCalls.Load() != 0 {
		t.Fatal("preview started a backup")
	}
	result := invoke(workflowRequest(t, leagueservice.BackupNowCommand, "execute",
		`{"league":"native.hosted"}`, ""), bearer, proof, 200)
	var saved leagueservice.BackupResult
	if err := json.Unmarshal(result.Data, &saved); err != nil || saved.League != "native.hosted" || !saved.Checked ||
		saved.Where != "Cloudflare R2" || saved.Message == "" || f.backupCalls.Load() != 1 {
		t.Fatalf("backup result = %+v, calls=%d, err=%v", saved, f.backupCalls.Load(), err)
	}
	owner := f.login(t, "owner", "private-password")
	invoke(workflowRequest(t, leagueservice.BackupNowCommand, "preview",
		`{"league":"native.hosted"}`, ""), owner, "", http.StatusConflict)
	token := f.agentToken(t, bearer, []string{"commish.jobs.read"}, "read")
	invoke(workflowRequest(t, leagueservice.BackupNowCommand, "execute",
		`{"league":"native.hosted"}`, ""), token.Bearer, "", http.StatusForbidden)
	f.backupBusy.Store(true)
	busyRequest := workflowRequest(t, leagueservice.BackupNowCommand, "execute",
		`{"league":"native.hosted"}`, "")
	busyBody, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &busyRequest})
	if err != nil {
		t.Fatal(err)
	}
	busy := f.post(t, serverhttp.WorkflowPath, string(busyBody), bearer, proof, http.StatusConflict)
	if string(busy) != leagueservice.BackupInProgressMessage+"\n" {
		t.Fatalf("busy backup words = %q", busy)
	}
	f.backupBusy.Store(false)
	if f.backupCalls.Load() != 1 {
		t.Fatal("a refused owner, agent or overlapping request started another backup")
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
	raw.Code = testTOTP(t, setup.Seed, f.clockTime())
	if _, err := desktop.Act(t.Context(), raw); err != nil {
		t.Fatal(err)
	}
	f.advanceTime(30 * time.Second)
	raw.Action = "totp-step-up"
	raw.Code = testTOTP(t, setup.Seed, f.clockTime())
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

func TestHostedCommissionerControlRoomReads(t *testing.T) {
	f := newHostedFixture(t)
	calendarRules := []struct{ field, value string }{
		{"calendar.timezone", `"America/Chicago"`},
		{"calendar.boundary", `"start_of_day"`},
		{"calendar.windows", `{"rookie_draft":{"start":{"month":1,"day":1},"opens":["draft_pick","lineup","ir","taxi","waiver_claim","ufa_bid","trade"]}}`},
	}
	for _, rule := range calendarRules {
		assets, err := f.history.NativeAssets(t.Context(), "native.hosted")
		if err != nil {
			t.Fatal(err)
		}
		command := commissioner.RawCommand{League: "native.hosted", Command: commissioner.RulesSet,
			Args:     commissioner.RawArgs{Field: rule.field, Value: json.RawMessage(rule.value)},
			Expected: assets.Revision, Reason: "Configure the disposable league calendar",
			Request: "workflow.calendar-rule." + rule.field, Effective: "now", Visibility: "league"}
		if _, err := f.history.ExecuteCommissioner(t.Context(), command, f.principal, nil); err != nil {
			t.Fatal(err)
		}
	}
	events, err := f.history.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	records := make([]commissioner.Record, len(events))
	for i, event := range events {
		records[i] = *event.Payload.Commissioner
	}
	state, err := commissioner.GovernanceState(records)
	if err != nil {
		t.Fatal(err)
	}
	if state.HoldsSeat("franchise.01", governance.Principal(f.principal)) {
		t.Fatal("commissioner still holds a franchise seat")
	}
	eventCount := len(events)
	bearer, proof := f.directCommissioner(t)
	reads := []struct{ command, arguments string }{
		{"commish.jobs.read", `{"league":"native.hosted"}`},
		{"commish.people.read", `{"league":"native.hosted"}`},
		{"commish.schedule.read", `{"league":"native.hosted"}`},
		{"commish.ledger.read", `{"league":"native.hosted"}`},
		{"commish.ledger.filtered.read", `{"league":"native.hosted","filter":{"person":"","kind":"rules","from":"","before":""}}`},
		{"commish.notices.read", `{"league":"native.hosted"}`},
		{"calendar.read", `{"league":"native.hosted","season":2026,"localZone":"UTC","franchise":"franchise.01","action":"lineup"}`},
	}
	for _, read := range reads {
		t.Run(read.command, func(t *testing.T) {
			raw := workflowRequest(t, read.command, "read", read.arguments, "")
			body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
			if err != nil {
				t.Fatal(err)
			}
			status, response := f.send(t, serverhttp.WorkflowPath, string(body), bearer, proof)
			if status != http.StatusOK {
				t.Fatalf("status %d: %s", status, response)
			}
			var result serverhttp.WorkflowResponse
			if err := json.Unmarshal(response, &result); err != nil {
				t.Fatal(err)
			}
			var reading struct {
				League string `json:"league"`
				Actors []struct {
					Person               string   `json:"person"`
					EffectivePermissions []string `json:"effectivePermissions"`
				} `json:"actors"`
			}
			if err := json.Unmarshal(result.Data, &reading); err != nil || reading.League != "native.hosted" {
				t.Fatalf("wrong %s response: %s (%v)", read.command, result.Data, err)
			}
			if read.command == "commish.people.read" {
				for _, actor := range reading.Actors {
					if actor.Person != f.principal {
						continue
					}
					for _, permission := range actor.EffectivePermissions {
						if permission == string(governance.ChangeRules) {
							return
						}
					}
				}
				t.Fatalf("People response omitted the commissioner's effective permissions: %s", result.Data)
			}
		})
	}

	ownerFixture := newHostedFixture(t, true)
	owner := ownerFixture.login(t, "owner", "private-password")
	for _, command := range []string{"commish.jobs.read", "commish.schedule.read"} {
		t.Run(command+" denies owner", func(t *testing.T) {
			raw := workflowRequest(t, command, "read", `{"league":"native.hosted"}`, "")
			body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
			if err != nil {
				t.Fatal(err)
			}
			status, _ := ownerFixture.send(t, serverhttp.WorkflowPath, string(body), owner, "")
			if status != http.StatusConflict {
				t.Fatalf("owner read status %d, want %d", status, http.StatusConflict)
			}
		})
	}
	finalEvents, err := f.history.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
	if err != nil || len(finalEvents) != eventCount {
		t.Fatalf("Control Room reads changed the ledger: events=%d want=%d err=%v", len(finalEvents), eventCount, err)
	}
}
