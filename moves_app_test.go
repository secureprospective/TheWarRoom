package main

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func constructedMove(t *testing.T, at time.Time, spec envelope.Spec, blocked bool) envelope.Envelope {
	t.Helper()
	from, event := envelope.Ready, envelope.ChecksPass
	if blocked {
		from, event = envelope.Blocked, envelope.ChecksBlock
	}
	e, err := envelope.Restore("move-1", at, spec, []envelope.AuditEntry{
		{At: at, From: envelope.Draft, Event: event, To: from, Note: "constructed check"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func moveSpec(t *testing.T) envelope.Spec {
	t.Helper()
	id, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	return envelope.Spec{
		Intent: "roster.ir", LeagueID: ingestion.LeagueID, FranchiseID: "0001",
		Subject:  envelope.Subject{Players: []playerid.PlayerID{id}},
		Expected: envelope.ExpectedChange{Player: id, RosterStatus: domain.RosterIR},
		Gravity:  envelope.G2, Undo: envelope.Reversible,
		Target: envelope.Target{
			Kind: envelope.Mapped,
			URL:  "https://www47.myfantasyleague.com/2026/options?L=14432&O=18",
		},
	}
}

func saveMove(t *testing.T, a *App, e envelope.Envelope) {
	t.Helper()
	if err := a.moves.Save(a.ctx, e); err != nil {
		t.Fatal(err)
	}
}

func storedMove(t *testing.T, a *App) envelope.Envelope {
	t.Helper()
	e, err := a.moves.Get(a.ctx, "move-1")
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestTargetHandOffSaveBeforeOpen(t *testing.T) {
	a := targetTestApp(t)
	at := time.Now().UTC().Add(-time.Minute)
	e := constructedMove(t, at, moveSpec(t), false)
	saveMove(t, a, e)
	events, opens := 0, 0
	var logs []string
	a.movesChanged = func(context.Context) { events++ }
	a.movesLog = func(line string) { logs = append(logs, line) }
	a.openURL = func(_ context.Context, target string) {
		opens++
		if storedMove(t, a).State() != envelope.HandedOff || target != e.Receipt().Spec.Target.URL {
			t.Fatal("open preceded save or changed URL")
		}
	}
	r, err := a.TargetHandOff(e.ID())
	if err != nil || r.State != envelope.HandedOff || opens != 1 || events != 1 || len(logs) != 1 {
		t.Fatalf("hand-off: %+v, %v, opens %d events %d logs %v", r, err, opens, events, logs)
	}
	select {
	case <-a.movesWake:
	default:
		t.Fatal("watcher not woken")
	}
}

func TestTargetHandOffRefused(t *testing.T) {
	for _, name := range []string{"blocked", "unmapped", "expired", "unknown", "foreign", "repeated"} {
		t.Run(name, func(t *testing.T) {
			a := targetTestApp(t)
			at := time.Now().UTC().Add(-time.Minute)
			spec := moveSpec(t)
			if name == "unmapped" {
				spec.Target = envelope.Target{Kind: envelope.Unmapped}
			}
			if name == "expired" {
				spec.Deadline = &at
			}
			if name == "foreign" {
				spec.Target.URL = "https://example.com/options"
			}
			e := constructedMove(t, at, spec, name == "blocked" || name == "unmapped")
			if name == "repeated" {
				var err error
				e, err = e.HandOff(at)
				if err != nil {
					t.Fatal(err)
				}
			}
			saveMove(t, a, e)
			before := storedMove(t, a).Receipt()
			a.openURL = func(context.Context, string) { t.Fatal("opened refused move") }
			a.movesChanged = func(context.Context) { t.Fatal("event for refused move") }
			id := e.ID()
			if name == "unknown" {
				id = "absent"
			}
			_, err := a.TargetHandOff(id)
			if err == nil || !reflect.DeepEqual(before, storedMove(t, a).Receipt()) {
				t.Fatalf("refusal: %v", err)
			}
			if name != "unknown" && !strings.Contains(err.Error(), string(e.State())) {
				t.Fatalf("missing current state: %v", err)
			}
		})
	}
}

func TestMoveURLSecondLock(t *testing.T) {
	for _, target := range []string{
		"https://example.com/", "http://myfantasyleague.com/",
		"https://owner@myfantasyleague.com/", "https://myfantasyleague.com.evil.test/",
		"https://evilmyfantasyleague.com/", "https:///options", ":bad",
	} {
		if err := validateMoveURL(target); err == nil {
			t.Fatal("accepted", target)
		}
	}
	for _, target := range []string{
		"https://myfantasyleague.com/", "https://www47.myfantasyleague.com/options?O=18",
	} {
		if err := validateMoveURL(target); err != nil {
			t.Fatal(err)
		}
	}
	// Invalid HTTP/user-info targets cannot reach the store in the first place.
	for _, target := range []string{"http://myfantasyleague.com/", "https://owner@myfantasyleague.com/"} {
		spec := moveSpec(t)
		spec.Target.URL = target
		if _, err := envelope.New(time.Now(), spec, func() string { return "bad" }); err == nil {
			t.Fatal("invalid target constructed")
		}
	}
}

func TestMoveBindingsNotReady(t *testing.T) {
	a := NewApp()
	a.startupErr = errors.New("not ready")
	close(a.started)
	if _, err := a.TargetHandOff("missing"); err == nil {
		t.Fatal("hand-off ignored startup failure")
	}
	if err := a.TargetCheckMoves(); err == nil {
		t.Fatal("check ignored startup failure")
	}
}

func awaitingMove(t *testing.T, a *App, at time.Time) envelope.Envelope {
	t.Helper()
	e := constructedMove(t, at, moveSpec(t), false)
	e, err := e.HandOff(at)
	if err != nil {
		t.Fatal(err)
	}
	saveMove(t, a, e)
	return e
}

func irMirror(t *testing.T, a *App) {
	t.Helper()
	_, err := a.league.Replace(a.ctx, state.MirrorSnapshot{
		Season:  2026,
		Players: []state.MirrorPlayer{{MFLID: "0042", FranchiseID: "0001", RosterStatus: domain.RosterIR}},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWatcherIRFreshnessAndDeclaredSources(t *testing.T) {
	for _, stale := range []bool{false, true} {
		t.Run(map[bool]string{false: "fresh", true: "old"}[stale], func(t *testing.T) {
			a := targetTestApp(t)
			at := time.Now().Add(-time.Minute)
			if stale {
				at = time.Now().Add(time.Minute)
			}
			awaitingMove(t, a, at)
			a.movesNow = func() time.Time { return at.Add(2 * time.Minute) }
			var sources []envelope.Source
			a.movesRefresh = func(_ context.Context, source envelope.Source) error {
				sources = append(sources, source)
				irMirror(t, a)
				return nil
			}
			events := 0
			var logs []string
			a.movesChanged = func(context.Context) { events++ }
			a.movesLog = func(line string) { logs = append(logs, line) }
			if err := a.checkMoves(a.ctx); err != nil {
				t.Fatal(err)
			}
			want, count := envelope.Landed, 1
			if stale {
				want, count = envelope.HandedOff, 0
			}
			if storedMove(t, a).State() != want || events != count || len(logs) != count ||
				!reflect.DeepEqual(sources, []envelope.Source{envelope.Rosters}) {
				t.Fatalf("state %s events %d logs %v sources %v", storedMove(t, a).State(), events, logs, sources)
			}
			if !stale && !strings.Contains(logs[0], "move move-1 handed_off → landed:") {
				t.Fatal(logs)
			}
		})
	}
}

func TestWatcherEmptyMakesNoRequests(t *testing.T) {
	a := targetTestApp(t)
	a.movesRefresh = func(context.Context, envelope.Source) error {
		t.Fatal("idle refresh")
		return nil
	}
	a.movesChanged = func(context.Context) { t.Fatal("idle event") }
	a.movesLog = func(string) { t.Fatal("idle log") }
	if err := a.checkMoves(a.ctx); err != nil {
		t.Fatal(err)
	}
}

func TestWatcherRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "restart.db")
	first := targetTestAppAt(t, path)
	awaitingMove(t, first, time.Now().Add(-time.Minute))
	second := targetTestAppAt(t, path)
	landed := make(chan struct{}, 1)
	second.movesChanged = func(context.Context) { landed <- struct{}{} }
	second.movesLog = func(string) {}
	second.movesRefresh = func(context.Context, envelope.Source) error {
		irMirror(t, second)
		return nil
	}
	ctx, cancel := context.WithCancel(second.ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		second.watchMoves(ctx)
	}()
	defer func() {
		cancel()
		<-done
	}()
	select {
	case <-landed:
	case <-time.After(5 * time.Second):
		t.Fatal("restarted watcher did not observe saved hand-off")
	}
	if storedMove(t, first).State() != envelope.Landed {
		t.Fatal("second app did not persist observation")
	}
}

type manualMoveTimer struct {
	delay time.Duration
	tick  chan time.Time
}

type manualMoveClock struct {
	mu     sync.Mutex
	now    time.Time
	timers chan manualMoveTimer
}

func (c *manualMoveClock) read() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *manualMoveClock) set(at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = at
}

func (c *manualMoveClock) timer(delay time.Duration) (<-chan time.Time, func()) {
	tick := make(chan time.Time, 1)
	c.timers <- manualMoveTimer{delay: delay, tick: tick}
	return tick, func() {}
}

func receiveMoveTimer(t *testing.T, c *manualMoveClock) manualMoveTimer {
	t.Helper()
	select {
	case timer := <-c.timers:
		return timer
	case <-time.After(5 * time.Second):
		t.Fatal("watcher did not arm timer")
		return manualMoveTimer{}
	}
}

func TestWatcherFastModeAndCoalescedWakes(t *testing.T) {
	a := targetTestApp(t)
	at := time.Now().Add(time.Minute)
	awaitingMove(t, a, at)
	clock := &manualMoveClock{now: at, timers: make(chan manualMoveTimer, 20)}
	a.movesNow, a.movesTimer = clock.read, clock.timer
	passes := make(chan time.Time, 40)
	a.movesRefresh = func(context.Context, envelope.Source) error {
		passes <- clock.read()
		return nil
	}
	a.movesChanged = func(context.Context) {}
	a.movesLog = func(string) {}
	ctx, cancel := context.WithCancel(a.ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		a.watchMoves(ctx)
	}()
	defer func() {
		cancel()
		<-done
	}()
	timer := receiveMoveTimer(t, clock)
	if timer.delay != 0 {
		t.Fatal(timer.delay)
	}
	timer.tick <- at
	<-passes
	timer = receiveMoveTimer(t, clock)
	for range 10 {
		if err := a.TargetCheckMoves(); err != nil {
			t.Fatal(err)
		}
		// Waiting for each rearm proves each wake was consumed without running a pass.
		timer = receiveMoveTimer(t, clock)
		if timer.delay != time.Minute || len(passes) != 0 {
			t.Fatal("wake bypassed rate limit")
		}
	}
	for minute := 1; minute < 30; minute++ {
		now := at.Add(time.Duration(minute) * time.Minute)
		clock.set(now)
		timer.tick <- now
		if got := <-passes; !got.Equal(now) {
			t.Fatal(got, now)
		}
		timer = receiveMoveTimer(t, clock)
		if timer.delay != time.Minute {
			t.Fatal("fast pass not 60s apart", timer.delay)
		}
	}
	clock.set(at.Add(30 * time.Minute))
	timer.tick <- clock.read()
	// Once the fast window ends there is no timer. A season wake rearms it.
	if err := a.TargetCheckMoves(); err != nil {
		t.Fatal(err)
	}
	timer = receiveMoveTimer(t, clock)
	if timer.delay != 0 || len(passes) != 0 {
		t.Fatal("expired fast mode made a pass")
	}
	timer.tick <- clock.read()
	if got := <-passes; !got.Equal(clock.read()) {
		t.Fatal(got)
	}
}

func TestWatcherCancellationStopsBlockedRefresh(t *testing.T) {
	a := targetTestApp(t)
	awaitingMove(t, a, time.Now().Add(-time.Minute))
	entered := make(chan struct{})
	a.movesRefresh = func(ctx context.Context, _ envelope.Source) error {
		close(entered)
		<-ctx.Done()
		return ctx.Err()
	}
	a.movesLog = func(string) {}
	a.workCtx, a.workCancel = context.WithCancel(a.ctx)
	a.weekWorkers.Add(1)
	go func() {
		defer a.weekWorkers.Done()
		a.watchMoves(a.workCtx)
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("refresh never started")
	}
	a.workCancel()
	done := make(chan struct{})
	go func() {
		a.weekWorkers.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("watcher ignored cancellation")
	}
	if storedMove(t, a).State() != envelope.HandedOff {
		t.Fatal("cancelled refresh changed move")
	}
}

func TestWatcherUnknownIntentLoggedOnce(t *testing.T) {
	a := targetTestApp(t)
	at := time.Now().Add(-time.Minute)
	spec := moveSpec(t)
	spec.Intent = "future.intent"
	e := constructedMove(t, at, spec, false)
	e, err := e.HandOff(at)
	if err != nil {
		t.Fatal(err)
	}
	saveMove(t, a, e)
	logs := 0
	a.movesLog = func(string) { logs++ }
	a.movesRefresh = func(context.Context, envelope.Source) error {
		t.Fatal("unknown refresh")
		return nil
	}
	for range 2 {
		if err := a.checkMoves(a.ctx); err != nil {
			t.Fatal(err)
		}
	}
	if logs != 1 || storedMove(t, a).State() != envelope.HandedOff {
		t.Fatal("unknown intent mutated or logged repeatedly")
	}
}

type moveFeedTransport struct{ requests []string }

func (m *moveFeedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	feed := r.URL.Query().Get("TYPE")
	m.requests = append(m.requests, feed)
	bodies := map[string]string{
		"league": `{"league":{"baseURL":"https://www47.myfantasyleague.com","id":"14432",
			"salaryCapAmount":"120","usesSalaries":"1",
			"history":{"league":{"year":"2026","url":"https://www47.myfantasyleague.com/2026/14432"}}}}`,
		"rules": `{"rules":{"positionRules":{"positions":"QB","rule":
			{"event":{"$t":"#P"},"points":{"$t":"*0"},"range":{"$t":"0-0"}}}}}`,
		"rosters": `{"rosters":{"franchise":{"id":"0001","player":
			{"id":"0042","status":"INJURED_RESERVE","salary":"1","contractYear":"2026"}}}}`,
		"salaryAdjustments": `{"salaryAdjustments":{}}`,
	}
	body, ok := bodies[feed]
	if !ok {
		return nil, errors.New("unexpected feed " + feed)
	}
	return seasonResponse(body), nil
}

func TestWatcherIRRefreshThroughRealAdapters(t *testing.T) {
	a := targetTestApp(t)
	transport := &moveFeedTransport{}
	var err error
	a.mflClient, err = mfl.New("api", 10000, mfl.WithTransport(transport))
	if err != nil {
		t.Fatal(err)
	}
	if err := a.rulebook.Initialize(a.ctx, discoverSource{app: a}); err != nil {
		t.Fatal(err)
	}
	transport.requests = nil
	awaitingMove(t, a, time.Now().Add(-time.Minute))
	events := 0
	a.movesChanged = func(context.Context) { events++ }
	a.movesLog = func(line string) {
		if !strings.Contains(line, "→ landed") {
			t.Error(line)
		}
	}
	if err := a.checkMoves(a.ctx); err != nil {
		t.Fatal(err)
	}
	if storedMove(t, a).State() != envelope.Landed || events != 1 ||
		a.rostersCheckedAt.IsZero() || a.rulesCheckedAt.Load() == 0 ||
		!reflect.DeepEqual(transport.requests, []string{"league", "rules", "rosters", "salaryAdjustments"}) {
		t.Fatalf("state %s events %d requests %v", storedMove(t, a).State(), events, transport.requests)
	}
}

func TestSeasonRefreshWakesMoves(t *testing.T) {
	a, _, _, _ := seasonTestApp(t)
	a.refreshSeason(a.ctx)
	select {
	case <-a.movesWake:
	default:
		t.Fatal("season refresh did not wake landing watcher")
	}
}

func TestWatcherUnchangedStateDoesNotAppendIdleAudit(t *testing.T) {
	a := targetTestApp(t)
	awaitingMove(t, a, time.Now().Add(-time.Minute))
	_, err := a.league.Replace(a.ctx, state.MirrorSnapshot{
		Season:  2026,
		Players: []state.MirrorPlayer{{MFLID: "0042", FranchiseID: "0001", RosterStatus: domain.RosterActive}},
	})
	if err != nil {
		t.Fatal(err)
	}
	a.movesRefresh = func(context.Context, envelope.Source) error { return nil }
	events, logs := 0, 0
	a.movesChanged = func(context.Context) { events++ }
	a.movesLog = func(string) { logs++ }
	if err := a.checkMoves(a.ctx); err != nil {
		t.Fatal(err)
	}
	before := storedMove(t, a).Receipt()
	if err := a.checkMoves(a.ctx); err != nil {
		t.Fatal(err)
	}
	if before.State != envelope.NotYetDone || events != 1 || logs != 1 ||
		!reflect.DeepEqual(before, storedMove(t, a).Receipt()) {
		t.Fatal("idle observation appended audit, logged, or emitted")
	}
}

func TestWatcherUnchangedRosterCountsAsObserved(t *testing.T) {
	a := targetTestApp(t)
	// The roster content dates from before the hand-off; a later successful refresh confirms it.
	_, err := a.league.Replace(a.ctx, state.MirrorSnapshot{
		Season:  2026,
		Players: []state.MirrorPlayer{{MFLID: "0042", FranchiseID: "0001", RosterStatus: domain.RosterActive}},
	})
	if err != nil {
		t.Fatal(err)
	}
	handed := time.Now().Add(time.Minute)
	awaitingMove(t, a, handed)
	a.movesNow = func() time.Time { return handed.Add(2 * time.Minute) }
	a.movesRefresh = func(context.Context, envelope.Source) error {
		a.rostersCheckedAt = handed.Add(time.Minute)
		return nil
	}
	a.movesChanged = func(context.Context) {}
	a.movesLog = func(string) {}
	if err := a.checkMoves(a.ctx); err != nil {
		t.Fatal(err)
	}
	if got := storedMove(t, a).State(); got != envelope.NotYetDone {
		t.Fatalf("confirmed unchanged roster: state %s, want not_yet_done", got)
	}
}

func TestHandOffDoesNotWaitForWatcherRefresh(t *testing.T) {
	a := targetTestApp(t)
	awaitingMove(t, a, time.Now().Add(-time.Minute))
	at := time.Now().UTC().Add(-time.Minute)
	second, err := envelope.Restore("move-2", at, moveSpec(t), []envelope.AuditEntry{
		{At: at, From: envelope.Draft, Event: envelope.ChecksPass, To: envelope.Ready, Note: "constructed check"},
	})
	if err != nil {
		t.Fatal(err)
	}
	saveMove(t, a, second)
	entered, release := make(chan struct{}), make(chan struct{})
	a.movesRefresh = func(context.Context, envelope.Source) error {
		close(entered)
		<-release
		return nil
	}
	a.movesLog = func(string) {}
	a.movesChanged = func(context.Context) {}
	a.openURL = func(context.Context, string) {}
	checked := make(chan error, 1)
	go func() { checked <- a.checkMoves(a.ctx) }()
	<-entered
	handed := make(chan error, 1)
	go func() {
		_, err := a.TargetHandOff("move-2")
		handed <- err
	}()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case err := <-handed:
		if err != nil {
			t.Error(err)
		}
	case <-timer.C:
		t.Error("hand-off waited for the watcher's MFL refresh")
	}
	close(release)
	if err := <-checked; err != nil {
		t.Fatal(err)
	}
}
