package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leagueload"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type fakeMFLServer struct {
	t          *testing.T
	srv        *httptest.Server
	failType   string
	echoType   string
	echoBody   []byte
	mu         sync.Mutex
	seenPaths  []string
	seenTypes  []string
	seenAPIKey []string
	league     map[string][]byte
	rules      map[string][]byte
	rosters    []byte
	draft      []byte
	future     []byte
	players    []byte
	schedule   []byte
	results    []byte
	live       []byte
	txns       []byte
	nflWeeks   map[string][]byte
}

func newFakeMFLServer(t *testing.T, season int) *fakeMFLServer {
	t.Helper()
	fixture, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeMFLServer{
		t:        t,
		league:   map[string][]byte{},
		rules:    map[string][]byte{},
		nflWeeks: map[string][]byte{},
	}
	f.league["2026"] = readRepoFile(t, "internal/ingestion/league/testdata/league-2026.json")
	f.league["2025"] = readRepoFile(t, "internal/ingestion/league/testdata/league-2025.json")
	f.rules["2026"] = readRepoFile(t, "internal/ingestion/league/testdata/rules-2026.json")
	f.rules["2025"] = readRepoFile(t, "internal/ingestion/league/testdata/rules-2025.json")
	f.rosters = readRepoFile(t, "internal/ingestion/testdata/comms/rosters-2026.json")
	f.draft = readRepoFile(t, "internal/ingestion/draftreports/testdata/results-2026.json")
	f.future = readRepoFile(t, "internal/ingestion/draftreports/testdata/future-2026.json")
	f.players = buildFakePlayersJSON(t, fixture.Players)
	f.schedule = []byte(`{"schedule":{"weeklySchedule":[` +
		`{"week":"1","matchup":{"franchise":[` +
		`{"id":"0001","isHome":"1","score":"112.50"},` +
		`{"id":"0002","isHome":"0","score":"98.25"}]}},` +
		`{"week":"2","matchup":{"franchise":[` +
		`{"id":"0001","isHome":"1","score":"105.00"},` +
		`{"id":"0002","isHome":"0","score":"101.50"}]}}` +
		`]}}`)
	f.results = []byte(`{"allWeeklyResults":{"weeklyResults":[` +
		`{"week":"1","matchup":{"regularSeason":"1","franchise":[` +
		`{"id":"0001","isHome":"1","score":"112.50","opt_pts":"120.00","result":"W","player":[]},` +
		`{"id":"0002","isHome":"0","score":"98.25","opt_pts":"105.00","result":"L","player":[]}]}},` +
		`{"week":"2","matchup":{"regularSeason":"1","franchise":[` +
		`{"id":"0001","isHome":"1","score":"105.00","opt_pts":"110.00","result":"W","player":[]},` +
		`{"id":"0002","isHome":"0","score":"101.50","opt_pts":"104.00","result":"L","player":[]}]}},` +
		`{"week":"3","matchup":{"regularSeason":"1","franchise":[` +
		`{"id":"0001","isHome":"1","score":"","result":"","player":[]},` +
		`{"id":"0002","isHome":"0","score":"","result":"","player":[]}]}}` +
		`]}}`)
	f.live = []byte(`{"liveScoring":{"week":"3","matchup":{"franchise":[` +
		`{"id":"0001","isHome":"1","score":"0","gameSecondsRemaining":"3600","players":{"player":[]}},` +
		`{"id":"0002","isHome":"0","score":"0","gameSecondsRemaining":"3600","players":{"player":[]}}` +
		`]}}}`)
	ts1 := strconv.FormatInt(time.Date(season, 9, 10, 12, 0, 0, 0, time.UTC).Unix(), 10)
	ts2 := strconv.FormatInt(time.Date(season, 9, 17, 12, 0, 0, 0, time.UTC).Unix(), 10)
	f.txns = []byte(fmt.Sprintf(`{"transactions":{"transaction":[`+
		`{"type":"FREE_AGENT","timestamp":%q,"franchise":"0001","transaction":"|"},`+
		`{"type":"FREE_AGENT","timestamp":%q,"franchise":"0002","transaction":"|"}`+
		`]}}`, ts1, ts2))
	baseKickoff := time.Date(season, 9, 10, 17, 0, 0, 0, time.UTC)
	for wk := 1; wk <= 3; wk++ {
		ko := strconv.FormatInt(baseKickoff.Add(time.Duration(wk-1)*7*24*time.Hour).Unix(), 10)
		f.nflWeeks[strconv.Itoa(wk)] = []byte(fmt.Sprintf(
			`{"nflSchedule":{"week":"%d","matchup":{"kickoff":%q,"gameSecondsRemaining":"0",`+
				`"team":[{"id":"DAL","isHome":"1","score":"24"},`+
				`{"id":"NYG","isHome":"0","score":"17"}]}}}`, wk, ko))
	}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serveHTTP))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeMFLServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET is allowed", http.StatusMethodNotAllowed)
		return
	}
	kind := r.URL.Query().Get("TYPE")
	apiKey := r.URL.Query().Get("APIKEY")
	f.mu.Lock()
	f.seenPaths = append(f.seenPaths, r.URL.Path)
	f.seenTypes = append(f.seenTypes, kind)
	if apiKey != "" {
		f.seenAPIKey = append(f.seenAPIKey, apiKey)
	}
	fail := f.failType
	echo := f.echoType
	echoBody := f.echoBody
	f.mu.Unlock()
	if fail != "" && kind == fail {
		http.Error(w, "simulated MFL outage", http.StatusInternalServerError)
		return
	}
	if echo != "" && kind == echo && len(echoBody) > 0 {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(echoBody)
		return
	}
	year := strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/export"), "/")
	var body []byte
	switch kind {
	case "league":
		body = f.league[year]
	case "rules":
		body = f.rules[year]
	case "rosters":
		body = f.rosters
	case "salaryAdjustments":
		body = []byte(`{"salaryAdjustments":{}}`)
	case "draftResults":
		body = f.draft
	case "futureDraftPicks":
		body = f.future
	case "players":
		body = f.players
	case "schedule":
		body = f.schedule
	case "weeklyResults":
		body = f.results
	case "liveScoring":
		body = f.live
	case "transactions":
		body = f.txns
	case "nflSchedule":
		body = f.nflWeeks[r.URL.Query().Get("W")]
	default:
		http.Error(w, "unexpected export "+kind, http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		http.Error(w, "missing fixture for "+year+"/"+kind, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (f *fakeMFLServer) transport(t *testing.T) http.RoundTripper {
	t.Helper()
	target, err := url.Parse(f.srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	inner := f.srv.Client().Transport
	return mflRedirectTransport{target: target, inner: inner}
}

type mflRedirectTransport struct {
	target *url.URL
	inner  http.RoundTripper
}

func (rt mflRedirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = rt.target.Scheme
	cloned.URL.Host = rt.target.Host
	return rt.inner.RoundTrip(cloned)
}

func readRepoFile(t *testing.T, rel string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return body
}

func buildFakePlayersJSON(t *testing.T, playersMap map[string]string) []byte {
	t.Helper()
	type row struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Position  string `json:"position"`
		Team      string `json:"team"`
		DraftYear string `json:"draft_year"`
	}
	ids := make([]string, 0, len(playersMap))
	for id := range playersMap {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	rows := make([]row, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, row{
			ID: id, Name: playersMap[id], Position: "WR", Team: "DAL", DraftYear: "2022",
		})
	}
	payload := map[string]any{"players": map[string]any{"player": rows}}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestMirrorFetchHappyPathAndReplayRefresh(t *testing.T) {
	dir := t.TempDir()
	mirrorPath := filepath.Join(dir, "history.db")
	fake := newFakeMFLServer(t, 2026)
	fixture, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	fixedNow := time.Date(2026, 11, 20, 18, 0, 0, 0, time.UTC)
	var out strings.Builder
	env := mirrorFetchEnv{
		transport: fake.transport(t),
		now:       func() time.Time { return fixedNow },
		rps:       10000,
		keySource: func(context.Context, string, int) (mflkey.Key, error) {
			return "", mflkey.ErrNoKey
		},
	}
	args := []string{"-db", mirrorPath, "-mfl-league", "14432"}
	if err := runMirrorFetchWithEnv(t.Context(), args, env, &out); err != nil {
		t.Fatalf("mirror-fetch failed: %v", err)
	}
	text := out.String()
	wantParts := []string{
		"14432",
		"2026",
		fmt.Sprintf("Franchises: %d", len(fixture.Config.Franchises)),
		fmt.Sprintf("Players: %d", len(fixture.Players)),
		fmt.Sprintf("Contracts: %d", len(fixture.Rosters)),
		"Transactions: 2",
		"Weeks of scores: 2",
		"2026-11-20 18:00:00 UTC",
		"twr-server league-refresh",
		mirrorPath,
		leaguemigration.AppLeagueID("14432", 2026),
		"-dry-run",
	}
	for _, want := range wantParts {
		if !strings.Contains(text, want) {
			t.Errorf("output missing %q:\n%s", want, text)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "history.db" {
		t.Fatalf("unexpected files left in dir: %+v", entries)
	}

	// Prove the fetched mirror works end-to-end with league-load and league-refresh -dry-run.
	serverDir := t.TempDir()
	serverDB := filepath.Join(serverDir, "server.db")
	pinPath := filepath.Join(serverDir, "football.sha256")
	if err := prepareFirstLoad(serverDB); err != nil {
		t.Fatal(err)
	}
	creds := accounts.RawCredentials{Username: "christopher", Password: "a-long-private-password"}
	var loadOut strings.Builder
	if err := loadLeague(t.Context(), serverDB, mirrorPath, "14432", pinPath, creds, &loadOut); err != nil {
		t.Fatalf("league-load from fetched mirror: %v", err)
	}
	var refreshOut strings.Builder
	refreshOpts := refreshOptions{
		dbPath:     serverDB,
		mirrorPath: mirrorPath,
		league:     leaguemigration.AppLeagueID("14432", 2026),
		dryRun:     true,
	}
	if err := refreshFromMirror(t.Context(), refreshOpts, creds, &refreshOut); err != nil {
		t.Fatalf("league-refresh -dry-run from fetched mirror: %v", err)
	}
	if !strings.Contains(refreshOut.String(), "already up to date") {
		t.Fatalf("expected up-to-date dry run, got:\n%s", refreshOut.String())
	}
}

func TestMirrorFetchExplicitYear(t *testing.T) {
	dir := t.TempDir()
	mirrorPath := filepath.Join(dir, "history-2025.db")
	fake := newFakeMFLServer(t, 2025)
	fixedNow := time.Date(2026, 11, 20, 18, 0, 0, 0, time.UTC)
	var out strings.Builder
	env := mirrorFetchEnv{
		transport: fake.transport(t),
		now:       func() time.Time { return fixedNow },
		rps:       10000,
		keySource: func(context.Context, string, int) (mflkey.Key, error) {
			return "", mflkey.ErrNoKey
		},
	}
	args := []string{"-db", mirrorPath, "-mfl-league", "14432", "-year", "2025"}
	if err := runMirrorFetchWithEnv(t.Context(), args, env, &out); err != nil {
		t.Fatalf("mirror-fetch -year 2025: %v", err)
	}
	if !strings.Contains(out.String(), "native.mfl-14432-2025") {
		t.Fatalf("expected 2025 league ID in output:\n%s", out.String())
	}
	fake.mu.Lock()
	paths := slices.Clone(fake.seenPaths)
	fake.mu.Unlock()
	if len(paths) == 0 {
		t.Fatal("expected HTTP requests to fake MFL server")
	}
	for _, p := range paths {
		if !strings.HasPrefix(p, "/2025/") {
			t.Fatalf("expected only /2025/ paths, saw %q", p)
		}
	}

	// Prove the explicit-year mirror replays offline in league-load and league-refresh.
	serverDir := t.TempDir()
	serverDB := filepath.Join(serverDir, "server.db")
	pinPath := filepath.Join(serverDir, "football.sha256")
	if err := prepareFirstLoad(serverDB); err != nil {
		t.Fatal(err)
	}
	creds := accounts.RawCredentials{Username: "christopher", Password: "a-long-private-password"}
	var loadOut strings.Builder
	if err := loadLeague(
		t.Context(), serverDB, mirrorPath, "14432", pinPath, creds, &loadOut,
	); err != nil {
		t.Fatalf("league-load from 2025 mirror: %v", err)
	}
	var refreshOut strings.Builder
	refreshOpts := refreshOptions{
		dbPath:     serverDB,
		mirrorPath: mirrorPath,
		league:     leaguemigration.AppLeagueID("14432", 2025),
		dryRun:     true,
	}
	if err := refreshFromMirror(t.Context(), refreshOpts, creds, &refreshOut); err != nil {
		t.Fatalf("league-refresh -dry-run from 2025 mirror: %v", err)
	}
	if !strings.Contains(refreshOut.String(), "already up to date") {
		t.Fatalf("expected up-to-date 2025 dry run, got:\n%s", refreshOut.String())
	}
}

func TestMirrorFetchFailureKeepsExistingMirror(t *testing.T) {
	dir := t.TempDir()
	mirrorPath := filepath.Join(dir, "history.db")
	sentinel := []byte("prior-known-good-mirror-bytes")
	if err := os.WriteFile(mirrorPath, sentinel, 0o600); err != nil {
		t.Fatal(err)
	}
	fake := newFakeMFLServer(t, 2026)
	fake.failType = "rosters"
	var out strings.Builder
	env := mirrorFetchEnv{
		transport: fake.transport(t),
		now:       func() time.Time { return time.Date(2026, 11, 20, 18, 0, 0, 0, time.UTC) },
		rps:       10000,
		keySource: func(context.Context, string, int) (mflkey.Key, error) {
			return "", mflkey.ErrNoKey
		},
	}
	err := runMirrorFetchWithEnv(
		t.Context(), []string{"-db", mirrorPath, "-mfl-league", "14432"}, env, &out,
	)
	if err == nil {
		t.Fatal("expected mirror-fetch to fail on rosters 500")
	}
	got, readErr := os.ReadFile(mirrorPath)
	if readErr != nil {
		t.Fatalf("read existing mirror after failed fetch: %v", readErr)
	}
	if !bytes.Equal(got, sentinel) {
		t.Fatalf("existing mirror was modified on failure")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "history.db" {
		t.Fatalf("leftover temp files after failure: %+v", entries)
	}
}

func TestMirrorFetchPrepareValidationAndCounts(t *testing.T) {
	dir := t.TempDir()
	mirrorPath := filepath.Join(dir, "history.db")
	sentinel := []byte("prior-known-good-mirror-bytes")
	if err := os.WriteFile(mirrorPath, sentinel, 0o600); err != nil {
		t.Fatal(err)
	}
	fake := newFakeMFLServer(t, 2026)
	fake.rosters = []byte(`{"rosters":{"franchise":[{"id":"0001","player":[` +
		`{"id":"13116","status":"ROSTER","salary":"not-a-number",` +
		`"contractYear":"2027","contractStatus":"UFA"}]}]}}`)
	env := mirrorFetchEnv{
		transport: fake.transport(t),
		now:       func() time.Time { return time.Date(2026, 11, 20, 18, 0, 0, 0, time.UTC) },
		rps:       10000,
		keySource: func(context.Context, string, int) (mflkey.Key, error) {
			return "", mflkey.ErrNoKey
		},
	}
	var out strings.Builder
	err := runMirrorFetchWithEnv(
		t.Context(), []string{"-db", mirrorPath, "-mfl-league", "14432"}, env, &out,
	)
	if err == nil {
		t.Fatal("expected mirror-fetch to fail Prepare validation on invalid salary")
	}
	got, readErr := os.ReadFile(mirrorPath)
	if readErr != nil || !bytes.Equal(got, sentinel) {
		t.Fatalf("existing mirror was altered on Prepare failure: %v", readErr)
	}

	// Partial contracts and partially-scored weeks must follow Prepare and RecordedResults rules.
	fixture, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	fake2 := newFakeMFLServer(t, 2026)
	fake2.rosters = []byte(fmt.Sprintf(`{"rosters":{"franchise":[`+
		`{"id":"0001","player":[`+
		`{"id":%q,"status":"ROSTER","salary":"5.00",`+
		`"contractYear":"2027","contractStatus":"UFA"},`+
		`{"id":%q,"status":"ROSTER","salary":"2.50","contractYear":"","contractStatus":""}`+
		`]}]}}`, fixture.Rosters[0].Player, fixture.Rosters[1].Player))
	fake2.results = []byte(`{"allWeeklyResults":{"weeklyResults":[` +
		`{"week":"1","matchup":{"regularSeason":"1","franchise":[` +
		`{"id":"0001","isHome":"1","score":"112.50","opt_pts":"120.00","result":"W","player":[]},` +
		`{"id":"0002","isHome":"0","score":"98.25","opt_pts":"105.00","result":"L","player":[]}]}},` +
		`{"week":"2","matchup":[` +
		`{"regularSeason":"1","franchise":[` +
		`{"id":"0001","isHome":"1","score":"105.00","opt_pts":"110.00","result":"W","player":[]},` +
		`{"id":"0002","isHome":"0","score":"101.50","opt_pts":"104.00","result":"L","player":[]}]},` +
		`{"regularSeason":"1","franchise":[` +
		`{"id":"0003","isHome":"1","score":"","result":"","player":[]},` +
		`{"id":"0004","isHome":"0","score":"","result":"","player":[]}]}]}` +
		`]}}`)
	env2 := mirrorFetchEnv{
		transport: fake2.transport(t),
		now:       func() time.Time { return time.Date(2026, 11, 20, 18, 0, 0, 0, time.UTC) },
		rps:       10000,
		keySource: func(context.Context, string, int) (mflkey.Key, error) {
			return "", mflkey.ErrNoKey
		},
	}
	var out2 strings.Builder
	mirrorPath2 := filepath.Join(dir, "history-partial.db")
	err = runMirrorFetchWithEnv(
		t.Context(), []string{"-db", mirrorPath2, "-mfl-league", "14432"}, env2, &out2,
	)
	if err != nil {
		t.Fatalf("mirror-fetch with partial contract/week failed: %v", err)
	}
	if !strings.Contains(out2.String(), "Contracts: 1") {
		t.Fatalf("expected Contracts: 1 (only complete contracts), got:\n%s", out2.String())
	}
	if !strings.Contains(out2.String(), "Weeks of scores: 1") {
		t.Fatalf("expected Weeks of scores: 1 (only completed weeks), got:\n%s", out2.String())
	}
}

func TestMirrorFetchCredentialsNeverLeak(t *testing.T) {
	dir := t.TempDir()
	mirrorPath := filepath.Join(dir, "history.db")
	fake := newFakeMFLServer(t, 2026)
	var rawBytes [16]byte
	if _, err := rand.Read(rawBytes[:]); err != nil {
		t.Fatal(err)
	}
	ephemeralKey := hex.EncodeToString(rawBytes[:])
	env := mirrorFetchEnv{
		transport: fake.transport(t),
		now:       func() time.Time { return time.Date(2026, 11, 20, 18, 0, 0, 0, time.UTC) },
		rps:       10000,
		keySource: func(context.Context, string, int) (mflkey.Key, error) {
			return mflkey.Key(ephemeralKey), nil
		},
	}
	var out strings.Builder
	err := runMirrorFetchWithEnv(
		t.Context(), []string{"-db", mirrorPath, "-mfl-league", "14432"}, env, &out,
	)
	if err != nil {
		t.Fatalf("mirror-fetch with keySource: %v", err)
	}
	if strings.Contains(out.String(), ephemeralKey) {
		t.Fatal("credential appeared in stdout output")
	}
	rawDB, err := os.ReadFile(mirrorPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(rawDB, []byte(ephemeralKey)) {
		t.Fatal("credential appeared in written mirror database")
	}
	pools, err := db.OpenReadOnly(t.Context(), mirrorPath, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	mirror := leagueload.Mirror{Store: history.New(pools, registry)}
	if _, err := mirror.Players(t.Context()); err != nil {
		t.Fatalf("mirror missing player list: %v", err)
	}

	// Exercise real keyed round-trips through archive.Transport so APIKEY redaction
	// and echoed-secret omission are proven rather than vacuously skipped.
	keyedDB := filepath.Join(dir, "keyed.db")
	keyedPools, err := db.Open(t.Context(), keyedDB, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = keyedPools.Close() }()
	keyedStore := history.New(keyedPools, registry)
	if err := keyedStore.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	keyCalls := 0
	fake.echoType = "rosters"
	fake.echoBody = []byte(fmt.Sprintf(`{"error":"echoed %s"}`, ephemeralKey))
	client, err := mfl.New(
		"api",
		10000,
		mfl.WithTransport(&archive.Transport{
			Base:  fake.transport(t),
			Sink:  keyedStore,
			Judge: ingestion.MFLRefusal,
		}),
		mfl.WithKeySource(func(context.Context) (mflkey.Key, error) {
			keyCalls++
			return mflkey.Key(ephemeralKey), nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.DiscoverHost(t.Context(), "2026", "14432"); err != nil {
		t.Fatalf("discover host for keyed round-trip: %v", err)
	}
	if _, err := client.Do(t.Context(), mfl.Request{
		Year: "2026", Type: "salaryAdjustments", Params: map[string]string{"L": "14432"}, Keyed: true,
	}); err != nil {
		t.Fatalf("keyed salaryAdjustments fetch: %v", err)
	}
	if _, err := client.Do(t.Context(), mfl.Request{
		Year: "2026", Type: "rosters", Params: map[string]string{"L": "14432"}, Keyed: true,
	}); err != nil {
		t.Fatalf("keyed echoed-secret fetch: %v", err)
	}
	if keyCalls < 2 {
		t.Fatalf("expected keySource to be called for keyed requests, got %d", keyCalls)
	}
	fake.mu.Lock()
	seenKeys := slices.Clone(fake.seenAPIKey)
	fake.mu.Unlock()
	if !slices.Contains(seenKeys, ephemeralKey) {
		t.Fatalf("fake server never received APIKEY on keyed round-trip")
	}
	var redactedURL string
	found, err := keyedStore.ArchivedBodies(
		t.Context(), "TYPE=salaryAdjustments", func(fetched string, _ []byte, _ time.Time) bool {
			redactedURL = fetched
			return true
		},
	)
	if err != nil || !found {
		t.Fatalf("archived keyed salaryAdjustments fetch missing: found=%v err=%v", found, err)
	}
	if !strings.Contains(redactedURL, "APIKEY=REDACTED") || strings.Contains(redactedURL, ephemeralKey) {
		t.Fatalf("expected redacted APIKEY in fetch_log URL, got %q", redactedURL)
	}
	echoedFound, err := keyedStore.ArchivedBodies(
		t.Context(), "TYPE=rosters", func(_ string, _ []byte, _ time.Time) bool { return true },
	)
	if err != nil || echoedFound {
		t.Fatalf("expected echoed-secret body to be omitted, found=%v err=%v", echoedFound, err)
	}
	if err := checkpointLeague(t.Context(), keyedPools); err != nil {
		t.Fatal(err)
	}
	keyedBytes, err := os.ReadFile(keyedDB)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(keyedBytes, []byte(ephemeralKey)) {
		t.Fatal("credential appeared in keyed archive database bytes")
	}
}

func TestMirrorFetchFlagValidation(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		args []string
	}{
		{"missing all", nil},
		{"missing league", []string{"-db", filepath.Join(dir, "h.db")}},
		{"missing db", []string{"-mfl-league", "14432"}},
		{"extra arg", []string{"-db", filepath.Join(dir, "h.db"), "-mfl-league", "14432", "extra"}},
		{"bad year low", []string{"-db", filepath.Join(dir, "h.db"), "-mfl-league", "14432", "-year", "1800"}},
		{"bad year high", []string{"-db", filepath.Join(dir, "h.db"), "-mfl-league", "14432", "-year", "10000"}},
		{"db is dir", []string{"-db", dir, "-mfl-league", "14432"}},
		{"missing parent", []string{"-db", filepath.Join(dir, "missing", "h.db"), "-mfl-league", "14432"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := run(t.Context(), append([]string{"mirror-fetch"}, tc.args...), &out); err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
		})
	}
}
