package serverhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestRequestTallyWorkflow(t *testing.T) {
	f := newHostedFixture(t, true)
	bearer, proof := f.directCommissioner(t)
	raw := workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, "")
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		f.post(t, serverhttp.WorkflowPath, string(body), bearer, proof, 200)
	}
	_, counts := f.tally.Snapshot()
	got := counts["native.assets.read read"]
	if got.Calls != 2 || got.Computed != 1 || got.Cached != 1 || got.ReplyBytes == 0 {
		t.Fatalf("counts: %+v", got)
	}
	raw = workflowRequest(t, "native.assets.read", "read", `{"league":"native.other"}`, "")
	body, err = json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	f.post(t, serverhttp.WorkflowPath, string(body), bearer, proof, 403)
	raw = workflowRequest(t, "bids.read", "read", `{"league":"native.hosted","franchise":"franchise.private-sentinel"}`, "")
	body, err = json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.send(t, serverhttp.WorkflowPath, string(body), bearer, proof)
	_, counts = f.tally.Snapshot()
	got = counts["native.assets.read read"]
	if got.Refused != 1 {
		t.Fatalf("refused: %+v", got)
	}
	encoded, err := json.Marshal(counts)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "native.hosted") || strings.Contains(string(encoded), "native.other") || strings.Contains(string(encoded), "franchise.private-sentinel") {
		t.Fatalf("private input in counts: %s", encoded)
	}
	f.tally.Reset()
	_, counts = f.tally.Snapshot()
	if len(counts) != 0 {
		t.Fatalf("reset: %+v", counts)
	}
}
func TestRequestTallyConcurrent(t *testing.T) {
	tally := serverhttp.NewRequestTally()
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 1000 {
				tally.Unchanged("native.assets.read", "read")
			}
		}()
	}
	wg.Wait()
	_, counts := tally.Snapshot()
	if counts["native.assets.read read"].Unchanged != 50000 {
		t.Fatal(counts)
	}
}

func BenchmarkHostedPulseBurst(b *testing.B) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(2))
	f := newHostedFixture(b, true, true)
	seedBurstGames(f)
	phones := burstPhones(b, f)
	burst := func() {
		start := make(chan struct{})
		var wg sync.WaitGroup
		for _, phone := range phones {
			wg.Go(func() {
				<-start
				if err := phone.read(b.Context(), f.server.URL); err != nil {
					b.Error(err)
				}
			})
		}
		close(start)
		wg.Wait()
	}
	burst() // Warm each phone's TLS connection and the server's read paths.
	if b.Failed() {
		b.FailNow()
	}
	b.ReportAllocs()
	for b.Loop() {
		burst()
	}
}

func seedBurstGames(f *hostedFixture) {
	var board []livegame.Game
	for i := range 16 {
		board = append(board, livegame.Game{ID: fmt.Sprint(401 + i), Status: "pre", Kickoff: f.clockTime().Add(time.Hour),
			Home: livegame.Team{ID: "2", Abbreviation: "BUF"}, Away: livegame.Team{ID: "15", Abbreviation: "MIA"}})
	}
	board[0].Venue = &livegame.Venue{Name: "Highmark Stadium", City: "Orchard Park", State: "NY"}
	f.games.Set(board, f.clockTime(), "ESPN")
	f.games.SetDetails(map[string]leagueservice.NFLGame{"401": {Game: &board[0], At: f.clockTime()}})
}

type burstPhone struct {
	client  *http.Client
	bearer  string
	body    []byte
	command string
}

func burstPhones(b *testing.B, f *hostedFixture) []burstPhone {
	b.Helper()
	var phones []burstPhone
	for i := range 62 {
		user, franchise := "owner", "franchise.02"
		if i/6%2 == 1 {
			user, franchise = "partner", "franchise.03"
		}
		credentials := accounts.RawCredentials{Username: user, Password: "private-password"}
		session, err := f.sessions.Login(b.Context(), credentials, fmt.Sprintf("phone %d", i+1))
		if err != nil {
			b.Fatal(err)
		}
		transport := f.server.Client().Transport.(*http.Transport).Clone()
		b.Cleanup(transport.CloseIdleConnections)
		command, body := burstWorkflow(b, franchise, i)
		phones = append(phones, burstPhone{client: &http.Client{Transport: transport}, bearer: session.Bearer, body: body, command: command})
	}
	return phones
}

func burstWorkflow(b *testing.B, franchise string, i int) (string, []byte) {
	b.Helper()
	reads := [...]struct{ command, args string }{
		{"results.read", `{"league":"native.hosted","season":2026,"week":1}`},
		{"standings.read", `{"league":"native.hosted","season":2026}`},
		{"lineup.read", fmt.Sprintf(`{"league":"native.hosted","franchise":%q,"week":1,"at":""}`, franchise)},
		{"nfl.games.read", `{"league":"native.hosted"}`},
		{"nfl.game.read", `{"league":"native.hosted","game":"401"}`},
		{"nfl.game.pregame.read", `{"league":"native.hosted","game":"401"}`},
	}
	read := reads[i%len(reads)]
	raw := serverhttp.RawWorkflow{Command: read.command, Version: "1", Mode: "read", Arguments: json.RawMessage(read.args)}
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		b.Fatal(err)
	}
	return read.command, body
}

func (p burstPhone) read(ctx context.Context, baseURL string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+serverhttp.WorkflowPath, bytes.NewReader(p.body))
	if err != nil {
		return fmt.Errorf("%s request: %w", p.command, err)
	}
	request.Header.Set("Authorization", "Bearer "+p.bearer)
	request.Header.Set("Content-Type", "application/json")
	// Drain the wire bytes without adding client-side decompression to the profile.
	request.Header.Set("Accept-Encoding", "gzip")
	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("%s transport: %w", p.command, err)
	}
	defer func() { _ = response.Body.Close() }()
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		return fmt.Errorf("%s body: %w", p.command, err)
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s status %d", p.command, response.StatusCode)
	}
	return nil
}
