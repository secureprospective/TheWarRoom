package planwatch

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
)

type MovePolling struct {
	Now          func() time.Time
	Timer        func(time.Duration) (<-chan time.Time, func())
	Wake         <-chan struct{}
	Timeout      time.Duration
	Check        func(context.Context) error
	Log          func(string)
	Lock, Unlock func()
	Awaiting     func(context.Context) ([]envelope.Envelope, error)
}

// A restart checks persisted work once; wakes and fast polling share the same rate limit.
func (s MovePolling) Run(parent context.Context) {
	var last, fastUntil time.Time
	pending := true
	for parent.Err() == nil {
		now := s.Now()
		var tick <-chan time.Time
		stop := func() {}
		if pending || now.Before(fastUntil) {
			delay := max(time.Duration(0), last.Add(time.Minute).Sub(now))
			tick, stop = s.Timer(delay)
		}
		select {
		case <-parent.Done():
			stop()
			return
		case <-s.Wake:
			stop()
			pending = true
		case <-tick:
			stop()
			now = s.Now()
			if !pending && !now.Before(fastUntil) {
				continue
			}
			// A queued wake belongs to this pass, not to another pass a minute later.
			select {
			case <-s.Wake:
			default:
			}
			last, pending = now, false
			ctx, cancel := context.WithTimeout(parent, s.Timeout)
			if err := s.Check(ctx); err != nil && parent.Err() == nil {
				s.Log(err.Error())
			}
			fastUntil = s.FastUntil(ctx)
			cancel()
		}
	}
}

func (s MovePolling) FastUntil(ctx context.Context) time.Time {
	s.Lock()
	defer s.Unlock()
	entries, err := s.Awaiting(ctx)
	if err != nil {
		if ctx.Err() == nil {
			s.Log(fmt.Sprintf("watch moves: fast mode: %v", err))
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
