package leaguemigration

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// ScheduleCommands uses the same draft/publication path as a commissioner-chosen schedule.
func ScheduleCommands(league, reason string, weeks []commissioner.ScheduleWeek, revision int64) []commissioner.RawCommand {
	if len(weeks) == 0 {
		return nil
	}
	request := fmt.Sprintf("schedule.sync.%d", revision)
	return []commissioner.RawCommand{
		{League: league, Command: commissioner.ScheduleGenerate, Args: commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: len(weeks), Fixtures: weeks}}, Expected: revision, Request: request + ".draft", Reason: reason, Effective: "now", Visibility: "league"},
		{League: league, Command: commissioner.SchedulePublish, Args: commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}, Request: request + ".publish", Reason: reason, Effective: "now", Visibility: "league"},
	}
}

// SyncSchedule publishes only a changed schedule, atomically and with an optimistic revision fence.
// Callers supply already mapped app franchise identities, never source IDs.
func SyncSchedule(ctx context.Context, ledger Ledger, league, reason, principal string, season int, weeks []commissioner.ScheduleWeek) ([]commissioner.Receipt, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("write the reason for updating the schedule")
	}
	records, err := ledger.Records(ctx, league)
	if err != nil {
		return nil, fmt.Errorf("read league before updating schedule: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("create the league before updating its schedule")
	}
	held, err := commissioner.ScheduleReading(league, records)
	if err != nil {
		return nil, fmt.Errorf("read published schedule: %w", err)
	}
	if held.Season != season {
		return nil, fmt.Errorf("the source schedule must match the league's current season")
	}
	if len(weeks) == 0 || reflect.DeepEqual(held.Weeks, weeks) {
		return []commissioner.Receipt{}, nil
	}
	receipts, err := ledger.Batch(ctx, ScheduleCommands(league, reason, weeks, int64(len(records))), principal)
	if err != nil {
		return nil, fmt.Errorf("publish schedule (nothing recorded): %w", err)
	}
	return receipts, nil
}

func checkScheduleMapping(records []commissioner.Record, mapping map[string]string) error {
	assets, err := commissioner.NativeAssetsReading(records[0].Command.League, records)
	if err != nil {
		return fmt.Errorf("read existing franchises: %w", err)
	}
	if assets.Snapshot == nil {
		return fmt.Errorf("record starting franchises before importing the schedule")
	}
	for _, f := range assets.Snapshot.Franchises {
		if mapping[f.HeldID] != f.ID {
			return fmt.Errorf("the source franchise directory changed; backfill using the league's recorded identities")
		}
	}
	return nil
}
