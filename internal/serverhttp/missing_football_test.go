package serverhttp_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

// A league whose pinned football is not on the server is still served (Christopher, 2026-11-14:
// "league comes back, football waits"): its rosters and contracts read, the reply says it cannot be
// changed, every change is held with plain words, and it opens again once the football is back.
func TestMissingFootballHoldsChangesAndKeepsReadings(t *testing.T) {
	c := workflowClient(t)
	if got := c.invoke("native.assets.read", "read", []byte(`{"league":"native.hosted"}`), 200); !got.Writable {
		t.Fatal("league not writable with its football present")
	}
	c.f.footballGone.Store(true)
	got := c.invoke("native.assets.read", "read", []byte(`{"league":"native.hosted"}`), 200)
	if got.Writable || len(got.Data) == 0 {
		t.Fatalf("reading without football: writable=%v data=%d bytes", got.Writable, len(got.Data))
	}
	var assets struct {
		Snapshot json.RawMessage `json:"snapshot"`
	}
	if err := json.Unmarshal(got.Data, &assets); err != nil || len(assets.Snapshot) < 3 {
		t.Fatalf("rosters and contracts not read: %v %s", err, got.Data)
	}
	bid := c.command(commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "6001", Franchise: "franchise.01", Salary: 500000000, Years: 1}}, "bid.held")
	args, err := json.Marshal(bid)
	if err != nil {
		t.Fatal(err)
	}
	raw := workflowRequest(t, bid.Command, "execute", string(args), c.football)
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	words := string(c.f.post(t, serverhttp.WorkflowPath, string(body), c.bearer, c.proof, 409))
	if !strings.Contains(words, "is not on this server right now, so changes wait until it is back") {
		t.Fatalf("change not held in plain words: %q", words)
	}
	c.f.footballGone.Store(false)
	if got := c.invoke("native.assets.read", "read", []byte(`{"league":"native.hosted"}`), 200); !got.Writable {
		t.Fatal("league did not open again once its football was back")
	}
	c.run(c.command(commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "6001", Franchise: "franchise.01", Salary: 500000000, Years: 1}}, "bid.after"))
}
