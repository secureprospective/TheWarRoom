package mflread

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// fakeMFL serves recorded-shape bodies by TYPE and counts every request it sees.
type fakeMFL struct {
	calls atomic.Int32
}

func (f *fakeMFL) RoundTrip(req *http.Request) (*http.Response, error) {
	f.calls.Add(1)
	body := `{"error":"unknown"}`
	switch req.URL.Query().Get("TYPE") {
	case "league":
		body = `{"league":{"baseURL":"https://www47.myfantasyleague.com/2026"}}`
	case "freeAgents":
		body = `{"freeAgents":{"leagueUnit":{"unit":"LEAGUE","player":[{"id":"0531"},{"id":"13593"}]}}}`
	case "players":
		body = `{"players":{"player":[` +
			`{"id":"0531","name":"Doe, John","position":"WR","team":"BUF","draft_year":"2020","birthdate":"820000000"},` +
			`{"id":"13593","name":"Roe, Rich","position":"RB","team":"FA","draft_year":"2025","birthdate":"1000000000"}]}}`
	}
	rec := httptest.NewRecorder()
	_, _ = io.WriteString(rec, body)
	rec.Header().Set("Content-Type", "application/json")
	return rec.Result(), nil
}

func TestMirrorGuardRefusesWrites(t *testing.T) {
	fake := &fakeMFL{}
	rt := ReadOnly(fake)
	cases := []struct {
		name, method, url string
	}{
		{"post", http.MethodPost, "https://www47.myfantasyleague.com/2026/export?TYPE=freeAgents&L=1"},
		{"put", http.MethodPut, "https://api.myfantasyleague.com/2026/export?TYPE=league&L=1"},
		{"import path", http.MethodGet, "https://www47.myfantasyleague.com/2026/import?TYPE=fcfsWaiver&L=1"},
		{"import type", http.MethodGet, "https://www47.myfantasyleague.com/2026/export?TYPE=importLineup&L=1"},
		{"submit type", http.MethodGet, "https://www47.myfantasyleague.com/2026/export?TYPE=submitBid&L=1"},
		{"key", http.MethodGet, "https://www47.myfantasyleague.com/2026/export?TYPE=rosters&L=1&APIKEY=x"},
	}
	for _, c := range cases {
		req, _ := http.NewRequest(c.method, c.url, strings.NewReader(""))
		if c.method == http.MethodGet {
			req, _ = http.NewRequest(c.method, c.url, nil)
		}
		if resp, err := rt.RoundTrip(req); err == nil {
			_ = resp.Body.Close()
			t.Errorf("%s: request was sent", c.name)
		}
	}
	if fake.calls.Load() != 0 {
		t.Fatalf("guard let %d requests through", fake.calls.Load())
	}
	req, _ := http.NewRequest(http.MethodGet, "https://api.myfantasyleague.com/2026/export?TYPE=league&L=1", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil || fake.calls.Load() != 1 {
		t.Fatalf("plain public GET refused: %v", err)
	}
	_ = resp.Body.Close()
}

func TestMirrorFetcherReadsPublicExportsOverGetOnly(t *testing.T) {
	fake := &fakeMFL{}
	f := Fetcher{Transport: fake, RPS: 1000}
	ids, err := f.FetchFreeAgents(t.Context(), "14432", 2026)
	if err != nil || len(ids) != 2 || ids[0] != "0531" {
		t.Fatalf("free agents = %v, %v", ids, err)
	}
	raw, err := f.FetchPlayers(t.Context(), "14432", 2026)
	if err != nil || !strings.Contains(string(raw), "Roe, Rich") {
		t.Fatalf("players = %v, %v", raw, err)
	}
}

func TestMirrorDirectoryReadsHeldPlayers(t *testing.T) {
	f := Fetcher{Transport: &fakeMFL{}, RPS: 1000}
	raw, err := f.FetchPlayers(t.Context(), "14432", 2026)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := f.Directory(raw)
	if err != nil {
		t.Fatal(err)
	}
	facts, ok := dir.Facts("13593")
	if !ok || facts.Name != "Roe, Rich" || string(facts.Position) != "RB" {
		t.Fatalf("facts = %+v ok=%v", facts, ok)
	}
}
