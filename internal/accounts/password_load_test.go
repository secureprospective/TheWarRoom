package accounts

import (
	"errors"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"
)

// This intentionally runs real fixed-cost Argon2, so the same test binary can
// measure the target CT without a Go installation or any production credentials.
func TestPasswordWorkRealKDFLoad(t *testing.T) {
	encoded, err := HashPassword("disposable-load-password")
	if err != nil {
		t.Fatal(err)
	}
	work := NewPasswordWork()
	start := make(chan struct{})
	samples := make(chan time.Duration, 100)
	var wg sync.WaitGroup
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range 100 {
		wg.Go(func() {
			<-start
			release, err := work.acquire(t.Context())
			if errors.Is(err, ErrPasswordBusy) {
				return
			}
			if err != nil {
				t.Error(err)
				return
			}
			defer release()
			began := time.Now()
			ok, err := VerifyPassword(encoded, "disposable-load-password")
			if err != nil || !ok {
				t.Errorf("real password verify: %v", err)
			}
			samples <- time.Since(began)
		})
	}
	close(start)
	wg.Wait()
	close(samples)
	runtime.ReadMemStats(&after)
	durations := make([]time.Duration, 0, 100)
	for duration := range samples {
		durations = append(durations, duration)
	}
	if len(durations) < 2 {
		t.Fatalf("insufficient admitted hashes: %d", len(durations))
	}
	slices.Sort(durations)
	index := (95*len(durations)+99)/100 - 1
	t.Logf("100 attempts: admitted=%d refused=%d real hash p95=%s heap before=%d after=%d; target gate <=750ms, memory requires external RSS/cgroup measurement", len(durations), 100-len(durations), durations[index], before.HeapAlloc, after.HeapAlloc)
}
