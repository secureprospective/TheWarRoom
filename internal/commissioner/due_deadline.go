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
	assets, state, err := nativeAssetsState(league, records, at)
	if err != nil {
		return time.Time{}, err
	}
	if !assets.Seeded {
		return time.Time{}, nil
	}
	rules, err := RulesNow(records, at)
	if err != nil {
		return time.Time{}, err
	}
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
