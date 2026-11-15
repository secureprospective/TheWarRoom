package serverhttp_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverclient"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

type fakeVersionSource struct {
	afterSnapshot func()
	values        string
	mu            sync.Mutex
	epoch         string
	revisions     map[string]int64
	versions      map[string]int
	topics        []serverhttp.WorkflowTopic
}

func newFakeVersionSource() *fakeVersionSource {
	return &fakeVersionSource{epoch: "epoch.one", revisions: make(map[string]int64), versions: make(map[string]int)}
}
func (s *fakeVersionSource) Observe(account, league string, revision int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := account + "|" + league
	if before, ok := s.revisions[key]; !ok || before != revision {
		s.versions[key]++
		s.revisions[key] = revision
	}
}
func (s *fakeVersionSource) Snapshot(account, league string) serverhttp.WorkflowVersionSnapshot {
	s.mu.Lock()
	topics := []serverhttp.WorkflowTopic{{Kind: "league", Key: "league", Version: fmt.Sprint(s.versions[account+"|"+league])}}
	topics = append(topics, s.topics...)
	snapshot := serverhttp.WorkflowVersionSnapshot{ServerEpoch: s.epoch, Topics: topics, Values: s.values}
	hook := s.afterSnapshot
	s.mu.Unlock()
	if hook != nil {
		hook()
	}
	return snapshot
}

func taggedWorkflow(t *testing.T, f *hostedFixture, raw serverhttp.RawWorkflow, bearer, proof string, status int) serverhttp.WorkflowResponse {
	t.Helper()
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	result := f.post(t, serverhttp.WorkflowPath, string(body), bearer, proof, status)
	var reply serverhttp.WorkflowResponse
	if status == 200 {
		if err := json.Unmarshal(result, &reply); err != nil {
			t.Fatal(err)
		}
		if err := reply.Validate(); err != nil {
			t.Fatal(err)
		}
		if len(result) >= 256<<10 {
			t.Fatal("desktop cap exceeded")
		}
		if reply.Unchanged {
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(result, &fields); err != nil {
				t.Fatal(err)
			}
			if len(fields) != 2 {
				t.Fatalf("unchanged carries body: %s", result)
			}
		}
	}
	return reply
}

func TestWorkflowTagsObserveDisconnectedCorrection(t *testing.T) {
	source := newFakeVersionSource()
	f := newVersionedHostedFixture(t, serverhttp.NewWorkflowVersions(source), true)
	bearer, proof := f.directCommissioner(t)
	read := workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, "")
	first := taggedWorkflow(t, f, read, bearer, proof, 200)
	if first.Tags == nil || first.Tags.League["league"] != "1" {
		t.Fatalf("missing tags: %+v", first)
	}
	read.Known = first.Tags
	same := taggedWorkflow(t, f, read, bearer, proof, 200)
	if !same.Unchanged || len(same.Data) != 0 {
		t.Fatal("same snapshot not unchanged")
	}
	_, counts := f.tally.Snapshot()
	if got := counts["native.assets.read read"]; got.Computed != 1 || got.Cached != 0 || got.Unchanged != 1 {
		t.Fatalf("conditional read ran an engine or reply cache: %+v", got)
	}
	var assets commissioner.NativeAssets
	if err := json.Unmarshal(first.Data, &assets); err != nil {
		t.Fatal(err)
	}
	command := commissioner.RawCommand{League: "native.hosted", Command: commissioner.RosterReserve, Args: commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0100", Franchise: "franchise.01", Bucket: "ir"}}, Expected: assets.Revision, Request: "tag.reserve", Effective: "now", Visibility: "league"}
	body, err := json.Marshal(command)
	if err != nil {
		t.Fatal(err)
	}
	taggedWorkflow(t, f, workflowRequest(t, command.Command, "execute", string(body), first.Football), bearer, proof, 200)
	next := taggedWorkflow(t, f, read, bearer, proof, 200)
	if next.Unchanged || next.Tags.League["league"] == first.Tags.League["league"] {
		t.Fatal("disconnected correction falsely unchanged")
	}
	if err := json.Unmarshal(next.Data, &assets); err != nil {
		t.Fatal(err)
	}
	if assets.Snapshot.Players[1].Status != "IR" {
		t.Fatal("correction not visible")
	}
}

func TestWorkflowTagsBumpedDependencyAndRestart(t *testing.T) {
	source := newFakeVersionSource()
	source.topics = []serverhttp.WorkflowTopic{{Kind: "scores", Key: "scores:2026:1", Season: 2026, Week: 1, Version: "1"}, {Kind: "scores", Key: "scores:2025:1", Season: 2025, Week: 1, Version: "9"}}
	f := newVersionedHostedFixture(t, serverhttp.NewWorkflowVersions(source), true)
	bearer, proof := f.directCommissioner(t)
	read := workflowRequest(t, "results.read", "read", `{"league":"native.hosted","season":2026,"week":1}`, "")
	first := taggedWorkflow(t, f, read, bearer, proof, 200)
	if first.Tags.League["scores:2026:1"] != "1" || first.Tags.League["scores:2025:1"] != "" {
		t.Fatal("scoped dependencies lost")
	}
	source.mu.Lock()
	source.topics[0].Version = "2"
	source.mu.Unlock()
	read.Known = first.Tags
	next := taggedWorkflow(t, f, read, bearer, proof, 200)
	if next.Unchanged || next.Tags.League["scores:2026:1"] != "2" || first.Tags.League["scores:2026:1"] != "1" {
		t.Fatal("new ping rejected or older reply relabeled")
	}
	source.mu.Lock()
	source.epoch = "epoch.two"
	source.mu.Unlock()
	read.Known = next.Tags
	restarted := taggedWorkflow(t, f, read, bearer, proof, 200)
	if restarted.Unchanged || restarted.Tags.ServerEpoch != "epoch.two" {
		t.Fatal("restart failed to resync")
	}
}

func TestWorkflowWithoutVersionSourceNeverAnswersUnchanged(t *testing.T) {
	f := newHostedFixture(t, true)
	bearer, proof := f.directCommissioner(t)
	read := workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, "")
	read.Known = &serverhttp.ReadTags{ServerEpoch: "old", League: map[string]string{"league": "1"}}
	reply := taggedWorkflow(t, f, read, bearer, proof, 200)
	if reply.Unchanged || reply.Tags != nil || len(reply.Data) == 0 {
		t.Fatal("unwired source changed production behavior")
	}
}

func TestWorkflowKnownCannotAuthorizeAnotherLeague(t *testing.T) {
	source := newFakeVersionSource()
	f := newVersionedHostedFixture(t, serverhttp.NewWorkflowVersions(source), true)
	bearer, proof := f.directCommissioner(t)
	read := workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, "")
	first := taggedWorkflow(t, f, read, bearer, proof, 200)
	read.Arguments = json.RawMessage(`{"league":"native.other"}`)
	read.Known = first.Tags
	taggedWorkflow(t, f, read, bearer, proof, 403)
}

func TestVersionsReadReconcilesSessionAndAuthorizedTopics(t *testing.T) {
	source := newFakeVersionSource()
	source.topics = []serverhttp.WorkflowTopic{{Kind: "scores", Key: "scores:2026:1", Season: 2026, Week: 1, Version: "8"}}
	f := newVersionedHostedFixture(t, serverhttp.NewWorkflowVersions(source), true)
	bearer, proof := f.directCommissioner(t)
	read := workflowRequest(t, "versions.read", "read", `{"league":"native.hosted"}`, "")
	first := taggedWorkflow(t, f, read, bearer, proof, 200)
	var reading struct {
		Access     string            `json:"access"`
		Season     int               `json:"season"`
		Topics     map[string]string `json:"topics"`
		ServerTime string            `json:"serverTime"`
	}
	if err := json.Unmarshal(first.Data, &reading); err != nil {
		t.Fatal(err)
	}
	if len(reading.Access) != 64 || reading.Season != 2026 || reading.Topics["scores:2026:1"] != "8" || reading.ServerTime == "" {
		t.Fatalf("incomplete reconciliation: %+v", reading)
	}
	generation := reading.Access
	read.Known = first.Tags
	again := taggedWorkflow(t, f, read, bearer, proof, 200)
	if again.Unchanged {
		t.Fatal("reconciliation skipped authentication generation")
	}
	if err := json.Unmarshal(again.Data, &reading); err != nil {
		t.Fatal(err)
	}
	if generation != reading.Access {
		t.Fatal("unchanged login moved authentication generation")
	}
	session, err := f.sessions.Login(t.Context(), accounts.RawCredentials{Username: "commissioner", Password: "private-password"}, "new desktop")
	if err != nil {
		t.Fatal(err)
	}
	relogged := taggedWorkflow(t, f, read, session.Bearer, proof, 200)
	if err := json.Unmarshal(relogged.Data, &reading); err != nil {
		t.Fatal(err)
	}
	if generation == reading.Access {
		t.Fatal("new login kept old authentication generation")
	}
}

func TestWorkflowRevocationDuringConditionalSnapshotRefuses(t *testing.T) {
	source := newFakeVersionSource()
	f := newVersionedHostedFixture(t, serverhttp.NewWorkflowVersions(source), true)
	bearer, proof := f.directCommissioner(t)
	read := workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, "")
	first := taggedWorkflow(t, f, read, bearer, proof, 200)
	read.Known = first.Tags
	entered, release := make(chan struct{}), make(chan struct{})
	source.mu.Lock()
	source.afterSnapshot = func() { close(entered); <-release }
	source.mu.Unlock()
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &read})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan int, 1)
	go func() { status, _ := f.send(t, serverhttp.WorkflowPath, string(body), bearer, proof); done <- status }()
	<-entered
	if err := f.sessions.Revoke(t.Context(), accounts.RawSession{Bearer: bearer}, false); err != nil {
		t.Fatal(err)
	}
	close(release)
	if status := <-done; status != http.StatusForbidden {
		t.Fatalf("revoked request answered %d", status)
	}
}

func TestWorkflowModelPublicationDoesNotMoveScores(t *testing.T) {
	source := newFakeVersionSource()
	source.values = "run.one"
	source.topics = []serverhttp.WorkflowTopic{{Kind: "scores", Key: "scores:2026:1", Season: 2026, Week: 1, Version: "8"}, {Kind: "values", Key: "values:2026:run.one", Season: 2026, Version: "1"}}
	f := newVersionedHostedFixture(t, serverhttp.NewWorkflowVersions(source), true)
	bearer, proof := f.directCommissioner(t)
	read := workflowRequest(t, "versions.read", "read", `{"league":"native.hosted"}`, "")
	first := taggedWorkflow(t, f, read, bearer, proof, 200)
	source.mu.Lock()
	source.values = "run.two"
	source.topics[1].Version = "2"
	source.mu.Unlock()
	second := taggedWorkflow(t, f, read, bearer, proof, 200)
	if first.Tags.Values == second.Tags.Values || first.Tags.League["scores:2026:1"] != second.Tags.League["scores:2026:1"] || first.Tags.League["league"] != second.Tags.League["league"] {
		t.Fatal("model publication changed unrelated evidence")
	}
}

func TestWorkflowPrivateChangeDoesNotMoveAnotherRecipientsTags(t *testing.T) {
	source := newFakeVersionSource()
	f := newVersionedHostedFixture(t, serverhttp.NewWorkflowVersions(source), true)
	bearer, proof := f.directCommissioner(t)
	ownerBearer := f.login(t, "owner", "private-password")
	partnerBearer := f.login(t, "partner", "private-password")
	read := workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, "")
	commissionerReply := taggedWorkflow(t, f, read, bearer, proof, 200)
	ownerBefore := taggedWorkflow(t, f, read, ownerBearer, "", 200)
	partnerBefore := taggedWorkflow(t, f, read, partnerBearer, "", 200)
	owner := hostedWorkflowClient{t: t, f: f, bearer: ownerBearer, football: commissionerReply.Football, previews: map[string][]commissioner.PreviewField{}}
	owner.run(owner.command(commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "6001", Franchise: "franchise.02", Salary: 500000000, Years: 1}}, "tags.private.bid"))
	ownerRead := read
	ownerRead.Known = ownerBefore.Tags
	afterOwner := taggedWorkflow(t, f, ownerRead, ownerBearer, "", 200)
	read.Known = partnerBefore.Tags
	afterPartner := taggedWorkflow(t, f, read, partnerBearer, "", 200)
	if afterOwner.Unchanged || !afterPartner.Unchanged {
		t.Fatalf("private bump escaped recipient: owner %v partner %v", afterOwner.Unchanged, afterPartner.Unchanged)
	}
}

func TestDesktopRoundTripsMinimalUnchangedWorkflow(t *testing.T) {
	source := newFakeVersionSource()
	f := newVersionedHostedFixture(t, serverhttp.NewWorkflowVersions(source), true)
	desktop := desktopFor(t, f)
	if _, err := desktop.Act(t.Context(), serverclient.RawDesktop{Action: "login", Username: "owner", Password: "private-password"}); err != nil {
		t.Fatal(err)
	}
	read := workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, "")
	first, err := desktop.Workflow(t.Context(), read)
	if err != nil {
		t.Fatal(err)
	}
	read.Known = first.Tags
	same, err := desktop.Workflow(t.Context(), read)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(same)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatal(err)
	}
	if !same.Unchanged || len(fields) != 2 || same.Tags.League["league"] != "1" {
		t.Fatalf("desktop lost conditional envelope: %s", body)
	}
}

func TestWorkflowPublisherScoresConditionalRead(t *testing.T) {
	publisher := serverhttp.NewPublisher()
	f := newVersionedHostedFixture(t, serverhttp.NewWorkflowVersions(serverhttp.PublisherVersionSource{Publisher: publisher}), true)
	commissionerBearer, proof := f.directCommissioner(t)
	ownerBearer := f.login(t, "owner", "private-password")
	read := workflowRequest(t, "results.read", "read", `{"league":"native.hosted","season":2026,"week":1}`, "")
	first := taggedWorkflow(t, f, read, ownerBearer, "", 200)
	publisher.PublishScores("native.hosted", 2026, 1)
	read.Known = first.Tags
	next := taggedWorkflow(t, f, read, ownerBearer, "", 200)
	if next.Unchanged || next.Tags.League["scores:2026:1"] != "1" {
		t.Fatal("score publication did not invalidate read")
	}
	read.Known = next.Tags
	if !taggedWorkflow(t, f, read, ownerBearer, "", 200).Unchanged {
		t.Fatal("new tags did not reconcile")
	}
	for _, role := range []struct{ name, bearer, proof string }{{"owner", ownerBearer, ""}, {"commissioner", commissionerBearer, proof}} {
		t.Run(role.name, func(t *testing.T) {
			reply := taggedWorkflow(t, f, read, role.bearer, role.proof, 200)
			if reply.Tags.League["scores:2026:1"] != "1" || reply.Tags.League["league"] == "" {
				t.Fatal("missing authorized topics")
			}
		})
	}
}
