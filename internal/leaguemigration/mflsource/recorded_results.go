package mflsource

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// RecordedResults captures only scored weeks. A new import starts app scoring at
// the first unfinished week; refreshes retain the already recorded boundary.
func RecordedResults(weeks []leaguefeed.WeekResult, season, cutover int, mapping map[string]string) (*commissioner.RawRecordedResults, error) {
	if cutover == 0 {
		cutover = 1
		for _, week := range weeks {
			if !completedWeek(week) {
				continue
			}
			cutover = max(cutover, week.Week+1)
		}
	}
	out := &commissioner.RawRecordedResults{Season: season, CutoverWeek: cutover, Teams: []commissioner.RecordedTeamResult{}}
	for _, week := range weeks {
		if week.Week >= cutover || !completedWeek(week) {
			continue
		}
		sides := slices.Clone(week.Others)
		for _, game := range week.Games {
			if err := checkRecordedGame(game); err != nil {
				return nil, err
			}
			sides = append(sides, game.Home, game.Away)
		}
		for _, side := range sides {
			team, err := recordedSide(side, week.Week, mapping)
			if err != nil {
				return nil, err
			}
			out.Teams = append(out.Teams, team)
		}
	}
	slices.SortFunc(out.Teams, func(a, b commissioner.RecordedTeamResult) int {
		if a.Week != b.Week {
			return a.Week - b.Week
		}
		if a.Franchise < b.Franchise {
			return -1
		}
		if a.Franchise > b.Franchise {
			return 1
		}
		return 0
	})
	if err := out.Validate(); err != nil {
		return nil, fmt.Errorf("check recorded results: %w", err)
	}
	return out, nil
}

func completedWeek(week leaguefeed.WeekResult) bool {
	if len(week.Games)+len(week.Others) == 0 {
		return false
	}
	for _, game := range week.Games {
		if !game.Home.Scored || !game.Away.Scored {
			return false
		}
	}
	for _, side := range week.Others {
		if !side.Scored {
			return false
		}
	}
	return true
}

func checkRecordedGame(game leaguefeed.Game) error {
	home, away := game.Home.Result, game.Away.Result
	switch home + away {
	case "WL", "LW", "TT":
		return nil
	}
	return fmt.Errorf("source game needs complementary recorded W/L/T results")
}

// ImportWeek names the week in play at an instant (the offseason is week 1); ok is false when unknown.
type ImportWeek func(ctx context.Context, season int, at time.Time) (week int, ok bool)

// BackfillRecordedResults uses the existing schedule ledger evidence path without
// replaying starting rosters, and never changes a boundary already captured. For a
// league imported by an older build, the boundary is the week in play when the league
// was imported (never later), because the app has scored every week since.
func BackfillRecordedResults(ctx context.Context, ledger leaguemigration.Ledger, league, reason, principal string,
	season int, weeks []leaguefeed.WeekResult, importWeek ImportWeek) ([]commissioner.Receipt, error) {
	records, err := ledger.Records(ctx, league)
	if err != nil {
		return nil, fmt.Errorf("read recorded result identities: %w", err)
	}
	assets, err := commissioner.NativeAssetsReading(league, records)
	if err != nil {
		return nil, fmt.Errorf("read recorded result franchises: %w", err)
	}
	if assets.Snapshot == nil {
		return nil, fmt.Errorf("record starting franchises before results")
	}
	mapping := map[string]string{}
	for _, f := range assets.Snapshot.Franchises {
		mapping[f.HeldID] = f.ID
	}
	boundary := 0
	if held := commissioner.RecordedResults(records, season); held != nil {
		boundary = held.CutoverWeek
	} else {
		if len(records) == 0 || importWeek == nil {
			return nil, fmt.Errorf("the import week is not known, so recorded results wait")
		}
		at, known := importWeek(ctx, season, records[0].Receipt.RecordedAt)
		if !known {
			return nil, fmt.Errorf("the import week is not known, so recorded results wait")
		}
		auto, err := RecordedResults(weeks, season, 0, mapping)
		if err != nil {
			return nil, err
		}
		boundary = max(1, min(auto.CutoverWeek, at))
	}
	captured, err := RecordedResults(weeks, season, boundary, mapping)
	if err != nil {
		return nil, err
	}
	schedule, err := commissioner.ScheduleReading(league, records)
	if err != nil {
		return nil, fmt.Errorf("read schedule for recorded results: %w", err)
	}
	receipts, err := leaguemigration.SyncSchedule(ctx, ledger, league, reason, principal, season, schedule.Weeks, captured)
	if err != nil {
		return receipts, fmt.Errorf("capture recorded results: %w", err)
	}
	return receipts, nil
}

func recordedSide(side leaguefeed.Side, week int, mapping map[string]string) (commissioner.RecordedTeamResult, error) {
	id := mapping[side.Franchise]
	if id == "" {
		return commissioner.RecordedTeamResult{}, fmt.Errorf("recorded week %d names unknown franchise %s", week, side.Franchise)
	}
	// The feed uses hundredths; exact rule decimals use ten-thousandths.
	if side.Points > (1<<63-1)/100 || side.Points < (-1<<63)/100 {
		return commissioner.RecordedTeamResult{}, fmt.Errorf("recorded score is outside the supported range")
	}
	return commissioner.RecordedTeamResult{Franchise: id, Week: week,
		Points: leaguerules.Decimal(side.Points) * 100, Result: string(side.Result)}, nil
}
