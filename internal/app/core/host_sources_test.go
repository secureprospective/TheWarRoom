package core_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
)

func TestSourcesLoadSerializerReleasesBeforeEvent(t *testing.T) {
	s := coretest.New(t).Build()
	var active, events atomic.Int32
	s.SeasonChanged = func(context.Context) {
		// A callback can re-enter the serializer: no serializer or holder lock spans an event.
		if !s.LoadingSignals.TryLock() {
			t.Error("event emitted before serializer released")
			return
		}
		s.LoadingSignals.Unlock()
		events.Add(1)
	}
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		s.SerializeSourcesLoad(t.Context(), func() { active.Add(1); close(entered); <-release; active.Add(-1) })
		close(done)
	}()
	<-entered
	if s.LoadingSignals.TryLock() {
		s.LoadingSignals.Unlock()
		t.Fatal("whole load not serialized")
	}
	if events.Load() != 0 {
		t.Fatal("health event before publication")
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("load blocked")
	}
	if active.Load() != 0 || events.Load() != 1 {
		t.Fatal("load/event order changed")
	}
}
