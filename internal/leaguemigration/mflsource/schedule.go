package mflsource

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
)

// Schedule maps only source identities. Empty playoff weeks and resting teams remain empty;
// played weeks are not filtered by today's date. Scores are not lineup evidence.
func Schedule(raw []leagueschedule.RawScheduleWeek, mapping map[string]string) ([]commissioner.ScheduleWeek, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	weeks := map[int]commissioner.ScheduleWeek{}
	last := 0
	for _, w := range raw {
		if err := w.Validate(); err != nil {
			return nil, fmt.Errorf("check schedule: %w", err)
		}
		n, err := strconv.Atoi(w.Week)
		if err != nil || n < 1 || n > 52 {
			return nil, fmt.Errorf("invalid schedule week %q", w.Week)
		}
		if _, ok := weeks[n]; ok {
			return nil, fmt.Errorf("repeated schedule week %d", n)
		}
		week, err := scheduleWeek(w, n, mapping)
		if err != nil {
			return nil, err
		}
		weeks[n] = week
		last = max(last, n)
	}
	out := make([]commissioner.ScheduleWeek, last)
	for i := range out {
		week, ok := weeks[i+1]
		if !ok {
			return nil, fmt.Errorf("schedule is missing week %d", i+1)
		}
		out[i] = week
	}
	args := commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: last, Fixtures: out}}
	rawCommand := commissioner.RawCommand{League: "native.check", Command: commissioner.ScheduleGenerate, Args: args, Request: "check", Reason: "check", Effective: "now", Visibility: "league"}
	if err := rawCommand.Validate(); err != nil {
		return nil, fmt.Errorf("check mapped schedule: %w", err)
	}
	return out, nil
}

func scheduleWeek(w leagueschedule.RawScheduleWeek, n int, mapping map[string]string) (commissioner.ScheduleWeek, error) {
	week := commissioner.ScheduleWeek{Week: n, Games: []commissioner.ScheduleGame{}}
	for _, matchup := range w.Matchups {
		home, away := matchup.Franchises[0], matchup.Franchises[1]
		if home.IsHome == "0" {
			home, away = away, home
		}
		h, a := mapping[home.FranchiseID], mapping[away.FranchiseID]
		if h == "" || a == "" {
			return commissioner.ScheduleWeek{}, fmt.Errorf("week %d names an unknown franchise", n)
		}
		week.Games = append(week.Games, commissioner.ScheduleGame{ID: fmt.Sprintf("week.%02d.%s.%s", n, h, a), Home: h, Away: a})
	}
	slices.SortFunc(week.Games, func(a, b commissioner.ScheduleGame) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return week, nil
}

// BackfillSchedule updates an existing league without replaying creation or replacing its roster.
// The held snapshot's IDs, not today's sorted MFL directory, own the mapping.
func BackfillSchedule(ctx context.Context, ledger leaguemigration.Ledger, league, reason, principal string, season int, raw []leagueschedule.RawScheduleWeek) ([]commissioner.Receipt, error) {
	records, err := ledger.Records(ctx, league)
	if err != nil {
		return nil, fmt.Errorf("read franchise mapping: %w", err)
	}
	assets, err := commissioner.NativeAssetsReading(league, records)
	if err != nil {
		return nil, fmt.Errorf("read starting franchises: %w", err)
	}
	if assets.Snapshot == nil {
		return nil, fmt.Errorf("record the starting franchises before updating the schedule")
	}
	mapping := map[string]string{}
	for _, f := range assets.Snapshot.Franchises {
		mapping[f.HeldID] = f.ID
	}
	weeks, err := Schedule(raw, mapping)
	if err != nil {
		return nil, err
	}
	receipts, err := leaguemigration.SyncSchedule(ctx, ledger, league, reason, principal, season, weeks)
	if err != nil {
		return nil, fmt.Errorf("backfill schedule: %w", err)
	}
	return receipts, nil
}
