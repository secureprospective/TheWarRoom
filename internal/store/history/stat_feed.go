package history

import (
	"context"
	"database/sql"
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
	weeks, err := (seasonStatFeed{store: s, season: season}).Weeks(ctx)
	if err != nil {
		return fmt.Errorf("read feed revisions: %w", err)
	}
	facts := map[string]commissioner.StatFacts{}
	for _, week := range weeks {
		facts[fmt.Sprintf("%s/%d", week.Player, week.Week)] = commissioner.StatFacts{Versions: week.Evidence.Versions(), OLine: week.Evidence.OLine}
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("begin feed revisions: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	records, err := s.transactionRecords(ctx, tx, league)
	if err != nil {
		return err
	}
	if !nativeSeeded(records) {
		return nil
	}
	moves, err := commissioner.FeedStatMoves(league, records, weeks, season)
	if err != nil {
		return fmt.Errorf("plan feed revisions: %w", err)
	}
	// Feed approvals cannot change permissions, rules, rosters or manual pins. Check the
	// batch against the locked starting snapshot; appended events still carry sequential revisions.
	initial := records
	for _, raw := range moves {
		raw.Expected = int64(len(records))
		player, _, week := raw.Args.Stat.Scope()
		record, err := s.appendFeedMove(ctx, tx, raw, appendFacts{stat: facts[fmt.Sprintf("%s/%d", player, week)]}, initial, records)
		if err != nil {
			return err
		}
		if record.Receipt.Entry != "" {
			records = append(records, record)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit feed revisions: %w", err)
	}
	if len(moves) != 0 {
		s.signalCommissionerCommit()
	}
	return nil
}

func (s *Store) appendFeedMove(ctx context.Context, tx *sql.Tx, raw commissioner.RawCommand,
	gathered appendFacts, initial, records []commissioner.Record) (commissioner.Record, error) {
	prepared, canonical, err := preparedCommand(raw, commissioner.StatFeedPrincipal, true)
	if err != nil {
		return commissioner.Record{}, err
	}
	now := s.now().UTC()
	check := prepared
	check.Expected = int64(len(initial))
	role, retry, err := s.checkCommissionerCommand(ctx, check, commissioner.StatFeedPrincipal, statFeedAuthorizer{}, initial, gathered, now)
	if err != nil {
		return commissioner.Record{}, fmt.Errorf("check counted feed revision: %w", err)
	}
	if retry != nil {
		return commissioner.Record{}, nil
	}
	receipt, err := s.appendCommissioner(ctx, tx, appendRequest{raw: prepared, principal: commissioner.StatFeedPrincipal, role: role, canonical: canonical, at: now, records: records})
	if err != nil {
		return commissioner.Record{}, fmt.Errorf("record counted feed revision: %w", err)
	}
	return commissioner.Record{Command: prepared, Principal: commissioner.StatFeedPrincipal, Receipt: receipt}, nil
}
