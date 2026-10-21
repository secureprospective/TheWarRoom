package coretest

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// Transport serves only in-memory fixtures and records calls safely during concurrent tests.
type Transport struct {
	mu              sync.Mutex
	bodies          map[string]string
	requests, keyed []string
	failure         string
}

func NewTransport(t *testing.T, bodies map[string]string) (*Transport, *mfl.Client) {
	t.Helper()
	transport := &Transport{bodies: bodies}
	client, err := mfl.New("api", 10000, mfl.WithTransport(transport))
	if err != nil {
		t.Fatal(err)
	}
	return transport, client
}

func (s *Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	export := r.URL.Query().Get("TYPE")
	if export == "league" {
		// Discovery and all subsequent calls terminate in this transport, never a socket.
		return response(`{"league":{"baseURL":"https://www47.myfantasyleague.com"}}`), nil
	}
	s.requests = append(s.requests, export)
	if r.URL.Query().Get("APIKEY") != "" {
		s.keyed = append(s.keyed, export)
	}
	if s.failure == export {
		return nil, errors.New("synthetic offline")
	}
	return response(s.bodies[export]), nil
}

func response(body string) *http.Response {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}
}

func (s *Transport) Fail(export string) { s.mu.Lock(); defer s.mu.Unlock(); s.failure = export }

func (s *Transport) Requests() ([]string, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...), append([]string(nil), s.keyed...)
}
