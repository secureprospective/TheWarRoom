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
	check, err := s.feedBatchCheck(ctx, moves, records, facts)
	if err != nil {
		return err
	}
	initial := records
	for _, raw := range commissioner.BatchFeedMoves(moves) {
		raw.Expected = int64(len(records))
		record, err := s.appendFeedMove(ctx, tx, raw, facts, check, initial, records)
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

// Feed approvals cannot change permissions, rules, rosters or manual pins. Validate those
// invariant checks once under the write lock, then check each scope, evidence and retry separately.
func (s *Store) feedBatchCheck(ctx context.Context, moves []commissioner.RawCommand, records []commissioner.Record,
	facts map[string]commissioner.StatFacts) (func(commissioner.RawCommand, commissioner.StatFacts) error, error) {
	if len(moves) == 0 {
		return nil, nil
	}
	raw := moves[0]
	raw.Expected = int64(len(records))
	player, _, week := raw.Args.Stat.Scope()
	if _, _, err := s.checkCommissionerCommand(ctx, raw, commissioner.StatFeedPrincipal, statFeedAuthorizer{},
		records, appendFacts{stat: facts[fmt.Sprintf("%s/%d", player, week)]}, s.now().UTC()); err != nil {
		return nil, fmt.Errorf("check feed batch: %w", err)
	}
	return commissioner.FeedStatAppendChecker(records) //nolint:wrapcheck // The checker supplies the stat refusal.
}

func (s *Store) appendFeedMove(ctx context.Context, tx *sql.Tx, raw commissioner.RawCommand,
	facts map[string]commissioner.StatFacts, check func(commissioner.RawCommand, commissioner.StatFacts) error,
	initial, records []commissioner.Record) (commissioner.Record, error) {
	prepared, canonical, err := preparedCommand(raw, commissioner.StatFeedPrincipal, true)
	if err != nil {
		return commissioner.Record{}, err
	}
	now := s.now().UTC()
	if _, retry, err := retriedReceipt(ctx, prepared, commissioner.StatFeedPrincipal, initial); err != nil {
		return commissioner.Record{}, err
	} else if retry {
		return commissioner.Record{}, nil
	}
	for _, item := range prepared.Args.Stat.Items {
		line := commissioner.FeedLine(prepared, item)
		player, _, week := line.Args.Stat.Scope()
		if err := check(line, facts[fmt.Sprintf("%s/%d", player, week)]); err != nil {
			return commissioner.Record{}, fmt.Errorf("check counted feed revision: %w", err)
		}
	}
	receipt, err := s.appendCommissioner(ctx, tx, appendRequest{raw: prepared, principal: commissioner.StatFeedPrincipal, role: "automatic", canonical: canonical, at: now, records: records})
	if err != nil {
		return commissioner.Record{}, fmt.Errorf("record counted feed revision: %w", err)
	}
	return commissioner.Record{Command: prepared, Canonical: canonical, Principal: commissioner.StatFeedPrincipal, Receipt: receipt}, nil
}
