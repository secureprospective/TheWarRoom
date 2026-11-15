package leaguemigration

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// SyncLineups records historical starters through the commissioner's ordinary lineup correction.
// Source identities must already be mapped. Each lineup is atomic; retries skip unchanged sets.
// A failure returns the receipts already recorded so the next sync can resume.
func SyncLineups(ctx context.Context, ledger Ledger, league, reason, principal string, season int, lineups []commissioner.RawLineupChange) ([]commissioner.Receipt, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("write the reason for updating lineups")
	}
	records, err := ledger.Records(ctx, league)
	if err != nil {
		return nil, fmt.Errorf("read league before updating lineups: %w", err)
	}
	held, err := commissioner.ScheduleReading(league, records)
	if err != nil {
		return nil, fmt.Errorf("read lineup season: %w", err)
	}
	if held.Season != season {
		return nil, fmt.Errorf("the source lineups must match the league's current season")
	}
	commands, err := lineupSyncCommands(records, league, reason, season, lineups)
	if err != nil {
		return nil, err
	}
	if len(commands) == 0 {
		return []commissioner.Receipt{}, nil
	}
	receipts := make([]commissioner.Receipt, 0, len(commands))
	failures := []error{}
	for _, raw := range commands {
		raw.Expected = int64(len(records) + len(receipts))
		receipt, err := ledger.Execute(ctx, raw, principal)
		if err != nil {
			failures = append(failures, fmt.Errorf("record week %d lineup of %s: %w", raw.Args.Lineup.Week, raw.Args.Lineup.Franchise, err))
			continue
		}
		receipts = append(receipts, receipt)
	}
	return receipts, errors.Join(failures...)
}

func lineupSyncCommands(records []commissioner.Record, league, reason string, season int, lineups []commissioner.RawLineupChange) ([]commissioner.RawCommand, error) {
	commands := []commissioner.RawCommand{}
	seen := map[string]bool{}
	for _, lineup := range lineups {
		key := fmt.Sprintf("%s.%d", lineup.Franchise, lineup.Week)
		if lineup.Season != season || seen[key] {
			return nil, fmt.Errorf("lineups must name this season and each franchise-week once")
		}
		seen[key] = true
		raw := commissioner.RawCommand{League: league, Command: commissioner.LineupCorrect,
			Args: commissioner.RawArgs{Lineup: &lineup}, Request: fmt.Sprintf("lineup.sync.%d.%s", len(records), key),
			Reason: reason, Effective: "now", Visibility: "league"}
		if err := raw.Validate(); err != nil {
			return nil, fmt.Errorf("check lineup %s: %w", key, err)
		}
		previous, found, err := commissioner.RecordedLineupFor(records, lineup.Franchise, season, lineup.Week)
		if err != nil {
			return nil, fmt.Errorf("read lineup %s: %w", key, err)
		}
		want, have := slices.Clone(lineup.Starters), slices.Clone(previous.Starters)
		slices.Sort(want)
		slices.Sort(have)
		if found && slices.Equal(want, have) {
			continue
		}
		commands = append(commands, raw)
	}
	return commands, nil
}
