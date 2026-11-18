package accounts

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateHomeLayoutBoundsAndKnownBoards(t *testing.T) {
	valid := json.RawMessage(`{"mode":"custom","tiles":[["news","1x2"],["wire","1x1"]],"over":["race"],"pins":{"cap":true}}`)
	if _, err := ValidateHomeLayout(valid); err != nil {
		t.Fatalf("valid layout rejected: %v", err)
	}
	tooLarge := json.RawMessage(`{"mode":"default","tiles":[],"over":[],"pins":{}}` + strings.Repeat(" ", HomeLayoutMaxBytes))
	unknown := json.RawMessage(`{"mode":"custom","tiles":[["hidden","1x1"]],"over":[],"pins":{}}`)
	badSpan := json.RawMessage(`{"mode":"custom","tiles":[["picks","2x1"]],"over":[],"pins":{}}`)
	tooMany := json.RawMessage(`{"mode":"default","tiles":[],"over":[` + strings.TrimRight(strings.Repeat(`"due",`, 41), ",") + `],"pins":{}}`)
	unpacked := json.RawMessage(`{"mode":"custom","tiles":[["news","1x2"],["wire","1x2"],["mu","1x2"],["race","1x1"]],"over":[],"pins":{}}`)
	unknownField := json.RawMessage(`{"mode":"default","tiles":[],"over":[],"pins":{},"other":true}`)
	for name, raw := range map[string]json.RawMessage{
		"too large": tooLarge, "unknown board": unknown, "invalid span": badSpan,
		"too many entries": tooMany, "does not pack": unpacked, "unknown field": unknownField,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ValidateHomeLayout(raw); err == nil {
				t.Fatal("invalid layout accepted")
			}
		})
	}
}

func TestHomeLayoutCommandsAreNotDelegable(t *testing.T) {
	for _, command := range []string{"home.layout.read", "home.layout.set"} {
		if DelegableCommand(command) {
			t.Fatalf("agent token may carry %s", command)
		}
	}
}

func TestRawHomeLayoutSetRequiresValidRevisionAndLayout(t *testing.T) {
	layout := json.RawMessage(`{"mode":"default","tiles":[],"over":[],"pins":{}}`)
	for name, input := range map[string]RawHomeLayoutSet{
		"negative revision": {Layout: layout, ExpectedRevision: -1},
		"missing layout":    {ExpectedRevision: 0},
	} {
		t.Run(name, func(t *testing.T) {
			if err := input.Validate(); err == nil {
				t.Fatal("invalid request accepted")
			}
		})
	}
}
