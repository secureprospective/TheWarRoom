package history

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// StandingsOrder reads native recorded priority, never an imported league's standings or inferred rank.
func (s *Store) StandingsOrder(ctx context.Context, league string, season int, at time.Time) ([]string, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return nil, err
	}
	order, err := commissioner.RecordedStandingsOrder(league, records, season, at)
	if err != nil {
		return nil, fmt.Errorf("read native standings order: %w", err)
	}
	return order, nil
}

func (s *Store) NativeWaivers(ctx context.Context, league, franchise, principal string, at time.Time,
	zone *time.Location, facts nativecheck.Facts) (commissioner.NativeWaivers, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativeWaivers{}, err
	}
	reading, err := commissioner.NativeWaiversReading(league, franchise, principal, records, at, zone, facts)
	if err != nil {
		return commissioner.NativeWaivers{}, fmt.Errorf("read native waivers: %w", err)
	}
	return reading, nil
}

// clockWaiverAuthorizer is deliberately narrower than any human role; it cannot override an award.
type clockWaiverAuthorizer struct{}

func (clockWaiverAuthorizer) Allow(_ string, principal, command string) (string, error) {
	if principal != "native.waiver.clock" || command != commissioner.WaiverProcess {
		return "", fmt.Errorf("the waiver clock can only process closed windows")
	}
	return "automatic", nil
}

// ProcessDueWaivers uses the same append transaction and deterministic request keys as manual processing.
func (s *Store) ProcessDueWaivers(ctx context.Context, league string, facts nativecheck.Facts) error {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return err
	}
	due, err := commissioner.DueWaiverWindows(league, records, s.now())
	if err != nil {
		return fmt.Errorf("read due waivers: %w", err)
	}
	for _, window := range due {
		records, err = s.nativeRecords(ctx, league)
		if err != nil {
			return err
		}
		raw := commissioner.RawCommand{League: league, Command: commissioner.WaiverProcess,
			Args:     commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Window: window}},
			Expected: int64(len(records)), Request: fmt.Sprintf("waiver-clock.%s.%d", window, len(records)),
			Effective: "now", Visibility: "league"}
		if _, err := s.ExecuteCommissionerWithFacts(ctx, raw, "native.waiver.clock", clockWaiverAuthorizer{}, facts, time.UTC); err != nil {
			return fmt.Errorf("process due waiver: %w", err)
		}
	}
	return nil
}

// NativeBids reads the open and decided bids as one franchise's owner is allowed to see them.
func (s *Store) NativeBids(ctx context.Context, league, franchise, principal string, at time.Time,
	zone *time.Location, facts nativecheck.Facts) (commissioner.NativeBids, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return commissioner.NativeBids{}, err
	}
	reading, err := commissioner.NativeBidsReading(league, franchise, principal, records, at, zone, facts)
	if err != nil {
		return commissioner.NativeBids{}, fmt.Errorf("read native bids: %w", err)
	}
	if s.bidEnv != nil {
		env, err := s.bidEnv.BidEnvironment(ctx, league, at)
		if err != nil {
			return commissioner.NativeBids{}, fmt.Errorf("read bidding season: %w", err)
		}
		if env.Season.Offseason {
			reading.Notes = append(reading.Notes, "It is the offseason: the week limits on multi-year and playoff-only bidding do not apply.")
		}
	}
	return reading, nil
}
