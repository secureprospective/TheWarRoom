package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/zalando/go-keyring"
)

type seasonTransport struct {
	requests []string
	weeks    []string
	starts   []time.Time
	now      time.Time
	bodies   map[string]string
	failure  string
}

func (s *seasonTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	export := r.URL.Query().Get("TYPE")
	if export == "league" {
		return seasonResponse(`{"league":{"baseURL":"https://www47.myfantasyleague.com"}}`), nil
	}
	s.requests = append(s.requests, export)
	s.weeks = append(s.weeks, r.URL.Query().Get("W"))
	s.starts = append(s.starts, s.now)
	if s.failure == export {
		return nil, errors.New("synthetic offline")
	}
	return seasonResponse(s.bodies[export]), nil
}

func seasonResponse(body string) *http.Response {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}
}

func seasonFixture(t *testing.T, feed, name string) string {
	t.Helper()
	body, err := os.ReadFile("internal/ingestion/" + feed + "/testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func seasonTestApp(t *testing.T) (*App, *seasonTransport, *[]time.Duration, *int) {
	t.Helper()
	keyring.MockInit()
	a := targetTestApp(t)
	a.keyStore = mflkey.New(ingestion.LeagueID, "2026")
	transport := &seasonTransport{
		now: time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC),
		bodies: map[string]string{
			"transactions":  seasonFixture(t, "transactions", "transactions.json"),
			"liveScoring":   seasonFixture(t, "livescoring", "liveScoring-w5.json"),
			"pendingTrades": `{"pendingTrades":{}}`,
		},
	}
	var err error
	a.mflClient, err = mfl.New("api", 10000, mfl.WithTransport(transport), mfl.WithKeySource(a.keyStore.Get))
	if err != nil {
		t.Fatal(err)
	}
	pauses := []time.Duration{}
	a.seasonPause = func(ctx context.Context, d time.Duration) error {
		pauses = append(pauses, d)
		transport.now = transport.now.Add(d)
		return ctx.Err()
	}
	events := 0
	a.seasonChanged = func(context.Context) { events++ }
	t.Cleanup(func() { a.weekWorkers.Wait() })
	return a, transport, &pauses, &events
}

func seasonReading(t *testing.T, a *App) SeasonReading {
	t.Helper()
	reading, err := a.TargetSeason()
	if err != nil {
		t.Fatal(err)
	}
	return reading
}

func TestSeasonHeldOnlyAndRefreshOrder(t *testing.T) {
	a, transport, pauses, events := seasonTestApp(t)
	for range 2 {
		r := seasonReading(t, a)
		for _, p := range []Freshness{
			r.Transactions.Provenance.Freshness, r.Lineups.Provenance.Freshness, r.PendingTrades.Provenance.Freshness,
		} {
			if p.State != FreshFail || !strings.Contains(p.Note, "not fetched yet") {
				t.Fatal(p)
			}
		}
		body, err := json.Marshal(r)
		if err != nil || strings.Contains(string(body), ":null") {
			t.Fatalf("null slice: %s %v", body, err)
		}
	}
	if len(transport.requests) != 0 {
		t.Fatal("binding fetched")
	}
	if err := a.keyStore.Set(a.ctx, "synthetic-stored-key"); err != nil {
		t.Fatal(err)
	}
	a.week.Number = 5
	a.refreshSeason(a.ctx)
	r := seasonReading(t, a)
	for _, p := range []Freshness{
		r.Transactions.Provenance.Freshness, r.Lineups.Provenance.Freshness, r.PendingTrades.Provenance.Freshness,
	} {
		if p.State != FreshLive || p.FetchedAt == "" {
			t.Fatal(p)
		}
	}
	if len(r.Transactions.Value) != 770 || len(r.Lineups.Value.Franchises) != 32 || r.Lineups.Value.Week != 5 {
		t.Fatal("wrong real feed counts or week")
	}
	if !reflect.DeepEqual(transport.requests, []string{"transactions", "liveScoring", "pendingTrades"}) ||
		!reflect.DeepEqual(*pauses, []time.Duration{time.Second, time.Second}) || *events != 1 {
		t.Fatalf("requests %v pauses %v events %d", transport.requests, *pauses, *events)
	}
	for i := 1; i < len(transport.starts); i++ {
		if transport.starts[i].Sub(transport.starts[i-1]) < time.Second {
			t.Fatal("requests not spaced")
		}
	}
	if transport.weeks[1] != "5" || r.Lineups.Provenance.Source != "liveScoring W=5" {
		t.Fatal("requested wrong week")
	}
	a.week.Number = 0
	transport.bodies["liveScoring"] = seasonFixture(t, "livescoring", "liveScoring.json")
	a.refreshSeason(a.ctx)
	r = seasonReading(t, a)
	if transport.weeks[4] != "" || r.Lineups.Value.Week != 4 ||
		r.Lineups.Provenance.Source != "liveScoring default week" {
		t.Fatal("default week not respected")
	}
}

func recordSeasonArchive(t *testing.T, a *App, export, week, body string, at time.Time) {
	t.Helper()
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(body))
	src := "/2026/export?TYPE=" + export + "&L=" + ingestion.LeagueID
	if week != "" {
		src += "&W=" + week
	}
	err := a.history.Record(a.ctx, archive.Fetch{
		URL: src, Status: 200, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(body)),
		Gzip: compressed.Bytes(), FetchedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSeasonArchiveFallback(t *testing.T) {
	for _, archived := range []bool{false, true} {
		t.Run(strconvBool(archived), func(t *testing.T) {
			a, transport, _, _ := seasonTestApp(t)
			at := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
			if archived {
				recordSeasonArchive(t, a, "transactions", "", transport.bodies["transactions"], at)
			}
			transport.failure = "transactions"
			a.refreshSeasonTransactions(a.ctx)
			r := seasonReading(t, a)
			want := FreshFail
			if archived {
				want = FreshStale
				if r.Transactions.Provenance.Freshness.FetchedAt != at.Format(time.RFC3339) ||
					len(r.Transactions.Value) != 770 {
					t.Fatal("archive time or counts lost")
				}
			}
			if r.Transactions.Provenance.Freshness.State != want ||
				!strings.Contains(r.Transactions.Provenance.Freshness.Note, "synthetic offline") {
				t.Fatal(r.Transactions.Provenance)
			}
		})
	}
}

func strconvBool(b bool) string {
	if b {
		return "archived"
	}
	return "missing"
}

func TestSeasonLineupArchiveWeekIsolation(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	at := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	recordSeasonArchive(t, a, "liveScoring", "4", seasonFixture(t, "livescoring", "liveScoring.json"), at)
	a.week.Number = 5
	transport.failure = "liveScoring"
	a.refreshSeasonLineups(a.ctx)
	r := seasonReading(t, a)
	if r.Lineups.Provenance.Freshness.State != FreshFail || r.Lineups.Value.Week != 0 {
		t.Fatal("another week's archived lineup used")
	}
	recordSeasonArchive(t, a, "liveScoring", "5", transport.bodies["liveScoring"], at.Add(-time.Hour))
	a.refreshSeasonLineups(a.ctx)
	r = seasonReading(t, a)
	if r.Lineups.Provenance.Freshness.State != FreshStale || r.Lineups.Value.Week != 5 ||
		r.Lineups.Provenance.Freshness.FetchedAt != at.Add(-time.Hour).Format(time.RFC3339) {
		t.Fatal(r.Lineups)
	}
	a.week.Number = 0
	a.refreshSeasonLineups(a.ctx)
	r = seasonReading(t, a)
	if r.Lineups.Provenance.Freshness.State != FreshFail || r.Lineups.Value.Week != 0 {
		t.Fatal("explicit week used for default")
	}
	recordSeasonArchive(t, a, "liveScoring", "", seasonFixture(t, "livescoring", "liveScoring.json"), at)
	a.refreshSeasonLineups(a.ctx)
	r = seasonReading(t, a)
	if r.Lineups.Provenance.Freshness.State != FreshStale || r.Lineups.Value.Week != 4 {
		t.Fatal("default archive ignored")
	}
}

func TestSeasonPendingTradesKeyStates(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	a.seasonPendingTrades.value = []leaguefeed.PendingTrade{{ID: "forgotten"}}
	a.refreshPendingTrades(a.ctx)
	r := seasonReading(t, a)
	if len(transport.requests) != 0 || len(r.PendingTrades.Value) != 0 ||
		r.PendingTrades.Provenance.Freshness.Note != "MFL not connected" {
		t.Fatal("absent key sent request or retained trades")
	}
	if err := a.keyStore.Set(a.ctx, "synthetic-stored-key"); err != nil {
		t.Fatal(err)
	}
	// Synthetic: an authenticated trade body is not present in the real captures.
	transport.bodies["pendingTrades"] = `{"pendingTrades":{"pendingTrade":{` +
		`"trade_id":"101","offeringteam":"0001","offeredto":"0002",` +
		`"will_give_up":"12345,","will_receive":"BB_10.50,","timestamp":"100","expires":"200"}}}`
	a.refreshPendingTrades(a.ctx)
	r = seasonReading(t, a)
	at := r.PendingTrades.Provenance.Freshness.FetchedAt
	if len(r.PendingTrades.Value) != 1 || r.PendingTrades.Provenance.Freshness.State != FreshLive {
		t.Fatal(r.PendingTrades)
	}
	transport.failure = "pendingTrades"
	a.refreshPendingTrades(a.ctx)
	r = seasonReading(t, a)
	if len(r.PendingTrades.Value) != 1 || r.PendingTrades.Provenance.Freshness.State != FreshStale ||
		r.PendingTrades.Provenance.Freshness.FetchedAt != at {
		t.Fatal("failure lost copy")
	}
	calls := len(transport.requests)
	keyring.MockInitWithError(errors.New("locked"))
	a.keyStore = mflkey.New(ingestion.LeagueID, "2026")
	a.refreshPendingTrades(a.ctx)
	r = seasonReading(t, a)
	if len(transport.requests) != calls || len(r.PendingTrades.Value) != 1 ||
		r.PendingTrades.Provenance.Freshness.State != FreshStale ||
		r.PendingTrades.Provenance.Freshness.Note != "MFL keyring is locked or unavailable" {
		t.Fatal("unavailable keyring discarded copy or fetched")
	}
}

func TestSeasonBindingDoesNotWaitForRefresh(t *testing.T) {
	a, _, _, _ := seasonTestApp(t)
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	done := make(chan error, 1)
	go func() {
		_, err := a.TargetSeason()
		done <- err
	}()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-timer.C:
		t.Fatal("binding blocked on refresh")
	}
}

func TestSeasonKeyChangesRefreshAndClear(t *testing.T) {
	a, transport, _, events := seasonTestApp(t)
	status, err := a.SetMFLKey("synthetic-stored-key")
	if err != nil || status.State != "connected" {
		t.Fatal(status, err)
	}
	a.weekWorkers.Wait()
	if !reflect.DeepEqual(transport.requests, []string{"pendingTrades", "pendingTrades"}) || *events != 1 {
		t.Fatalf("requests %v events %d", transport.requests, *events)
	}
	a.seasonPendingTrades.value = []leaguefeed.PendingTrade{{ID: "forgotten"}}
	// Holding refreshMu proves deletion returns and clears without waiting for the worker.
	a.refreshMu.Lock()
	status, err = a.DeleteMFLKey()
	r := seasonReading(t, a)
	a.refreshMu.Unlock()
	a.weekWorkers.Wait()
	if err != nil || status.State != "absent" || len(r.PendingTrades.Value) != 0 ||
		r.PendingTrades.Provenance.Freshness.Note != "MFL not connected" || *events != 2 ||
		len(transport.requests) != 2 {
		t.Fatal("delete did not clear before returning or sent keyed request")
	}
}

func TestSeasonPauseCancellationAndNotReady(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := seasonPause(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	a := NewApp()
	a.startupErr = errors.New("not ready")
	close(a.started)
	if _, err := a.TargetSeason(); !errors.Is(err, a.startupErr) {
		t.Fatal(err)
	}
}

func TestSeasonWorkerUsesRefreshedWeek(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	scheduleApp, _, _ := weekTestApp(t, "17", weekFixture(t, "nflSchedule-w5.json"))
	a.whatif, a.rulebook, a.coordinator = scheduleApp.whatif, scheduleApp.rulebook, scheduleApp.coordinator
	a.clockChanged = func(context.Context) {}
	transport.bodies["nflSchedule"] = weekFixture(t, "nflSchedule-w5.json")
	if err := a.keyStore.Set(a.ctx, "synthetic-stored-key"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()
	a.seasonChanged = func(context.Context) { cancel() }
	a.refreshInBackground(ctx)
	a.weekWorkers.Wait()
	if !reflect.DeepEqual(transport.requests, []string{
		"nflSchedule", "transactions", "liveScoring", "pendingTrades",
	}) || transport.weeks[2] != "5" {
		t.Fatalf("order %v weeks %v", transport.requests, transport.weeks)
	}
}

type blockedSeasonTrade struct {
	base    *seasonTransport
	entered chan struct{}
	release chan struct{}
}

func (b blockedSeasonTrade) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Query().Get("TYPE") == "pendingTrades" {
		close(b.entered)
		<-b.release
	}
	return b.base.RoundTrip(r)
}

// heldSeasonTrade blocks a pendingTrades request until its context ends.
type heldSeasonTrade struct {
	base    *seasonTransport
	entered chan struct{}
}

func (h heldSeasonTrade) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Query().Get("TYPE") == "pendingTrades" {
		close(h.entered)
		<-r.Context().Done()
		return nil, r.Context().Err()
	}
	return h.base.RoundTrip(r)
}

func TestSeasonShutdownCancelsKeyChangeRefresh(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	if err := a.keyStore.Set(a.ctx, "synthetic-stored-key"); err != nil {
		t.Fatal(err)
	}
	a.workCtx, a.workCancel = context.WithCancel(a.ctx)
	entered := make(chan struct{})
	var err error
	a.mflClient, err = mfl.New("api", 10000,
		mfl.WithTransport(heldSeasonTrade{base: transport, entered: entered}),
		mfl.WithKeySource(a.keyStore.Get))
	if err != nil {
		t.Fatal(err)
	}
	a.refreshPendingTradesInBackground()
	<-entered
	done := make(chan struct{})
	go func() {
		a.workCancel()
		a.weekWorkers.Wait()
		close(done)
	}()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		t.Fatal("shutdown waited for the key-change refresh")
	}
}

func TestSeasonDeleteDoesNotWaitForPendingFetch(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	if err := a.keyStore.Set(a.ctx, "synthetic-stored-key"); err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	var err error
	a.mflClient, err = mfl.New("api", 10000,
		mfl.WithTransport(blockedSeasonTrade{base: transport, entered: entered, release: release}),
		mfl.WithKeySource(a.keyStore.Get))
	if err != nil {
		t.Fatal(err)
	}
	a.refreshPendingTradesInBackground()
	<-entered
	deleted := make(chan MFLKeyStatus, 1)
	go func() {
		status, err := a.DeleteMFLKey()
		if err != nil {
			status.State = err.Error()
		}
		deleted <- status
	}()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case status := <-deleted:
		if status.State != "absent" {
			t.Error(status)
		}
	case <-timer.C:
		t.Error("delete waited for pendingTrades network request")
	}
	close(release)
	a.weekWorkers.Wait()
	r := seasonReading(t, a)
	if r.PendingTrades.Provenance.Freshness.Note != "MFL not connected" ||
		r.PendingTrades.Provenance.Freshness.State != FreshFail || len(r.PendingTrades.Value) != 0 {
		t.Fatal("in-flight result republished after deletion")
	}
}

func TestSeasonPendingKeySourceErrors(t *testing.T) {
	for _, cause := range []error{mflkey.ErrNoKey, mflkey.ErrUnavailable, mflkey.ErrTimeout} {
		t.Run(cause.Error(), func(t *testing.T) {
			a, transport, _, _ := seasonTestApp(t)
			if err := a.keyStore.Set(a.ctx, "synthetic-stored-key"); err != nil {
				t.Fatal(err)
			}
			a.seasonPendingTrades = heldSeasonFeed[[]leaguefeed.PendingTrade]{
				value:     []leaguefeed.PendingTrade{{ID: "kept"}},
				fetchedAt: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
			}
			a.mflClient = a.mflClient.WithKeySource(func(context.Context) (mflkey.Key, error) {
				return "", cause
			})
			a.refreshPendingTrades(a.ctx)
			r := seasonReading(t, a)
			if len(transport.requests) != 0 {
				t.Fatal("key source failure sent keyed request")
			}
			if errors.Is(cause, mflkey.ErrNoKey) {
				if r.PendingTrades.Provenance.Freshness.State != FreshFail || len(r.PendingTrades.Value) != 0 ||
					r.PendingTrades.Provenance.Freshness.Note != "MFL not connected" {
					t.Fatal(r.PendingTrades)
				}
			} else if r.PendingTrades.Provenance.Freshness.State != FreshStale || len(r.PendingTrades.Value) != 1 ||
				r.PendingTrades.Provenance.Freshness.Note != "MFL keyring is locked or unavailable" {
				t.Fatal(r.PendingTrades)
			}
		})
	}
}

func TestSeasonPendingTradesNeverUsesArchive(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	if err := a.keyStore.Set(a.ctx, "synthetic-stored-key"); err != nil {
		t.Fatal(err)
	}
	recordSeasonArchive(t, a, "pendingTrades", "", transport.bodies["pendingTrades"],
		time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC))
	transport.failure = "pendingTrades"
	a.refreshPendingTrades(a.ctx)
	r := seasonReading(t, a)
	if r.PendingTrades.Provenance.Freshness.State != FreshFail ||
		!strings.Contains(r.PendingTrades.Provenance.Freshness.Note, "synthetic offline") {
		t.Fatal("pending trades used archived offers")
	}
}
