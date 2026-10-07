package mfl

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/mflkey"
)

type keyedTransport struct{ seen []string }

func (f *keyedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.seen = append(f.seen, r.URL.Query().Get("APIKEY"))
	body := `{"pendingTrades":{}}`
	if r.URL.Query().Get("TYPE") == "league" {
		body = `{"league":{"baseURL":"https://www47.myfantasyleague.com"}}`
	}
	return &http.Response{
		StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{},
	}, nil
}

func TestKeySourceCopyAndRequestIsolation(t *testing.T) {
	base := &keyedTransport{}
	original, err := New("api", 10000, WithTransport(base), WithKeySource(
		func(context.Context) (mflkey.Key, error) { return "stored-key", nil }))
	if err != nil {
		t.Fatal(err)
	}
	candidate := original.WithKeySource(func(context.Context) (mflkey.Key, error) {
		return "candidate-key", nil
	})
	if candidate.http != original.http || candidate.limiter != original.limiter ||
		candidate.hostState != original.hostState {
		t.Fatal("copy did not share transport state")
	}
	ctx := context.Background()
	req := Request{Type: "pendingTrades", Year: "2026", Params: map[string]string{"L": "league"}, Keyed: true}
	if _, err := candidate.Do(ctx, req); err == nil {
		t.Fatal("accepted undiscovered host")
	}
	if err := candidate.DiscoverHost(ctx, req.Year, "league"); err != nil {
		t.Fatal(err)
	}
	if _, err := candidate.Do(ctx, req); err != nil {
		t.Fatal(err)
	}
	if _, err := original.Do(ctx, req); err != nil {
		t.Fatal(err)
	}
	if base.seen[1] != "candidate-key" || base.seen[2] != "stored-key" {
		t.Fatal(base.seen)
	}
	if len(req.Params) != 1 {
		t.Fatal("mutated caller params")
	}
	noSource, err := New("api", 10000)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := noSource.Do(ctx, req); err == nil {
		t.Fatal("accepted absent source")
	}
}

func TestScrubURLError(t *testing.T) {
	cause := errors.New("failed private-value")
	original := &url.Error{Op: "Get", URL: "https://example.test/?APIKEY=private-value", Err: cause}
	err := scrubError(original, "https://example.test/export", "private-value")
	if strings.Contains(err.Error(), "private-value") {
		t.Fatal("leaked error")
	}
	var got *url.Error
	if !errors.As(err, &got) || got.URL != "https://example.test/export" || got.Op != "Get" {
		t.Fatal("url error lost safe URL or operation")
	}
	if !errors.Is(err, cause) {
		t.Fatal("lost underlying cause")
	}
}
