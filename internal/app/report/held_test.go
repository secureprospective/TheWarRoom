package report

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/offline"
)

// A real refused connection, wrapped the way startup wraps it, reads as Offline alone.
func TestPlainFailureHidesNetworkChain(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	resp, cause := (&http.Client{Timeout: 5 * time.Second}).Get("http://" + addr + "/export")
	if resp != nil {
		_ = resp.Body.Close()
	}
	if cause == nil {
		t.Fatal("closed port answered")
	}
	err = fmt.Errorf("startup: initialize rulebook: %w", fmt.Errorf("league: discover season from 2026: %w", errors.Join(cause, errors.New("previous"))))
	plain := offline.Plain(err)
	if plain.Error() != offline.Sentence("127.0.0.1") || !errors.Is(plain, cause) || offline.Plain(plain) != plain {
		t.Fatalf("plain=%q", plain)
	}
	var h HeldFeed[int]
	HoldFeed(&h, 0, domain.Freshness{}, err)
	if h.RefreshError != plain.Error() || strings.Contains(h.RefreshError, addr) {
		t.Fatalf("held note: %q", h.RefreshError)
	}
	other := errors.New("league: unexpected export shape")
	if offline.Plain(other) != other || FailureText(other) != other.Error() {
		t.Fatal("a non-network failure was rewritten")
	}
}

// The frontend finds Go's sentences by their advice, so the two copies must match.
func TestOfflineSentenceMatchesFrontend(t *testing.T) {
	body, err := os.ReadFile("../../../frontend/src/app/data/plainFailure.ts")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"`+offline.Advice+`"`) || !strings.Contains(string(body), `"Can't reach the league site." + ADVICE`) ||
		Offline != "Can't reach the league site."+offline.Advice {
		t.Fatal("frontend ADVICE or OFFLINE differs from internal/offline")
	}
}

func TestScopedArchiveBodiesRetainOnlyRequestedParameter(t *testing.T) {
	t.Parallel()
	reads := 0
	bodies := func(_ context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error) {
		if part != "TYPE=test" {
			t.Fatal(part)
		}
		for _, src := range []string{"%", "https://fixture.invalid/?W=1", "https://fixture.invalid/?W=2"} {
			if accept(src, []byte("body"), time.Time{}) {
				return true, nil
			}
		}
		return false, nil
	}
	found, err := ScopedBodies(bodies, "W", "2")(t.Context(), "TYPE=test", func(src string, body []byte, _ time.Time) bool {
		reads++
		if src != "https://fixture.invalid/?W=2" || string(body) != "body" {
			t.Fatal(src)
		}
		return true
	})
	if err != nil || !found || reads != 1 {
		t.Fatalf("scope %v %v reads %d", found, err, reads)
	}
}

func TestReadHeldArchiveAvailabilityAndFailedBody(t *testing.T) {
	source := ArchivedFeed[string]{Export: "injuries", Leagueless: true,
		Parse: func([]byte) ([]string, error) { return nil, errors.New("bad body") },
		Bodies: func(_ context.Context, _ string, accept func(string, []byte, time.Time) bool) (bool, error) {
			if accept("https://api.myfantasyleague.com/2026/export?TYPE=injuries", nil, time.Now()) {
				t.Fatal("accepted failed parse")
			}
			return false, nil
		},
	}
	if got := ReadHeldArchive(context.Background(), false, 2026, source); got.RefreshError != "No held archive" {
		t.Fatal(got)
	}
	if got := ReadHeldArchive(context.Background(), true, 2026, source); got.RefreshError != "No usable held injuries export" {
		t.Fatal(got)
	}
}
