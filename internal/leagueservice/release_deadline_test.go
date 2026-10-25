package leagueservice

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestReleaseMatrixBidDeadlineWithoutWorker(t *testing.T) {
	t.Parallel()
	for _, command := range []string{commissioner.BidPlace, commissioner.BidRaise, commissioner.BidWithdraw} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			f := newDueFixture(t)
			bid := f.bid(t)
			// The only successful append runs last, so refusals share the same fixture.
			for _, delta := range []time.Duration{0, time.Nanosecond, -time.Nanosecond} {
				t.Run(delta.String(), func(t *testing.T) {
					raw, principal := releaseBidCommand(t, f, command, bid.Entry)
					before := f.events(t)
					f.clock.instant.Store(bid.RecordedAt.Add(24*time.Hour + delta).UnixNano())
					// No worker has run: admission must use the authoritative clock, not award progress.
					_, err := f.store.ExecuteCommissionerWithFacts(t.Context(), raw, principal, nil, f.facts, time.UTC)
					if delta < 0 {
						if err != nil {
							t.Fatal("on-time bid refused", err)
						}
						if len(f.events(t)) != len(before)+1 {
							t.Fatal("on-time bid did not append once")
						}
					} else if err == nil || !reflect.DeepEqual(before, f.events(t)) {
						t.Fatalf("late bid accepted or changed history: %v", err)
					}
				})
			}
		})
	}
}

func TestReleaseMatrixWaiverDeadlineWithoutWorker(t *testing.T) {
	t.Parallel()
	f := newDueFixture(t)
	cut := f.execute(t, "owner.two", commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0099", Franchise: "franchise.02"}})
	for _, delta := range []time.Duration{0, time.Nanosecond, -time.Nanosecond} {
		t.Run(delta.String(), func(t *testing.T) {
			raw := f.raw(t, commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Window: cut.Entry, Franchise: "franchise.01"}})
			before := f.events(t)
			f.clock.instant.Store(cut.RecordedAt.Add(24*time.Hour + delta).UnixNano())
			_, err := f.store.ExecuteCommissionerWithFacts(t.Context(), raw, "owner.one", nil, f.facts, time.UTC)
			if delta < 0 {
				if err != nil {
					t.Fatal("on-time claim refused", err)
				}
				if len(f.events(t)) != len(before)+1 {
					t.Fatal("claim did not append once")
				}
			} else if err == nil || !reflect.DeepEqual(before, f.events(t)) {
				t.Fatalf("late claim accepted or changed history: %v", err)
			}
		})
	}
}

func releaseBidCommand(t *testing.T, f *dueFixture, command, entry string) (commissioner.RawCommand, string) {
	t.Helper()
	args := &commissioner.RawBidArgs{Franchise: "franchise.01", Bid: entry}
	principal := "owner.one"
	switch command {
	case commissioner.BidPlace:
		args = &commissioner.RawBidArgs{Franchise: "franchise.02", Player: "5001", Salary: 600000000, Years: 1}
		principal = "owner.two"
	case commissioner.BidRaise:
		args.Salary, args.Years = 600000000, 1
	}
	return f.raw(t, command, commissioner.RawArgs{Bid: args}), principal
}

func TestReleaseMatrixQueuedBidsUseCommitClock(t *testing.T) {
	t.Parallel()
	for _, command := range []string{commissioner.BidPlace, commissioner.BidRaise, commissioner.BidWithdraw} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			f := newDueFixture(t)
			bid := f.bid(t)
			raw, principal := releaseBidCommand(t, f, command, bid.Entry)
			before := f.events(t)
			tx, err := f.pools.Write().BeginTx(t.Context(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback() }()
			waits := f.pools.Write().Stats().WaitCount
			done := make(chan error, 1)
			go func() {
				_, err := f.store.ExecuteCommissionerWithFacts(t.Context(), raw, principal, nil, f.facts, time.UTC)
				done <- err
			}()
			limit := time.After(5 * time.Second)
			for f.pools.Write().Stats().WaitCount == waits {
				select {
				case <-limit:
					t.Fatal("bid never queued for the writer")
				default:
					time.Sleep(time.Millisecond)
				}
			}
			f.clock.instant.Store(bid.RecordedAt.Add(24 * time.Hour).UnixNano())
			if err := tx.Rollback(); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if err == nil || !reflect.DeepEqual(before, f.events(t)) {
					t.Fatalf("queued late bid accepted or changed history: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("queued bid did not finish")
			}
		})
	}
}
