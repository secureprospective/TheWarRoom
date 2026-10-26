package serverhttp

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// Frozen from the pre-spine workflow allowlist, not generated from the new registry.
type workflowBefore struct {
	Command string   `json:"command"`
	Modes   []string `json:"modes"`
}

func workflowParity(entries map[string]workflowCommand, before []workflowBefore) error {
	if len(entries) != len(before) {
		return fmt.Errorf("workflow command exposure changed: %d != %d", len(entries), len(before))
	}
	for _, old := range before {
		entry, ok := entries[old.Command]
		if !ok || entry.validate == nil || entry.run == nil {
			return fmt.Errorf("workflow command %s has no executable entry", old.Command)
		}
	}
	return nil
}

func TestWorkflowRegistryParity(t *testing.T) {
	data, err := os.ReadFile("testdata/workflow-before.json")
	if err != nil {
		t.Fatal(err)
	}
	var before []workflowBefore
	if err := json.Unmarshal(data, &before); err != nil {
		t.Fatal(err)
	}
	entries := workflowCommands()
	if len(before) != 42 {
		t.Fatal("pre-migration workflow fixture changed")
	}
	if err := workflowParity(entries, before); err != nil {
		t.Fatal(err)
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
	// Prove the parity gate detects a missing entry, a side door and a missing handler.
	for _, defect := range []string{"missing", "side-door", "handler"} {
		bad := workflowCommands()
		switch defect {
		case "missing":
			delete(bad, before[0].Command)
		case "side-door":
			bad["unregistered"] = bad[before[0].Command]
		case "handler":
			bad[before[0].Command] = workflowCommand{}
		}
		if err := workflowParity(bad, before); err == nil {
			t.Fatalf("gate accepted %s", defect)
		}
	}
	if err := (RawWorkflow{Command: "unregistered", Version: "1", Mode: "execute"}).Validate(); err == nil || err.Error() != "this league action is not available here" {
		t.Fatal("unknown command admitted")
	}
	if _, err := runWorkflow(t.Context(), nil, RawWorkflow{Command: "unregistered", Version: "1"}); err == nil {
		t.Fatal("dispatch bypassed validation")
	}
}
