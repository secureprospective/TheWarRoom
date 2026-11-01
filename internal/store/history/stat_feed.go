package history

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

type statFeedAuthorizer struct{}

func (statFeedAuthorizer) Allow(_, principal, command string) (string, error) {
	if principal != commissioner.StatFeedPrincipal || command != commissioner.StatApprove {
		return "", fmt.Errorf("the stat feed may only count feed revisions")
	}
	return "automatic", nil
}

// ProcessStatFeed records each automatic counted-revision move through the existing stat approval flow.
func (s *Store) ProcessStatFeed(ctx context.Context, league string, season int) error {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return err
	}
	if !nativeSeeded(records) {
		return nil
	}
	weeks, err := (seasonStatFeed{store: s, season: season}).Weeks(ctx)
	if err != nil {
		return fmt.Errorf("read feed revisions: %w", err)
	}
	moves, err := commissioner.FeedStatMoves(league, records, weeks, season)
	if err != nil {
		return fmt.Errorf("plan feed revisions: %w", err)
	}
	for _, raw := range moves {
		raw.Expected = int64(len(records))
		receipt, err := s.ExecuteCommissioner(ctx, raw, commissioner.StatFeedPrincipal, statFeedAuthorizer{})
		if err != nil {
			return fmt.Errorf("record counted feed revision: %w", err)
		}
		records = append(records, commissioner.Record{Command: raw, Principal: commissioner.StatFeedPrincipal, Receipt: receipt})
	}
	return nil
}
