package leagueservice

import (
	"context"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"golang.org/x/sys/unix"
)

func TestDueWorkerHostOnlyWakesAfterCommit(t *testing.T) {
	f := newDueFixture(t)
	host := f.worker.Host(fixtureChangeHost{Store: f.store})
	raw := f.raw(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5001", Franchise: "franchise.01", Salary: 500000000, Years: 1}})
	ctx := commissioner.WithCredentialAdmission(t.Context(), admissionFunc(func(context.Context, commissioner.Access) error { return nil }))
	receipt, err := host.ExecuteCommissionerWithFacts(ctx, raw, "owner.one", nil, f.facts, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-f.worker.wake:
	default:
		t.Fatal("commit did not wake the worker")
	}
	if int64(len(f.events(t))) != receipt.Revision {
		t.Fatal("wake preceded commit")
	}
	f.clock.untrusted.Store(true)
	if _, err := host.ExecuteCommissionerWithFacts(ctx, raw, "owner.one", nil, f.facts, time.UTC); err == nil {
		t.Fatal("retry bypassed clock health")
	}
	if int64(len(f.events(t))) != receipt.Revision {
		t.Fatal("refused retry appended events")
	}
}

func TestKernelClockSynchronized(t *testing.T) {
	for _, tc := range []struct {
		state  int
		status int32
		want   bool
	}{
		{unix.TIME_OK, 0, true},
		{unix.TIME_INS, unix.STA_PLL, true},
		{unix.TIME_OK, unix.STA_UNSYNC, false},
		{unix.TIME_OK, unix.STA_PLL | unix.STA_UNSYNC, false},
		{unix.TIME_ERROR, 0, false},
	} {
		if got := kernelClockSynchronized(tc.state, tc.status); got != tc.want {
			t.Fatalf("state %d status %#x: got %v", tc.state, tc.status, got)
		}
	}
}
