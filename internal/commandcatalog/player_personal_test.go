package catalog

import (
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

func TestHostedPersonalPlayerCommandsAreStrictAndUncachedWorkflows(t *testing.T) {
	arguments := map[string]json.RawMessage{
		leagueservice.PlayerNoteRead:   json.RawMessage(`{"league":"native.test","player":"0101"}`),
		leagueservice.PlayerNoteSet:    json.RawMessage(`{"league":"native.test","player":"0101","text":"private"}`),
		leagueservice.PlayerWatchRead:  json.RawMessage(`{"league":"native.test"}`),
		leagueservice.PlayerWatchSet:   json.RawMessage(`{"league":"native.test","player":"0101","watched":true}`),
		leagueservice.PlayerWatchOrder: json.RawMessage(`{"league":"native.test","player":"0101","with":"0102"}`),
		leagueservice.WhatIfStackRead:  json.RawMessage(`{"league":"native.test","viewerZone":"UTC","moves":[]}`),
	}
	commands := map[string]Command{}
	for _, command := range NativeCommands(Schema{}) {
		commands[command.ID] = command
	}
	for id, args := range arguments {
		command, ok := commands[id]
		if !ok || command.Availability != "server" || command.Gravity != "G0" || command.Permission != "franchise-owner" {
			t.Fatalf("private command metadata missing or broad: %+v", command)
		}
		mode := Execute
		if id == leagueservice.PlayerNoteRead || id == leagueservice.PlayerWatchRead || id == leagueservice.WhatIfStackRead {
			mode = Read
		}
		if len(command.Modes) != 1 || command.Modes[0] != mode {
			t.Fatalf("%s modes = %v", id, command.Modes)
		}
		if id == leagueservice.WhatIfStackRead &&
			(command.Facts != "private-player-data" || command.Safeguards != "authenticated seated franchise; no request franchise; uncached private result") {
			t.Fatalf("private what-if metadata is weak: %+v", command)
		}
		request := RawWorkflow{Version: "1", Command: id, Mode: string(mode), Arguments: args}
		if err := request.Validate(); err != nil {
			t.Fatalf("%s valid request refused: %v", id, err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(args, &fields); err != nil {
			t.Fatal(err)
		}
		fields["franchise"] = json.RawMessage(`"franchise.02"`)
		spoofed, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		request.Arguments = spoofed
		if err := request.Validate(); err == nil {
			t.Fatalf("%s accepted request-selected franchise", id)
		}
	}
}
