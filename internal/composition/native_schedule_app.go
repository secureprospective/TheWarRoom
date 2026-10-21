package composition

import (
	"context"
	"fmt"
	"log"
	"time"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// This is a local reference read. It has no fetch path and never reads a fantasy league's schedule.
func (a *App) nativeScheduleDates(parent context.Context, season int, at time.Time) ([]leagueweek.Week,
	error) {
	ctx, cancel := context.WithTimeout(parent, refreshTimeout)
	defer cancel()
	var weeks []leagueweek.Week
	seen := make(map[int]bool)
	_, err := a.core.History.ReferenceBodies(ctx, "TYPE="+nflschedule.Export,
		func(src string, body []byte, _ time.Time) bool {
			if !sameExport(src, nflschedule.Export, season, true) {
				return false
			}
			raw, err := nflschedule.Parse(body)
			if err != nil {
				return false
			}
			week, err := nflschedule.ToWeek(raw, at)
			if err == nil && !seen[week.Number] {
				weeks = append(weeks, week)
				seen[week.Number] = true
			}
			return false
		})
	if err != nil {
		return nil, fmt.Errorf("read held NFL week dates: %w", err)
	}
	return weeks, nil
}

type appBidEnv struct{ app *App }

func (e appBidEnv) BidEnvironment(ctx context.Context, league string,
	at time.Time) (commissioner.BidEnvironment, error) {
	records, err := e.app.leagueRecords(ctx, league)
	if err != nil {
		return commissioner.BidEnvironment{}, err
	}
	season, err := commissioner.SeasonAt(records, commissioner.ReadingInstant(records, at))
	if err != nil {
		return commissioner.BidEnvironment{}, fmt.Errorf("read the league season: %w", err)
	}
	weeks, err := e.app.nativeScheduleDates(ctx, season, at)
	if err != nil {
		return commissioner.BidEnvironment{}, err
	}
	phase := nativeapp.WeekAt(weeks, at)
	if !phase.WeekKnown && !phase.Offseason {
		phase.Offseason, err = commissioner.BidOffseasonAt(records, at, season)
		if err != nil {
			return commissioner.BidEnvironment{}, fmt.Errorf("read bidding offseason: %w", err)
		}
	}
	field := map[string]nativecheck.PlayoffStanding{}
	if phase.WeekKnown {
		bracket, err := e.app.core.History.NativeBracket(ctx, league, season)
		if err != nil {
			return commissioner.BidEnvironment{}, fmt.Errorf("read bidding playoff field: %w", err)
		}
		directory, err := commissioner.NativeDirectoryReading(league, records, at)
		if err != nil {
			return commissioner.BidEnvironment{}, fmt.Errorf("read bidding franchises: %w", err)
		}
		field = nativeapp.PlayoffField(bracket, directory)
	}
	return commissioner.BidEnvironment{Season: phase, Playoff: field}, nil
}

func (a *App) processNativeBidsDue(ctx context.Context) {
	leagues, err := a.core.History.NativeLeagues(ctx)
	if err != nil {
		log.Printf("the war room: read native bid leagues: %v", err)
		return
	}
	facts := a.nativeFacts()
	for _, league := range leagues {
		if err := a.core.History.ProcessDueBids(ctx, league.ID, facts); err != nil {
			log.Printf("the war room: award native bids for %s: %v", league.ID, err)
		}
	}
}

const draftLookEvery = 15 * time.Second

func (a *App) nativeWaiverClock(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var lastDrafts time.Time
	for {
		a.processNativeWaiversDue(ctx)
		a.processNativeBidsDue(ctx)
		// A finished draft is found by reading each draft, so that look is taken every few seconds.
		if time.Since(lastDrafts) >= draftLookEvery {
			lastDrafts = time.Now()
			a.processNativeDraftsDue(ctx)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *App) processNativeWaiversDue(ctx context.Context) {
	leagues, err := a.core.History.NativeLeagues(ctx)
	if err != nil {
		log.Printf("the war room: read native waiver leagues: %v", err)
		return
	}
	facts := a.nativeFacts()
	for _, league := range leagues {
		if err := a.core.History.ProcessDueWaivers(ctx, league.ID, facts); err != nil {
			log.Printf("the war room: process native waivers for %s: %v", league.ID, err)
		}
	}
}
