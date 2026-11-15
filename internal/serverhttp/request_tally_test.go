package serverhttp_test

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestRequestTallyWorkflow(t *testing.T) {
	f := newHostedFixture(t, true)
	bearer, proof := f.directCommissioner(t)
	raw := workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, "")
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		f.post(t, serverhttp.WorkflowPath, string(body), bearer, proof, 200)
	}
	_, counts := f.tally.Snapshot()
	got := counts["native.assets.read read"]
	if got.Calls != 2 || got.Computed != 1 || got.Cached != 1 || got.ReplyBytes == 0 {
		t.Fatalf("counts: %+v", got)
	}
	raw = workflowRequest(t, "native.assets.read", "read", `{"league":"native.other"}`, "")
	body, err = json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	f.post(t, serverhttp.WorkflowPath, string(body), bearer, proof, 403)
	raw = workflowRequest(t, "bids.read", "read", `{"league":"native.hosted","franchise":"franchise.private-sentinel"}`, "")
	body, err = json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.send(t, serverhttp.WorkflowPath, string(body), bearer, proof)
	_, counts = f.tally.Snapshot()
	got = counts["native.assets.read read"]
	if got.Refused != 1 {
		t.Fatalf("refused: %+v", got)
	}
	encoded, err := json.Marshal(counts)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "native.hosted") || strings.Contains(string(encoded), "native.other") || strings.Contains(string(encoded), "franchise.private-sentinel") {
		t.Fatalf("private input in counts: %s", encoded)
	}
	f.tally.Reset()
	_, counts = f.tally.Snapshot()
	if len(counts) != 0 {
		t.Fatalf("reset: %+v", counts)
	}
}
func TestRequestTallyConcurrent(t *testing.T) {
	tally := serverhttp.NewRequestTally()
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 1000 {
				tally.Unchanged("native.assets.read", "read")
			}
		}()
	}
	wg.Wait()
	_, counts := tally.Snapshot()
	if counts["native.assets.read read"].Unchanged != 50000 {
		t.Fatal(counts)
	}
}
