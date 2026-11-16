package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
)

const hostedPlayersBody = `{"players":{"player":[{"id":"0816","name":"Listed, Player","position":"WR","team":"FA"},{"id":"0151","position":"Def"}]}}`

type playersTransport struct {
	calls         int
	status        int
	body          string
	failDiscovery bool
}

func (p *playersTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	p.calls++
	body, status := `{"league":{"baseURL":"https://www42.myfantasyleague.com"}}`, http.StatusOK
	if p.failDiscovery {
		return nil, fmt.Errorf("discovery unavailable")
	}
	if r.URL.Query().Get("TYPE") == "players" {
		if r.URL.Hostname() != "www42.myfantasyleague.com" || r.URL.Query().Get("DETAILS") != "1" {
			return nil, fmt.Errorf("players request bypassed discovery or details")
		}
		body, status = p.body, p.status
	}
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

func playersKeeper(t *testing.T, path string, transport *playersTransport) (*signalsKeeper, stateFiles) {
	t.Helper()
	files, err := openSignInFiles(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = files.Close() })
	v := newServerValues(newLeaguePins(files.leagues), t.TempDir())
	return &signalsKeeper{values: v, client: &http.Client{Transport: transport}, now: time.Now, logf: t.Logf}, files
}

func playersReference(t *testing.T, files stateFiles, name, mflLeague string) {
	t.Helper()
	store := leagueOn(t, files, name)
	records, err := store.SharedCommissionerRecords(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) == 0 {
		raw := commissioner.RawCommand{League: name, Command: commissioner.Create, Request: "create", Reason: "Test listing scope", Effective: "now", Visibility: "league",
			Args: commissioner.RawArgs{Name: "Listing scope", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2}}
		if _, err := store.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
			t.Fatal(err)
		}
		records, err = store.SharedCommissionerRecords(t.Context(), name)
		if err != nil {
			t.Fatal(err)
		}
	}
	last := records[len(records)-1]
	raw := commissioner.RawCommand{League: name, Command: commissioner.RulesSet, Request: "reference", Reason: "Test listing scope", Effective: "now", Visibility: "league", Expected: last.Receipt.Revision,
		Args: commissioner.RawArgs{Field: "schedule.follow", Value: json.RawMessage(fmt.Sprintf(`{"Source":"mfl","League":%q,"Season":2026}`, mflLeague))}}
	if _, err := store.ExecuteCommissioner(t.Context(), raw, records[0].Principal, nil); err != nil {
		t.Fatal(err)
	}
}

func TestServerPlayersRecordsAreListed(t *testing.T) {
	transport := &playersTransport{status: http.StatusOK, body: hostedPlayersBody}
	k, files := playersKeeper(t, newTwoLeagueServer(t).path, transport)
	playersReference(t, files, fixtureLeague, "00123")
	playersReference(t, files, secondLeague, "00456")
	archivePlayers(t, k, "00456", strings.ReplaceAll(hostedPlayersBody, "0816", "0820"), time.Now())
	k.writeRecords(t.Context())
	records, err := k.values.records().Records(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("wanted both leagues' players, not aggregates: %+v", records)
	}
	for _, record := range records {
		if !record.Listed || record.Position != domain.PosWR || record.Name != "Listed, Player" || (record.MFL != "0816" && record.MFL != "0820") {
			t.Fatalf("listing did not reach stored record: %+v", record)
		}
	}
	select {
	case <-k.values.recorded:
	default:
		t.Fatal("records did not wake value preparation")
	}
}

func TestServerPlayersFreshArchiveSurvivesRestart(t *testing.T) {
	transport := &playersTransport{status: http.StatusOK, body: hostedPlayersBody}
	path := newOneFileServer(t).path
	k, files := playersKeeper(t, path, transport)
	before := time.Now()
	_, found, err := k.players(t.Context(), "00123", 2026)
	if err != nil || !found || transport.calls != 2 {
		t.Fatalf("first read: held=%v calls=%d err=%v", found, transport.calls, err)
	}
	if err := files.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, _ := playersKeeper(t, path, transport)
	restarted.now = func() time.Time { return before.Add(23 * time.Hour) }
	lookup, held, err := restarted.players(t.Context(), "00123", 2026)
	if err != nil || !held {
		t.Fatalf("restart lost archived listing: held=%v err=%v", held, err)
	}
	if _, ok := lookup.Facts("0816"); !ok || transport.calls != 2 {
		t.Fatalf("fresh archive made a second MFL call or lost its player: calls=%d", transport.calls)
	}
	restarted.now = func() time.Time { return before.Add(25 * time.Hour) }
	if _, _, err := restarted.players(t.Context(), "00123", 2026); err != nil || transport.calls != 4 {
		t.Fatalf("expired archive did not refresh: calls=%d err=%v", transport.calls, err)
	}
}

func archivePlayers(t *testing.T, k *signalsKeeper, league, body string, at time.Time) {
	t.Helper()
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	sha := sha256.Sum256([]byte(body))
	err := k.values.reference.Record(t.Context(), archive.Fetch{URL: "https://www42.myfantasyleague.com/2026/export?DETAILS=1&JSON=1&L=" + league + "&TYPE=players",
		Status: http.StatusOK, SHA256: hex.EncodeToString(sha[:]), Size: int64(len(body)), Gzip: compressed.Bytes(), FetchedAt: at})
	if err != nil {
		t.Fatal(err)
	}
}

func TestServerPlayersFailureKeepsListingAndBacksOff(t *testing.T) {
	for _, failure := range []struct {
		name   string
		status int
		body   string
	}{
		{"server", http.StatusServiceUnavailable, "unavailable"},
		{"rate limit", http.StatusTooManyRequests, "slow down"},
		{"refusal", http.StatusOK, `{"error":"unavailable"}`},
		{"invalid", http.StatusOK, `{"players":{"player":[{"id":"0816"}]}}`},
	} {
		t.Run(failure.name, func(t *testing.T) {
			transport := &playersTransport{status: failure.status, body: failure.body}
			path := newOneFileServer(t).path
			k, files := playersKeeper(t, path, transport)
			playersReference(t, files, fixtureLeague, "00123")
			archivePlayers(t, k, "00123", hostedPlayersBody, time.Now().Add(-25*time.Hour))
			logs := 0
			k.logf = func(format string, _ ...any) {
				if strings.Contains(format, "keeping the last listing") {
					logs++
				}
			}
			k.writeRecords(t.Context())
			if transport.calls != 2 || logs != 1 {
				t.Fatalf("failure: calls=%d logs=%d", transport.calls, logs)
			}
			records, err := k.values.records().Records(t.Context())
			if err != nil || len(records) != 1 || !records[0].Listed {
				t.Fatalf("failure lost listed records: %+v %v", records, err)
			}
			k.writeRecords(t.Context())
			if transport.calls != 2 || logs != 1 {
				t.Fatalf("failure repeated before backoff: calls=%d logs=%d", transport.calls, logs)
			}
			if err := files.Close(); err != nil {
				t.Fatal(err)
			}
			restarted, _ := playersKeeper(t, path, transport)
			restarted.now = func() time.Time { return time.Now().Add(59 * time.Minute) }
			lookup, held, err := restarted.players(t.Context(), "00123", 2026)
			if _, ok := lookup.Facts("0816"); !ok || !held || err != nil || transport.calls != 2 {
				t.Fatalf("restart discarded listing or retried: held=%v calls=%d err=%v", held, transport.calls, err)
			}
			transport.status, transport.body = http.StatusOK, hostedPlayersBody
			restarted.now = func() time.Time { return time.Now().Add(time.Hour + time.Second) }
			if _, _, err := restarted.players(t.Context(), "00123", 2026); err != nil || transport.calls != 4 {
				t.Fatalf("no recovery after backoff: calls=%d err=%v", transport.calls, err)
			}
		})
	}
}

func TestServerPlayersArchiveIsScoped(t *testing.T) {
	transport := &playersTransport{status: http.StatusOK, body: hostedPlayersBody}
	k, _ := playersKeeper(t, newOneFileServer(t).path, transport)
	archivePlayers(t, k, "00999", hostedPlayersBody, time.Now())
	archivePlayers(t, k, "00123", `{"players":{"player":[]}}`, time.Now())
	if _, _, err := k.players(t.Context(), "00123", 2026); err != nil {
		t.Fatal(err)
	}
	if transport.calls != 0 {
		t.Fatal("invalid recent response bypassed retry backoff")
	}
	k.now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	if _, held, err := k.players(t.Context(), "00123", 2026); err != nil || !held || transport.calls != 2 {
		t.Fatalf("another league's or invalid body used as listing: held=%v calls=%d err=%v", held, transport.calls, err)
	}
}

func TestServerPlayersDiscoveryFailureBacksOffAfterRestart(t *testing.T) {
	transport := &playersTransport{failDiscovery: true}
	path := newOneFileServer(t).path
	k, files := playersKeeper(t, path, transport)
	if _, held, err := k.players(t.Context(), "00123", 2026); err == nil || held || transport.calls != 1 {
		t.Fatalf("discovery failure: held=%v calls=%d err=%v", held, transport.calls, err)
	}
	if err := files.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, _ := playersKeeper(t, path, transport)
	restarted.now = func() time.Time { return time.Now().Add(59 * time.Minute) }
	if _, held, err := restarted.players(t.Context(), "00123", 2026); err != nil || held || transport.calls != 1 {
		t.Fatalf("restart repeated failed discovery: held=%v calls=%d err=%v", held, transport.calls, err)
	}
}
