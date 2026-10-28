package leagueservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// DueClock must report untrusted wall time until the host has established clock health.
// Implementations must be safe for simultaneous worker and transaction admission calls.
type DueClock interface {
	Now() time.Time
	Trusted() error
}

// HostClock rereads the kernel's NTP discipline state on each admission; nothing is cached.
// Platforms without a kernel status source remain untrusted.
type HostClock struct{ synchronized func() (bool, error) }

func NewHostClock() HostClock    { return HostClock{synchronized: kernelSynchronized} }
func (HostClock) Now() time.Time { return time.Now() }
func (c HostClock) Trusted() error {
	if c.synchronized == nil {
		return fmt.Errorf("clock synchronization source unavailable")
	}
	ok, err := c.synchronized()
	if err != nil {
		return fmt.Errorf("read clock synchronization: %w", err)
	}
	if !ok {
		return fmt.Errorf("host clock is not NTP synchronized")
	}
	return nil
}

// DueWorker owns hosted scheduling; it never starts the desktop ticker. All effects
// use the existing due processors and their transaction-time legality checks.
type DueWorker struct {
	store     *history.Store
	facts     func() nativecheck.Facts
	clock     DueClock
	wake      chan struct{}
	reconcile time.Duration
}

// NewDueWorker shares the store clock configured by its composition root.
func NewDueWorker(store *history.Store, facts func() nativecheck.Facts, clock DueClock) *DueWorker {
	return &DueWorker{store: store, facts: facts, clock: clock, wake: make(chan struct{}, 1), reconcile: time.Minute}
}

// Host adds post-commit wakes and transaction-time clock admission to a hosted adapter.
// Every hosted mutation must use this adapter, including non-HTTP entry points.
func (w *DueWorker) Host(host Host) Host { return dueHost{Host: host, worker: w} }

type dueHost struct {
	Host
	worker *DueWorker
}

func (h dueHost) ExecuteCommissionerWithFacts(ctx context.Context, raw commissioner.RawCommand,
	principal string, auth commissioner.Authorizer, facts nativecheck.Facts, zone *time.Location) (commissioner.Receipt, error) {
	guarded := commissioner.WithCredentialAdmission(ctx, dueAdmission{worker: h.worker, credential: func(access commissioner.Access) error {
		return commissioner.AdmitCredential(ctx, access)
	}})
	receipt, err := h.Host.ExecuteCommissionerWithFacts(guarded, raw, principal, auth, facts, zone)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("hosted change: %w", err)
	}
	h.worker.Wake()
	return receipt, nil
}

// Wake coalesces notifications. Correctness does not depend on delivery: startup and
// bounded reconciliation also reread committed state, including another store's writes.
func (w *DueWorker) Wake() {
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

func (w *DueWorker) trusted() error {
	if w.clock == nil {
		return fmt.Errorf("server clock health is unavailable")
	}
	if err := w.clock.Trusted(); err != nil {
		return fmt.Errorf("server clock is untrusted: %w", err)
	}
	return nil
}

// Run catches up before waiting. Errors are reported and retried; a revision race
// cannot kill scheduling. Cancellation joins the worker before the store is closed.
func (w *DueWorker) Run(ctx context.Context, report func(error)) {
	for ctx.Err() == nil {
		wait := w.reconcile
		next, err := w.step(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			report(err)
			wait = min(wait, time.Second)
		}
		if !next.IsZero() {
			wait = min(wait, max(time.Duration(0), next.Sub(w.clock.Now())))
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
		case <-w.wake:
		case <-w.store.CommissionerCommits():
		case <-timer.C:
		}
		timer.Stop()
	}
}

func (w *DueWorker) step(ctx context.Context) (time.Time, error) {
	if err := w.trusted(); err != nil {
		return time.Time{}, err
	}
	leagues, err := w.store.NativeLeagues(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("read due leagues: %w", err)
	}
	var next time.Time
	var failures []error
	for _, league := range leagues {
		deadline, err := w.processLeague(ctx, league.ID)
		if err != nil {
			failures = append(failures, fmt.Errorf("process due league %s: %w", league.ID, err))
			continue
		}
		if !deadline.IsZero() && (next.IsZero() || deadline.Before(next)) {
			next = deadline
		}
	}
	return next, errors.Join(failures...)
}

func (w *DueWorker) processLeague(ctx context.Context, league string) (time.Time, error) {
	ctx = commissioner.WithCredentialAdmission(ctx, dueAdmission{worker: w, system: true})
	at := w.clock.Now()
	facts := w.facts()
	// Waivers clear before bids are evaluated; a draft is closed once its clock has used the last pick.
	for _, process := range []func(context.Context, string, nativecheck.Facts) error{
		w.store.ProcessDueReserves, w.store.ProcessDueWaivers, w.store.ProcessDueBids, w.store.ProcessDueOLines, w.store.CloseFinishedDrafts,
	} {
		if err := process(ctx, league, facts); err != nil {
			return time.Time{}, fmt.Errorf("record due work: %w", err)
		}
	}
	events, err := w.store.Events(ctx, league, commissioner.Stream, 0)
	if err != nil {
		return time.Time{}, fmt.Errorf("read pending deadlines: %w", err)
	}
	records := make([]commissioner.Record, 0, len(events))
	for _, event := range events {
		if event.Payload.Commissioner == nil {
			return time.Time{}, fmt.Errorf("pending deadline has no commissioner record")
		}
		records = append(records, *event.Payload.Commissioner)
	}
	next, err := market.NextDueDeadline(league, records, at)
	if err != nil {
		return time.Time{}, fmt.Errorf("read next deadline: %w", err)
	}
	draft, err := w.store.NextDraftDeadline(ctx, league, at)
	if err != nil {
		return time.Time{}, fmt.Errorf("read next draft deadline: %w", err)
	}
	if !draft.IsZero() && (next.IsZero() || draft.Before(next)) {
		next = draft
	}
	return next, nil
}
