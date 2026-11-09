package trades

import (
	"context"
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// ReadLineupDelta adapts the held local reference to the same D11/lineup read as app leagues.
func ReadLineupDelta(ctx context.Context, config league.RawConfig, roster state.Reader, season, week int, trade history.TradeSides,
	read func(context.Context, string, int, int, history.TradeSides, commissioner.NativeBase) (history.NativeLineupDelta, error)) (history.NativeLineupDelta, error) {
	if roster == nil || read == nil {
		return history.NativeLineupDelta{}, fmt.Errorf("lineup fit: the held roster and shared read must be open")
	}
	franchises := roster.Franchises()
	if !slices.Contains(franchises, trade.Franchise) || !slices.Contains(franchises, trade.Partner) {
		return history.NativeLineupDelta{}, fmt.Errorf("lineup fit: choose two held franchises")
	}
	rules, problems, _ := mflsource.Settings(config, leaguerules.RuleSet{}, "")
	if len(problems) > 0 {
		return history.NativeLineupDelta{}, fmt.Errorf("lineup fit: held league settings cannot be read: %v", problems)
	}
	base := commissioner.NativeBase{Rules: rules}
	base.Assets.Seeded = true
	base.Assets.Snapshot = &commissioner.RawNativeSnapshot{}
	for _, fid := range franchises {
		players, ok := roster.Roster(fid)
		if !ok {
			return history.NativeLineupDelta{}, fmt.Errorf("lineup fit: held roster %s is unavailable", fid)
		}
		for _, p := range players {
			base.Assets.Snapshot.Players = append(base.Assets.Snapshot.Players, commissioner.NativePlayer{ID: p.MFLID, Franchise: fid, Status: p.RosterStatus})
		}
	}
	got, err := read(ctx, "", season, week, trade, base)
	if err != nil {
		return got, fmt.Errorf("lineup fit: %w", err)
	}
	return got, nil
}
