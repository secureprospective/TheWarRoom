package serverhttp_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestHostedReadingsGzipOnlyWhenAccepted(t *testing.T) {
	f := newHostedFixture(t, true)
	bearer, proof := f.directCommissioner(t)
	send := func(raw serverhttp.RawWorkflow, encoding string) (http.Header, []byte) {
		t.Helper()
		body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
		if err != nil {
			t.Fatal(err)
		}
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.server.URL+serverhttp.WorkflowPath, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Authorization", "Bearer "+bearer)
		request.Header.Set("X-TWR-Recent-Auth", proof)
		request.Header.Set("Accept-Encoding", encoding)
		response, err := f.server.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = response.Body.Close() }()
		got, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != http.StatusOK {
			t.Fatalf("status %d %v: %s", response.StatusCode, err, got)
		}
		return response.Header, got
	}
	read := workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, "")
	plainResponse, plain := send(read, "identity")
	if plainResponse.Get("Content-Encoding") != "" || len(plain) < 1024 {
		t.Fatalf("plain reading: encoding %q, %d bytes", plainResponse.Get("Content-Encoding"), len(plain))
	}
	for _, refused := range []string{"gzip;q=0", "br"} {
		if r, _ := send(read, refused); r.Get("Content-Encoding") != "" {
			t.Fatalf("%s: compressed anyway", refused)
		}
	}
	zipped, packed := send(read, "br, gzip")
	if zipped.Get("Content-Encoding") != "gzip" || !strings.Contains(zipped.Get("Vary"), "Accept-Encoding") || len(packed) >= len(plain) {
		t.Fatalf("gzip reading: encoding %q vary %q, %d of %d bytes", zipped.Get("Content-Encoding"), zipped.Get("Vary"), len(packed), len(plain))
	}
	zr, err := gzip.NewReader(bytes.NewReader(packed))
	if err != nil {
		t.Fatal(err)
	}
	unpacked, err := io.ReadAll(zr)
	if err != nil || !bytes.Equal(unpacked, plain) {
		t.Fatalf("decompressed reading differs from the plain one (%v)", err)
	}
	var reading serverhttp.WorkflowResponse
	if err := json.Unmarshal(plain, &reading); err != nil {
		t.Fatal(err)
	}
	var assets commissioner.NativeAssets
	if err := json.Unmarshal(reading.Data, &assets); err != nil {
		t.Fatal(err)
	}
	command, err := json.Marshal(commissioner.RawCommand{League: "native.hosted", Command: commissioner.RosterReserve,
		Args:     commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0100", Franchise: "franchise.01", Bucket: "ir"}},
		Expected: assets.Revision, Request: "reserve.gzip", Effective: "now", Visibility: "league"})
	if err != nil {
		t.Fatal(err)
	}
	preview, body := send(workflowRequest(t, commissioner.RosterReserve, "preview", string(command), reading.Football), "gzip")

	if preview.Get("Content-Encoding") != "" || preview.Get("Vary") != "" || len(body) == 0 {
		t.Fatalf("preview went through the compressing writer: encoding %q vary %q", preview.Get("Content-Encoding"), preview.Get("Vary"))
	}
}

func TestJoinClaimRaceExactlyOneWins(t *testing.T) {
	f := newHostedFixture(t, true)
	code, err := f.join.Rotate(t.Context(), "native.hosted", governance.Principal(f.principal))
	if err != nil {
		t.Fatal(err)
	}
	statuses := make([]int, 2)
	var wg sync.WaitGroup
	for i := range statuses {
		wg.Go(func() {
			statuses[i], _ = f.send(t, serverhttp.JoinClaimPath, joinBody(code.Code, "franchise.05", fmt.Sprintf("racer%d", i)), "", "")
		})
	}
	wg.Wait()
	won, lost := 0, 0
	for _, status := range statuses {
		switch status {
		case http.StatusCreated:
			won++
		case http.StatusConflict:
			lost++
		}
	}
	if won != 1 || lost != 1 {
		t.Fatalf("statuses = %v, want one 201 and one 409", statuses)
	}
	accountsMade := 0
	for i := range statuses {
		if _, err := accounts.New(f.accounts).Authenticate(t.Context(), accounts.RawCredentials{Username: fmt.Sprintf("racer%d", i), Password: "private-password"}); err == nil {
			accountsMade++
		}
	}
	seats := 0
	events, err := f.history.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if c := event.Payload.Commissioner; c != nil && c.Command.Command == "commish.seat.replace" && c.Command.Args.People.Franchise == "franchise.05" {
			seats++
		}
	}
	if accountsMade != 1 || seats != 1 {
		t.Fatalf("accounts=%d seat events=%d, want 1 and 1", accountsMade, seats)
	}
}

func TestHostedWorkflowRefusesLocalMFLPlans(t *testing.T) {
	c := workflowClient(t)
	before := c.invoke("native.assets.read", "read", []byte(`{"league":"native.hosted"}`), 200).Data
	for _, command := range []string{"mfl.pool.read", "mfl.pool.refresh", "mfl.bid.plans", "mfl.bid.preview", "bid.submit", "mfl.bid.submit", "mfl.bid.revise", "mfl.bid.handoff", "mfl.bid.posted", "mfl.bid.withdraw", "SetMFLKey"} {
		raw := serverhttp.RawWorkflow{Command: command, Version: "1", Mode: "execute", Arguments: json.RawMessage(`{}`)}
		if err := raw.Validate(); err == nil {
			t.Fatalf("hosted workflow accepts %s", command)
		}
		body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
		if err != nil {
			t.Fatal(err)
		}
		c.f.post(t, serverhttp.WorkflowPath, string(body), c.bearer, c.proof, 400)
	}
	after := c.invoke("native.assets.read", "read", []byte(`{"league":"native.hosted"}`), 200).Data
	if string(before) != string(after) {
		t.Fatal("rejected MFL commands changed hosted assets")
	}
}
