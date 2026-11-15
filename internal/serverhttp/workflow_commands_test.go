package serverhttp

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
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
	if len(before) != 93 {
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

type admitAll struct{}

func (admitAll) Admit(context.Context, commissioner.Access) error { return nil }

// TestHostedAdmissionKeepsCreationLocal: the hosted creation rule (accountAdmission.Admit) refuses a
// league with no records, so a signed-in commissioner cannot start a league on the server.
func TestHostedAdmissionKeepsCreationLocal(t *testing.T) {
	gate := accountAdmission{sessions: admitAll{}, totp: &accounts.TOTPService{}}
	create := commissioner.RawCommand{League: "native.new", Command: commissioner.Create}
	err := gate.Admit(t.Context(), commissioner.Access{League: "native.new", Raw: create, Records: []commissioner.Record{}})
	if err == nil || err.Error() != "league access unavailable" {
		t.Fatalf("hosted creation: %v", err)
	}
}
