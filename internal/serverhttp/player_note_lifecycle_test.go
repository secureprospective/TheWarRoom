package serverhttp_test

import (
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

func TestHostedPlayerNoteIsPrivateAndEditable(t *testing.T) {
	ownerA := workflowClient(t)
	const player = "0101"
	read := func(client hostedWorkflowClient) leagueservice.PlayerNotes {
		t.Helper()
		response := client.invoke(leagueservice.PlayerNoteRead, "read", []byte(`{"league":"native.hosted","player":"0101"}`), 200)
		var notes leagueservice.PlayerNotes
		if err := json.Unmarshal(response.Data, &notes); err != nil {
			t.Fatal(err)
		}
		return notes
	}
	ownerA.invoke(leagueservice.PlayerNoteSet, "execute", []byte(`{"league":"native.hosted","player":"0101","text":"first private note"}`), 200)
	if got := read(ownerA); got.Player != player || got.Count != 1 || len(got.Notes) != 1 || got.Notes[0].Text != "first private note" {
		t.Fatalf("owner A readback = %+v", got)
	}

	ownerB := ownerA
	ownerB.bearer, ownerB.proof = ownerA.f.login(t, "owner", "private-password"), ""
	if got := read(ownerB); got.Count != 0 || len(got.Notes) != 0 {
		t.Fatalf("owner B saw owner A's note: %+v", got)
	}

	old := read(ownerA).Notes[0].At
	// An edit is what the phone note field sends (shell/playerCardParts.tsx): add the new text, then remove the old note by its time.
	ownerA.invoke(leagueservice.PlayerNoteSet, "execute", []byte(`{"league":"native.hosted","player":"0101","text":"edited private note"}`), 200)
	ownerA.invoke(leagueservice.PlayerNoteSet, "execute", []byte(`{"league":"native.hosted","player":"0101","text":"","at":"`+old+`"}`), 200)
	if got := read(ownerA); got.Count != 1 || len(got.Notes) != 1 || got.Notes[0].Text != "edited private note" {
		t.Fatalf("edited note was not replaced: %+v", got)
	}
	if got := read(ownerB); got.Count != 0 || len(got.Notes) != 0 {
		t.Fatalf("owner B saw owner A's edited note: %+v", got)
	}
}
