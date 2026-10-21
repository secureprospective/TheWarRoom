package sources_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"
)

func TestHeldIngressAndEgressDetached(t *testing.T) {
	h := &sourcesapp.Held{}
	cw := sourcesapp.CrosswalkReport{Rates: []sourcesapp.CrosswalkRate{{Position: "QB"}}, Unmatched: []sourcesapp.CrosswalkMiss{{MFLID: "0042"}}}
	signals := sourcesapp.SignalsReport{Loads: []sourcesapp.SignalLoad{{Feed: "stats", Missing: []string{"held"}}}}
	h.PublishCrosswalk(cw)
	h.PublishSignals(signals)
	cw.Rates[0].Position, cw.Unmatched[0].MFLID = "mutated", "mutated"
	signals.Loads[0].Feed, signals.Loads[0].Missing[0] = "mutated", "mutated"
	assertHeld := func() {
		t.Helper()
		c, s := h.SnapshotCrosswalk(), h.SnapshotSignals()
		if c.Rates[0].Position != "QB" || c.Unmatched[0].MFLID != "0042" || s.Loads[0].Feed != "stats" || s.Loads[0].Missing[0] != "held" {
			t.Fatal("holder aliases caller storage", c, s)
		}
	}
	assertHeld()
	c, s := h.SnapshotCrosswalk(), h.SnapshotSignals()
	c.Rates[0].Position, c.Unmatched[0].MFLID = "mutated", "mutated"
	s.Loads[0].Feed, s.Loads[0].Missing[0] = "mutated", "mutated"
	assertHeld()
}

func TestHeldPreservesNilAndEmptyAtEveryDepth(t *testing.T) {
	h := &sourcesapp.Held{}
	for _, cw := range []sourcesapp.CrosswalkReport{{}, {Rates: []sourcesapp.CrosswalkRate{}, Unmatched: []sourcesapp.CrosswalkMiss{}}} {
		h.PublishCrosswalk(cw)
		if !reflect.DeepEqual(cw, h.SnapshotCrosswalk()) {
			t.Fatal("crosswalk slice shape changed")
		}
	}
	for _, s := range []sourcesapp.SignalsReport{{}, {Loads: []sourcesapp.SignalLoad{}}, {Loads: []sourcesapp.SignalLoad{{}, {Missing: []string{}}}}} {
		h.PublishSignals(s)
		if !reflect.DeepEqual(s, h.SnapshotSignals()) {
			t.Fatal("signals nested slice shape changed")
		}
	}
}

func TestHeldPublishVersusRead(t *testing.T) {
	t.Parallel()
	h := &sourcesapp.Held{}
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 250 {
				h.PublishCrosswalk(sourcesapp.CrosswalkReport{Rates: []sourcesapp.CrosswalkRate{{Position: "QB"}}})
				h.PublishSignals(sourcesapp.SignalsReport{Loads: []sourcesapp.SignalLoad{{Missing: []string{"held"}}}})
				cw, s := h.SnapshotCrosswalk(), h.SnapshotSignals()
				if len(cw.Rates) > 0 {
					cw.Rates[0].Position = "caller"
				}
				if len(s.Loads) > 0 {
					s.Loads[0].Missing[0] = "caller"
				}
			}
		})
	}
	wg.Wait()
}

type blockedTransport struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (b *blockedTransport) RoundTrip(*http.Request) (*http.Response, error) {
	b.once.Do(func() { close(b.entered); <-b.release })
	return nil, errors.New("synthetic offline; no socket")
}

func TestReadsCompleteDuringActualSignalsLoad(t *testing.T) {
	h := testHost(t)
	transport := &blockedTransport{entered: make(chan struct{}), release: make(chan struct{})}
	h.transport = transport
	h.changed = func() {
		if len(h.held.SnapshotSignals().Loads) == 0 {
			t.Error("health event before held publication")
		}
		if !h.loading.TryLock() {
			t.Error("health event before serializer release")
			return
		}
		h.loading.Unlock()
	}
	var release sync.Once
	defer release.Do(func() { close(transport.release) })
	finished := make(chan struct{})
	go func() { sourcesapp.LoadSignals(context.Background(), h, h.held); close(finished) }()
	t.Cleanup(func() {
		release.Do(func() { close(transport.release) })
		select {
		case <-finished:
		case <-time.After(10 * time.Second):
			t.Error("load worker did not join")
		}
	})
	select {
	case <-transport.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("load did not reach transport")
	}
	read := make(chan error, 1)
	go func() {
		h.api().GetCrosswalkReport()
		h.api().TargetCrosswalk()
		_, err := h.api().TargetDataSources()
		read <- err
	}()
	select {
	case err := <-read:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("reading blocked behind load I/O")
	}
	var readers sync.WaitGroup
	readers.Go(func() {
		for {
			select {
			case <-finished:
				return
			default:
				h.api().GetCrosswalkReport()
				h.api().TargetCrosswalk()
				h.held.SnapshotSignals()
			}
		}
	})
	release.Do(func() { close(transport.release) })
	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("load did not finish")
	}
	readers.Wait()
	if len(h.held.SnapshotSignals().Loads) == 0 {
		t.Fatal("load was not published")
	}
}
