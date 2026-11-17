package catalog

import (
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

func TestHostedPersonalPlayerCommandsAreStrictAndUncachedWorkflows(t *testing.T) {
	arguments := map[string]json.RawMessage{
		leagueservice.PlayerNoteRead: json.RawMessage(`{"league":"native.test","player":"0101"}`),
		leagueservice.PlayerNoteSet:  json.RawMessage(`{"league":"native.test","player":"0101","text":"private"}`),
		leagueservice.PlayerWatchSet: json.RawMessage(`{"league":"native.test","player":"0101","watched":true}`),
	}
	commands := map[string]Command{}
	for _, command := range NativeCommands(Schema{}) {
		commands[command.ID] = command
	}
	for id, args := range arguments {
		command, ok := commands[id]
		if !ok || command.Availability != "both" || command.Gravity != "G0" || command.Permission != "franchise-owner" {
			t.Fatalf("private command metadata missing or broad: %+v", command)
		}
		mode := Execute
		if id == leagueservice.PlayerNoteRead {
			mode = Read
		}
		if len(command.Modes) != 1 || command.Modes[0] != mode {
			t.Fatalf("%s modes = %v", id, command.Modes)
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
