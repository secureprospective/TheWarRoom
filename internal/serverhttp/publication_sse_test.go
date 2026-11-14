package serverhttp_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func openV2(t *testing.T, f *hostedFixture, bearer string) <-chan string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, f.server.URL+serverhttp.InvalidationPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("X-TWR-League", "native.hosted")
	req.Header.Set("X-TWR-Invalidation", "2")
	response, err := f.server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	if response.StatusCode != 200 {
		t.Fatalf("v2 status: %d", response.StatusCode)
	}
	return streamEvents(response)
}

func nextV2(t *testing.T, events <-chan string) serverhttp.Invalidation {
	t.Helper()
	select {
	case frame := <-events:
		var message serverhttp.Invalidation
		data := strings.TrimSuffix(strings.TrimPrefix(frame, "event: invalidate\ndata: "), "\n\n")
		if err := json.Unmarshal([]byte(data), &message); err != nil {
			t.Fatalf("frame %q: %v", frame, err)
		}
		if message.Protocol != 2 || message.ServerEpoch == "" || message.League != "native.hosted" {
			t.Fatalf("v2: %+v", message)
		}
		return message
	case <-time.After(3 * time.Second):
		t.Fatal("missing v2 event")
	}
	return serverhttp.Invalidation{}
}

func TestSSEV2PublicPublicationAndPrivateVisibility(t *testing.T) {
	f := newHostedFixture(t)
	bearer := streamOwner(t, f)
	events := openV2(t, f, bearer)
	initial := nextV2(t, events)
	if len(initial.Topics) != 1 || initial.Topics[0].Kind != "league" {
		t.Fatalf("initial topics: %+v", initial)
	}
	appendPrivateRole(t, f)
	expectQuiet(t, events)
	f.publisher.PublishScores("native.hosted", 2026, 10)
	score := nextV2(t, events)
	if len(score.Topics) != 1 || score.Topics[0].Kind != "scores" || score.Topics[0].Week != 10 || score.Topics[0].Version != "1" {
		t.Fatalf("score topics: %+v", score)
	}
	appendStreamNote(t, f, "league")
	league := nextV2(t, events)
	if len(league.Topics) != 1 || league.Topics[0].Kind != "league" {
		t.Fatalf("view change moved public topics: %+v", league)
	}
}

func TestSSEV2LineupChangeBroadAuthorizedScope(t *testing.T) {
	f := newHostedFixture(t, true)
	bearer := f.login(t, "commissioner", "private-password")
	events := openV2(t, f, bearer)
	_ = nextV2(t, events)
	head, err := f.history.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	command := commissioner.RawCommand{League: "native.hosted", Command: commissioner.LineupSet,
		Args:     commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: 2026, Franchise: "franchise.01", Week: 1, Starters: []string{"0099"}}},
		Expected: int64(len(head)), Request: "lineup.v2", Effective: "now", Visibility: "private"}
	if _, err := f.history.ExecuteCommissioner(t.Context(), command, f.principal, nil); err != nil {
		t.Fatal(err)
	}
	message := nextV2(t, events)
	if len(message.Topics) != 1 || message.Topics[0].Kind != "league" {
		t.Fatalf("lineup broad hint: %+v", message)
	}
}
