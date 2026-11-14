package serverhttp

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// countingHeads is a fake committed-head reader that counts reads per league.
type countingHeads struct {
	reads atomic.Int64
	head  atomic.Int64
}

func (c *countingHeads) read(context.Context, string) (int64, error) {
	c.reads.Add(1)
	return c.head.Load(), nil
}

// TestLeagueWatchReadsDoNotScaleWithStreams: one league's head is read once per tick whether one
// device or the full 104-stream budget is listening.
func TestLeagueWatchReadsDoNotScaleWithStreams(t *testing.T) {
	const every, window = 5 * time.Millisecond, 200 * time.Millisecond
	for _, streams := range []int{1, leagueStreamBudget(32)} {
		heads := &countingHeads{}
		watch := newLeagueWatch(heads.read, every)
		var leave []func()
		for range streams {
			_, unsubscribe := watch.subscribe("native.hosted")
			leave = append(leave, unsubscribe)
		}
		time.Sleep(window)
		reads := heads.reads.Load()
		for _, unsubscribe := range leave {
			unsubscribe()
		}
		if limit := int64(window/every) + 2; reads > limit || reads == 0 {
			t.Fatalf("%d streams: %d head reads in %v, want 1..%d", streams, reads, window, limit)
		}
		t.Logf("%d streams: %d head reads in %v", streams, reads, window)
	}
}

// TestLeagueWatchStopsWithLastStream: the watcher's goroutine ends when its last stream leaves, and a
// stream opened later starts a fresh one.
func TestLeagueWatchStopsWithLastStream(t *testing.T) {
	heads := &countingHeads{}
	watch := newLeagueWatch(heads.read, time.Millisecond)
	_, first := watch.subscribe("native.hosted")
	_, second := watch.subscribe("native.hosted")
	watch.mu.Lock()
	observer := watch.leagues["native.hosted"]
	watch.mu.Unlock()
	first()
	select {
	case <-observer.done:
		t.Fatal("watcher stopped while a stream was open")
	case <-time.After(20 * time.Millisecond):
	}
	second()
	select {
	case <-observer.done:
	case <-time.After(time.Second):
		t.Fatal("watcher outlived its last stream")
	}
	stopped := heads.reads.Load()
	time.Sleep(20 * time.Millisecond)
	if heads.reads.Load() != stopped || len(watch.leagues) != 0 {
		t.Fatal("stopped watcher kept reading or stayed registered")
	}
	news, again := watch.subscribe("native.hosted")
	defer again()
	select {
	case <-news:
	case <-time.After(time.Second):
		t.Fatal("a new stream did not restart the watcher")
	}
}

// TestLeagueWatchHandsNewestHeadToLateSubscriber: a stream joining after a commit was seen still gets
// that head, so a commit between its own head read and its subscribe is never skipped.
func TestLeagueWatchHandsNewestHeadToLateSubscriber(t *testing.T) {
	heads := &countingHeads{}
	heads.head.Store(9)
	watch := newLeagueWatch(heads.read, time.Millisecond)
	early, leave := watch.subscribe("native.hosted")
	defer leave()
	if got := <-early; got.head != 9 {
		t.Fatalf("early head=%d", got.head)
	}
	late, leaveLate := watch.subscribe("native.hosted")
	defer leaveLate()
	select {
	case got := <-late:
		if got.head != 9 {
			t.Fatalf("late head=%d", got.head)
		}
	default:
		t.Fatal("late subscriber was not handed the newest head")
	}
}

// TestCommitWakesOnlyChangedRecipients: one commit reaches every stream of the league through the
// shared watcher, and each recipient emits only if its own authorized view moved.
func TestCommitWakesOnlyChangedRecipients(t *testing.T) {
	recipients := []struct {
		name          string
		before, after int64
		wake          bool
	}{
		{"sees the public entry", 3, 4, true},
		{"private entry for someone else", 3, 3, false},
		{"commissioner sees it too", 7, 8, true},
	}
	heads := &countingHeads{}
	heads.head.Store(10)
	watch := newLeagueWatch(heads.read, time.Millisecond)
	type stream struct {
		poll *invalidationPoll
		news <-chan headNews
	}
	streams := make([]stream, len(recipients))
	for i, r := range recipients {
		after := r.after
		news, leave := watch.subscribe("native.hosted")
		defer leave()
		streams[i] = stream{poll: &invalidationPoll{head: 10, revision: r.before,
			read: func(context.Context) (commissioner.Ledger, error) { return commissioner.Ledger{Revision: after}, nil }}, news: news}
	}
	heads.head.Store(11)
	for i, r := range recipients {
		t.Run(r.name, func(t *testing.T) {
			deadline := time.After(time.Second)
			for {
				select {
				case got := <-streams[i].news:
					if got.head != 11 {
						continue
					}
					changed, err := streams[i].poll.changed(t.Context(), got.head)
					if err != nil || changed != r.wake {
						t.Fatalf("changed=%v err=%v want %v", changed, err, r.wake)
					}
					return
				case <-deadline:
					t.Fatal("commit did not reach the stream")
				}
			}
		})
	}
}

func TestSessionCheckCadenceIsJitteredAndBounded(t *testing.T) {
	seen := map[time.Duration]bool{}
	for range 64 {
		every := sessionCheckEvery()
		if every < 2*time.Second || every >= 4*time.Second {
			t.Fatalf("session check every %v, want [2s,4s)", every)
		}
		seen[every] = true
	}
	if len(seen) < 2 {
		t.Fatal("session checks are not jittered")
	}
}

// serveStream runs streamInvalidations behind a real HTTP server and returns the frames it writes.
func serveStream(t *testing.T, wakes streamWakes, poll *invalidationPoll, checkSession func(context.Context) error) (<-chan string, <-chan struct{}) {
	t.Helper()
	ended := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(ended)
		w.WriteHeader(http.StatusOK)
		_ = http.NewResponseController(w).Flush() // the real handler's first emit sends headers
		streamInvalidations(r.Context(), http.NewResponseController(w), w, wakes, poll, checkSession)
	}))
	t.Cleanup(server.Close)
	response, err := server.Client().Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	frames := make(chan string, 8)
	go func() {
		defer close(frames)
		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			if line := scanner.Text(); line != "" {
				frames <- line
			}
		}
	}()
	return frames, ended
}

// TestStreamWakesOnDeadlineWithoutCommit: with no head news at all, the stream's own timer wakes the
// recipient when the public deadline passes.
func TestStreamWakesOnDeadlineWithoutCommit(t *testing.T) {
	next := time.Now().Add(50 * time.Millisecond)
	poll := &invalidationPoll{head: 5, revision: 1, next: next, now: time.Now,
		deadline: func(context.Context) (time.Time, time.Time, bool) { return next.Add(time.Hour), time.Now(), true },
		read:     func(context.Context) (commissioner.Ledger, error) { return commissioner.Ledger{Revision: 1}, nil },
	}
	wakes := streamWakes{news: make(chan headNews), session: func() time.Duration { return time.Hour }}
	frames, _ := serveStream(t, wakes, poll, func(context.Context) error { return nil })
	select {
	case frame := <-frames:
		if frame != "event: invalidate" {
			t.Fatalf("frame %q", frame)
		}
	case <-time.After(time.Second):
		t.Fatal("deadline did not wake the stream")
	}
}

// TestStreamClosesOnRevokedSession: a revoked session ends the stream at the next session check, and a
// head move checks the session at once.
func TestStreamClosesOnRevokedSession(t *testing.T) {
	for _, tc := range []struct {
		name    string
		session time.Duration
		commit  bool
	}{
		{"next cadence check", 20 * time.Millisecond, false},
		{"checked on a commit wake", time.Hour, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var revoked atomic.Bool
			news := make(chan headNews, 1)
			poll := &invalidationPoll{head: 5, revision: 1,
				read: func(context.Context) (commissioner.Ledger, error) { return commissioner.Ledger{Revision: 1}, nil }}
			wakes := streamWakes{news: news, session: func() time.Duration { return tc.session }}
			_, ended := serveStream(t, wakes, poll, func(context.Context) error {
				if revoked.Load() {
					return errors.New("revoked")
				}
				return nil
			})
			revoked.Store(true)
			if tc.commit {
				news <- headNews{head: 6}
			}
			select {
			case <-ended:
			case <-time.After(time.Second):
				t.Fatal("revoked stream stayed open")
			}
		})
	}
}
