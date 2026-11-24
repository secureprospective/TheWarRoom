package mflread

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// The league-export cache must keep the MFL client's body cap: a reply past it is refused, not held.
func TestOnceLeagueTransportRefusesOversizedBody(t *testing.T) {
	calls := 0
	huge := bytes.Repeat([]byte("x"), mfl.MaxBodyBytes+1)
	tr := &onceLeagueTransport{base: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{},
			Body: io.NopCloser(bytes.NewReader(huge)), Request: r}, nil
	})}
	req, err := http.NewRequest(http.MethodGet, "https://api.myfantasyleague.com/2026/export?TYPE=league&L=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := tr.RoundTrip(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if !errors.Is(err, mfl.ErrBodyTooLarge) {
		t.Fatalf("oversized league export: err=%v, want mfl.ErrBodyTooLarge", err)
	}
	if tr.body != nil {
		t.Fatal("an oversized body was held for reuse")
	}
	if calls != 1 {
		t.Fatalf("calls=%d want 1", calls)
	}
}
