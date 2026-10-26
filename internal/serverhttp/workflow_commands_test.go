package serverhttp

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Frozen from the pre-spine workflow allowlist, not generated from the new registry.
type workflowBefore struct {
	Command string   `json:"command"`
	Modes   []string `json:"modes"`
}

func TestWorkflowModesAndUnknownCommand(t *testing.T) {
	data, err := os.ReadFile("testdata/workflow-before.json")
	if err != nil {
		t.Fatal(err)
	}
	var before []workflowBefore
	if err := json.Unmarshal(data, &before); err != nil {
		t.Fatal(err)
	}
	if len(before) != 42 {
		t.Fatal("pre-migration workflow fixture changed")
	}
	for _, old := range before {
		for _, mode := range []string{"read", "preview", "execute", "unknown"} {
			t.Run(old.Command+"/"+mode, func(t *testing.T) {
				raw := RawWorkflow{Command: old.Command, Version: "1", Mode: mode, Arguments: json.RawMessage(`{}`)}
				err := raw.Validate()
				allowed := false
				for _, want := range old.Modes {
					allowed = allowed || mode == want
				}
				// An empty object must reach the existing Raw/domain validator only in the old modes.
				wrongMode := err != nil && (err.Error() == "this is a reading only" || err.Error() == "check or confirm the change" || err.Error() == "confirm your rookie pick")
				if allowed == wrongMode || err == nil {
					t.Fatalf("mode exposure changed: allowed=%v err=%v", allowed, err)
				}
				raw.Version = "2"
				if err := raw.Validate(); err == nil || !strings.Contains(err.Error(), "update the app") {
					t.Fatal("unknown version admitted")
				}
			})
		}
	}
	if err := (RawWorkflow{Command: "unregistered", Version: "1", Mode: "execute"}).Validate(); err == nil || err.Error() != "this league action is not available here" {
		t.Fatal("unknown command admitted")
	}
	if _, err := runWorkflow(t.Context(), nil, RawWorkflow{Command: "unregistered", Version: "1"}); err == nil {
		t.Fatal("dispatch bypassed validation")
	}
}
