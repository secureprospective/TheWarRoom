package core_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
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

func TestSourcesFactsSerializerKeepsTryLockAndCallbackSpan(t *testing.T) {
	s := coretest.New(t).Build()
	s.FactsMu.Lock()
	called := false
	msg, err := s.SerializeSourcesCompile(func() (string, error) { called = true; return "", nil })
	s.FactsMu.Unlock()
	if called || err != nil || msg != "The checked facts are already compiling." {
		t.Fatal(msg, err, called)
	}
	_, err = s.SerializeSourcesCompile(func() (string, error) {
		if s.FactsMu.TryLock() {
			s.FactsMu.Unlock()
			t.Error("compile callback lost original lock span")
		}
		return "compiled", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !s.FactsMu.TryLock() {
		t.Fatal("compile did not release serializer")
	}
	s.FactsMu.Unlock()
}

func TestSourcesSharedReadsDetached(t *testing.T) {
	s := coretest.New(t).Build()
	lookup, err := normalize.NewLookup([]players.RawPlayer{{ID: "0042", Name: "Held", Position: "QB"}})
	if err != nil {
		t.Fatal(err)
	}
	s.Lookup, s.HasLookup = lookup, true
	_, err = s.League.Replace(t.Context(), state.MirrorSnapshot{Season: 2026,
		Players: []state.MirrorPlayer{{MFLID: "0042", FranchiseID: "0001", RosterStatus: domain.RosterActive}}})
	if err != nil {
		t.Fatal(err)
	}
	positions, rostered := s.SourcesPositions(), s.SourcesRostered()
	if positions["0042"] != "QB" || rostered["0042"] != "0001" {
		t.Fatal(positions, rostered)
	}
	positions["0042"], rostered["0042"] = "mutated", "mutated"
	if s.SourcesPositions()["0042"] != "QB" || s.SourcesRostered()["0042"] != "0001" {
		t.Fatal("shared reads alias state")
	}
}
