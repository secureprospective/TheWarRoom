package composition

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

type blockingNewsTransport struct {
	entered chan struct{}
	release chan struct{}
}

func (f blockingNewsTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Query().Get("TYPE") != "playerProfile" {
		return nil, errors.New("unexpected fixture request")
	}
	close(f.entered)
	select {
	case <-f.release:
	case <-r.Context().Done():
		return nil, fmt.Errorf("fixture news: %w", r.Context().Err())
	}
	body := []byte(`{"playerProfiles":{"playerProfile":{"id":"0042","name":"Fixture Player","news":{}}}}`)
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(body)), Request: r}, nil
}

func TestResearchNewsDoesNotBlockNowAndRejectsConcurrentNews(t *testing.T) {
	t.Parallel()
	a, f, done := blockedResearchNews(t)
	defer close(f.release)
	other := make(chan error, 1)
	go func() { other <- researchAPI(a).TargetResearchNewsRefresh("") }()
	select {
	case err := <-other:
		if err == nil || !strings.Contains(err.Error(), "already refreshing") {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("second news refresh waited")
	}
	go func() { other <- a.nowAPI().TargetNowRefresh(0, false) }()
	select {
	case err := <-other:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Now refresh blocked behind news")
	}
	f.release <- struct{}{}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func blockedResearchNews(t *testing.T) (*App, blockingNewsTransport, <-chan error) {
	t.Helper()
	a := targetTestApp(t)
	f := blockingNewsTransport{entered: make(chan struct{}), release: make(chan struct{})}
	client, err := mfl.New("api", 10000, mfl.WithTransport(f))
	if err != nil {
		t.Fatal(err)
	}
	a.core.MFLClient = client
	a.core.SeasonPause = func(context.Context, time.Duration) error { return nil }
	a.core.FallbackLog = func(string) {}
	at := time.Now()
	h := a.raceHeld.Snapshot()
	h.Weeks.FetchedAt, h.Brackets.FetchedAt, h.Standings.FetchedAt = at, at, at
	a.raceHeld.PublishWeeks(h.Weeks)
	a.raceHeld.PublishBrackets(h.Brackets)
	a.raceHeld.PublishStandings(h.Standings, h.Power, true)
	a.core.Week.Number = 0
	done := make(chan error, 1)
	go func() { done <- researchAPI(a).TargetResearchNewsRefresh("") }()
	select {
	case <-f.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("news transport never entered")
	}
	return a, f, done
}
