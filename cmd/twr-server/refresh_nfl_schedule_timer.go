package main

import (
	"context"
	"errors"
	"io"
	"log"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

const (
	nflScheduleRefreshCadence = 6 * time.Hour
	nflScheduleRefreshStartup = 2 * time.Minute
)

type nflSchedulePass struct {
	weeksFetched     int
	leaguesRevised   int
	leaguesUnchanged int
	errors           int
}

func startHostedNFLScheduleRefresh(ctx context.Context, pins *leaguePins, signIns *accountstore.Store, reference *history.Store) func() {
	workerCtx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		run := func() {
			attempt, cancel := context.WithTimeout(workerCtx, nflScheduleRefreshTimeout)
			defer cancel()
			pass := refreshHostedNFLSchedulePass(attempt, pins, signIns, nflSeasonAt(time.Now()), newNFLWeekFetcher(reference, nil))
			log.Printf("hosted NFL schedule: weeks fetched=%d, leagues revised=%d, leagues unchanged=%d, errors=%d", pass.weeksFetched, pass.leaguesRevised, pass.leaguesUnchanged, pass.errors)
		}
		timer := time.NewTimer(nflScheduleRefreshStartup)
		defer timer.Stop()
		select {
		case <-workerCtx.Done():
			return
		case <-timer.C:
			run()
		}
		ticker := time.NewTicker(nflScheduleRefreshCadence)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
	return func() { stop(); <-done }
}

func refreshHostedNFLSchedulePass(ctx context.Context, pins *leaguePins, signIns *accountstore.Store, season int, fetch nflWeekFetcher) nflSchedulePass {
	pass := nflSchedulePass{}
	memo := make(map[[2]int]fetchResult)
	memoized := func(ctx context.Context, season, week int) (leagueweek.Week, error) {
		key := [2]int{season, week}
		if result, ok := memo[key]; ok {
			return result.week, result.err
		}
		weekResult, err := fetch(ctx, season, week)
		memo[key] = fetchResult{week: weekResult, err: err}
		pass.weeksFetched++
		return weekResult, err
	}
	for _, league := range pins.files.leagues() {
		if err := ctx.Err(); err != nil {
			pass.errors++
			log.Printf("hosted NFL schedule: league %s: %v", league, err)
			break
		}
		changed, err := refreshHostedNFLScheduleLeague(ctx, pins, signIns, league, season, memoized)
		if err != nil {
			pass.errors++
			log.Printf("hosted NFL schedule: league %s: %v", league, err)
			continue
		}
		if changed {
			pass.leaguesRevised++
		} else {
			pass.leaguesUnchanged++
		}
	}
	return pass
}

func refreshHostedNFLScheduleLeague(ctx context.Context, pins *leaguePins, signIns *accountstore.Store, league string, season int, fetch nflWeekFetcher) (bool, error) {
	store, ok := pins.files.store(league)
	if !ok {
		return false, errors.New("league is not on this server")
	}
	expected, err := publishedNFLWeeks(ctx, store, league, season)
	if err != nil || len(expected) == 0 {
		return false, err
	}
	digest := pins.digest(league)
	if digest == "" {
		return false, errors.New("league has no pinned football evidence")
	}
	return reviseNFLWeekEvidence(ctx, store, league, season, digest, expected, fetch, io.Discard, func(ctx context.Context, next string) error {
		return pins.setPin(ctx, signIns, league, next)
	})
}

type fetchResult struct {
	week leagueweek.Week
	err  error
}
