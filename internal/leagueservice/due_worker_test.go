package leagueservice

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestDueWorkerNearestDeadlineCatchupAndRestart(t *testing.T) {
	f := newDueFixture(t)
	bid := f.bid(t)
	next, err := f.worker.step(t.Context())
	if err != nil || !next.Equal(bid.RecordedAt.Add(24*time.Hour)) {
		t.Fatalf("next=%s err=%v", next, err)
	}
	f.clock.instant.Add(int64(12 * time.Hour))
	cut := f.execute(t, "owner.two", commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0099", Franchise: "franchise.02"}})
	f.clock.instant.Store(bid.RecordedAt.Add(24 * time.Hour).UnixNano())
	next, err = f.worker.step(t.Context())
	if err != nil || !next.Equal(cut.RecordedAt.Add(24*time.Hour)) {
		t.Fatalf("waiver next=%s err=%v", next, err)
	}
	events := f.events(t)
	last := events[len(events)-1].Payload.Commissioner
	if last.Principal != "native.bid.clock" || last.Command.Command != commissioner.BidsProcess {
		t.Fatalf("not an automatic bid award: %+v", last)
	}
	// A fresh store and worker catch up from disk, without a prior notification.
	if err := f.pools.Close(); err != nil {
		t.Fatal(err)
	}
	f.open(t)
	f.clock.instant.Store(cut.RecordedAt.Add(24 * time.Hour).UnixNano())
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	count := len(f.events(t))
	if count != len(events)+1 {
		t.Fatalf("catch-up appended %d events", count-len(events))
	}
	if err := f.pools.Close(); err != nil {
		t.Fatal(err)
	}
	f.open(t)
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(f.events(t)) != count {
		t.Fatal("restart processed an award twice")
	}
}

func startDueWorker(t *testing.T, w *DueWorker) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); w.Run(ctx, func(err error) { t.Errorf("worker: %v", err) }) }()
	stop := func() { cancel(); <-done }
	t.Cleanup(stop)
	return stop
}

func waitDueEvents(t *testing.T, f *dueFixture, count int) {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for len(f.events(t)) < count {
		select {
		case <-timeout:
			t.Fatal("worker did not reconcile committed state")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func TestDueWorkerLostWakeAndCommitWake(t *testing.T) {
	for _, lost := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "lost"}[lost], func(t *testing.T) {
			f := newDueFixture(t)
			f.worker.reconcile = 20 * time.Millisecond
			startDueWorker(t, f.worker)
			f.bid(t)
			count := len(f.events(t))
			f.clock.instant.Add(int64(24 * time.Hour))
			if !lost {
				f.worker.Wake()
				f.worker.Wake()
			}
			waitDueEvents(t, f, count+1)
		})
	}
}

func TestDueWorkerTimerUsesNearestDeadline(t *testing.T) {
	f := newDueFixture(t)
	f.execute(t, "creator", commissioner.RulesSet, commissioner.RawArgs{Field: "bids.standFor", Value: []byte(`50000000`)})
	f.bid(t)
	count := len(f.events(t))
	// Real advancing time exercises the deadline timer, not the minute reconciliation.
	at := f.clock.Now()
	start := time.Now()
	clock := advancingDueClock{at: at, start: start}
	f.store.SetClock(clock.Now)
	f.worker.clock = clock
	startDueWorker(t, f.worker)
	waitDueEvents(t, f, count+1)
}

type advancingDueClock struct{ at, start time.Time }

func (c advancingDueClock) Now() time.Time { return c.at.Add(time.Since(c.start)) }
func (advancingDueClock) Trusted() error   { return nil }

func TestDueWorkerAwardBidRaceProcessesOnce(t *testing.T) {
	f := newDueFixture(t)
	bid := f.bid(t)
	f.clock.instant.Add(int64(24 * time.Hour))
	raw := f.raw(t, commissioner.BidRaise, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Bid: bid.Entry, Franchise: "franchise.01", Salary: 600000000, Years: 1}})
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() { _, _ = f.worker.step(t.Context()) })
	}
	wg.Go(func() {
		if _, err := f.store.ExecuteCommissionerWithFacts(t.Context(), raw, "owner.one", nil, f.facts, time.UTC); err == nil {
			t.Error("late raise won against award")
		}
	})
	wg.Wait()
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	events := f.events(t)
	if int64(len(events)) != raw.Expected+1 {
		t.Fatalf("race appended %d events", int64(len(events))-raw.Expected)
	}
}

func TestDueWorkerLateClaimWithoutProgress(t *testing.T) {
	f := newDueFixture(t)
	cut := f.execute(t, "owner.two", commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0099", Franchise: "franchise.02"}})
	for _, offset := range []time.Duration{0, time.Nanosecond} {
		f.clock.instant.Store(cut.RecordedAt.Add(24*time.Hour + offset).UnixNano())
		raw := f.raw(t, commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Window: cut.Entry, Franchise: "franchise.01"}})
		if _, err := f.store.ExecuteCommissionerWithFacts(t.Context(), raw, "owner.one", nil, f.facts, time.UTC); err == nil || !strings.Contains(err.Error(), "closed") {
			t.Fatalf("late claim: %v", err)
		}
		if int64(len(f.events(t))) != raw.Expected {
			t.Fatal("late claim appended events")
		}
	}
}

func TestDueWorkerBlockedAwardDoesNotSpinAtOldDeadline(t *testing.T) {
	f := newDueFixture(t)
	f.execute(t, "creator", commissioner.RulesSet, commissioner.RawArgs{Field: "bids.ufa", Value: []byte(`"blind"`)})
	f.bid(t)
	f.execute(t, "owner.two", commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5001", Franchise: "franchise.02", Salary: 500000000, Years: 1}})
	f.clock.instant.Add(int64(24 * time.Hour))
	count := len(f.events(t))
	next, err := f.worker.step(t.Context())
	if err != nil || !next.IsZero() {
		t.Fatalf("blocked award should wait for reconciliation: %s %v", next, err)
	}
	if len(f.events(t)) != count {
		t.Fatal("worker broke an unresolved tie")
	}
}
