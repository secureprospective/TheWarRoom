package report

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
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
	plain := PlainFailure(err)
	if plain.Error() != Offline || !errors.Is(plain, cause) || PlainFailure(plain) != plain {
		t.Fatalf("plain=%q", plain)
	}
	var h HeldFeed[int]
	HoldFeed(&h, 0, domain.Freshness{}, err)
	if h.RefreshError != Offline || strings.Contains(h.RefreshError, addr) {
		t.Fatalf("held note: %q", h.RefreshError)
	}
	other := errors.New("league: unexpected export shape")
	if PlainFailure(other) != other || FailureText(other) != other.Error() {
		t.Fatal("a non-network failure was rewritten")
	}
}

// The frontend trims Go's context from this exact sentence, so the two copies must match.
func TestOfflineSentenceMatchesFrontend(t *testing.T) {
	body, err := os.ReadFile("../../../frontend/src/app/data/plainFailure.ts")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"`+Offline+`"`) {
		t.Fatal("frontend OFFLINE differs from report.Offline")
	}
}
