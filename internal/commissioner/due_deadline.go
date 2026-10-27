package commissioner

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// NextDueDeadline returns the nearest future auction or waiver close. Already-closed,
// blocked awards are retried on reconciliation, not in a busy loop at their old deadline.
func NextDueDeadline(league string, records []Record, at time.Time) (time.Time, error) {
	f, err := marketFrame(league, records, at, nil, nil, true)
	if err != nil {
		return time.Time{}, err
	}
	assets, state := f.Assets, f.Book
	if !assets.Seeded {
		return time.Time{}, nil
	}
	if err := f.readRules(); err != nil {
		return time.Time{}, err
	}
	rules := f.Rules
	var next time.Time
	consider := func(deadline time.Time) {
		next = nearerFutureDeadline(next, deadline, at)
	}
	if len(state.Windows) > 0 {
		policy, err := nativecheck.NewWaiverPolicy(rules.Waivers)
		if err != nil {
			return time.Time{}, fmt.Errorf("date pending waivers: %w", err)
		}
		for _, window := range state.Windows {
			if window.Processed {
				continue
			}
			closeAt, err := policy.Close(window.CutAt)
			if err != nil {
				return time.Time{}, fmt.Errorf("date pending waiver: %w", err)
			}
			consider(closeAt)
		}
	}
	stand := bidStandFor(rules)
	if stand.State == leaguerules.IsKnown && stand.Value > 0 {
		for i := range state.Bids.Auctions {
			auction := &state.Bids.Auctions[i]
			if !auction.Decided {
				consider(auction.standingAt().Add(stand.Value))
			}
		}
	}
	return next, nil
}

func nearerFutureDeadline(next, deadline, at time.Time) time.Time {
	if deadline.After(at) && (next.IsZero() || deadline.Before(next)) {
		return deadline
	}
	return next
}
