package composition

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

type weekTransport struct {
	bodies   []string
	requests []string
	err      error
}

func (w *weekTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Query().Get("TYPE") != "nflSchedule" {
		return nil, errors.New("season feeds not provided by schedule fixture")
	}
	w.requests = append(w.requests, req.URL.Query().Get("W"))
	if w.err != nil {
		return nil, w.err
	}
	body := w.bodies[0]
	if len(w.bodies) > 1 {
		w.bodies = w.bodies[1:]
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}, nil
}

type weekRules struct{ end string }

func (w weekRules) Fetch(context.Context) (league.RawConfig, error) {
	return league.RawConfig{
		Source: "test", StartWeek: "1", LastRegularSeasonWeek: "13", EndWeek: w.end,
	}, nil
}

type phaseRecorder struct {
	state.Writer
	phases []domain.Phase
	notes  []string
}

func (p *phaseRecorder) WriteTx(ctx context.Context, fn func(state.TxWriter) error) error {
	return p.Writer.WriteTx(ctx, func(w state.TxWriter) error {
		return fn(recordedPhase{TxWriter: w, recorder: p})
	})
}

type recordedPhase struct {
	state.TxWriter
	recorder *phaseRecorder
}

func (r recordedPhase) AppendPhaseTransition(ctx context.Context, to domain.Phase, note string) error {
	if err := r.TxWriter.AppendPhaseTransition(ctx, to, note); err != nil {
		return err
	}
	r.recorder.phases = append(r.recorder.phases, to)
	r.recorder.notes = append(r.recorder.notes, note)
	return nil
}

func weekFixture(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile("internal/ingestion/nflschedule/testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func weekTestApp(t *testing.T, end string, bodies ...string) (*App, *weekTransport, *phaseRecorder) {
	t.Helper()
	a := targetTestApp(t)
	if err := a.core.Rulebook.Initialize(a.core.Ctx, weekRules{end: end}); err != nil {
		t.Fatal(err)
	}
	p, err := db.Open(a.core.Ctx, filepath.Join(t.TempDir(), "whatif.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Error(err)
		}
	})
	a.core.Whatif = state.New(p, ingestion.LeagueID, 2026, a.core.Rulebook)
	if err := a.core.Whatif.Initialize(a.core.Ctx, a.core.League); err != nil {
		t.Fatal(err)
	}
	recorder := &phaseRecorder{Writer: a.core.Whatif.Writer()}
	a.core.Coordinator, err = transactions.New(recorder, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	transport := &weekTransport{bodies: bodies}
	a.core.MFLClient, err = mfl.New("", 1000, mfl.WithTransport(transport))
	if err != nil {
		t.Fatal(err)
	}
	a.core.ClockChanged = func(context.Context) {}
	a.core.SeasonChanged = func(context.Context) {}
	return a, transport, recorder
}

func TestRefreshWeekAndClock(t *testing.T) {
	t.Parallel()
	a, transport, recorder := weekTestApp(t, "17", weekFixture(t, "nflSchedule-w5.json"))
	// Inside week 5, before its first kickoff: the wall clock passed the fixture's last kickoff on
	// 2026-10-12, after which no lineup lock is left and the reading has no week.
	a.core.ClockNow = func() time.Time { return time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC) }
	before, err := a.platformAPI().TargetClock()
	if err != nil || before.Value.Week != nil || before.Provenance.Freshness.State != FreshLive ||
		!strings.Contains(before.Provenance.Freshness.Note, "NFL schedule not fetched yet") {
		t.Fatalf("before fetch: %+v, %v", before, err)
	}
	if len(transport.requests) != 0 {
		t.Fatal("clock fetched")
	}
	emitted := 0
	a.core.ClockChanged = func(context.Context) { emitted++ }
	a.refreshWeek(a.core.Ctx)
	if a.core.Week.Number != 5 || len(recorder.phases) != 1 || recorder.phases[0] != domain.PhaseRegularSeason {
		t.Fatalf("week %+v, phases %v", a.core.Week, recorder.phases)
	}
	wantNote := "derived from NFL week 5 (bounds 1/13/17), schedule fetched " +
		a.core.WeekFetchedAt.Format(time.RFC3339)
	if recorder.notes[0] != wantNote {
		t.Fatalf("note %q, want %q", recorder.notes[0], wantNote)
	}
	a.refreshWeek(a.core.Ctx)
	if len(recorder.phases) != 1 || emitted != 2 {
		t.Fatalf("phases %v, events %d", recorder.phases, emitted)
	}
	calls := len(transport.requests)
	clock, err := a.platformAPI().TargetClock()
	if err != nil || clock.Value.Week == nil || *clock.Value.Week != 5 {
		t.Fatalf("after fetch: %+v, %v", clock, err)
	}
	if len(clock.Value.Deadlines) != 1 || clock.Value.Deadlines[0].ID != "lineup-w5" {
		t.Fatal(clock.Value.Deadlines)
	}
	if len(transport.requests) != calls ||
		clock.Provenance.Freshness.FetchedAt != a.core.WeekFetchedAt.Format(time.RFC3339) {
		t.Fatal("clock fetched or replaced fetch time")
	}
	transport.err = errors.New("schedule offline")
	fetched := a.core.WeekFetchedAt
	a.refreshWeek(a.core.Ctx)
	if a.core.Week.Number != 5 || !a.core.WeekFetchedAt.Equal(fetched) || a.core.WeekRefreshError == "" {
		t.Fatal("failure lost last good week")
	}
	clock, err = a.platformAPI().TargetClock()
	if err != nil || clock.Provenance.Freshness.State != FreshStale ||
		!strings.Contains(clock.Provenance.Freshness.Note, "; showing week 5 fetched ") {
		t.Fatalf("stale: %+v, %v", clock, err)
	}
	if len(recorder.phases) != 1 || emitted != 3 {
		t.Fatal("failure changed phase or did not emit")
	}
}

func TestRefreshWeekAdvancesPastFinalDefault(t *testing.T) {
	t.Parallel()
	a, transport, _ := weekTestApp(t, "17",
		weekFixture(t, "nflSchedule.json"), weekFixture(t, "nflSchedule-w5.json"))
	a.refreshWeek(a.core.Ctx)
	if a.core.Week.Number != 5 || len(transport.requests) != 2 ||
		transport.requests[0] != "" || transport.requests[1] != "5" {
		t.Fatalf("week %+v, requests %v", a.core.Week, transport.requests)
	}
}

func TestRefreshWeekDoesNotAppendBeyondBounds(t *testing.T) {
	t.Parallel()
	a, _, recorder := weekTestApp(t, "17", weekFixture(t, "nflSchedule-w5.json"))
	cfg := a.core.Rulebook.ActiveConfig()
	cfg.LastRegularSeasonWeek, cfg.EndWeek = "3", "4"
	if _, err := a.core.Rulebook.Sync(a.core.Ctx, cfg); err != nil {
		t.Fatal(err)
	}
	a.refreshWeek(a.core.Ctx)
	phase, err := a.core.Whatif.CurrentPhase(a.core.Ctx)
	if err != nil || phase != domain.PhaseOffseason || len(recorder.phases) != 0 {
		t.Fatalf("phase %s, appends %v, error %v", phase, recorder.phases, err)
	}
}

func TestRefreshWeekNoLineupAndBadBounds(t *testing.T) {
	t.Parallel()
	body := strings.Replace(weekFixture(t, "nflSchedule.json"), `"week":"4"`, `"week":"18"`, 1)
	a, transport, recorder := weekTestApp(t, "17", body)
	a.refreshWeek(a.core.Ctx)
	if a.core.Week.Number != 0 || a.core.WeekRefreshError != "" || len(transport.requests) != 1 {
		t.Fatalf("final week 18: %+v, %q, %v", a.core.Week, a.core.WeekRefreshError, transport.requests)
	}
	cfg := a.core.Rulebook.ActiveConfig()
	cfg.StartWeek = "unknown"
	if _, err := a.core.Rulebook.Sync(a.core.Ctx, cfg); err != nil {
		t.Fatal(err)
	}
	transport.bodies = []string{weekFixture(t, "nflSchedule-w5.json")}
	a.refreshWeek(a.core.Ctx)
	if a.core.Week.Number != 5 || len(recorder.phases) != 0 {
		t.Fatal("bad bounds appended or discarded schedule")
	}
}

func TestRefreshWeekSerializesAndCancels(t *testing.T) {
	t.Parallel()
	a, transport, recorder := weekTestApp(t, "17", weekFixture(t, "nflSchedule-w5.json"))
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() { a.refreshWeek(a.core.Ctx) })
	}
	wg.Wait()
	if len(recorder.phases) != 1 || len(transport.requests) != 4 {
		t.Fatal("concurrent refresh did not serialize")
	}
	ctx, cancel := context.WithCancel(a.core.Ctx)
	cancel()
	a.refreshWeeksInBackground(ctx)
	if len(transport.requests) != 4 {
		t.Fatal("cancelled worker fetched")
	}
}

func TestWeekWorkerStopsOnShutdown(t *testing.T) {
	t.Parallel()
	a, transport, recorder := weekTestApp(t, "17", weekFixture(t, "nflSchedule-w5.json"))
	refreshed := make(chan struct{}, 1)
	a.core.ClockChanged = func(context.Context) { refreshed <- struct{}{} }
	a.refreshInBackground(a.core.Ctx)
	t.Cleanup(func() { a.shutdown(a.core.Ctx) })
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-refreshed:
	case <-timer.C:
		t.Fatal("worker did not fetch when no launch league refresh was due")
	}
	a.shutdown(a.core.Ctx)
	if len(transport.requests) != 1 || len(recorder.phases) != 1 {
		t.Fatalf("requests %v, phases %v", transport.requests, recorder.phases)
	}
}
