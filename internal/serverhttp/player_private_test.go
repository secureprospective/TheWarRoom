package serverhttp_test

import (
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestHostedPlayerNotesAndWatchesStayWithAuthenticatedFranchise(t *testing.T) {
	c := workflowClient(t)
	const player = "0101"
	c.invoke(leagueservice.PlayerNoteSet, "execute", []byte(`{"league":"native.hosted","player":"0101","text":"owner one private note"}`), 200)
	c.invoke(leagueservice.PlayerWatchSet, "execute", []byte(`{"league":"native.hosted","player":"0101","watched":true}`), 200)
	read := func(client hostedWorkflowClient) leagueservice.PlayerNotes {
		t.Helper()
		response := client.invoke(leagueservice.PlayerNoteRead, "read", []byte(`{"league":"native.hosted","player":"0101"}`), 200)
		var notes leagueservice.PlayerNotes
		if err := json.Unmarshal(response.Data, &notes); err != nil {
			t.Fatal(err)
		}
		return notes
	}
	first := read(c)
	if first.Player != player || first.Count != 1 || first.Notes[0].Text != "owner one private note" || !first.Watched {
		t.Fatalf("owner one private data = %+v", first)
	}

	other := c
	other.bearer, other.proof = c.f.login(t, "owner", "private-password"), ""
	second := read(other)
	if second.Count != 0 || second.Watched {
		t.Fatalf("another franchise read owner one's private data: %+v", second)
	}
	other.invoke(leagueservice.PlayerNoteSet, "execute", []byte(`{"league":"native.hosted","player":"0101","text":"owner two private note"}`), 200)
	other.invoke(leagueservice.PlayerWatchSet, "execute", []byte(`{"league":"native.hosted","player":"0101","watched":true}`), 200)
	if second = read(other); second.Count != 1 || second.Notes[0].Text != "owner two private note" || !second.Watched {
		t.Fatalf("owner two private data = %+v", second)
	}
	if first = read(c); first.Count != 1 || first.Notes[0].Text != "owner one private note" || !first.Watched {
		t.Fatalf("owner two changed owner one's private data: %+v", first)
	}

	spoof := workflowRequest(t, leagueservice.PlayerNoteSet, "execute",
		`{"league":"native.hosted","player":"0101","franchise":"franchise.01","text":"spoofed"}`, c.football)
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &spoof})
	if err != nil {
		t.Fatal(err)
	}
	c.f.post(t, serverhttp.WorkflowPath, string(body), other.bearer, "", 400)
	if first = read(c); first.Notes[0].Text != "owner one private note" {
		t.Fatalf("spoofed franchise changed private note: %+v", first)
	}
}
