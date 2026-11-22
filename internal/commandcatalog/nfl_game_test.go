package catalog

import (
	"encoding/json"
	"testing"
)

func TestNFLGameReadValidatesScope(t *testing.T) {
	for _, cmd := range []string{"nfl.game.read", "nfl.game.pregame.read"} {
		for _, tc := range []struct {
			args  string
			valid bool
		}{
			{`{"league":"native.test","game":"401"}`, true},
			{`{"league":"native.test"}`, false},
			{`{"league":"native.test","game":"../401"}`, false},
			{`{"league":"native.test","game":"401","franchise":"private"}`, false},
		} {
			raw := RawWorkflow{
				Command: cmd, Version: "1", Mode: "read", Arguments: json.RawMessage(tc.args),
			}
			if err := raw.Validate(); (err == nil) != tc.valid {
				t.Errorf("%s %s: %v", cmd, tc.args, err)
			}
		}
	}
}

func TestNFLGamePregameReadCatalogEntry(t *testing.T) {
	var found bool
	for _, command := range NativeCommands(Schema{}) {
		if command.ID != "nfl.game.pregame.read" {
			continue
		}
		found = true
		if command.Method != "TargetNFLGamePregame" || command.Availability != "server" ||
			command.Permission != "league-view" || len(command.Modes) != 1 || command.Modes[0] != Read {
			t.Fatalf("pregame catalog entry = %+v", command)
		}
	}
	if !found {
		t.Fatal("nfl.game.pregame.read missing from native catalog")
	}
	if _, ok := WorkflowCommands()["nfl.game.pregame.read"]; !ok {
		t.Fatal("nfl.game.pregame.read missing from hosted workflow")
	}
}
