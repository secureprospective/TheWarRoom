package history

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// BidEnvironmentSource says where the NFL season stands and which franchises are in the playoffs when an
// owner's bid is judged. It reads only what the app already holds, and never fetches.
type BidEnvironmentSource interface {
	BidEnvironment(ctx context.Context, league string, at time.Time) (commissioner.BidEnvironment, error)
}

// SetBidEnvironment installs the season facts bids are judged against. Without it the week is unknown, and
// a bid that depends on it is refused rather than guessed.
func (s *Store) SetBidEnvironment(src BidEnvironmentSource) { s.bidEnv = src }

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

// appendFacts is what the held files and the football environment add to the ledger when a change is checked.
type appendFacts struct {
	lineup commissioner.LineupFacts
	stat   commissioner.StatFacts
	bid    commissioner.BidEnvironment
}

// appendFactsFor gathers a change's facts before the write begins; most commands need none.
func (s *Store) appendFactsFor(ctx context.Context, raw commissioner.RawCommand) (appendFacts, error) {
	lineup, err := s.lineupFactsFor(ctx, raw)
	if err != nil {
		return appendFacts{}, err
	}
	bid, err := s.bidEnvironmentFor(ctx, raw)
	if err != nil {
		return appendFacts{}, err
	}
	facts := appendFacts{lineup: lineup, bid: bid}
	if !commissioner.IsStatCommand(raw.Command) {
		return facts, nil
	}
	week, err := s.statWeekFor(ctx, raw)
	if err != nil {
		return appendFacts{}, err
	}
	facts.stat = commissioner.StatFacts{Versions: week.Evidence.Versions()}
	return facts, nil
}
