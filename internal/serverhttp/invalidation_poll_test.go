package serverhttp

import (
	"context"
	"errors"
	"testing"

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
