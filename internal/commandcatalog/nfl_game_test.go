package catalog

import (
	"encoding/json"
	"testing"
)

func TestNFLGameReadValidatesScope(t *testing.T) {
	for _, tc := range []struct {
		args  string
		valid bool
	}{
		{`{"league":"native.test","game":"401"}`, true},
		{`{"league":"native.test"}`, false},
		{`{"league":"native.test","game":"../401"}`, false},
		{`{"league":"native.test","game":"401","franchise":"private"}`, false},
	} {
		raw := RawWorkflow{Command: "nfl.game.read", Version: "1", Mode: "read", Arguments: json.RawMessage(tc.args)}
		if err := raw.Validate(); (err == nil) != tc.valid {
			t.Errorf("%s: %v", tc.args, err)
		}
	}
}
