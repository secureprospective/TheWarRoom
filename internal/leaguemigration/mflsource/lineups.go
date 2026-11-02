package mflsource

import (
	"context"
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
)

// Lineups maps played franchise-weeks, not projected or optimal starters. MFL's
// 0000 empty slot is omitted; an unknown real player refuses the import.
func Lineups(weeks []leaguefeed.WeekResult, season int, franchises, players map[string]string) ([]commissioner.RawLineupChange, error) {
	out := []commissioner.RawLineupChange{}
	for _, week := range weeks {
		sides := slices.Clone(week.Others)
		for _, game := range week.Games {
			sides = append(sides, game.Home, game.Away)
		}
		for _, side := range sides {
			if !side.Scored {
				continue
			}
			lineup := commissioner.RawLineupChange{Season: season, Week: week.Week, Franchise: franchises[side.Franchise], Starters: []string{}}
			if lineup.Franchise == "" {
				return nil, fmt.Errorf("week %d has unknown franchise %s", week.Week, side.Franchise)
			}
			for _, player := range side.Players {
				if !player.Starter || player.ID.String() == "0000" {
					continue
				}
				mapped := players[player.ID.String()]
				if mapped == "" {
					return nil, fmt.Errorf("week %d franchise %s starter %s has no player mapping", week.Week, side.Franchise, player.ID.String())
				}
				lineup.Starters = append(lineup.Starters, mapped)
			}
			slices.Sort(lineup.Starters)
			if _, err := commissioner.DecodeLineupChange(commissioner.RawArgs{Lineup: &lineup}, commissioner.LineupCorrect); err != nil {
				return nil, fmt.Errorf("week %d franchise %s: %w", week.Week, side.Franchise, err)
			}
			out = append(out, lineup)
		}
	}
	return out, nil
}

// BackfillLineups uses the held franchise identities and the app's player directory.
// Player IDs are shared today; that identity mapping belongs only to this importer.
func BackfillLineups(ctx context.Context, ledger leaguemigration.Ledger, league, reason, principal string, season int, weeks []leaguefeed.WeekResult, heldPlayers map[string]string) ([]commissioner.Receipt, error) {
	records, err := ledger.Records(ctx, league)
	if err != nil {
		return nil, fmt.Errorf("read lineup identities: %w", err)
	}
	assets, err := commissioner.NativeAssetsReading(league, records)
	if err != nil {
		return nil, fmt.Errorf("read lineup franchises: %w", err)
	}
	if assets.Snapshot == nil {
		return nil, fmt.Errorf("record starting franchises before importing lineups")
	}
	franchises, players := map[string]string{}, map[string]string{}
	for _, f := range assets.Snapshot.Franchises {
		franchises[f.HeldID] = f.ID
	}
	for id := range heldPlayers {
		players[id] = id
	}
	lineups, err := Lineups(weeks, season, franchises, players)
	if err != nil {
		return nil, err
	}
	receipts, err := leaguemigration.SyncLineups(ctx, ledger, league, reason, principal, season, lineups)
	if err != nil {
		return receipts, fmt.Errorf("backfill played lineups: %w", err)
	}
	return receipts, nil
}

// LiveWeekResult turns MFL's live-scoring export for one week into the week-result shape the
// lineup import reads. A franchise that MFL lists with starters has set its lineup, whether or
// not the week has been scored, so each such side counts as recorded; a franchise with no
// starters is left out, so an unset lineup stays a missed lineup rather than an import refusal.
func LiveWeekResult(live leaguefeed.Lineups) leaguefeed.WeekResult {
	sides := map[string]leaguefeed.Side{}
	for _, f := range live.Franchises {
		if len(f.Starters) == 0 {
			continue
		}
		side := leaguefeed.Side{Franchise: f.Franchise, Scored: true, Players: []leaguefeed.WeekPlayer{}}
		for _, id := range f.Starters {
			side.Players = append(side.Players, leaguefeed.WeekPlayer{ID: id, Starter: true})
		}
		for _, id := range f.NonStarters {
			side.Players = append(side.Players, leaguefeed.WeekPlayer{ID: id})
		}
		sides[f.Franchise] = side
	}
	out := leaguefeed.WeekResult{Week: live.Week}
	paired := map[string]bool{}
	for _, m := range live.Matchups {
		home, okHome := sides[m.Home]
		away, okAway := sides[m.Away]
		paired[m.Home], paired[m.Away] = true, true
		if !okHome && !okAway {
			continue
		}
		if !okHome || !okAway {
			// One side set a lineup and the other did not; the set one is still recorded.
			for _, side := range []leaguefeed.Side{home, away} {
				if side.Franchise != "" {
					out.Others = append(out.Others, side)
				}
			}
			continue
		}
		out.Games = append(out.Games, leaguefeed.Game{Regular: true, Home: home, Away: away})
	}
	for _, f := range live.Franchises {
		if side, ok := sides[f.Franchise]; ok && !paired[f.Franchise] {
			out.Others = append(out.Others, side)
		}
	}
	return out
}

// BackfillLiveLineups records the starters MFL holds for one week in progress, through the same
// ordinary lineup correction the played-week import uses.
func BackfillLiveLineups(ctx context.Context, ledger leaguemigration.Ledger, league, reason, principal string,
	season int, live leaguefeed.Lineups, heldPlayers map[string]string) ([]commissioner.Receipt, error) {
	week := LiveWeekResult(live)
	if len(week.Games) == 0 && len(week.Others) == 0 {
		return []commissioner.Receipt{}, nil
	}
	return BackfillLineups(ctx, ledger, league, reason, principal, season, []leaguefeed.WeekResult{week}, heldPlayers)
}
