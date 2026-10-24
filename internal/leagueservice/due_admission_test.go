package leagueservice

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestDueWorkerClockUntrustedSuspendsQueuedMutationAndCatchup(t *testing.T) {
	f := newDueFixture(t)
	bid := f.bid(t)
	raw := f.raw(t, commissioner.BidRaise, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Bid: bid.Entry, Franchise: "franchise.01", Salary: 600000000, Years: 1}})
	var admissions atomic.Int32
	ctx := commissioner.WithCredentialAdmission(t.Context(), admissionFunc(func(context.Context, commissioner.Access) error { admissions.Add(1); return nil }))
	host := f.worker.Host(fixtureChangeHost{Store: f.store})
	tx, err := f.pools.Write().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	waits := f.pools.Write().Stats().WaitCount
	done := make(chan error, 1)
	go func() {
		_, err := host.ExecuteCommissionerWithFacts(ctx, raw, "owner.one", nil, f.facts, time.UTC)
		done <- err
	}()
	limit := time.After(5 * time.Second)
	for f.pools.Write().Stats().WaitCount == waits {
		select {
		case <-limit:
			t.Fatal("mutation never queued")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	f.clock.untrusted.Store(true)
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err == nil || !strings.Contains(err.Error(), "untrusted") {
		t.Fatalf("queued clock admission: %v", err)
	}
	if admissions.Load() != 1 {
		t.Fatal("clock wrapper bypassed the original credential")
	}
	if int64(len(f.events(t))) != raw.Expected {
		t.Fatal("queued clock refusal appended events")
	}
	select {
	case <-f.worker.wake:
		t.Fatal("rollback published a commit wake")
	default:
	}
	f.clock.instant.Add(int64(24 * time.Hour))
	if _, err := f.worker.step(t.Context()); err == nil {
		t.Fatal("untrusted clock processed awards")
	}
	if int64(len(f.events(t))) != raw.Expected {
		t.Fatal("untrusted catchup appended events")
	}
	f.clock.untrusted.Store(false)
	if _, err := f.worker.step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if int64(len(f.events(t))) != raw.Expected+1 {
		t.Fatal("trusted clock did not resume due work")
	}
}

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

func TestDueWorkerSystemPrincipalCannotUpgradeAuthority(t *testing.T) {
	f := newDueFixture(t)
	for _, tc := range []struct {
		principal, command string
		want               bool
	}{
		{"native.bid.clock", commissioner.BidsProcess, true},
		{"native.waiver.clock", commissioner.WaiverProcess, true},
		{commissioner.DraftClockPrincipal, commissioner.DraftArchive, true},
		{"native.bid.clock", commissioner.BidPlace, false},
		{"native.bid.clock", commissioner.BidsOverride, false},
		{"native.bid.clock", commissioner.WaiverProcess, false},
		{"creator", commissioner.BidsProcess, false},
	} {
		access := commissioner.Access{Principal: tc.principal, Operation: tc.command, Mode: "execute"}
		err := (dueAdmission{worker: f.worker, system: true}).Admit(t.Context(), access)
		if (err == nil) != tc.want {
			t.Fatalf("system admission %s/%s: %v", tc.principal, tc.command, err)
		}
	}
	ctx := commissioner.WithCredentialAdmission(t.Context(), dueAdmission{worker: f.worker, system: true})
	raw := f.raw(t, commissioner.NotePost, commissioner.RawArgs{Note: &commissioner.RawNote{Text: "A worker must not post notes"}})
	if _, err := f.store.ExecuteCommissioner(ctx, raw, "native.bid.clock", nil); err == nil {
		t.Fatal("worker posted an unrelated command")
	}
	if int64(len(f.events(t))) != raw.Expected {
		t.Fatal("system denial changed state")
	}
	f.worker.clock = nil
	if _, err := f.worker.step(t.Context()); err == nil {
		t.Fatal("missing clock health failed open")
	}
}
