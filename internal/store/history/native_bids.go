package history

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// BidEnvironmentSource says where the NFL season stands and which franchises are in the playoffs when an
// owner's bid is judged. It reads only what the app already holds, and never fetches.
type BidEnvironmentSource interface {
	BidEnvironment(ctx context.Context, league string, at time.Time) (commissioner.BidEnvironment, error)
}

// SetBidEnvironment installs the season facts bids are judged against. Without it the week is unknown, and
// a bid that depends on it is refused rather than guessed.
func (s *Store) SetBidEnvironment(src BidEnvironmentSource) { s.bidEnv = src }

// environmentFor gathers the held football facts a command is judged against before the write begins: a
// lineup command's week, an owner bid's season. Any other command needs none.
func (s *Store) environmentFor(ctx context.Context, raw commissioner.RawCommand) (
	commissioner.LineupFacts, commissioner.BidEnvironment, error) {
	lineup, err := s.lineupFactsFor(ctx, raw)
	if err != nil {
		return commissioner.LineupFacts{}, commissioner.BidEnvironment{}, err
	}
	bid, err := s.bidEnvironmentFor(ctx, raw)
	return lineup, bid, err
}

// bidEnvironmentFor gathers an owner bid's season facts before the write begins; nothing else needs them.
func (s *Store) bidEnvironmentFor(ctx context.Context, raw commissioner.RawCommand) (commissioner.BidEnvironment, error) {
	if commissioner.OwnerBidAction(raw.Command) == "" || s.bidEnv == nil {
		return commissioner.BidEnvironment{}, nil
	}
	env, err := s.bidEnv.BidEnvironment(ctx, raw.League, s.now())
	if err != nil {
		return commissioner.BidEnvironment{}, fmt.Errorf("read the NFL week for this bid: %w", err)
	}
	return env, nil
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
	return reading, nil
}
