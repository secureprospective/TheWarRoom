package serverhttp_test

import (
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestHostedWorkflowRefusesLocalMFLPlans(t *testing.T) {
	c := workflowClient(t)
	before := c.invoke("native.assets.read", "read", []byte(`{"league":"native.hosted"}`), 200).Data
	for _, command := range []string{"mfl.pool.read", "mfl.pool.refresh", "mfl.bid.plans", "mfl.bid.preview", "bid.submit", "mfl.bid.submit", "mfl.bid.revise", "mfl.bid.handoff", "mfl.bid.posted", "mfl.bid.withdraw", "SetMFLKey"} {
		raw := serverhttp.RawWorkflow{Command: command, Version: "1", Mode: "execute", Arguments: json.RawMessage(`{}`)}
		if err := raw.Validate(); err == nil {
			t.Fatalf("hosted workflow accepts %s", command)
		}
		body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
		if err != nil {
			t.Fatal(err)
		}
		c.f.post(t, serverhttp.WorkflowPath, string(body), c.bearer, c.proof, 400)
	}
	after := c.invoke("native.assets.read", "read", []byte(`{"league":"native.hosted"}`), 200).Data
	if string(before) != string(after) {
		t.Fatal("rejected MFL commands changed hosted assets")
	}
}
