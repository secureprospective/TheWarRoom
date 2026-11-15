package sources_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"
)

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
