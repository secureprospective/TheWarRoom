package history

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// bidClockPrincipal is who the ledger records for an automatic bid award.
const bidClockPrincipal = "native.bid.clock"

// clockBidAuthorizer cannot place a bid, change a rule or override an award.
type clockBidAuthorizer struct{}

func (clockBidAuthorizer) Allow(_ string, principal, command string) (string, error) {
	if principal != bidClockPrincipal || command != commissioner.BidsProcess {
		return "", fmt.Errorf("the bid clock can only award bids that have stood their full time")
	}
	return "automatic", nil
}

// ProcessDueBids records automatic awards through the same versioned transaction as bids.process.
// The transaction rechecks the clock's candidates against its current ledger before appending.
func (s *Store) ProcessDueBids(ctx context.Context, league string, facts nativecheck.Facts) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("stop bid clock: %w", err)
	}
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return err
	}
	due, err := market.DueBids(league, records, s.now(), facts)
	if err != nil {
		return fmt.Errorf("read due bids: %w", err)
	}
	for _, player := range due {
		records, err = s.nativeRecords(ctx, league)
		if err != nil {
			return err
		}
		raw := commissioner.RawCommand{League: league, Command: commissioner.BidsProcess,
			Args:     commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: player}},
			Expected: int64(len(records)), Request: fmt.Sprintf("bid-clock.%s.%d", player, len(records)),
			Effective: "now", Visibility: "league"}
		if _, err := s.ExecuteCommissionerWithFacts(ctx, raw, bidClockPrincipal, clockBidAuthorizer{}, facts, time.UTC); err != nil {
			return fmt.Errorf("award due bid: %w", err)
		}
	}
	return nil
}
