package serverhttp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestInvalidationPollIdleAndRecipientChanges(t *testing.T) {
	head, token := int64(10), int64(3)
	heads, reads := 0, 0
	poll := invalidationPoll{head: head, revision: token,
		committedHead: func(context.Context) (int64, error) { heads++; return head, nil },
		read: func(context.Context) (commissioner.Ledger, error) {
			reads++
			return commissioner.Ledger{Revision: token}, nil
		},
	}
	for range 24 {
		changed, err := poll.changed(t.Context())
		if err != nil || changed {
			t.Fatalf("idle changed=%v err=%v", changed, err)
		}
	}
	if heads != 24 || reads != 0 {
		t.Fatalf("idle heads=%d ledger reads=%d", heads, reads)
	}
	head++ // A hidden commit changes only the work gate.
	if changed, err := poll.changed(t.Context()); err != nil || changed {
		t.Fatalf("hidden changed=%v err=%v", changed, err)
	}
	if reads != 1 {
		t.Fatalf("hidden ledger reads=%d", reads)
	}
	head++
	token++
	if changed, err := poll.changed(t.Context()); err != nil || !changed {
		t.Fatalf("visible changed=%v err=%v", changed, err)
	}
	for range 24 {
		if changed, err := poll.changed(t.Context()); err != nil || changed {
			t.Fatalf("idle after commit changed=%v err=%v", changed, err)
		}
	}
	if reads != 2 {
		t.Fatalf("ledger reads=%d want=2", reads)
	}
}

func TestInvalidationPollFailureDoesNotAdvance(t *testing.T) {
	refused := errors.New("read refused")
	poll := invalidationPoll{head: 1, revision: 7,
		committedHead: func(context.Context) (int64, error) { return 2, nil },
		read:          func(context.Context) (commissioner.Ledger, error) { return commissioner.Ledger{}, refused },
	}
	if changed, err := poll.changed(t.Context()); changed || !errors.Is(err, refused) {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if poll.head != 1 || poll.revision != 7 {
		t.Fatal("failed read advanced poll")
	}
	poll.committedHead = func(context.Context) (int64, error) { return 0, refused }
	if changed, err := poll.changed(t.Context()); changed || !errors.Is(err, refused) {
		t.Fatalf("head failure changed=%v err=%v", changed, err)
	}
}

// TestInvalidationPollWakesOnDeadline: the clock passing the next public deadline wakes the device once
// with no new entry; a failed read does not consume it; an untimed league never wakes on the clock.
func TestInvalidationPollWakesOnDeadline(t *testing.T) {
	start := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	now, next, timed := start, start.Add(time.Hour), true
	head := int64(10)
	var readErr error
	poll := invalidationPoll{head: head, revision: 3, next: next,
		committedHead: func(context.Context) (int64, error) { return head, nil },
		read:          func(context.Context) (commissioner.Ledger, error) { return commissioner.Ledger{Revision: 3}, readErr },
		deadline:      func(context.Context) (time.Time, time.Time, bool) { return next, now, timed },
	}
	if changed, err := poll.changed(t.Context()); err != nil || changed {
		t.Fatalf("idle changed=%v err=%v", changed, err)
	}
	now, next = start.Add(time.Hour), start.Add(2*time.Hour)
	if changed, err := poll.changed(t.Context()); err != nil || !changed {
		t.Fatalf("deadline changed=%v err=%v", changed, err)
	}
	if changed, _ := poll.changed(t.Context()); changed {
		t.Fatal("one deadline wakes once")
	}
	now, next, head = start.Add(2*time.Hour), start.Add(3*time.Hour), head+1
	readErr = errors.New("read refused")
	if _, err := poll.changed(t.Context()); err == nil {
		t.Fatal("the read must fail")
	}
	readErr = nil
	if changed, err := poll.changed(t.Context()); err != nil || !changed {
		t.Fatalf("a failed read must not consume the deadline: changed=%v err=%v", changed, err)
	}
	now, timed = start.Add(4*time.Hour), false
	if changed, _ := poll.changed(t.Context()); !changed {
		t.Fatal("the passed deadline still wakes")
	}
	now = start.Add(5 * time.Hour)
	if changed, _ := poll.changed(t.Context()); changed {
		t.Fatal("an untimed league never wakes on the clock")
	}
}
