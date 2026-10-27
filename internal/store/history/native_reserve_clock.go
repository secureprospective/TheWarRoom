package history

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type clockReserveAuthorizer struct{}

func (clockReserveAuthorizer) Allow(_ string, principal, command string) (string, error) {
	if principal != commissioner.ReserveClockPrincipal || command != commissioner.RosterReserve {
		return "", fmt.Errorf("the reserve clock can only return ineligible reserve players")
	}
	return "automatic", nil
}

// ProcessDueReserves records each return through the versioned ledger, with transaction-time rechecking.
func (s *Store) ProcessDueReserves(ctx context.Context, league string, facts nativecheck.Facts) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("stop reserve clock: %w", err)
	}
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return err
	}
	due, err := commissioner.DueReserves(league, records, s.now(), facts)
	if err != nil {
		return fmt.Errorf("read due reserves: %w", err)
	}
	for _, move := range due {
		records, err = s.nativeRecords(ctx, league)
		if err != nil {
			return err
		}
		raw := commissioner.RawCommand{League: league, Command: commissioner.RosterReserve,
			Args: commissioner.RawArgs{Waiver: &move}, Expected: int64(len(records)),
			Request: fmt.Sprintf("reserve-clock.%s.%d", move.Player, len(records)), Effective: "now", Visibility: "league"}
		if _, err := s.ExecuteCommissionerWithFacts(ctx, raw, commissioner.ReserveClockPrincipal, clockReserveAuthorizer{}, facts, time.UTC); err != nil {
			return fmt.Errorf("return due reserve: %w", err)
		}
	}
	return nil
}
