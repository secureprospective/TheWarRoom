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
	c.invoke(leagueservice.PlayerWatchSet, "execute", []byte(`{"league":"native.hosted","player":"0102","watched":true}`), 200)
	c.invoke(leagueservice.PlayerWatchOrder, "execute", []byte(`{"league":"native.hosted","player":"0102","with":"0101"}`), 200)
	read := func(client hostedWorkflowClient) leagueservice.PlayerNotes {
		t.Helper()
		response := client.invoke(leagueservice.PlayerNoteRead, "read", []byte(`{"league":"native.hosted","player":"0101"}`), 200)
		var notes leagueservice.PlayerNotes
		if err := json.Unmarshal(response.Data, &notes); err != nil {
			t.Fatal(err)
		}
		return notes
	}
	readWatches := func(client hostedWorkflowClient) leagueservice.PlayerWatchList {
		t.Helper()
		response := client.invoke(leagueservice.PlayerWatchRead, "read", []byte(`{"league":"native.hosted"}`), 200)
		var watches leagueservice.PlayerWatchList
		if err := json.Unmarshal(response.Data, &watches); err != nil {
			t.Fatal(err)
		}
		return watches
	}
	first := read(c)
	if first.Player != player || first.Count != 1 || first.Notes[0].Text != "owner one private note" || !first.Watched {
		t.Fatalf("owner one private data = %+v", first)
	}
	if watches := readWatches(c); len(watches.Players) != 2 || watches.Players[0] != "0102" || watches.Players[1] != "0101" {
		t.Fatalf("owner one saved watch order = %+v", watches)
	}

	other := c
	other.bearer, other.proof = c.f.login(t, "owner", "private-password"), ""
	second := read(other)
	if second.Count != 0 || second.Watched {
		t.Fatalf("another franchise read owner one's private data: %+v", second)
	}
	if watches := readWatches(other); len(watches.Players) != 0 {
		t.Fatalf("another franchise read owner one's targets: %+v", watches)
	}
	other.invoke(leagueservice.PlayerNoteSet, "execute", []byte(`{"league":"native.hosted","player":"0101","text":"owner two private note"}`), 200)
	other.invoke(leagueservice.PlayerWatchSet, "execute", []byte(`{"league":"native.hosted","player":"0101","watched":false}`), 200)
	other.invoke(leagueservice.PlayerWatchOrder, "execute", []byte(`{"league":"native.hosted","player":"0101","with":"0102"}`), 409)
	other.invoke(leagueservice.PlayerWatchSet, "execute", []byte(`{"league":"native.hosted","player":"0101","watched":true}`), 200)
	if second = read(other); second.Count != 1 || second.Notes[0].Text != "owner two private note" || !second.Watched {
		t.Fatalf("owner two private data = %+v", second)
	}
	if watches := readWatches(other); len(watches.Players) != 1 || watches.Players[0] != "0101" {
		t.Fatalf("owner two private targets = %+v", watches)
	}
	if first = read(c); first.Count != 1 || first.Notes[0].Text != "owner one private note" || !first.Watched {
		t.Fatalf("owner two changed owner one's private data: %+v", first)
	}
	if watches := readWatches(c); len(watches.Players) != 2 || watches.Players[0] != "0102" || watches.Players[1] != "0101" {
		t.Fatalf("owner two changed owner one's target order: %+v", watches)
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
