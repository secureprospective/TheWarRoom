package research

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

func TestDepthPreparationCoalescesPlayerReads(t *testing.T) {
	var held Held
	var calls atomic.Int32
	started, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(finished)
		held.PrepareDepth(context.Background(), func() {
			calls.Add(1)
			close(started)
			<-release
			held.PublishStatus("depth ready")
		})
	}()
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var readers sync.WaitGroup
	for range 21 {
		readers.Go(func() {
			held.PrepareDepth(ctx, func() { calls.Add(1) })
		})
	}
	readers.Wait()
	if got := calls.Load(); got != 1 || held.SnapshotStatus() != "" {
		t.Errorf("waiters launched work or published early: calls=%d", got)
	}
	close(release)
	<-finished
	if held.SnapshotStatus() != "depth ready" {
		t.Fatal("cancelling a waiter cancelled the producer")
	}
	held.PrepareDepth(context.Background(), func() { calls.Add(1) })
	if calls.Load() != 2 {
		t.Fatal("completed refresh blocked the next retry")
	}
}
