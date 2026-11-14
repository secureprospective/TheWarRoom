package serverhttp_test

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

type fakeVersionSource struct {
	mu        sync.Mutex
	epoch     string
	revisions map[string]int64
	versions  map[string]int
	topics    []serverhttp.WorkflowTopic
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
	defer s.mu.Unlock()
	topics := []serverhttp.WorkflowTopic{{Kind: "league", Key: "league", Version: fmt.Sprint(s.versions[account+"|"+league])}}
	topics = append(topics, s.topics...)
	return serverhttp.WorkflowVersionSnapshot{ServerEpoch: s.epoch, Topics: topics}
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
