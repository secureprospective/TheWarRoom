package composition

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
)

func newMovesTimer(delay time.Duration) (<-chan time.Time, func()) {
	timer := time.NewTimer(delay)
	return timer.C, func() { timer.Stop() }
}

// A restart checks persisted work once; wakes and fast polling share the same rate limit.
func (a *App) watchMoves(parent context.Context) {
	var last, fastUntil time.Time
	pending := true
	for parent.Err() == nil {
		now := a.core.MovesNow()
		var tick <-chan time.Time
		stop := func() {}
		if pending || now.Before(fastUntil) {
			delay := max(time.Duration(0), last.Add(time.Minute).Sub(now))
			tick, stop = a.core.MovesTimer(delay)
		}
		select {
		case <-parent.Done():
			stop()
			return
		case <-a.core.MovesWake:
			stop()
			pending = true
		case <-tick:
			stop()
			now = a.core.MovesNow()
			if !pending && !now.Before(fastUntil) {
				continue
			}
			// A queued wake belongs to this pass, not to another pass a minute later.
			select {
			case <-a.core.MovesWake:
			default:
			}
			last, pending = now, false
			ctx, cancel := context.WithTimeout(parent, refreshTimeout)
			if err := a.checkMoves(ctx); err != nil && parent.Err() == nil {
				a.core.MovesLog(err.Error())
			}
			fastUntil = a.movesFastUntil(ctx)
			cancel()
		}
	}
}

func (a *App) movesFastUntil(ctx context.Context) time.Time {
	a.core.MovesMu.Lock()
	defer a.core.MovesMu.Unlock()
	entries, err := a.core.Moves.Awaiting(ctx)
	if err != nil {
		if ctx.Err() == nil {
			a.core.MovesLog(fmt.Sprintf("watch moves: fast mode: %v", err))
		}
		return time.Time{}
	}
	var latest time.Time
	for _, e := range entries {
		for _, entry := range e.Receipt().Audit {
			if entry.Event == envelope.HandOff && entry.At.After(latest) {
				latest = entry.At
			}
		}
	}
	if latest.IsZero() {
		return latest
	}
	return latest.Add(30 * time.Minute)
}
