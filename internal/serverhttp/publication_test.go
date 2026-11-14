package serverhttp

import (
	"context"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestPublisherRecipientPrivacyAndPublicSnapshots(t *testing.T) {
	p := NewPublisher()
	p.ObserveView("owner", "native.test", 1)
	p.ObserveView("other", "native.test", 1)
	before := p.Versions("other", "native.test")
	p.PublishScores("native.test", 2026, 10)
	public := p.Versions("other", "native.test")
	p.ObserveView("owner", "native.test", 2)
	after := p.Versions("other", "native.test")
	if len(changedTopics(public.Topics, after.Topics)) != 0 {
		t.Fatal("private revision moved another recipient")
	}
	delta := changedTopics(before.Topics, after.Topics)
	if len(delta) != 1 || delta[0].Kind != "scores" || delta[0].Week != 10 || delta[0].Version != "1" {
		t.Fatalf("score scope: %+v", delta)
	}
	p.ObserveView("owner", "native.test", 2)
	if got := p.Versions("owner", "native.test"); got.Topics[0].Version != "2" {
		t.Fatalf("duplicate stream bumped revision: %+v", got)
	}
	if NewPublisher().Versions("owner", "native.test").ServerEpoch == before.ServerEpoch {
		t.Fatal("restart reused epoch")
	}
}

func TestPublisherPreservesRecipientViewGate(t *testing.T) {
	p := NewPublisher()
	p.ObserveView("owner", "native.test", 7)
	before := p.Versions("owner", "native.test")
	poll := invalidationPoll{head: 1, revision: 7, read: func(context.Context) (commissioner.Ledger, error) { return commissioner.Ledger{Revision: 7}, nil }}
	changed, err := poll.changed(t.Context(), 2)
	if err != nil || changed {
		t.Fatalf("invisible head: %v %v", changed, err)
	}
	if got := changedTopics(before.Topics, p.Versions("owner", "native.test").Topics); len(got) != 0 {
		t.Fatalf("private head moved topics: %+v", got)
	}
}

func TestPublisherClockBoundaryOnceAcrossStreams(t *testing.T) {
	p := NewPublisher()
	at := time.Now().UTC().Truncate(time.Second)
	p.clockBoundary("native.test", at, at.Add(time.Hour))
	p.clockBoundary("native.test", at, at.Add(time.Hour))
	got := p.Versions("owner", "native.test")
	if len(got.Topics) != 1 || got.Topics[0].Version != "1" || got.Topics[0].NextBoundary != at.Add(time.Hour).Format(time.RFC3339) {
		t.Fatalf("boundary: %+v", got)
	}
}
