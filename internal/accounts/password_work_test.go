package accounts

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPasswordWorkHundredRequests(t *testing.T) {
	work := NewPasswordWork()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	release1, err := work.acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	release2, err := work.acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var refused atomic.Int32
	var completed atomic.Int32
	var wg sync.WaitGroup
	for range 98 {
		wg.Go(func() {
			release, err := work.acquire(ctx)
			if errors.Is(err, ErrPasswordBusy) {
				refused.Add(1)
				return
			}
			if err != nil {
				t.Error(err)
				return
			}
			completed.Add(1)
			release()
		})
	}
	deadline := time.Now().Add(3 * time.Second)
	for len(work.admitted) != 6 || refused.Load() != 94 {
		if time.Now().After(deadline) {
			t.Fatalf("active=%d admitted=%d refused=%d", len(work.active), len(work.admitted), refused.Load())
		}
		time.Sleep(time.Millisecond)
	}
	if len(work.active) != 2 {
		t.Fatal("active hash limit changed")
	}
	release1()
	release2()
	wg.Wait()
	if completed.Load() != 4 || len(work.active) != 0 || len(work.admitted) != 0 {
		t.Fatal("queued work did not drain")
	}
}

func TestPasswordWorkCanceledQueueReleasesCapacity(t *testing.T) {
	work := NewPasswordWork()
	first, err := work.acquire(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer first()
	second, err := work.acquire(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer second()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	if _, err := work.Hash(ctx, "private-password"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("canceled hash: %v", err)
	}
	if len(work.admitted) != 2 || len(work.active) != 2 {
		t.Fatal("canceled waiter leaked capacity")
	}
	cancel()
	if _, err := work.Verify(ctx, "bad hash", "password"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("canceled verification: %v", err)
	}
}
