package history

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type olineClockAuthorizer struct{}

func (olineClockAuthorizer) Allow(_ string, principal, command string) (string, error) {
	if principal != commissioner.OLineClockPrincipal || command != commissioner.OLineAuctionClose {
		return "", fmt.Errorf("the O-Line worker may only complete reserved assignments")
	}
	return "automatic", nil
}

// ProcessDueOLines retries closed-auction assignments after cuts or other cap changes.
func (s *Store) ProcessDueOLines(ctx context.Context, league string, facts nativecheck.Facts) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("stop O-Line worker: %w", err)
	}
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return err
	}
	due, err := commissioner.OLineAssignmentsDue(league, records, s.now())
	if err != nil {
		return fmt.Errorf("read pending O-Line assignments: %w", err)
	}
	if !due {
		return nil
	}
	raw := commissioner.RawCommand{League: league, Command: commissioner.OLineAuctionClose, Expected: int64(len(records)),
		Request: fmt.Sprintf("oline-clock.%d", len(records)), Effective: "now", Visibility: "league"}
	if _, err := s.ExecuteCommissionerWithFacts(ctx, raw, commissioner.OLineClockPrincipal, olineClockAuthorizer{}, facts, time.UTC); err != nil {
		return fmt.Errorf("complete O-Line assignments: %w", err)
	}
	return nil
}

func checkPendingOLineAppend(raw commissioner.RawCommand, records []commissioner.Record, now time.Time) error {
	due, err := commissioner.OLineAssignmentsDue(raw.League, records, now)
	if err != nil {
		return fmt.Errorf("recheck reserved O-Line assignments: %w", err)
	}
	if !due || raw.Command != commissioner.OLineAuctionClose {
		return fmt.Errorf("recheck O-Line worker: %w", commissioner.Refused(fmt.Errorf("the O-Line worker may only complete a reserved assignment once there is room")))
	}
	return nil
}
