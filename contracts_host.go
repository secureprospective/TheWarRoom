package main

import (
	"context"
	"fmt"
	"strconv"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// contractsHost is the contracts API's Host: core's shared state, plus unbound adapters over the
// features still at root that contracts reads (the season feeds, the personal roster check, the
// held archive, and the moves hand-off target and URL check). It is never bound.
type contractsHost struct {
	*core.State
	app *App
}

// contracts is the contracts API over this App; root callers use it, never an App wrapper.
func (a *App) contracts() *contractsapp.API {
	return contractsapp.New(contractsHost{State: a.core, app: a})
}

func (h contractsHost) ContractsSeason() (snapshot.Sourced[leaguefeed.Lineups], error) {
	season, err := h.app.TargetSeason()
	return season.Lineups, err
}

func (h contractsHost) RosterSnapshot(mine string) (snapshot.Snapshot, error) {
	return h.app.personalSnapshot(mine)
}

// RosterFeeds reads the season and last-week playerScores and the bye weeks from the held archive,
// in that order.
func (h contractsHost) RosterFeeds(ctx context.Context, lastWeek int) (contractsapp.Scores, contractsapp.Scores, contractsapp.Byes) {
	season := scorePeriod(ctx, h.app, "YTD")
	week := scorePeriod(ctx, h.app, strconv.Itoa(lastWeek))
	byes := archivedOnly(ctx, h.app, nflschedule.ByeExport, true, nflschedule.ParseByes)
	return contractsapp.Scores{Feed: season.FeedState(), Rows: season.Value},
		contractsapp.Scores{Feed: week.FeedState(), Rows: week.Value}, contractsapp.Byes{Feed: byes.FeedState(), Rows: byes.Value}
}

// ReleaseTarget is MFL's drop page, or an unmapped target until the league host is known.
func (h contractsHost) ReleaseTarget() envelope.Target {
	target, _ := h.app.rosterMoveTarget("257")
	return target
}

func (h contractsHost) ValidateMoveURL(target string) error { return validateMoveURL(target) }

func scorePeriod(ctx context.Context, a *App, period string) heldSeasonFeed[[]ingestion.RawPlayerScore] {
	return archivedOnly(ctx, a, "playerScores", false, func(body []byte) ([]ingestion.RawPlayerScore, error) {
		rows, err := ingestion.ParsePlayerScores(body)
		if err != nil {
			return nil, fmt.Errorf("player scores: parse: %w", err)
		}
		out := []ingestion.RawPlayerScore{}
		for _, r := range rows {
			if r.Week == period {
				out = append(out, r)
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("player scores: period %s not held", period)
		}
		return out, nil
	})
}
