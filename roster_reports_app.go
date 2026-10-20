package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/lineup"
)

func (a *App) TargetRosterReports(mine string) ([]Report, error) {
	if err := a.ready(); err != nil {
		return nil, fmt.Errorf("roster reports: startup: %w", err)
	}
	snap, err := a.personalSnapshot(mine)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, m4Timeout)
	defer cancel()
	a.weekMu.Lock()
	lastWeek := a.week.Number - 1
	a.weekMu.Unlock()
	season := scorePeriod(ctx, a, "YTD")
	week := scorePeriod(ctx, a, strconv.Itoa(lastWeek))
	byes := archivedOnly(ctx, a, nflschedule.ByeExport, true, nflschedule.ParseByes)
	rules, err := lineup.ParseRules(a.rulebook.ActiveConfig().Starters)
	stats, bye := contractsapp.RosterReports(snap, mine, contractsapp.Scores{Feed: season.state(), Rows: season.value},
		contractsapp.Scores{Feed: week.state(), Rows: week.value}, contractsapp.Byes{Feed: byes.state(), Rows: byes.value}, rules, time.Local)
	if err != nil {
		bye.Note += " Starter bounds unavailable: " + err.Error()
	}
	return []Report{wireReport(stats), wireReport(bye)}, nil
}

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
